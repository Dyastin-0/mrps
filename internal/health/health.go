package health

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/common"
	"github.com/Dyastin-0/mrps/internal/config"
	http "github.com/Dyastin-0/mrps/internal/http"
	lbcommon "github.com/Dyastin-0/mrps/internal/loadbalancer/common"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/rs/zerolog/log"
)

var Data = sync.Map{}
var Subscribers = sync.Map{}

func InitPinger(ctx context.Context) {
	log.Info().Str("status", "running").Msg("health")

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info().Str("status", "stopping").Msg("health")
				return
			case <-ticker.C:
				pingAll()
			}

		}
	}()
}

func pingAll() {
	wg := sync.WaitGroup{}
	Domains := config.DomainTrie.GetAll()

	for _, config := range Domains {
		wg.Add(1)
		go ping(&config, &wg)
	}

	wg.Wait()
	notifySubscribers()
}

func ping(config *common.Config, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, route := range config.Routes {
		destsI := route.Balancer.GetDests()
		dests, _ := destsI.([]*lbcommon.Dest)

		for _, dest := range dests {
			status, _ := Check(dest.URL)
			log.Info().Int("status", status).Str("url", dest.URL).Msg("health")
			Data.Store(dest.URL, status)
		}
	}
}

func notifySubscribers() {
	mapHealth := make(map[string]int)
	Data.Range(func(key, value interface{}) bool {
		mapHealth[key.(string)] = value.(int)
		return true
	})

	data := struct {
		Type   string         `json:"type"`
		Health map[string]int `json:"health"`
	}{
		Type:   "health",
		Health: mapHealth,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Fatal().Err(err).Msg("health")
		return
	}

	Subscribers.Range(func(key, value interface{}) bool {
		token := key.(string)
		go func() {
			ws.Clients.Send(token, dataBytes)
		}()
		return true
	})
}

func Check(url string) (int, error) {
	resp, err := http.Client.Get(url)
	if err != nil {
		log.Warn().Str("url", url).Str("status", "down").Msg("health")
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}
