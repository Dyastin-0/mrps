package reverseproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Dyastin-0/mrps/internal/config"
	reverseproxy "github.com/Dyastin-0/mrps/pkg/reverse_proxy"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		path := r.URL.Path

		configPtr := config.DomainTrie.Match(host)
		if configPtr != nil {
			config := *configPtr
			for _, routePath := range config.SortedRoutes {
				if strings.HasPrefix(path, routePath) {
					fmt.Println("Proxying to", routePath)
					proxyTarget := config.Routes[routePath]
					reverseproxy.New(proxyTarget).ServeHTTP(w, r)
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}
