package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestFrontendDeepLinksAndAssets(t *testing.T) {
	const index = `<!doctype html><html><body><div id="root"></div></body></html>`
	handler := frontendHandler(fstest.MapFS{
		"index.html":    {Data: []byte(index)},
		"assets/app.js": {Data: []byte("console.log('app')")},
	})
	tests := []struct {
		method string
		url    string
		status int
		body   string
	}{
		{http.MethodGet, "/", http.StatusOK, index},
		{http.MethodGet, "/services", http.StatusOK, index},
		{http.MethodGet, "/services/e19a03db-a6d3-4abc-aad3-815965cd59c5/health", http.StatusOK, index},
		{http.MethodGet, "/services/svc-1/availability?environment=prod", http.StatusOK, index},
		{http.MethodGet, "/services/svc-1/instances/", http.StatusOK, index},
		{http.MethodHead, "/services/svc-1/health", http.StatusOK, ""},
		{http.MethodGet, "/dashboard", http.StatusOK, index},
		{http.MethodGet, "/environments", http.StatusOK, index},
		{http.MethodGet, "/health", http.StatusOK, index},
		{http.MethodGet, "/health/results?serviceId=svc-1&port=8080", http.StatusOK, index},
		{http.MethodGet, "/incidents", http.StatusOK, index},
		{http.MethodGet, "/alerts", http.StatusOK, index},
		{http.MethodGet, "/events", http.StatusOK, index},
		{http.MethodGet, "/security/sessions", http.StatusOK, index},
		{http.MethodGet, "/assets/app.js", http.StatusOK, "console.log('app')"},
		{http.MethodGet, "/assets/missing.js", http.StatusNotFound, ""},
		{http.MethodGet, "/services/missing.js", http.StatusNotFound, ""},
		{http.MethodGet, "/api/v1/missing", http.StatusNotFound, ""},
		{http.MethodGet, "/registry.v1.HealthService/UnknownMethod", http.StatusNotFound, ""},
		{http.MethodGet, "/unknown", http.StatusNotFound, ""},
		{http.MethodPost, "/services/svc-1/health", http.StatusMethodNotAllowed, ""},
	}
	for _, test := range tests {
		t.Run(test.method+" "+test.url, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.url, nil)
			originalPath := req.URL.Path
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, test.status, rec.Body.String())
			}
			if test.body != "" && rec.Body.String() != test.body {
				t.Fatalf("body = %q, want %q", rec.Body.String(), test.body)
			}
			if test.method == http.MethodHead && rec.Body.Len() != 0 {
				t.Fatal("HEAD returned a response body")
			}
			if rec.Header().Get("Location") != "" || req.URL.Path != originalPath {
				t.Fatal("frontend fallback redirected or mutated the original URL")
			}
			if test.body == index && !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html") {
				t.Fatalf("entry page content type = %q", rec.Header().Get("Content-Type"))
			}
			if test.status != http.StatusOK && strings.Contains(rec.Body.String(), index) {
				t.Fatal("unexpected SPA fallback for a missing asset/API or unsupported method")
			}
		})
	}
}
