package api

import (
	"encoding/json"
	"net/http"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/types"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func handleGet(w http.ResponseWriter, r *http.Request) {
	domains := config.DomainTrie.GetAll()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domains)
}

func handleSync(w http.ResponseWriter, r *http.Request) {
	config.ParseToYAML()
	w.WriteHeader(http.StatusAccepted)
}

type enableRequest struct {
	Enabled bool `json:"enabled"`
}

func handleEnable(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	token := r.Header.Get("Authorization")
	token = token[7:]

	decoder := json.NewDecoder(r.Body)
	req := enableRequest{}

	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ok := config.DomainTrie.SetEnabled(domain, req.Enabled)
	if !ok {
		status := "enabled"
		if !req.Enabled {
			status = "disabled"
		}
		http.Error(w, "Domain not modified, it is either not defined or already "+status, http.StatusNotFound)
		return
	}

	con := config.DomainTrie.GetAll()

	conf := struct {
		Type   string              `json:"type"`
		Config types.DomainsConfig `json:"config"`
	}{
		Type:   "config",
		Config: con,
	}

	configBytes, err := json.Marshal(conf)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		log.Error().Err(err).Msg("api")
		return
	}

	go ws.Clients.Send(token, configBytes)

	w.WriteHeader(http.StatusOK)
}

func configRoute() *chi.Mux {
	router := chi.NewRouter()

	router.Use(jwt)

	router.Get("/", handleGet)
	router.Post("/sync", handleSync)
	router.Post("/{domain}/enable", handleEnable)

	return router
}
