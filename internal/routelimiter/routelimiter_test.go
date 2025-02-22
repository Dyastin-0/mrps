package routelimiter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/types"
)

func setupMockConfig() {
	config.ClientMngr = sync.Map{}
	config.DomainTrie = types.NewDomainTrie()
}

func TestDomainHandler(t *testing.T) {
	setupMockConfig()

	// Configure the DomainTrie with rate-limiting
	routeConfig := types.Config{
		Enabled: true,
		Routes: types.RouteConfig{
			"/": types.PathConfig{Dests: []types.Dest{{URL: "localhost"}}},
		},
		RateLimit: types.RateLimitConfig{
			Rate:            2,
			Burst:           2,
			Cooldown:        1000,
			DefaultCooldown: 1 * time.Second,
		},
	}
	config.DomainTrie.Insert("localhost", &routeConfig)

	// Create a mock request handler
	handler := Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

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
			name:         "Recover after cooldown",
			requestCount: 10,
			expectedResults: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusTooManyRequests,
				http.StatusOK,
			},
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

			for i := 0; i < tt.requestCount; i++ {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					t.Fatal(err)
				}

				req.Host = "localhost"
				req.RemoteAddr = "127.0.0.1:12345"

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				if rr.Code != tt.expectedResults[i] {
					t.Errorf("Name: %s, Expected status %d but got %d for request %d", tt.name, tt.expectedResults[i], rr.Code, i+1)
				}

				if i < tt.requestCount-1 {
					time.Sleep(tt.waitBetweenReqs)
				}
			}

			// Allow cooldown recovery between test runs
			time.Sleep(2 * time.Second)
		})
	}
}
