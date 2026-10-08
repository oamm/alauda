package server

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func frontendHandler(webFS fs.FS) http.Handler {
	files := http.FileServer(http.FS(webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/registry.v1.") {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		_, err := fs.Stat(webFS, name)
		if errors.Is(err, fs.ErrNotExist) && isFrontendRoute(r.URL.Path) {
			// Serve the entry page without redirecting away from the client route.
			entry := r.Clone(r.Context())
			entry.URL.Path = "/"
			entry.URL.RawPath = ""
			files.ServeHTTP(w, entry)
			return
		}
		files.ServeHTTP(w, r)
	})
}

func isFrontendRoute(urlPath string) bool {
	if path.Ext(urlPath) != "" {
		return false
	}
	root := strings.Split(strings.Trim(urlPath, "/"), "/")[0]
	switch root {
	case "dashboard", "services", "environments", "health", "incidents", "alerts", "security", "events":
		return true
	default:
		return false
	}
}
