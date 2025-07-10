package iphash

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/hijack"
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/hash"
	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

type IPHash struct {
	Dests  []*lbcommon.Dest
	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, path, host string) *IPHash {
	context, cancel := context.WithCancel(ctx)

	ip := &IPHash{
		Dests:  make([]*lbcommon.Dest, len(dests)),
		cancel: cancel,
	}

	for idx, dst := range dests {
		newDest := &lbcommon.Dest{URL: dst.URL}
		go newDest.Check(context, host, 10*time.Second)
		newDest.Proxy = reverseproxy.New(dst.URL, path, rewriteRule)
		ip.Dests[idx] = newDest
	}

	log.Info().Str("path", path).Str("status", "initialized").Int("count", len(ip.Dests)).Msg("balancer")
	return ip
}

func (ih *IPHash) Serve(w http.ResponseWriter, r *http.Request, retries int) bool {
	if len(ih.Dests) == 0 {
		return false
	}

	ih.mu.Lock()
	defer ih.mu.Unlock()

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	hash := hash.FNV(ip)
	index := int(hash) % len(ih.Dests)

	dest := ih.Dests[index]

	statusCode := hijack.StatusCode(dest.Proxy, w, r)

	if statusCode >= 500 {
		ih.Serve(w, r, retries-1)
	}

	return true
}

func (ih *IPHash) First() *lbcommon.Dest {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	if len(ih.Dests) == 0 {
		return nil
	}

	return ih.Dests[0]
}

func (ih *IPHash) GetDests() []*lbcommon.Dest { return ih.Dests }

func (ih *IPHash) StopHealthChecks() {
	if ih.cancel != nil {
		log.Info().Str("balancer", "iphash").Str("status", "stopped").Msg("health")
		ih.cancel()
	}
}
