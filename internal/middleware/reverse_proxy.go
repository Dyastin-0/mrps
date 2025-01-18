package middleware

import (
	"net/http"
	"strings"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	reverse "github.com/Dyastin-0/reverse-proxy-server/pkg/reverse_proxy"
)

func ReverseProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fullPath := r.Host + r.URL.Path

		for route, target := range config.Routes {
			if strings.HasPrefix(fullPath, route) {
				proxy := reverse.New(target)
				proxy.ServeHTTP(w, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
