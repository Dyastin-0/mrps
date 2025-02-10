package loadbalancer

import (
	"context"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

type balancer interface {
	Next() *lbcommon.Dest
	GetDests() interface{}
}

func New(ctx context.Context, dests []string, path string, host string, rewriteRule rewriter.RewriteRule) balancer {
	return rr.New(ctx, dests, path, host, rewriteRule)
}
