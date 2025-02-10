package common

import (
	"context"
	"net/http"
	"time"

	httputil "github.com/Dyastin-0/mrps/internal/http"
	"github.com/rs/zerolog/log"
)

type Dest struct {
	URL   string
	Alive bool
	Proxy http.Handler `yaml:"-" json:"-"`
}

func (d *Dest) Check(ctx context.Context, host string, delay time.Duration) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Str("host", host).Str("url", d.URL).Str("Status", "Stopping").Msg("health")
			return
		case <-ticker.C:
			d.ping(host)
		}
	}
}

func (d *Dest) ping(host string) {
	resp, err := httputil.Client.Get(d.URL)
	if err != nil {
		log.Warn().Str("host", host).Str("url", d.URL).Str("status", "down").Msg("health")
		d.Alive = false
	} else {
		resp.Body.Close()
		log.Info().Str("host", host).Str("url", d.URL).Str("status", "up").Msg("health")
		d.Alive = true
	}
}
