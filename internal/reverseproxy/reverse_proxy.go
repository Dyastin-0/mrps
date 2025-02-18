package reverseproxy

import (
	"net/http"
	"strings"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/types"
)

func routeAndServe(routes types.RouteConfig, sortedRoutes []string, w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path

	for _, routePath := range sortedRoutes {
		if strings.HasPrefix(path, routePath) {
			route := routes[routePath]

			if route.BalancerType != "" {
				if served := route.Balancer.Serve(w, r, len(sortedRoutes)); served {
					return true
				}
			}

			if dest := route.Balancer.First(); dest != nil {
				dest.Proxy.ServeHTTP(w, r)
				return true
			}
		}
	}

	return false
}

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		if matchedConfig := config.DomainTrie.Match(host); matchedConfig != nil {
			if routeAndServe(matchedConfig.Routes, matchedConfig.SortedRoutes, w, r) {
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		if dest := config.DomainTrie.Match(host); dest != nil {
			http.Redirect(w, r, "https://"+host, http.StatusPermanentRedirect)
			return
		}
		if routeAndServe(config.HTTP.Routes, config.HTTP.SortedRoutes, w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
}
