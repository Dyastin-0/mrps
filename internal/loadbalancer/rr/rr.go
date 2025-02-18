package rr

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/hijack"
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/rs/zerolog/log"
)

type RR struct {
	Dests  []*lbcommon.Dest
	index  int
	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, path, host string) *RR {
	context, cancel := context.WithCancel(ctx)

	rr := &RR{
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

func (rr *RR) Stop() {
	if rr.cancel != nil {
		log.Info().Msg("Stopping all health checks")
		rr.cancel()
	}
}

func (rr *RR) Serve(w http.ResponseWriter, r *http.Request, retries int) bool {
	if len(rr.Dests) == 0 || retries <= 0 {
		http.Error(w, "All backend servers are down", http.StatusBadGateway)
		return false
	}

	rr.mu.Lock()
	dest := rr.Dests[rr.index]
	rr.index = (rr.index + 1) % len(rr.Dests)
	rr.mu.Unlock()

	statusCode := hijack.StatusCode(dest.Proxy, w, r)

	if statusCode >= 500 {
		log.Printf("Server %s failed with status %d, retrying (%d retries left)...", dest.URL, statusCode, retries-1)
		return rr.Serve(w, r, retries-1)
	}

	return true
}

func (rr *RR) First() *lbcommon.Dest {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.Dests) == 0 {
		return nil
	}

	return rr.Dests[0]
}

func (rr *RR) GetDests() []*lbcommon.Dest { return rr.Dests }
