package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kobylinski/sundew/internal/core"
	"github.com/kobylinski/sundew/internal/server"
)

func TestHealthAndRegistrations(t *testing.T) {
	d := core.Deps{Config: core.Config{Addr: ":1234"}}
	h := server.New(d, []func(*http.ServeMux, core.Deps){func(mux *http.ServeMux, got core.Deps) {
		if got.Config.Addr != d.Config.Addr {
			t.Fatal("dependencies not forwarded")
		}
		mux.HandleFunc("GET /example", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	}})
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/healthz", 200}, {"HEAD", "/healthz", 200}, {"POST", "/healthz", 405}, {"GET", "/example", 204}, {"GET", "/missing", 404}} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(tc.method, tc.path, nil))
		if rr.Code != tc.status {
			t.Fatalf("%s %s: %d, want %d", tc.method, tc.path, rr.Code, tc.status)
		}
	}
}
