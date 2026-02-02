package utils

import (
	"net/http"

	"golang.org/x/time/rate"
)

// RateLimiter wraps the rate.Limiter for HTTP middleware usage
type RateLimiter struct {
	limiter *rate.Limiter
}

// Global rate limiter instance
// Configured for 1000 requests per second with burst of 5000 for stability under high load
var globalLimiter = rate.NewLimiter(rate.Limit(1000), 5000)

// NewRateLimiter creates a new RateLimiter with the specified rate and burst
func NewRateLimiter(rps int, burst int) *RateLimiter {
	return &RateLimiter{
		limiter: rate.NewLimiter(rate.Limit(rps), burst),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow() bool {
	return rl.limiter.Allow()
}

// RateLimitMiddleware creates a middleware that limits request rate
// Uses the global rate limiter configured for 1000 req/s + 5000 burst
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !globalLimiter.Allow() {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			// Log rate limit hit asynchronously
			go LogError("rate_limit", nil, "Rate limit exceeded for request: "+r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// GetGlobalLimiter returns the global rate limiter for testing/monitoring
func GetGlobalLimiter() *rate.Limiter {
	return globalLimiter
}

// SetGlobalLimiter allows configuring the global limiter (for testing)
func SetGlobalLimiter(rps int, burst int) {
	globalLimiter = rate.NewLimiter(rate.Limit(rps), burst)
}
