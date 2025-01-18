package main

import (
	"log"
	"os"

	"github.com/Dyastin-0/reverse-proxy-server/internal/router"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file: ", err)
	}

	err = config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatal("Failed to load config file: ", err)
	}

	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = os.Getenv("CERT_EMAIL")
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", router.New())

	log.Println("Reverse proxy server is running on HTTPS")

	//Cofigure domains at internal/config/domain.go
	err = certmagic.HTTPS(config.Domains, mainRouter)
	if err != nil {
		log.Fatal("Failed to start HTTPS server: ", err)
	}
}
