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
		w.Write([]byte("Hello from the service!"))
	}))
	defer mockService.Close()

	mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from the service-1!"))
	}))
	defer mockService1.Close()

	config.ReverseProxy = map[string]string{
		"/service/api":   mockService.URL,
		"/service-1/api": mockService1.URL,
	}

	handler := middleware.ReverseProxy(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))

	t.Run("Test /service/api", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/service/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		resp := recorder.Result()

		log.Printf("Response for /service/api: %d %s", resp.StatusCode, recorder.Body.String())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body := recorder.Body.String()
		assert.Equal(t, "Hello from the service!", body)
	})

	t.Run("Test /service-1/api", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/service-1/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		resp := recorder.Result()

		log.Printf("Response for /service-1/api: %d %s", resp.StatusCode, recorder.Body.String())

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body := recorder.Body.String()
		assert.Equal(t, "Hello from the service-1!", body)
	})

	t.Run("Test non-matching path", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/unknown/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)

		resp := recorder.Result()

		log.Printf("Response for /unknown/api: %d %s", resp.StatusCode, recorder.Body.String())

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		body := recorder.Body.String()
		assert.Equal(t, "not found\n", body)
	})
}
