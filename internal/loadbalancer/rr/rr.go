package rr

import (
	"sync"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/rs/zerolog/log"
)

type RR struct {
	Dests []*lbcommon.Dest
	index int
	mu    sync.Mutex
}

func New(dests []string, path string) *RR {
	rr := &RR{
		Dests: make([]*lbcommon.Dest, len(dests)),
	}

	for idx, dst := range dests {
		newDest := &lbcommon.Dest{URL: dst}
		if _, err := lbcommon.Check(dst); err != nil {
			newDest.Alive = false
			log.Warn().Str("path", path).Str("dest", dst).Bool("alive", false).Msg("balancer")
		} else {
			newDest.Alive = true
		}
		rr.Dests[idx] = newDest
	}

	log.Info().Str("path", path).Str("status", "initialized").Int("count", len(rr.Dests)).Msg("balancer")
	return rr
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
