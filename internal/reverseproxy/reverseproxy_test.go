package reverseproxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/loadbalancer"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/stretchr/testify/assert"
)

func TestReverseProxyMiddlewareWithDomainTrie(t *testing.T) {
	// Create mock services
	mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, values := range r.Header {
			w.Header().Set(header, values[0])
		}
		w.Write([]byte("Hello from the service!"))
	}))
	defer mockService.Close()

	mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for header, values := range r.Header {
			w.Header().Set(header, values[0])
		}
		w.Write([]byte("Hello from the mockService1!"))
	}))
	defer mockService1.Close()

	// Initialize DomainTrie
	config.DomainTrie = types.NewDomainTrie()
	dests := []types.Dest{{URL: mockService.URL}}
	dests1 := []types.Dest{{URL: mockService1.URL}}
	bl, _ := loadbalancer.New(context.Background(), dests1, rewriter.RewriteRule{}, "http", "rr", "/mock", "localhost", 1000*time.Millisecond)
	bl1, _ := loadbalancer.New(context.Background(), dests, rewriter.RewriteRule{}, "http", "rr", "/api", "localhost", 1000*time.Millisecond)

	conf := &types.Config{
		Routes: types.RouteConfig{
			"/api":  types.PathConfig{Dests: dests, Balancer: bl1},
			"/mock": types.PathConfig{Dests: dests1, Balancer: bl},
		},
	}

	conf.SortedRoutes = []string{"/api", "/mock"}
	config.DomainTrie.Insert("localhost", conf)

	// Create reverse proxy handler
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found\n"))
	}))

	// Test cases
	t.Run("Test /api with header", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")
		req.Host = "localhost"

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Hello from the service!", recorder.Body.String())
	})

	t.Run("Test /mock with header", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/mock", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")
		req.Host = "localhost"

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Hello from the mockService1!", recorder.Body.String())
	})
}
