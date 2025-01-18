package middleware_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	"github.com/Dyastin-0/reverse-proxy-server/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestReverseProxyMiddleware(t *testing.T) {
	mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Mock Service received X-Custom-Header: %s",
			r.Header.Get("X-Custom-Header"))

		for header, values := range r.Header {
			if header != "Accept-Encoding" && header != "User-Agent" {
				w.Header().Set(header, values[0])
			}
		}
		w.Write([]byte("Hello from the service!"))
		log.Printf("Mock Service sending body: Hello from the service!")
	}))
	defer mockService.Close()

	mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Mock Service 1 received X-Custom-Header: %s",
			r.Header.Get("X-Custom-Header"))

		for header, values := range r.Header {
			if header != "Accept-Encoding" && header != "User-Agent" {
				w.Header().Set(header, values[0])
			}
		}
		w.Write([]byte("Hello from the service-1!"))
		log.Printf("Mock Service 1 sending body: Hello from the service-1!")
	}))
	defer mockService1.Close()

	config.ReverseProxy = map[string]string{
		"/service/api":   mockService.URL,
		"/service-1/api": mockService1.URL,
	}

	handler := middleware.ReverseProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Not Found handler received X-Custom-Header: %s",
			r.Header.Get("X-Custom-Header"))
		http.Error(w, "not found", http.StatusNotFound)
		log.Printf("Not Found handler sending body: not found")
	}))

	t.Run("Test /service/api with header", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/service/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("X-Custom-Header", "custom-value")
		log.Printf("Sending request to /service/api with X-Custom-Header: %s",
			req.Header.Get("X-Custom-Header"))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()

		log.Printf("Response from /service/api - X-Custom-Header: %s",
			resp.Header.Get("X-Custom-Header"))
		log.Printf("Response body: %s", recorder.Body.String())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Hello from the service!", recorder.Body.String())
	})

	t.Run("Test /service-1/api with header", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/service-1/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("X-Custom-Header", "custom-value")
		log.Printf("Sending request to /service-1/api with X-Custom-Header: %s",
			req.Header.Get("X-Custom-Header"))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()

		log.Printf("Response from /service-1/api - X-Custom-Header: %s",
			resp.Header.Get("X-Custom-Header"))
		log.Printf("Response body: %s", recorder.Body.String())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))
		assert.Equal(t, "Hello from the service-1!", recorder.Body.String())
	})

	t.Run("Test header preservation for non-matching path", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/unknown/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		req.Header.Set("X-Custom-Header", "custom-value")
		log.Printf("Sending request to /unknown/api with X-Custom-Header: %s",
			req.Header.Get("X-Custom-Header"))

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		resp := recorder.Result()

		log.Printf("Response from /unknown/api - X-Custom-Header: %s",
			resp.Header.Get("X-Custom-Header"))
		log.Printf("Response body: %s", recorder.Body.String())

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Equal(t, "not found\n", recorder.Body.String())
	})
}
