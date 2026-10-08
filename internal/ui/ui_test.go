package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kobylinski/sundew/internal/core"
)

func TestApplicationRoutesAndHeaders(t *testing.T) {
	h := handler(fstest.MapFS{
		"index.html":          {Data: []byte("<!doctype html><div id=app></div>")},
		"assets/app-a123.js":  {Data: []byte("export const ready = true;")},
		"assets/app-a123.css": {Data: []byte("body { color: green; }")},
	})
	for _, tc := range []struct{ path, content, cache string }{
		{"/", "text/html", "no-cache"},
		{"/install", "text/html", "no-cache"},
		{"/messages/a", "text/html", "no-cache"},
		{"/unknown/deep/link", "text/html", "no-cache"},
		{"/assets/app-a123.js", "javascript", "immutable"},
		{"/assets/app-a123.css", "text/css", "immutable"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, httptest.NewRequest("GET", tc.path, nil))
			if rr.Code != 200 || !strings.Contains(rr.Header().Get("Content-Type"), tc.content) || !strings.Contains(rr.Header().Get("Cache-Control"), tc.cache) {
				t.Fatalf("response %d, headers %v, body %q", rr.Code, rr.Header(), rr.Body.String())
			}
			if rr.Header().Get("Content-Security-Policy") != csp || rr.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatal("missing security headers")
			}
		})
	}
	for _, path := range []string{"/assets/missing.js", "/api", "/api/unknown", "/healthz", "/healthz/missing", "/2010-04-01/missing"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
		if rr.Code != 404 {
			t.Fatalf("%s: got %d", path, rr.Code)
		}
	}
	for _, tc := range []struct {
		method string
		code   int
	}{{"HEAD", 200}, {"POST", 405}} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(tc.method, "/install", nil))
		if rr.Code != tc.code || (tc.method == "HEAD" && rr.Body.Len() != 0) {
			t.Fatalf("%s: %d %q", tc.method, rr.Code, rr.Body.String())
		}
	}
}

func TestPlaceholderAndMoreSpecificRegistrations(t *testing.T) {
	h := handler(fstest.MapFS{"placeholder.html": {Data: []byte("UI is not built")}})
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	if rr.Code != 200 || !strings.Contains(rr.Body.String(), "UI is not built") {
		t.Fatal(rr)
	}
	mux := http.NewServeMux()
	Register(mux, core.Deps{})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})
	mux.HandleFunc("/2010-04-01/Accounts/{account}/Messages.json", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/healthz", 200}, {"POST", "/healthz", 405},
		{"GET", "/api/v1/messages", 200}, {"POST", "/2010-04-01/Accounts/a/Messages.json", 201},
	} {
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, httptest.NewRequest(tc.method, tc.path, nil))
		if rr.Code != tc.status || rr.Header().Get("Content-Security-Policy") != "" {
			t.Fatalf("%s %s: %d %v", tc.method, tc.path, rr.Code, rr.Header())
		}
	}
}
