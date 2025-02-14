package wrr_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/wrr"
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

func TestWeightedRoundRobin(t *testing.T) {
	dests := []types.Dest{
		{URL: startTestServer(":8081", true), Weight: 3},
		{URL: startTestServer(":8082", true), Weight: 2},
		{URL: startTestServer(":8083", true), Weight: 1},
	}
	path := "/api/v1"

	wrrInstance := wrr.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost")

	assert.Equal(t, 3, len(wrrInstance.Dests), "should initialize with 3 destinations")

	counts := map[string]int{
		"http://localhost:8081": 0,
		"http://localhost:8082": 0,
		"http://localhost:8083": 0,
	}

	for i := 0; i < 12; i++ {
		dest := wrrInstance.Serve(&http.Request{})
		assert.NotNil(t, dest, "destination should be returned")
		counts[dest.URL]++
	}

	// Check if higher-weighted servers got more requests
	assert.Equal(t, 6, counts["http://localhost:8081"], "server1 should get the most requests")
	assert.Equal(t, 4, counts["http://localhost:8082"], "server2 should get medium requests")
	assert.Equal(t, 2, counts["http://localhost:8083"], "server3 should get the least requests")
}
