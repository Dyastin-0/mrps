package main

import (
	"log"

	"github.com/Dyastin-0/reverse-proxy-server/internal/router"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"

	"github.com/Dyastin-0/reverse-proxy-server/internal/config"
)

func main() {
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.Email = "mail@dyastin.tech"
	certmagic.Default.Storage = &certmagic.FileStorage{Path: "certs"}

	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", router.New())

	log.Println("Reverse proxy server is running on HTTPS")

	//Cofigure domain at internal/config/config.go
	err := certmagic.HTTPS(config.Domains, mainRouter)
	if err != nil {
		log.Fatal("Failed to start HTTPS server: ", err)
	}
}
