package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"go-microservice/handlers"
	"go-microservice/metrics"
	"go-microservice/services"
	"go-microservice/utils"
)

func main() {
	// Initialize router
	router := mux.NewRouter()

	// Apply middleware chain
	// Order matters: metrics -> rate limiting -> handlers
	router.Use(metrics.MetricsMiddleware)
	router.Use(utils.RateLimitMiddleware)

	// Register Prometheus metrics endpoint
	router.Handle("/metrics", metrics.Handler()).Methods("GET")

	// Initialize and register handlers
	userHandler := handlers.NewUserHandler()
	userHandler.RegisterRoutes(router)

	integrationHandler := handlers.NewIntegrationHandler()
	integrationHandler.RegisterRoutes(router)

	// Try to connect to MinIO on startup (non-blocking)
	go func() {
		config := services.GetDefaultConfig()
		if err := services.GetIntegrationService().Connect(config); err != nil {
			log.Printf("MinIO connection failed on startup (will retry on demand): %v", err)
		}
	}()

	// Get server port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Configure HTTP server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Endpoints available:")
		log.Printf("  - GET    /api/users         - List all users")
		log.Printf("  - GET    /api/users/{id}    - Get user by ID")
		log.Printf("  - POST   /api/users         - Create new user")
		log.Printf("  - PUT    /api/users/{id}    - Update user")
		log.Printf("  - DELETE /api/users/{id}    - Delete user")
		log.Printf("  - GET    /api/health        - Health check")
		log.Printf("  - GET    /metrics           - Prometheus metrics")
		log.Printf("Rate limit: 1000 req/s with burst of 5000")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}
