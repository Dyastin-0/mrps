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
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var clients = make(map[string]*client)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Error parsing IP: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if _, found := clients[ip]; !found {
			clients[ip] = &client{
				limiter: rate.NewLimiter(config.RateLimit.Rate, config.RateLimit.Burst),
			}
		}

		if !clients[ip].limiter.Allow() {
			log.Printf("Rate limit exceeded for IP: %s", ip)
			http.Error(w, "too many request 💔", http.StatusTooManyRequests)
			return
		}

		clients[ip].lastSeen = time.Now()

		next.ServeHTTP(w, r)
	})
}
