package rr_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
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

func TestRoundRobinBasic(t *testing.T) {
	dests := []string{
		startTestServer(":8081", true),
		startTestServer(":8082", true),
		startTestServer(":8083", true),
	}
	path := "/api/v1"

	rrInstance := rr.New(dests, path, rewriter.RewriteRule{})

	assert.Equal(t, 3, len(rrInstance.Dests), "should initialize with 3 destinations")
	assert.Len(t, rrInstance.Dests, 3, "activeKeys should have 3 keys")

	dest1 := rrInstance.Next()
	assert.NotNil(t, dest1, "first destination should be returned")
	assert.Equal(t, dest1.URL, "http://localhost:8081", "should return the first destination")

	dest2 := rrInstance.Next()
	assert.NotNil(t, dest2, "second destination should be returned")
	assert.Equal(t, dest2.URL, "http://localhost:8082", "should return the second destination")

	dest3 := rrInstance.Next()
	assert.NotNil(t, dest3, "third destination should be returned")
	assert.Equal(t, dest3.URL, "http://localhost:8083", "should return the third destination")

	dest1Again := rrInstance.Next()
	assert.NotNil(t, dest1Again, "should cycle back to the first destination")
	assert.Equal(t, dest1Again.URL, "http://localhost:8081", "should return the first destination again")
}
