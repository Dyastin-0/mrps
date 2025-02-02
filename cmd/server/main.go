package main

import (
	"context"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/api"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/router"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"os"
	"os/signal"
	"syscall"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	err := config.Load("mrps.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("ENV")
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = string(config.Misc.Email)
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", router.New())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config.StartTime = time.Now()

	logger.Init()
	go health.InitPinger(ctx)
	go logger.InitNotifier(ctx)

	// go startReverseProxyServer(mainRouter)

	if config.Misc.MetricsEnabled {
		go startMetricsServer()
	}

	if config.Misc.APIEnabled {
		go startAPI()
	}

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Info().Msg("Shutting down gracefully...")
}

func startReverseProxyServer(router chi.Router) {
	log.Info().Str("Status", "running").Msg("Proxy")

	err := certmagic.HTTPS(config.Domains, router)
	if err != nil {
		log.Fatal().Err(err).Msg("Proxy")
	}
}

func startMetricsServer() {
	metricsRouter := chi.NewRouter()

	metricsRouter.Handle("/metrics", promhttp.Handler())

	log.Info().Str("Status", "running").Str("Port", config.Misc.MetricsPort).Msg("Metrics")
	err := http.ListenAndServe(":"+config.Misc.MetricsPort, metricsRouter)
	if err != nil {
		log.Fatal().Err(err).Msg("Metrics")
	}
}

func startAPI() {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(api.CORS)

	router.Mount("/config", api.ProtectedRoute())
	router.Handle("/refresh", api.Refresh())
	router.Handle("/signout", api.Signout())
	router.Handle("/auth", api.Auth())
	router.Get("/ws", ws.WS(&health.Subscribers, &logger.Subscribers))

	log.Info().Str("Status", "running").Str("Port", config.Misc.MetricsPort).Msg("API")
	err := http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal().Err(err).Msg("API")
	}
}
