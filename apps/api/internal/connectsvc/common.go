package connectsvc

import (
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- shared model → proto converters used across multiple service files ---

func modelMemoryInfoToProto(m model.MemoryInfo) *labv1.MemoryInfo {
	return &labv1.MemoryInfo{Used: m.Used, Total: m.Total}
}

func modelDiskInfoToProto(d model.DiskInfo) *labv1.DiskInfo {
	return &labv1.DiskInfo{Used: d.Used, Total: d.Total}
}

func modelSwapInfoToProto(s model.SwapInfo) *labv1.SwapInfo {
	return &labv1.SwapInfo{Used: s.Used, Total: s.Total}
}

func modelCPUInfoPartialToProto(c model.CPUInfoPartial) *labv1.CpuInfoPartial {
	return &labv1.CpuInfoPartial{
		Used:    c.Used,
		Sockets: int32(c.Sockets),
		Cores:   int32(c.Cores),
	}
}

func modelCPUInfoToProto(c model.CPUInfo) *labv1.CpuInfo {
	return &labv1.CpuInfo{
		Used:  c.Used,
		Total: c.Total,
		Cores: int32(c.Cores),
	}
}

func modelLoadAvgToProto(l model.LoadAvg) *labv1.LoadAvg {
	return &labv1.LoadAvg{One: l[0], Five: l[1], Fifteen: l[2]}
}

func modelOSTypeToProto(t model.OSType) labv1.OsType {
	switch t {
	case model.OSTypeLinux:
		return labv1.OsType_OS_TYPE_LINUX
	case model.OSTypeWindows:
		return labv1.OsType_OS_TYPE_WINDOWS
	case model.OSTypeSolaris:
		return labv1.OsType_OS_TYPE_SOLARIS
	case model.OSTypeOther:
		return labv1.OsType_OS_TYPE_OTHER
	default:
		return labv1.OsType_OS_TYPE_UNSPECIFIED
	}
}

func protoOSTypeToModel(t labv1.OsType) model.OSType {
	switch t {
	case labv1.OsType_OS_TYPE_LINUX:
		return model.OSTypeLinux
	case labv1.OsType_OS_TYPE_WINDOWS:
		return model.OSTypeWindows
	case labv1.OsType_OS_TYPE_SOLARIS:
		return model.OSTypeSolaris
	case labv1.OsType_OS_TYPE_OTHER:
		return model.OSTypeOther
	default:
		return model.OSTypeOther
	}
}

func modelOSConfigToProto(o model.OSConfig) *labv1.OsConfig {
	return &labv1.OsConfig{
		OsType:  modelOSTypeToProto(o.Type),
		Version: o.Version,
	}
}

func protoOSConfigToModel(o *labv1.OsConfig) model.OSConfig {
	if o == nil {
		return model.OSConfig{Type: model.OSTypeOther}
	}
	return model.OSConfig{
		Type:    protoOSTypeToModel(o.OsType),
		Version: o.Version,
	}
}

func modelNetworkTypeToProto(t model.NetworkType) labv1.NetworkType {
	switch t {
	case model.NetworkTypeUser:
		return labv1.NetworkType_NETWORK_TYPE_USER
	case model.NetworkTypeBridge:
		return labv1.NetworkType_NETWORK_TYPE_BRIDGE
	default:
		return labv1.NetworkType_NETWORK_TYPE_UNSPECIFIED
	}
}

func protoNetworkTypeToModel(t labv1.NetworkType) model.NetworkType {
	switch t {
	case labv1.NetworkType_NETWORK_TYPE_BRIDGE:
		return model.NetworkTypeBridge
	default:
		return model.NetworkTypeUser
	}
}

func modelNetworkModelToProto(m model.NetworkModel) labv1.NetworkModel {
	switch m {
	case model.NetworkModelVirtio:
		return labv1.NetworkModel_NETWORK_MODEL_VIRTIO
	case model.NetworkModelE1000:
		return labv1.NetworkModel_NETWORK_MODEL_E1000
	case model.NetworkModelRTL8139:
		return labv1.NetworkModel_NETWORK_MODEL_RTL8139
	default:
		return labv1.NetworkModel_NETWORK_MODEL_UNSPECIFIED
	}
}

func protoNetworkModelToModel(m labv1.NetworkModel) model.NetworkModel {
	switch m {
	case labv1.NetworkModel_NETWORK_MODEL_E1000:
		return model.NetworkModelE1000
	case labv1.NetworkModel_NETWORK_MODEL_RTL8139:
		return model.NetworkModelRTL8139
	default:
		return model.NetworkModelVirtio
	}
}

func modelNetworkConfigsToProto(nets []model.NetworkConfig) []*labv1.NetworkConfig {
	out := make([]*labv1.NetworkConfig, len(nets))
	for i, n := range nets {
		out[i] = &labv1.NetworkConfig{
			Type:         modelNetworkTypeToProto(n.Type),
			Bridge:       n.Bridge,
			Model:        modelNetworkModelToProto(n.Model),
			Vlan:         int32(n.VLAN),
			PortForwards: n.PortForwards,
		}
	}
	return out
}

func protoNetworkConfigsToModel(nets []*labv1.NetworkConfig) []model.NetworkConfig {
	out := make([]model.NetworkConfig, len(nets))
	for i, n := range nets {
		out[i] = model.NetworkConfig{
			Type:         protoNetworkTypeToModel(n.Type),
			Bridge:       n.Bridge,
			Model:        protoNetworkModelToModel(n.Model),
			VLAN:         int(n.Vlan),
			PortForwards: n.PortForwards,
		}
	}
	return out
}

func modelMachineTypeToProto(t model.MachineType) labv1.MachineType {
	switch t {
	case model.MachineTypePC:
		return labv1.MachineType_MACHINE_TYPE_PC
	case model.MachineTypeQ35:
		return labv1.MachineType_MACHINE_TYPE_Q35
	case model.MachineTypeVirt:
		return labv1.MachineType_MACHINE_TYPE_VIRT
	default:
		return labv1.MachineType_MACHINE_TYPE_UNSPECIFIED
	}
}

func protoMachineTypeToModel(t labv1.MachineType) model.MachineType {
	switch t {
	case labv1.MachineType_MACHINE_TYPE_Q35:
		return model.MachineTypeQ35
	case labv1.MachineType_MACHINE_TYPE_VIRT:
		return model.MachineTypeVirt
	case labv1.MachineType_MACHINE_TYPE_PC:
		return model.MachineTypePC
	default:
		return "" // empty → service layer will apply default
	}
}

func modelBIOSTypeToProto(t model.BIOSType) labv1.BiosType {
	switch t {
	case model.BIOSTypeSeaBIOS:
		return labv1.BiosType_BIOS_TYPE_SEABIOS
	case model.BIOSTypeOVMF:
		return labv1.BiosType_BIOS_TYPE_OVMF
	default:
		return labv1.BiosType_BIOS_TYPE_UNSPECIFIED
	}
}

func protoBIOSTypeToModel(t labv1.BiosType) model.BIOSType {
	switch t {
	case labv1.BiosType_BIOS_TYPE_OVMF:
		return model.BIOSTypeOVMF
	case labv1.BiosType_BIOS_TYPE_SEABIOS:
		return model.BIOSTypeSeaBIOS
	default:
		return "" // empty → service layer will apply default
	}
}
