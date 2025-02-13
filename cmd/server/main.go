package main

import (
	"context"
	"flag"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(500 * time.Millisecond)
	}()

	configPath := *flag.String("config", "mrps.yaml", "Path to the config file")
	flag.Parse()

	log.Debug().Str("config", configPath).Msg("DEBUG")

	err := config.Load(ctx, configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("env")
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = string(config.Misc.Email)
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	config.StartTime = time.Now()

	logger.Init()

	go health.InitHealthBroadcaster(ctx)
	go logger.InitNotifier(ctx)
	go ws.Clients.Run(ctx)

	go startReverseProxyServer(ctx)

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

	log.Info().Msg("shutting down gracefully...")
}

func startReverseProxyServer(ctx context.Context) {
	log.Info().Str("status", "running").Msg("proxy")

	httpServer := &http.Server{
		Addr:    ":80",
		Handler: router.NewHTTP(),
	}

	go func() {
		log.Info().Str("status", "listening").Msg("http")
		err := httpServer.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("http")
		}
	}()

	if len(config.Domains) != 0 {
		magic := certmagic.NewDefault()
		err := magic.ManageSync(ctx, config.Domains)
		if err != nil {
			log.Fatal().Err(err).Msg("https")
		}

		httpsServer := &http.Server{
			Addr:      ":443",
			TLSConfig: magic.TLSConfig(),
			Handler:   router.New(),
		}

		log.Info().Str("status", "listening").Msg("https")
		err = httpsServer.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal().Err(err).Msg("https")
		}
	}
}

func startMetricsServer() {
	metricsRouter := chi.NewRouter()

	metricsRouter.Handle("/metrics", promhttp.Handler())

	log.Info().Str("status", "running").Str("Port", config.Misc.MetricsPort).Msg("metrics")
	err := http.ListenAndServe(":"+config.Misc.MetricsPort, metricsRouter)
	if err != nil {
		log.Fatal().Err(err).Msg("metrics")
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
	router.Get("/ws", ws.WS(&health.Subscribers, &logger.Subscribers, &logger.LeftBehind))

	log.Info().Str("status", "running").Str("port", config.Misc.MetricsPort).Msg("api")
	err := http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal().Err(err).Msg("api")
	}
}
