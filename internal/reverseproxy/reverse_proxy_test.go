package reverseproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
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
	config.DomainTrie = config.NewDomainTrie()
	conf := &config.Config{
		Routes: config.RouteConfig{
			"/api":  mockService.URL,
			"/mock": mockService1.URL,
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
