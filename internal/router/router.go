package router

import (
	"context"
	nhttp "net/http"
	"strings"

	"github.com/Dyastin-0/mrps/internal/allowedhost"
	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/limiter"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/metrics"
	"github.com/Dyastin-0/mrps/internal/reverseproxy"
	"github.com/Dyastin-0/mrps/internal/routelimiter"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

func httpsRouter() *chi.Mux {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(metrics.UpdateHandler)
	router.Use(allowedhost.Handler)
	router.Use(limiter.Handler)
	router.Use(routelimiter.Handler)
	router.Use(reverseproxy.Handler)

	router.Get("/", func(w nhttp.ResponseWriter, r *nhttp.Request) {
		w.Write([]byte("Hello, mrps https 🚀\n"))
	})

	return router
}

func httpRouter() *chi.Mux {
	router := chi.NewRouter()

	router.Use(logger.Handler)
	router.Use(metrics.UpdateHandler)
	router.Use(limiter.Handler)
	router.Use(routelimiter.Handler)
	router.Use(reverseproxy.HTTPHandler)

	router.Get("/", func(w nhttp.ResponseWriter, r *nhttp.Request) {
		w.Write([]byte("Hello, from mrps http 🚀\n"))
	})

	return router
}

func startHTTPS(ctx context.Context) {
	magic := certmagic.NewDefault()

	if config.Misc.Email != "" {
		certmagic.DefaultACME.Email = config.Misc.Email
	}
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA

	err := magic.ManageSync(ctx, config.Domains)
	if err != nil && strings.Contains(err.Error(), "too many failed authorizations") {
		log.Fatal().Err(err).Msg("https")
	} else {
		log.Warn().Err(err).Msg("https")
	}

	httpsServer := &nhttp.Server{
		Addr:      ":443",
		TLSConfig: magic.TLSConfig(),
		Handler:   httpsRouter(),
	}

	log.Info().Str("status", "listening").Msg("https")
	err = httpsServer.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("https")
	}
}

func startHTTP() {
	log.Info().Str("status", "running").Msg("proxy")

	httpServer := &nhttp.Server{
		Addr:    ":80",
		Handler: httpRouter(),
	}

	log.Info().Str("status", "listening").Msg("http")
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("http")
	}
}

func Start(ctx context.Context) {
	go startHTTPS(ctx)
	if config.Misc.AllowHTTP {
		go startHTTP()
	}
}
