package libvirt

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	"github.com/doomedramen/lab/apps/api/pkg/osinfo"
)

// VMRepository implements repository.VMRepository using libvirt
type VMRepository struct {
	client *libvirtx.Client
	mu     sync.RWMutex
	cfg    *config.Config
}

// NewVMRepository creates a new libvirt VM repository
func NewVMRepository(client *libvirtx.Client, cfg *config.Config) *VMRepository {
	return &VMRepository{
		client: client,
		cfg:    cfg,
	}
}

// GetAll returns all VMs from libvirt
func (r *VMRepository) GetAll(_ context.Context) ([]*model.VM, error) {
	domains, err := r.client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	var vms []*model.VM
	for _, domain := range domains {
		vm := r.domainToVM(domain)
		if vm != nil {
			vms = append(vms, vm)
		}
		domain.Free()
	}

	return vms, nil
}

// GetByNode returns all VMs on a specific node
// For local libvirt, all VMs are on the same node
func (r *VMRepository) GetByNode(ctx context.Context, node string) ([]*model.VM, error) {
	hostname, err := r.client.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// If node matches hostname (or is empty), return all VMs
	if node == "" || node == hostname {
		return r.GetAll(ctx)
	}

	return []*model.VM{}, nil
}

// GetByID returns a VM by its string ID
func (r *VMRepository) GetByID(_ context.Context, id string) (*model.VM, error) {
	domain, err := r.client.GetDomainByName(id)
	if err != nil {
		return nil, fmt.Errorf("domain %q not found: %w", id, err)
	}
	defer domain.Free()

	vm := r.domainToVM(domain)
	if vm == nil {
		return nil, fmt.Errorf("failed to convert domain %q to VM", id)
	}
	return vm, nil
}

// GetByVMID returns a VM by its numeric VMID
// For libvirt, we use the domain ID or parse from name
func (r *VMRepository) GetByVMID(_ context.Context, vmid int) (*model.VM, error) {
	// Try to find domain with name matching vmid
	domains, err := r.client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	for _, domain := range domains {
		vm := r.domainToVM(domain)
		domain.Free()

		if vm != nil && vm.VMID == vmid {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("VM with VMID %d not found", vmid)
}

// Create creates a new VM
func (r *VMRepository) Create(_ context.Context, req *model.VMCreateRequest) (*model.VM, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	// Generate a unique VM ID
	vmid := r.generateVMID()

	// Use vm-<ID> as the unique libvirt domain name
	domainName := fmt.Sprintf("vm-%d", vmid)

	// Convert memory from GB to KiB
	memoryKiB := req.Memory * 1024 * 1024

	// Create the disk image - use domainName for the filename
	diskPath := fmt.Sprintf("%s/%s.%s", r.cfg.Storage.VMDiskDir, domainName, r.cfg.VM.DiskFormat)
	if err := r.createDisk(diskPath, int(req.Disk)); err != nil {
		return nil, fmt.Errorf("failed to create disk image: %w", err)
	}

	// Build domain XML
	domainXML := r.buildDomainXML(req, vmid, memoryKiB)

	// Define the domain
	domain, err := conn.DomainDefineXML(domainXML)
	if err != nil {
		return nil, fmt.Errorf("failed to define domain: %w", err)
	}
	defer domain.Free()

	// Return the created VM
	return &model.VM{
		ID:          domainName,
		VMID:        vmid,
		Name:        req.Name, // User provided name
		Node:        r.getNodeName(),
		Status:      model.VMStatusStopped,
		CPU:         model.CPUInfoPartial{Used: 0, Sockets: req.CPUSockets, Cores: req.CPUCores},
		Memory:      model.MemoryInfo{Used: 0, Total: req.Memory},
		Disk:        model.DiskInfo{Used: 0, Total: req.Disk},
		Uptime:      "0d 0h",
		OS:          req.OS,
		Arch:        req.Arch,
		MachineType: req.MachineType,
		BIOS:        req.BIOS,
		CPUModel:    req.CPUModel,
		Network:     req.Network,
		IP:          "",
		Tags:        req.Tags,
		HA:          false,
		Description: req.Description,
		NestedVirt:  req.NestedVirt,
		StartOnBoot: req.StartOnBoot,
		Agent:       req.Agent,
	}, nil
}

// generateVMID generates a unique VM ID, starting from 100
func (r *VMRepository) generateVMID() int {
	domains, err := r.client.ListDomains()
	if err != nil {
		return 100
	}

	usedIDs := make(map[int]bool)
	for _, domain := range domains {
		name, err := domain.GetName()
		if err == nil {
			usedIDs[r.extractVMID(name)] = true
		}
		domain.Free()
	}

	for id := 100; ; id++ {
		if !usedIDs[id] {
			return id
		}
	}
}

// buildDomainXML builds the libvirt domain XML for a new VM
func (r *VMRepository) buildDomainXML(req *model.VMCreateRequest, vmid int, memoryKiB float64) string {
	osArch := req.Arch
	if osArch == "" {
		osArch = "x86_64"
	}

	emulatorPath := r.cfg.GetEmulatorPath(osArch)
	machineType := string(req.MachineType)

	// --- BIOS / firmware block ---
	biosXML := ""
	if req.BIOS == model.BIOSTypeOVMF {
		// Select secure boot firmware variant if SecureBoot is enabled
		var ovmfPath string
		if req.SecureBoot {
			ovmfPath = r.cfg.GetOVMFSecureBootPathForArch(osArch)
		} else {
			ovmfPath = r.cfg.GetOVMFPathForArch(osArch)
		}
		if osArch == "aarch64" {
			// aarch64: pflash loader only — no separate NVRAM needed for Alpine/basic EFI
			biosXML = fmt.Sprintf(`
    <loader readonly='yes' type='pflash'>%s</loader>`, ovmfPath)
		} else {
			// x86_64: pflash code + NVRAM vars
			// Use secure boot template for NVRAM if SecureBoot is enabled
			nvramTemplate := ""
			if req.SecureBoot {
				nvramTemplate = " template='/usr/share/OVMF/OVMF_VARS.secboot.fd'"
			}
			nvramPath := fmt.Sprintf("%s/nvram/%s_VARS.fd", r.cfg.Storage.VMDiskDir, req.Name)
			biosXML = fmt.Sprintf(`
    <loader readonly='yes' type='pflash' secure='%s'>%s</loader>
    <nvram%s>%s</nvram>`, boolToStr(req.SecureBoot), ovmfPath, nvramTemplate, nvramPath)
		}
	}

	// --- Disk XML ---
	diskXML := ""
	if req.ISO != "" {
		diskXML = fmt.Sprintf(`
    <disk type='file' device='cdrom'>
      <driver name='qemu' type='raw'/>
      <source file='%s'/>
      <target dev='sda' bus='sata'/>
      <readonly/>
    </disk>`, req.ISO)
	}

	diskPath := fmt.Sprintf("%s/vm-%d.%s", r.cfg.Storage.VMDiskDir, vmid, r.cfg.VM.DiskFormat)
	os.MkdirAll(r.cfg.Storage.VMDiskDir, 0755)

	diskXML += fmt.Sprintf(`
    <disk type='file' device='disk'>
      <driver name='qemu' type='%s'/>
      <source file='%s'/>
      <target dev='vda' bus='%s'/>
    </disk>`, r.cfg.VM.DiskFormat, diskPath, r.cfg.VM.DiskBus)

	// --- CPU block ---
	vcpus := req.CPUSockets * req.CPUCores
	if vcpus == 0 {
		vcpus = 1
	}
	nestedVirtFeature := ""
	if req.NestedVirt {
		nestedVirtFeature = `
      <feature policy='require' name='vmx'/>`
	}
	cpuXML := fmt.Sprintf(`
  <cpu mode='%s' check='none'>
    <topology sockets='%d' cores='%d' threads='1'/>%s
  </cpu>`, req.CPUModel, req.CPUSockets, req.CPUCores, nestedVirtFeature)

	// --- PCI controller XML ---
	// aarch64/virt and q35 use PCIe root; pc (i440fx) uses legacy PCI root
	var pciControllerXML string
	switch {
	case osArch == "aarch64":
		pciControllerXML = `<controller type='pci' index='0' model='pcie-root'/>`
	case req.MachineType == model.MachineTypeQ35:
		pciControllerXML = `<controller type='pci' index='0' model='pcie-root'/>
    <controller type='pci' index='1' model='pcie-root-port'/>`
	default: // pc / i440fx
		pciControllerXML = `<controller type='pci' index='0' model='pci-root'/>`
	}

	// --- Network interface XML ---
	networkXML := buildNetworkXML(req.Network, vmid)

	// --- QEMU guest agent channel ---
	agentChannelXML := ""
	if req.Agent {
		agentChannelXML = `
    <channel type='unix'>
      <target type='virtio' name='org.qemu.guest_agent.0'/>
    </channel>`
	}

	// --- TPM 2.0 device (for Windows 11) ---
	// TPM requires swtpm to be installed on the host
	tpmXML := ""
	if req.TPM && req.BIOS == model.BIOSTypeOVMF && osArch != "aarch64" {
		// TPM state directory for this VM
		tpmStatePath := fmt.Sprintf("%s/tpm/vm-%d", r.cfg.Storage.VMDiskDir, vmid)
		os.MkdirAll(tpmStatePath, 0755)
		tpmXML = fmt.Sprintf(`
    <tpm model='tpm-crb'>
      <backend type='emulator' version='2.0'>
        <active_persistent_state>%s/tpm2-00.permall</active_persistent_state>
      </backend>
    </tpm>`, tpmStatePath)
	}

	// --- PCI Passthrough XML ---
	pciHostdevXML := ""
	for _, pciDev := range req.PCIDevices {
		// Parse PCI address: 0000:01:00.0 -> domain, bus, slot, function
		domain, bus, slot, function := parsePCIAddress(pciDev.Address)
		pciHostdevXML += fmt.Sprintf(`
    <hostdev mode='subsystem' type='pci' managed='yes'>
      <source>
        <address domain='0x%s' bus='0x%s' slot='0x%s' function='0x%s'/>
      </source>
    </hostdev>`, domain, bus, slot, function)
	}

	// --- Arch-specific device / platform XML ---
	// features: x86 uses apic; aarch64 uses GIC
	featuresXML := `  <features>
    <acpi/>
    <apic/>
  </features>`
	if osArch == "aarch64" {
		featuresXML = `  <features>
    <acpi/>
    <gic version='2'/>
  </features>`
	}

	// clock: rtc/pit/hpet timers are x86-only
	clockXML := `  <clock offset='utc'>
    <timer name='rtc' tickpolicy='catchup'/>
    <timer name='pit' tickpolicy='delay'/>
    <timer name='hpet' present='no'/>
  </clock>`
	if osArch == "aarch64" {
		clockXML = `  <clock offset='utc'/>`
	}

	// pm: ACPI S3/S4 sleep states not supported on aarch64 virt
	pmXML := `  <pm>
    <suspend-to-mem enabled='no'/>
    <suspend-to-disk enabled='no'/>
  </pm>`
	if osArch == "aarch64" {
		pmXML = ""
	}

	// USB controller: ich9-ehci1 is x86-only; aarch64 uses xhci
	usbControllerXML := `<controller type='usb' index='0' model='ich9-ehci1'/>`
	if osArch == "aarch64" {
		usbControllerXML = `<controller type='usb' index='0' model='nec-xhci'/>`
	}

	// Serial: isa-serial is x86-only; aarch64 virt uses pl011
	serialXML := `<serial type='pty'>
      <target type='isa-serial' port='0'>
        <model name='isa-serial'/>
      </target>
    </serial>`
	if osArch == "aarch64" {
		serialXML = `<serial type='pty'>
      <target type='system-serial' port='0'>
        <model name='pl011'/>
      </target>
    </serial>`
	}

	// Input: PS/2 is x86-only; aarch64 uses USB
	inputXML := `<input type='tablet' bus='usb'/>
    <input type='mouse' bus='ps2'/>
    <input type='keyboard' bus='ps2'/>`
	if osArch == "aarch64" {
		inputXML = `<input type='tablet' bus='usb'/>
    <input type='mouse' bus='usb'/>
    <input type='keyboard' bus='usb'/>`
	}

	// QEMU commandline arguments for macOS hvf compatibility
	qemuCmdXML := ""
	if osArch == "aarch64" {
		// highmem=off is required for some ARM guests on macOS
		// See: https://gist.github.com/davidandreoletti/af2a17ea095af9476ad012b4a2365a40
		qemuCmdXML = `
  <qemu:commandline>
    <qemu:arg value='-machine'/>
    <qemu:arg value='highmem=off'/>
  </qemu:commandline>`
	}

	// --- Boot order XML ---
	// Default boot order: hd, cdrom (network boot optional)
	bootOrder := req.BootOrder
	if len(bootOrder) == 0 {
		bootOrder = []string{"hd", "cdrom"}
	}
	bootXML := buildBootOrderXML(bootOrder)

	domainXML := fmt.Sprintf(`<domain type='qemu'>
  <name>vm-%d</name>
  <title>%s</title>
  <uuid>%s</uuid>
  <metadata>
    <libosinfo:libosinfo xmlns:libosinfo="http://libosinfo.org/xmlns/libvirt/domain/1.0">
      <libosinfo:os id="%s"/>
    </libosinfo:libosinfo>
  </metadata>
  <memory unit='KiB'>%.0f</memory>
  <currentMemory unit='KiB'>%.0f</currentMemory>
  <vcpu placement='static'>%d</vcpu>
%s
  <os>
    <type arch='%s' machine='%s'>hvm</type>%s
%s
  </os>
%s
%s
  <on_poweroff>destroy</on_poweroff>
  <on_reboot>restart</on_reboot>
  <on_crash>destroy</on_crash>
%s
  <devices>
    <emulator>%s</emulator>
    %s
    %s
    %s
    %s%s
    %s
    <console type='pty'>
      <target type='serial' port='0'/>
    </console>
    %s
    <graphics type='vnc' port='-1' autoport='yes' listen='127.0.0.1'>
      <listen type='address' address='127.0.0.1'/>
    </graphics>
    <video>
      <model type='virtio' heads='1' primary='yes'/>
    </video>
    %s
    %s
    <memballoon model='virtio'/>
  </devices>
%s
</domain>`,
		vmid,
		req.Name,
		generateUUID(),
		mapOSToLibosinfo(req.OS),
		memoryKiB,
		memoryKiB,
		vcpus,
		cpuXML,
		osArch,
		machineType,
		biosXML,
		bootXML,
		featuresXML,
		clockXML,
		pmXML,
		emulatorPath,
		diskXML,
		usbControllerXML,
		pciControllerXML,
		networkXML,
		agentChannelXML,
		serialXML,
		inputXML,
		tpmXML,
		pciHostdevXML,
		qemuCmdXML,
	)

	return domainXML
}

// buildBootOrderXML generates <boot dev='...'/> XML elements for the given boot order.
// Valid devices: "hd" (hard disk), "cdrom" (CD-ROM), "network" (PXE boot)
func buildBootOrderXML(bootOrder []string) string {
	var sb strings.Builder
	for _, dev := range bootOrder {
		// Validate device type
		validDev := dev
		switch dev {
		case "hd", "cdrom", "network":
			// Valid
		default:
			// Skip invalid device types
			continue
		}
		sb.WriteString(fmt.Sprintf("\n    <boot dev='%s'/>", validDev))
	}
	return sb.String()
}

// buildNetworkXML generates <interface> XML blocks for each NetworkConfig entry.
func buildNetworkXML(networks []model.NetworkConfig, vmid int) string {
	var sb strings.Builder
	for i, net := range networks {
		// Derive a deterministic MAC from vmid + interface index
		mac := fmt.Sprintf("52:54:00:%02x:%02x:%02x", byte(vmid>>16), byte(vmid>>8+i), byte(vmid+i))
		switch net.Type {
		case model.NetworkTypeBridge:
			vlanXML := ""
			if net.VLAN > 0 {
				vlanXML = fmt.Sprintf(`
      <vlan>
        <tag id='%d'/>
      </vlan>`, net.VLAN)
			}
			// On macOS, use vmnet-shared instead of vmnet-bridged
			// vmnet-shared provides NAT + VM-to-VM + host-to-VM communication
			// vmnet-bridged requires additional permissions and is more complex
			sb.WriteString(fmt.Sprintf(`
    <interface type='vmnet'>
      <mac address='%s'/>
      <source mode='shared'/>
      <model type='%s'/>%s
    </interface>`, mac, string(net.Model), vlanXML))
		default: // user-mode
			// Build port forwarding configuration
			hostfwd := ""
			if len(net.PortForwards) > 0 {
				for _, pf := range net.PortForwards {
					hostfwd += fmt.Sprintf(",hostfwd=tcp::%s", pf)
				}
			}
			
			sb.WriteString(fmt.Sprintf(`
    <interface type='user'%s>
      <mac address='%s'/>
      <model type='%s'/>
    </interface>`, hostfwd, mac, string(net.Model)))
		}
	}
	return sb.String()
}

// mapOSToLibosinfo maps an OSConfig to a libosinfo OS ID
func mapOSToLibosinfo(osCfg model.OSConfig) string {
	registry := osinfo.New()
	
	// Convert model.OSType to osinfo.OSFamily
	var family osinfo.OSFamily
	switch osCfg.Type {
	case model.OSTypeLinux:
		family = osinfo.OSFamilyLinux
	case model.OSTypeWindows:
		family = osinfo.OSFamilyWindows
	default:
		family = osinfo.OSFamilyOther
	}
	
	return registry.FromOSConfig(family, osCfg.Version)
}

// generateUUID generates a random UUID
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Update updates an existing VM.
// Live updates (no restart required): Name, Description, Tags
// Offline updates (VM must be stopped): CPUSockets, CPUCores, Memory, Agent, NestedVirt, StartOnBoot
func (r *VMRepository) Update(_ context.Context, vmid int, req *model.VMUpdateRequest) (*model.VM, error) {
	// Find the domain by VMID
	domains, err := r.client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}

	var domain libvirtx.Domain
	var found bool
	for _, d := range domains {
		vm := r.domainToVM(d)
		if vm != nil && vm.VMID == vmid {
			domain = d
			found = true
			break
		}
		d.Free()
	}

	if !found {
		return nil, fmt.Errorf("VM with VMID %d not found", vmid)
	}
	defer domain.Free()

	// Check if VM is running
	state, _, err := domain.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain state: %w", err)
	}
	isRunning := state == libvirt.DOMAIN_RUNNING

	// Check for offline-only changes when VM is running
	if isRunning {
		var offlineChanges []string
		if req.CPUSockets != nil {
			offlineChanges = append(offlineChanges, "CPU sockets")
		}
		if req.CPUCores != nil {
			offlineChanges = append(offlineChanges, "CPU cores")
		}
		if req.Memory != nil {
			offlineChanges = append(offlineChanges, "memory")
		}
		if req.Agent != nil {
			offlineChanges = append(offlineChanges, "guest agent")
		}
		if req.NestedVirt != nil {
			offlineChanges = append(offlineChanges, "nested virtualization")
		}
		if req.StartOnBoot != nil {
			offlineChanges = append(offlineChanges, "start on boot")
		}
		if req.TPM != nil {
			offlineChanges = append(offlineChanges, "TPM")
		}
		if req.SecureBoot != nil {
			offlineChanges = append(offlineChanges, "secure boot")
		}
		if len(req.BootOrder) > 0 {
			offlineChanges = append(offlineChanges, "boot order")
		}

		if len(offlineChanges) > 0 {
			return nil, fmt.Errorf("VM must be stopped to change: %s", strings.Join(offlineChanges, ", "))
		}
	}

	// Get current domain XML
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %w", err)
	}

	// Parse XML
	domainXML := &libvirtxml.Domain{}
	if err := domainXML.Unmarshal(xmlDesc); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	// Apply live updates (metadata)
	if req.Name != nil {
		domainXML.Title = *req.Name
	}
	if req.Description != nil {
		domainXML.Description = *req.Description
	}

	// Apply offline updates (require XML modification)
	if !isRunning {
		// CPU topology
		if req.CPUSockets != nil || req.CPUCores != nil {
			if domainXML.CPU == nil {
				domainXML.CPU = &libvirtxml.DomainCPU{}
			}
			if domainXML.CPU.Topology == nil {
				domainXML.CPU.Topology = &libvirtxml.DomainCPUTopology{}
			}
			if req.CPUSockets != nil {
				domainXML.CPU.Topology.Sockets = *req.CPUSockets
			}
			if req.CPUCores != nil {
				domainXML.CPU.Topology.Cores = *req.CPUCores
			}
			// Update vcpu count
			vcpus := uint(domainXML.CPU.Topology.Sockets * domainXML.CPU.Topology.Cores)
			if vcpus > 0 {
				domainXML.VCPU = &libvirtxml.DomainVCPU{
					Value: vcpus,
				}
			}
		}

		// Memory
		if req.Memory != nil {
			memoryKiB := uint(*req.Memory * 1024 * 1024)
			domainXML.Memory = &libvirtxml.DomainMemory{
				Value: memoryKiB,
				Unit:  "KiB",
			}
			domainXML.CurrentMemory = &libvirtxml.DomainCurrentMemory{
				Value: memoryKiB,
				Unit:  "KiB",
			}
		}

		// Guest agent channel
		if req.Agent != nil {
			hasAgent := hasGuestAgentChannel(domainXML)
			if *req.Agent && !hasAgent {
				// Add agent channel
				if domainXML.Devices == nil {
					domainXML.Devices = &libvirtxml.DomainDeviceList{}
				}
				domainXML.Devices.Channels = append(domainXML.Devices.Channels, libvirtxml.DomainChannel{
					Target: &libvirtxml.DomainChannelTarget{
						VirtIO: &libvirtxml.DomainChannelTargetVirtIO{
							Name: "org.qemu.guest_agent.0",
						},
					},
				})
			} else if !*req.Agent && hasAgent {
				// Remove agent channel
				if domainXML.Devices != nil {
					newChannels := []libvirtxml.DomainChannel{}
					for _, ch := range domainXML.Devices.Channels {
						if ch.Target == nil || ch.Target.VirtIO == nil || ch.Target.VirtIO.Name != "org.qemu.guest_agent.0" {
							newChannels = append(newChannels, ch)
						}
					}
					domainXML.Devices.Channels = newChannels
				}
			}
		}

		// Nested virtualization
		if req.NestedVirt != nil {
			if domainXML.CPU == nil {
				domainXML.CPU = &libvirtxml.DomainCPU{}
			}
			hasNestedVirt := hasNestedVirtFeature(domainXML)
			if *req.NestedVirt && !hasNestedVirt {
				// Add nested virt feature
				domainXML.CPU.Features = append(domainXML.CPU.Features, libvirtxml.DomainCPUFeature{
					Policy: "require",
					Name:   "vmx",
				})
			} else if !*req.NestedVirt && hasNestedVirt {
				// Remove nested virt feature
				newFeatures := []libvirtxml.DomainCPUFeature{}
				for _, f := range domainXML.CPU.Features {
					if f.Name != "vmx" && f.Name != "svm" {
						newFeatures = append(newFeatures, f)
					}
				}
				domainXML.CPU.Features = newFeatures
			}
		}

		// Boot order
		if len(req.BootOrder) > 0 {
			if domainXML.OS == nil {
				domainXML.OS = &libvirtxml.DomainOS{}
			}
			// Convert boot order strings to libvirt boot devices
			bootDevices := make([]libvirtxml.DomainBootDevice, 0, len(req.BootOrder))
			for _, dev := range req.BootOrder {
				// Validate device type
				switch dev {
				case "hd", "cdrom", "network":
					bootDevices = append(bootDevices, libvirtxml.DomainBootDevice{
						Dev: dev,
					})
				}
			}
			domainXML.OS.BootDevices = bootDevices
		}
	}

	// Marshal updated XML
	newXML, err := domainXML.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal domain XML: %w", err)
	}

	// Get connection to define the updated domain
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	// Define the updated domain
	if _, err := conn.DomainDefineXML(newXML); err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}

	// Handle StartOnBoot (autostart) - can be changed while running
	if req.StartOnBoot != nil {
		if err := domain.SetAutostart(*req.StartOnBoot); err != nil {
			log.Printf("Update: failed to set autostart: %v", err)
		}
	}

	// Handle tags - stored in metadata (simplified: we don't have a tags field in XML)
	// For now, tags are not persisted in libvirt XML

	// Return updated VM
	return r.GetByVMID(context.Background(), vmid)
}

// hasGuestAgentChannel checks if the domain XML has a guest agent channel
func hasGuestAgentChannel(domainXML *libvirtxml.Domain) bool {
	if domainXML.Devices == nil {
		return false
	}
	for _, ch := range domainXML.Devices.Channels {
		if ch.Target != nil && ch.Target.VirtIO != nil && ch.Target.VirtIO.Name == "org.qemu.guest_agent.0" {
			return true
		}
	}
	return false
}

// hasNestedVirtFeature checks if the domain XML has nested virtualization enabled
func hasNestedVirtFeature(domainXML *libvirtxml.Domain) bool {
	if domainXML.CPU == nil {
		return false
	}
	for _, f := range domainXML.CPU.Features {
		if f.Name == "vmx" || f.Name == "svm" {
			return true
		}
	}
	return false
}

// Delete removes a VM. The VM must be stopped before deletion.
// It undefines the domain and removes associated disk files.
func (r *VMRepository) Delete(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	domain, err := conn.LookupDomainByName(vm.ID)
	if err != nil {
		return fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	// Collect disk paths and NVRAM path from domain XML before undefining
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return fmt.Errorf("failed to get domain XML: %w", err)
	}

	var diskPaths []string
	var nvramPath string
	var hasOVMF bool

	parsed := &libvirtxml.Domain{}
	if parseErr := parsed.Unmarshal(xmlDesc); parseErr == nil {
		if parsed.OS != nil && parsed.OS.Loader != nil {
			hasOVMF = true
		}
		if parsed.OS != nil && parsed.OS.NVRam != nil {
			nvramPath = parsed.OS.NVRam.NVRam
		}
		if parsed.Devices != nil {
			for _, disk := range parsed.Devices.Disks {
				if disk.Device == "disk" && disk.Source != nil && disk.Source.File != nil {
					diskPaths = append(diskPaths, disk.Source.File.File)
				}
			}
		}
	}

	// Undefine the domain with appropriate flags
	undefineFlags := libvirt.DOMAIN_UNDEFINE_MANAGED_SAVE | libvirt.DOMAIN_UNDEFINE_SNAPSHOTS_METADATA
	if hasOVMF {
		undefineFlags |= libvirt.DOMAIN_UNDEFINE_NVRAM
	}

	if err := domain.UndefineFlags(undefineFlags); err != nil {
		return fmt.Errorf("failed to undefine domain: %w", err)
	}

	// Remove disk files
	for _, path := range diskPaths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Printf("Delete: failed to remove disk %s: %v", path, err)
		}
	}

	// Remove NVRAM file if not handled by libvirt (belt-and-suspenders)
	if nvramPath != "" {
		if err := os.Remove(nvramPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Delete: failed to remove NVRAM %s: %v", nvramPath, err)
		}
	}

	return nil
}

// GetVNCPort returns the active VNC port for a running VM.
func (r *VMRepository) GetVNCPort(ctx context.Context, vmid int) (int, error) {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return 0, fmt.Errorf("VM not found: %w", err)
	}

	conn, err := r.client.Connection()
	if err != nil {
		return 0, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	domain, err := conn.LookupDomainByName(vm.ID)
	if err != nil {
		return 0, fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	// Get the active (running) domain XML — flag 0 = active XML with live port
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return 0, fmt.Errorf("failed to get domain XML: %w", err)
	}

	parsed := &libvirtxml.Domain{}
	if err := parsed.Unmarshal(xmlDesc); err != nil {
		return 0, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	if parsed.Devices == nil {
		return 0, fmt.Errorf("no devices found in domain XML")
	}

	for _, graphic := range parsed.Devices.Graphics {
		if graphic.VNC != nil && graphic.VNC.Port > 0 {
			return graphic.VNC.Port, nil
		}
	}

	return 0, fmt.Errorf("no active VNC port found for VM %d", vmid)
}

// Start starts a VM
func (r *VMRepository) Start(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	// Check current state
	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state == libvirt.DOMAIN_RUNNING {
		return fmt.Errorf("VM is already running")
	}

	// Create (start) the domain
	err = domain.Create()
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	return nil
}

// Stop stops a VM (power off)
func (r *VMRepository) Stop(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != libvirt.DOMAIN_RUNNING {
		return fmt.Errorf("VM is not running")
	}

	// Force power off
	err = domain.DestroyFlags(libvirt.DOMAIN_DESTROY_DEFAULT)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down a VM
func (r *VMRepository) Shutdown(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != libvirt.DOMAIN_RUNNING {
		return fmt.Errorf("VM is not running")
	}

	err = domain.ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_DEFAULT)
	if err != nil {
		return fmt.Errorf("failed to shutdown VM: %w", err)
	}

	return nil
}

// Pause pauses/suspends a VM
func (r *VMRepository) Pause(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != libvirt.DOMAIN_RUNNING {
		return fmt.Errorf("VM is not running")
	}

	err = domain.Suspend()
	if err != nil {
		return fmt.Errorf("failed to pause VM: %w", err)
	}

	return nil
}

// Resume resumes a paused VM
func (r *VMRepository) Resume(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != libvirt.DOMAIN_PMSUSPENDED && state != libvirt.DOMAIN_PAUSED {
		return fmt.Errorf("VM is not paused")
	}

	err = domain.Resume()
	if err != nil {
		return fmt.Errorf("failed to resume VM: %w", err)
	}

	return nil
}

// Reboot reboots a VM via hard reset (equivalent to pressing the physical reset button).
// Using Reset instead of ACPI reboot ensures the operation succeeds even on VMs
// without a guest OS that could respond to ACPI signals.
func (r *VMRepository) Reboot(ctx context.Context, vmid int) error {
	vm, err := r.GetByVMID(ctx, vmid)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	domain, err := r.client.GetDomainByName(vm.ID)
	if err != nil {
		return err
	}
	defer domain.Free()

	state, _, err := domain.GetState()
	if err != nil {
		return fmt.Errorf("failed to get domain state: %w", err)
	}

	if state != libvirt.DOMAIN_RUNNING {
		return fmt.Errorf("VM is not running")
	}

	// Hard reset — works without a guest OS (ACPI reboot requires OS support)
	if err = domain.Reset(0); err != nil {
		return fmt.Errorf("failed to reboot VM: %w", err)
	}

	return nil
}

// createDisk creates a new disk image using qemu-img
func (r *VMRepository) createDisk(path string, sizeGB int) error {
	// Ensure directory exists
	if err := os.MkdirAll(r.cfg.Storage.VMDiskDir, 0755); err != nil {
		return fmt.Errorf("failed to create disk directory: %w", err)
	}

	// Create disk image
	cmd := exec.Command("qemu-img", "create", "-f", r.cfg.VM.DiskFormat, path, fmt.Sprintf("%dG", sizeGB))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("qemu-img failed: %v, output: %s", err, string(out))
	}

	return nil
}

// domainToVM converts a libvirt domain to our VM model
func (r *VMRepository) domainToVM(domain libvirtx.Domain) *model.VM {
	// Get domain name (our ID)
	name, err := domain.GetName()
	if err != nil {
		return nil
	}

	// Get domain state
	state, _, err := domain.GetState()
	if err != nil {
		return nil
	}

	// Get domain XML for detailed info
	xmlDesc, err := domain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		xmlDesc = ""
	}

	vm := &model.VM{
		ID:     name,
		VMID:   r.extractVMID(name),
		Name:   name,
		Node:   r.getNodeName(),
		Status: r.convertState(state),
	}

	// Parse XML for additional info
	if xmlDesc != "" {
		r.parseDomainXML(xmlDesc, vm)
	}

	// Get runtime info if running
	if state == libvirt.DOMAIN_RUNNING {
		r.getRuntimeInfo(&domain, vm)
	}

	return vm
}

// extractVMID extracts a numeric ID from domain name
// e.g., "vm-100-myvm" -> 100, "ubuntu-2204" -> 2204
func (r *VMRepository) extractVMID(name string) int {
	// Prioritize our Proxmox-style prefix: vm-<ID>-...
	if strings.HasPrefix(name, "vm-") {
		parts := strings.Split(name, "-")
		if len(parts) >= 2 {
			if val, err := strconv.Atoi(parts[1]); err == nil {
				return val
			}
		}
	}

	// Fallback: Try to extract number from end of name
	parts := strings.Split(name, "-")
	if len(parts) > 1 {
		if val, err := strconv.ParseInt(parts[len(parts)-1], 10, 64); err == nil {
			if val > 0 && val <= 2147483647 {
				return int(val)
			}
		}
	}

	// Fallback 2: Try to find any number in the name
	for i := 0; i < len(name); i++ {
		if name[i] >= '0' && name[i] <= '9' {
			end := i
			for end < len(name) && name[end] >= '0' && name[end] <= '9' {
				end++
			}
			if val, err := strconv.ParseInt(name[i:end], 10, 64); err == nil {
				if val > 0 && val <= 2147483647 {
					return int(val)
				}
			}
		}
	}

	// Fallback 3: generate ID from hash
	hash := uint32(0)
	for _, c := range name {
		hash = hash*31 + uint32(c)
	}
	return int(hash%1000000) + 10000 // Higher range for hashes to avoid collisions with low IDs
}

// convertState converts libvirt domain state to our status
func (r *VMRepository) convertState(state libvirt.DomainState) model.VMStatus {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return model.VMStatusRunning
	case libvirt.DOMAIN_BLOCKED:
		return model.VMStatusRunning // Blocked is still running
	case libvirt.DOMAIN_PAUSED:
		return model.VMStatusPaused
	case libvirt.DOMAIN_SHUTDOWN:
		return model.VMStatusStopped
	case libvirt.DOMAIN_SHUTOFF:
		return model.VMStatusStopped
	case libvirt.DOMAIN_CRASHED:
		return model.VMStatusStopped
	case libvirt.DOMAIN_PMSUSPENDED:
		return model.VMStatusSuspended
	default:
		return model.VMStatusStopped
	}
}

// getNodeName returns the current node name
func (r *VMRepository) getNodeName() string {
	hostname, err := r.client.GetHostname()
	if err != nil {
		return "localhost"
	}
	return hostname
}

// parseDomainXML parses domain XML for configuration info
func (r *VMRepository) parseDomainXML(xmlDesc string, vm *model.VM) {
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(xmlDesc); err != nil {
		return
	}

	// Use domain title for VM Name if present, fall back to domain Name (ID)
	if domain.Title != "" {
		vm.Name = domain.Title
	} else {
		vm.Name = domain.Name
	}

	// CPU info: prefer topology sockets*cores; fall back to vcpu count
	if domain.CPU != nil && domain.CPU.Topology != nil {
		vm.CPU.Sockets = int(domain.CPU.Topology.Sockets)
		vm.CPU.Cores = int(domain.CPU.Topology.Cores)
	}
	if domain.VCPU != nil && vm.CPU.Cores == 0 {
		vm.CPU.Cores = int(domain.VCPU.Value)
		vm.CPU.Sockets = 1
	}

	// Memory info (in KiB, convert to GB)
	if domain.Memory != nil {
		memKiB := domain.Memory.Value
		switch domain.Memory.Unit {
		case "KiB":
			vm.Memory.Total = float64(memKiB) / 1024 / 1024
		case "MiB":
			vm.Memory.Total = float64(memKiB) / 1024
		case "GiB":
			vm.Memory.Total = float64(memKiB)
		case "TiB":
			vm.Memory.Total = float64(memKiB) * 1024
		default:
			// Assume bytes
			vm.Memory.Total = float64(memKiB) / 1024 / 1024 / 1024
		}
	}

	// OS info — derive machine type and BIOS from domain XML
	if domain.OS != nil && domain.OS.Type != nil {
		vm.MachineType = model.MachineType(domain.OS.Type.Machine)
		// If the machine string contains "q35" treat it as q35
		if strings.Contains(domain.OS.Type.Machine, "q35") {
			vm.MachineType = model.MachineTypeQ35
		} else if vm.MachineType == "" {
			vm.MachineType = model.MachineTypePC
		}
	}
	if domain.OS != nil && domain.OS.Loader != nil {
		vm.BIOS = model.BIOSTypeOVMF
	} else {
		vm.BIOS = model.BIOSTypeSeaBIOS
	}

	// Boot order from OS section
	if domain.OS != nil && len(domain.OS.BootDevices) > 0 {
		vm.BootOrder = []string{}
		for _, dev := range domain.OS.BootDevices {
			vm.BootOrder = append(vm.BootOrder, string(dev.Dev))
		}
	}

	// OS config — default to linux/unknown; libosinfo metadata could be parsed here in future
	vm.OS = model.OSConfig{Type: model.OSTypeLinux, Version: "unknown"}

	// CPU model
	if domain.CPU != nil {
		vm.CPUModel = domain.CPU.Mode
		if domain.CPU.Model != nil {
			vm.CPUModel = domain.CPU.Model.Value
		}
		if vm.CPUModel == "" {
			vm.CPUModel = "host-passthrough"
		}
	}

	// Network interfaces — type is determined by which Source sub-field is set
	if domain.Devices != nil {
		vm.Network = []model.NetworkConfig{}
		for _, iface := range domain.Devices.Interfaces {
			nc := model.NetworkConfig{}
			if iface.Source != nil && iface.Source.Bridge != nil {
				nc.Type = model.NetworkTypeBridge
				nc.Bridge = iface.Source.Bridge.Bridge
			} else {
				nc.Type = model.NetworkTypeUser
			}
			if iface.Model != nil {
				nc.Model = model.NetworkModel(iface.Model.Type)
			}
			if iface.VLan != nil && len(iface.VLan.Tags) > 0 {
				nc.VLAN = int(iface.VLan.Tags[0].ID)
			}
			vm.Network = append(vm.Network, nc)
		}
	}

	// Tags from metadata
	if domain.Metadata != nil {
		// Parse custom metadata for tags
		vm.Tags = []string{}
	}

	// Description
	if domain.Description != "" {
		vm.Description = domain.Description
	}
}

// getRuntimeInfo gets runtime information for a running domain
func (r *VMRepository) getRuntimeInfo(domain *libvirtx.Domain, vm *model.VM) {
	// Get memory stats
	memStats, err := domain.MemoryStats(5, 0)
	if err == nil {
		for _, stat := range memStats {
			if stat.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_AVAILABLE) {
				availableGB := float64(stat.Val) / 1024 / 1024
				vm.Memory.Used = roundTo(vm.Memory.Total-availableGB, 2)
			}
		}
	}

	// Get interface info for IP
	ifaces, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err == nil {
		for _, iface := range ifaces {
			if len(iface.Addrs) > 0 {
				vm.IP = iface.Addrs[0].Addr
				break
			}
		}
	}

	// Set default values if not set
	if vm.Memory.Used == 0 && vm.Memory.Total > 0 {
		vm.Memory.Used = roundTo(vm.Memory.Total*0.3, 2) // Estimate 30% usage
	}
	if vm.CPU.Used == 0 {
		vm.CPU.Used = 5 // Default low usage
	}
}

// roundTo rounds a float to specified decimal places
func roundTo(val float64, places int) float64 {
	multiplier := 1.0
	for i := 0; i < places; i++ {
		multiplier *= 10
	}
	return float64(int(val*multiplier+0.5)) / multiplier
}

// Clone creates a clone of an existing VM.
// For full clones: copies all disk images using qemu-img convert
// For linked clones: creates new disks with backing file pointing to source
func (r *VMRepository) Clone(_ context.Context, req *model.VMCloneRequest, progressFunc func(int, string)) (*model.VM, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Report initial progress
	if progressFunc != nil {
		progressFunc(5, "Finding source VM")
	}

	// Get source domain
	sourceDomainName := fmt.Sprintf("vm-%d", req.SourceVMID)
	sourceDomain, err := r.client.GetDomainByName(sourceDomainName)
	if err != nil {
		return nil, fmt.Errorf("source VM %d not found: %w", req.SourceVMID, err)
	}
	defer sourceDomain.Free()

	// Verify source VM is stopped
	state, _, err := sourceDomain.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get source VM state: %w", err)
	}
	if state == libvirt.DOMAIN_RUNNING || state == libvirt.DOMAIN_PAUSED {
		return nil, fmt.Errorf("source VM must be stopped before cloning (current state: %s)", domainStateToString(state))
	}

	if progressFunc != nil {
		progressFunc(10, "Reading VM configuration")
	}

	// Get source VM XML
	sourceXML, err := sourceDomain.GetXMLDesc(libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to get source VM XML: %w", err)
	}

	// Parse the XML
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(sourceXML); err != nil {
		return nil, fmt.Errorf("failed to parse source VM XML: %w", err)
	}

	// Generate new VM ID
	newVMID := r.generateVMID()
	newDomainName := fmt.Sprintf("vm-%d", newVMID)
	newUUID := generateUUID()

	if progressFunc != nil {
		progressFunc(15, "Analyzing disk configuration")
	}

	// Extract disk information from source
	type diskInfo struct {
		sourcePath string
		targetDev  string
		bus        string
		format     string
		device     string // "disk" or "cdrom"
		readonly   bool
	}
	var disks []diskInfo

	if domain.Devices != nil {
		for _, disk := range domain.Devices.Disks {
			if disk.Source == nil || disk.Source.File == nil {
				continue
			}
			info := diskInfo{
				sourcePath: disk.Source.File.File,
				targetDev:  disk.Target.Dev,
				bus:        disk.Target.Bus,
				device:     disk.Device,
				readonly:   disk.ReadOnly != nil,
			}
			if disk.Driver != nil {
				info.format = disk.Driver.Type
			}
			// Skip CD-ROM drives (they reference ISO files)
			if disk.Device == "cdrom" {
				continue
			}
			disks = append(disks, info)
		}
	}

	if len(disks) == 0 {
		return nil, fmt.Errorf("source VM has no disks to clone")
	}

	// Copy disks
	totalDisks := len(disks)
	for i, disk := range disks {
		progressPercent := 20 + (i * 60 / totalDisks)
		if progressFunc != nil {
			progressFunc(progressPercent, fmt.Sprintf("Copying disk %d of %d (%s)", i+1, totalDisks, disk.targetDev))
		}

		// Generate new disk path
		var newDiskPath string
		if i == 0 {
			// Primary disk uses vm-<id>.<format>
			newDiskPath = fmt.Sprintf("%s/%s.%s", r.cfg.Storage.VMDiskDir, newDomainName, disk.format)
		} else {
			// Additional disks use vm-<id>-<target>.<format>
			newDiskPath = fmt.Sprintf("%s/%s-%s.%s", r.cfg.Storage.VMDiskDir, newDomainName, disk.targetDev, disk.format)
		}

		// Copy the disk
		if err := r.cloneDisk(disk.sourcePath, newDiskPath, disk.format, req.Full); err != nil {
			// Cleanup any already copied disks
			r.cleanupClonedDisks(newDomainName, i)
			return nil, fmt.Errorf("failed to clone disk %s: %w", disk.targetDev, err)
		}

		// Update the XML to point to the new disk path
		if domain.Devices != nil {
			for j, d := range domain.Devices.Disks {
				if d.Target.Dev == disk.targetDev {
					domain.Devices.Disks[j].Source.File.File = newDiskPath
					break
				}
			}
		}
	}

	if progressFunc != nil {
		progressFunc(85, "Creating VM configuration")
	}

	// Update domain XML for the clone
	domain.Name = newDomainName
	domain.UUID = newUUID

	// Update title (user-visible name)
	domain.Title = req.Name

	// Update description if provided
	if req.Description != "" {
		domain.Description = req.Description
	}

	// Generate new MAC addresses for all network interfaces
	if domain.Devices != nil {
		for i := range domain.Devices.Interfaces {
			// Generate a unique MAC based on new VMID and interface index
			mac := fmt.Sprintf("52:54:00:%02x:%02x:%02x", byte(newVMID>>16), byte(newVMID>>8+i), byte(newVMID+i))
			if domain.Devices.Interfaces[i].MAC == nil {
				domain.Devices.Interfaces[i].MAC = &libvirtxml.DomainInterfaceMAC{}
			}
			domain.Devices.Interfaces[i].MAC.Address = mac
		}
	}

	// Clear any existing NVRAM path (will be regenerated on first boot if using OVMF)
	if domain.OS != nil && domain.OS.NVRam != nil {
		// Generate new NVRAM path for the clone
		nvramPath := fmt.Sprintf("%s/nvram/%s_VARS.fd", r.cfg.Storage.VMDiskDir, newDomainName)
		domain.OS.NVRam.NVRam = nvramPath
	}

	// Marshal the updated XML
	newXML, err := domain.Marshal()
	if err != nil {
		r.cleanupClonedDisks(newDomainName, len(disks))
		return nil, fmt.Errorf("failed to marshal cloned VM XML: %w", err)
	}

	if progressFunc != nil {
		progressFunc(90, "Defining cloned VM")
	}

	// Define the new domain
	conn, err := r.client.Connection()
	if err != nil {
		r.cleanupClonedDisks(newDomainName, len(disks))
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	newDomain, err := conn.DomainDefineXML(newXML)
	if err != nil {
		r.cleanupClonedDisks(newDomainName, len(disks))
		return nil, fmt.Errorf("failed to define cloned VM: %w", err)
	}
	defer newDomain.Free()

	// Set autostart if requested (based on source VM setting)
	autostart := domain.OnPoweroff == "destroy" // Default behavior
	if autostart {
		if err := newDomain.SetAutostart(true); err != nil {
			log.Printf("Clone: failed to set autostart: %v", err)
		}
	}

	if progressFunc != nil {
		progressFunc(95, "Finalizing clone")
	}

	// Optionally start the clone
	if req.StartAfterClone {
		if err := newDomain.Create(); err != nil {
			// Log warning but don't fail - the clone was created successfully
			log.Printf("Warning: failed to start cloned VM %s: %v", newDomainName, err)
		}
	}

	if progressFunc != nil {
		progressFunc(100, "Clone complete")
	}

	// Get the source VM to copy metadata
	sourceVM := r.domainToVM(sourceDomain)

	// Return the cloned VM
	return &model.VM{
		ID:          newDomainName,
		VMID:        newVMID,
		Name:        req.Name,
		Node:        r.getNodeName(),
		Status:      model.VMStatusStopped,
		CPU:         sourceVM.CPU,
		Memory:      sourceVM.Memory,
		Disk:        sourceVM.Disk,
		Uptime:      "0",
		OS:          sourceVM.OS,
		Arch:        sourceVM.Arch,
		MachineType: sourceVM.MachineType,
		BIOS:        sourceVM.BIOS,
		CPUModel:    sourceVM.CPUModel,
		Network:     sourceVM.Network,
		IP:          "",
		Tags:        sourceVM.Tags,
		HA:          sourceVM.HA,
		Description: req.Description,
		NestedVirt:  sourceVM.NestedVirt,
		StartOnBoot: sourceVM.StartOnBoot,
		Agent:       sourceVM.Agent,
	}, nil
}

// cloneDisk copies a disk image using qemu-img.
// For full clones: uses convert to create an independent copy
// For linked clones: uses create with backing file
func (r *VMRepository) cloneDisk(sourcePath, destPath, format string, full bool) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create disk directory: %w", err)
	}

	var cmd *exec.Cmd
	if full {
		// Full clone: convert creates an independent copy
		cmd = exec.Command("qemu-img", "convert", "-f", format, "-O", format, sourcePath, destPath)
	} else {
		// Linked clone: create new disk with backing file
		cmd = exec.Command("qemu-img", "create", "-f", format, "-b", sourcePath, "-F", format, destPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("qemu-img failed: %w, output: %s", err, string(output))
	}

	return nil
}

// cleanupClonedDisks removes any disks that were created during a failed clone operation
func (r *VMRepository) cleanupClonedDisks(domainName string, diskCount int) {
	for i := 0; i < diskCount; i++ {
		var diskPath string
		if i == 0 {
			diskPath = fmt.Sprintf("%s/%s.qcow2", r.cfg.Storage.VMDiskDir, domainName)
		} else {
			// Try common target names
			for _, target := range []string{"vdb", "vdc", "vdd", "sdb", "sdc"} {
				diskPath = fmt.Sprintf("%s/%s-%s.qcow2", r.cfg.Storage.VMDiskDir, domainName, target)
				if _, err := os.Stat(diskPath); err == nil {
					os.Remove(diskPath)
				}
				// Also try raw format
				diskPath = fmt.Sprintf("%s/%s-%s.raw", r.cfg.Storage.VMDiskDir, domainName, target)
				if _, err := os.Stat(diskPath); err == nil {
					os.Remove(diskPath)
				}
			}
		}
		if _, err := os.Stat(diskPath); err == nil {
			os.Remove(diskPath)
		}
		// Also try raw format
		diskPath = fmt.Sprintf("%s/%s.raw", r.cfg.Storage.VMDiskDir, domainName)
		if _, err := os.Stat(diskPath); err == nil {
			os.Remove(diskPath)
		}
	}
}

// domainStateToString converts libvirt domain state to a human-readable string
func domainStateToString(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_RUNNING:
		return "running"
	case libvirt.DOMAIN_BLOCKED:
		return "blocked"
	case libvirt.DOMAIN_PAUSED:
		return "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "stopped"
	case libvirt.DOMAIN_CRASHED:
		return "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "suspended"
	default:
		return "unknown"
	}
}

// boolToStr converts a bool to "yes" or "no" for libvirt XML attributes
func boolToStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// parsePCIAddress parses a PCI address string into domain, bus, slot, function
// Input format: 0000:01:00.0 or 01:00.0
func parsePCIAddress(addr string) (domain, bus, slot, function string) {
	parts := strings.Split(addr, ":")
	if len(parts) == 3 {
		// Full format: domain:bus:slot.function
		domain = parts[0]
		bus = parts[1]
		slotFunc := strings.Split(parts[2], ".")
		if len(slotFunc) == 2 {
			slot = slotFunc[0]
			function = slotFunc[1]
		}
	} else if len(parts) == 2 {
		// Short format: bus:slot.function (domain defaults to 0000)
		domain = "0000"
		bus = parts[0]
		slotFunc := strings.Split(parts[1], ".")
		if len(slotFunc) == 2 {
			slot = slotFunc[0]
			function = slotFunc[1]
		}
	}
	return
}
