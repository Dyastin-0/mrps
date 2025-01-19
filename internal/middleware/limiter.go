package middleware

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	"golang.org/x/time/rate"
)

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Error parsing IP: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		config.Cooldowns.MU.Lock()
		cooldownEnd, inCooldown := config.Cooldowns.Client[ip]
		config.Cooldowns.MU.Unlock()

		if inCooldown && time.Now().Before(cooldownEnd) {
			w.Header().Set("Retry-After", cooldownEnd.Format(time.RFC1123))
			http.Error(w, "too many requests 💔⏳", http.StatusTooManyRequests)
			return
		}

		if _, found := config.Clients[ip]; !found {
			config.Clients[ip] = &config.Client{
				Limiter: rate.NewLimiter(config.RateLimit.Rate, config.RateLimit.Burst),
			}
		}

		if !config.Clients[ip].Limiter.Allow() {
			//You can also use the default cooldown time: config.Cooldowns.DefaultWaitTime
			cooldownDuration := config.RateLimit.Cooldown

			config.Cooldowns.MU.Lock()
			config.Cooldowns.Client[ip] = time.Now().Add(cooldownDuration)
			config.Cooldowns.MU.Unlock()

			w.Header().Set("Retry-After", time.Now().Add(cooldownDuration).Format(time.RFC1123))
			http.Error(w, "too many requests 💔", http.StatusTooManyRequests)
			return
		}

		config.Clients[ip].LastRequest = time.Now()

		next.ServeHTTP(w, r)
	})
}
