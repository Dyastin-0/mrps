package config

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

var RouteHealth = sync.Map{}
var HealthSubscribers = sync.Map{}

func InitHealth() {
	log.Println("Health check is running")

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			checkAll()
			notifySubscribers()
		}
	}()
}

func checkAll() {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	Domains := DomainTrie.GetAll()

	for _, config := range Domains {
		watchHealth(client, &config)
	}
}

func watchHealth(client *http.Client, config *Config) {
	for _, route := range config.Routes {
		resp, err := client.Get(route.Dest)
		if err != nil {
			RouteHealth.Store(route.Dest, 0)
		} else {
			resp.Body.Close()
			RouteHealth.Store(route.Dest, resp.StatusCode)
		}
	}
}

func notifySubscribers() {
	mapHealth := make(map[string]int)
	RouteHealth.Range(func(key, value interface{}) bool {
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
		func() {
			err := SendData(token, marshalHealth)
			if err != nil {
				log.Println("Failed to send health data:", err)
				HealthSubscribers.Delete(token)
			}
		}()
		return true
	})
}
