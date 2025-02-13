package reverseproxy

import (
	"net/http"
	"strings"

	"github.com/Dyastin-0/mrps/internal/config"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		path := r.URL.Path

		matchedConfig := config.DomainTrie.Match(host)
		if matchedConfig != nil {
			for _, routePath := range matchedConfig.SortedRoutes {
				if strings.HasPrefix(path, routePath) {
					if matchedConfig.Routes[routePath].BalancerType != "" {
						if dest := matchedConfig.Routes[routePath].Balancer.Next(); dest != nil {
							dest.Proxy.ServeHTTP(w, r)
							return
						}
					}
					if dest := matchedConfig.Routes[routePath].Balancer.First(); dest != nil {
						dest.Proxy.ServeHTTP(w, r)
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		for _, routePath := range config.HTTP.SortedRoutes {
			if strings.HasPrefix(path, routePath) {
				if config.HTTP.Routes[routePath].BalancerType != "" {
					if dest := config.HTTP.Routes[routePath].Balancer.Next(); dest != nil {
						dest.Proxy.ServeHTTP(w, r)
						return
					}
				}
				if dest := config.HTTP.Routes[routePath].Balancer.First(); dest != nil {
					dest.Proxy.ServeHTTP(w, r)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
