package main

import (
	"log"
	"net/http"

	"github.com/Dyastin-0/mrps/internal/router"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"

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

	log.Println("Metrics service is running on port:" + config.Misc.MetricsPort)
	err := http.ListenAndServe(":"+config.Misc.MetricsPort, metricsRouter)
	if err != nil {
		log.Fatal("Failed to start metrics server: ", err)
	}
}
