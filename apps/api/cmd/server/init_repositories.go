package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	dockerRepo "github.com/doomedramen/lab/apps/api/internal/repository/docker"
	"github.com/doomedramen/lab/apps/api/internal/repository/libvirt"
	sqliteRepo "github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
	libvirtx "github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// Repositories holds all repository instances
type Repositories struct {
	// SQLite repositories
	MetricRepo  *sqliteRepo.MetricRepository
	EventRepo   *sqliteRepo.EventRepository
	LogRepo     *sqliteRepo.VMLogRepository
	UserRepo    *auth.UserRepository
	TokenRepo   *auth.RefreshTokenRepository
	APIKeyRepo  *auth.APIKeyRepository
	AuditRepo   *auth.AuditLogRepository
	SessionRepo *auth.SessionRepository

	// Core repositories
	NodeRepo      repository.NodeRepository
	VMRepo        repository.VMRepository
	ContainerRepo repository.ContainerRepository
	StackRepo     repository.StackRepository
	ISORepo       repository.ISORepository

	// Feature repositories
	SnapshotRepo   repository.SnapshotRepository
	SnapshotLib    *libvirt.SnapshotRepository
	BackupRepo     repository.BackupRepository
	BackupSchedule repository.BackupScheduleRepository
	BackupLib      *libvirt.BackupRepository
	TaskRepo       repository.TaskRepository
	PoolRepo       repository.StoragePoolRepository
	DiskRepo       repository.StorageDiskRepository
	DiskLib        *libvirt.DiskRepository
	NetworkRepo    repository.NetworkRepository
	IfaceRepo      repository.NetworkInterfaceRepository
	FirewallRepo   repository.FirewallRuleRepository
	GroupRepo      repository.FirewallGroupRepository
	DHCPRepo       repository.DHCPLeaseRepository
	AlertRepo      *sqliteRepo.AlertRepository
	ProxyRepo      *sqliteRepo.ProxyRepository

	// Libvirt client
	LibvirtClient *libvirtx.Client

	// Guest agent and PCI repositories
	GuestAgentRepo repository.GuestAgentRepository
	PCIRepo        repository.PCIRepository
}

// NewRepositories initializes all repositories
func NewRepositories(ctx context.Context, cfg *config.Config, db *sqlitePkg.DB) (*Repositories, error) {
	repos := &Repositories{}

	// Initialize libvirt client first (required for most repositories)
	var err error
	repos.LibvirtClient, err = libvirtx.NewClient(&libvirtx.Config{URI: cfg.Libvirt.URI})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %w", err)
	}

	log.Printf("Using libvirt backend (URI: %s)", cfg.Libvirt.URI)

	// Initialize SQLite repositories
	if db != nil {
		repos.MetricRepo = sqliteRepo.NewMetricRepository(db)
		repos.EventRepo = sqliteRepo.NewEventRepository(db)
		repos.LogRepo = sqliteRepo.NewVMLogRepository(db)
		repos.UserRepo = auth.NewUserRepository(db.DB)
		repos.TokenRepo = auth.NewRefreshTokenRepository(db.DB)
		repos.APIKeyRepo = auth.NewAPIKeyRepository(db.DB)
		repos.AuditRepo = auth.NewAuditLogRepository(db.DB)
		repos.SessionRepo = auth.NewSessionRepository(db.DB)
		repos.SnapshotRepo = sqliteRepo.NewSnapshotRepository(db)
		repos.BackupRepo = sqliteRepo.NewBackupRepository(db)
		repos.BackupSchedule = sqliteRepo.NewBackupScheduleRepository(db)
		repos.TaskRepo = sqliteRepo.NewTaskRepository(db)
		repos.PoolRepo = sqliteRepo.NewStoragePoolRepository(db)
		repos.DiskRepo = sqliteRepo.NewStorageDiskRepository(db)
		repos.NetworkRepo = sqliteRepo.NewNetworkRepository(db)
		repos.IfaceRepo = sqliteRepo.NewNetworkInterfaceRepository(db)
		repos.FirewallRepo = sqliteRepo.NewFirewallRuleRepository(db)
		repos.GroupRepo = sqliteRepo.NewFirewallGroupRepository(db)
		repos.DHCPRepo = sqliteRepo.NewDHCPLeaseRepository(db)
		repos.AlertRepo = sqliteRepo.NewAlertRepository(db)
		repos.ProxyRepo = sqliteRepo.NewProxyRepository(db)
	}

	// Initialize core libvirt repositories
	repos.NodeRepo = libvirt.NewNodeRepository(repos.LibvirtClient)
	repos.VMRepo = libvirt.NewVMRepository(repos.LibvirtClient, cfg)
	repos.ISORepo = libvirt.NewISORepository(repos.LibvirtClient, cfg)

	// Initialize container repository (LXC or fallback)
	lxcContainerDir := cfg.Storage.DataDir + "/containers"
	if lxcClient, err := libvirtx.NewClient(&libvirtx.Config{URI: "lxc:///system"}); err != nil {
		log.Printf("LXC not available (%v) — container support disabled", err)
		repos.ContainerRepo = sqliteRepo.NewContainerRepository()
	} else {
		log.Println("LXC support enabled via lxc:///system")
		repos.ContainerRepo = libvirt.NewContainerRepository(lxcClient, lxcContainerDir)
	}

	// Initialize stack repository (Docker Compose or fallback)
	if cfg.Storage.StacksDir == "" {
		log.Println("Warning: stacks_dir not configured — Docker Compose stacks feature disabled")
		repos.StackRepo = sqliteRepo.NewStackRepository()
	} else {
		log.Printf("Docker Compose stacks enabled (dir: %s)", cfg.Storage.StacksDir)
		repos.StackRepo = dockerRepo.NewStackRepository(cfg.Storage.StacksDir)
	}

	// Initialize libvirt feature repositories
	repos.SnapshotLib = libvirt.NewSnapshotRepository(repos.LibvirtClient)

	// Initialize backup libvirt repository
	if repos.BackupRepo != nil {
		backupDir := cfg.Storage.DataDir + "/backups"
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			log.Printf("Warning: Failed to create backup directory: %v", err)
		}
		repos.BackupLib = libvirt.NewBackupRepository(repos.LibvirtClient, backupDir)
	}

	// Initialize disk libvirt repository
	if repos.PoolRepo != nil {
		repos.DiskLib = libvirt.NewDiskRepository(repos.LibvirtClient)
	}

	// Initialize guest agent and PCI repositories
	repos.GuestAgentRepo = libvirt.NewGuestAgentRepository(repos.LibvirtClient)
	repos.PCIRepo = libvirt.NewPCIRepository(repos.LibvirtClient)

	return repos, nil
}

// Close closes all repositories
func (r *Repositories) Close() error {
	if r.LibvirtClient != nil {
		r.LibvirtClient.Disconnect()
	}
	return nil
}
