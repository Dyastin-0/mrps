package loadbalancer

import (
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

type balancer interface {
	Next() *lbcommon.Dest
	GetDests() interface{}
}

func New(dests []string, path string, rewriteRule rewriter.RewriteRule) balancer {
	return rr.New(dests, path, rewriteRule)
}
