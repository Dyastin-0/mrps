package limiter

import (
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"golang.org/x/time/rate"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If the rate limit is not set, assume there is no rate limit
		if config.GlobalRateLimit.Burst == 0 || config.GlobalRateLimit.Rate == 0 {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		config.Cooldowns.MU.Lock()
		if config.Cooldowns.Client["global"] == nil {
			config.Cooldowns.Client["global"] = make(map[string]time.Time)
		}
		cooldownEnd, inCooldown := config.Cooldowns.Client["global"][ip]
		config.Cooldowns.MU.Unlock()

		if inCooldown && time.Now().Before(cooldownEnd) {
			w.Header().Set("Retry-After", cooldownEnd.Format(time.RFC1123))
			http.Error(w, "too many requests 💔⏳", http.StatusTooManyRequests)
			return
		}

		config.Cooldowns.MU.Lock()
		if config.Clients["global"] == nil {
			config.Clients["global"] = make(map[string]*config.Client)
		}

		if _, found := config.Clients["global"][ip]; !found {
			config.Clients["global"][ip] = &config.Client{
				Limiter: rate.NewLimiter(config.GlobalRateLimit.Rate, config.GlobalRateLimit.Burst),
			}
		}
		config.Cooldowns.MU.Unlock()

		client := config.Clients["global"][ip]
		if !client.Limiter.Allow() {
			cooldownDuration := config.GlobalRateLimit.Cooldown
			if cooldownDuration == 0 {
				cooldownDuration = config.Cooldowns.DefaultWaitTime
			}

			config.Cooldowns.MU.Lock()
			config.Cooldowns.Client["global"][ip] = time.Now().Add(cooldownDuration)
			config.Cooldowns.MU.Unlock()

			w.Header().Set("Retry-After", time.Now().Add(cooldownDuration).Format(time.RFC1123))
			http.Error(w, "too many requests 💔", http.StatusTooManyRequests)
			return
		}

		client.LastRequest = time.Now()

		next.ServeHTTP(w, r)
	})
}
