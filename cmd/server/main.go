package main

import (
	"log"
	"net/http"

	"github.com/Dyastin-0/reverse-proxy-server/internal/router"
	"github.com/go-chi/chi/v5"
)

func main() {
	mainRouter := chi.NewRouter()

	mainRouter.Mount("/", router.New())

	log.Println("Reverse proxy server is running on port 3000")
	if err := http.ListenAndServe(":3000", mainRouter); err != nil {
		log.Fatal("Server failed: ", err)
	}
}
