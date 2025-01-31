package main

import (
	"log"
	"net/http"

	"github.com/Dyastin-0/mrps/internal/router"
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

	config.InitHealth()

	go startReverseProxyServer(mainRouter)
	go startMetricsServer()
	go startAPI()

	// Handle graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown // Wait for a signal

	log.Println("Shutting down gracefully...")

	// You can add any cleanup logic here if needed
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

	router.Use(config.CORS)

	router.Mount("/config", config.ProtectedRoute())
	router.Handle("/refresh", config.Refresh())
	router.Handle("/auth", config.Auth())
	router.Get("/ws", config.WS)

	log.Println("API service is running on port: " + config.Misc.ConfigAPIPort)
	err = http.ListenAndServe(":"+config.Misc.ConfigAPIPort, router)
	if err != nil {
		log.Fatal("Failed to start API server: ", err)
	}
}
