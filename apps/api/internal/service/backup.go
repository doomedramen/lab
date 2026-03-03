package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

// BackupService handles VM backup and restore operations
type BackupService struct {
	backupRepo       repository.BackupRepository
	scheduleRepo     repository.BackupScheduleRepository
	backupLib        backupLibRepository
	vmRepo           repository.VMRepository
	taskSvc          *TaskService
	guestAgentRepo   repository.GuestAgentRepository
	scheduler        *cron.Cron
	schedulerRunning bool
}

// backupLibRepository defines the interface for libvirt backup operations
type backupLibRepository interface {
	Create(backup *model.Backup, compress bool, encrypt bool, passphrase string) error
	Restore(backup *model.Backup, targetVMID int, startAfter bool, passphrase string) error
	Delete(backup *model.Backup) error
	Verify(backup *model.Backup) (string, error)
}

// NewBackupService creates a new backup service.
// It accepts a context for the background scheduler.
func NewBackupService(
	ctx context.Context,
	backupRepo repository.BackupRepository,
	scheduleRepo repository.BackupScheduleRepository,
	backupLib backupLibRepository,
	vmRepo repository.VMRepository,
	taskSvc *TaskService,
) *BackupService {
	svc := &BackupService{
		backupRepo:   backupRepo,
		scheduleRepo: scheduleRepo,
		backupLib:    backupLib,
		vmRepo:       vmRepo,
		taskSvc:      taskSvc,
	}

	// Start scheduler for backup schedules
	svc.startScheduler(ctx)

	return svc
}

// WithGuestAgentRepo sets the guest agent repository for consistent backups
func (s *BackupService) WithGuestAgentRepo(repo repository.GuestAgentRepository) *BackupService {
	s.guestAgentRepo = repo
	return s
}

// freezeFilesystems attempts to freeze VM filesystems via guest agent
func (s *BackupService) freezeFilesystems(ctx context.Context, vmid int) (bool, int) {
	if s.guestAgentRepo == nil {
		return false, 0
	}

	count, err := s.guestAgentRepo.FreezeFilesystems(ctx, vmid)
	if err != nil {
		slog.Warn("Failed to freeze filesystems, continuing without freeze", "vmid", vmid, "error", err)
		return false, 0
	}

	slog.Info("Filesystems frozen for consistent backup", "vmid", vmid, "count", count)
	return true, count
}

// thawFilesystems attempts to thaw VM filesystems via guest agent
func (s *BackupService) thawFilesystems(ctx context.Context, vmid int) {
	if s.guestAgentRepo == nil {
		return
	}

	count, err := s.guestAgentRepo.ThawFilesystems(ctx, vmid)
	if err != nil {
		slog.Warn("Failed to thaw filesystems", "vmid", vmid, "error", err)
		return
	}

	slog.Info("Filesystems thawed after backup", "vmid", vmid, "count", count)
}

// startScheduler starts the cron scheduler for backup schedules.
// It accepts a context for graceful shutdown.
func (s *BackupService) startScheduler(ctx context.Context) {
	s.scheduler = cron.New(cron.WithSeconds())
	s.schedulerRunning = true

	// Check for due schedules every minute
	s.scheduler.AddFunc("0 * * * * *", func() {
		s.checkDueSchedules(ctx)
	})

	s.scheduler.Start()
	slog.Info("Backup scheduler started")
}

// StopScheduler stops the backup scheduler
func (s *BackupService) StopScheduler() {
	if s.scheduler != nil && s.schedulerRunning {
		s.scheduler.Stop()
		s.schedulerRunning = false
		slog.Info("Backup scheduler stopped")
	}
}

// checkDueSchedules checks and runs due backup schedules
func (s *BackupService) checkDueSchedules(ctx context.Context) {
	schedules, err := s.scheduleRepo.GetDueSchedules(ctx)
	if err != nil {
		slog.Error("Failed to get due backup schedules", "error", err)
		return
	}

	for _, schedule := range schedules {
		slog.Info("Running scheduled backup", "schedule_id", schedule.ID, "entity_id", schedule.EntityID)
		go s.runScheduledBackup(ctx, schedule)
	}
}

// runScheduledBackup runs a single scheduled backup
func (s *BackupService) runScheduledBackup(ctx context.Context, schedule *model.BackupSchedule) {

	// Create backup
	backup := &model.Backup{
		ID:            uuid.New().String(),
		VMID:          schedule.EntityID,
		VMName:        fmt.Sprintf("vm-%d", schedule.EntityID),
		Name:          fmt.Sprintf("Scheduled backup - %s", schedule.Name),
		Type:          schedule.BackupType,
		Status:        model.BackupStatusPending,
		StoragePool:   schedule.StoragePool,
		RetentionDays: schedule.RetentionDays,
		Encrypted:     schedule.Encrypt,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	// Update status to running
	backup.Status = model.BackupStatusRunning
	if err := s.backupRepo.Create(ctx, backup); err != nil {
		slog.Error("Failed to create backup record", "error", err)
		return
	}

	// Perform backup (no passphrase for scheduled backups - encryption uses keyfile)
	if err := s.backupLib.Create(backup, true, schedule.Encrypt, ""); err != nil {
		slog.Error("Failed to create backup", "backup_id", backup.ID, "error", err)
		backup.Status = model.BackupStatusFailed
		backup.ErrorMessage = err.Error()
		s.backupRepo.Update(ctx, backup)
		return
	}

	// Update backup record
	if err := s.backupRepo.Update(ctx, backup); err != nil {
		slog.Error("Failed to update backup record", "error", err)
	}

	// Apply retention count policy if set
	if schedule.RetainCount > 0 {
		s.applyRetainCountPolicy(ctx, schedule.EntityID, schedule.RetainCount)
	}

	// Update schedule
	nextRunAt := s.calculateNextRun(schedule.Schedule)
	if err := s.scheduleRepo.UpdateRunInfo(ctx, schedule.ID, time.Now().Format(time.RFC3339), nextRunAt, true); err != nil {
		slog.Error("Failed to update schedule", "error", err)
	}

	slog.Info("Scheduled backup completed", "backup_id", backup.ID)
}

// applyRetainCountPolicy deletes old backups beyond the retain count
func (s *BackupService) applyRetainCountPolicy(ctx context.Context, vmid int, retainCount int) {
	// Get all completed backups for this VM, ordered by created_at desc
	backups, err := s.backupRepo.List(ctx, vmid, model.BackupStatusCompleted, "")
	if err != nil {
		slog.Error("Failed to list backups for retention policy", "vmid", vmid, "error", err)
		return
	}

	// Delete backups beyond retain count
	if len(backups) > retainCount {
		for i := retainCount; i < len(backups); i++ {
			backup := backups[i]
			slog.Info("Deleting old backup per retention policy", "backup_id", backup.ID, "vmid", vmid)
			if err := s.backupLib.Delete(backup); err != nil {
				slog.Error("Failed to delete backup file", "backup_id", backup.ID, "error", err)
			}
			s.backupRepo.Delete(ctx, backup.ID)
		}
	}
}

// calculateNextRun calculates the next run time for a cron schedule
func (s *BackupService) calculateNextRun(cronExpr string) string {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return ""
	}
	return schedule.Next(time.Now()).Format(time.RFC3339)
}

// ListBackups returns backups with optional filters
func (s *BackupService) ListBackups(ctx context.Context, vmid int, status labv1.BackupStatus, storagePool string) ([]*labv1.Backup, int32, error) {
	modelStatus := protoToBackupStatus(status)
	backups, err := s.backupRepo.List(ctx, vmid, modelStatus, storagePool)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list backups: %w", err)
	}

	var protoBackups []*labv1.Backup
	for _, b := range backups {
		protoBackups = append(protoBackups, s.modelToProto(b))
	}

	return protoBackups, int32(len(protoBackups)), nil
}

// ListBackupsForAlerts returns backups for alert evaluation
// This method returns model types to avoid proto dependency in alert service
func (s *BackupService) ListBackupsForAlerts(ctx context.Context, status model.BackupStatus) ([]*model.Backup, error) {
	backups, err := s.backupRepo.List(ctx, 0, status, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	return backups, nil
}

// GetBackup returns details of a specific backup
func (s *BackupService) GetBackup(ctx context.Context, backupID string) (*labv1.Backup, error) {
	backup, err := s.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}
	if backup == nil {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}
	return s.modelToProto(backup), nil
}

// CreateBackup creates a new backup
func (s *BackupService) CreateBackup(ctx context.Context, req *labv1.CreateBackupRequest) (*labv1.Backup, string, error) {
	// Verify VM exists
	vm, err := s.vmRepo.GetByVMID(ctx, int(req.Vmid))
	if err != nil {
		return nil, "", fmt.Errorf("VM %d not found: %w", req.Vmid, err)
	}

	// Create backup record
	backup := &model.Backup{
		ID:            uuid.New().String(),
		VMID:          int(req.Vmid),
		VMName:        vm.Name,
		Name:          req.Name,
		Type:          protoToModelBackupType(req.Type),
		Status:        model.BackupStatusRunning,
		StoragePool:   req.StoragePool,
		RetentionDays: int(req.RetentionDays),
		Encrypted:     req.Encrypt,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	if err := s.backupRepo.Create(ctx, backup); err != nil {
		return nil, "", fmt.Errorf("failed to create backup record: %w", err)
	}

	// Create task for tracking
	resourceType := model.ResourceTypeVM
	if s.taskSvc != nil {
		task, err := s.taskSvc.Start(ctx, model.TaskTypeBackup, resourceType, fmt.Sprintf("vm/%d", req.Vmid), "Creating backup")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			go func(taskID string) {
				ctx := context.Background()

				// Try to freeze filesystems for consistent backup
				frozen, _ := s.freezeFilesystems(ctx, int(req.Vmid))
				if frozen {
					defer s.thawFilesystems(ctx, int(req.Vmid))
				}

				if err := s.backupLib.Create(backup, req.Compress, req.Encrypt, req.EncryptionPassphrase); err != nil {
					slog.Error("Failed to create backup", "backup_id", backup.ID, "error", err)
					backup.Status = model.BackupStatusFailed
					backup.ErrorMessage = err.Error()
					s.backupRepo.UpdateStatus(context.Background(), backup.ID, backup.Status, err.Error())
					s.taskSvc.Fail(ctx, taskID, err.Error())
					return
				}
				backup.Encrypted = req.Encrypt
				s.backupRepo.Update(context.Background(), backup)
				s.taskSvc.Progress(ctx, taskID, 100, "Backup completed")
				s.taskSvc.Complete(ctx, taskID)
			}(task.ID)
		}
	} else {
		// Fallback without task tracking
		go func() {
			ctx := context.Background()

			// Try to freeze filesystems for consistent backup
			frozen, _ := s.freezeFilesystems(ctx, int(req.Vmid))
			if frozen {
				defer s.thawFilesystems(ctx, int(req.Vmid))
			}

			if err := s.backupLib.Create(backup, req.Compress, req.Encrypt, req.EncryptionPassphrase); err != nil {
				slog.Error("Failed to create backup", "backup_id", backup.ID, "error", err)
				backup.Status = model.BackupStatusFailed
				backup.ErrorMessage = err.Error()
				s.backupRepo.UpdateStatus(context.Background(), backup.ID, backup.Status, err.Error())
				return
			}
			backup.Encrypted = req.Encrypt
			s.backupRepo.Update(context.Background(), backup)
		}()
	}

	return s.modelToProto(backup), backup.ID, nil
}

// RestoreBackup restores a VM from a backup
func (s *BackupService) RestoreBackup(ctx context.Context, req *labv1.RestoreBackupRequest) (string, int32, error) {
	backup, err := s.backupRepo.GetByID(ctx, req.BackupId)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get backup: %w", err)
	}
	if backup == nil {
		return "", 0, fmt.Errorf("backup not found: %s", req.BackupId)
	}

	// Check if passphrase is required for encrypted backups
	if backup.Encrypted && req.DecryptionPassphrase == "" {
		return "", 0, fmt.Errorf("decryption passphrase required for encrypted backup")
	}

	targetVMID := int32(req.TargetVmid)
	if targetVMID == 0 {
		targetVMID = int32(backup.VMID)
	}

	// Create task for tracking
	var taskID string
	if s.taskSvc != nil {
		resourceType := model.ResourceTypeBackup
		task, err := s.taskSvc.Start(ctx, model.TaskTypeRestore, resourceType, backup.ID, "Restoring backup")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			taskID = task.ID
			go func() {
				ctx := context.Background()
				if err := s.backupLib.Restore(backup, int(targetVMID), req.StartAfter, req.DecryptionPassphrase); err != nil {
					slog.Error("Failed to restore backup", "backup_id", backup.ID, "error", err)
					s.taskSvc.Fail(ctx, taskID, err.Error())
					return
				}
				s.taskSvc.Progress(ctx, taskID, 100, "Restore completed")
				s.taskSvc.Complete(ctx, taskID)
				slog.Info("Backup restored successfully", "backup_id", backup.ID, "target_vmid", targetVMID)
			}()
		}
	} else {
		// Fallback without task tracking
		taskID = uuid.New().String()
		go func() {
			if err := s.backupLib.Restore(backup, int(targetVMID), req.StartAfter, req.DecryptionPassphrase); err != nil {
				slog.Error("Failed to restore backup", "backup_id", backup.ID, "error", err)
				return
			}
			slog.Info("Backup restored successfully", "backup_id", backup.ID, "target_vmid", targetVMID)
		}()
	}

	return taskID, targetVMID, nil
}

// DeleteBackup deletes a backup
func (s *BackupService) DeleteBackup(ctx context.Context, backupID string) (string, error) {
	backup, err := s.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return "", fmt.Errorf("failed to get backup: %w", err)
	}
	if backup == nil {
		return "", fmt.Errorf("backup not found: %s", backupID)
	}

	// Create task for tracking
	var taskID string
	if s.taskSvc != nil {
		resourceType := model.ResourceTypeBackup
		task, err := s.taskSvc.Start(ctx, model.TaskTypeSnapshotDelete, resourceType, backupID, "Deleting backup")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			taskID = task.ID
			go func() {
				ctx := context.Background()
				if err := s.backupLib.Delete(backup); err != nil {
					slog.Error("Failed to delete backup file", "backup_id", backup.ID, "error", err)
					s.taskSvc.Fail(ctx, taskID, err.Error())
					return
				}
				s.backupRepo.Delete(ctx, backupID)
				s.taskSvc.Progress(ctx, taskID, 100, "Delete completed")
				s.taskSvc.Complete(ctx, taskID)
			}()
		}
	} else {
		// Fallback without task tracking
		taskID = uuid.New().String()
		go func() {
			ctx := context.Background()
			if err := s.backupLib.Delete(backup); err != nil {
				slog.Error("Failed to delete backup file", "backup_id", backup.ID, "error", err)
			}
			s.backupRepo.Delete(ctx, backupID)
		}()
	}

	return taskID, nil
}

// VerifyBackup verifies a backup's integrity using qemu-img check
func (s *BackupService) VerifyBackup(ctx context.Context, backupID string) (bool, string, string, error) {
	backup, err := s.backupRepo.GetByID(ctx, backupID)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to get backup: %w", err)
	}
	if backup == nil {
		return false, "", "", fmt.Errorf("backup not found: %s", backupID)
	}

	// Only verify completed backups
	if backup.Status != model.BackupStatusCompleted {
		return false, "", "", fmt.Errorf("can only verify completed backups")
	}

	// Run verification
	output, err := s.backupLib.Verify(backup)
	verifiedAt := time.Now().Format(time.RFC3339)

	if err != nil {
		// Verification failed
		backup.VerificationStatus = model.VerificationStatusFailed
		backup.VerifiedAt = verifiedAt
		backup.VerificationError = err.Error()
		s.backupRepo.Update(ctx, backup)
		return false, verifiedAt, output, nil
	}

	// Verification succeeded
	backup.VerificationStatus = model.VerificationStatusVerified
	backup.VerifiedAt = verifiedAt
	backup.VerificationError = ""
	s.backupRepo.Update(ctx, backup)
	return true, verifiedAt, output, nil
}

// ListBackupSchedules returns backup schedules
func (s *BackupService) ListBackupSchedules(ctx context.Context, entityType string, entityID int32) ([]*labv1.BackupSchedule, error) {
	schedules, err := s.scheduleRepo.List(ctx, entityType, int(entityID))
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	var protoSchedules []*labv1.BackupSchedule
	for _, sched := range schedules {
		protoSchedules = append(protoSchedules, s.scheduleModelToProto(sched))
	}

	return protoSchedules, nil
}

// CreateBackupSchedule creates a new backup schedule
func (s *BackupService) CreateBackupSchedule(ctx context.Context, req *labv1.CreateBackupScheduleRequest) (*labv1.BackupSchedule, error) {
	schedule := &model.BackupSchedule{
		ID:            uuid.New().String(),
		Name:          req.Name,
		EntityType:    req.EntityType,
		EntityID:      int(req.EntityId),
		StoragePool:   req.StoragePool,
		Schedule:      req.Schedule,
		BackupType:    protoToModelBackupType(req.BackupType),
		RetentionDays: int(req.RetentionDays),
		RetainCount:   int(req.RetainCount),
		Encrypt:       req.Encrypt,
		Enabled:       req.Enabled,
		CreatedAt:     time.Now().Format(time.RFC3339),
		UpdatedAt:     time.Now().Format(time.RFC3339),
	}

	// Calculate next run time
	schedule.NextRunAt = s.calculateNextRun(req.Schedule)

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	return s.scheduleModelToProto(schedule), nil
}

// UpdateBackupSchedule updates an existing schedule
func (s *BackupService) UpdateBackupSchedule(ctx context.Context, req *labv1.UpdateBackupScheduleRequest) (*labv1.BackupSchedule, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	if schedule == nil {
		return nil, fmt.Errorf("schedule not found: %s", req.Id)
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.Schedule != "" {
		schedule.Schedule = req.Schedule
		schedule.NextRunAt = s.calculateNextRun(req.Schedule)
	}
	if req.BackupType != labv1.BackupType_BACKUP_TYPE_UNSPECIFIED {
		schedule.BackupType = protoToModelBackupType(req.BackupType)
	}
	if req.RetentionDays > 0 {
		schedule.RetentionDays = int(req.RetentionDays)
	}
	if req.RetainCount > 0 {
		schedule.RetainCount = int(req.RetainCount)
	}
	schedule.Encrypt = req.Encrypt
	// Note: enabled field is always set in proto, use it directly
	schedule.Enabled = req.Enabled
	schedule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return nil, fmt.Errorf("failed to update schedule: %w", err)
	}

	return s.scheduleModelToProto(schedule), nil
}

// DeleteBackupSchedule deletes a schedule
func (s *BackupService) DeleteBackupSchedule(ctx context.Context, scheduleID string) error {
	return s.scheduleRepo.Delete(ctx, scheduleID)
}

// RunBackupSchedule manually runs a backup schedule
func (s *BackupService) RunBackupSchedule(ctx context.Context, scheduleID string) (string, string, error) {
	schedule, err := s.scheduleRepo.GetByID(ctx, scheduleID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get schedule: %w", err)
	}
	if schedule == nil {
		return "", "", fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Run the scheduled backup
	backupID := uuid.New().String()
	go s.runScheduledBackup(context.Background(), schedule)

	return uuid.New().String(), backupID, nil
}

// modelToProto converts model.Backup to labv1.Backup
func (s *BackupService) modelToProto(backup *model.Backup) *labv1.Backup {
	if backup == nil {
		return nil
	}

	return &labv1.Backup{
		Id:                backup.ID,
		Vmid:              int32(backup.VMID),
		VmName:            backup.VMName,
		Name:              backup.Name,
		Type:              modelBackupTypeToProto(backup.Type),
		Status:            modelBackupStatusToProto(backup.Status),
		SizeBytes:         backup.SizeBytes,
		StoragePool:       backup.StoragePool,
		BackupPath:        backup.BackupPath,
		CreatedAt:         backup.CreatedAt,
		CompletedAt:       backup.CompletedAt,
		ExpiresAt:         backup.ExpiresAt,
		ErrorMessage:      backup.ErrorMessage,
		RetentionDays:     int32(backup.RetentionDays),
		Encrypted:         backup.Encrypted,
		VerifiedAt:        backup.VerifiedAt,
		VerificationStatus: modelVerificationStatusToProto(backup.VerificationStatus),
		VerificationError: backup.VerificationError,
	}
}

// scheduleModelToProto converts model.BackupSchedule to labv1.BackupSchedule
func (s *BackupService) scheduleModelToProto(schedule *model.BackupSchedule) *labv1.BackupSchedule {
	if schedule == nil {
		return nil
	}

	return &labv1.BackupSchedule{
		Id:            schedule.ID,
		Name:          schedule.Name,
		EntityType:    schedule.EntityType,
		EntityId:      int32(schedule.EntityID),
		StoragePool:   schedule.StoragePool,
		Schedule:      schedule.Schedule,
		BackupType:    modelBackupTypeToProto(schedule.BackupType),
		RetentionDays: int32(schedule.RetentionDays),
		Enabled:       schedule.Enabled,
		CreatedAt:     schedule.CreatedAt,
		UpdatedAt:     schedule.UpdatedAt,
		LastRunAt:     schedule.LastRunAt,
		NextRunAt:     schedule.NextRunAt,
		TotalBackups:  int32(schedule.TotalBackups),
		RetainCount:   int32(schedule.RetainCount),
		Encrypt:       schedule.Encrypt,
	}
}

// Helper functions for type conversion
func protoToModelBackupType(t labv1.BackupType) model.BackupType {
	switch t {
	case labv1.BackupType_BACKUP_TYPE_FULL:
		return model.BackupTypeFull
	case labv1.BackupType_BACKUP_TYPE_INCREMENTAL:
		return model.BackupTypeIncremental
	case labv1.BackupType_BACKUP_TYPE_SNAPSHOT:
		return model.BackupTypeSnapshot
	default:
		return model.BackupTypeFull
	}
}

func modelBackupTypeToProto(t model.BackupType) labv1.BackupType {
	switch t {
	case model.BackupTypeFull:
		return labv1.BackupType_BACKUP_TYPE_FULL
	case model.BackupTypeIncremental:
		return labv1.BackupType_BACKUP_TYPE_INCREMENTAL
	case model.BackupTypeSnapshot:
		return labv1.BackupType_BACKUP_TYPE_SNAPSHOT
	default:
		return labv1.BackupType_BACKUP_TYPE_UNSPECIFIED
	}
}

func protoToBackupStatus(s labv1.BackupStatus) model.BackupStatus {
	switch s {
	case labv1.BackupStatus_BACKUP_STATUS_PENDING:
		return model.BackupStatusPending
	case labv1.BackupStatus_BACKUP_STATUS_RUNNING:
		return model.BackupStatusRunning
	case labv1.BackupStatus_BACKUP_STATUS_COMPLETED:
		return model.BackupStatusCompleted
	case labv1.BackupStatus_BACKUP_STATUS_FAILED:
		return model.BackupStatusFailed
	case labv1.BackupStatus_BACKUP_STATUS_DELETING:
		return model.BackupStatusDeleting
	default:
		return ""
	}
}

func modelBackupStatusToProto(s model.BackupStatus) labv1.BackupStatus {
	switch s {
	case model.BackupStatusPending:
		return labv1.BackupStatus_BACKUP_STATUS_PENDING
	case model.BackupStatusRunning:
		return labv1.BackupStatus_BACKUP_STATUS_RUNNING
	case model.BackupStatusCompleted:
		return labv1.BackupStatus_BACKUP_STATUS_COMPLETED
	case model.BackupStatusFailed:
		return labv1.BackupStatus_BACKUP_STATUS_FAILED
	case model.BackupStatusDeleting:
		return labv1.BackupStatus_BACKUP_STATUS_DELETING
	default:
		return labv1.BackupStatus_BACKUP_STATUS_UNSPECIFIED
	}
}

func modelVerificationStatusToProto(s model.VerificationStatus) labv1.VerificationStatus {
	switch s {
	case model.VerificationStatusNotRun:
		return labv1.VerificationStatus_VERIFICATION_STATUS_NOT_RUN
	case model.VerificationStatusPending:
		return labv1.VerificationStatus_VERIFICATION_STATUS_PENDING
	case model.VerificationStatusVerified:
		return labv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED
	case model.VerificationStatusFailed:
		return labv1.VerificationStatus_VERIFICATION_STATUS_FAILED
	default:
		return labv1.VerificationStatus_VERIFICATION_STATUS_UNSPECIFIED
	}
}

func protoToVerificationStatus(s labv1.VerificationStatus) model.VerificationStatus {
	switch s {
	case labv1.VerificationStatus_VERIFICATION_STATUS_NOT_RUN:
		return model.VerificationStatusNotRun
	case labv1.VerificationStatus_VERIFICATION_STATUS_PENDING:
		return model.VerificationStatusPending
	case labv1.VerificationStatus_VERIFICATION_STATUS_VERIFIED:
		return model.VerificationStatusVerified
	case labv1.VerificationStatus_VERIFICATION_STATUS_FAILED:
		return model.VerificationStatusFailed
	default:
		return model.VerificationStatusNotRun
	}
}
