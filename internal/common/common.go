package common

import (
	"net"
	"net/http"

	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
)

type Balancer interface {
	Serve(w http.ResponseWriter, r *http.Request, retries int) bool
	First() *lbcommon.Dest
	GetDests() []*lbcommon.Dest
	StopHealthChecks()
}

type BalancerTCP interface {
	Serve(conn net.Conn, sni string) bool
	First() *lbcommon.Dest
	GetDests() []*lbcommon.Dest
	StopHealthChecks()
}
