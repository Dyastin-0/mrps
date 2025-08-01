package health

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/rs/zerolog/log"
)

var Subscribers = sync.Map{}

func InitBroadcaster(ctx context.Context) {
	log.Info().Str("status", "running").Msg("health")

	ticker := time.NewTicker(time.Duration(config.Misc.HealthCheckInterval) * time.Millisecond)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Info().Str("status", "stopping").Msg("health")
				return
			case <-ticker.C:
				broadcastHealthData()
			}
		}
	}()
}

func broadcastHealthData() {
	data := struct {
		Type   string                     `json:"type"`
		Health map[string]map[string]bool `json:"health"`
	}{
		Type:   "health",
		Health: config.DomainTrie.GetHealth(),
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
