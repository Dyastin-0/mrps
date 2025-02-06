package rr

import (
	"sync"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/types"
	"github.com/rs/zerolog/log"
)

type RR struct {
	Dests map[uint32]*types.Dest
	index uint32
	mu    sync.Mutex
}

func NewRR(pathConfig *common.PathConfig) *RR {
	rr := &RR{
		Dests: make(map[uint32]*types.Dest),
	}

	for idx, dst := range pathConfig.Dests {
		newDst := &types.Dest{URL: dst}
		_, err := health.Check(dst)
		if err != nil {
			newDst.Alive = false
			log.Warn().Str("dest", dst).Bool("alive", false).Msg("balancer")
		} else {
			newDst.Alive = true
		}
		rr.Dests[uint32(idx)] = newDst
	}

	return rr
}

func (rr *RR) Next() *types.Dest {
	prev := rr.index
	rr.index = (prev + 1) % uint32(len(rr.Dests))
	return rr.Dests[prev]
}
