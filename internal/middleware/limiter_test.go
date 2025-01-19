package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	"github.com/Dyastin-0/reverse-proxy-server/internal/middleware"
)

func TestPerClientRateLimiter(t *testing.T) {
	config.RateLimit = config.RateLimitConfig{
		Burst: 2,
		Rate:  2,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := middleware.RateLimiter(handler)

	tests := []struct {
		name            string
		requestCount    int
		expectedResults []int
		waitBetweenReqs time.Duration
	}{
		{
			name:            "Exceed rate limit",
			requestCount:    3,
			expectedResults: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
			waitBetweenReqs: 100 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(limiter)
			defer server.Close()

			client := &http.Client{}

			for i := 0; i < tt.requestCount; i++ {
				req, err := http.NewRequest("GET", server.URL, nil)
				if err != nil {
					t.Fatalf("Error creating request: %v", err)
				}

				resp, err := client.Do(req)
				if err != nil {
					t.Fatalf("Error sending request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.expectedResults[i] {
					t.Errorf("Request %d: Expected status %d, got %d", i+1, tt.expectedResults[i], resp.StatusCode)
				}

				if i < tt.requestCount-1 {
					time.Sleep(tt.waitBetweenReqs)
				}
			}
		})
	}
}
