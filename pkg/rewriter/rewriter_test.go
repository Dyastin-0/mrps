package rewriter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRewriter(t *testing.T) {
	tests := []struct {
		name     string
		rule     RewriteRule
		path     string
		wantPath string
	}{
		{
			name: "Prefix rewrite",
			rule: RewriteRule{
				Type:       PrefixRewrite,
				Value:      "/api",
				ReplaceVal: "",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
		{
			name: "Regex rewrite",
			rule: RewriteRule{
				Type:       RegexRewrite,
				Value:      "^/api/(.*)$",
				ReplaceVal: "/$1",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
		{
			name: "Mixed rules (Prefix match first)",
			rule: RewriteRule{
				Type:       PrefixRewrite,
				Value:      "/api",
				ReplaceVal: "",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
		{
			name: "Regex replace",
			rule: RewriteRule{
				Type:       RegexRewrite,
				Value:      "^/api/(.*)$",
				ReplaceVal: "api/v2/$1",
			},
			path:     "/api/test",
			wantPath: "api/v2/test",
		},
		{
			name: "Prefix empty replace",
			rule: RewriteRule{
				Type:       PrefixRewrite,
				Value:      "/api",
				ReplaceVal: "",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
		{
			name: "Regex empty replace",
			rule: RewriteRule{
				Type:       RegexRewrite,
				Value:      "^/api/(.*)$",
				ReplaceVal: "",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := New(tt.rule)

			gotPath := rw.RewritePath(tt.path)
			if gotPath != tt.wantPath {
				t.Errorf("RewritePath(%q) = %q, want %q", tt.path, gotPath, tt.wantPath)
			}
		})
	}
}

func TestRewriterMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		rule     RewriteRule
		path     string
		wantPath string
	}{
		{
			name: "Prefix rewrite middleware",
			rule: RewriteRule{
				Type:       PrefixRewrite,
				Value:      "/api",
				ReplaceVal: "",
			},
			path:     "/api/test",
			wantPath: "/test",
		},
		{
			name: "Regex rewrite middleware",
			rule: RewriteRule{
				Type:       RegexRewrite,
				Value:      "^/api/(.*)$",
				ReplaceVal: "/$1",
			},
			path:     "/api/v1/user",
			wantPath: "/v1/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := New(tt.rule)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			handler := rw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.wantPath {
					t.Errorf("Middleware rewrite for path %q = %q, want %q", tt.path, r.URL.Path, tt.wantPath)
				}
			}))

			handler.ServeHTTP(rr, req)
		})
	}
}
