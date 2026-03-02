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

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/handler"
	"github.com/doomedramen/lab/apps/api/internal/middleware"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	dockerRepo "github.com/doomedramen/lab/apps/api/internal/repository/docker"
	"github.com/doomedramen/lab/apps/api/internal/repository/libvirt"
	sqliteRepo "github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/internal/router"
	"github.com/doomedramen/lab/apps/api/internal/service"
	libvirtx "github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	"github.com/doomedramen/lab/apps/api/pkg/tus"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
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
	db, err := sqlitePkg.New(sqlitePkg.Config{
		Path:           cfg.Storage.DataDir + "/metrics.db",
		RetentionDays:  30,
		EventRetention: 90,
		LogRetention:   cfg.Logging.VMLogRetentionDays,
	})
	if err != nil {
		log.Printf("Warning: Failed to initialize SQLite: %v, metrics will not be collected", err)
	} else {
		defer db.Close()
		log.Printf("SQLite database initialized at %s", db.Path())
	}

	// Initialize SQLite repositories
	var (
		metricRepo *sqliteRepo.MetricRepository
		eventRepo  *sqliteRepo.EventRepository
		logRepo    *sqliteRepo.VMLogRepository
	)
	if db != nil {
		metricRepo = sqliteRepo.NewMetricRepository(db)
		eventRepo = sqliteRepo.NewEventRepository(db)
		logRepo = sqliteRepo.NewVMLogRepository(db)
	}

	// Initialize auth repositories
	var (
		userRepo       *auth.UserRepository
		tokenRepo      *auth.RefreshTokenRepository
		apiKeyRepo     *auth.APIKeyRepository
		auditRepo      *auth.AuditLogRepository
		sessionRepo    *auth.SessionRepository
	)
	if db != nil {
		userRepo = auth.NewUserRepository(db.DB)
		tokenRepo = auth.NewRefreshTokenRepository(db.DB)
		apiKeyRepo = auth.NewAPIKeyRepository(db.DB)
		auditRepo = auth.NewAuditLogRepository(db.DB)
		sessionRepo = auth.NewSessionRepository(db.DB)
	}

	// Initialize auth service
	var authService *service.AuthService
	if userRepo != nil {
		// Validate JWT secret
		if cfg.Auth.JWTSecret == "" {
			if cfg.Server.Env == "production" {
				log.Fatalf("JWT_SECRET is required in production")
			}
			log.Println("Warning: JWT_SECRET not set, using insecure default")
		}

		authService = service.NewAuthService(
			userRepo,
			tokenRepo,
			apiKeyRepo,
			auditRepo,
			sessionRepo,
			service.AuthServiceConfig{
				JWTSecret:       []byte(cfg.Auth.JWTSecret),
				AccessTokenExp:  parseDuration(cfg.Auth.AccessTokenExpiry),
				RefreshTokenExp: parseDuration(cfg.Auth.RefreshTokenExpiry),
				Issuer:          cfg.Auth.Issuer,
			},
		)
		log.Println("Authentication service initialized")
	}

	// Initialize repositories based on backend
	var (
		nodeRepo      repository.NodeRepository
		vmRepo        repository.VMRepository
		containerRepo repository.ContainerRepository
		stackRepo     repository.StackRepository
		isoRepo       repository.ISORepository
		snapshotRepo  repository.SnapshotRepository
		backupRepo    repository.BackupRepository
		scheduleRepo  repository.BackupScheduleRepository
		poolRepo      repository.StoragePoolRepository
		diskRepo      repository.StorageDiskRepository
		networkRepo   repository.NetworkRepository
		ifaceRepo     repository.NetworkInterfaceRepository
		firewallRepo  repository.FirewallRuleRepository
		groupRepo     repository.FirewallGroupRepository
		dhcpRepo      repository.DHCPLeaseRepository
		libvirtClient *libvirtx.Client
	)

	log.Printf("Using libvirt backend (URI: %s)", cfg.Libvirt.URI)

	libvirtClient, err = libvirtx.NewClient(&libvirtx.Config{URI: cfg.Libvirt.URI})
	if err != nil {
		log.Fatalf("Failed to connect to libvirt: %v", err)
	}
	defer libvirtClient.Disconnect()

	nodeRepo = libvirt.NewNodeRepository(libvirtClient)
	vmRepo = libvirt.NewVMRepository(libvirtClient, cfg)

	// Try to connect to libvirt LXC driver for container management.
	// Falls back to the no-op sqlite stub if LXC is not available on this host.
	lxcContainerDir := cfg.Storage.DataDir + "/containers"
	if lxcClient, err := libvirtx.NewClient(&libvirtx.Config{URI: "lxc:///system"}); err != nil {
		log.Printf("LXC not available (%v) — container support disabled", err)
		containerRepo = sqliteRepo.NewContainerRepository()
	} else {
		log.Println("LXC support enabled via lxc:///system")
		containerRepo = libvirt.NewContainerRepository(lxcClient, lxcContainerDir)
	}

	// Docker Compose stack repository — requires stacks_dir to be configured
	if cfg.Storage.StacksDir == "" {
		log.Println("Warning: stacks_dir not configured — Docker Compose stacks feature disabled")
		stackRepo = sqliteRepo.NewStackRepository()
	} else {
		log.Printf("Docker Compose stacks enabled (dir: %s)", cfg.Storage.StacksDir)
		stackRepo = dockerRepo.NewStackRepository(cfg.Storage.StacksDir)
	}
	isoRepo = libvirt.NewISORepository(libvirtClient, cfg)

	// Snapshot repositories (SQLite for metadata, libvirt for actual snapshots)
	if db != nil {
		snapshotRepo = sqliteRepo.NewSnapshotRepository(db)
	}
	snapshotLib := libvirt.NewSnapshotRepository(libvirtClient)

	// Backup repositories (SQLite for metadata, libvirt for actual backup operations)
	var backupLib *libvirt.BackupRepository
	if db != nil {
		backupRepo = sqliteRepo.NewBackupRepository(db)
		scheduleRepo = sqliteRepo.NewBackupScheduleRepository(db)
		// Create backup directory if it doesn't exist
		backupDir := cfg.Storage.DataDir + "/backups"
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			log.Printf("Warning: Failed to create backup directory: %v", err)
		}
		backupLib = libvirt.NewBackupRepository(libvirtClient, backupDir)
	}

	// Task repositories (SQLite for task tracking)
	var taskRepo repository.TaskRepository
	if db != nil {
		taskRepo = sqliteRepo.NewTaskRepository(db)
	}

	// Storage repositories (SQLite for metadata, libvirt for actual storage operations)
	if db != nil {
		poolRepo = sqliteRepo.NewStoragePoolRepository(db)
		diskRepo = sqliteRepo.NewStorageDiskRepository(db)
	}

	// Network repositories (SQLite for metadata)
	if db != nil {
		networkRepo = sqliteRepo.NewNetworkRepository(db)
		ifaceRepo = sqliteRepo.NewNetworkInterfaceRepository(db)
		firewallRepo = sqliteRepo.NewFirewallRuleRepository(db)
		groupRepo = sqliteRepo.NewFirewallGroupRepository(db)
		dhcpRepo = sqliteRepo.NewDHCPLeaseRepository(db)
	}

	// Alert repositories (SQLite for alert rules, channels, and fired alerts)
	var alertRepo *sqliteRepo.AlertRepository
	if db != nil {
		alertRepo = sqliteRepo.NewAlertRepository(db)
	}

	// Proxy repository (SQLite for proxy host config and certs)
	var proxyRepo *sqliteRepo.ProxyRepository
	if db != nil {
		proxyRepo = sqliteRepo.NewProxyRepository(db)
	}

	// Initialize services
	clusterSvc := service.NewClusterService(nodeRepo, vmRepo, containerRepo, stackRepo, metricRepo)
	nodeSvc := service.NewNodeService(nodeRepo)

	// Task service (if database is available) - must be initialized before backup/snapshot/VM services
	var taskSvc *service.TaskService
	if taskRepo != nil {
		taskSvc = service.NewTaskService(taskRepo)
		log.Println("Task service initialized")
	}

	vmSvc := service.NewVMService(vmRepo, isoRepo, logRepo, taskSvc, cfg.Logging.VMLogRetentionDays)

	// Guest agent repository for VM IP discovery and consistent backups
	guestAgentRepo := libvirt.NewGuestAgentRepository(libvirtClient)
	vmSvc.WithGuestAgentRepo(guestAgentRepo)
	log.Println("Guest agent repository initialized")

	// PCI device repository for GPU/PCI passthrough
	pciRepo := libvirt.NewPCIRepository(libvirtClient)
	vmSvc.WithPCIRepo(pciRepo)
	log.Println("PCI device repository initialized")

	containerSvc := service.NewContainerService(containerRepo)
	stackSvc := service.NewStackService(stackRepo)
	isoSvc := service.NewISOService(isoRepo, cfg.Storage.ISODir, cfg.Storage.ISODownloadTempDir, cfg.Storage.MaxISOSize)

	// Snapshot service (if database is available)
	var snapshotSvc *service.SnapshotService
	if snapshotRepo != nil {
		snapshotSvc = service.NewSnapshotService(snapshotRepo, snapshotLib, vmRepo, taskSvc)
		log.Println("Snapshot service initialized")
	}

	// Backup service (if database is available)
	var backupSvc *service.BackupService
	if backupRepo != nil && scheduleRepo != nil && backupLib != nil {
		backupSvc = service.NewBackupService(backupRepo, scheduleRepo, backupLib, vmRepo, taskSvc)
		backupSvc.WithGuestAgentRepo(guestAgentRepo)
		log.Println("Backup service initialized")
	}

	// Storage service (if database is available)
	var storageSvc *service.StorageService
	if poolRepo != nil && diskRepo != nil {
		// Create libvirt disk repository for VM disk operations
		diskLib := libvirt.NewDiskRepository(libvirtClient)
		storageSvc = service.NewStorageService(poolRepo, diskRepo, diskLib)
		log.Println("Storage service initialized")
	}

	// Network service (if database is available)
	var networkSvc *service.NetworkService
	var firewallSvc *service.FirewallService
	if networkRepo != nil && ifaceRepo != nil && firewallRepo != nil {
		networkSvc = service.NewNetworkService(networkRepo, ifaceRepo, firewallRepo)
		// Wire up libvirt for actual network lifecycle management
		libvirtNetRepo := libvirt.NewLibvirtNetworkRepository(libvirtClient)
		networkSvc.WithLibvirtNetworkRepo(libvirtNetRepo)
		if dhcpRepo != nil {
			networkSvc.WithDHCPLeaseRepo(dhcpRepo)
		}
		firewallSvc = service.NewFirewallService(firewallRepo, groupRepo, networkRepo)
		log.Println("Network and Firewall services initialized")
	}

	// Alert service (if database is available)
	var alertSvc *service.AlertService
	if alertRepo != nil {
		alertSvc = service.NewAlertService(alertRepo, service.AlertServiceConfig{
			EvaluationInterval: 60 * time.Second,
			RetentionDays:      30,
		})
		// Wire up metric providers for alert evaluation
		alertSvc.WithNodeProvider(nodeRepo)
		alertSvc.WithVMProvider(vmRepo)
		alertSvc.WithStorageProvider(storageSvc)
		alertSvc.WithBackupProvider(backupSvc)
		alertSvc.Start()
		defer alertSvc.Stop()
		log.Println("Alert service initialized")
	}

	// Proxy service (if database is available and enabled in config)
	var proxySvc *service.ProxyService
	if proxyRepo != nil && cfg.Proxy.Enabled {
		proxySvc = service.NewProxyService(proxyRepo, cfg.Proxy.HTTPPort, cfg.Proxy.HTTPSPort)
		// Wire alert sender for uptime failure notifications
		if alertSvc != nil {
			proxySvc.WithAlertSender(alertSvc)
			alertSvc.WithUptimeProvider(proxySvc)
		}
		if err := proxySvc.Start(context.Background()); err != nil {
			log.Printf("Warning: Proxy service failed to start: %v", err)
			proxySvc = nil
		} else {
			defer proxySvc.Stop()
			log.Printf("Proxy service started (HTTP :%d, HTTPS :%d)", cfg.Proxy.HTTPPort, cfg.Proxy.HTTPSPort)
		}
	}

	// Initialize metrics collector
	var collector *service.Collector
	if metricRepo != nil && eventRepo != nil {
		collector = service.NewCollector(
			service.DefaultCollectorConfig(),
			libvirtClient,
			metricRepo,
			eventRepo,
			vmSvc,
			nodeRepo,
			vmRepo,
			containerRepo,
		)
		collector.Start()
		defer collector.Stop()
		log.Println("Metrics collector started")
	}

	// Initialize handlers
	metricsHandler := handler.NewMetricsHandler(metricRepo, eventRepo)
	eventsHandler := handler.NewEventsHandler(eventRepo)

	// Initialize Tus handler for resumable uploads
	tusHandler, err := tus.NewHandler(tus.Config{
		BasePath:  "/tus/files/",
		UploadDir: cfg.Storage.ISODir,
		MaxSize:   cfg.Storage.MaxISOSize,
	})
	if err != nil {
		log.Printf("Warning: Failed to create Tus handler: %v", err)
	}

	// Create auth interceptor
	var authInterceptor *middleware.AuthInterceptor
	if authService != nil {
		jwtSecret := []byte(cfg.Auth.JWTSecret)
		if jwtSecret == nil || len(jwtSecret) == 0 {
			// Insecure default for development only
			jwtSecret = []byte("insecure-dev-secret-change-in-production")
		}

		authInterceptor = middleware.NewAuthInterceptor(
			middleware.AuthInterceptorConfig{
				JWTSecret: jwtSecret,
				Issuer:    cfg.Auth.Issuer,
			},
			userRepo,
			apiKeyRepo,
			auditRepo,
			sessionRepo,
		)
	}

	// Enhanced readiness health check
	healthHandler := handler.NewHealthHandler(libvirtClient, db, version)

	// Create router
	r := router.Router(clusterSvc, nodeSvc, vmSvc, containerSvc, stackSvc, cfg.Storage.StacksDir, isoSvc, tusHandler, authService, snapshotSvc, backupSvc, taskSvc, storageSvc, networkSvc, firewallSvc, alertSvc, proxySvc, authInterceptor, healthHandler)

	// Register additional routes for metrics and events
	if metricRepo != nil && eventRepo != nil {
		handler.RegisterMetricsRoutes(r, metricsHandler)
		handler.RegisterEventsRoutes(r, eventsHandler)
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

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s (env: %s)", addr, cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
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
