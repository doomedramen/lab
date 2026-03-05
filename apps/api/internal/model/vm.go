package model

// VMStatus represents the operational status of a VM
type VMStatus string

const (
	VMStatusRunning   VMStatus = "running"
	VMStatusStopped   VMStatus = "stopped"
	VMStatusPaused    VMStatus = "paused"
	VMStatusSuspended VMStatus = "suspended"
)

// OSType represents the operating system family
type OSType string

const (
	OSTypeLinux   OSType = "linux"
	OSTypeWindows OSType = "windows"
	OSTypeSolaris OSType = "solaris"
	OSTypeOther   OSType = "other"
)

// MachineType represents the QEMU machine type
type MachineType string

const (
	MachineTypePC   MachineType = "pc"   // i440fx (x86)
	MachineTypeQ35  MachineType = "q35"  // ICH9/Q35 (x86)
	MachineTypeVirt MachineType = "virt" // ARM virt
)

// BIOSType represents the firmware/BIOS type
type BIOSType string

const (
	BIOSTypeSeaBIOS BIOSType = "seabios"
	BIOSTypeOVMF    BIOSType = "ovmf"
)

// NetworkModel represents the NIC emulation model
type NetworkModel string

const (
	NetworkModelVirtio  NetworkModel = "virtio"
	NetworkModelE1000   NetworkModel = "e1000"
	NetworkModelRTL8139 NetworkModel = "rtl8139"
)

// NetworkType represents the network attachment type
type NetworkType string

const (
	NetworkTypeUser   NetworkType = "user"
	NetworkTypeBridge NetworkType = "bridge"
)

// OSConfig represents structured OS identification
type OSConfig struct {
	Type    OSType `json:"type"`
	Version string `json:"version"` // e.g. "ubuntu-24.04", "11", "2022"
}

// NetworkConfig represents network interface configuration for a VM
type NetworkConfig struct {
	Type         NetworkType  `json:"type"`
	Bridge       string       `json:"bridge,omitempty"` // required when type=bridge
	Model        NetworkModel `json:"model"`
	VLAN         int          `json:"vlan,omitempty"`
	PortForwards []string     `json:"port_forwards,omitempty"` // Format: "host_port:guest_port" (e.g., "2222:22")
}

// VM represents a virtual machine
type VM struct {
	ID          string          `json:"id"`
	VMID        int             `json:"vmid"`
	Name        string          `json:"name"`
	Node        string          `json:"node"`
	Status      VMStatus        `json:"status"`
	CPU         CPUInfoPartial  `json:"cpu"`
	Memory      MemoryInfo      `json:"memory"`
	Disk        DiskInfo        `json:"disk"`
	Uptime      string          `json:"uptime"`
	OS          OSConfig        `json:"os"`
	Arch        string          `json:"arch"`        // e.g. "x86_64", "aarch64"
	MachineType MachineType     `json:"machineType"`
	BIOS        BIOSType        `json:"bios"`
	CPUModel    string          `json:"cpuModel"`
	Network     []NetworkConfig `json:"network"`
	IP          string          `json:"ip"`
	Tags        []string        `json:"tags"`
	HA          bool            `json:"ha"`
	Description string          `json:"description"`
	NestedVirt  bool            `json:"nestedVirt"`
	StartOnBoot bool            `json:"startOnBoot"`
	Agent       bool            `json:"agent"`
	TPM         bool            `json:"tpm"`         // TPM 2.0 device enabled
	SecureBoot  bool            `json:"secureBoot"`  // Secure Boot enabled (requires OVMF)
	PCIDevices  []PCIDevice     `json:"pciDevices"`  // PCI passthrough devices
	BootOrder   []string        `json:"bootOrder"`   // Boot device priority (e.g., ["hd", "cdrom", "network"])
}

// VMCreateRequest represents the request body for creating a VM
type VMCreateRequest struct {
	// General
	Name        string   `json:"name"`
	Node        string   `json:"node"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	StartOnBoot bool     `json:"startOnBoot"`

	// OS
	OS OSConfig `json:"os"`

	// System
	Arch        string      `json:"arch"`        // "" → "x86_64"; also "aarch64"
	MachineType MachineType `json:"machineType"` // "" → smart default based on arch/OS
	BIOS        BIOSType    `json:"bios"`        // "" → smart default based on arch/OS
	Agent       bool        `json:"agent"`
	TPM         bool        `json:"tpm"`        // TPM 2.0 device (requires OVMF)
	SecureBoot  bool        `json:"secureBoot"` // Secure Boot (requires OVMF)

	// Disks
	ISO     string  `json:"iso"`      // Path to local ISO or empty
	ISOURL  string  `json:"isoUrl"`   // URL to download ISO from (template-based)
	ISOName string  `json:"isoName"`  // Name for downloaded ISO
	Disk    float64 `json:"disk"`     // GB

	// CPU
	CPUSockets int    `json:"cpuSockets"` // default 1
	CPUCores   int    `json:"cpuCores"`   // default 1
	CPUModel   string `json:"cpuModel"`   // default "host-passthrough"
	NestedVirt bool   `json:"nestedVirt"`

	// Memory
	Memory float64 `json:"memory"` // GB

	// Network
	Network []NetworkConfig `json:"network"` // default: [{type:user, model:virtio}]

	// Additional disks (optional)
	AdditionalDisks []DiskConfig `json:"additionalDisks,omitempty"`

	// PCI passthrough devices
	PCIDevices []PCIDevice `json:"pciDevices,omitempty"` // PCI addresses to pass through

	// Boot order - devices to boot from in priority order (e.g., ["hd", "cdrom", "network"])
	BootOrder []string `json:"bootOrder,omitempty"`
}

// VMUpdateRequest represents the request body for updating a VM
// Fields are pointers to distinguish between "not set" and "set to zero/false"
// Live updates (no restart required): Name, Description, Tags
// Offline updates (VM must be stopped): CPUSockets, CPUCores, Memory, Agent, NestedVirt, StartOnBoot, TPM, SecureBoot, PCIDevices
type VMUpdateRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// CPU/Memory - require VM to be stopped for persistent changes
	CPUSockets *int `json:"cpuSockets,omitempty"`
	CPUCores   *int `json:"cpuCores,omitempty"`
	Memory     *float64 `json:"memory,omitempty"`

	// Features - require VM to be stopped
	Agent       *bool `json:"agent,omitempty"`
	NestedVirt  *bool `json:"nestedVirt,omitempty"`
	StartOnBoot *bool `json:"startOnBoot,omitempty"`
	TPM         *bool `json:"tpm,omitempty"`        // TPM 2.0 device (requires OVMF)
	SecureBoot  *bool `json:"secureBoot,omitempty"` // Secure Boot (requires OVMF)

	// PCI devices - require VM to be stopped
	PCIDevices []PCIDevice `json:"pciDevices,omitempty"` // PCI addresses to pass through

	// Boot order - devices to boot from in priority order (e.g., ["hd", "cdrom", "network"])
	BootOrder []string `json:"bootOrder,omitempty"`
}

// VMDisk represents a disk attached to a VM
type VMDisk struct {
	ID        string    `json:"id"`
	VMID      int       `json:"vmid"`
	Target    string    `json:"target"` // vda, vdb, sda, etc.
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	Bus       DiskBus   `json:"bus"`
	Format    DiskFormat `json:"format"`
	Readonly  bool      `json:"readonly"`
	BootOrder int       `json:"boot_order"` // 1 = first boot disk, 0 = no boot order
}

// DiskConfig represents configuration for a disk at VM creation
type DiskConfig struct {
	SizeGB   float64 `json:"size_gb"`
	Bus      DiskBus `json:"bus"`
	Format   DiskFormat `json:"format"`
	Readonly bool    `json:"readonly"`
	StoragePool string `json:"storage_pool"` // Target storage pool for the disk
}

// VMLogEntry represents a VM log entry
type VMLogEntry struct {
	ID        int64             `json:"id"`
	VMID      int               `json:"vmid"`
	Level     string            `json:"level"`
	Source    string            `json:"source"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// VMCloneRequest represents a request to clone a VM
type VMCloneRequest struct {
	SourceVMID      int    `json:"source_vmid"`       // VMID of the source VM
	Name            string `json:"name"`              // Name for the cloned VM
	Full            bool   `json:"full"`              // true = full clone, false = linked clone
	TargetPool      string `json:"target_pool"`       // Storage pool for new disks (empty = same as source)
	Description     string `json:"description"`       // Optional description for the clone
	StartAfterClone bool   `json:"start_after_clone"` // Start the clone after creation
}

// GuestNetworkInterface represents a network interface from the guest agent
type GuestNetworkInterface struct {
	Name           string            `json:"name"`           // Interface name (e.g., "eth0", "ens3")
	MACAddress     string            `json:"mac_address"`    // Hardware/MAC address
	IPAddresses    []GuestIPAddress  `json:"ip_addresses"`   // List of IP addresses
}

// GuestIPAddress represents an IP address from the guest agent
type GuestIPAddress struct {
	Address     string `json:"address"`      // IP address (e.g., "192.168.1.100")
	Prefix      int    `json:"prefix"`       // Network prefix/CIDR (e.g., 24)
	AddressType string `json:"address_type"` // "ipv4" or "ipv6"
}

// GuestAgentStatus represents the connection status of the QEMU guest agent
type GuestAgentStatus struct {
	Connected bool   `json:"connected"` // Whether the agent is connected
	Version   string `json:"version"`   // Agent version if available
}

// PCIDevice represents a PCI device on the host or attached to a VM
type PCIDevice struct {
	Address     string `json:"address"`     // PCI address (e.g., "0000:01:00.0")
	VendorID    string `json:"vendor_id"`   // Vendor ID (hex string, e.g., "10de")
	VendorName  string `json:"vendor_name"` // Vendor name (e.g., "NVIDIA Corporation")
	ProductID   string `json:"product_id"`  // Product ID (hex string, e.g., "1b80")
	ProductName string `json:"product_name"` // Product name (e.g., "GP104 [GeForce GTX 1080]")
	Driver      string `json:"driver"`      // Current kernel driver (e.g., "nvidia", "vfio-pci")
	IOMMUGroup  int    `json:"iommu_group"` // IOMMU group number (-1 if not available)
	Class       string `json:"class"`       // Device class (e.g., "0300" for VGA)
	ClassName   string `json:"class_name"`  // Human-readable class name (e.g., "VGA compatible controller")
}
