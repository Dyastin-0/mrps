package limiter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	"github.com/Dyastin-0/reverse-proxy-server/internal/limiter"
)

func TestPerClientRateLimiter(t *testing.T) {
	config.RateLimit = config.RateLimitConfig{
		Burst:    2,
		Rate:     2,
		Cooldown: 1000,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	limiter := limiter.Handler(handler)

	tests := []struct {
		name            string
		requestCount    int
		expectedResults []int
		waitBetweenReqs time.Duration
	}{
		{
			name:            "Exceed rate limit quickly",
			requestCount:    5,
			expectedResults: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests},
			waitBetweenReqs: 100 * time.Millisecond,
		},
		{
			name:            "Stay within rate limit",
			requestCount:    4,
			expectedResults: []int{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK},
			waitBetweenReqs: 600 * time.Millisecond,
		},
		{
			name:            "Recover after cooldown",
			requestCount:    6,
			expectedResults: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusTooManyRequests, http.StatusOK},
			waitBetweenReqs: 100 * time.Millisecond,
		},
		{
			name:            "Test burst behavior",
			requestCount:    3,
			expectedResults: []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests},
			waitBetweenReqs: 200 * time.Millisecond,
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
					t.Errorf("Name: %s, Request %d: Expected status %d, got %d", tt.name, i+1, tt.expectedResults[i], resp.StatusCode)
				}

				if i < tt.requestCount-1 {
					time.Sleep(tt.waitBetweenReqs)
				}
			}
		})

		// Wait for the cooldown to reset
		time.Sleep(1 * time.Second)
	}
}
