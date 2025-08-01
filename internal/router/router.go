package router

import (
	"context"
	nhttp "net/http"
	"os"

	"github.com/Dyastin-0/mrps/internal/allowedhost"
	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/limiter"
	"github.com/Dyastin-0/mrps/internal/logger"
	"github.com/Dyastin-0/mrps/internal/metrics"
	"github.com/Dyastin-0/mrps/internal/reverseproxy"
	"github.com/Dyastin-0/mrps/internal/routelimiter"
	"github.com/Dyastin-0/mrps/internal/tls"
	"github.com/caddyserver/certmagic"
	"github.com/go-chi/chi/v5"
	cf "github.com/libdns/cloudflare"
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
	apiToken := os.Getenv("CLOUDFLARE_API_TOKEN")
	if apiToken == "" {
		log.Fatal().Msg("CLOUDFLARE_API_TOKEN environment variable is required")
	}

	provider := &cf.Provider{
		APIToken: apiToken,
	}

	certmagic.DefaultACME.Email = config.Misc.Email
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.DisableHTTPChallenge = true
	certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
	certmagic.DefaultACME.DNS01Solver = &certmagic.DNS01Solver{
		DNSManager: certmagic.DNSManager{
			DNSProvider: provider,
		},
	}

	magic := certmagic.NewDefault()

	err := magic.ManageSync(ctx, config.Domains)
	if err != nil {
		log.Warn().Err(err).Msg("failed to obtain certificates")
	}

	httpsServer := &nhttp.Server{
		Addr:      ":443",
		TLSConfig: magic.TLSConfig(),
		Handler:   httpsRouter(),
	}

	go func() {
		<-ctx.Done()
		httpsServer.Shutdown(context.Background())
	}()

	log.Info().Str("status", "listening").Msg("https")
	err = httpsServer.ListenAndServeTLS("", "")
	if err != nil {
		log.Fatal().Err(err).Msg("https")
	}
}

func startHTTP(ctx context.Context) {
	httpServer := &nhttp.Server{
		Addr:    ":80",
		Handler: httpRouter(),
	}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	log.Info().Str("status", "listening").Msg("http")
	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal().Err(err).Msg("http")
	}
}

func startTLS(ctx context.Context) {
	log.Info().Str("status", "listening").Msg("tcp")

	s := tls.New(":8443", config.Misc.Domain)

	err := s.Start(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("tcp")
	}
}

func Start(ctx context.Context) {
	go startHTTPS(ctx)
	go startTLS(ctx)
	if config.Misc.AllowHTTP {
		go startHTTP(ctx)
	}
}
