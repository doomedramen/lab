package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/handler"
	appmiddleware "github.com/doomedramen/lab/apps/api/internal/middleware"
	"github.com/doomedramen/lab/apps/api/internal/router"
	sqliteRepo "github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/tus"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
	libvirtx "github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// Server wraps the HTTP server and its dependencies
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	config     *config.Config
	version    string
}

// ServerDependencies holds dependencies for server initialization
type ServerDependencies struct {
	Services         *Services
	Repos            *Repositories
	Config           *config.Config
	AuthInterceptor  *appmiddleware.AuthInterceptor
	HealthHandler    *handler.HealthHandler
	MetricsHandler   *handler.MetricsHandler
	EventsHandler    *handler.EventsHandler
	TusHandler       *tus.Handler
	Version          string
}

// NewServer creates and configures the HTTP server
func NewServer(deps *ServerDependencies) *Server {
	cfg := deps.Config

	// Create router
	r := router.Router(
		deps.Services.ClusterSvc,
		deps.Services.NodeSvc,
		deps.Services.VMSvc,
		deps.Services.ContainerSvc,
		deps.Services.StackSvc,
		cfg.Storage.StacksDir,
		deps.Services.ISOSvc,
		deps.TusHandler,
		deps.Services.AuthService,
		deps.Services.SnapshotSvc,
		deps.Services.BackupSvc,
		deps.Services.TaskSvc,
		deps.Services.StorageSvc,
		deps.Services.NetworkSvc,
		deps.Services.FirewallSvc,
		deps.Services.AlertSvc,
		deps.Services.ProxySvc,
		deps.Services.GitOpsSvc,
		deps.AuthInterceptor,
		deps.HealthHandler,
		deps.Repos.AuditRepo,
		cfg.GetMaxRequestBodySize(),
	)

	// Register additional routes for metrics and events
	if deps.Repos.MetricRepo != nil && deps.Repos.EventRepo != nil {
		handler.RegisterMetricsRoutes(r, deps.MetricsHandler)
		handler.RegisterEventsRoutes(r, deps.EventsHandler)
	}

	// Register status API routes (for Homepage and other dashboards)
	if deps.Repos.MetricRepo != nil && deps.Repos.EventRepo != nil {
		apiKeyAuth := appmiddleware.NewAPIKeyAuthMiddleware(
			deps.Repos.APIKeyRepo,
			deps.Repos.UserRepo,
			deps.Repos.AuditRepo,
		)
		statusHandler := handler.NewStatusHandler(
			deps.Repos.MetricRepo,
			deps.Repos.EventRepo,
			deps.Repos.LibvirtClient,
		)
		handler.RegisterStatusRoutes(r, statusHandler, apiKeyAuth.RequireAPIKey)
	}

	// Create server
	// WriteTimeout is 0 so that long-lived WebSocket connections (e.g. VNC proxy)
	// are not killed mid-stream. Individual handlers are responsible for their
	// own timeouts.
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		router:     r,
		config:     cfg,
		version:    deps.Version,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.Server.Port)
	protocol := "http"
	if s.config.Server.TLSEnabled {
		protocol = "https"
	}
	log.Printf("Starting server on %s://%s (env: %s, version: %s)", protocol, addr, s.config.Server.Env, s.version)

	go func() {
		var err error
		if s.config.Server.TLSEnabled {
			err = s.httpServer.ListenAndServeTLS(
				s.config.Server.TLSCertFile,
				s.config.Server.TLSKeyFile,
			)
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	return s.httpServer.Shutdown(ctx)
}

// WaitForSignal blocks until a shutdown signal is received
func WaitForSignal() <-chan os.Signal {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	return quit
}

// NewAuthInterceptor creates the auth interceptor middleware.
// jwtSecret must be provided and validated before calling this function.
func NewAuthInterceptor(cfg *config.Config, repos *Repositories, jwtSecret []byte) *appmiddleware.AuthInterceptor {
	if repos.UserRepo == nil {
		return nil
	}

	return appmiddleware.NewAuthInterceptor(
		appmiddleware.AuthInterceptorConfig{
			JWTSecret: jwtSecret,
			Issuer:    cfg.Auth.Issuer,
		},
		repos.UserRepo,
		repos.APIKeyRepo,
		repos.AuditRepo,
		repos.SessionRepo,
	)
}

// NewHealthHandler creates the health check handler
func NewHealthHandler(libvirtClient libvirtx.LibvirtClient, db *sqlitePkg.DB, version string) *handler.HealthHandler {
	return handler.NewHealthHandler(libvirtClient, db, version)
}

// NewMetricsHandler creates the metrics handler
func NewMetricsHandler(metricRepo *sqliteRepo.MetricRepository, eventRepo *sqliteRepo.EventRepository) *handler.MetricsHandler {
	return handler.NewMetricsHandler(metricRepo, eventRepo)
}

// NewEventsHandler creates the events handler
func NewEventsHandler(eventRepo *sqliteRepo.EventRepository) *handler.EventsHandler {
	return handler.NewEventsHandler(eventRepo)
}

// NewTusHandler creates the Tus upload handler
func NewTusHandler(cfg *config.Config) (*tus.Handler, error) {
	return tus.NewHandler(tus.Config{
		BasePath:  "/tus/files/",
		UploadDir: cfg.Storage.ISODir,
		MaxSize:   cfg.Storage.MaxISOSize,
	})
}
