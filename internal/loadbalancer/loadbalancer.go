package loadbalancer

import (
	"context"
	"fmt"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/iphash"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/wrr"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

func new(
	btype string,
	ctx context.Context,
	dests []types.Dest,
	rewriteRule rewriter.RewriteRule,
	path, host string,
	healthCheckInterval time.Duration,
) (common.Balancer, error) {
	switch btype {
	case "rr", "":
		return rr.New(ctx, dests, rewriteRule, path, host, healthCheckInterval), nil
	case "wrr":
		return wrr.New(ctx, dests, rewriteRule, path, host, healthCheckInterval), nil
	case "ih":
		return iphash.New(ctx, dests, rewriteRule, path, host, healthCheckInterval), nil
	default:
		return nil, fmt.Errorf("unsupported balancer type: %s", btype)
	}
}

func New(
	ctx context.Context,
	dests []types.Dest,
	rewriteRule rewriter.RewriteRule,
	proto, btype, path, host string,
	healthCheckInterval time.Duration,
) (common.Balancer, error) {
	return new(btype, ctx, dests, rewriteRule, path, host, healthCheckInterval)
}

func NewTCP(
	btype string,
	ctx context.Context,
	dests []types.Dest,
	healthCheckInterval time.Duration,
) (common.BalancerTCP, error) {
	switch btype {
	case "ih", "":
		return iphash.NewTCP(ctx, dests, healthCheckInterval), nil

	default:
		return nil, fmt.Errorf("unsupported balancer type: %s", btype)
	}
}
