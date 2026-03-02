package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- mock VM repository for cluster tests ---

type mockVMRepo struct {
	vms []*model.VM
}

func (m *mockVMRepo) GetAll(_ context.Context) ([]*model.VM, error) { return m.vms, nil }
func (m *mockVMRepo) GetByNode(_ context.Context, _ string) ([]*model.VM, error) {
	return nil, nil
}
func (m *mockVMRepo) GetByID(_ context.Context, _ string) (*model.VM, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockVMRepo) GetByVMID(_ context.Context, _ int) (*model.VM, error) {
	return nil, fmt.Errorf("not found")
}
func (m *mockVMRepo) Create(_ context.Context, _ *model.VMCreateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockVMRepo) Update(_ context.Context, _ int, _ *model.VMUpdateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockVMRepo) Delete(_ context.Context, _ int) error          { return fmt.Errorf("not implemented") }
func (m *mockVMRepo) Clone(_ context.Context, _ *model.VMCloneRequest, _ func(int, string)) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockVMRepo) Start(_ context.Context, _ int) error           { return nil }
func (m *mockVMRepo) Stop(_ context.Context, _ int) error            { return nil }
func (m *mockVMRepo) Shutdown(_ context.Context, _ int) error        { return nil }
func (m *mockVMRepo) Pause(_ context.Context, _ int) error           { return nil }
func (m *mockVMRepo) Resume(_ context.Context, _ int) error          { return nil }
func (m *mockVMRepo) Reboot(_ context.Context, _ int) error          { return nil }
func (m *mockVMRepo) GetVNCPort(_ context.Context, _ int) (int, error) { return 0, nil }

// --- tests ---

func TestClusterService_GetSummary_Empty(t *testing.T) {
	svc := NewClusterService(
		newMockNodeRepo(),
		&mockVMRepo{},
		newMockContainerRepo(),
		newMockStackRepo(),
		nil,
	)

	summary := svc.GetSummary(context.Background())

	if summary.Nodes.Total != 0 {
		t.Errorf("Nodes.Total = %d, want 0", summary.Nodes.Total)
	}
	if summary.VMs.Total != 0 {
		t.Errorf("VMs.Total = %d, want 0", summary.VMs.Total)
	}
	if summary.Containers.Total != 0 {
		t.Errorf("Containers.Total = %d, want 0", summary.Containers.Total)
	}
	if summary.Stacks.Total != 0 {
		t.Errorf("Stacks.Total = %d, want 0", summary.Stacks.Total)
	}
	if summary.CPU.AvgUsage != 0 {
		t.Errorf("CPU.AvgUsage = %v, want 0", summary.CPU.AvgUsage)
	}
}

func TestClusterService_GetSummary_WithData(t *testing.T) {
	nodeRepo := newMockNodeRepo()
	nodeRepo.addNode(&model.HostNode{
		ID:     "n1",
		Status: model.NodeStatusOnline,
		CPU:    model.CPUInfo{Cores: 8, Used: 50.0},
		Memory: model.MemoryInfo{Used: 8.0, Total: 32.0},
		Disk:   model.DiskInfo{Used: 100.0, Total: 500.0},
	})
	nodeRepo.addNode(&model.HostNode{
		ID:     "n2",
		Status: model.NodeStatusOffline,
		CPU:    model.CPUInfo{Cores: 4, Used: 0},
		Memory: model.MemoryInfo{Used: 0, Total: 16.0},
		Disk:   model.DiskInfo{Used: 0, Total: 250.0},
	})

	vmRepo := &mockVMRepo{
		vms: []*model.VM{
			{VMID: 100, Status: model.VMStatusRunning},
			{VMID: 101, Status: model.VMStatusStopped},
			{VMID: 102, Status: model.VMStatusRunning},
		},
	}

	containerRepo := newMockContainerRepo()
	containerRepo.addContainer(&model.Container{CTID: 200, Status: model.ContainerStatusRunning})
	containerRepo.addContainer(&model.Container{CTID: 201, Status: model.ContainerStatusStopped})

	stackRepo := newMockStackRepo()

	svc := NewClusterService(nodeRepo, vmRepo, containerRepo, stackRepo, nil)
	summary := svc.GetSummary(context.Background())

	// Nodes
	if summary.Nodes.Total != 2 {
		t.Errorf("Nodes.Total = %d, want 2", summary.Nodes.Total)
	}
	if summary.Nodes.Running != 1 {
		t.Errorf("Nodes.Running = %d, want 1 (online)", summary.Nodes.Running)
	}

	// VMs
	if summary.VMs.Total != 3 {
		t.Errorf("VMs.Total = %d, want 3", summary.VMs.Total)
	}
	if summary.VMs.Running != 2 {
		t.Errorf("VMs.Running = %d, want 2", summary.VMs.Running)
	}

	// Containers
	if summary.Containers.Total != 2 {
		t.Errorf("Containers.Total = %d, want 2", summary.Containers.Total)
	}
	if summary.Containers.Running != 1 {
		t.Errorf("Containers.Running = %d, want 1", summary.Containers.Running)
	}

	// CPU — only online nodes contribute
	if summary.CPU.Cores != 12 {
		t.Errorf("CPU.Cores = %d, want 12 (8+4)", summary.CPU.Cores)
	}
	if summary.CPU.AvgUsage != 50 {
		t.Errorf("CPU.AvgUsage = %v, want 50 (only 1 online node at 50%%)", summary.CPU.AvgUsage)
	}

	// Memory — only online nodes contribute to used
	if summary.Memory.Total != 48 {
		t.Errorf("Memory.Total = %v, want 48 (32+16)", summary.Memory.Total)
	}
	if summary.Memory.Used != 8.0 {
		t.Errorf("Memory.Used = %v, want 8.0 (only online node)", summary.Memory.Used)
	}

	// Disk
	if summary.Disk.Total != 750 {
		t.Errorf("Disk.Total = %v, want 750 (500+250)", summary.Disk.Total)
	}
	if summary.Disk.Used != 100.0 {
		t.Errorf("Disk.Used = %v, want 100.0 (only online node)", summary.Disk.Used)
	}
}

func TestClusterService_GetSummary_AllOffline(t *testing.T) {
	nodeRepo := newMockNodeRepo()
	nodeRepo.addNode(&model.HostNode{
		ID:     "n1",
		Status: model.NodeStatusOffline,
		CPU:    model.CPUInfo{Cores: 8, Used: 0},
	})

	svc := NewClusterService(nodeRepo, &mockVMRepo{}, newMockContainerRepo(), newMockStackRepo(), nil)
	summary := svc.GetSummary(context.Background())

	if summary.CPU.AvgUsage != 0 {
		t.Errorf("CPU.AvgUsage = %v, want 0 (no online nodes)", summary.CPU.AvgUsage)
	}
}

func TestClusterService_GetMetrics_DefaultPoints(t *testing.T) {
	svc := NewClusterService(newMockNodeRepo(), &mockVMRepo{}, newMockContainerRepo(), newMockStackRepo(), nil)

	metrics := svc.GetMetrics(0)
	if len(metrics.CPUUsage) != 24 {
		t.Errorf("CPUUsage points = %d, want 24 (default)", len(metrics.CPUUsage))
	}
}

func TestClusterService_GetMetrics_CustomPoints(t *testing.T) {
	svc := NewClusterService(newMockNodeRepo(), &mockVMRepo{}, newMockContainerRepo(), newMockStackRepo(), nil)

	metrics := svc.GetMetrics(10)
	if len(metrics.CPUUsage) != 10 {
		t.Errorf("CPUUsage points = %d, want 10", len(metrics.CPUUsage))
	}
	if len(metrics.MemoryUsage) != 10 {
		t.Errorf("MemoryUsage points = %d, want 10", len(metrics.MemoryUsage))
	}
	if len(metrics.NetworkIn) != 10 {
		t.Errorf("NetworkIn points = %d, want 10", len(metrics.NetworkIn))
	}
	if len(metrics.NetworkOut) != 10 {
		t.Errorf("NetworkOut points = %d, want 10", len(metrics.NetworkOut))
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		val    float64
		places int
		want   float64
	}{
		{1.234, 2, 1.23},
		{1.235, 2, 1.24},
		{1.5, 0, 2},
		{1.4, 0, 1},
		{100.0, 1, 100.0},
		{0, 3, 0},
	}

	for _, tt := range tests {
		got := roundTo(tt.val, tt.places)
		if got != tt.want {
			t.Errorf("roundTo(%v, %d) = %v, want %v", tt.val, tt.places, got, tt.want)
		}
	}
}

func TestFormatHour(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "00:00"},
		{1, "01:00"},
		{12, "12:00"},
		{23, "23:00"},
		{24, "00:00"},
		{25, "01:00"},
	}

	for _, tt := range tests {
		got := formatHour(tt.input)
		if got != tt.want {
			t.Errorf("formatHour(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
