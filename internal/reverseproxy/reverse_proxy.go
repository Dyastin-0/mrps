package reverseproxy

import (
	"log"
	"net/http"
	"strings"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	reverseproxy "github.com/Dyastin-0/reverse-proxy-server/pkg/reverse_proxy"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		path := r.URL.Path

		if domainConfig, exists := config.Routes[host]; exists {
			for routePath, proxyTarget := range domainConfig.Routes {
				if strings.HasPrefix(path, routePath) {
					log.Println("[DEBUG] Proxying request to", proxyTarget)
					r.URL.Path = strings.TrimPrefix(r.URL.Path, routePath)
					reverseproxy.New(proxyTarget).ServeHTTP(w, r)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
