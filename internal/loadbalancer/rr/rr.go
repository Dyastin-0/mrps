package rr

import (
	"context"
	"net/http"
	"sync"
	"time"

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

func (rr *RR) Serve(r *http.Request) *lbcommon.Dest {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.Dests) == 0 {
		return nil
	}

	dest := rr.Dests[rr.index]
	rr.index = (rr.index + 1) % len(rr.Dests)

	return dest
}

func (rr *RR) ServeAlive(r *http.Request) *lbcommon.Dest {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.Dests) == 0 {
		return nil
	}

	startIndex := rr.index
	for {
		dest := rr.Serve(r)

		if dest.Alive {
			return dest
		}

		if rr.index == startIndex {
			break
		}
	}

	return nil
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
