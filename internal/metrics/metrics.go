package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
	"github.com/Dyastin-0/mrps/internal/hijack"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var mu sync.Mutex

var (
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "host", "code"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "host", "code"},
	)

	ActiveRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_requests",
			Help: "Number of active HTTP requests",
		},
	)

	ActiveSSHConns = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ssh_active_connections",
			Help: "Number of active SSH connections",
		},
	)

	ActiveWSConns = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "ws_active_connections",
			Help: "Number of active WS connections",
		},
	)
)

type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func init() {
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveRequests)
	prometheus.MustRegister(ActiveSSHConns)
	prometheus.MustRegister(ActiveWSConns)
}

func Handler() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

func UpdateHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := r.Method
		host := r.Host

		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		statusCode := hijack.StatusCode(next, w, r)

		duration := time.Since(start).Seconds()

		mu.Lock()
		RequestCount.WithLabelValues(method, host, fmt.Sprint(statusCode)).Inc()
		RequestDuration.WithLabelValues(method, host, fmt.Sprint(statusCode)).Observe(duration)
		mu.Unlock()
	})
}

func Start() {
	metricsRouter := chi.NewRouter()

	metricsRouter.Handle("/metrics", promhttp.Handler())

	log.Info().Str("status", "running").Str("Port", config.Misc.MetricsPort).Msg("metrics")
	err := http.ListenAndServe(":"+config.Misc.MetricsPort, metricsRouter)
	if err != nil {
		log.Fatal().Err(err).Msg("metrics")
	}
}
