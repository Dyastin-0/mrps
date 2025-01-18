package reverseproxy_test

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	reverseproxy "github.com/Dyastin-0/reverse-proxy-server/pkg/reverse_proxy"
	"github.com/stretchr/testify/assert"
)

func logRequestResponse(req *http.Request, resp *http.Response, err error) {
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	resp.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

	log.Printf("X-Custom-Header: %s", req.Header.Get("X-Custom-Header"))
	log.Printf("Response Body: %s", string(body))
}

func TestReverseProxy(t *testing.T) {
	t.Run("Test /service/api", func(t *testing.T) {
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Mock Service received X-Custom-Header: %s", r.Header.Get("X-Custom-Header"))
			w.Header().Set("X-Custom-Header", r.Header.Get("X-Custom-Header"))
			w.Write([]byte("Hello, from service"))
			log.Printf("Mock Service sending body: Hello, from service")
		}))
		defer mockService.Close()

		proxy := reverseproxy.New(mockService.URL)
		proxyServer := httptest.NewServer(proxy)
		defer proxyServer.Close()

		req, err := http.NewRequest("GET", proxyServer.URL+"/service/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")
		log.Printf("Sending request to /service/api with X-Custom-Header: %s", req.Header.Get("X-Custom-Header"))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		logRequestResponse(req, resp, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		assert.Equal(t, "Hello, from service", string(body))
	})

	t.Run("Test /service-1/api", func(t *testing.T) {
		mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Mock Service 1 received X-Custom-Header: %s", r.Header.Get("X-Custom-Header"))
			w.Header().Set("X-Custom-Header", r.Header.Get("X-Custom-Header"))
			w.Write([]byte("Hello, from service-1"))
			log.Printf("Mock Service 1 sending body: Hello, from service-1")
		}))
		defer mockService1.Close()

		proxy := reverseproxy.New(mockService1.URL)
		proxyServer := httptest.NewServer(proxy)
		defer proxyServer.Close()

		req, err := http.NewRequest("GET", proxyServer.URL+"/service-1/api", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("X-Custom-Header", "custom-value")
		log.Printf("Sending request to /service-1/api with X-Custom-Header: %s", req.Header.Get("X-Custom-Header"))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		logRequestResponse(req, resp, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "custom-value", resp.Header.Get("X-Custom-Header"))

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		assert.Equal(t, "Hello, from service-1", string(body))
	})
}
