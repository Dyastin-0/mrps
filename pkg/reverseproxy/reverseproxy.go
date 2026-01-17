package reverseproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/rs/zerolog/log"
)

func New(target string, rr rewriter.RewriteRule) http.Handler {
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Fatal().Err(err).Msg("proxy")
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

		rw := rewriter.New(rr)

		rewrittenPath := rw.RewritePath(req.URL.Path)

		req.URL.Path = rewrittenPath

		if req.Header.Get("Upgrade") != "" {
			req.Header.Set("Connection", "Upgrade")
			req.Header.Set("Upgrade", "websocket")
		} else {
			req.Header.Set("Connection", "keep-alive")
		}

		req.Header.Set("X-Forwarded-For", req.RemoteAddr)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-Ip", req.RemoteAddr)
	}

	return proxy
}
