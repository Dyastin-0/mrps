package api

import (
	"net/http"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func Start() {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(cors)

	router.Mount("/", authRoute())
	router.Mount("/config", configRoute())
	router.Mount("/ssh", sshRoute())
	router.Mount("/logs", logRoute())

	log.Info().Str("status", "running").Str("port", config.Misc.MetricsPort).Msg("api")
	err := http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal().Err(err).Msg("api")
	}
}
