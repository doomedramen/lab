package connectsvc

import (
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelContainerStatusToProto(t *testing.T) {
	tests := []struct {
		input    model.ContainerStatus
		expected labv1.ContainerStatus
	}{
		{model.ContainerStatusRunning, labv1.ContainerStatus_CONTAINER_STATUS_RUNNING},
		{model.ContainerStatusStopped, labv1.ContainerStatus_CONTAINER_STATUS_STOPPED},
		{model.ContainerStatusFrozen, labv1.ContainerStatus_CONTAINER_STATUS_FROZEN},
		{model.ContainerStatus("unknown"), labv1.ContainerStatus_CONTAINER_STATUS_UNSPECIFIED},
	}

	for _, tt := range tests {
		result := modelContainerStatusToProto(tt.input)
		if result != tt.expected {
			t.Errorf("modelContainerStatusToProto(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestModelContainerToProto(t *testing.T) {
	c := &model.Container{
		ID:           "ct-1",
		CTID:         200,
		Name:         "web-server",
		Node:         "pve1",
		Status:       model.ContainerStatusRunning,
		CPU:          model.CPUInfoPartial{Used: 25.0, Sockets: 1, Cores: 2},
		Memory:       model.MemoryInfo{Used: 1.0, Total: 4.0},
		Disk:         model.DiskInfo{Used: 5.0, Total: 20.0},
		Uptime:       "2d 5h",
		OS:           "debian-12",
		IP:           "10.0.0.50",
		Tags:         []string{"web", "prod"},
		Unprivileged: true,
		Swap:         model.SwapInfo{Used: 0.1, Total: 1.0},
		Description:  "Production web server",
		StartOnBoot:  true,
	}

	proto := modelContainerToProto(c)

	if proto.Id != "ct-1" {
		t.Errorf("Id = %q, want ct-1", proto.Id)
	}
	if proto.Ctid != 200 {
		t.Errorf("Ctid = %d, want 200", proto.Ctid)
	}
	if proto.Name != "web-server" {
		t.Errorf("Name = %q, want web-server", proto.Name)
	}
	if proto.Status != labv1.ContainerStatus_CONTAINER_STATUS_RUNNING {
		t.Errorf("Status = %v, want RUNNING", proto.Status)
	}
	if proto.Cpu.Cores != 2 {
		t.Errorf("Cpu.Cores = %d, want 2", proto.Cpu.Cores)
	}
	if proto.Memory.Total != 4.0 {
		t.Errorf("Memory.Total = %v, want 4.0", proto.Memory.Total)
	}
	if proto.Os != "debian-12" {
		t.Errorf("Os = %q, want debian-12", proto.Os)
	}
	if !proto.Unprivileged {
		t.Error("expected Unprivileged=true")
	}
	if !proto.StartOnBoot {
		t.Error("expected StartOnBoot=true")
	}
	if len(proto.Tags) != 2 {
		t.Errorf("Tags count = %d, want 2", len(proto.Tags))
	}
	if proto.Description != "Production web server" {
		t.Errorf("Description = %q, want 'Production web server'", proto.Description)
	}
}
