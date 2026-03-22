package libvirt

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// StorageBackend defines the interface for storage backend operations
type StorageBackend interface {
	// Initialize the storage pool
	Initialize() error
	// Get pool statistics
	GetStats() (capacity, used, available int64, err error)
	// Create a disk image
	CreateDisk(name string, sizeBytes int64, format model.DiskFormat, sparse bool) (path string, err error)
	// Delete a disk image
	DeleteDisk(path string) error
	// Resize a disk image
	ResizeDisk(path string, newSizeBytes int64) error
	// Move a disk image to another pool
	MoveDisk(sourcePath, targetPath string, targetFormat model.DiskFormat) error
}

// DirStorageBackend implements storage backend for directory-based storage
type DirStorageBackend struct {
	path string
}

// NewDirStorageBackend creates a new directory storage backend
func NewDirStorageBackend(path string) *DirStorageBackend {
	return &DirStorageBackend{path: path}
}

// Initialize creates the directory if it doesn't exist
func (b *DirStorageBackend) Initialize() error {
	return os.MkdirAll(b.path, 0755)
}

// GetStats returns disk space statistics for the directory
func (b *DirStorageBackend) GetStats() (capacity, used, available int64, err error) {
	cmd := exec.Command("df", "-B1", b.path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get disk stats: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0, 0, 0, fmt.Errorf("unexpected df output")
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 6 {
		return 0, 0, 0, fmt.Errorf("unexpected df output format")
	}

	capacity, err = strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse capacity: %w", err)
	}

	used, err = strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse used: %w", err)
	}

	available, err = strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse available: %w", err)
	}

	return capacity, used, available, nil
}

// CreateDisk creates a disk image file
func (b *DirStorageBackend) CreateDisk(name string, sizeBytes int64, format model.DiskFormat, sparse bool) (string, error) {
	path := filepath.Join(b.path, name)

	// Determine qemu-img format
	qemuFormat := string(format)
	if qemuFormat == "" {
		qemuFormat = "qcow2"
	}

	// Build qemu-img command
	args := []string{"create", "-f", qemuFormat}
	if sparse && qemuFormat == "qcow2" {
		args = append(args, "-o", "cluster_size=64k,preallocation=off")
	}
	args = append(args, path, fmt.Sprintf("%d", sizeBytes))

	cmd := exec.Command("qemu-img", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create disk: %w, output: %s", err, string(output))
	}

	return path, nil
}

// DeleteDisk deletes a disk image file
func (b *DirStorageBackend) DeleteDisk(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete disk: %w", err)
	}
	return nil
}

// ResizeDisk resizes a disk image
func (b *DirStorageBackend) ResizeDisk(path string, newSizeBytes int64) error {
	cmd := exec.Command("qemu-img", "resize", path, fmt.Sprintf("%d", newSizeBytes))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to resize disk: %w, output: %s", err, string(output))
	}
	return nil
}

// MoveDisk moves a disk image to another location
func (b *DirStorageBackend) MoveDisk(sourcePath, targetPath string, targetFormat model.DiskFormat) error {
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Use qemu-img convert for format conversion if needed
	if targetFormat != "" {
		cmd := exec.Command("qemu-img", "convert", "-f", "auto", "-O", string(targetFormat), sourcePath, targetPath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to convert disk: %w, output: %s", err, string(output))
		}
		// Remove source after successful conversion
		os.Remove(sourcePath)
	} else {
		// Simple move
		if err := os.Rename(sourcePath, targetPath); err != nil {
			return fmt.Errorf("failed to move disk: %w", err)
		}
	}

	return nil
}

// LVMStorageBackend implements storage backend for LVM
type LVMStorageBackend struct {
	vgName string
}

// NewLVMStorageBackend creates a new LVM storage backend
func NewLVMStorageBackend(vgName string) *LVMStorageBackend {
	return &LVMStorageBackend{vgName: vgName}
}

// Initialize verifies the volume group exists
func (b *LVMStorageBackend) Initialize() error {
	cmd := exec.Command("vgs", b.vgName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("volume group %s not found: %w", b.vgName, err)
	}
	return nil
}

// GetStats returns LVM volume group statistics
func (b *LVMStorageBackend) GetStats() (capacity, used, available int64, err error) {
	cmd := exec.Command("vgs", "--noheadings", "--units", "b", "-o", "vg_size,vg_free", b.vgName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get VG stats: %w", err)
	}

	fields := strings.Fields(strings.TrimSpace(string(output)))
	if len(fields) < 2 {
		return 0, 0, 0, fmt.Errorf("unexpected vgs output")
	}

	// Parse size (remove 'B' suffix)
	capacity, err = strconv.ParseInt(strings.TrimSuffix(fields[0], "B"), 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse capacity: %w", err)
	}

	available, err = strconv.ParseInt(strings.TrimSuffix(fields[1], "B"), 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse available: %w", err)
	}

	used = capacity - available
	return capacity, used, available, nil
}

// CreateDisk creates an LVM logical volume
func (b *LVMStorageBackend) CreateDisk(name string, sizeBytes int64, format model.DiskFormat, sparse bool) (string, error) {
	lvName := fmt.Sprintf("vm-%s", name)
	path := fmt.Sprintf("/dev/%s/%s", b.vgName, lvName)

	// Create thin volume if sparse, otherwise thick
	args := []string{"-L", fmt.Sprintf("%db", sizeBytes), "-n", lvName}
	if sparse {
		args = append([]string{"-T"}, args...)
	}
	args = append(args, b.vgName)

	cmd := exec.Command("lvcreate", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create LV: %w, output: %s", err, string(output))
	}

	return path, nil
}

// DeleteDisk deletes an LVM logical volume
func (b *LVMStorageBackend) DeleteDisk(path string) error {
	cmd := exec.Command("lvremove", "-f", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete LV: %w, output: %s", err, string(output))
	}
	return nil
}

// ResizeDisk resizes an LVM logical volume
func (b *LVMStorageBackend) ResizeDisk(path string, newSizeBytes int64) error {
	cmd := exec.Command("lvresize", "-f", "-L", fmt.Sprintf("%db", newSizeBytes), path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to resize LV: %w, output: %s", err, string(output))
	}
	return nil
}

// MoveDisk for LVM uses dd or lvconvert
func (b *LVMStorageBackend) MoveDisk(sourcePath, targetPath string, targetFormat model.DiskFormat) error {
	// For LVM, we typically use dd to copy
	cmd := exec.Command("dd", "if="+sourcePath, "of="+targetPath, "bs=4M", "status=progress")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy LV: %w", err)
	}
	return nil
}

// ZFSStorageBackend implements storage backend for ZFS
type ZFSStorageBackend struct {
	poolName string
	dataset  string
}

// NewZFSStorageBackend creates a new ZFS storage backend
func NewZFSStorageBackend(poolName, dataset string) *ZFSStorageBackend {
	return &ZFSStorageBackend{
		poolName: poolName,
		dataset:  dataset,
	}
}

// Initialize verifies the ZFS dataset exists
func (b *ZFSStorageBackend) Initialize() error {
	cmd := exec.Command("zfs", "list", b.dataset)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ZFS dataset %s not found: %w", b.dataset, err)
	}
	return nil
}

// GetStats returns ZFS dataset statistics
func (b *ZFSStorageBackend) GetStats() (capacity, used, available int64, err error) {
	cmd := exec.Command("zfs", "get", "-Hpo", "value", "size,used,available", b.dataset)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get ZFS stats: %w", err)
	}

	values := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(values) < 3 {
		return 0, 0, 0, fmt.Errorf("unexpected zfs output")
	}

	capacity, err = strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse capacity: %w", err)
	}

	used, err = strconv.ParseInt(values[1], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse used: %w", err)
	}

	available, err = strconv.ParseInt(values[2], 10, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse available: %w", err)
	}

	return capacity, used, available, nil
}

// CreateDisk creates a ZFS volume
func (b *ZFSStorageBackend) CreateDisk(name string, sizeBytes int64, format model.DiskFormat, sparse bool) (string, error) {
	volName := fmt.Sprintf("%s/vm-%s", b.dataset, name)
	path := fmt.Sprintf("/dev/zvol/%s", volName)

	cmd := exec.Command("zfs", "create", "-V", fmt.Sprintf("%db", sizeBytes), volName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create ZFS volume: %w, output: %s", err, string(output))
	}

	return path, nil
}

// DeleteDisk deletes a ZFS volume
func (b *ZFSStorageBackend) DeleteDisk(path string) error {
	// Extract volume name from path
	volName := strings.TrimPrefix(path, "/dev/zvol/")
	cmd := exec.Command("zfs", "destroy", volName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete ZFS volume: %w, output: %s", err, string(output))
	}
	return nil
}

// ResizeDisk resizes a ZFS volume
func (b *ZFSStorageBackend) ResizeDisk(path string, newSizeBytes int64) error {
	volName := strings.TrimPrefix(path, "/dev/zvol/")
	cmd := exec.Command("zfs", "set", fmt.Sprintf("volsize=%db", newSizeBytes), volName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to resize ZFS volume: %w, output: %s", err, string(output))
	}
	return nil
}

// MoveDisk for ZFS uses zfs send/recv or dd
func (b *ZFSStorageBackend) MoveDisk(sourcePath, targetPath string, targetFormat model.DiskFormat) error {
	cmd := exec.Command("dd", "if="+sourcePath, "of="+targetPath, "bs=4M", "status=progress")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy ZFS volume: %w", err)
	}
	return nil
}

// GetStorageBackend returns the appropriate backend for the storage type
func GetStorageBackend(poolType model.StorageType, path string, options map[string]string) (StorageBackend, error) {
	switch poolType {
	case model.StorageTypeDir:
		return NewDirStorageBackend(path), nil
	case model.StorageTypeLVM:
		vgName := options["vg_name"]
		if vgName == "" {
			return nil, fmt.Errorf("vg_name option required for LVM storage")
		}
		return NewLVMStorageBackend(vgName), nil
	case model.StorageTypeZFS:
		poolName := options["pool_name"]
		dataset := options["dataset"]
		if dataset == "" {
			dataset = poolName
		}
		if poolName == "" {
			return nil, fmt.Errorf("pool_name option required for ZFS storage")
		}
		return NewZFSStorageBackend(poolName, dataset), nil
	default:
		// Default to directory backend for unknown types
		return NewDirStorageBackend(path), nil
	}
}

// RefreshPoolStats updates the statistics for a storage pool
func RefreshPoolStats(client libvirtx.LibvirtClient, pool *model.StoragePool) (capacity, used, available int64, err error) {
	// Try to get stats from libvirt storage pools first
	// Fall back to backend-specific methods

	backend, err := GetStorageBackend(pool.Type, pool.Path, pool.Options)
	if err != nil {
		return 0, 0, 0, err
	}

	return backend.GetStats()
}
