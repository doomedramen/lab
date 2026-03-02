package connectsvc

import (
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelMemoryInfoToProto(t *testing.T) {
	m := model.MemoryInfo{Used: 4.5, Total: 16.0}
	p := modelMemoryInfoToProto(m)

	if p.Used != m.Used {
		t.Errorf("Used: got %v, want %v", p.Used, m.Used)
	}
	if p.Total != m.Total {
		t.Errorf("Total: got %v, want %v", p.Total, m.Total)
	}
}

func TestModelDiskInfoToProto(t *testing.T) {
	d := model.DiskInfo{Used: 100.0, Total: 500.0}
	p := modelDiskInfoToProto(d)

	if p.Used != d.Used {
		t.Errorf("Used: got %v, want %v", p.Used, d.Used)
	}
	if p.Total != d.Total {
		t.Errorf("Total: got %v, want %v", p.Total, d.Total)
	}
}

func TestModelSwapInfoToProto(t *testing.T) {
	s := model.SwapInfo{Used: 2.0, Total: 8.0}
	p := modelSwapInfoToProto(s)

	if p.Used != s.Used {
		t.Errorf("Used: got %v, want %v", p.Used, s.Used)
	}
	if p.Total != s.Total {
		t.Errorf("Total: got %v, want %v", p.Total, s.Total)
	}
}

func TestModelCPUInfoPartialToProto(t *testing.T) {
	c := model.CPUInfoPartial{Used: 50.0, Sockets: 1, Cores: 4}
	p := modelCPUInfoPartialToProto(c)

	if p.Used != c.Used {
		t.Errorf("Used: got %v, want %v", p.Used, c.Used)
	}
	if p.Sockets != int32(c.Sockets) {
		t.Errorf("Sockets: got %v, want %v", p.Sockets, c.Sockets)
	}
	if p.Cores != int32(c.Cores) {
		t.Errorf("Cores: got %v, want %v", p.Cores, c.Cores)
	}
}

func TestModelCPUInfoToProto(t *testing.T) {
	c := model.CPUInfo{Used: 50.0, Total: 100.0, Cores: 8}
	p := modelCPUInfoToProto(c)

	if p.Used != c.Used {
		t.Errorf("Used: got %v, want %v", p.Used, c.Used)
	}
	if p.Total != c.Total {
		t.Errorf("Total: got %v, want %v", p.Total, c.Total)
	}
	if p.Cores != int32(c.Cores) {
		t.Errorf("Cores: got %v, want %v", p.Cores, c.Cores)
	}
}

func TestModelLoadAvgToProto(t *testing.T) {
	l := model.LoadAvg{1.0, 0.8, 0.5}
	p := modelLoadAvgToProto(l)

	if p.One != l[0] {
		t.Errorf("One: got %v, want %v", p.One, l[0])
	}
	if p.Five != l[1] {
		t.Errorf("Five: got %v, want %v", p.Five, l[1])
	}
	if p.Fifteen != l[2] {
		t.Errorf("Fifteen: got %v, want %v", p.Fifteen, l[2])
	}
}

func TestModelOSTypeToProto(t *testing.T) {
	tests := []struct {
		input    model.OSType
		expected labv1.OsType
	}{
		{model.OSTypeLinux, labv1.OsType_OS_TYPE_LINUX},
		{model.OSTypeWindows, labv1.OsType_OS_TYPE_WINDOWS},
		{model.OSTypeSolaris, labv1.OsType_OS_TYPE_SOLARIS},
		{model.OSTypeOther, labv1.OsType_OS_TYPE_OTHER},
		{model.OSType("unknown"), labv1.OsType_OS_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelOSTypeToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelOSTypeToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestProtoOSTypeToModel(t *testing.T) {
	tests := []struct {
		input    labv1.OsType
		expected model.OSType
	}{
		{labv1.OsType_OS_TYPE_LINUX, model.OSTypeLinux},
		{labv1.OsType_OS_TYPE_WINDOWS, model.OSTypeWindows},
		{labv1.OsType_OS_TYPE_SOLARIS, model.OSTypeSolaris},
		{labv1.OsType_OS_TYPE_OTHER, model.OSTypeOther},
		{labv1.OsType_OS_TYPE_UNSPECIFIED, model.OSTypeOther},
	}

	for _, tt := range tests {
		result := protoOSTypeToModel(tt.input)
		if result != tt.expected {
			t.Errorf("protoOSTypeToModel(%v): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelOSConfigToProto(t *testing.T) {
	o := model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-24.04"}
	p := modelOSConfigToProto(o)

	if p.OsType != labv1.OsType_OS_TYPE_LINUX {
		t.Errorf("OsType: got %v, want OS_TYPE_LINUX", p.OsType)
	}
	if p.Version != "ubuntu-24.04" {
		t.Errorf("Version: got %q, want ubuntu-24.04", p.Version)
	}
}

func TestProtoOSConfigToModel(t *testing.T) {
	p := &labv1.OsConfig{OsType: labv1.OsType_OS_TYPE_WINDOWS, Version: "11"}
	m := protoOSConfigToModel(p)

	if m.Type != model.OSTypeWindows {
		t.Errorf("Type: got %v, want Windows", m.Type)
	}
	if m.Version != "11" {
		t.Errorf("Version: got %q, want 11", m.Version)
	}
}

func TestProtoOSConfigToModel_Nil(t *testing.T) {
	m := protoOSConfigToModel(nil)

	if m.Type != model.OSTypeOther {
		t.Errorf("Type: got %v, want Other", m.Type)
	}
}

func TestModelNetworkTypeToProto(t *testing.T) {
	tests := []struct {
		input    model.NetworkType
		expected labv1.NetworkType
	}{
		{model.NetworkTypeUser, labv1.NetworkType_NETWORK_TYPE_USER},
		{model.NetworkTypeBridge, labv1.NetworkType_NETWORK_TYPE_BRIDGE},
		{model.NetworkType("unknown"), labv1.NetworkType_NETWORK_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelNetworkTypeToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelNetworkTypeToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestProtoNetworkTypeToModel(t *testing.T) {
	tests := []struct {
		input    labv1.NetworkType
		expected model.NetworkType
	}{
		{labv1.NetworkType_NETWORK_TYPE_BRIDGE, model.NetworkTypeBridge},
		{labv1.NetworkType_NETWORK_TYPE_USER, model.NetworkTypeUser},
		{labv1.NetworkType_NETWORK_TYPE_UNSPECIFIED, model.NetworkTypeUser},
	}

	for _, tt := range tests {
		result := protoNetworkTypeToModel(tt.input)
		if result != tt.expected {
			t.Errorf("protoNetworkTypeToModel(%v): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelNetworkModelToProto(t *testing.T) {
	tests := []struct {
		input    model.NetworkModel
		expected labv1.NetworkModel
	}{
		{model.NetworkModelVirtio, labv1.NetworkModel_NETWORK_MODEL_VIRTIO},
		{model.NetworkModelE1000, labv1.NetworkModel_NETWORK_MODEL_E1000},
		{model.NetworkModelRTL8139, labv1.NetworkModel_NETWORK_MODEL_RTL8139},
		{model.NetworkModel("unknown"), labv1.NetworkModel_NETWORK_MODEL_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelNetworkModelToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelNetworkModelToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestProtoNetworkModelToModel(t *testing.T) {
	tests := []struct {
		input    labv1.NetworkModel
		expected model.NetworkModel
	}{
		{labv1.NetworkModel_NETWORK_MODEL_E1000, model.NetworkModelE1000},
		{labv1.NetworkModel_NETWORK_MODEL_RTL8139, model.NetworkModelRTL8139},
		{labv1.NetworkModel_NETWORK_MODEL_VIRTIO, model.NetworkModelVirtio},
		{labv1.NetworkModel_NETWORK_MODEL_UNSPECIFIED, model.NetworkModelVirtio},
	}

	for _, tt := range tests {
		result := protoNetworkModelToModel(tt.input)
		if result != tt.expected {
			t.Errorf("protoNetworkModelToModel(%v): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelNetworkConfigsToProto(t *testing.T) {
	nets := []model.NetworkConfig{
		{Type: model.NetworkTypeBridge, Bridge: "br0", Model: model.NetworkModelVirtio, VLAN: 10, PortForwards: []string{"8080:80"}},
		{Type: model.NetworkTypeUser, Model: model.NetworkModelE1000},
	}

	result := modelNetworkConfigsToProto(nets)

	if len(result) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(result))
	}

	if result[0].Bridge != "br0" {
		t.Errorf("Bridge: got %q, want br0", result[0].Bridge)
	}
	if result[0].Vlan != 10 {
		t.Errorf("VLAN: got %d, want 10", result[0].Vlan)
	}
	if len(result[0].PortForwards) != 1 {
		t.Errorf("PortForwards: expected 1, got %d", len(result[0].PortForwards))
	}
}

func TestProtoNetworkConfigsToModel(t *testing.T) {
	nets := []*labv1.NetworkConfig{
		{Type: labv1.NetworkType_NETWORK_TYPE_BRIDGE, Bridge: "virbr0", Model: labv1.NetworkModel_NETWORK_MODEL_VIRTIO, Vlan: 100},
	}

	result := protoNetworkConfigsToModel(nets)

	if len(result) != 1 {
		t.Fatalf("expected 1 config, got %d", len(result))
	}

	if result[0].Bridge != "virbr0" {
		t.Errorf("Bridge: got %q, want virbr0", result[0].Bridge)
	}
	if result[0].VLAN != 100 {
		t.Errorf("VLAN: got %d, want 100", result[0].VLAN)
	}
}

func TestModelMachineTypeToProto(t *testing.T) {
	tests := []struct {
		input    model.MachineType
		expected labv1.MachineType
	}{
		{model.MachineTypePC, labv1.MachineType_MACHINE_TYPE_PC},
		{model.MachineTypeQ35, labv1.MachineType_MACHINE_TYPE_Q35},
		{model.MachineTypeVirt, labv1.MachineType_MACHINE_TYPE_VIRT},
		{model.MachineType("unknown"), labv1.MachineType_MACHINE_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelMachineTypeToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelMachineTypeToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestProtoMachineTypeToModel(t *testing.T) {
	tests := []struct {
		input    labv1.MachineType
		expected model.MachineType
	}{
		{labv1.MachineType_MACHINE_TYPE_PC, model.MachineTypePC},
		{labv1.MachineType_MACHINE_TYPE_Q35, model.MachineTypeQ35},
		{labv1.MachineType_MACHINE_TYPE_VIRT, model.MachineTypeVirt},
		{labv1.MachineType_MACHINE_TYPE_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		result := protoMachineTypeToModel(tt.input)
		if result != tt.expected {
			t.Errorf("protoMachineTypeToModel(%v): got %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestModelBIOSTypeToProto(t *testing.T) {
	tests := []struct {
		input    model.BIOSType
		expected labv1.BiosType
	}{
		{model.BIOSTypeSeaBIOS, labv1.BiosType_BIOS_TYPE_SEABIOS},
		{model.BIOSTypeOVMF, labv1.BiosType_BIOS_TYPE_OVMF},
		{model.BIOSType("unknown"), labv1.BiosType_BIOS_TYPE_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelBIOSTypeToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelBIOSTypeToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestProtoBIOSTypeToModel(t *testing.T) {
	tests := []struct {
		input    labv1.BiosType
		expected model.BIOSType
	}{
		{labv1.BiosType_BIOS_TYPE_SEABIOS, model.BIOSTypeSeaBIOS},
		{labv1.BiosType_BIOS_TYPE_OVMF, model.BIOSTypeOVMF},
		{labv1.BiosType_BIOS_TYPE_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		result := protoBIOSTypeToModel(tt.input)
		if result != tt.expected {
			t.Errorf("protoBIOSTypeToModel(%v): got %q, want %q", tt.input, result, tt.expected)
		}
	}
}
