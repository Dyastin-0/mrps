package config

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

var HealthData = sync.Map{}
var HealthSubscribers = sync.Map{}

var httpClient = &http.Client{
	Timeout: 3 * time.Second,
}

func InitHealth(ctx context.Context) {
	log.Println("Health check is running")

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping health checks")
				return
			case <-ticker.C:
				pingAll()
			}

		}
	}()
}

func pingAll() {
	wg := sync.WaitGroup{}
	Domains := DomainTrie.GetAll()

	for _, config := range Domains {
		wg.Add(1)
		go ping(&config, &wg)
	}

	wg.Wait()
	notifySubscribers()
}

func ping(config *Config, wg *sync.WaitGroup) {
	defer wg.Done()
	for _, route := range config.Routes {
		resp, err := httpClient.Get(route.Dest)
		if err != nil {
			HealthData.Store(route.Dest, 0)
		} else {
			resp.Body.Close()
			HealthData.Store(route.Dest, resp.StatusCode)
		}
	}
}

func notifySubscribers() {
	mapHealth := make(map[string]int)
	HealthData.Range(func(key, value interface{}) bool {
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

	marshalHealth, err := json.Marshal(data)
	if err != nil {
		log.Println("Failed to marshal health:", err)
		return
	}

	HealthSubscribers.Range(func(key, value interface{}) bool {
		token := key.(string)
		go func() {
			err := SendData(token, marshalHealth)
			if err != nil {
				log.Println("Failed to send health data:", err)
				HealthSubscribers.Delete(token)
			}
		}()
		return true
	})
}
