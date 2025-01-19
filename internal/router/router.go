package router

import (
	"net/http"

	"github.com/Dyastin-0/reverse-proxy-server/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func New() chi.Router {
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.RateLimiter)
	router.Use(middleware.ReverseProxy)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, from reverse proxy server 🚀"))
	})

	return router
}
