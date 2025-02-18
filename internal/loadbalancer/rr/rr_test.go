package rr_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/stretchr/testify/assert"
)

func startTestServer(healthy bool) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	server := httptest.NewServer(handler)
	return server
}

func TestRoundRobinBasic(t *testing.T) {
	// Start test servers
	server1 := startTestServer(true)
	defer server1.Close()

	server2 := startTestServer(true)
	defer server2.Close()

	server3 := startTestServer(true)
	defer server3.Close()

	dests := []types.Dest{
		{URL: server1.URL},
		{URL: server2.URL},
		{URL: server3.URL},
	}

	path := "/api/v1"
	rrInstance := rr.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost")

	assert.Len(t, rrInstance.Dests, 3, "should initialize with 3 destinations")

	for i := 0; i < 3; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		success := rrInstance.Serve(rec, req, 5)
		assert.True(t, success, "Serve should return true for a healthy destination")
		assert.Equal(t, http.StatusOK, rec.Code, "Each destination should return 200 OK")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	success := rrInstance.Serve(rec, req, 5)
	assert.True(t, success, "Should cycle back to the first destination")
	assert.Equal(t, http.StatusOK, rec.Code, "Response should still be 200 OK")
}
