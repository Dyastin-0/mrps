package rr

import (
	"context"
	"sync"
	"time"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
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

func New(ctx context.Context, dests []string, path string, host string, rewriteRule rewriter.RewriteRule) *RR {
	context, cancel := context.WithCancel(ctx)

	rr := &RR{
		Dests:  make([]*lbcommon.Dest, len(dests)),
		cancel: cancel,
	}

	for idx, dst := range dests {
		newDest := &lbcommon.Dest{URL: dst}
		go newDest.Check(context, host, 10*time.Second)
		newDest.Proxy = reverseproxy.New(dst, path, rewriteRule)
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

func (rr *RR) Next() *lbcommon.Dest {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	if len(rr.Dests) == 0 {
		return nil
	}

	dest := rr.Dests[rr.index]
	rr.index = (rr.index + 1) % len(rr.Dests)

	return dest
}

func (rr *RR) GetDests() interface{} {
	return rr.Dests
}
