package api

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kobylinski/sundew/internal/core"
)

type fakeStore struct {
	core.Store
	list   func(core.Filter) ([]*core.Message, string, error)
	latest func(core.Filter) (*core.Message, error)
	get    func(string) (*core.Message, error)
	delete func(string) error
	reset  func() error
	insert func(*core.Message) error
}

func (s *fakeStore) List(_ context.Context, f core.Filter) ([]*core.Message, string, error) {
	return s.list(f)
}
func (s *fakeStore) Latest(_ context.Context, f core.Filter) (*core.Message, error) {
	return s.latest(f)
}
func (s *fakeStore) Get(_ context.Context, id string) (*core.Message, error) { return s.get(id) }
func (s *fakeStore) Delete(_ context.Context, id string) error               { return s.delete(id) }
func (s *fakeStore) Reset(context.Context) error                             { return s.reset() }
func (s *fakeStore) Insert(_ context.Context, m *core.Message) error         { return s.insert(m) }

type fakeBus struct {
	core.Bus
	events       chan core.Event
	subscribed   chan struct{}
	disconnected chan struct{}
	once         sync.Once
}

func newBus() *fakeBus {
	return &fakeBus{events: make(chan core.Event, 32), subscribed: make(chan struct{}), disconnected: make(chan struct{})}
}
func (b *fakeBus) Subscribe(ctx context.Context) <-chan core.Event {
	close(b.subscribed)
	go func() { <-ctx.Done(); b.once.Do(func() { close(b.disconnected) }) }()
	return b.events
}
func (b *fakeBus) Publish(e core.Event) { b.events <- e }

type fakeProvider struct {
	core.Provider
	inbound func(context.Context, core.InboundMessage, string) (*http.Request, error)
}

func (p fakeProvider) InboundRequest(ctx context.Context, in core.InboundMessage, u string) (*http.Request, error) {
	return p.inbound(ctx, in, u)
}
func handler(s *fakeStore) http.Handler {
	mux := http.NewServeMux()
	Register(mux, core.Deps{Store: s})
	return mux
}
func request(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(method, path, strings.NewReader(body)))
	return r
}
func assertError(t *testing.T, r *httptest.ResponseRecorder, status int) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("got %d: %s; want %d", r.Code, r.Body, status)
	}
	var v struct {
		Error struct{ Code, Message string }
	}
	if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil || v.Error.Code == "" || v.Error.Message == "" {
		t.Fatalf("invalid error: %s", r.Body)
	}
}

func TestFilterForwardingAndPaging(t *testing.T) {
	since := time.Date(2026, 10, 8, 3, 0, 0, 123, time.UTC)
	tests := []struct {
		name string
		want core.Filter
	}{
		{"to", core.Filter{To: "+15550001"}}, {"from", core.Filter{From: "+15550002"}}, {"body", core.Filter{BodyContains: "VeRiFy code"}},
		{"account", core.Filter{Account: "invented-account"}}, {"provider", core.Filter{Provider: "twilio"}}, {"since", core.Filter{Since: since}},
		{"combined", core.Filter{To: "+15550001", From: "+15550002", BodyContains: "Code", Account: "invented-account", Provider: "twilio", Since: since, Limit: 2, Cursor: "opaque-next"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := url.Values{}
			f := tc.want
			if f.To != "" {
				q.Set("to", f.To)
			}
			if f.From != "" {
				q.Set("from", f.From)
			}
			if f.BodyContains != "" {
				q.Set("body", f.BodyContains)
			}
			if f.Account != "" {
				q.Set("account", f.Account)
			}
			if f.Provider != "" {
				q.Set("provider", f.Provider)
			}
			if !f.Since.IsZero() {
				q.Set("since", f.Since.Format(time.RFC3339Nano))
			}
			if f.Limit != 0 {
				q.Set("limit", "2")
			}
			if f.Cursor != "" {
				q.Set("cursor", f.Cursor)
			}
			s := &fakeStore{list: func(got core.Filter) ([]*core.Message, string, error) {
				if !reflect.DeepEqual(got, f) {
					t.Errorf("filter=%+v; want %+v", got, f)
				}
				return []*core.Message{{ID: "first"}}, "opaque-next", nil
			}, latest: func(got core.Filter) (*core.Message, error) {
				if !reflect.DeepEqual(got, f) {
					t.Errorf("latest filter=%+v", got)
				}
				return &core.Message{ID: "first"}, nil
			}}
			h := handler(s)
			r := request(h, "GET", "/api/v1/messages?"+q.Encode(), "")
			if r.Code != 200 {
				t.Fatal(r.Body)
			}
			var page struct {
				Items []core.Message `json:"items"`
				Next  string         `json:"next_cursor"`
			}
			_ = json.Unmarshal(r.Body.Bytes(), &page)
			if len(page.Items) != 1 || page.Items[0].ID != "first" || page.Next != "opaque-next" {
				t.Fatalf("page=%s", r.Body)
			}
			r = request(h, "GET", "/api/v1/messages/latest?"+q.Encode(), "")
			if r.Code != 200 {
				t.Fatal(r.Body)
			}
		})
	}
	s := &fakeStore{list: func(f core.Filter) ([]*core.Message, string, error) {
		if f.Cursor != "opaque-next" {
			t.Errorf("cursor=%q", f.Cursor)
		}
		return nil, "", nil
	}}
	r := request(handler(s), "GET", "/api/v1/messages?cursor=opaque-next", "")
	if r.Body.String() != "{\"items\":[],\"next_cursor\":\"\"}\n" {
		t.Fatal(r.Body)
	}
}

func TestRoutesAndStoreErrors(t *testing.T) {
	for _, op := range []string{"get", "latest", "delete", "reset", "delete-all", "list"} {
		for _, failed := range []bool{false, true} {
			t.Run(op+map[bool]string{false: "/ok", true: "/failure"}[failed], func(t *testing.T) {
				var err error
				if failed {
					err = core.ErrNotFound
				}
				calls := 0
				s := &fakeStore{get: func(id string) (*core.Message, error) {
					calls++
					if id != "m1" {
						t.Error(id)
					}
					return &core.Message{ID: id}, err
				}, latest: func(core.Filter) (*core.Message, error) { calls++; return &core.Message{ID: "m1"}, err }, delete: func(id string) error {
					calls++
					if id != "m1" {
						t.Error(id)
					}
					return err
				}, reset: func() error {
					calls++
					if failed {
						return errors.New("disk")
					}
					return nil
				}, list: func(core.Filter) ([]*core.Message, string, error) {
					calls++
					if failed {
						return nil, "", errors.New("disk")
					}
					return nil, "", nil
				}}
				method, path, status := "GET", "/api/v1/messages/m1", 200
				switch op {
				case "latest":
					path = "/api/v1/messages/latest"
				case "delete":
					method = "DELETE"
					status = 204
				case "reset":
					method = "POST"
					path = "/api/v1/reset"
					status = 204
				case "delete-all":
					method = "DELETE"
					path = "/api/v1/messages"
					status = 204
				case "list":
					path = "/api/v1/messages"
				}
				r := request(handler(s), method, path, "")
				if failed {
					status = 404
					if op == "reset" || op == "delete-all" || op == "list" {
						status = 500
					}
					assertError(t, r, status)
				} else if r.Code != status {
					t.Fatalf("%d %s", r.Code, r.Body)
				}
				if calls != 1 {
					t.Fatal(calls)
				}
			})
		}
	}
	s := &fakeStore{get: func(string) (*core.Message, error) { return nil, errors.New("private detail") }}
	assertError(t, request(handler(s), "GET", "/api/v1/messages/m1", ""), 500)
	for _, path := range []string{"/api/v1/messages", "/api/v1/messages/latest", "/api/v1/messages/stream", "/api/v1/messages/m1", "/api/v1/inbound", "/api/v1/reset", "/api/v1/openapi.json"} {
		r := request(handler(nil), "PATCH", path, "")
		assertError(t, r, 405)
		if r.Header().Get("Allow") == "" {
			t.Fatal("missing Allow")
		}
	}
	assertError(t, request(handler(nil), "GET", "/api/v1/unknown", ""), 404)
	for _, q := range []string{"since=bad", "since=", "limit=", "limit=0", "limit=-1", "limit=501", "limit=x", "to=%zz"} {
		for _, path := range []string{"/api/v1/messages", "/api/v1/messages/latest", "/api/v1/messages/stream"} {
			assertError(t, request(handler(nil), "GET", path+"?"+q, ""), 400)
		}
	}
}

func TestInbound(t *testing.T) {
	var got core.InboundMessage
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) }))
	defer receiver.Close()
	for _, useDefault := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit", true: "default"}[useDefault], func(t *testing.T) {
			var stored *core.Message
			p := fakeProvider{inbound: func(ctx context.Context, in core.InboundMessage, u string) (*http.Request, error) {
				got = in
				if u != receiver.URL {
					t.Errorf("url=%q", u)
				}
				return http.NewRequestWithContext(ctx, "POST", u, strings.NewReader(in.Body))
			}}
			d := core.Deps{Store: &fakeStore{insert: func(m *core.Message) error { m.ID = "inbound-id"; stored = m; return nil }}, Providers: map[string]core.Provider{"twilio": p}}
			input := map[string]any{"from": "+1", "to": "+2", "body": "STOP", "account": "invented", "media_urls": []string{"https://example.test/m.png"}}
			if useDefault {
				d.Config.InboundURL = receiver.URL
			} else {
				d.Config.InboundURL = "https://unused.test"
				input["url"] = receiver.URL
				input["provider"] = "twilio"
			}
			data, _ := json.Marshal(input)
			mux := http.NewServeMux()
			Register(mux, d)
			r := request(mux, "POST", "/api/v1/inbound", string(data))
			if r.Code != 201 {
				t.Fatal(r.Body)
			}
			var result struct {
				Message core.Message
				Status  int `json:"response_status"`
			}
			if err := json.Unmarshal(r.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Status != 202 || result.Message.ID != "inbound-id" || stored.Direction != core.Inbound || stored.Status != core.StatusReceived || got.Body != "STOP" || got.Account != "invented" || !reflect.DeepEqual(got.MediaURLs, []string{"https://example.test/m.png"}) {
				t.Fatalf("result=%+v stored=%+v input=%+v", result, stored, got)
			}
		})
	}
	for _, body := range []string{"null", "[1]", "{} {}", "{", "{\"unknown\":1}", "{\"provider\":\"unknown\"}", "{}", "{\"url\":\"file:///tmp/a\"}", "{\"url\":\"http://\"}"} {
		mux := http.NewServeMux()
		Register(mux, core.Deps{Providers: map[string]core.Provider{"twilio": fakeProvider{}}})
		assertError(t, request(mux, "POST", "/api/v1/inbound", body), 400)
	}
	for _, failure := range []string{"provider", "transport", "store"} {
		t.Run(failure, func(t *testing.T) {
			p := fakeProvider{inbound: func(ctx context.Context, _ core.InboundMessage, u string) (*http.Request, error) {
				if failure == "provider" {
					return nil, errors.New("build")
				}
				if failure == "transport" {
					u = "http://127.0.0.1:0"
				}
				return http.NewRequestWithContext(ctx, "POST", u, nil)
			}}
			mux := http.NewServeMux()
			Register(mux, core.Deps{Config: core.Config{InboundURL: receiver.URL}, Providers: map[string]core.Provider{"twilio": p}, Store: &fakeStore{insert: func(*core.Message) error { return errors.New("disk") }}})
			assertError(t, request(mux, "POST", "/api/v1/inbound", "{}"), 500)
		})
	}
}

func TestStreamEventsFilterHeartbeatAndDisconnect(t *testing.T) {
	b := newBus()
	mux := http.NewServeMux()
	a := &api{deps: core.Deps{Bus: b}, heartbeat: time.Millisecond}
	a.register(mux)
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer close(done); mux.ServeHTTP(w, r) }))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/v1/messages/stream?to=match", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatal(resp.Header)
	}
	lines := make(chan string, 128)
	go func() {
		scan := bufio.NewScanner(resp.Body)
		for scan.Scan() {
			lines <- scan.Text()
		}
		close(lines)
	}()
	receive := func(want string) {
		t.Helper()
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					t.Fatalf("stream closed before %s", want)
				}
				if strings.HasPrefix(line, want) {
					return
				}
				if strings.HasPrefix(line, "data:") && strings.Contains(line, "excluded") {
					t.Fatal(line)
				}
			case <-timer.C:
				t.Fatalf("timed out waiting for %s", want)
			}
		}
	}
	receive(": connected")
	receive(": heartbeat")
	b.Publish(core.Event{Type: core.EventCreated, Message: &core.Message{ID: "excluded", To: "other"}})
	for _, kind := range []string{core.EventCreated, core.EventUpdated, core.EventDeleted, core.EventReset} {
		event := core.Event{Type: kind}
		if kind != core.EventReset {
			event.Message = &core.Message{ID: "included", To: "match"}
		}
		b.Publish(event)
		receive("event: " + kind)
		receive("data: ")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler did not return")
	}
	select {
	case <-b.disconnected:
	case <-time.After(time.Second):
		t.Fatal("subscription not canceled")
	}
}

func TestStreamMatching(t *testing.T) {
	m := &core.Message{To: "to", From: "from", Account: "account", Provider: "twilio", Body: "Verification CODE", CreatedAt: time.Now().UTC()}
	f := core.Filter{To: "to", From: "from", Account: "account", Provider: "twilio", BodyContains: "code", Since: m.CreatedAt}
	if !matches(m, f) {
		t.Fatal("combined match rejected")
	}
	for _, bad := range []core.Filter{{To: "x"}, {From: "x"}, {Account: "x"}, {Provider: "x"}, {BodyContains: "x"}, {Since: m.CreatedAt.Add(time.Nanosecond)}} {
		if matches(m, bad) {
			t.Errorf("matched %+v", bad)
		}
	}
	if matches(nil, core.Filter{}) {
		t.Fatal("nil matched")
	}
}

func TestOpenAPIReflectsRegisteredRoutesAndMessage(t *testing.T) {
	h := handler(nil)
	r := request(h, "GET", "/api/v1/openapi.json", "")
	if r.Code != 200 {
		t.Fatal(r.Body)
	}
	var doc struct {
		OpenAPI    string
		Paths      map[string]map[string]json.RawMessage
		Components struct {
			Schemas map[string]struct{ Properties map[string]json.RawMessage }
		}
	}
	if err := json.Unmarshal(r.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.OpenAPI != "3.1.0" {
		t.Fatal(doc.OpenAPI)
	}
	// Independent inventory catches omissions from both the registration and document.
	expected := map[string][]string{"/api/v1/messages": {"get", "delete"}, "/api/v1/messages/{id}": {"get", "delete"}, "/api/v1/messages/latest": {"get"}, "/api/v1/messages/stream": {"get"}, "/api/v1/inbound": {"post"}, "/api/v1/reset": {"post"}, "/api/v1/openapi.json": {"get"}}
	if len(doc.Paths) != len(expected) {
		t.Fatalf("paths=%v", doc.Paths)
	}
	for path, methods := range expected {
		for _, method := range methods {
			if _, ok := doc.Paths[path][method]; !ok {
				t.Fatalf("missing %s %s", method, path)
			}
			r = request(h, "OPTIONS", strings.ReplaceAll(path, "{id}", "sample"), "")
			if !strings.Contains(r.Header().Get("Allow"), strings.ToUpper(method)) {
				t.Fatalf("route absent %s %s", method, path)
			}
		}
	}
	typ := reflect.TypeFor[core.Message]()
	properties := doc.Components.Schemas["Message"].Properties
	for i := 0; i < typ.NumField(); i++ {
		name := strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]
		if _, ok := properties[name]; !ok {
			t.Fatalf("missing message field %s", name)
		}
	}
	for _, schema := range []string{"Message", "Event", "Error"} {
		if _, ok := doc.Components.Schemas[schema]; !ok {
			t.Fatal(schema)
		}
	}
	if !strings.Contains(r.Header().Get("Content-Type"), "json") {
		t.Fatal(r.Header())
	}
}

func TestInvalidStoreFilterIsBadRequest(t *testing.T) {
	err := fmt.Errorf("cursor validation: %w", core.ErrInvalidFilter)
	s := &fakeStore{list: func(core.Filter) ([]*core.Message, string, error) { return nil, "", err }, latest: func(core.Filter) (*core.Message, error) { return nil, err }}
	for _, path := range []string{"/api/v1/messages?cursor=malformed", "/api/v1/messages/latest?cursor=malformed"} {
		r := request(handler(s), "GET", path, "")
		assertError(t, r, 400)
		if !strings.Contains(r.Body.String(), `"code":"invalid_filter"`) {
			t.Fatal(r.Body)
		}
	}
}
