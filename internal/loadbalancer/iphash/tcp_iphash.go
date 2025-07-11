package iphash

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/pkg/hash"
	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
)

type IPHashTCP struct {
	Dests  []*lbcommon.Dest
	mu     sync.Mutex
	cancel context.CancelFunc
}

func NewTCP(ctx context.Context, dests []types.Dest) common.BalancerTCP {
	healthctx, cancel := context.WithCancel(ctx)

	iptcp := &IPHashTCP{
		Dests:  make([]*lbcommon.Dest, len(dests)),
		cancel: cancel,
	}

	for idx, dst := range dests {
		newDest := &lbcommon.Dest{URL: dst.URL}
		go newDest.CheckTCP(healthctx, dst.URL, 10*time.Second)
		newDest.ProxyTCP = &reverseproxy.TCPProxy{
			Addr: dst.URL,
		}
		iptcp.Dests[idx] = newDest
	}

	return iptcp
}

func (ip *IPHashTCP) Serve(conn net.Conn) bool {
	if len(ip.Dests) == 0 {
		return false
	}

	ip.mu.Lock()
	defer ip.mu.Unlock()

	addr := conn.RemoteAddr().String()
	ipAddr, _, _ := net.SplitHostPort(addr)
	hash := hash.FNV(ipAddr)
	index := int(hash) % len(ip.Dests)

	dest := ip.Dests[index]

	dest.ProxyTCP.Forward(conn)

	return true
}

func (ip *IPHashTCP) First() *lbcommon.Dest {
	ip.mu.Lock()
	defer ip.mu.Unlock()

	if len(ip.Dests) == 0 {
		return nil
	}

	return ip.Dests[0]
}

func (ip *IPHashTCP) GetDests() []*lbcommon.Dest { return ip.Dests }

func (ip *IPHashTCP) StopHealthChecks() {
	if ip.cancel != nil {
		ip.cancel()
	}
}
