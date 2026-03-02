package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// MockBackupRepository mocks the backup repository
type MockBackupRepository struct {
	backups map[string]*model.Backup
}

func NewMockBackupRepository() *MockBackupRepository {
	return &MockBackupRepository{
		backups: make(map[string]*model.Backup),
	}
}

func (m *MockBackupRepository) Create(ctx context.Context, backup *model.Backup) error {
	m.backups[backup.ID] = backup
	return nil
}

func (m *MockBackupRepository) GetByID(ctx context.Context, id string) (*model.Backup, error) {
	if b, ok := m.backups[id]; ok {
		return b, nil
	}
	return nil, nil
}

func (m *MockBackupRepository) List(ctx context.Context, vmid int, status model.BackupStatus, storagePool string) ([]*model.Backup, error) {
	var result []*model.Backup
	for _, b := range m.backups {
		if vmid > 0 && b.VMID != vmid {
			continue
		}
		if status != "" && b.Status != status {
			continue
		}
		if storagePool != "" && b.StoragePool != storagePool {
			continue
		}
		result = append(result, b)
	}
	return result, nil
}

func (m *MockBackupRepository) Update(ctx context.Context, backup *model.Backup) error {
	m.backups[backup.ID] = backup
	return nil
}

func (m *MockBackupRepository) UpdateStatus(ctx context.Context, id string, status model.BackupStatus, errorMessage string) error {
	if b, ok := m.backups[id]; ok {
		b.Status = status
		b.ErrorMessage = errorMessage
	}
	return nil
}

func (m *MockBackupRepository) Delete(ctx context.Context, id string) error {
	delete(m.backups, id)
	return nil
}

func (m *MockBackupRepository) GetExpired(ctx context.Context) ([]*model.Backup, error) {
	return nil, nil
}

// MockBackupScheduleRepository mocks the backup schedule repository
type MockBackupScheduleRepository struct {
	schedules map[string]*model.BackupSchedule
}

func NewMockBackupScheduleRepository() *MockBackupScheduleRepository {
	return &MockBackupScheduleRepository{
		schedules: make(map[string]*model.BackupSchedule),
	}
}

func (m *MockBackupScheduleRepository) Create(ctx context.Context, schedule *model.BackupSchedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *MockBackupScheduleRepository) GetByID(ctx context.Context, id string) (*model.BackupSchedule, error) {
	if s, ok := m.schedules[id]; ok {
		return s, nil
	}
	return nil, nil
}

func (m *MockBackupScheduleRepository) List(ctx context.Context, entityType string, entityID int) ([]*model.BackupSchedule, error) {
	var result []*model.BackupSchedule
	for _, s := range m.schedules {
		if entityType != "" && s.EntityType != entityType {
			continue
		}
		if entityID > 0 && s.EntityID != entityID {
			continue
		}
		result = append(result, s)
	}
	return result, nil
}

func (m *MockBackupScheduleRepository) Update(ctx context.Context, schedule *model.BackupSchedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *MockBackupScheduleRepository) UpdateRunInfo(ctx context.Context, id string, lastRunAt, nextRunAt string, incrementBackups bool) error {
	if s, ok := m.schedules[id]; ok {
		s.LastRunAt = lastRunAt
		s.NextRunAt = nextRunAt
		if incrementBackups {
			s.TotalBackups++
		}
	}
	return nil
}

func (m *MockBackupScheduleRepository) Delete(ctx context.Context, id string) error {
	delete(m.schedules, id)
	return nil
}

func (m *MockBackupScheduleRepository) GetDueSchedules(ctx context.Context) ([]*model.BackupSchedule, error) {
	return nil, nil
}

// MockBackupLib defines interface for libvirt backup operations
type MockBackupLib struct {
	createFunc func(backup *model.Backup, compress bool, encrypt bool, passphrase string) error
}

func (m *MockBackupLib) Create(backup *model.Backup, compress bool, encrypt bool, passphrase string) error {
	if m.createFunc != nil {
		return m.createFunc(backup, compress, encrypt, passphrase)
	}
	backup.Status = model.BackupStatusCompleted
	backup.SizeBytes = 1024 * 1024 * 100 // 100MB
	backup.Encrypted = encrypt
	return nil
}

func (m *MockBackupLib) Restore(backup *model.Backup, targetVMID int, startAfter bool, passphrase string) error {
	return nil
}

func (m *MockBackupLib) Delete(backup *model.Backup) error {
	return nil
}

func (m *MockBackupLib) Verify(backup *model.Backup) (string, error) {
	return "Mock verification output", nil
}

// MockTaskRepository mocks the task repository
type MockTaskRepository struct {
	tasks map[string]*model.Task
}

func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*model.Task),
	}
}

func (m *MockTaskRepository) Create(ctx context.Context, task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *MockTaskRepository) List(ctx context.Context, filter model.TaskFilter) ([]*model.Task, error) {
	var result []*model.Task
	for _, t := range m.tasks {
		if filter.Status != "" && t.Status != filter.Status {
			continue
		}
		if filter.Type != "" && t.Type != filter.Type {
			continue
		}
		result = append(result, t)
	}
	return result, nil
}

func (m *MockTaskRepository) Update(ctx context.Context, task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) UpdateProgress(ctx context.Context, id string, progress int, message string) error {
	if t, ok := m.tasks[id]; ok {
		t.Progress = progress
		t.Message = message
	}
	return nil
}

func (m *MockTaskRepository) UpdateStatus(ctx context.Context, id string, status model.TaskStatus, message string) error {
	if t, ok := m.tasks[id]; ok {
		t.Status = status
		t.Message = message
	}
	return nil
}

func (m *MockTaskRepository) Delete(ctx context.Context, id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *MockTaskRepository) DeleteCompleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

func TestBackupService_ListBackups(t *testing.T) {
	mockBackupRepo := NewMockBackupRepository()
	mockScheduleRepo := NewMockBackupScheduleRepository()
	mockBackupLib := &MockBackupLib{}
	mockVMRepo := &MockVMRepo{vms: make(map[int]*model.VM)}
	mockTaskRepo := &MockTaskRepository{}

	// Add a test VM
	mockVMRepo.vms[100] = &model.VM{VMID: 100, Name: "test-vm"}

	service := NewBackupService(mockBackupRepo, mockScheduleRepo, mockBackupLib, mockVMRepo, NewTaskService(mockTaskRepo))
	defer service.StopScheduler()

	ctx := context.Background()
	backups, total, err := service.ListBackups(ctx, 100, labv1.BackupStatus_BACKUP_STATUS_UNSPECIFIED, "")

	if err != nil {
		t.Fatalf("ListBackups returned error: %v", err)
	}

	if total != 0 {
		t.Errorf("Expected 0 backups, got %d", total)
	}

	// Empty slice is acceptable for no backups
	if backups == nil {
		t.Log("Got nil backups (acceptable for empty list)")
	}
}

func TestBackupService_CreateBackup_VMNotFound(t *testing.T) {
	mockBackupRepo := NewMockBackupRepository()
	mockScheduleRepo := NewMockBackupScheduleRepository()
	mockBackupLib := &MockBackupLib{}
	mockVMRepo := &MockVMRepo{vms: make(map[int]*model.VM)}
	mockTaskRepo := NewMockTaskRepository()

	service := NewBackupService(mockBackupRepo, mockScheduleRepo, mockBackupLib, mockVMRepo, NewTaskService(mockTaskRepo))
	defer service.StopScheduler()

	ctx := context.Background()
	_, _, err := service.CreateBackup(ctx, &labv1.CreateBackupRequest{
		Vmid:        999,
		StoragePool: "default",
	})

	if err == nil {
		t.Error("Expected error for non-existent VM")
	}
}

func TestBackupService_CreateBackup_Success(t *testing.T) {
	mockBackupRepo := NewMockBackupRepository()
	mockScheduleRepo := NewMockBackupScheduleRepository()
	mockBackupLib := &MockBackupLib{}
	mockVMRepo := &MockVMRepo{vms: make(map[int]*model.VM)}
	mockTaskRepo := NewMockTaskRepository()

	// Add a test VM
	mockVMRepo.vms[100] = &model.VM{VMID: 100, Name: "test-vm"}

	service := NewBackupService(mockBackupRepo, mockScheduleRepo, mockBackupLib, mockVMRepo, NewTaskService(mockTaskRepo))
	defer service.StopScheduler()

	ctx := context.Background()
	backup, taskID, err := service.CreateBackup(ctx, &labv1.CreateBackupRequest{
		Vmid:        100,
		Name:        "test-backup",
		StoragePool: "default",
		Compress:    true,
	})

	if err != nil {
		t.Fatalf("CreateBackup returned error: %v", err)
	}

	if backup == nil {
		t.Fatal("Expected non-nil backup")
	}

	if backup.Name != "test-backup" {
		t.Errorf("Expected name 'test-backup', got '%s'", backup.Name)
	}

	if taskID == "" {
		t.Error("Expected non-empty task ID")
	}
}

func TestBackupService_ModelToProto(t *testing.T) {
	mockBackupRepo := NewMockBackupRepository()
	mockScheduleRepo := NewMockBackupScheduleRepository()
	mockBackupLib := &MockBackupLib{}
	mockVMRepo := &MockVMRepo{vms: make(map[int]*model.VM)}
	mockTaskRepo := NewMockTaskRepository()

	service := NewBackupService(mockBackupRepo, mockScheduleRepo, mockBackupLib, mockVMRepo, NewTaskService(mockTaskRepo))
	defer service.StopScheduler()

	modelBackup := &model.Backup{
		ID:            "backup-1",
		VMID:          100,
		VMName:        "test-vm",
		Name:          "test-backup",
		Type:          model.BackupTypeFull,
		Status:        model.BackupStatusCompleted,
		SizeBytes:     1024 * 1024 * 100,
		StoragePool:   "default",
		RetentionDays: 30,
	}

	protoBackup := service.modelToProto(modelBackup)

	if protoBackup == nil {
		t.Fatal("Expected non-nil proto backup")
	}

	if protoBackup.Id != "backup-1" {
		t.Errorf("Expected ID 'backup-1', got '%s'", protoBackup.Id)
	}

	if protoBackup.Vmid != 100 {
		t.Errorf("Expected VMID 100, got %d", protoBackup.Vmid)
	}

	if protoBackup.Type != labv1.BackupType_BACKUP_TYPE_FULL {
		t.Errorf("Expected type FULL, got %v", protoBackup.Type)
	}
}

func TestBackupService_ScheduleModelToProto(t *testing.T) {
	mockBackupRepo := NewMockBackupRepository()
	mockScheduleRepo := NewMockBackupScheduleRepository()
	mockBackupLib := &MockBackupLib{}
	mockVMRepo := &MockVMRepo{vms: make(map[int]*model.VM)}
	mockTaskRepo := NewMockTaskRepository()

	service := NewBackupService(mockBackupRepo, mockScheduleRepo, mockBackupLib, mockVMRepo, NewTaskService(mockTaskRepo))
	defer service.StopScheduler()

	modelSchedule := &model.BackupSchedule{
		ID:            "schedule-1",
		Name:          "daily-backup",
		EntityType:    "vm",
		EntityID:      100,
		StoragePool:   "default",
		Schedule:      "0 2 * * *",
		BackupType:    model.BackupTypeFull,
		RetentionDays: 30,
		Enabled:       true,
		TotalBackups:  5,
	}

	protoSchedule := service.scheduleModelToProto(modelSchedule)

	if protoSchedule == nil {
		t.Fatal("Expected non-nil proto schedule")
	}

	if protoSchedule.Id != "schedule-1" {
		t.Errorf("Expected ID 'schedule-1', got '%s'", protoSchedule.Id)
	}

	if protoSchedule.Name != "daily-backup" {
		t.Errorf("Expected name 'daily-backup', got '%s'", protoSchedule.Name)
	}

	if protoSchedule.Schedule != "0 2 * * *" {
		t.Errorf("Expected schedule '0 2 * * *', got '%s'", protoSchedule.Schedule)
	}
}

// MockVMRepo is a minimal mock for repository.VMRepository
type MockVMRepo struct {
	vms map[int]*model.VM
}

func (m *MockVMRepo) GetAll(_ context.Context) ([]*model.VM, error)               { return nil, nil }
func (m *MockVMRepo) GetByNode(_ context.Context, _ string) ([]*model.VM, error)  { return nil, nil }
func (m *MockVMRepo) GetByID(_ context.Context, _ string) (*model.VM, error) {
	return nil, fmt.Errorf("not found")
}
func (m *MockVMRepo) GetByVMID(_ context.Context, vmid int) (*model.VM, error) {
	if vm, ok := m.vms[vmid]; ok {
		return vm, nil
	}
	return nil, fmt.Errorf("VM %d not found", vmid)
}
func (m *MockVMRepo) Create(_ context.Context, _ *model.VMCreateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepo) Update(_ context.Context, _ int, _ *model.VMUpdateRequest) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepo) Delete(_ context.Context, _ int) error            { return fmt.Errorf("not implemented") }
func (m *MockVMRepo) Clone(_ context.Context, _ *model.VMCloneRequest, _ func(int, string)) (*model.VM, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *MockVMRepo) Start(_ context.Context, _ int) error             { return nil }
func (m *MockVMRepo) Stop(_ context.Context, _ int) error              { return nil }
func (m *MockVMRepo) Shutdown(_ context.Context, _ int) error          { return nil }
func (m *MockVMRepo) Pause(_ context.Context, _ int) error             { return nil }
func (m *MockVMRepo) Resume(_ context.Context, _ int) error            { return nil }
func (m *MockVMRepo) Reboot(_ context.Context, _ int) error            { return nil }
func (m *MockVMRepo) GetVNCPort(_ context.Context, _ int) (int, error) { return 0, nil }
