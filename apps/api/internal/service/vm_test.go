package service

import (
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/sysinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyVMDefaults_ZeroValues(t *testing.T) {
	sys := sysinfo.New()
	req := &model.VMCreateRequest{}
	applyVMDefaults(req)

	expectedArch := sys.HostArch()
	assert.Equal(t, expectedArch, req.Arch)
	assert.Equal(t, 1, req.CPUSockets)
	assert.Equal(t, 1, req.CPUCores)
	
	expectedCPUModel := sys.DefaultCPUModel(expectedArch)
	assert.Equal(t, expectedCPUModel, req.CPUModel)
	require.Len(t, req.Network, 1)

	expectedType := model.NetworkType(sys.DefaultNetworkType())
	expectedBridge := sys.DefaultBridgeName()
	assert.Equal(t, expectedType, req.Network[0].Type)
	
	if expectedBridge != "" {
		assert.Equal(t, expectedBridge, req.Network[0].Bridge)
	}
	assert.Equal(t, model.NetworkModelVirtio, req.Network[0].Model)
	assert.Equal(t, model.OSTypeOther, req.OS.Type)
}

func TestApplyVMDefaults_AArch64SmartDefaults(t *testing.T) {
	req := &model.VMCreateRequest{
		Arch: "aarch64",
		OS:   model.OSConfig{Type: model.OSTypeLinux, Version: "alpine"},
	}
	applyVMDefaults(req)

	assert.Equal(t, model.MachineTypeVirt, req.MachineType)
	assert.Equal(t, model.BIOSTypeOVMF, req.BIOS)
	assert.Equal(t, "maximum", req.CPUModel)
}

func TestApplyVMDefaults_WindowsSmartDefaults(t *testing.T) {
	req := &model.VMCreateRequest{
		Arch: "x86_64",
		OS:   model.OSConfig{Type: model.OSTypeWindows, Version: "11"},
	}
	applyVMDefaults(req)

	assert.Equal(t, model.MachineTypeQ35, req.MachineType)
	assert.Equal(t, model.BIOSTypeOVMF, req.BIOS)
}

func TestApplyVMDefaults_LinuxSmartDefaults(t *testing.T) {
	req := &model.VMCreateRequest{
		Arch: "x86_64",
		OS:   model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-24.04"},
	}
	applyVMDefaults(req)

	assert.Equal(t, model.MachineTypePC, req.MachineType)
	assert.Equal(t, model.BIOSTypeSeaBIOS, req.BIOS)
}

func TestApplyVMDefaults_ExplicitOverridesPreserved(t *testing.T) {
	req := &model.VMCreateRequest{
		OS:          model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-24.04"},
		MachineType: model.MachineTypeQ35,
		BIOS:        model.BIOSTypeOVMF,
		CPUSockets:  2,
		CPUCores:    4,
		CPUModel:    "kvm64",
		Network: []model.NetworkConfig{
			{Type: model.NetworkTypeBridge, Bridge: "br0", Model: model.NetworkModelE1000},
		},
	}
	applyVMDefaults(req)

	assert.Equal(t, model.MachineTypeQ35, req.MachineType)
	assert.Equal(t, model.BIOSTypeOVMF, req.BIOS)
	assert.Equal(t, 2, req.CPUSockets)
	assert.Equal(t, 4, req.CPUCores)
	assert.Equal(t, "kvm64", req.CPUModel)
	require.Len(t, req.Network, 1)
	assert.Equal(t, "br0", req.Network[0].Bridge)
}

func TestApplyVMDefaults_NetworkNotOverwritten(t *testing.T) {
	req := &model.VMCreateRequest{
		Network: []model.NetworkConfig{
			{Type: model.NetworkTypeBridge, Bridge: "vmbr0", Model: model.NetworkModelVirtio},
			{Type: model.NetworkTypeUser, Model: model.NetworkModelE1000},
		},
	}
	applyVMDefaults(req)

	assert.Len(t, req.Network, 2)
}
