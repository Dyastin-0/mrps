package iphash_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/iphash"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/stretchr/testify/assert"
)

// Helper function to start a test HTTP server
func startTestServer(port string, healthy bool) string {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	server := &http.Server{
		Addr:    port,
		Handler: handler,
	}
	go server.ListenAndServe()
	time.Sleep(500 * time.Millisecond) // Ensure the server starts
	return "http://localhost" + port
}

// Helper function to create a fake HTTP request with a specified client IP
func newTestRequest(clientIP string) *http.Request {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = clientIP + ":12345" // Fake remote address
	return req
}

func TestIPHashBasic(t *testing.T) {
	dests := []types.Dest{
		{URL: startTestServer(":8081", true)},
		{URL: startTestServer(":8082", true)},
		{URL: startTestServer(":8083", true)},
	}
	path := "/api/v1"

	ipHashInstance := iphash.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost")

	assert.Equal(t, 3, len(ipHashInstance.Dests), "should initialize with 3 destinations")

	req1 := newTestRequest("192.168.1.1")
	req2 := newTestRequest("192.168.1.2")
	req3 := newTestRequest("10.0.0.5")

	dest1 := ipHashInstance.Serve(req1)
	dest2 := ipHashInstance.Serve(req2)
	dest3 := ipHashInstance.Serve(req3)

	assert.NotNil(t, dest1, "should return a destination for client IP 1")
	assert.NotNil(t, dest2, "should return a destination for client IP 2")
	assert.NotNil(t, dest3, "should return a destination for client IP 3")

	assert.Equal(t, dest1, ipHashInstance.Serve(req1), "same IP should get same backend")
	assert.Equal(t, dest2, ipHashInstance.Serve(req2), "same IP should get same backend")
	assert.Equal(t, dest3, ipHashInstance.Serve(req3), "same IP should get same backend")
}

func TestIPHashFailover(t *testing.T) {
	dests := []types.Dest{
		{URL: startTestServer(":8081", true)},
		{URL: startTestServer(":8082", false)},
		{URL: startTestServer(":8083", true)},
	}
	path := "/api/v1"

	ipHashInstance := iphash.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost")
	time.Sleep(11 * time.Second) // Allow health checks to run

	req := newTestRequest("192.168.1.10")
	dest := ipHashInstance.ServeAlive(req)

	assert.NotNil(t, dest, "should return a destination")
	assert.NotEqual(t, "http://localhost:8082", dest.URL, "should not return an unhealthy backend")
}
