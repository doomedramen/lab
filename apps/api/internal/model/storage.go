package model

// StorageType represents the type of storage backend
type StorageType string

const (
	StorageTypeDir     StorageType = "dir"
	StorageTypeLVM     StorageType = "lvm"
	StorageTypeZFS     StorageType = "zfs"
	StorageTypeNFS     StorageType = "nfs"
	StorageTypeISCSI   StorageType = "iscsi"
	StorageTypeCeph    StorageType = "ceph"
	StorageTypeGluster StorageType = "gluster"
)

// StorageStatus represents the operational status of a storage pool
type StorageStatus string

const (
	StorageStatusActive      StorageStatus = "active"
	StorageStatusInactive    StorageStatus = "inactive"
	StorageStatusMaintenance StorageStatus = "maintenance"
	StorageStatusError       StorageStatus = "error"
)

// DiskFormat represents the disk image format
type DiskFormat string

const (
	DiskFormatQCOW2 DiskFormat = "qcow2"
	DiskFormatRaw   DiskFormat = "raw"
	DiskFormatVMDK  DiskFormat = "vmdk"
	DiskFormatVDI   DiskFormat = "vdi"
	DiskFormatVHDX  DiskFormat = "vhdx"
)

// DiskBus represents the disk bus type
type DiskBus string

const (
	DiskBusVirtIO DiskBus = "virtio"
	DiskBusSATA   DiskBus = "sata"
	DiskBusSCSI   DiskBus = "scsi"
	DiskBusIDE    DiskBus = "ide"
	DiskBusUSB    DiskBus = "usb"
	DiskBusNVMe   DiskBus = "nvme"
)

// StoragePool represents a storage pool
type StoragePool struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           StorageType       `json:"type"`
	Status         StorageStatus     `json:"status"`
	Path           string            `json:"path"`
	CapacityBytes  int64             `json:"capacity_bytes"`
	UsedBytes      int64             `json:"used_bytes"`
	AvailableBytes int64             `json:"available_bytes"`
	UsagePercent   float64           `json:"usage_percent"`
	Options        map[string]string `json:"options,omitempty"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
	Enabled        bool              `json:"enabled"`
	DiskCount      int               `json:"disk_count"`
	Description    string            `json:"description"`
}

// StorageDisk represents a virtual disk in a storage pool
type StorageDisk struct {
	ID          string    `json:"id"`
	PoolID      string    `json:"pool_id"`
	Name        string    `json:"name"`
	SizeBytes   int64     `json:"size_bytes"`
	Format      DiskFormat `json:"format"`
	Bus         DiskBus   `json:"bus"`
	VMID        int       `json:"vmid"`
	Path        string    `json:"path"`
	UsedBytes   int64     `json:"used_bytes"`
	Sparse      bool      `json:"sparse"`
	CreatedAt   string    `json:"created_at"`
	Description string    `json:"description"`
}

// StorageContent represents content in a storage pool
type StorageContent struct {
	ID          string `json:"id"`
	PoolID      string `json:"pool_id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "iso", "disk", "backup", "template"
	SizeBytes   int64  `json:"size_bytes"`
	Format      string `json:"format"`
	CreatedAt   string `json:"created_at"`
	Description string `json:"description"`
}

// StoragePoolCreateRequest represents a request to create a storage pool
type StoragePoolCreateRequest struct {
	Name        string            `json:"name"`
	Type        StorageType       `json:"type"`
	Path        string            `json:"path"`
	Options     map[string]string `json:"options,omitempty"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
}

// StoragePoolUpdateRequest represents a request to update a storage pool
type StoragePoolUpdateRequest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Status      StorageStatus     `json:"status"`
	Options     map[string]string `json:"options,omitempty"`
	Description string            `json:"description"`
	Enabled     *bool             `json:"enabled"`
}

// StorageDiskCreateRequest represents a request to create a disk
type StorageDiskCreateRequest struct {
	PoolID      string     `json:"pool_id"`
	Name        string     `json:"name"`
	SizeBytes   int64      `json:"size_bytes"`
	Format      DiskFormat `json:"format"`
	Bus         DiskBus    `json:"bus"`
	Sparse      bool       `json:"sparse"`
	Description string     `json:"description"`
}

// StorageDiskResizeRequest represents a request to resize a disk
type StorageDiskResizeRequest struct {
	DiskID      string `json:"disk_id"`
	NewSizeBytes int64  `json:"new_size_bytes"`
	Shrink      bool   `json:"shrink"`
}

// StorageDiskMoveRequest represents a request to move a disk
type StorageDiskMoveRequest struct {
	DiskID         string     `json:"disk_id"`
	TargetPoolID   string     `json:"target_pool_id"`
	TargetFormat   DiskFormat `json:"target_format"`
	DeleteOriginal bool       `json:"delete_original"`
}
