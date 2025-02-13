package metrics

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Dyastin-0/mrps/internal/config"
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
)

func init() {
	prometheus.MustRegister(RequestCount)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveRequests)
}

func Handler() http.HandlerFunc {
	return promhttp.Handler().ServeHTTP
}

func NewWrapResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}
}

func (w *ResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (rw *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("ResponseWriter does not support Hijack")
}

func UpdateHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := r.Method
		host := r.Host

		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		rec := NewWrapResponseWriter(w)

		next.ServeHTTP(rec, r)

		code := fmt.Sprint(rec.StatusCode)
		duration := time.Since(start).Seconds()

		mu.Lock()
		RequestCount.WithLabelValues(method, host, code).Inc()
		RequestDuration.WithLabelValues(method, host, code).Observe(duration)
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
