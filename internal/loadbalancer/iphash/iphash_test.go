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
	time.Sleep(500 * time.Millisecond)
	return "http://localhost" + port
}

func newTestRequest(clientIP string) *http.Request {
	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.RemoteAddr = clientIP + ":12345"
	return req
}

func TestIPHashBasic(t *testing.T) {
	dests := []types.Dest{
		{URL: startTestServer(":8081", true)},
		{URL: startTestServer(":8082", true)},
		{URL: startTestServer(":8083", true)},
	}
	path := "/api/v1"

	ipHashInstance := iphash.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost", 1000*time.Millisecond)

	assert.Equal(t, 3, len(ipHashInstance.Dests), "should initialize with 3 destinations")

	req1 := newTestRequest("192.168.1.1")
	req2 := newTestRequest("192.168.1.2")
	req3 := newTestRequest("10.0.0.5")

	rec1 := httptest.NewRecorder()
	rec2 := httptest.NewRecorder()
	rec3 := httptest.NewRecorder()

	assert.True(t, ipHashInstance.Serve(rec1, req1, 3), "should serve request for client IP 1")
	assert.True(t, ipHashInstance.Serve(rec2, req2, 3), "should serve request for client IP 2")
	assert.True(t, ipHashInstance.Serve(rec3, req3, 3), "should serve request for client IP 3")

	rec4 := httptest.NewRecorder()
	rec5 := httptest.NewRecorder()
	rec6 := httptest.NewRecorder()

	assert.True(t, ipHashInstance.Serve(rec4, req1, 3), "same IP should get same backend")
	assert.True(t, ipHashInstance.Serve(rec5, req2, 3), "same IP should get same backend")
	assert.True(t, ipHashInstance.Serve(rec6, req3, 3), "same IP should get same backend")
}
