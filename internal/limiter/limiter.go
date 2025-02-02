package limiter

import (
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/config"
	"golang.org/x/time/rate"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.GlobalRateLimit.Burst == 0 || config.GlobalRateLimit.Rate == 0 {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, _ := net.SplitHostPort(r.RemoteAddr)

		key := "global:" + ip
		value, exists := config.ClientMngr.Load(key)
		var client *common.ClientLimiter

		if !exists {
			client = &common.ClientLimiter{
				Limiter:  rate.NewLimiter(config.GlobalRateLimit.Rate, config.GlobalRateLimit.Burst),
				LastReq:  time.Now(),
				Cooldown: time.Now(),
			}
			config.ClientMngr.Store(key, client)
		} else {
			client = value.(*common.ClientLimiter)
		}

		if time.Now().Before(client.Cooldown) {
			w.Header().Set("Retry-After", client.Cooldown.Format(time.RFC1123))
			http.Error(w, "too many requests 💔⏳", http.StatusTooManyRequests)
			return
		}

		if !client.Limiter.Allow() {
			cooldownDuration := config.GlobalRateLimit.Cooldown
			if cooldownDuration == 0 {
				cooldownDuration = 60000
			}

			client.Cooldown = time.Now().Add(time.Duration(cooldownDuration) * time.Millisecond)

			config.ClientMngr.Store(key, client)

			w.Header().Set("Retry-After", client.Cooldown.Format(time.RFC1123))
			http.Error(w, "too many requests 💔", http.StatusTooManyRequests)
			return
		}

		client.LastReq = time.Now()
		config.ClientMngr.Store(key, client)

		next.ServeHTTP(w, r)
	})
}
