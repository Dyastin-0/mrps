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
		log.Println("====================================")
		if domainConfig, exists := config.Routes[host]; exists {
			for routePath, proxyTarget := range domainConfig.Routes {
				log.Println("[DEBUG] Current path", routePath)
				if strings.HasPrefix(path, routePath) {
					log.Println("[DEBUG] matched path", routePath, "Request path", path)
					log.Println("[DEBUG] Proxying request to", proxyTarget)
					reverseproxy.New(proxyTarget).ServeHTTP(w, r)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
