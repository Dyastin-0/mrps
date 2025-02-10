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

func InitHealthBroadcaster(ctx context.Context) {
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
				broadcastHealthData()
			}
		}
	}()
}

func broadcastHealthData() {
	healthData := config.DomainTrie.GetHealth()

	data := struct {
		Type   string                     `json:"type"`
		Health map[string]map[string]bool `json:"health"`
	}{
		Type:   "health",
		Health: healthData,
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
