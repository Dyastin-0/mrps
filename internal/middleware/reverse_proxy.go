package middleware

import (
	"net/http"
	"strings"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	reverse "github.com/Dyastin-0/reverse-proxy-server/pkg/reverse_proxy"
)

func ReverseProxy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Configure prefixes at internal/config/config.go
		for prefix, target := range config.Prefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				proxy := reverse.New(target)
				proxy.ServeHTTP(w, r)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
