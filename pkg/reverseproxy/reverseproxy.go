package reverseproxy

import (
	"fmt"
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
		fmt.Println("Path: ", req.URL.Path)
		rr := configPtr.Routes[req.URL.Path].RewriteRule
		rw := rewriter.New(rr)

		rewrittenPath := rw.RewritePath(req.URL.Path)
		fmt.Println("Rule: ", rr.ReplaceVal, rr.Value, rr.Type)
		fmt.Println("Original: ", req.URL.Path)
		fmt.Println("Rewritten: ", rewrittenPath)
		req.URL.Path = rewrittenPath

		req.Host = targetURL.Host
	}

	return proxy
}
