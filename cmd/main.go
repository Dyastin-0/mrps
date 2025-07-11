package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dyastin-0/mrps/internal/api"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/metrics"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/Dyastin-0/mrps/internal/config"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(500 * time.Millisecond)
	}()

	configPath := flag.String("config", "mrps.yaml", "Path to the config file")
	flag.Parse()

	log.Info().Str("path", *configPath).Msg("config")

	err := config.Load(ctx, *configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("config")
	}

	err = godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("env")
	}

	config.StartTime = time.Now()

	logger.Init()

	// go config.Watch(ctx, *configPath)
	go health.InitBroadcaster(ctx)
	go logger.InitNotifier(ctx)
	// go router.Start(ctx)

	if config.Misc.MetricsEnabled {
		go metrics.Start()
	}

	if config.Misc.APIEnabled {
		go ws.Clients.Start(ctx)
		go api.Start()
	}

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Info().Msg("shutting down gracefully...")
}
