package common

import (
	"context"
	"net"
	"net/http"
	"time"

	httputil "github.com/Dyastin-0/mrps/internal/http"
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
		log.Warn().Str("host", host).Str("url", d.URL).Str("proto", "tcp").Str("status", "down").Msg("health")
		d.Alive = false
	} else {
		conn.Close()
		d.Alive = true
		log.Warn().Str("host", host).Str("url", d.URL).Str("proto", "tcp").Str("status", "up").Msg("health")
	}
}

func (d *Dest) ping(host string) {
	resp, err := httputil.Client.Get(d.URL)
	if err != nil {
		log.Warn().Str("host", host).Str("url", d.URL).Str("proto", "http").Str("status", "down").Msg("health")
		d.Alive = false
	} else {
		resp.Body.Close()
		log.Info().Str("host", host).Str("url", d.URL).Str("proto", "http").Str("status", "up").Msg("health")
		d.Alive = true
	}
}
