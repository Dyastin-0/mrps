package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Dyastin-0/mrps/internal/api"
	"github.com/Dyastin-0/mrps/internal/health"
	"github.com/Dyastin-0/mrps/internal/router"
	"github.com/Dyastin-0/mrps/internal/ws"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"os"
	"os/signal"
	"syscall"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	err := config.Load("mrps.yaml")
	if err != nil {
		log.Fatal("Failed to load config file: ", err)
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = string(config.Misc.Email)
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", router.New())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	health.InitPinger(ctx)
	config.StartTime = time.Now()

	// go startReverseProxyServer(mainRouter)
	go startMetricsServer()
	go startAPI()

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	log.Println("Shutting down gracefully...")
}

func startReverseProxyServer(router chi.Router) {
	log.Println("Reverse proxy server is running on HTTPS")

	err := certmagic.HTTPS(config.Domains, router)
	if err != nil {
		log.Fatal("Failed to start HTTPS server: ", err)
	}
}

func startMetricsServer() {
	metricsRouter := chi.NewRouter()

	metricsRouter.Handle("/metrics", promhttp.Handler())

	log.Println("Metrics service is running on port:" + config.Misc.MetricsPort)
	err := http.ListenAndServe(":"+config.Misc.MetricsPort, metricsRouter)
	if err != nil {
		log.Fatal("Failed to start metrics server: ", err)
	}
}

func startAPI() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	router := chi.NewRouter()

	router.Use(api.CORS)

	router.Mount("/config", api.ProtectedRoute())
	router.Handle("/refresh", api.Refresh())
	router.Handle("/signout", api.Signout())
	router.Handle("/auth", api.Auth())
	router.Get("/ws", ws.WS(&health.Subscribers))

	log.Println("API service is running on port: " + config.Misc.ConfigAPIPort)
	err = http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal("Failed to start API server: ", err)
	}
}
