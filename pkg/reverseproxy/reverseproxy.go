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
		req.URL.Path = targetURL.Path + rw.RewritePath(req.URL.Path)

		req.Header.Set("X-Forwarded-Host", targetURL.Host)
		req.Header.Set("X-Forwarded-Proto", targetURL.Scheme)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)

		req.Host = targetURL.Host
	}

	return proxy
}
