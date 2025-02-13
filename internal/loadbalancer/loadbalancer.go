package loadbalancer

import (
	"context"
	"errors"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

type Balancer interface {
	Next() *lbcommon.Dest
	NextAlive() *lbcommon.Dest
	First() *lbcommon.Dest
	GetDests() []*lbcommon.Dest
}

type Default interface {
}

func new(btype string, ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, path, host string) (Balancer, error) {
	switch btype {
	case "rr", "":
		return rr.New(ctx, dests, rewriteRule, path, host), nil
	default:
		return nil, errors.New("unsupported balancer type")
	}
}

func New(ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, btype, path, host string) (Balancer, error) {
	return new(btype, ctx, dests, rewriteRule, path, host)
}
