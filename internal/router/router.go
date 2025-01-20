package router

import (
	"net/http"

	"github.com/Dyastin-0/reverse-proxy-server/internal/limiter"
	"github.com/Dyastin-0/reverse-proxy-server/internal/logger"
	"github.com/Dyastin-0/reverse-proxy-server/internal/reverseproxy"
	"github.com/go-chi/chi/v5"
)

func New() chi.Router {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(limiter.Handler)
	router.Use(reverseproxy.Handler)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, from reverse proxy server 🚀\n"))
	})

	return router
}
