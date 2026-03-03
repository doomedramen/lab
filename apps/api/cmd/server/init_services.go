package main

import (
	"context"
	"log"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/repository/libvirt"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// Services holds all service instances
type Services struct {
	AuthService   *service.AuthService
	ClusterSvc    *service.ClusterService
	NodeSvc       *service.NodeService
	VMSvc         *service.VMService
	ContainerSvc  *service.ContainerService
	StackSvc      *service.StackService
	ISOSvc        *service.ISOService
	SnapshotSvc   *service.SnapshotService
	BackupSvc     *service.BackupService
	TaskSvc       *service.TaskService
	StorageSvc    *service.StorageService
	NetworkSvc    *service.NetworkService
	FirewallSvc   *service.FirewallService
	AlertSvc      *service.AlertService
	ProxySvc      *service.ProxyService
	Collector     *service.Collector
}

// ServiceDependencies holds dependencies for service initialization
type ServiceDependencies struct {
	Repos       *Repositories
	Config      *config.Config
	JWTSecret   []byte
	Version     string
}

// NewServices initializes all services
func NewServices(ctx context.Context, deps *ServiceDependencies) (*Services, error) {
	svcs := &Services{}
	repos := deps.Repos
	cfg := deps.Config

	// Initialize auth service
	if repos.UserRepo != nil {
		// JWT secret validation already done in main.go
		svcs.AuthService = service.NewAuthService(
			repos.UserRepo,
			repos.TokenRepo,
			repos.APIKeyRepo,
			repos.AuditRepo,
			repos.SessionRepo,
			service.AuthServiceConfig{
				JWTSecret:       deps.JWTSecret,
				AccessTokenExp:  parseDuration(cfg.Auth.AccessTokenExpiry),
				RefreshTokenExp: parseDuration(cfg.Auth.RefreshTokenExpiry),
				Issuer:          cfg.Auth.Issuer,
			},
		)
		log.Println("Authentication service initialized")
	}

	// Initialize task service first (required by other services)
	if repos.TaskRepo != nil {
		svcs.TaskSvc = service.NewTaskService(repos.TaskRepo)
		log.Println("Task service initialized")
	}

	// Initialize core services
	svcs.ClusterSvc = service.NewClusterService(
		repos.NodeRepo,
		repos.VMRepo,
		repos.ContainerRepo,
		repos.StackRepo,
		repos.MetricRepo,
	)
	svcs.NodeSvc = service.NewNodeService(repos.NodeRepo)

	// Initialize VM service with guest agent and PCI support
	svcs.VMSvc = service.NewVMService(
		repos.VMRepo,
		repos.ISORepo,
		repos.LogRepo,
		svcs.TaskSvc,
		cfg.Logging.VMLogRetentionDays,
	)
	svcs.VMSvc.WithGuestAgentRepo(repos.GuestAgentRepo)
	svcs.VMSvc.WithPCIRepo(repos.PCIRepo)
	log.Println("Guest agent and PCI repositories initialized")

	// Initialize other services
	svcs.ContainerSvc = service.NewContainerService(repos.ContainerRepo)
	svcs.StackSvc = service.NewStackService(repos.StackRepo)
	svcs.ISOSvc = service.NewISOService(
		repos.ISORepo,
		cfg.Storage.ISODir,
		cfg.Storage.ISODownloadTempDir,
		cfg.Storage.MaxISOSize,
	)

	// Initialize snapshot service
	if repos.SnapshotRepo != nil {
		svcs.SnapshotSvc = service.NewSnapshotService(
			repos.SnapshotRepo,
			repos.SnapshotLib,
			repos.VMRepo,
			svcs.TaskSvc,
		)
		log.Println("Snapshot service initialized")
	}

	// Initialize backup service
	if repos.BackupRepo != nil && repos.BackupSchedule != nil && repos.BackupLib != nil {
		svcs.BackupSvc = service.NewBackupService(
			ctx,
			repos.BackupRepo,
			repos.BackupSchedule,
			repos.BackupLib,
			repos.VMRepo,
			svcs.TaskSvc,
		)
		svcs.BackupSvc.WithGuestAgentRepo(repos.GuestAgentRepo)
		log.Println("Backup service initialized")
	}

	// Initialize storage service
	if repos.PoolRepo != nil && repos.DiskRepo != nil {
		svcs.StorageSvc = service.NewStorageService(
			repos.PoolRepo,
			repos.DiskRepo,
			repos.DiskLib,
		)
		log.Println("Storage service initialized")
	}

	// Initialize network and firewall services
	if repos.NetworkRepo != nil && repos.IfaceRepo != nil && repos.FirewallRepo != nil {
		svcs.NetworkSvc = service.NewNetworkService(
			repos.NetworkRepo,
			repos.IfaceRepo,
			repos.FirewallRepo,
		)
		// Wire up libvirt for actual network lifecycle management
		libvirtNetRepo := libvirt.NewLibvirtNetworkRepository(repos.LibvirtClient)
		svcs.NetworkSvc.WithLibvirtNetworkRepo(libvirtNetRepo)
		if repos.DHCPRepo != nil {
			svcs.NetworkSvc.WithDHCPLeaseRepo(repos.DHCPRepo)
		}

		svcs.FirewallSvc = service.NewFirewallService(
			repos.FirewallRepo,
			repos.GroupRepo,
			repos.NetworkRepo,
		)
		log.Println("Network and Firewall services initialized")
	}

	// Initialize alert service
	if repos.AlertRepo != nil {
		svcs.AlertSvc = service.NewAlertService(repos.AlertRepo, service.AlertServiceConfig{
			EvaluationInterval: 60 * time.Second,
			RetentionDays:      30,
		})
		// Wire up metric providers for alert evaluation
		svcs.AlertSvc.WithNodeProvider(repos.NodeRepo)
		svcs.AlertSvc.WithVMProvider(repos.VMRepo)
		svcs.AlertSvc.WithStorageProvider(svcs.StorageSvc)
		svcs.AlertSvc.WithBackupProvider(svcs.BackupSvc)
		svcs.AlertSvc.Start()
		log.Println("Alert service initialized")
	}

	// Initialize proxy service
	if repos.ProxyRepo != nil && cfg.Proxy.Enabled {
		svcs.ProxySvc = service.NewProxyService(
			repos.ProxyRepo,
			cfg.Proxy.HTTPPort,
			cfg.Proxy.HTTPSPort,
		)
		// Wire alert sender for uptime failure notifications
		if svcs.AlertSvc != nil {
			svcs.ProxySvc.WithAlertSender(svcs.AlertSvc)
			svcs.AlertSvc.WithUptimeProvider(svcs.ProxySvc)
		}
		if err := svcs.ProxySvc.Start(ctx); err != nil {
			log.Printf("Warning: Proxy service failed to start: %v", err)
			svcs.ProxySvc = nil
		} else {
			log.Printf("Proxy service started (HTTP :%d, HTTPS :%d)", cfg.Proxy.HTTPPort, cfg.Proxy.HTTPSPort)
		}
	}

	// Initialize metrics collector
	if repos.MetricRepo != nil && repos.EventRepo != nil {
		svcs.Collector = service.NewCollector(
			service.DefaultCollectorConfig(),
			repos.LibvirtClient,
			repos.MetricRepo,
			repos.EventRepo,
			svcs.VMSvc,
			repos.NodeRepo,
			repos.VMRepo,
			repos.ContainerRepo,
		)
		svcs.Collector.Start(ctx)
		log.Println("Metrics collector started")
	}

	return svcs, nil
}

// Stop gracefully stops all services
func (s *Services) Stop() {
	if s.Collector != nil {
		s.Collector.Stop()
	}
	if s.AlertSvc != nil {
		s.AlertSvc.Stop()
	}
	if s.ProxySvc != nil {
		s.ProxySvc.Stop()
	}
	if s.BackupSvc != nil {
		s.BackupSvc.StopScheduler()
	}
}
