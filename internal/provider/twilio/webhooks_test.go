package twilio

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"testing"

	"github.com/kobylinski/sundew/internal/core"
)

func requestForm(t *testing.T, r *http.Request) url.Values {
	t.Helper()
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatal(r)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}
	f, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestStatusRequest(t *testing.T) {
	target := "https://app.test:8443/status?run=one&other=%2B"
	m := &core.Message{ProviderID: "SM00000000000000000000000000000000", Account: testAccount, To: "+15551234567", From: "+15557654321", Options: map[string]string{core.OptStatusCallback: target}}
	original := &http.Request{Header: make(http.Header)}
	original.SetBasicAuth("different-credential-account", testToken)
	m.Exchange.Header = original.Header
	p := New(core.Config{})
	for _, tc := range []struct {
		status         core.Status
		code, wantCode string
	}{
		{core.StatusQueued, "", ""}, {core.StatusSent, "", ""}, {core.StatusDelivered, "", ""},
		{core.StatusFailed, "", "30008"}, {core.StatusFailed, "30003", "30003"},
		{core.StatusDelivered, "30003", ""},
	} {
		t.Run(string(tc.status)+tc.code, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), struct{}{}, "marker")
			r, err := p.StatusRequest(ctx, m, tc.status, tc.code)
			if err != nil {
				t.Fatal(err)
			}
			if r.URL.String() != target || r.Context() != ctx {
				t.Fatal("target or context changed")
			}
			f := requestForm(t, r)
			want := url.Values{"MessageSid": {m.ProviderID}, "MessageStatus": {string(tc.status)}, "AccountSid": {testAccount}, "To": {m.To}, "From": {m.From}, "ApiVersion": {apiVersion}}
			if tc.wantCode != "" {
				want.Set("ErrorCode", tc.wantCode)
			}
			if !reflect.DeepEqual(f, want) {
				t.Fatalf("%v want %v", f, want)
			}
			if r.Header.Get("X-Twilio-Signature") == "" {
				t.Fatal("missing signature")
			}
			if tc.status == core.StatusDelivered && r.Header.Get("X-Twilio-Signature") != "bOMo3a+vcfZmzbPjvGqsaDjRdRw=" {
				t.Fatal("signature did not use captured token and exact payload")
			}
		})
	}
	m.Options = nil
	r, err := p.StatusRequest(context.Background(), m, core.StatusSent, "")
	if r != nil || err != nil {
		t.Fatal(r, err)
	}
}

func TestInboundRequest(t *testing.T) {
	p := New(core.Config{})
	in := core.InboundMessage{Account: testAccount, From: "+15557654321", To: "+15551234567", Body: "STOP", MediaURLs: []string{"https://media.invalid/a", "https://media.invalid/b"}}
	r, err := p.InboundRequest(context.Background(), in, "https://app.test/inbound")
	if err != nil {
		t.Fatal(err)
	}
	f := requestForm(t, r)
	sid := f.Get("MessageSid")
	if !regexp.MustCompile(`^SM[0-9a-f]{32}$`).MatchString(sid) {
		t.Fatal(sid)
	}
	want := url.Values{"MessageSid": {sid}, "SmsSid": {sid}, "SmsMessageSid": {sid}, "AccountSid": {testAccount}, "From": {in.From}, "To": {in.To}, "Body": {"STOP"}, "NumMedia": {"2"}, "NumSegments": {"1"}, "ApiVersion": {apiVersion}, "MediaUrl0": {in.MediaURLs[0]}, "MediaUrl1": {in.MediaURLs[1]}}
	if !reflect.DeepEqual(f, want) {
		t.Fatalf("%v want %v", f, want)
	}
	if r.Header.Get("X-Twilio-Signature") != "" {
		t.Fatal("inbound has no credential to sign with")
	}
	r2, err := p.InboundRequest(context.Background(), in, "http://app.test/inbound")
	if err != nil {
		t.Fatal(err)
	}
	if requestForm(t, r2).Get("MessageSid") == sid {
		t.Fatal("SID reused")
	}
}

func TestSignaturePublicVector(t *testing.T) {
	// Twilio's published, invented security-doc example, not a provider credential.
	f := url.Values{"CallSid": {"CA1234567890ABCDE"}, "Caller": {"+14158675310"}, "Digits": {"1234"}, "From": {"+14158675310"}, "To": {"+18005551212"}}
	target := "https://example.com/myapp.php?foo=1&bar=2"
	if got := signature(target, f, "12345"); got != "L/OH5YylLD5NRKLltdqwSvS0BnU=" {
		t.Fatal(got)
	}
	p := New(core.Config{})
	r, err := p.webhook(context.Background(), "https://invented-user:invented-password@example.com/myapp.php?foo=1&bar=2", "12345", true, f)
	if err != nil {
		t.Fatal(err)
	}
	if r.Header.Get("X-Twilio-Signature") != "L/OH5YylLD5NRKLltdqwSvS0BnU=" {
		t.Fatal("userinfo was included in signature")
	}
	for _, target := range []string{"http://example.com:8080/hook?x=%2B&y=1", "https://example.com:8443/hook?x=1"} {
		r, err := p.webhook(context.Background(), target, "12345", true, f)
		if err != nil {
			t.Fatal(err)
		}
		if r.Header.Get("X-Twilio-Signature") != signature(target, f, "12345") {
			t.Fatal("port or query changed")
		}
	}
}

func TestUnsignedStatusAndInvalidURL(t *testing.T) {
	p := New(core.Config{})
	for _, auth := range []string{"", "Basic invalid-base64", "Bearer invented"} {
		m := &core.Message{Exchange: core.Exchange{Header: http.Header{"Authorization": {auth}}}, Options: map[string]string{core.OptStatusCallback: "http://app.test/hook"}}
		r, err := p.StatusRequest(context.Background(), m, core.StatusSent, "")
		if err != nil {
			t.Fatal(err)
		}
		if r.Header.Get("X-Twilio-Signature") != "" {
			t.Fatal("signed without a Basic token")
		}
	}
	for _, target := range []string{"", "/relative", "ftp://app.test/hook", "://broken"} {
		if _, err := p.InboundRequest(context.Background(), core.InboundMessage{}, target); err == nil {
			t.Fatalf("accepted %q", target)
		}
		if _, err := p.StatusRequest(context.Background(), &core.Message{Options: map[string]string{core.OptStatusCallback: target}}, core.StatusSent, ""); err == nil && target != "" {
			t.Fatalf("accepted %q", target)
		}
	}
}
