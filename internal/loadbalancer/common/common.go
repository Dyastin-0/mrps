package common

import (
	"net/http"

	httputil "github.com/Dyastin-0/mrps/internal/http"
	"github.com/rs/zerolog/log"
)

type Dest struct {
	URL   string
	Alive bool
	Proxy http.Handler `yaml:"-" json:"-"`
}

func Check(url string) (int, error) {
	resp, err := httputil.Client.Get(url)
	if err != nil {
		log.Warn().Str("url", url).Str("status", "down").Msg("health")
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}
