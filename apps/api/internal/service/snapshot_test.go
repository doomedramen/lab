package service

import (
	"context"
	"fmt"
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// MockSnapshotLibRepository mocks the libvirt snapshot repository
type MockSnapshotLibRepository struct {
	snapshots map[string]*model.Snapshot
}

func NewMockSnapshotLibRepository() *MockSnapshotLibRepository {
	return &MockSnapshotLibRepository{
		snapshots: make(map[string]*model.Snapshot),
	}
}

func (m *MockSnapshotLibRepository) Create(vmid int, name, description string, live, includeMemory bool) (*model.Snapshot, error) {
	snapshot := &model.Snapshot{
		ID:          name,
		VMID:        vmid,
		Name:        name,
		Description: description,
		CreatedAt:   "2024-01-01T00:00:00Z",
		SizeBytes:   1024,
		Status:      model.SnapshotStatusReady,
		VMState:     model.VMStateStopped,
		HasChildren: false,
	}
	m.snapshots[name] = snapshot
	return snapshot, nil
}

func (m *MockSnapshotLibRepository) Delete(vmid int, snapshotID string) error {
	delete(m.snapshots, snapshotID)
	return nil
}

func (m *MockSnapshotLibRepository) Restore(vmid int, snapshotID string) error {
	return nil
}

func (m *MockSnapshotLibRepository) List(vmid int) ([]model.Snapshot, error) {
	result := []model.Snapshot{}
	for _, s := range m.snapshots {
		if s.VMID == vmid {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *MockSnapshotLibRepository) GetInfo(vmid int, snapshotID string) (*model.Snapshot, error) {
	if s, ok := m.snapshots[snapshotID]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *MockSnapshotLibRepository) GetSnapshotTree(vmid int) (*model.SnapshotTree, error) {
	return nil, nil
}

// MockSnapshotRepository is a mock implementation of repository.SnapshotRepository
type MockSnapshotRepository struct {
	snapshots map[string]*model.Snapshot
}

func NewMockSnapshotRepository() *MockSnapshotRepository {
	return &MockSnapshotRepository{
		snapshots: make(map[string]*model.Snapshot),
	}
}

func (m *MockSnapshotRepository) Create(ctx context.Context, snapshot *model.Snapshot) error {
	m.snapshots[snapshot.ID] = snapshot
	return nil
}

func (m *MockSnapshotRepository) GetByID(ctx context.Context, vmid int, id string) (*model.Snapshot, error) {
	for _, s := range m.snapshots {
		if s.VMID == vmid && s.ID == id {
			return s, nil
		}
	}
	return nil, nil
}

func (m *MockSnapshotRepository) ListByVMID(ctx context.Context, vmid int) ([]*model.Snapshot, error) {
	var result []*model.Snapshot
	for _, s := range m.snapshots {
		if s.VMID == vmid {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *MockSnapshotRepository) Update(ctx context.Context, snapshot *model.Snapshot) error {
	m.snapshots[snapshot.ID] = snapshot
	return nil
}

func (m *MockSnapshotRepository) Delete(ctx context.Context, vmid int, id string) error {
	delete(m.snapshots, id)
	return nil
}

func (m *MockSnapshotRepository) DeleteWithChildren(ctx context.Context, vmid int, id string) error {
	delete(m.snapshots, id)
	return nil
}

func (m *MockSnapshotRepository) UpdateStatus(ctx context.Context, id string, vmid int, status model.SnapshotStatus) error {
	if s, ok := m.snapshots[id]; ok {
		s.Status = status
	}
	return nil
}

func (m *MockSnapshotRepository) UpdateSize(ctx context.Context, id string, vmid int, sizeBytes int64) error {
	if s, ok := m.snapshots[id]; ok {
		s.SizeBytes = sizeBytes
	}
	return nil
}

func (m *MockSnapshotRepository) Exists(ctx context.Context, vmid int, id string) bool {
	_, ok := m.snapshots[id]
	return ok
}

func (m *MockSnapshotRepository) GetTree(ctx context.Context, vmid int) (*model.SnapshotTree, error) {
	return nil, nil
}

// MockVMRepository is a mock implementation of repository.VMRepository
type MockVMRepository struct {
	vms map[int]*model.VM
}

func NewMockVMRepository() *MockVMRepository {
	return &MockVMRepository{
		vms: make(map[int]*model.VM),
	}
}

func (m *MockVMRepository) GetAll(_ context.Context) ([]*model.VM, error)              { return nil, nil }
func (m *MockVMRepository) GetByNode(_ context.Context, _ string) ([]*model.VM, error) { return nil, nil }
func (m *MockVMRepository) GetByID(_ context.Context, _ string) (*model.VM, error) {
	return nil, fmt.Errorf("not found")
}
func (m *MockVMRepository) GetByVMID(_ context.Context, vmid int) (*model.VM, error) {
	if vm, ok := m.vms[vmid]; ok {
		return vm, nil
	}
	return nil, fmt.Errorf("VM %d not found", vmid)
}
func (m *MockVMRepository) Create(_ context.Context, _ *model.VMCreateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepository) Update(_ context.Context, _ int, _ *model.VMUpdateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepository) Delete(_ context.Context, _ int) error            { return fmt.Errorf("not implemented") }
func (m *MockVMRepository) Clone(_ context.Context, _ *model.VMCloneRequest, _ func(int, string)) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepository) Start(_ context.Context, _ int) error             { return nil }
func (m *MockVMRepository) Stop(_ context.Context, _ int) error              { return nil }
func (m *MockVMRepository) Shutdown(_ context.Context, _ int) error          { return nil }
func (m *MockVMRepository) Pause(_ context.Context, _ int) error             { return nil }
func (m *MockVMRepository) Resume(_ context.Context, _ int) error            { return nil }
func (m *MockVMRepository) Reboot(_ context.Context, _ int) error            { return nil }
func (m *MockVMRepository) GetVNCPort(_ context.Context, _ int) (int, error) { return 0, nil }
func (m *MockVMRepository) AddVM(vm *model.VM)                               { m.vms[vm.VMID] = vm }

func TestSnapshotService_ListSnapshots(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	// Add a test VM
	mockVMRepo.AddVM(&model.VM{VMID: 100, Name: "test-vm"})

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	snapshots, tree, err := service.ListSnapshots(ctx, 100)

	if err != nil {
		t.Fatalf("ListSnapshots returned error: %v", err)
	}

	// snapshots can be nil or empty slice - both are valid
	if snapshots == nil {
		t.Log("Got nil snapshots (acceptable for empty list)")
	}

	if tree != nil {
		t.Error("Expected nil tree for empty snapshot list")
	}
}

func TestSnapshotService_CreateSnapshot_VMNotFound(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	_, _, err := service.CreateSnapshot(ctx, &labv1.CreateSnapshotRequest{
		Vmid: 999, // Non-existent VM
		Name: "test-snapshot",
	})

	if err == nil {
		t.Error("Expected error for non-existent VM")
	}
}

func TestSnapshotService_DeleteSnapshot_VMNotFound(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	_, err := service.DeleteSnapshot(ctx, &labv1.DeleteSnapshotRequest{
		Vmid:       999,
		SnapshotId: "test-snapshot",
	})

	if err == nil {
		t.Error("Expected error for non-existent VM")
	}
}

func TestSnapshotService_RestoreSnapshot_VMNotFound(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	_, err := service.RestoreSnapshot(ctx, &labv1.RestoreSnapshotRequest{
		Vmid:       999,
		SnapshotId: "test-snapshot",
	})

	if err == nil {
		t.Error("Expected error for non-existent VM")
	}
}

func TestSnapshotService_GetSnapshotInfo_VMNotFound(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	_, _, err := service.GetSnapshotInfo(ctx, 999, "test-snapshot")

	if err == nil {
		t.Error("Expected error for non-existent VM")
	}
}

func TestSnapshotService_ModelToProto(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	modelSnapshot := &model.Snapshot{
		ID:          "snap-1",
		VMID:        100,
		Name:        "test-snapshot",
		Description: "Test snapshot",
		CreatedAt:   "2024-01-01T00:00:00Z",
		SizeBytes:   1024,
		Status:      model.SnapshotStatusReady,
		VMState:     model.VMStateStopped,
		HasChildren: false,
	}

	protoSnapshot := service.modelToProto(modelSnapshot)

	if protoSnapshot == nil {
		t.Fatal("Expected non-nil proto snapshot")
	}

	if protoSnapshot.Id != "snap-1" {
		t.Errorf("Expected ID 'snap-1', got '%s'", protoSnapshot.Id)
	}

	if protoSnapshot.Vmid != 100 {
		t.Errorf("Expected VMID 100, got %d", protoSnapshot.Vmid)
	}

	if protoSnapshot.Name != "test-snapshot" {
		t.Errorf("Expected name 'test-snapshot', got '%s'", protoSnapshot.Name)
	}
}

func TestSnapshotService_CreateSnapshot_Success(t *testing.T) {
	mockSnapshotRepo := NewMockSnapshotRepository()
	mockSnapshotLib := NewMockSnapshotLibRepository()
	mockVMRepo := NewMockVMRepository()
	mockTaskRepo := NewMockTaskRepository()

	// Add a test VM
	mockVMRepo.AddVM(&model.VM{VMID: 100, Name: "test-vm"})

	service := NewSnapshotService(mockSnapshotRepo, mockSnapshotLib, mockVMRepo, NewTaskService(mockTaskRepo))

	ctx := context.Background()
	snapshot, taskID, err := service.CreateSnapshot(ctx, &labv1.CreateSnapshotRequest{
		Vmid:        100,
		Name:        "test-snapshot",
		Description: "Test snapshot",
		Live:        false,
	})

	if err != nil {
		t.Fatalf("CreateSnapshot returned error: %v", err)
	}

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	if snapshot.Name != "test-snapshot" {
		t.Errorf("Expected name 'test-snapshot', got '%s'", snapshot.Name)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}
