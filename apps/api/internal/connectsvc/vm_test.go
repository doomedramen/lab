package connectsvc

import (
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelVMStatusToProto(t *testing.T) {
	tests := []struct {
		input    model.VMStatus
		expected labv1.VmStatus
	}{
		{model.VMStatusRunning, labv1.VmStatus_VM_STATUS_RUNNING},
		{model.VMStatusStopped, labv1.VmStatus_VM_STATUS_STOPPED},
		{model.VMStatusPaused, labv1.VmStatus_VM_STATUS_PAUSED},
		{model.VMStatusSuspended, labv1.VmStatus_VM_STATUS_SUSPENDED},
		{model.VMStatus("unknown"), labv1.VmStatus_VM_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelVMStatusToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelVMStatusToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelVMToProto(t *testing.T) {
	vm := &model.VM{
		ID:          "vm-123",
		VMID:        1,
		Name:        "test-vm",
		Node:        "node-1",
		Status:      model.VMStatusRunning,
		CPU:         model.CPUInfoPartial{Used: 50.0, Sockets: 1, Cores: 2},
		Memory:      model.MemoryInfo{Used: 4.0, Total: 8.0},
		Disk:        model.DiskInfo{Used: 20.0, Total: 100.0},
		Uptime:      "1h0m0s",
		OS:          model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-22.04"},
		Arch:        "x86_64",
		MachineType: model.MachineTypeQ35,
		BIOS:        model.BIOSTypeSeaBIOS,
		CPUModel:    "host",
		Network: []model.NetworkConfig{
			{Type: model.NetworkTypeBridge, Bridge: "br0", Model: model.NetworkModelVirtio},
		},
		IP:          "192.168.1.100",
		Tags:        []string{"production", "web"},
		HA:          true,
		Description: "Test VM",
		NestedVirt:  false,
		StartOnBoot: true,
		Agent:       true,
	}

	p := modelVMToProto(vm)

	if p.Id != vm.ID {
		t.Errorf("Id: got %q, want %q", p.Id, vm.ID)
	}
	if p.Vmid != int32(vm.VMID) {
		t.Errorf("Vmid: got %d, want %d", p.Vmid, vm.VMID)
	}
	if p.Name != vm.Name {
		t.Errorf("Name: got %q, want %q", p.Name, vm.Name)
	}
	if p.Node != vm.Node {
		t.Errorf("Node: got %q, want %q", p.Node, vm.Node)
	}
	if p.Status != labv1.VmStatus_VM_STATUS_RUNNING {
		t.Errorf("Status: got %v, want RUNNING", p.Status)
	}
	if p.Uptime != vm.Uptime {
		t.Errorf("Uptime: got %q, want %q", p.Uptime, vm.Uptime)
	}
	if p.Arch != vm.Arch {
		t.Errorf("Arch: got %q, want %q", p.Arch, vm.Arch)
	}
	if p.CpuModel != vm.CPUModel {
		t.Errorf("CpuModel: got %q, want %q", p.CpuModel, vm.CPUModel)
	}
	if p.Ip != vm.IP {
		t.Errorf("Ip: got %q, want %q", p.Ip, vm.IP)
	}
	if len(p.Tags) != 2 {
		t.Errorf("Tags: expected 2, got %d", len(p.Tags))
	}
	if !p.Ha {
		t.Error("Ha: expected true")
	}
	if p.Description != vm.Description {
		t.Errorf("Description: got %q, want %q", p.Description, vm.Description)
	}
	if p.NestedVirt {
		t.Error("NestedVirt: expected false")
	}
	if !p.StartOnBoot {
		t.Error("StartOnBoot: expected true")
	}
	if !p.Agent {
		t.Error("Agent: expected true")
	}
}

func TestProtoCreateVMRequestToModel(t *testing.T) {
	req := &labv1.CreateVMRequest{
		Name:        "test-vm",
		Node:        "node-1",
		Tags:        []string{"tag1"},
		Description: "Test VM",
		StartOnBoot: true,
		Os:          &labv1.OsConfig{OsType: labv1.OsType_OS_TYPE_LINUX, Version: "ubuntu"},
		Arch:        "x86_64",
		MachineType: labv1.MachineType_MACHINE_TYPE_Q35,
		Bios:        labv1.BiosType_BIOS_TYPE_SEABIOS,
		Agent:       true,
		Iso:         "ubuntu.iso",
		IsoUrl:      "http://example.com/ubuntu.iso",
		IsoName:     "ubuntu-22.04.iso",
		DiskGb:      50.0,
		CpuSockets:  1,
		CpuCores:    2,
		CpuModel:    "host",
		NestedVirt:  false,
		MemoryGb:    4.0,
		Network: []*labv1.NetworkConfig{
			{Type: labv1.NetworkType_NETWORK_TYPE_BRIDGE, Bridge: "br0", Model: labv1.NetworkModel_NETWORK_MODEL_VIRTIO},
		},
	}

	result := protoCreateVMRequestToModel(req)

	if result.Name != "test-vm" {
		t.Errorf("Name: got %q, want test-vm", result.Name)
	}
	if result.Node != "node-1" {
		t.Errorf("Node: got %q, want node-1", result.Node)
	}
	if len(result.Tags) != 1 {
		t.Errorf("Tags: expected 1, got %d", len(result.Tags))
	}
	if result.Description != "Test VM" {
		t.Errorf("Description: got %q", result.Description)
	}
	if !result.StartOnBoot {
		t.Error("StartOnBoot: expected true")
	}
	if result.OS.Type != model.OSTypeLinux {
		t.Errorf("OS.Type: got %v, want Linux", result.OS.Type)
	}
	if result.Arch != "x86_64" {
		t.Errorf("Arch: got %q", result.Arch)
	}
	if result.MachineType != model.MachineTypeQ35 {
		t.Errorf("MachineType: got %v, want Q35", result.MachineType)
	}
	if result.BIOS != model.BIOSTypeSeaBIOS {
		t.Errorf("BIOS: got %v, want SeaBIOS", result.BIOS)
	}
	if !result.Agent {
		t.Error("Agent: expected true")
	}
	if result.ISO != "ubuntu.iso" {
		t.Errorf("ISO: got %q", result.ISO)
	}
	if result.ISOURL != "http://example.com/ubuntu.iso" {
		t.Errorf("ISOURL: got %q", result.ISOURL)
	}
	if result.Disk != 50.0 {
		t.Errorf("Disk: got %v, want 50", result.Disk)
	}
	if result.CPUSockets != 1 {
		t.Errorf("CPUSockets: got %d, want 1", result.CPUSockets)
	}
	if result.CPUCores != 2 {
		t.Errorf("CPUCores: got %d, want 2", result.CPUCores)
	}
	if result.CPUModel != "host" {
		t.Errorf("CPUModel: got %q", result.CPUModel)
	}
	if result.NestedVirt {
		t.Error("NestedVirt: expected false")
	}
	if result.Memory != 4.0 {
		t.Errorf("Memory: got %v, want 4", result.Memory)
	}
	if len(result.Network) != 1 {
		t.Errorf("Network: expected 1, got %d", len(result.Network))
	}
}

func TestVMLogLevelStringToProto(t *testing.T) {
	tests := []struct {
		input    string
		expected labv1.VMLogLevel
	}{
		{"DEBUG", labv1.VMLogLevel_VM_LOG_LEVEL_DEBUG},
		{"INFO", labv1.VMLogLevel_VM_LOG_LEVEL_INFO},
		{"WARNING", labv1.VMLogLevel_VM_LOG_LEVEL_WARNING},
		{"ERROR", labv1.VMLogLevel_VM_LOG_LEVEL_ERROR},
		{"CRITICAL", labv1.VMLogLevel_VM_LOG_LEVEL_CRITICAL},
		{"UNKNOWN", labv1.VMLogLevel_VM_LOG_LEVEL_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := vmLogLevelStringToProto(tt.input)
		if result != tt.expected {
			t.Errorf("vmLogLevelStringToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestResolveArchPlaceholder(t *testing.T) {
	tests := []struct {
		input    string
		arch     string
		expected string
	}{
		{"ubuntu-${arch}.iso", "x86_64", "ubuntu-amd64.iso"},
		{"ubuntu-${arch}.iso", "aarch64", "ubuntu-arm64.iso"},
		{"ubuntu-${arch}.iso", "amd64", "ubuntu-amd64.iso"},
		{"ubuntu-${arch}.iso", "arm64", "ubuntu-arm64.iso"},
		{"ubuntu.iso", "x86_64", "ubuntu.iso"},
		{"${arch}-ubuntu-${arch}.iso", "x86_64", "amd64-ubuntu-amd64.iso"},
	}

	for _, tt := range tests {
		result := resolveArchPlaceholder(tt.input, tt.arch)
		if result != tt.expected {
			t.Errorf("resolveArchPlaceholder(%q, %q): got %q, want %q", tt.input, tt.arch, result, tt.expected)
		}
	}
}

func TestModelVMToProto_WithPCIDevices(t *testing.T) {
	vm := &model.VM{
		ID:          "vm-123",
		VMID:        1,
		Name:        "test-vm-with-gpu",
		Node:        "node-1",
		Status:      model.VMStatusStopped,
		CPU:         model.CPUInfoPartial{Used: 0, Sockets: 1, Cores: 2},
		Memory:      model.MemoryInfo{Used: 0, Total: 8.0},
		Disk:        model.DiskInfo{Used: 0, Total: 100.0},
		Uptime:      "0d0h0m",
		OS:          model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-22.04"},
		Arch:        "x86_64",
		PCIDevices: []model.PCIDevice{
			{
				Address:     "0000:01:00.0",
				VendorID:    "10de",
				VendorName:  "NVIDIA Corporation",
				ProductID:   "1b80",
				ProductName: "GP104 [GeForce GTX 1080]",
				Driver:      "vfio-pci",
				IOMMUGroup:  1,
				Class:       "0300",
				ClassName:   "VGA compatible controller",
			},
		},
	}

	p := modelVMToProto(vm)

	if len(p.PciDevices) != 1 {
		t.Fatalf("PciDevices: expected 1, got %d", len(p.PciDevices))
	}

	pciDev := p.PciDevices[0]
	if pciDev.Address != "0000:01:00.0" {
		t.Errorf("PCI Address: got %q, want 0000:01:00.0", pciDev.Address)
	}
	if pciDev.VendorId != "10de" {
		t.Errorf("VendorId: got %q, want 10de", pciDev.VendorId)
	}
	if pciDev.VendorName != "NVIDIA Corporation" {
		t.Errorf("VendorName: got %q, want NVIDIA Corporation", pciDev.VendorName)
	}
	if pciDev.ProductId != "1b80" {
		t.Errorf("ProductId: got %q, want 1b80", pciDev.ProductId)
	}
	if pciDev.ProductName != "GP104 [GeForce GTX 1080]" {
		t.Errorf("ProductName: got %q", pciDev.ProductName)
	}
	if pciDev.Driver != "vfio-pci" {
		t.Errorf("Driver: got %q, want vfio-pci", pciDev.Driver)
	}
	if pciDev.IommuGroup != 1 {
		t.Errorf("IommuGroup: got %d, want 1", pciDev.IommuGroup)
	}
	if pciDev.Class != "0300" {
		t.Errorf("Class: got %q, want 0300", pciDev.Class)
	}
	if pciDev.ClassName != "VGA compatible controller" {
		t.Errorf("ClassName: got %q, want VGA compatible controller", pciDev.ClassName)
	}
}

func TestProtoCreateVMRequestToModel_WithPCIDevices(t *testing.T) {
	req := &labv1.CreateVMRequest{
		Name:              "test-vm-with-gpu",
		Node:              "node-1",
		CpuCores:          2,
		MemoryGb:          4.0,
		DiskGb:            50.0,
		PciDeviceAddresses: []string{"0000:01:00.0", "0000:01:00.1"},
	}

	result := protoCreateVMRequestToModel(req)

	if len(result.PCIDevices) != 2 {
		t.Fatalf("PCIDevices: expected 2, got %d", len(result.PCIDevices))
	}

	if result.PCIDevices[0].Address != "0000:01:00.0" {
		t.Errorf("PCIDevices[0].Address: got %q, want 0000:01:00.0", result.PCIDevices[0].Address)
	}
	if result.PCIDevices[1].Address != "0000:01:00.1" {
		t.Errorf("PCIDevices[1].Address: got %q, want 0000:01:00.1", result.PCIDevices[1].Address)
	}
}
