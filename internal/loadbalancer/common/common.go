package common

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/pkg/reverseproxy"
	"github.com/rs/zerolog/log"
)

type Dest struct {
	URL           string
	Alive         bool
	Weight        int
	CurrentWeight int
	Proxy         http.Handler           `yaml:"-" json:"-"`
	ProxyTCP      *reverseproxy.TCPProxy `yaml:"-" json:"-"`
}

var httpClient *http.Client = &http.Client{
	Timeout: 500 * time.Millisecond,
}

func (d *Dest) Check(ctx context.Context, host string, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	log.Info().Str("host", host).Str("url", d.URL).Str("proto", "http").Str("status", "running").Msg("health")

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("host", host).Str("url", d.URL).Str("proto", "http").Str("status", "stopping").Msg("health")
			return

		case <-ticker.C:
			d.ping(host)
		}
	}
}

func (d *Dest) CheckTCP(ctx context.Context, host string, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	log.Info().Str("host", host).Str("url", d.URL).Str("proto", "tcp").Str("status", "running").Msg("health")

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("host", host).Str("url", d.URL).Str("proto", "tcp").Str("status", "stopping").Msg("health")
			return

		case <-ticker.C:
			d.pingTCP(host)
		}
	}
}

func (d *Dest) pingTCP(host string) {
	conn, err := net.DialTimeout("tcp", host, time.Second)
	if err != nil {
		d.Alive = false
	} else {
		conn.Close()
		d.Alive = true
	}
}

func (d *Dest) ping(host string) {
	resp, err := httpClient.Get(d.URL)
	if err != nil {
		d.Alive = false
	} else {
		resp.Body.Close()
		d.Alive = true
	}
}
