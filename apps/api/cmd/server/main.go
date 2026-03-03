package main

import (
	"context"
	"log"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/config"
	appmiddleware "github.com/doomedramen/lab/apps/api/internal/middleware"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
	"golang.org/x/time/rate"
)

// Version info, set at build time via ldflags
var version = "dev"

func main() {
	// Load configuration
	cfg := config.Load()

	// Ensure required directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		log.Printf("Warning: Failed to create directories: %v", err)
	}

	// Initialize SQLite database
	db, err := initDatabase(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize SQLite: %v, metrics will not be collected", err)
	} else {
		defer db.Close()
		log.Printf("SQLite database initialized at %s", db.Path())
	}

	// Validate JWT secret early (fail fast)
	jwtSecret := validateJWTSecret(cfg)

	// Initialize repositories
	ctx := context.Background()
	repos, err := NewRepositories(ctx, cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize repositories: %v", err)
	}
	defer repos.Close()

	// Initialize services
	svcs, err := NewServices(ctx, &ServiceDependencies{
		Repos:     repos,
		Config:    cfg,
		JWTSecret: jwtSecret,
		Version:   version,
	})
	if err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}
	defer svcs.Stop()

	// Create auth interceptor
	authInterceptor := NewAuthInterceptor(cfg, repos)

	// Create handlers
	healthHandler := NewHealthHandler(repos.LibvirtClient, db, version)
	metricsHandler := NewMetricsHandler(repos.MetricRepo, repos.EventRepo)
	eventsHandler := NewEventsHandler(repos.EventRepo)

	// Initialize Tus handler for resumable uploads
	tusHandler, err := NewTusHandler(cfg)
	if err != nil {
		log.Printf("Warning: Failed to create Tus handler: %v", err)
	}

	// Add rate limiter for auth endpoints: 5 attempts per second, burst of 10.
	// This prevents brute-force on Login, Register, and MFA endpoints.
	authRateLimiter := appmiddleware.NewRateLimiter(rate.Limit(5), 10)
	if svcs.AuthService != nil {
		// Auth handler is created in router.Router with rate limiting
		_ = authRateLimiter
	}

	// Create and start server
	server := NewServer(&ServerDependencies{
		Services:        svcs,
		Repos:           repos,
		Config:          cfg,
		AuthInterceptor: authInterceptor,
		HealthHandler:   healthHandler,
		MetricsHandler:  metricsHandler,
		EventsHandler:   eventsHandler,
		TusHandler:      tusHandler,
		Version:         version,
	})

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for shutdown signal
	<-WaitForSignal()

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Stop services first
	svcs.Stop()

	// Then shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// validateJWTSecret validates the JWT secret configuration
func validateJWTSecret(cfg *config.Config) []byte {
	if cfg.Auth.JWTSecret == "" {
		log.Fatalf("JWT_SECRET is required. Generate a secure random secret with: openssl rand -base64 32")
	}

	if len(cfg.Auth.JWTSecret) < 16 {
		log.Fatalf("JWT_SECRET is too short (minimum 16 characters). Generate a secure random secret with: openssl rand -base64 32")
	}

	return []byte(cfg.Auth.JWTSecret)
}

// initDatabase initializes the SQLite database
func initDatabase(cfg *config.Config) (*sqlitePkg.DB, error) {
	return sqlitePkg.New(sqlitePkg.Config{
		Path:           cfg.Storage.DataDir + "/metrics.db",
		RetentionDays:  30,
		EventRetention: 90,
		LogRetention:   cfg.Logging.VMLogRetentionDays,
	})
}

// parseDuration parses a duration string (e.g., "15m", "7h") into time.Duration
func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		// Default to 15 minutes for access token, 7 days for refresh token
		if s == "" {
			return 15 * time.Minute
		}
		log.Printf("Warning: Invalid duration %q, using default", s)
		return 15 * time.Minute
	}
	return d
}
