package loadbalancer

import (
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
)

type balancer interface {
	Next() *lbcommon.Dest
	GetDests() interface{}
}

func New(dests []string, path string) balancer {
	return rr.New(dests, path)
}
