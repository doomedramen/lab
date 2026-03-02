package connectsvc

import (
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelNodeStatusToProto(t *testing.T) {
	tests := []struct {
		input    model.NodeStatus
		expected labv1.NodeStatus
	}{
		{model.NodeStatusOnline, labv1.NodeStatus_NODE_STATUS_ONLINE},
		{model.NodeStatusOffline, labv1.NodeStatus_NODE_STATUS_OFFLINE},
		{model.NodeStatusMaintenance, labv1.NodeStatus_NODE_STATUS_MAINTENANCE},
		{model.NodeStatus("unknown"), labv1.NodeStatus_NODE_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelNodeStatusToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelNodeStatusToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelNodeToProto(t *testing.T) {
	node := &model.HostNode{
		ID:         "n1",
		Name:       "pve-node-1",
		Status:     model.NodeStatusOnline,
		IP:         "10.0.0.1",
		CPU:        model.CPUInfo{Used: 50.0, Total: 100.0, Cores: 8},
		Memory:     model.MemoryInfo{Used: 16.0, Total: 64.0},
		Disk:       model.DiskInfo{Used: 200.0, Total: 1000.0},
		Uptime:     "1d 0h",
		Kernel:     "6.1.0-26",
		Version:    "8.3.0",
		VMs:        5,
		Containers: 3,
		CPUModel:   "AMD EPYC 7763",
		LoadAvg:    model.LoadAvg{1.5, 0.8, 0.4},
		NetworkIn:  1024000,
		NetworkOut: 512000,
		Arch:       "x86_64",
	}

	proto := modelNodeToProto(node)

	if proto.Id != "n1" {
		t.Errorf("Id = %q, want n1", proto.Id)
	}
	if proto.Name != "pve-node-1" {
		t.Errorf("Name = %q, want pve-node-1", proto.Name)
	}
	if proto.Status != labv1.NodeStatus_NODE_STATUS_ONLINE {
		t.Errorf("Status = %v, want ONLINE", proto.Status)
	}
	if proto.Ip != "10.0.0.1" {
		t.Errorf("Ip = %q, want 10.0.0.1", proto.Ip)
	}
	if proto.Cpu.Cores != 8 {
		t.Errorf("Cpu.Cores = %d, want 8", proto.Cpu.Cores)
	}
	if proto.Memory.Total != 64.0 {
		t.Errorf("Memory.Total = %v, want 64.0", proto.Memory.Total)
	}
	if proto.Disk.Used != 200.0 {
		t.Errorf("Disk.Used = %v, want 200.0", proto.Disk.Used)
	}
	if proto.Vms != 5 {
		t.Errorf("Vms = %d, want 5", proto.Vms)
	}
	if proto.Containers != 3 {
		t.Errorf("Containers = %d, want 3", proto.Containers)
	}
	if proto.CpuModel != "AMD EPYC 7763" {
		t.Errorf("CpuModel = %q, want AMD EPYC 7763", proto.CpuModel)
	}
	if proto.LoadAvg.One != 1.5 {
		t.Errorf("LoadAvg.One = %v, want 1.5", proto.LoadAvg.One)
	}
	if proto.Arch != "x86_64" {
		t.Errorf("Arch = %q, want x86_64", proto.Arch)
	}
}
