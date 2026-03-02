package repository

import (
	"context"
	"io"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// NodeRepository defines operations for node storage
type NodeRepository interface {
	GetAll(ctx context.Context) ([]*model.HostNode, error)
	GetByID(ctx context.Context, id string) (*model.HostNode, error)
	GetByName(ctx context.Context, name string) (*model.HostNode, error)
	Reboot(ctx context.Context, id string) error
	Shutdown(ctx context.Context, id string) error
}

// VMRepository defines operations for VM storage
type VMRepository interface {
	GetAll(ctx context.Context) ([]*model.VM, error)
	GetByNode(ctx context.Context, node string) ([]*model.VM, error)
	GetByID(ctx context.Context, id string) (*model.VM, error)
	GetByVMID(ctx context.Context, vmid int) (*model.VM, error)
	Create(ctx context.Context, req *model.VMCreateRequest) (*model.VM, error)
	Update(ctx context.Context, vmid int, req *model.VMUpdateRequest) (*model.VM, error)
	Delete(ctx context.Context, vmid int) error
	Clone(ctx context.Context, req *model.VMCloneRequest, progressFunc func(int, string)) (*model.VM, error)
	// Actions
	Start(ctx context.Context, vmid int) error
	Stop(ctx context.Context, vmid int) error
	Shutdown(ctx context.Context, vmid int) error
	Pause(ctx context.Context, vmid int) error
	Resume(ctx context.Context, vmid int) error
	Reboot(ctx context.Context, vmid int) error
	// Console
	GetVNCPort(ctx context.Context, vmid int) (int, error)
}

// ContainerRepository defines operations for container storage
type ContainerRepository interface {
	GetAll(ctx context.Context) ([]*model.Container, error)
	GetByNode(ctx context.Context, node string) ([]*model.Container, error)
	GetByID(ctx context.Context, id string) (*model.Container, error)
	GetByCTID(ctx context.Context, ctid int) (*model.Container, error)
	Create(ctx context.Context, req *model.ContainerCreateRequest) (*model.Container, error)
	Update(ctx context.Context, ctid int, req *model.ContainerUpdateRequest) (*model.Container, error)
	Delete(ctx context.Context, ctid int) error
	// Actions
	Start(ctx context.Context, ctid int) error
	Stop(ctx context.Context, ctid int) error
	Shutdown(ctx context.Context, ctid int) error
	Pause(ctx context.Context, ctid int) error
	Resume(ctx context.Context, ctid int) error
	Reboot(ctx context.Context, ctid int) error
}

// StackRepository defines operations for Docker Compose stack management
type StackRepository interface {
	GetAll(ctx context.Context) ([]*model.DockerStack, error)
	GetByID(ctx context.Context, id string) (*model.DockerStack, error)
	Create(ctx context.Context, req *model.StackCreateRequest) (*model.DockerStack, error)
	Update(ctx context.Context, id string, req *model.StackUpdateRequest) (*model.DockerStack, error)
	Delete(ctx context.Context, id string) error
	Start(ctx context.Context, id string) error        // docker compose up -d
	Stop(ctx context.Context, id string) error         // docker compose stop
	Restart(ctx context.Context, id string) error      // docker compose restart
	UpdateImages(ctx context.Context, id string) error // docker compose pull && up -d
	Down(ctx context.Context, id string) error         // docker compose down
}

// ISORepository defines operations for ISO storage
type ISORepository interface {
	GetAll(ctx context.Context) ([]*model.ISOImage, error)
	GetByID(ctx context.Context, id string) (*model.ISOImage, error)
	Upload(ctx context.Context, name string, reader io.Reader, size int64) (*model.ISOImage, error)
	Delete(ctx context.Context, id string) error
	GetStoragePools(ctx context.Context) ([]*model.StoragePool, error)
}

// SnapshotRepository defines operations for snapshot metadata storage
type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *model.Snapshot) error
	GetByID(ctx context.Context, vmid int, id string) (*model.Snapshot, error)
	ListByVMID(ctx context.Context, vmid int) ([]*model.Snapshot, error)
	Update(ctx context.Context, snapshot *model.Snapshot) error
	Delete(ctx context.Context, vmid int, id string) error
	DeleteWithChildren(ctx context.Context, vmid int, id string) error
	UpdateStatus(ctx context.Context, id string, vmid int, status model.SnapshotStatus) error
	UpdateSize(ctx context.Context, id string, vmid int, sizeBytes int64) error
	Exists(ctx context.Context, vmid int, id string) bool
	GetTree(ctx context.Context, vmid int) (*model.SnapshotTree, error)
}

// BackupRepository defines operations for backup metadata storage
type BackupRepository interface {
	Create(ctx context.Context, backup *model.Backup) error
	GetByID(ctx context.Context, id string) (*model.Backup, error)
	List(ctx context.Context, vmid int, status model.BackupStatus, storagePool string) ([]*model.Backup, error)
	Update(ctx context.Context, backup *model.Backup) error
	UpdateStatus(ctx context.Context, id string, status model.BackupStatus, errorMessage string) error
	Delete(ctx context.Context, id string) error
	GetExpired(ctx context.Context) ([]*model.Backup, error)
}

// BackupScheduleRepository defines operations for backup schedule storage
type BackupScheduleRepository interface {
	Create(ctx context.Context, schedule *model.BackupSchedule) error
	GetByID(ctx context.Context, id string) (*model.BackupSchedule, error)
	List(ctx context.Context, entityType string, entityID int) ([]*model.BackupSchedule, error)
	Update(ctx context.Context, schedule *model.BackupSchedule) error
	UpdateRunInfo(ctx context.Context, id string, lastRunAt, nextRunAt string, incrementBackups bool) error
	Delete(ctx context.Context, id string) error
	GetDueSchedules(ctx context.Context) ([]*model.BackupSchedule, error)
}

// StoragePoolRepository defines operations for storage pool storage
type StoragePoolRepository interface {
	Create(ctx context.Context, pool *model.StoragePool) error
	GetByID(ctx context.Context, id string) (*model.StoragePool, error)
	List(ctx context.Context, poolType model.StorageType, status model.StorageStatus, enabledOnly bool) ([]*model.StoragePool, error)
	Update(ctx context.Context, pool *model.StoragePool) error
	UpdateStats(ctx context.Context, id string, capacity, used, available int64, diskCount int) error
	Delete(ctx context.Context, id string) error
}

// StorageDiskRepository defines operations for storage disk storage
type StorageDiskRepository interface {
	Create(ctx context.Context, disk *model.StorageDisk) error
	GetByID(ctx context.Context, id string) (*model.StorageDisk, error)
	List(ctx context.Context, poolID string, vmid int, unassignedOnly bool) ([]*model.StorageDisk, error)
	Update(ctx context.Context, disk *model.StorageDisk) error
	Delete(ctx context.Context, id string) error
}

// NetworkRepository defines operations for virtual network storage
type NetworkRepository interface {
	Create(ctx context.Context, network *model.VirtualNetwork) error
	GetByID(ctx context.Context, id string) (*model.VirtualNetwork, error)
	List(ctx context.Context, networkType model.VirtualNetworkType, status model.NetworkStatus) ([]*model.VirtualNetwork, error)
	Update(ctx context.Context, network *model.VirtualNetwork) error
	Delete(ctx context.Context, id string) error
	UpdateInterfaceCount(ctx context.Context, id string, count int) error
}

// LibvirtNetworkRepository defines the libvirt-side network lifecycle operations.
// It complements NetworkRepository (SQLite metadata) with actual host networking.
type LibvirtNetworkRepository interface {
	// CreateNetwork defines and starts a libvirt virtual network.
	CreateNetwork(network *model.VirtualNetwork) error
	// DeleteNetwork stops and undefines a libvirt virtual network by name.
	DeleteNetwork(name string) error
	// GetDHCPLeases returns active DHCP leases from libvirt/dnsmasq.
	GetDHCPLeases(networkName string) ([]*model.DHCPLease, error)
	// AddStaticDHCPLease adds a static host entry to the libvirt network's DHCP.
	AddStaticDHCPLease(networkName, mac, ip, hostname string) error
	// RemoveStaticDHCPLease removes a static host entry from the libvirt network.
	RemoveStaticDHCPLease(networkName, mac string) error
	// IsActive reports whether the named libvirt network is running.
	IsActive(networkName string) (bool, error)
}

// FirewallGroupRepository defines operations for firewall group storage
type FirewallGroupRepository interface {
	Create(ctx context.Context, group *model.FirewallGroup) error
	GetByID(ctx context.Context, id string) (*model.FirewallGroup, error)
	List(ctx context.Context, scopeType, scopeID string) ([]*model.FirewallGroup, error)
	Update(ctx context.Context, group *model.FirewallGroup) error
	Delete(ctx context.Context, id string) error
}

// DHCPLeaseRepository defines operations for static DHCP lease storage
type DHCPLeaseRepository interface {
	Create(ctx context.Context, lease *model.DHCPLease) error
	Delete(ctx context.Context, networkID, mac string) error
	List(ctx context.Context, networkID string) ([]*model.DHCPLease, error)
}

// NetworkInterfaceRepository defines operations for network interface storage
type NetworkInterfaceRepository interface {
	Create(ctx context.Context, iface *model.NetworkInterface) error
	GetByID(ctx context.Context, id string) (*model.NetworkInterface, error)
	List(ctx context.Context, networkID string, entityID int, entityType string) ([]*model.NetworkInterface, error)
	Update(ctx context.Context, iface *model.NetworkInterface) error
	Delete(ctx context.Context, id string) error
}

// FirewallRuleRepository defines operations for firewall rule storage
type FirewallRuleRepository interface {
	Create(ctx context.Context, rule *model.FirewallRule) error
	GetByID(ctx context.Context, id string) (*model.FirewallRule, error)
	List(ctx context.Context, scopeType, scopeID string, enabledOnly bool) ([]*model.FirewallRule, error)
	Update(ctx context.Context, rule *model.FirewallRule) error
	Delete(ctx context.Context, id string) error
}

// TaskRepository defines operations for task tracking storage
type TaskRepository interface {
	Create(ctx context.Context, task *model.Task) error
	GetByID(ctx context.Context, id string) (*model.Task, error)
	List(ctx context.Context, filter model.TaskFilter) ([]*model.Task, error)
	Update(ctx context.Context, task *model.Task) error
	UpdateProgress(ctx context.Context, id string, progress int, message string) error
	UpdateStatus(ctx context.Context, id string, status model.TaskStatus, message string) error
	Delete(ctx context.Context, id string) error
	DeleteCompleted(ctx context.Context, olderThan time.Duration) (int64, error)
}

// PCIRepository defines operations for PCI device management
type PCIRepository interface {
	ListHostDevices(ctx context.Context) ([]model.PCIDevice, error)
	GetDevicesByIOMMUGroup(ctx context.Context) (map[int][]model.PCIDevice, error)
	IsIOMMUAvailable() bool
	IsVFIOAvailable() bool
	AttachPCIDeviceToVM(ctx context.Context, vmid int, pciAddr string) error
	DetachPCIDeviceFromVM(ctx context.Context, vmid int, pciAddr string) error
}

// LibvirtDiskRepository defines libvirt disk operations for VMs
type LibvirtDiskRepository interface {
	// AttachDisk attaches a disk to a VM and returns the target device name
	AttachDisk(ctx context.Context, vmid int, diskPath string, bus model.DiskBus, readonly bool) (string, error)
	// DetachDisk detaches a disk from a VM
	DetachDisk(ctx context.Context, vmid int, target string) error
	// ListVMDisks returns all disks attached to a VM
	ListVMDisks(ctx context.Context, vmid int) ([]model.VMDisk, error)
	// CreateDiskImage creates a new disk image
	CreateDiskImage(path string, sizeGB float64, format model.DiskFormat, sparse bool) error
	// ResizeDiskImage resizes a disk image
	ResizeDiskImage(path string, newSizeGB float64) error
	// GetDiskInfo returns information about a disk image
	GetDiskInfo(path string) (sizeBytes int64, format string, err error)
	// IsRootDisk checks if a disk is the root/boot disk
	IsRootDisk(ctx context.Context, vmid int, target string) (bool, error)
}

// ProxyRepository defines operations for reverse proxy host storage and uptime monitoring.
type ProxyRepository interface {
	// Proxy hosts
	Create(ctx context.Context, host *model.ProxyHost) error
	GetByID(ctx context.Context, id string) (*model.ProxyHost, error)
	GetByDomain(ctx context.Context, domain string) (*model.ProxyHost, error)
	List(ctx context.Context) ([]*model.ProxyHost, error)
	Update(ctx context.Context, host *model.ProxyHost) error
	Delete(ctx context.Context, id string) error
	SaveCert(ctx context.Context, cert *model.ProxyCert) error
	GetCert(ctx context.Context, proxyHostID string) (*model.ProxyCert, error)

	// Uptime monitors
	CreateMonitor(ctx context.Context, m *model.UptimeMonitor) error
	GetMonitorByID(ctx context.Context, id string) (*model.UptimeMonitor, error)
	ListMonitors(ctx context.Context) ([]*model.UptimeMonitor, error)
	ListEnabledMonitors(ctx context.Context) ([]*model.UptimeMonitor, error)
	UpdateMonitor(ctx context.Context, m *model.UptimeMonitor) error
	DeleteMonitor(ctx context.Context, id string) error

	// Uptime results
	LogUptimeResult(ctx context.Context, r *model.UptimeResult) error
	GetUptimeHistory(ctx context.Context, monitorID string, limit int) ([]*model.UptimeResult, error)
	GetUptimeStats(ctx context.Context, monitorID string) (*model.UptimeStats, error)
	PruneOldResults(ctx context.Context, olderThan time.Duration) (int64, error)
}

// GuestAgentRepository defines operations for QEMU guest agent communication
type GuestAgentRepository interface {
	// Ping checks if the guest agent is responsive
	Ping(ctx context.Context, vmid int) bool
	// GetNetworkInterfaces retrieves network interfaces and IP addresses from the guest
	GetNetworkInterfaces(ctx context.Context, vmid int) ([]model.GuestNetworkInterface, error)
	// GetPrimaryIP returns the primary IPv4 address from the guest agent
	GetPrimaryIP(ctx context.Context, vmid int) string
	// FreezeFilesystems freezes guest filesystems for consistent backups
	FreezeFilesystems(ctx context.Context, vmid int) (int, error)
	// ThawFilesystems thaws guest filesystems after backup
	ThawFilesystems(ctx context.Context, vmid int) (int, error)
}
