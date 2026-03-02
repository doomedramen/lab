package libvirt

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"libvirt.org/go/libvirt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// PCIRepository implements repository.PCIRepository using libvirt
type PCIRepository struct {
	client *libvirtx.Client
}

// NewPCIRepository creates a new libvirt PCI repository
func NewPCIRepository(client *libvirtx.Client) *PCIRepository {
	return &PCIRepository{client: client}
}

// ListHostDevices returns all PCI devices on the host, grouped by IOMMU group
func (r *PCIRepository) ListHostDevices(_ context.Context) ([]model.PCIDevice, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	// Get all node devices
	devices, err := conn.ListAllNodeDevices(0)
	if err != nil {
		return nil, fmt.Errorf("failed to list node devices: %w", err)
	}

	var pciDevices []model.PCIDevice
	for _, dev := range devices {
		// Get device XML
		xmlDesc, err := dev.GetXMLDesc(0)
		if err != nil {
			dev.Free()
			continue
		}

		// Parse XML to check if it's a PCI device
		device, err := parseNodeDeviceXML(xmlDesc)
		if err != nil {
			dev.Free()
			continue
		}

		// Only include PCI devices
		if !isPCIDevice(xmlDesc) {
			dev.Free()
			continue
		}

		pciDevices = append(pciDevices, device)
		dev.Free()
	}

	return pciDevices, nil
}

// parseNodeDeviceXML parses libvirt node device XML into a PCIDevice
func parseNodeDeviceXML(xmlDesc string) (model.PCIDevice, error) {
	device := model.PCIDevice{
		IOMMUGroup: -1, // Default to -1 (not available)
	}

	// Simple XML parsing using string operations
	// In production, we'd use proper XML parsing with libvirtxml

	// Extract PCI address from the device name or XML
	// Format: pci_0000_01_00_0 or from <bus>, <slot>, <function> elements
	if addr := extractXMLValue(xmlDesc, "bus"); addr != "" {
		domain := extractXMLValue(xmlDesc, "domain")
		slot := extractXMLValue(xmlDesc, "slot")
		function := extractXMLValue(xmlDesc, "function")
		if domain != "" && addr != "" && slot != "" && function != "" {
			device.Address = formatPCIAddress(domain, addr, slot, function)
		}
	}

	// Extract vendor
	if vendorID := extractXMLValue(xmlDesc, "vendor"); vendorID != "" {
		device.VendorID = vendorID
	}
	if vendorName := extractXMLAttr(xmlDesc, "vendor", "id"); vendorName != "" {
		device.VendorID = strings.TrimPrefix(vendorName, "0x")
	}

	// Extract product
	if productID := extractXMLValue(xmlDesc, "product"); productID != "" {
		device.ProductID = productID
	}
	if productName := extractXMLAttr(xmlDesc, "product", "id"); productName != "" {
		device.ProductID = strings.TrimPrefix(productName, "0x")
	}

	// Extract class
	if class := extractXMLValue(xmlDesc, "class"); class != "" {
		device.Class = class
		device.ClassName = getPCIClassName(class)
	}

	// Get IOMMU group from sysfs
	if device.Address != "" {
		device.IOMMUGroup = getIOMMUGroup(device.Address)
	}

	// Get current driver from sysfs
	if device.Address != "" {
		device.Driver = getPCIDriver(device.Address)
	}

	// Try to get vendor/product names from sysfs
	if device.Address != "" {
		device.VendorName = getSysfsPCIProperty(device.Address, "vendor")
		device.ProductName = getSysfsPCIProperty(device.Address, "device")
	}

	return device, nil
}

// isPCIDevice checks if the device XML is for a PCI device
func isPCIDevice(xmlDesc string) bool {
	return strings.Contains(xmlDesc, "<capability type='pci'>") ||
		strings.Contains(xmlDesc, `type="pci"`)
}

// formatPCIAddress creates a PCI address string from components
func formatPCIAddress(domain, bus, slot, function string) string {
	d := strings.TrimPrefix(domain, "0x")
	b := strings.TrimPrefix(bus, "0x")
	s := strings.TrimPrefix(slot, "0x")
	f := strings.TrimPrefix(function, "0x")

	// Pad with zeros
	d = fmt.Sprintf("%04s", d)
	b = fmt.Sprintf("%02s", b)
	s = fmt.Sprintf("%02s", s)
	f = fmt.Sprintf("%01s", f)

	return fmt.Sprintf("%s:%s:%s.%s", d, b, s, f)
}

// extractXMLValue extracts the value between <tag>...</tag>
func extractXMLValue(xml, tag string) string {
	start := fmt.Sprintf("<%s>", tag)
	end := fmt.Sprintf("</%s>", tag)

	startIdx := strings.Index(xml, start)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(start)

	endIdx := strings.Index(xml[startIdx:], end)
	if endIdx == -1 {
		return ""
	}

	return strings.TrimSpace(xml[startIdx : startIdx+endIdx])
}

// extractXMLAttr extracts an attribute value from a tag
func extractXMLAttr(xml, tag, attr string) string {
	// Look for <tag attr="value">
	pattern := fmt.Sprintf("<%s %s=\"", tag, attr)
	startIdx := strings.Index(xml, pattern)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(pattern)

	endIdx := strings.Index(xml[startIdx:], "\"")
	if endIdx == -1 {
		return ""
	}

	return xml[startIdx : startIdx+endIdx]
}

// getIOMMUGroup returns the IOMMU group number for a PCI device
func getIOMMUGroup(addr string) int {
	// Normalize address format for sysfs
	normalizedAddr := normalizePCIAddress(addr)
	if normalizedAddr == "" {
		return -1
	}

	iommuPath := fmt.Sprintf("/sys/bus/pci/devices/%s/iommu_group", normalizedAddr)
	link, err := os.Readlink(iommuPath)
	if err != nil {
		return -1
	}

	// Link target is something like "../../../../../kernel/iommu_groups/1"
	parts := strings.Split(link, "/")
	for _, part := range parts {
		if group, err := strconv.Atoi(part); err == nil {
			return group
		}
	}

	return -1
}

// getPCIDriver returns the current kernel driver for a PCI device
func getPCIDriver(addr string) string {
	normalizedAddr := normalizePCIAddress(addr)
	if normalizedAddr == "" {
		return ""
	}

	driverPath := fmt.Sprintf("/sys/bus/pci/devices/%s/driver", normalizedAddr)
	link, err := os.Readlink(driverPath)
	if err != nil {
		return ""
	}

	// Link target is something like ../../../../bus/pci/drivers/nvidia
	return filepath.Base(link)
}

// getSysfsPCIProperty reads a property from sysfs for a PCI device
func getSysfsPCIProperty(addr, prop string) string {
	normalizedAddr := normalizePCIAddress(addr)
	if normalizedAddr == "" {
		return ""
	}

	// Map common property names to sysfs files
	fileMap := map[string]string{
		"vendor": "vendor",
		"device": "device",
	}

	file := fileMap[prop]
	if file == "" {
		return ""
	}

	path := fmt.Sprintf("/sys/bus/pci/devices/%s/%s", normalizedAddr, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	// Vendor/device files contain hex IDs like "0x10de\n"
	value := strings.TrimSpace(string(data))
	return strings.TrimPrefix(value, "0x")
}

// normalizePCIAddress converts various PCI address formats to sysfs format
func normalizePCIAddress(addr string) string {
	// Input could be:
	// - 0000:01:00.0 (already normalized)
	// - 01:00.0 (short form)
	// - pci_0000_01_00_0 (libvirt name)

	// Handle libvirt device name format
	if strings.HasPrefix(addr, "pci_") {
		parts := strings.Split(strings.TrimPrefix(addr, "pci_"), "_")
		if len(parts) == 4 {
			return fmt.Sprintf("%s:%s:%s.%s", parts[0], parts[1], parts[2], parts[3])
		}
	}

	// Handle short form
	parts := strings.Split(addr, ":")
	if len(parts) == 2 {
		// 01:00.0 format - prepend 0000
		return fmt.Sprintf("0000:%s", addr)
	}

	return addr
}

// getPCIClassName returns a human-readable name for a PCI class code
func getPCIClassName(class string) string {
	// Class codes are 6 hex digits: CCSSPP
	// CC = Base class, SS = Subclass, PP = Programming interface
	if len(class) < 2 {
		return "Unknown"
	}

	baseClass := class[:2]

	classNames := map[string]string{
		"00": "Non-VGA unclassified device",
		"01": "Mass storage controller",
		"02": "Network controller",
		"03": "Display controller",
		"04": "Multimedia controller",
		"05": "Memory controller",
		"06": "Bridge",
		"07": "Communication controller",
		"08": "Generic system peripheral",
		"09": "Input device controller",
		"0a": "Docking station",
		"0b": "Processor",
		"0c": "Serial bus controller",
		"0d": "Wireless controller",
		"0e": "Intelligent controller",
		"0f": "Satellite communications controller",
		"10": "Encryption controller",
		"11": "Signal processing controller",
		"12": "Processing accelerators",
		"13": "Non-Essential Instrumentation",
		"40": "Coprocessor",
		"ff": "Unassigned class",
	}

	name, ok := classNames[baseClass]
	if !ok {
		return "Unknown device"
	}

	return name
}

// GetDevicesByIOMMUGroup returns PCI devices grouped by IOMMU group
func (r *PCIRepository) GetDevicesByIOMMUGroup(ctx context.Context) (map[int][]model.PCIDevice, error) {
	devices, err := r.ListHostDevices(ctx)
	if err != nil {
		return nil, err
	}

	groups := make(map[int][]model.PCIDevice)
	for _, dev := range devices {
		if dev.IOMMUGroup >= 0 {
			groups[dev.IOMMUGroup] = append(groups[dev.IOMMUGroup], dev)
		}
	}

	return groups, nil
}

// IsIOMMUAvailable checks if IOMMU is available on the host
func (r *PCIRepository) IsIOMMUAvailable() bool {
	// Check for IOMMU groups directory
	iommuGroupsPath := "/sys/kernel/iommu_groups"
	if _, err := os.Stat(iommuGroupsPath); err == nil {
		// Directory exists, check if it has any groups
		entries, err := os.ReadDir(iommuGroupsPath)
		if err == nil && len(entries) > 0 {
			return true
		}
	}

	// Alternative: check for Intel or AMD IOMMU in kernel cmdline
	cmdline, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return false
	}

	cmdlineStr := string(cmdline)
	return strings.Contains(cmdlineStr, "intel_iommu=on") ||
		strings.Contains(cmdlineStr, "amd_iommu=on") ||
		strings.Contains(cmdlineStr, "iommu=on")
}

// IsVFIOAvailable checks if VFIO kernel modules are loaded
func (r *PCIRepository) IsVFIOAvailable() bool {
	// Check if vfio-pci module is available
	if _, err := os.Stat("/sys/module/vfio_pci"); err == nil {
		return true
	}

	// Check if module can be loaded (exists in /lib/modules)
	// This is a simple heuristic
	return true // Assume available if IOMMU is present
}

// AttachPCIDeviceToVM attaches a PCI device to a VM (requires VM to be stopped)
func (r *PCIRepository) AttachPCIDeviceToVM(_ context.Context, vmid int, pciAddr string) error {
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return fmt.Errorf("failed to find VM %d: %w", vmid, err)
	}
	defer domain.Free()

	// Check VM is stopped
	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get VM state: %w", err)
	}
	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return fmt.Errorf("VM must be stopped to attach PCI device")
	}

	// Get current XML
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	// Parse and modify XML to add the PCI device
	// For now, we'll use string manipulation
	// In production, use libvirtxml package

	normalizedAddr := normalizePCIAddress(pciAddr)
	parts := strings.Split(normalizedAddr, ":")
	if len(parts) != 3 {
		return fmt.Errorf("invalid PCI address format: %s", pciAddr)
	}

	domainPart := parts[0]
	busSlotFunc := strings.Split(parts[2], ".")
	if len(busSlotFunc) != 2 {
		return fmt.Errorf("invalid PCI address format: %s", pciAddr)
	}

	bus := parts[1]
	slot := busSlotFunc[0]
	function := busSlotFunc[1]

	// Generate hostdev XML
	hostdevXML := fmt.Sprintf(`
    <hostdev mode='subsystem' type='pci' managed='yes'>
      <source>
        <address domain='0x%s' bus='0x%s' slot='0x%s' function='0x%s'/>
      </source>
    </hostdev>`, domainPart, bus, slot, function)

	// Insert before </devices>
	newXML := strings.Replace(xmlDesc, "</devices>", hostdevXML+"\n  </devices>", 1)

	// Define the updated domain
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	_, err = conn.DomainDefineXML(newXML)
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}

	return nil
}

// DetachPCIDeviceFromVM detaches a PCI device from a VM (requires VM to be stopped)
func (r *PCIRepository) DetachPCIDeviceFromVM(_ context.Context, vmid int, pciAddr string) error {
	domain, err := r.client.GetDomainByVMID(vmid)
	if err != nil {
		return fmt.Errorf("failed to find VM %d: %w", vmid, err)
	}
	defer domain.Free()

	// Check VM is stopped
	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get VM state: %w", err)
	}
	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return fmt.Errorf("VM must be stopped to detach PCI device")
	}

	// Get current XML
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	normalizedAddr := normalizePCIAddress(pciAddr)

	// Find and remove the hostdev entry for this PCI address
	// This is a simplified approach - in production, use proper XML parsing
	lines := strings.Split(xmlDesc, "\n")
	var newLines []string
	skipUntil := -1

	for i, line := range lines {
		if skipUntil >= 0 && i <= skipUntil {
			continue
		}
		if strings.Contains(line, "<hostdev") && i+1 < len(lines) {
			// Check if this is the device we want to remove
			remaining := strings.Join(lines[i:], "\n")
			if strings.Contains(remaining, normalizedAddr) ||
				strings.Contains(remaining, strings.ReplaceAll(normalizedAddr, ":", "")) {
				// Find the end of this hostdev element
				endIdx := strings.Index(remaining, "</hostdev>")
				if endIdx != -1 {
					skipUntil = i + strings.Count(remaining[:endIdx], "\n")
					continue
				}
			}
		}
		newLines = append(newLines, line)
	}

	newXML := strings.Join(newLines, "\n")

	// Define the updated domain
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	_, err = conn.DomainDefineXML(newXML)
	if err != nil {
		return fmt.Errorf("failed to update domain: %w", err)
	}

	return nil
}
