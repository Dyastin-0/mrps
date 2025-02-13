package iphash

import (
	"net"
	"net/http"
	"sync"
	"time"

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

	rr := &IPHash{
		Dests:  make([]*lbcommon.Dest, len(dests)),
		cancel: cancel,
	}

	for idx, dst := range dests {
		newDest := &lbcommon.Dest{URL: dst.URL}
		go newDest.Check(context, host, 10*time.Second)
		newDest.Proxy = reverseproxy.New(dst.URL, path, rewriteRule)
		rr.Dests[idx] = newDest
	}

	log.Info().Str("path", path).Str("status", "initialized").Int("count", len(rr.Dests)).Msg("balancer")
	return rr
}

func (ih *IPHash) Serve(r *http.Request) *lbcommon.Dest {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	if len(ih.Dests) == 0 {
		return nil
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	hash := hash.FNV(ip)
	index := int(hash) % len(ih.Dests)

	return ih.Dests[index]
}

func (ih *IPHash) ServeAlive(r *http.Request) *lbcommon.Dest {
	ih.mu.Lock()
	defer ih.mu.Unlock()

	if len(ih.Dests) == 0 {
		return nil
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	hash := hash.FNV(ip)
	startIndex := int(hash) % len(ih.Dests)

	for i := 0; i < len(ih.Dests); i++ {
		index := (startIndex + i) % len(ih.Dests)
		if ih.Dests[index].Alive {
			return ih.Dests[index]
		}
	}

	return nil
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

func (ih *IPHash) Stop() {
	if ih.cancel != nil {
		log.Info().Msg("Stopping all health checks")
		ih.cancel()
	}
}
