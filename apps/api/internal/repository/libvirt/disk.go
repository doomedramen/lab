package libvirt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	libvirt "libvirt.org/go/libvirt"
	libvirtxml "libvirt.org/go/libvirtxml"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// DiskRepository handles libvirt disk operations for VMs
type DiskRepository struct {
	client libvirtx.LibvirtClient
}

// NewDiskRepository creates a new disk repository
func NewDiskRepository(client libvirtx.LibvirtClient) *DiskRepository {
	return &DiskRepository{
		client: client,
	}
}

// AttachDisk attaches a disk to a VM using virDomainAttachDeviceFlags
func (r *DiskRepository) AttachDisk(ctx context.Context, vmid int, diskPath string, bus model.DiskBus, readonly bool) (string, error) {
	// Find the domain by VMID
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return "", fmt.Errorf("failed to get domain: %w", err)
	}
	defer domain.Free()

	// Determine target device name based on bus
	targetDev := r.getNextTargetDevice(domain, bus)

	// Build disk XML
	diskXML := r.buildDiskXML(diskPath, targetDev, bus, readonly)

	// Attach the disk
	// Use DOMAIN_DEVICE_MODIFY_CURRENT for live attach, DOMAIN_DEVICE_MODIFY_CONFIG for persistent
	flags := libvirt.DOMAIN_DEVICE_MODIFY_CURRENT
	if err := domain.AttachDeviceFlags(diskXML, flags); err != nil {
		return "", fmt.Errorf("failed to attach disk: %w", err)
	}

	return targetDev, nil
}

// DetachDisk detaches a disk from a VM using virDomainDetachDeviceFlags
func (r *DiskRepository) DetachDisk(ctx context.Context, vmid int, target string) error {
	// Find the domain by VMID
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}
	defer domain.Free()

	// Get domain XML to find the disk path
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	domainXML := &libvirtxml.Domain{}
	if err := domainXML.Unmarshal(xmlDesc); err != nil {
		return fmt.Errorf("failed to parse domain XML: %w", err)
	}

	// Find the disk by target
	var diskPath string
	var diskBus model.DiskBus
	if domainXML.Devices != nil {
		for _, disk := range domainXML.Devices.Disks {
			if disk.Target != nil && disk.Target.Dev == target {
				if disk.Source != nil && disk.Source.File != nil {
					diskPath = disk.Source.File.File
				}
				if disk.Target.Bus != "" {
					diskBus = model.DiskBus(disk.Target.Bus)
				}
				break
			}
		}
	}

	if diskPath == "" {
		return fmt.Errorf("disk with target %s not found", target)
	}

	// Build disk XML for detach (needs to match the attach XML)
	diskXML := r.buildDiskXML(diskPath, target, diskBus, false)

	// Detach the disk
	flags := libvirt.DOMAIN_DEVICE_MODIFY_CURRENT
	if err := domain.DetachDeviceFlags(diskXML, flags); err != nil {
		return fmt.Errorf("failed to detach disk: %w", err)
	}

	return nil
}

// ListVMDisks returns all disks attached to a VM
func (r *DiskRepository) ListVMDisks(ctx context.Context, vmid int) ([]model.VMDisk, error) {
	// Find the domain by VMID
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	defer domain.Free()

	// Get domain XML
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %w", err)
	}

	domainXML := &libvirtxml.Domain{}
	if err := domainXML.Unmarshal(xmlDesc); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	var disks []model.VMDisk
	if domainXML.Devices != nil {
		for i, disk := range domainXML.Devices.Disks {
			// Only include disk devices (not cdrom)
			if disk.Device != "disk" {
				continue
			}

			vmDisk := model.VMDisk{
				ID:     fmt.Sprintf("disk-%d", i),
				VMID:   vmid,
				Target: "",
				Bus:    model.DiskBusVirtIO,
				Format: model.DiskFormatQCOW2,
			}

			if disk.Target != nil {
				vmDisk.Target = disk.Target.Dev
			}

			if disk.Source != nil && disk.Source.File != nil {
				vmDisk.Path = disk.Source.File.File
				// Get disk size
				if info, err := os.Stat(vmDisk.Path); err == nil {
					vmDisk.SizeBytes = info.Size()
				}
			}

			if disk.Target != nil && disk.Target.Bus != "" {
				vmDisk.Bus = model.DiskBus(disk.Target.Bus)
			}

			if disk.Driver != nil && disk.Driver.Type != "" {
				vmDisk.Format = model.DiskFormat(disk.Driver.Type)
			}

			if disk.ReadOnly != nil {
				vmDisk.Readonly = true
			}

			disks = append(disks, vmDisk)
		}
	}

	return disks, nil
}

// CreateDiskImage creates a new disk image using qemu-img
func (r *DiskRepository) CreateDiskImage(path string, sizeGB float64, format model.DiskFormat, sparse bool) error {
	// Ensure directory exists
	dir := strings.TrimSuffix(path, "/"+strings.Split(path, "/")[len(strings.Split(path, "/"))-1])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create disk directory: %w", err)
	}

	// Build qemu-img command
	args := []string{"create", "-f", string(format)}
	if sparse && format == model.DiskFormatQCOW2 {
		args = append(args, "-o", "preallocation=metadata")
	}
	args = append(args, path, fmt.Sprintf("%.1fG", sizeGB))

	cmd := exec.Command("qemu-img", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu-img failed: %v, output: %s", err, string(out))
	}

	return nil
}

// ResizeDiskImage resizes a disk image using qemu-img
func (r *DiskRepository) ResizeDiskImage(path string, newSizeGB float64) error {
	cmd := exec.Command("qemu-img", "resize", path, fmt.Sprintf("%.1fG", newSizeGB))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu-img resize failed: %v, output: %s", err, string(out))
	}
	return nil
}

// buildDiskXML builds the XML for a disk device
func (r *DiskRepository) buildDiskXML(diskPath, targetDev string, bus model.DiskBus, readonly bool) string {
	readonlyXML := ""
	if readonly {
		readonlyXML = "<readonly/>"
	}

	return fmt.Sprintf(`<disk type='file' device='disk'>
    <driver name='qemu' type='%s'/>
    <source file='%s'/>
    <target dev='%s' bus='%s'/>
    %s
  </disk>`, model.DiskFormatQCOW2, diskPath, targetDev, string(bus), readonlyXML)
}

// getNextTargetDevice finds the next available target device name
func (r *DiskRepository) getNextTargetDevice(domain libvirtx.Domain, bus model.DiskBus) string {
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		// Default to vda if we can't read the XML
		return "vda"
	}

	domainXML := &libvirtxml.Domain{}
	if err := domainXML.Unmarshal(xmlDesc); err != nil {
		return "vda"
	}

	// Collect existing targets
	existingTargets := make(map[string]bool)
	if domainXML.Devices != nil {
		for _, disk := range domainXML.Devices.Disks {
			if disk.Target != nil {
				existingTargets[disk.Target.Dev] = true
			}
		}
	}

	// Generate target based on bus type
	prefix := "vd" // virtio
	if bus == model.DiskBusSATA || bus == model.DiskBusIDE {
		prefix = "sd" // SATA/IDE
	}

	// Find next available letter
	for i := 0; i < 26; i++ {
		target := fmt.Sprintf("%s%c", prefix, 'a'+i)
		if !existingTargets[target] {
			return target
		}
	}

	// If we run out of letters, use aa, ab, etc.
	for i := 0; i < 26; i++ {
		for j := 0; j < 26; j++ {
			target := fmt.Sprintf("%s%c%c", prefix, 'a'+i, 'a'+j)
			if !existingTargets[target] {
				return target
			}
		}
	}

	return "vda" // Fallback
}

// GetDiskInfo returns information about a disk image
func (r *DiskRepository) GetDiskInfo(path string) (sizeBytes int64, format string, err error) {
	// Get file size
	info, err := os.Stat(path)
	if err != nil {
		return 0, "", fmt.Errorf("failed to stat disk: %w", err)
	}
	sizeBytes = info.Size()

	// Get format using qemu-img info
	cmd := exec.Command("qemu-img", "info", "--output=json", path)
	output, err := cmd.Output()
	if err != nil {
		return sizeBytes, "", nil // Return size even if we can't get format
	}

	// Parse JSON output to get format
	outputStr := string(output)
	if strings.Contains(outputStr, `"format": "qcow2"`) {
		format = "qcow2"
	} else if strings.Contains(outputStr, `"format": "raw"`) {
		format = "raw"
	} else {
		format = "unknown"
	}

	return sizeBytes, format, nil
}

// IsRootDisk checks if a disk is the root disk (boot disk)
func (r *DiskRepository) IsRootDisk(ctx context.Context, vmid int, target string) (bool, error) {
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return false, fmt.Errorf("failed to get domain: %w", err)
	}
	defer domain.Free()

	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return false, fmt.Errorf("failed to get domain XML: %w", err)
	}

	domainXML := &libvirtxml.Domain{}
	if err := domainXML.Unmarshal(xmlDesc); err != nil {
		return false, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	// Check if this is the first disk (usually the root disk)
	if domainXML.Devices != nil {
		for i, disk := range domainXML.Devices.Disks {
			if disk.Device != "disk" {
				continue
			}
			if disk.Target != nil && disk.Target.Dev == target {
				return i == 0, nil // First disk is typically the root disk
			}
		}
	}

	return false, nil
}

// parseSizeString parses a size string from qemu-img (e.g., "10G", "512M")
func parseSizeString(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if len(sizeStr) < 2 {
		return 0, fmt.Errorf("invalid size string: %s", sizeStr)
	}

	unit := sizeStr[len(sizeStr)-1]
	numStr := sizeStr[:len(sizeStr)-1]

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'G', 'g':
		return int64(num * 1024 * 1024 * 1024), nil
	case 'M', 'm':
		return int64(num * 1024 * 1024), nil
	case 'K', 'k':
		return int64(num * 1024), nil
	case 'T', 't':
		return int64(num * 1024 * 1024 * 1024 * 1024), nil
	default:
		return 0, fmt.Errorf("unknown size unit: %c", unit)
	}
}
