package routelimiter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
)

func setupMockConfig() {
	config.Clients = make(map[string]map[string]*config.Client)
	config.Cooldowns = config.CoolDownConfig{
		DomainMutex: make(map[string]*sync.Mutex),
		Client:      make(map[string]map[string]time.Time),
	}
}

func TestDomainHandler(t *testing.T) {
	setupMockConfig()

	config.Routes = make(map[string]config.Config)
	routeConfig := config.Config{
		RateLimit: config.RateLimitConfig{
			Rate:            2,
			Burst:           2,
			Cooldown:        1000,
			DefaultCooldown: 1 * time.Second,
		},
	}
	config.Routes["localhost"] = routeConfig
	config.Cooldowns.DomainMutex["localhost"] = &sync.Mutex{}

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
			handler := DomainHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			for i := 0; i < tt.requestCount; i++ {
				req, err := http.NewRequest("GET", "/", nil)
				if err != nil {
					t.Fatal(err)
				}

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				if rr.Code != tt.expectedResults[i] {
					t.Errorf("Name: %s, Expected status %d but got %d for request %d", tt.name, tt.expectedResults[i], rr.Code, i+1)
				}

				if i < tt.requestCount-1 {
					time.Sleep(tt.waitBetweenReqs)
				}
			}
			time.Sleep(2 * time.Second)
		})
	}
}
