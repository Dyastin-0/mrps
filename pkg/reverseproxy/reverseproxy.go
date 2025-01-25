package reverseproxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

func New(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		configPtr := *config.DomainTrie.Match(req.Host)
		rr := configPtr.Routes[req.URL.Path].RewriteRule
		rw := rewriter.New(rr)
		req.URL.Path = rw.RewritePath(targetURL.Path)

		req.Host = targetURL.Host
	}

	return proxy
}
