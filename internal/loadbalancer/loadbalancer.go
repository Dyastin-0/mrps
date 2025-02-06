package loadbalancer

import (
	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/types"
)

type balancer interface {
	Next() *types.Dest
}

func New(pathConfig *common.PathConfig) balancer {

	return rr.NewRR(pathConfig)
}
