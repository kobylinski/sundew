package twilio

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

const testAccount = "AC00000000000000000000000000000000"
const testToken = "invented-sundew-test-token"

// Embedding catches unexpected operations: only Insert and GetByProviderID
// belong to this facade; an accidental call to any other method fails the test.
type fakeStore struct {
	core.Store
	message     *core.Message
	insertError error
	getError    error
}

func (s *fakeStore) Insert(_ context.Context, m *core.Message) error {
	if s.insertError != nil {
		return s.insertError
	}
	m.ID = "internal-id"
	copy := *m
	s.message = &copy
	return nil
}
func (s *fakeStore) GetByProviderID(_ context.Context, provider, sid string) (*core.Message, error) {
	if s.getError != nil {
		return nil, s.getError
	}
	if s.message == nil || s.message.Provider != provider || s.message.ProviderID != sid {
		return nil, core.ErrNotFound
	}
	copy := *s.message
	return &copy, nil
}

var testNow = time.Date(2026, 10, 8, 6, 0, 0, 0, time.FixedZone("test", 2*3600))

func handler(config core.Config, store *fakeStore) http.Handler {
	mux := http.NewServeMux()
	New(config).Register(mux, core.Deps{Config: config, Store: store, Now: func() time.Time { return testNow }})
	return mux
}
func sendRequest(h http.Handler, form url.Values, account, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/2010-04-01/Accounts/"+testAccount+"/Messages.json?trace=one", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	r.Header.Set("X-Trace", "retained")
	r.Header.Set("Cookie", "invented-cookie")
	r.SetBasicAuth(account, token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func goodForm() url.Values {
	return url.Values{"To": {"+15551234567"}, "From": {"+15557654321"}, "Body": {"Hello, dew!"}}
}
func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err, w.Body.String())
	}
	return result
}
func checkError(t *testing.T, w *httptest.ResponseRecorder, status, code int) {
	t.Helper()
	got := decode(t, w)
	if w.Code != status || got["status"] != float64(status) || got["code"] != float64(code) || got["message"] == "" || got["more_info"] == "" || w.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response %d %v %s", w.Code, w.Header(), w.Body.String())
	}
}

func TestSendStoresCanonicalMessageAndExactExchange(t *testing.T) {
	store := &fakeStore{}
	cfg := core.Config{BaseURL: "http://sundew.test:8025/"}
	h := handler(cfg, store)
	f := goodForm()
	f["MediaUrl"] = []string{"https://media.invalid/a.png", "https://media.invalid/b.png"}
	f.Set("MessagingServiceSid", "MG00000000000000000000000000000000")
	f.Set("StatusCallback", "http://app.test/callback?kind=sms")
	f.Set("ValidityPeriod", "600")
	f.Set("CustomOption", "kept")
	w := sendRequest(h, f, testAccount, "any-invented-value")
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", w.Code, w.Body)
	}
	got := decode(t, w)
	sid, ok := got["sid"].(string)
	if !ok || !regexp.MustCompile(`^SM[0-9a-f]{32}$`).MatchString(sid) {
		t.Fatalf("sid: %v", got["sid"])
	}
	expected := &core.Message{ID: "internal-id", Provider: "twilio", ProviderID: sid, Account: testAccount, Direction: core.Outbound,
		From: f.Get("From"), To: f.Get("To"), Body: f.Get("Body"), Segments: 1, MediaURLs: f["MediaUrl"], Status: core.StatusQueued,
		Options: map[string]string{"StatusCallback": f.Get("StatusCallback"), "ValidityPeriod": "600", "MessagingServiceSid": f.Get("MessagingServiceSid"), "CustomOption": "kept",
			core.OptStatusCallback: f.Get("StatusCallback"), core.OptValidity: "600", optMessagingService: f.Get("MessagingServiceSid")},
		CreatedAt: testNow.UTC(), UpdatedAt: testNow.UTC(),
		Exchange: core.Exchange{Method: "POST", Path: "/2010-04-01/Accounts/" + testAccount + "/Messages.json", Query: "trace=one",
			Header: http.Header{"Content-Type": {"application/x-www-form-urlencoded; charset=utf-8"}, "X-Trace": {"retained"}, "Cookie": {"invented-cookie"}, "Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(testAccount+":any-invented-value"))}},
			Body:   f.Encode(), ResponseStatus: 201, ResponseHeader: http.Header{"Content-Type": {"application/json"}}, ResponseBody: w.Body.String()}}
	if !reflect.DeepEqual(store.message, expected) {
		t.Fatalf("stored:\n%+v\nexpected:\n%+v", store.message, expected)
	}
	path := "http://sundew.test:8025/2010-04-01/Accounts/" + testAccount + "/Messages/" + sid
	want := map[string]any{"sid": sid, "account_sid": testAccount, "to": f.Get("To"), "from": f.Get("From"), "body": f.Get("Body"),
		"status": "queued", "num_segments": "1", "num_media": "2", "direction": "outbound-api", "date_created": "Thu, 08 Oct 2026 04:00:00 +0000", "date_updated": "Thu, 08 Oct 2026 04:00:00 +0000", "date_sent": nil,
		"uri": path + ".json", "subresource_uris": map[string]any{"media": path + "/Media.json"}, "error_code": nil, "error_message": nil,
		"price": nil, "price_unit": nil, "api_version": apiVersion, "messaging_service_sid": f.Get("MessagingServiceSid")}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("response:\n%v\nwant:\n%v", got, want)
	}
	if store.message.Exchange.ResponseHeader.Get("Content-Type") != w.Header().Get("Content-Type") {
		t.Fatal("stored response headers differ")
	}
}

func TestValidation(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		code        int
	}{
		{"missing recipient", "To", 21604}, {"missing sender", "From", 21603}, {"missing content", "Body", 21602},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeStore{}
			form := goodForm()
			form.Del(tc.field)
			checkError(t, sendRequest(handler(core.Config{}, store), form, testAccount, testToken), 400, tc.code)
			if store.message != nil {
				t.Fatal("invalid request was stored")
			}
		})
	}
	t.Run("query parameters cannot supply required fields", func(t *testing.T) {
		r := httptest.NewRequest("POST", "/2010-04-01/Accounts/"+testAccount+"/Messages.json?To=%2B15551234567", strings.NewReader("From=x&Body=hello"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.SetBasicAuth(testAccount, testToken)
		w := httptest.NewRecorder()
		handler(core.Config{}, &fakeStore{}).ServeHTTP(w, r)
		checkError(t, w, 400, 21604)
	})
	for _, tc := range []struct {
		name         string
		form         url.Values
		wantFrom     any
		wantSegments string
	}{
		{"messaging service sender", url.Values{"To": {"+15551234567"}, "MessagingServiceSid": {"MG00000000000000000000000000000000"}, "Body": {"hello"}}, nil, "1"},
		{"media only", url.Values{"To": {"+15551234567"}, "From": {"sender"}, "MediaUrl": {"https://media.invalid/image.png"}}, "sender", "0"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sendRequest(handler(core.Config{}, &fakeStore{}), tc.form, testAccount, testToken)
			if w.Code != 201 {
				t.Fatalf("%d: %s", w.Code, w.Body)
			}
			got := decode(t, w)
			if got["from"] != tc.wantFrom || got["num_segments"] != tc.wantSegments {
				t.Fatal(got)
			}
		})
	}
}

func TestAnyCredentialAcceptedAndStoredAsSent(t *testing.T) {
	for _, tc := range []struct{ name, auth, account string }{
		{"missing", "", testAccount},
		{"malformed", "Basic invalid-base64", testAccount},
		{"bearer", "Bearer invented", testAccount},
		{"matching Basic", "Basic " + base64.StdEncoding.EncodeToString([]byte(testAccount+":"+testToken)), testAccount},
		{"mismatched Basic", "Basic " + base64.StdEncoding.EncodeToString([]byte("arbitrary-account:"+testToken)), "arbitrary-account"},
		{"empty Basic", "Basic Og==", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/2010-04-01/Accounts/"+testAccount+"/Messages.json", strings.NewReader(goodForm().Encode()))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if tc.auth != "" {
				r.Header.Set("Authorization", tc.auth)
			}
			s := &fakeStore{}
			w := httptest.NewRecorder()
			handler(core.Config{}, s).ServeHTTP(w, r)
			if w.Code != 201 || s.message.Account != tc.account || !reflect.DeepEqual(s.message.Exchange.Header, r.Header) {
				t.Fatalf("%d, %+v", w.Code, s.message)
			}
		})
	}
	// Fetch also accepts missing, malformed or unrelated credentials.
	s := &fakeStore{}
	h := handler(core.Config{}, s)
	sendRequest(h, goodForm(), testAccount, testToken)
	for _, auth := range []string{"", "Bearer invented", "Basic invalid-base64"} {
		r := httptest.NewRequest("GET", "/2010-04-01/Accounts/"+testAccount+"/Messages/"+s.message.ProviderID+".json", nil)
		r.Header.Set("Authorization", auth)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestMagicNumbers(t *testing.T) {
	for _, tc := range []struct {
		field, number string
		code          int
	}{
		{"From", "+15005550001", 21212}, {"From", "+15005550007", 21606}, {"From", "+15005550008", 21611},
		{"To", "+15005550001", 21211}, {"To", "+15005550002", 21612}, {"To", "+15005550003", 21408},
		{"To", "+15005550004", 21610}, {"To", "+15005550009", 21614},
	} {
		t.Run(tc.field+tc.number, func(t *testing.T) {
			f := goodForm()
			f.Set(tc.field, tc.number)
			s := &fakeStore{}
			w := sendRequest(handler(core.Config{}, s), f, testAccount, testToken)
			checkError(t, w, 400, tc.code)
			if s.message != nil {
				t.Fatal("magic error was stored")
			}
		})
	}
	for _, field := range []string{"From", "To"} {
		for _, number := range []string{"+15005550006", "+15005550000", "+15551234567"} {
			f := goodForm()
			f.Set(field, number)
			w := sendRequest(handler(core.Config{}, &fakeStore{}), f, testAccount, testToken)
			if w.Code != 201 {
				t.Fatal(field, number, w.Code, w.Body.String())
			}
		}
	}
}

func TestFetch(t *testing.T) {
	store := &fakeStore{}
	h := handler(core.Config{}, store)
	initial := decode(t, sendRequest(h, goodForm(), testAccount, testToken))
	path := "/2010-04-01/Accounts/" + testAccount + "/Messages/" + initial["sid"].(string) + ".json"
	for _, status := range []core.Status{core.StatusQueued, core.StatusSent, core.StatusDelivered, core.StatusFailed} {
		store.message.Status = status
		store.message.UpdatedAt = testNow.Add(time.Minute)
		store.message.ErrorCode = ""
		if status == core.StatusFailed {
			store.message.ErrorCode = "30008"
		}
		r := httptest.NewRequest("GET", path, nil)
		r.SetBasicAuth(testAccount, testToken)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		got := decode(t, w)
		if w.Code != 200 || got["status"] != string(status) || got["date_updated"] != "Thu, 08 Oct 2026 04:01:00 +0000" {
			t.Fatal(w.Code, got)
		}
		if status == core.StatusFailed && (got["error_code"] != float64(30008) || got["error_message"] != "Unknown error") {
			t.Fatal(got)
		}
	}
	for _, tc := range []struct {
		name, path string
		modify     func()
	}{
		{"unknown", "/2010-04-01/Accounts/" + testAccount + "/Messages/SMunknown.json", func() {}},
		{"wrong suffix", strings.TrimSuffix(path, ".json"), func() {}},
		{"wrong account", path, func() { store.message.Account = "AC11111111111111111111111111111111" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.modify()
			r := httptest.NewRequest("GET", tc.path, nil)
			r.SetBasicAuth(testAccount, testToken)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			checkError(t, w, 404, 20404)
		})
	}
}

func TestMalformedAndStoreFailures(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body string
		status                  int
	}{
		{"malformed", "application/x-www-form-urlencoded", "%XX", 400},
		{"json", "application/json", `{}`, 400},
		{"too large", "application/x-www-form-urlencoded", strings.Repeat("x", (1<<20)+1), 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/2010-04-01/Accounts/"+testAccount+"/Messages.json", strings.NewReader(tc.body))
			r.SetBasicAuth(testAccount, testToken)
			r.Header.Set("Content-Type", tc.contentType)
			s := &fakeStore{}
			w := httptest.NewRecorder()
			handler(core.Config{}, s).ServeHTTP(w, r)
			checkError(t, w, tc.status, 20001)
			if s.message != nil {
				t.Fatal("malformed request stored")
			}
		})
	}
	s := &fakeStore{insertError: errors.New("private database details")}
	w := sendRequest(handler(core.Config{}, s), goodForm(), testAccount, testToken)
	checkError(t, w, 500, 20500)
	if strings.Contains(w.Body.String(), "private") {
		t.Fatal("store error exposed")
	}
	s.getError = errors.New("private lookup details")
	r := httptest.NewRequest("GET", "/2010-04-01/Accounts/"+testAccount+"/Messages/SMunknown.json", nil)
	r.SetBasicAuth(testAccount, testToken)
	w = httptest.NewRecorder()
	handler(core.Config{}, s).ServeHTTP(w, r)
	checkError(t, w, 500, 20500)
}

func TestSegments(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       int
	}{
		{"empty", "", 0}, {"GSM160", strings.Repeat("a", 160), 1}, {"GSM161", strings.Repeat("a", 161), 2},
		{"GSM306", strings.Repeat("a", 306), 2}, {"GSM307", strings.Repeat("a", 307), 3},
		{"UCS70", strings.Repeat("界", 70), 1}, {"UCS71", strings.Repeat("界", 71), 2},
		{"UCS134", strings.Repeat("界", 134), 2}, {"UCS135", strings.Repeat("界", 135), 3},
		{"all extension characters", "\f^{}\\[~]|€", 1}, {"extensions fit", strings.Repeat("^", 80), 1},
		{"extensions multipart", strings.Repeat("^", 81), 2}, {"extension cannot split", strings.Repeat("^", 153), 3},
		{"emoji fit", strings.Repeat("😀", 35), 1}, {"emoji multipart", strings.Repeat("😀", 36), 2},
		{"surrogate cannot split", strings.Repeat("😀", 67), 3}, {"non GSM changes whole message", strings.Repeat("a", 70) + "界", 2},
		{"GSM accented", strings.Repeat("£éΔ", 53) + "a", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := segments(tc.body); got != tc.want {
				t.Fatalf("segments %d want %d", got, tc.want)
			}
		})
	}
}
