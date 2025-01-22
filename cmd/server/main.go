package main

import (
	"log"
	"net/http"

	"github.com/Dyastin-0/reverse-proxy-server/internal/router"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file: ", err)
	}

	err = config.Load("config.yaml")
	if err != nil {
		log.Fatal("Failed to load config file: ", err)
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = string(config.Email)
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", router.New())

	go startReverseProxyServer(mainRouter)

	go startMetricsServer()

	select {}
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

	log.Println("Metrics service is running on port 9090")
	err := http.ListenAndServe(":7070", metricsRouter)
	if err != nil {
		log.Fatal("Failed to start metrics server: ", err)
	}
}
