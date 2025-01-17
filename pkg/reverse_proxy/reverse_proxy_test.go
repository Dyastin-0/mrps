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

	log.Printf("Request to %s %s", req.Method, req.URL.String())
	log.Printf("Response Status: %d", resp.StatusCode)
	log.Printf("Response Body: %s", string(body))
}

func TestReverseProxyService(t *testing.T) {
	t.Run("Test /service/api", func(t *testing.T) {
		mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, from service"))
		}))
		defer mockService.Close()

		proxy := reverseproxy.New(mockService.URL)
		proxyServer := httptest.NewServer(proxy)
		defer proxyServer.Close()

		resp, err := http.Get(proxyServer.URL + "/service/api")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		logRequestResponse(resp.Request, resp, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		assert.Equal(t, "Hello, from service", string(body))
	})

	t.Run("Test /service-1/api", func(t *testing.T) {
		mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello, from service-1"))
		}))
		defer mockService1.Close()

		proxy := reverseproxy.New(mockService1.URL)
		proxyServer := httptest.NewServer(proxy)
		defer proxyServer.Close()

		resp, err := http.Get(proxyServer.URL + "/service-1/api")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		logRequestResponse(resp.Request, resp, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		assert.Equal(t, "Hello, from service-1", string(body))
	})
}
