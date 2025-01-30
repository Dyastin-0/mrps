package reverseproxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

func New(target string, path string) http.Handler {
	targetURL, err := url.Parse(target)

	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = transport
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		configPtr := config.DomainTrie.Match(req.Host)
		rr := configPtr.Routes[path].RewriteRule
		rw := rewriter.New(rr)

		rewrittenPath := rw.RewritePath(req.URL.Path)
		log.Println("Rule: ", rr.Value)
		log.Println("Rewritten path: ", rewrittenPath)
		req.URL.Path = rewrittenPath

		req.Host = targetURL.Host

		req.Header.Set("X-Forwared-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-IP", req.RemoteAddr)
		req.Header.Set("Connection", "keep-alive")
		req.Header.Set("Connection", "Upgrade")
		req.Header.Set("Upgrade", "websocket")
	}

	return proxy
}
