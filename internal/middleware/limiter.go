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

		if _, found := config.Clients[ip]; !found {
			config.Clients[ip] = &config.Client{
				Limiter: rate.NewLimiter(config.RateLimit.Rate, config.RateLimit.Burst),
			}
		}

		if !config.Clients[ip].Limiter.Allow() {
			log.Printf("Rate limit exceeded for IP: %s", ip)
			http.Error(w, "too many request 💔", http.StatusTooManyRequests)
			return
		}

		config.Clients[ip].LastRequest = time.Now()

		next.ServeHTTP(w, r)
	})
}
