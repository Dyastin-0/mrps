package wrr

import (
	"context"
	"sync"
	"time"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
	"github.com/rs/zerolog/log"
)

type WRR struct {
	Dests       []*lbcommon.Dest
	mu          sync.Mutex
	cancel      context.CancelFunc
	totalWeight int
}

func New(ctx context.Context, dests map[string]int, rewriteRule rewriter.RewriteRule, path, host string) *WRR {
	context, cancel := context.WithCancel(ctx)

	wrr := &WRR{
		Dests:  make([]*lbcommon.Dest, 0, len(dests)),
		cancel: cancel,
	}

	for dst, weight := range dests {
		newDest := &lbcommon.Dest{URL: dst, Weight: weight, CurrentWeight: 0}
		go newDest.Check(context, host, 10*time.Second)
		newDest.Proxy = reverseproxy.New(dst, path, rewriteRule)
		wrr.Dests = append(wrr.Dests, newDest)
		wrr.totalWeight += weight
	}

	log.Info().Str("path", path).Str("status", "initialized").Int("count", len(wrr.Dests)).Msg("balancer")
	return wrr
}

func (wrr *WRR) Stop() {
	if wrr.cancel != nil {
		log.Info().Msg("Stopping all health checks")
		wrr.cancel()
	}
}

func (wrr *WRR) Next() *lbcommon.Dest {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.Dests) == 0 {
		return nil
	}

	bestIndex := -1
	for i, dest := range wrr.Dests {
		dest.CurrentWeight += dest.Weight
		if bestIndex == -1 || dest.CurrentWeight > wrr.Dests[bestIndex].CurrentWeight {
			bestIndex = i
		}
	}

	if bestIndex == -1 {
		return nil
	}

	selected := wrr.Dests[bestIndex]
	selected.CurrentWeight -= wrr.totalWeight
	return selected
}

func (wrr *WRR) NextAlive() *lbcommon.Dest {
	wrr.mu.Lock()
	defer wrr.mu.Unlock()

	if len(wrr.Dests) == 0 {
		return nil
	}

	startIndex := -1
	for i := 0; i < len(wrr.Dests); i++ {
		dest := wrr.Next()
		if startIndex == -1 {
			startIndex = dest.CurrentWeight
		}

		if dest.Alive {
			return dest
		}

		if dest.CurrentWeight == startIndex {
			break
		}
	}

	log.Warn().Msg("No healthy destinations available")
	return nil
}

func (wrr *WRR) GetDests() []*lbcommon.Dest { return wrr.Dests }
