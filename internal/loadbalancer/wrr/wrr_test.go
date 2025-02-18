package wrr_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/wrr"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/stretchr/testify/assert"
)

func startTestServer(healthy bool, key string) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if healthy {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(key))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})
	server := httptest.NewServer(handler)
	return server
}

func TestWeightedRoundRobin(t *testing.T) {
	server1 := startTestServer(true, "server1")
	defer server1.Close()

	server2 := startTestServer(true, "server2")
	defer server2.Close()

	server3 := startTestServer(true, "server3")
	defer server3.Close()

	dests := []types.Dest{
		{URL: server1.URL, Weight: 3},
		{URL: server2.URL, Weight: 2},
		{URL: server3.URL, Weight: 1},
	}

	path := "/api/v1"

	wrrInstance := wrr.New(context.Background(), dests, rewriter.RewriteRule{}, path, "localhost")

	assert.Len(t, wrrInstance.Dests, 3, "should initialize with 3 destinations")
	counts := map[string]int{
		server1.URL: 0,
		server2.URL: 0,
		server3.URL: 0,
	}

	for i := 0; i < 12; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)

		served := wrrInstance.Serve(rec, req, 3)
		assert.True(t, served, "a destination should be selected")

		switch rec.Body.String() {
		case "server1":
			counts[server1.URL]++
		case "server2":
			counts[server2.URL]++
		case "server3":
			counts[server3.URL]++
		}
	}

	assert.Equal(t, 6, counts[server1.URL], "server1 should get the most requests")
	assert.Equal(t, 4, counts[server2.URL], "server2 should get medium requests")
	assert.Equal(t, 2, counts[server3.URL], "server3 should get the least requests")
}
