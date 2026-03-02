package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	libvirtRepo "github.com/doomedramen/lab/apps/api/internal/repository/libvirt"
)

// StorageService handles storage pool and disk management
type StorageService struct {
	poolRepo repository.StoragePoolRepository
	diskRepo repository.StorageDiskRepository
	diskLib  repository.LibvirtDiskRepository
}

// NewStorageService creates a new storage service
func NewStorageService(
	poolRepo repository.StoragePoolRepository,
	diskRepo repository.StorageDiskRepository,
	diskLib repository.LibvirtDiskRepository,
) *StorageService {
	return &StorageService{
		poolRepo: poolRepo,
		diskRepo: diskRepo,
		diskLib:  diskLib,
	}
}

// ListStoragePools returns storage pools with optional filters
func (s *StorageService) ListStoragePools(ctx context.Context, poolType labv1.StorageType, status labv1.StorageStatus, enabledOnly bool) ([]*labv1.StoragePool, int32, error) {
	modelType := protoToModelStorageType(poolType)
	modelStatus := protoToModelStorageStatus(status)

	pools, err := s.poolRepo.List(ctx, modelType, modelStatus, enabledOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list storage pools: %w", err)
	}

	var protoPools []*labv1.StoragePool
	for _, p := range pools {
		protoPools = append(protoPools, s.modelToProto(p))
	}

	return protoPools, int32(len(protoPools)), nil
}

// GetStoragePool returns details of a specific storage pool
func (s *StorageService) GetStoragePool(ctx context.Context, id string) (*labv1.StoragePool, error) {
	pool, err := s.poolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage pool: %w", err)
	}
	if pool == nil {
		return nil, fmt.Errorf("storage pool not found: %s", id)
	}
	return s.modelToProto(pool), nil
}

// CreateStoragePool creates a new storage pool
func (s *StorageService) CreateStoragePool(ctx context.Context, req *labv1.CreateStoragePoolRequest) (*labv1.StoragePool, error) {
	// Validate storage type
	if req.Type == labv1.StorageType_STORAGE_TYPE_UNSPECIFIED {
		return nil, fmt.Errorf("storage type is required")
	}

	// Create pool record
	pool := &model.StoragePool{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Type:           protoToModelStorageType(req.Type),
		Status:         model.StorageStatusActive,
		Path:           req.Path,
		Options:        req.Options,
		Enabled:        req.Enabled,
		Description:    req.Description,
		CreatedAt:      time.Now().Format(time.RFC3339),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}

	// Initialize backend
	backend, err := libvirtRepo.GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage backend: %w", err)
	}

	if err := backend.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Get initial stats
	capacity, used, available, err := backend.GetStats()
	if err != nil {
		slog.Warn("Failed to get initial storage stats", "error", err)
		capacity, used, available = 0, 0, 0
	}

	pool.CapacityBytes = capacity
	pool.UsedBytes = used
	pool.AvailableBytes = available

	if err := s.poolRepo.Create(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to create storage pool: %w", err)
	}

	return s.modelToProto(pool), nil
}

// UpdateStoragePool updates an existing storage pool
func (s *StorageService) UpdateStoragePool(ctx context.Context, req *labv1.UpdateStoragePoolRequest) (*labv1.StoragePool, error) {
	pool, err := s.poolRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage pool: %w", err)
	}
	if pool == nil {
		return nil, fmt.Errorf("storage pool not found: %s", req.Id)
	}

	// Update fields
	if req.Name != "" {
		pool.Name = req.Name
	}
	if req.Status != labv1.StorageStatus_STORAGE_STATUS_UNSPECIFIED {
		pool.Status = protoToModelStorageStatus(req.Status)
	}
	if req.Options != nil {
		pool.Options = req.Options
	}
	if req.Description != "" {
		pool.Description = req.Description
	}
	pool.Enabled = req.Enabled
	pool.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.poolRepo.Update(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to update storage pool: %w", err)
	}

	return s.modelToProto(pool), nil
}

// DeleteStoragePool deletes a storage pool
func (s *StorageService) DeleteStoragePool(ctx context.Context, id string, force bool) error {
	pool, err := s.poolRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get storage pool: %w", err)
	}
	if pool == nil {
		return fmt.Errorf("storage pool not found: %s", id)
	}

	// Check if pool has disks
	disks, err := s.diskRepo.List(ctx, id, 0, false)
	if err != nil {
		return fmt.Errorf("failed to check for disks: %w", err)
	}

	if len(disks) > 0 && !force {
		return fmt.Errorf("storage pool has %d disks, use force=true to delete", len(disks))
	}

	if err := s.poolRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete storage pool: %w", err)
	}

	return nil
}

// RefreshStoragePool refreshes pool statistics
func (s *StorageService) RefreshStoragePool(ctx context.Context, id string) (*labv1.StoragePool, error) {
	pool, err := s.poolRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage pool: %w", err)
	}
	if pool == nil {
		return nil, fmt.Errorf("storage pool not found: %s", id)
	}

	// Get backend and refresh stats
	backend, err := libvirtRepo.GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage backend: %w", err)
	}

	capacity, used, available, err := backend.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage stats: %w", err)
	}

	pool.CapacityBytes = capacity
	pool.UsedBytes = used
	pool.AvailableBytes = available

	if err := s.poolRepo.Update(ctx, pool); err != nil {
		return nil, fmt.Errorf("failed to update storage pool: %w", err)
	}

	return s.modelToProto(pool), nil
}

// ListStorageDisks returns disks in a storage pool
func (s *StorageService) ListStorageDisks(ctx context.Context, poolID string, vmid int, unassignedOnly bool) ([]*labv1.StorageDisk, int32, error) {
	disks, err := s.diskRepo.List(ctx, poolID, vmid, unassignedOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list storage disks: %w", err)
	}

	var protoDisks []*labv1.StorageDisk
	for _, d := range disks {
		protoDisks = append(protoDisks, s.diskModelToProto(d))
	}

	return protoDisks, int32(len(protoDisks)), nil
}

// CreateStorageDisk creates a new disk in a storage pool
func (s *StorageService) CreateStorageDisk(ctx context.Context, req *labv1.CreateStorageDiskRequest) (*labv1.StorageDisk, error) {
	// Get pool
	pool, err := s.poolRepo.GetByID(ctx, req.PoolId)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage pool: %w", err)
	}
	if pool == nil {
		return nil, fmt.Errorf("storage pool not found: %s", req.PoolId)
	}

	// Get backend
	backend, err := libvirtRepo.GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage backend: %w", err)
	}

	// Create disk
	diskName := req.Name
	if diskName == "" {
		diskName = fmt.Sprintf("disk-%s", uuid.New().String()[:8])
	}

	format := protoToModelDiskFormat(req.Format)
	bus := protoToModelDiskBus(req.Bus)

	path, err := backend.CreateDisk(diskName, req.SizeBytes, format, req.Sparse)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk: %w", err)
	}

	// Create disk record
	disk := &model.StorageDisk{
		ID:          uuid.New().String(),
		PoolID:      req.PoolId,
		Name:        diskName,
		SizeBytes:   req.SizeBytes,
		Format:      format,
		Bus:         bus,
		Path:        path,
		Sparse:      req.Sparse,
		Description: req.Description,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := s.diskRepo.Create(ctx, disk); err != nil {
		// Clean up created file
		backend.DeleteDisk(path)
		return nil, fmt.Errorf("failed to create disk record: %w", err)
	}

	return s.diskModelToProto(disk), nil
}

// ResizeStorageDisk resizes a disk
func (s *StorageService) ResizeStorageDisk(ctx context.Context, diskID string, newSizeBytes int64, shrink bool) (*labv1.StorageDisk, error) {
	disk, err := s.diskRepo.GetByID(ctx, diskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk: %w", err)
	}
	if disk == nil {
		return nil, fmt.Errorf("disk not found: %s", diskID)
	}

	// Get pool and backend
	pool, err := s.poolRepo.GetByID(ctx, disk.PoolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}

	backend, err := libvirtRepo.GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}

	// Resize disk
	if err := backend.ResizeDisk(disk.Path, newSizeBytes); err != nil {
		return nil, fmt.Errorf("failed to resize disk: %w", err)
	}

	// Update record
	disk.SizeBytes = newSizeBytes
	if err := s.diskRepo.Update(ctx, disk); err != nil {
		return nil, fmt.Errorf("failed to update disk: %w", err)
	}

	return s.diskModelToProto(disk), nil
}

// DeleteStorageDisk deletes a disk
func (s *StorageService) DeleteStorageDisk(ctx context.Context, diskID string, purge bool) error {
	disk, err := s.diskRepo.GetByID(ctx, diskID)
	if err != nil {
		return fmt.Errorf("failed to get disk: %w", err)
	}
	if disk == nil {
		return fmt.Errorf("disk not found: %s", diskID)
	}

	// Get pool and backend
	pool, err := s.poolRepo.GetByID(ctx, disk.PoolID)
	if err != nil {
		return fmt.Errorf("failed to get pool: %w", err)
	}

	backend, err := libvirtRepo.GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	// Delete disk file
	if err := backend.DeleteDisk(disk.Path); err != nil {
		slog.Warn("Failed to delete disk file", "path", disk.Path, "error", err)
	}

	// Delete record
	if err := s.diskRepo.Delete(ctx, diskID); err != nil {
		return fmt.Errorf("failed to delete disk record: %w", err)
	}

	return nil
}

// MoveStorageDisk moves a disk to another pool
func (s *StorageService) MoveStorageDisk(ctx context.Context, diskID, targetPoolID string, targetFormat labv1.DiskFormat, deleteOriginal bool) (*labv1.StorageDisk, string, error) {
	disk, err := s.diskRepo.GetByID(ctx, diskID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get disk: %w", err)
	}
	if disk == nil {
		return nil, "", fmt.Errorf("disk not found: %s", diskID)
	}

	// Get source and target pools
	sourcePool, err := s.poolRepo.GetByID(ctx, disk.PoolID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get source pool: %w", err)
	}

	targetPool, err := s.poolRepo.GetByID(ctx, targetPoolID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get target pool: %w", err)
	}

	// Get backends
	sourceBackend, err := libvirtRepo.GetStorageBackend(sourcePool.Type, sourcePool.Path, sourcePool.Options)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get source backend: %w", err)
	}

	// Create target path
	targetPath := fmt.Sprintf("%s/%s", targetPool.Path, disk.Name)

	// Move disk
	if err := sourceBackend.MoveDisk(disk.Path, targetPath, protoToModelDiskFormat(targetFormat)); err != nil {
		return nil, "", fmt.Errorf("failed to move disk: %w", err)
	}

	// Update record
	disk.PoolID = targetPoolID
	disk.Path = targetPath
	if targetFormat != labv1.DiskFormat_DISK_FORMAT_UNSPECIFIED {
		disk.Format = protoToModelDiskFormat(targetFormat)
	}

	if err := s.diskRepo.Update(ctx, disk); err != nil {
		return nil, "", fmt.Errorf("failed to update disk: %w", err)
	}

	taskID := uuid.New().String()
	return s.diskModelToProto(disk), taskID, nil
}

// modelToProto converts model.StoragePool to labv1.StoragePool
func (s *StorageService) modelToProto(pool *model.StoragePool) *labv1.StoragePool {
	if pool == nil {
		return nil
	}

	return &labv1.StoragePool{
		Id:             pool.ID,
		Name:           pool.Name,
		Type:           modelStorageTypeToProto(pool.Type),
		Status:         modelStorageStatusToProto(pool.Status),
		Path:           pool.Path,
		CapacityBytes:  pool.CapacityBytes,
		UsedBytes:      pool.UsedBytes,
		AvailableBytes: pool.AvailableBytes,
		UsagePercent:   pool.UsagePercent,
		Options:        pool.Options,
		CreatedAt:      pool.CreatedAt,
		UpdatedAt:      pool.UpdatedAt,
		Enabled:        pool.Enabled,
		DiskCount:      int32(pool.DiskCount),
		Description:    pool.Description,
	}
}

// diskModelToProto converts model.StorageDisk to labv1.StorageDisk
func (s *StorageService) diskModelToProto(disk *model.StorageDisk) *labv1.StorageDisk {
	if disk == nil {
		return nil
	}

	return &labv1.StorageDisk{
		Id:          disk.ID,
		PoolId:      disk.PoolID,
		Name:        disk.Name,
		SizeBytes:   disk.SizeBytes,
		Format:      modelDiskFormatToProto(disk.Format),
		Bus:         modelDiskBusToProto(disk.Bus),
		Vmid:        int32(disk.VMID),
		Path:        disk.Path,
		UsedBytes:   disk.UsedBytes,
		Sparse:      disk.Sparse,
		CreatedAt:   disk.CreatedAt,
		Description: disk.Description,
	}
}

// Helper functions for type conversion
func protoToModelStorageType(t labv1.StorageType) model.StorageType {
	switch t {
	case labv1.StorageType_STORAGE_TYPE_DIR:
		return model.StorageTypeDir
	case labv1.StorageType_STORAGE_TYPE_LVM:
		return model.StorageTypeLVM
	case labv1.StorageType_STORAGE_TYPE_ZFS:
		return model.StorageTypeZFS
	case labv1.StorageType_STORAGE_TYPE_NFS:
		return model.StorageTypeNFS
	case labv1.StorageType_STORAGE_TYPE_ISCSI:
		return model.StorageTypeISCSI
	case labv1.StorageType_STORAGE_TYPE_CEPH:
		return model.StorageTypeCeph
	case labv1.StorageType_STORAGE_TYPE_GLUSTER:
		return model.StorageTypeGluster
	default:
		return model.StorageTypeDir
	}
}

func modelStorageTypeToProto(t model.StorageType) labv1.StorageType {
	switch t {
	case model.StorageTypeDir:
		return labv1.StorageType_STORAGE_TYPE_DIR
	case model.StorageTypeLVM:
		return labv1.StorageType_STORAGE_TYPE_LVM
	case model.StorageTypeZFS:
		return labv1.StorageType_STORAGE_TYPE_ZFS
	case model.StorageTypeNFS:
		return labv1.StorageType_STORAGE_TYPE_NFS
	case model.StorageTypeISCSI:
		return labv1.StorageType_STORAGE_TYPE_ISCSI
	case model.StorageTypeCeph:
		return labv1.StorageType_STORAGE_TYPE_CEPH
	case model.StorageTypeGluster:
		return labv1.StorageType_STORAGE_TYPE_GLUSTER
	default:
		return labv1.StorageType_STORAGE_TYPE_UNSPECIFIED
	}
}

func protoToModelStorageStatus(s labv1.StorageStatus) model.StorageStatus {
	switch s {
	case labv1.StorageStatus_STORAGE_STATUS_ACTIVE:
		return model.StorageStatusActive
	case labv1.StorageStatus_STORAGE_STATUS_INACTIVE:
		return model.StorageStatusInactive
	case labv1.StorageStatus_STORAGE_STATUS_MAINTENANCE:
		return model.StorageStatusMaintenance
	case labv1.StorageStatus_STORAGE_STATUS_ERROR:
		return model.StorageStatusError
	default:
		return ""
	}
}

func modelStorageStatusToProto(s model.StorageStatus) labv1.StorageStatus {
	switch s {
	case model.StorageStatusActive:
		return labv1.StorageStatus_STORAGE_STATUS_ACTIVE
	case model.StorageStatusInactive:
		return labv1.StorageStatus_STORAGE_STATUS_INACTIVE
	case model.StorageStatusMaintenance:
		return labv1.StorageStatus_STORAGE_STATUS_MAINTENANCE
	case model.StorageStatusError:
		return labv1.StorageStatus_STORAGE_STATUS_ERROR
	default:
		return labv1.StorageStatus_STORAGE_STATUS_UNSPECIFIED
	}
}

func protoToModelDiskFormat(f labv1.DiskFormat) model.DiskFormat {
	switch f {
	case labv1.DiskFormat_DISK_FORMAT_QCOW2:
		return model.DiskFormatQCOW2
	case labv1.DiskFormat_DISK_FORMAT_RAW:
		return model.DiskFormatRaw
	case labv1.DiskFormat_DISK_FORMAT_VMDK:
		return model.DiskFormatVMDK
	case labv1.DiskFormat_DISK_FORMAT_VDI:
		return model.DiskFormatVDI
	case labv1.DiskFormat_DISK_FORMAT_VHDX:
		return model.DiskFormatVHDX
	default:
		return model.DiskFormatQCOW2
	}
}

func modelDiskFormatToProto(f model.DiskFormat) labv1.DiskFormat {
	switch f {
	case model.DiskFormatQCOW2:
		return labv1.DiskFormat_DISK_FORMAT_QCOW2
	case model.DiskFormatRaw:
		return labv1.DiskFormat_DISK_FORMAT_RAW
	case model.DiskFormatVMDK:
		return labv1.DiskFormat_DISK_FORMAT_VMDK
	case model.DiskFormatVDI:
		return labv1.DiskFormat_DISK_FORMAT_VDI
	case model.DiskFormatVHDX:
		return labv1.DiskFormat_DISK_FORMAT_VHDX
	default:
		return labv1.DiskFormat_DISK_FORMAT_UNSPECIFIED
	}
}

func protoToModelDiskBus(b labv1.DiskBus) model.DiskBus {
	switch b {
	case labv1.DiskBus_DISK_BUS_VIRTIO:
		return model.DiskBusVirtIO
	case labv1.DiskBus_DISK_BUS_SATA:
		return model.DiskBusSATA
	case labv1.DiskBus_DISK_BUS_SCSI:
		return model.DiskBusSCSI
	case labv1.DiskBus_DISK_BUS_IDE:
		return model.DiskBusIDE
	case labv1.DiskBus_DISK_BUS_USB:
		return model.DiskBusUSB
	case labv1.DiskBus_DISK_BUS_NVME:
		return model.DiskBusNVMe
	default:
		return model.DiskBusVirtIO
	}
}

func modelDiskBusToProto(b model.DiskBus) labv1.DiskBus {
	switch b {
	case model.DiskBusVirtIO:
		return labv1.DiskBus_DISK_BUS_VIRTIO
	case model.DiskBusSATA:
		return labv1.DiskBus_DISK_BUS_SATA
	case model.DiskBusSCSI:
		return labv1.DiskBus_DISK_BUS_SCSI
	case model.DiskBusIDE:
		return labv1.DiskBus_DISK_BUS_IDE
	case model.DiskBusUSB:
		return labv1.DiskBus_DISK_BUS_USB
	case model.DiskBusNVMe:
		return labv1.DiskBus_DISK_BUS_NVME
	default:
		return labv1.DiskBus_DISK_BUS_UNSPECIFIED
	}
}

// ListVMDisks returns all disks attached to a VM
func (s *StorageService) ListVMDisks(ctx context.Context, vmid int) ([]*labv1.VMDisk, error) {
	if s.diskLib == nil {
		return nil, fmt.Errorf("disk libvirt repository not available")
	}

	disks, err := s.diskLib.ListVMDisks(ctx, vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to list VM disks: %w", err)
	}

	var protoDisks []*labv1.VMDisk
	for _, d := range disks {
		protoDisks = append(protoDisks, modelVMDiskToProto(d))
	}

	return protoDisks, nil
}

// AttachDiskToVM attaches a new disk to a VM
func (s *StorageService) AttachDiskToVM(ctx context.Context, vmid int, diskPath string, bus model.DiskBus, readonly bool) (string, error) {
	if s.diskLib == nil {
		return "", fmt.Errorf("disk libvirt repository not available")
	}

	target, err := s.diskLib.AttachDisk(ctx, vmid, diskPath, bus, readonly)
	if err != nil {
		return "", fmt.Errorf("failed to attach disk: %w", err)
	}

	return target, nil
}

// DetachDiskFromVM detaches a disk from a VM
func (s *StorageService) DetachDiskFromVM(ctx context.Context, vmid int, target string, deleteDisk bool) error {
	if s.diskLib == nil {
		return fmt.Errorf("disk libvirt repository not available")
	}

	// Check if this is the root disk
	isRoot, err := s.diskLib.IsRootDisk(ctx, vmid, target)
	if err != nil {
		return fmt.Errorf("failed to check if disk is root: %w", err)
	}
	if isRoot {
		return fmt.Errorf("cannot detach root disk")
	}

	// Get disk path before detaching (for deletion)
	disks, err := s.diskLib.ListVMDisks(ctx, vmid)
	var diskPath string
	for _, d := range disks {
		if d.Target == target {
			diskPath = d.Path
			break
		}
	}

	// Detach the disk
	if err := s.diskLib.DetachDisk(ctx, vmid, target); err != nil {
		return fmt.Errorf("failed to detach disk: %w", err)
	}

	// Delete the disk file if requested
	if deleteDisk && diskPath != "" {
		if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("Failed to delete disk file after detach", "path", diskPath, "error", err)
		}
	}

	return nil
}

// ResizeVMDisk resizes a disk attached to a VM
func (s *StorageService) ResizeVMDisk(ctx context.Context, vmid int, target string, newSizeGB float64) error {
	if s.diskLib == nil {
		return fmt.Errorf("disk libvirt repository not available")
	}

	// Get disk path
	disks, err := s.diskLib.ListVMDisks(ctx, vmid)
	if err != nil {
		return fmt.Errorf("failed to list VM disks: %w", err)
	}

	var diskPath string
	for _, d := range disks {
		if d.Target == target {
			diskPath = d.Path
			break
		}
	}

	if diskPath == "" {
		return fmt.Errorf("disk with target %s not found", target)
	}

	// Resize the disk image
	if err := s.diskLib.ResizeDiskImage(diskPath, newSizeGB); err != nil {
		return fmt.Errorf("failed to resize disk: %w", err)
	}

	return nil
}

// modelVMDiskToProto converts model.VMDisk to labv1.VMDisk
func modelVMDiskToProto(d model.VMDisk) *labv1.VMDisk {
	return &labv1.VMDisk{
		Id:        d.ID,
		Vmid:      int32(d.VMID),
		Target:    d.Target,
		Path:      d.Path,
		SizeBytes: d.SizeBytes,
		Bus:       modelDiskBusToProto(d.Bus),
		Format:    modelDiskFormatToProto(d.Format),
		Readonly:  d.Readonly,
		BootOrder: int32(d.BootOrder),
	}
}

// ListStoragePoolsForAlerts returns all storage pools for alert evaluation
// This method returns model types to avoid proto dependency in alert service
func (s *StorageService) ListStoragePoolsForAlerts(ctx context.Context) ([]*model.StoragePool, error) {
	// Use empty strings for type/status to get all pools
	pools, err := s.poolRepo.List(ctx, "", "", true)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage pools: %w", err)
	}
	return pools, nil
}
