package metrics

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "host"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of request durations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "host"},
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

func UpdateHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		method := r.Method
		endpoint := r.URL.Path
		host := r.Host
		ActiveRequests.Inc()
		defer ActiveRequests.Dec()

		rec := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(rec, r)

		duration := time.Since(start).Seconds()
		RequestCount.WithLabelValues(method, endpoint, host).Inc()
		RequestDuration.WithLabelValues(method, endpoint, host).Observe(duration)
	})
}
