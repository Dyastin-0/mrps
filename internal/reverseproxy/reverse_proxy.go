package reverseproxy

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(r.Host)
		path := r.URL.Path

		configPtr := config.DomainTrie.Match(host)
		if configPtr != nil {
			config := *configPtr
			for _, routePath := range config.SortedRoutes {
				fmt.Println(routePath, path)
				fmt.Println(strings.HasPrefix(path, routePath))
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
