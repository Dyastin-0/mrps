package routelimiter

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"golang.org/x/time/rate"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		routeConfigPtr := config.DomainTrie.Match(host)
		routeConfig := *routeConfigPtr

		// If the rate limit is not set, assume there is no rate limit
		if routeConfig.RateLimit.Burst == 0 || routeConfig.RateLimit.Rate == 0 {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		if config.Cooldowns.DomainMutex[host] == nil {
			config.Cooldowns.DomainMutex[host] = &sync.Mutex{}
		}

		config.Cooldowns.DomainMutex[host].Lock()
		cooldownEnd, inCooldown := config.Cooldowns.Client[host][ip]
		config.Cooldowns.DomainMutex[host].Unlock()

		if inCooldown && time.Now().Before(cooldownEnd) {
			w.Header().Set("Retry-After", cooldownEnd.Format(time.RFC1123))
			http.Error(w, "too many requests 💔⏳", http.StatusTooManyRequests)
			return
		}

		if config.Clients[host] == nil {
			config.Clients[host] = make(map[string]*config.Client)
		}

		if _, found := config.Clients[host][ip]; !found {
			config.Clients[host][ip] = &config.Client{
				Limiter: rate.NewLimiter(routeConfig.RateLimit.Rate, routeConfig.RateLimit.Burst),
			}
		}

		if !config.Clients[host][ip].Limiter.Allow() {
			cooldownDuration := routeConfig.RateLimit.Cooldown
			if cooldownDuration == 0 {
				// If the cooldown is not set, use the default cooldown
				// There is no interface for the default cooldown in the config
				// Set the route's cooldown in the config to override the default cooldown
				cooldownDuration = routeConfig.RateLimit.DefaultCooldown
			}

			if config.Cooldowns.Client[host] == nil {
				config.Cooldowns.Client[host] = make(map[string]time.Time)
			}

			config.Cooldowns.DomainMutex[host].Lock()
			config.Cooldowns.Client[host][ip] = time.Now().Add(cooldownDuration)
			config.Cooldowns.DomainMutex[host].Unlock()

			w.Header().Set("Retry-After", time.Now().Add(cooldownDuration).Format(time.RFC1123))
			http.Error(w, "too many requests 💔", http.StatusTooManyRequests)
			return
		}

		config.Clients[host][ip].LastRequest = time.Now()

		next.ServeHTTP(w, r)
	})
}
