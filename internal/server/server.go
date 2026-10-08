// Package server assembles Sundew's routes on a single standard-library mux.
package server

import (
	"net/http"

	"github.com/kobylinski/sundew/internal/core"
)

func New(d core.Deps, registrations []func(*http.ServeMux, core.Deps)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	for _, register := range registrations {
		register(mux, d)
	}
	return mux
}
