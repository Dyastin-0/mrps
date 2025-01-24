package router

import (
	"net/http"

	"github.com/Dyastin-0/mrps/internal/limiter"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/metrics"
	"github.com/Dyastin-0/mrps/internal/reverseproxy"
	routelimiter "github.com/Dyastin-0/mrps/internal/route_limiter"
	"github.com/go-chi/chi/v5"
)

func New() chi.Router {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(metrics.UpdateHandler)
	router.Use(limiter.Handler)
	router.Use(routelimiter.Handler)
	router.Use(reverseproxy.Handler)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, from reverse proxy server 🚀\n"))
	})

	return router
}
