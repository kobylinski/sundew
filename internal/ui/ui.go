// Package ui serves the compiled, embedded Svelte application. A Go-only clone
// has a small placeholder; the web/image build adds index.html and hashed assets.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/kobylinski/sundew/internal/core"
)

//go:embed all:static
var assets embed.FS

const csp = "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// Register mounts the UI fallback. More specific provider/API routes win on the
// mux; reserved service roots are never handled as client-side application paths.
func Register(mux *http.ServeMux, _ core.Deps) {
	files, _ := fs.Sub(assets, "static")
	mux.Handle("/", handler(files))
}

func handler(files fs.FS) http.Handler {
	index := "index.html"
	if _, err := fs.Stat(files, index); err != nil {
		index = "placeholder.html"
	}
	fileServer := http.FileServerFS(files)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, root := range []string{"/api", "/healthz", "/2010-04-01"} {
			if r.URL.Path == root || strings.HasPrefix(r.URL.Path, root+"/") {
				if root == "/healthz" && r.URL.Path == root && r.Method != http.MethodGet && r.Method != http.MethodHead {
					w.Header().Set("Allow", "GET, HEAD")
					http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
				} else {
					http.NotFound(w, r)
				}
				return
			}
		}
		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			name := strings.TrimPrefix(r.URL.Path, "/")
			info, err := fs.Stat(files, name)
			if err != nil || info.IsDir() || path.Clean(name) != name {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			fileServer.ServeHTTP(w, r)
			return
		}
		body, err := fs.ReadFile(files, index)
		if err != nil {
			http.Error(w, "UI unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
	})
}
