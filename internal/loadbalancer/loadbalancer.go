package loadbalancer

import (
	"context"
	"errors"
	"net/http"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/iphash"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/rr"
	"github.com/Dyastin-0/mrps/internal/loadbalancer/wrr"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/rewriter"
)

type Balancer interface {
	Serve(w http.ResponseWriter, r *http.Request) bool
	ServeAlive(w http.ResponseWriter, r *http.Request) bool
	First() *lbcommon.Dest
	GetDests() []*lbcommon.Dest
}

type Default interface {
}

func new(btype string, ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, path, host string) (Balancer, error) {
	switch btype {
	case "rr", "":
		return rr.New(ctx, dests, rewriteRule, path, host), nil
	case "wrr":
		return wrr.New(ctx, dests, rewriteRule, path, host), nil
	case "ih":
		return iphash.New(ctx, dests, rewriteRule, path, host), nil
	default:
		return nil, errors.New("unsupported balancer type")
	}
}

func New(ctx context.Context, dests []types.Dest, rewriteRule rewriter.RewriteRule, btype, path, host string) (Balancer, error) {
	return new(btype, ctx, dests, rewriteRule, path, host)
}
