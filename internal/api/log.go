package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/go-chi/chi/v5"
)

func getUptime() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config.StartTime.Unix())
	}
}

func getHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		data := struct {
			Type   string                     `json:"type"`
			Health map[string]map[string]bool `json:"health"`
		}{
			Type:   "health",
			Health: config.DomainTrie.GetHealth(),
		}

		health.Subscribers.Store(token, true)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(&data)
	}
}

func getLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		token = token[7:]

		readyChan := make(chan bool)

		logger.LeftBehind.Store(token, readyChan)

		go logger.CatchUp(token, readyChan)

		w.WriteHeader(http.StatusOK)

		retry := 20
		ok := false
		for retry > 0 {
			if ok = ws.Clients.Exists(token); ok {
				break
			}
			retry--
			time.Sleep(50 * time.Millisecond)
		}
		readyChan <- ok
	}
}

func logRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(jwt)

	router.Handle("/uptime", getUptime())
	router.Handle("/health/ws", getHealth())
	router.Handle("/logs/ws", getLogs())

	return router
}
