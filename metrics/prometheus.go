package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// TotalRequests counts total HTTP requests
	TotalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// RequestDuration measures request latency
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// RequestsInFlight tracks currently active requests
	RequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	// ErrorsTotal counts total errors
	ErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_errors_total",
			Help: "Total number of HTTP errors",
		},
		[]string{"method", "endpoint", "error_type"},
	)

	// RateLimitHits counts rate limit hits
	RateLimitHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_hits_total",
			Help: "Total number of rate limit hits",
		},
	)

	// ActiveUsers tracks number of users in the system
	ActiveUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users_active_total",
			Help: "Total number of active users in the system",
		},
	)
)

func init() {
	// Register all metrics with Prometheus
	prometheus.MustRegister(TotalRequests)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(RequestsInFlight)
	prometheus.MustRegister(ErrorsTotal)
	prometheus.MustRegister(RateLimitHits)
	prometheus.MustRegister(ActiveUsers)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// MetricsMiddleware is a middleware that collects Prometheus metrics
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint itself
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		RequestsInFlight.Inc()

		// Wrap response writer to capture status code
		wrapped := newResponseWriter(w)

		// Get route pattern for consistent labeling
		route := mux.CurrentRoute(r)
		path := r.URL.Path
		if route != nil {
			if tpl, err := route.GetPathTemplate(); err == nil {
				path = tpl
			}
		}

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(wrapped.statusCode)

		TotalRequests.WithLabelValues(r.Method, path, status).Inc()
		RequestDuration.WithLabelValues(r.Method, path).Observe(duration)
		RequestsInFlight.Dec()

		// Track errors
		if wrapped.statusCode >= 400 {
			errorType := "client_error"
			if wrapped.statusCode >= 500 {
				errorType = "server_error"
			}
			ErrorsTotal.WithLabelValues(r.Method, path, errorType).Inc()
		}
	})
}

// Handler returns the Prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}

// IncrementRateLimitHits increments the rate limit hit counter
func IncrementRateLimitHits() {
	RateLimitHits.Inc()
}

// SetActiveUsers sets the number of active users
func SetActiveUsers(count float64) {
	ActiveUsers.Set(count)
}
