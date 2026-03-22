package libvirt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"libvirt.org/go/libvirt"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// ISORepository implements repository.ISORepository using libvirt
type ISORepository struct {
	client libvirtx.LibvirtClient
	mu     sync.RWMutex
	cfg    *config.Config
}

// NewISORepository creates a new libvirt ISO repository
func NewISORepository(client libvirtx.LibvirtClient, cfg *config.Config) *ISORepository {
	// Ensure directory exists
	os.MkdirAll(cfg.Storage.ISODir, 0755)

	return &ISORepository{
		client: client,
		cfg:    cfg,
	}
}

// GetAll returns all ISO images
func (r *ISORepository) GetAll(_ context.Context) ([]*model.ISOImage, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Scan directory for ISO files
	isos, err := r.scanDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to scan ISO directory: %w", err)
	}

	return isos, nil
}

// scanDirectory scans the ISO directory for files
func (r *ISORepository) scanDirectory() ([]*model.ISOImage, error) {
	isoDir := r.cfg.Storage.ISODir

	// Create directory if it doesn't exist
	if err := os.MkdirAll(isoDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(isoDir)
	if err != nil {
		return nil, err
	}

	var isos []*model.ISOImage
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !r.isAllowedExtension(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		id := generateISOID(name)

		iso := &model.ISOImage{
			ID:        id,
			Name:      name,
			Size:      info.Size(),
			Path:      filepath.Join(isoDir, name),
			Status:    "available",
			OS:        detectOS(name),
			CreatedAt: info.ModTime().Format(time.RFC3339),
		}

		isos = append(isos, iso)
	}

	return isos, nil
}

// isAllowedExtension checks if the file has an allowed extension
func (r *ISORepository) isAllowedExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, allowed := range r.cfg.Storage.AllowedExtensions {
		if strings.ToLower(allowed) == ext {
			return true
		}
	}
	return ext == ".iso" // Default fallback
}

// generateISOID creates a unique ID for an ISO file
func generateISOID(name string) string {
	hash := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hash[:8])
}

// detectOS attempts to detect the OS from the ISO filename
func detectOS(name string) string {
	nameLower := strings.ToLower(name)
	switch {
	case strings.Contains(nameLower, "ubuntu"):
		return "Ubuntu"
	case strings.Contains(nameLower, "debian"):
		return "Debian"
	case strings.Contains(nameLower, "centos"):
		return "CentOS"
	case strings.Contains(nameLower, "rocky"):
		return "Rocky Linux"
	case strings.Contains(nameLower, "almalinux"):
		return "AlmaLinux"
	case strings.Contains(nameLower, "fedora"):
		return "Fedora"
	case strings.Contains(nameLower, "rhel"):
		return "RHEL"
	case strings.Contains(nameLower, "windows") || strings.Contains(nameLower, "win"):
		return "Windows"
	case strings.Contains(nameLower, "arch"):
		return "Arch Linux"
	case strings.Contains(nameLower, "alpine"):
		return "Alpine"
	case strings.Contains(nameLower, "freebsd"):
		return "FreeBSD"
	case strings.Contains(nameLower, "proxmox"):
		return "Proxmox"
	default:
		return "Unknown"
	}
}

// GetByID returns an ISO image by ID
func (r *ISORepository) GetByID(_ context.Context, id string) (*model.ISOImage, error) {
	isos, err := r.GetAll(context.Background())
	if err != nil {
		return nil, err
	}

	for _, iso := range isos {
		if iso.ID == id {
			return iso, nil
		}
	}
	return nil, fmt.Errorf("ISO not found")
}

// Upload registers a new ISO file (actual upload handled by Tus)
func (r *ISORepository) Upload(_ context.Context, name string, reader io.Reader, size int64) (*model.ISOImage, error) {
	// This is handled by the Tus handler
	// After upload completes, the file will be picked up by GetAll
	id := generateISOID(name)
	path := filepath.Join(r.cfg.Storage.ISODir, name)

	iso := &model.ISOImage{
		ID:        id,
		Name:      name,
		Size:      size,
		Path:      path,
		Status:    "available",
		OS:        detectOS(name),
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	return iso, nil
}

// Delete removes an ISO file
func (r *ISORepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find the ISO
	isos, err := r.scanDirectory()
	if err != nil {
		return err
	}

	for _, iso := range isos {
		if iso.ID == id {
			// Delete the actual file
			if err := os.Remove(iso.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to delete ISO file: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("ISO not found")
}

// GetStoragePools returns storage pool information
func (r *ISORepository) GetStoragePools(_ context.Context) ([]*model.StoragePool, error) {
	// Try to get libvirt storage pools
	conn, err := r.client.Connection()
	if err != nil {
		// Fallback to filesystem-based pool info
		return r.getFilesystemPools()
	}

	// List storage pools
	pools, err := conn.ListAllStoragePools(0)
	if err != nil {
		// Fallback to filesystem-based pool info
		return r.getFilesystemPools()
	}

	var storagePools []*model.StoragePool

	for _, pool := range pools {
		poolInfo, err := r.poolToStoragePool(&pool)
		if err == nil {
			storagePools = append(storagePools, poolInfo)
		}
		pool.Free()
	}

	// Always include our configured ISO directory pool
	isoPool, err := r.getISOPool()
	if err == nil {
		// Check if already included
		found := false
		for _, p := range storagePools {
			if p.Path == isoPool.Path {
				found = true
				break
			}
		}
		if !found {
			storagePools = append(storagePools, isoPool)
		}
	}

	// Add VM disk pool
	diskPool, err := r.getDiskPool()
	if err == nil {
		storagePools = append(storagePools, diskPool)
	}

	return storagePools, nil
}

// getFilesystemPools returns filesystem-based pool info
func (r *ISORepository) getFilesystemPools() ([]*model.StoragePool, error) {
	isoPool, err := r.getISOPool()
	if err != nil {
		return nil, err
	}

	diskPool, err := r.getDiskPool()
	if err != nil {
		return []*model.StoragePool{isoPool}, nil
	}

	return []*model.StoragePool{isoPool, diskPool}, nil
}

// getISOPool returns info about the ISO directory pool
func (r *ISORepository) getISOPool() (*model.StoragePool, error) {
	isoDir := r.cfg.Storage.ISODir

	var capacity int64 = 500 * 1024 * 1024 * 1024 // 500 GB default
	var available int64 = capacity

	// Get actual disk usage
	var stat syscall.Statfs_t
	if err := syscall.Statfs(isoDir, &stat); err == nil {
		capacity = int64(stat.Blocks) * int64(stat.Bsize)
		available = int64(stat.Bavail) * int64(stat.Bsize)
	}

	// Calculate used space from ISOs
	var used int64
	isos, _ := r.scanDirectory()
	for _, iso := range isos {
		used += iso.Size
	}

	return &model.StoragePool{
		ID:             "local-iso",
		Name:           "Local ISO Storage",
		Type:           model.StorageTypeDir,
		Path:           isoDir,
		CapacityBytes:  capacity,
		AvailableBytes: available,
		UsedBytes:      used,
		Status:         model.StorageStatusActive,
	}, nil
}

// getDiskPool returns info about the VM disk directory pool
func (r *ISORepository) getDiskPool() (*model.StoragePool, error) {
	diskDir := r.cfg.Storage.VMDiskDir

	var capacity int64 = 500 * 1024 * 1024 * 1024 // 500 GB default
	var available int64 = capacity

	// Get actual disk usage
	var stat syscall.Statfs_t
	if err := syscall.Statfs(diskDir, &stat); err == nil {
		capacity = int64(stat.Blocks) * int64(stat.Bsize)
		available = int64(stat.Bavail) * int64(stat.Bsize)
	}

	// Calculate used space from disk files
	var used int64
	if entries, err := os.ReadDir(diskDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				if info, err := entry.Info(); err == nil {
					used += info.Size()
				}
			}
		}
	}

	return &model.StoragePool{
		ID:             "local-disks",
		Name:           "Local VM Disk Storage",
		Type:           model.StorageTypeDir,
		Path:           diskDir,
		CapacityBytes:  capacity,
		AvailableBytes: available,
		UsedBytes:      used,
		Status:         model.StorageStatusActive,
	}, nil
}

// poolToStoragePool converts a libvirt storage pool to our model
func (r *ISORepository) poolToStoragePool(pool *libvirt.StoragePool) (*model.StoragePool, error) {
	name, err := pool.GetName()
	if err != nil {
		return nil, err
	}

	// Get pool info
	info, err := pool.GetInfo()
	if err != nil {
		return nil, err
	}

	// Get pool XML to extract path
	xmlDesc, err := pool.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	// Parse XML to get path (simplified - just look for <path>)
	path := ""
	if idx := strings.Index(xmlDesc, "<path>"); idx != -1 {
		endIdx := strings.Index(xmlDesc[idx:], "</path>")
		if endIdx != -1 {
			path = xmlDesc[idx+6 : idx+endIdx]
		}
	}

	// Map pool state to status
	status := "unknown"
	switch info.State {
	case libvirt.STORAGE_POOL_RUNNING:
		status = "active"
	case libvirt.STORAGE_POOL_BUILDING:
		status = "building"
	case libvirt.STORAGE_POOL_INACCESSIBLE:
		status = "inactive"
	}

	return &model.StoragePool{
		ID:             name,
		Name:           name,
		Type:           model.StorageTypeDir,
		Path:           path,
		CapacityBytes:  int64(info.Capacity),
		AvailableBytes: int64(info.Available),
		UsedBytes:      int64(info.Allocation),
		Status:         model.StorageStatus(status),
	}, nil
}
