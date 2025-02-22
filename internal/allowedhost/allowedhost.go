package allowedhost

import (
	"net/http"

	"github.com/Dyastin-0/mrps/internal/config"
)

func Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host

		if config.DomainTrie.Match(host) == nil {
			http.Error(w, "forbidden, host not allowed", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})

}
