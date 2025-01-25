package reverseproxy

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Dyastin-0/mrps/internal/config"
)

func New(target string) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	domainConfigPtr := config.DomainTrie.Match(targetURL.Host)

	if domainConfigPtr == nil {
		log.Fatalf("No domain config found for %s", targetURL.Host)
		return nil
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path + req.URL.Path

		req.Header.Set("X-Forwarded-Host", targetURL.Host)
		req.Header.Set("X-Forwarded-Proto", targetURL.Scheme)
		req.Header.Set("X-Forwarded-For", req.RemoteAddr)

		req.Host = targetURL.Host
	}

	return proxy
}
