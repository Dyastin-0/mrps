package reverseproxy_test

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	reverseproxy "github.com/Dyastin-0/reverse-proxy-server/pkg/reverse_proxy"
	"github.com/stretchr/testify/assert"
)

func logRequestResponse(resp *http.Response, err error) {
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return
	}

	log.Printf("Response Status: %d", resp.StatusCode)
	log.Printf("Response Headers: %v", resp.Header)
	log.Printf("Response Body: %s", string(body))
}

func TestReverseProxyService(t *testing.T) {
	mockService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from the service!"))
	}))
	defer mockService.Close()

	mockService1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello from the service-1!"))
	}))
	defer mockService1.Close()

	proxy := reverseproxy.New(mockService.URL)
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	log.Printf("Testing /api/service")
	resp, err := http.Get(proxyServer.URL + "/api/service")
	if err != nil {
		log.Printf("Request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	logRequestResponse(resp, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	proxy2 := reverseproxy.New(mockService1.URL)
	proxyServer2 := httptest.NewServer(proxy2)
	defer proxyServer2.Close()

	log.Printf("Testing /api/service-1")
	resp2, err2 := http.Get(proxyServer2.URL + "/api/service-1")
	if err2 != nil {
		log.Printf("Request failed: %v", err2)
		return
	}
	defer resp2.Body.Close()

	logRequestResponse(resp2, err2)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}
