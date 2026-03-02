package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// BackupServiceServer implements the BackupService Connect RPC server
type BackupServiceServer struct {
	backupService *service.BackupService
}

// NewBackupServiceServer creates a new backup service server
func NewBackupServiceServer(backupService *service.BackupService) *BackupServiceServer {
	return &BackupServiceServer{backupService: backupService}
}

// compile-time check that we implement the interface
var _ labv1connect.BackupServiceHandler = (*BackupServiceServer)(nil)

// ListBackups returns backups with optional filters
func (s *BackupServiceServer) ListBackups(
	ctx context.Context,
	req *connect.Request[labv1.ListBackupsRequest],
) (*connect.Response[labv1.ListBackupsResponse], error) {
	backups, total, err := s.backupService.ListBackups(
		ctx,
		int(req.Msg.Vmid),
		req.Msg.Status,
		req.Msg.StoragePool,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListBackupsResponse{
		Backups: backups,
		Total:   total,
	}), nil
}

// GetBackup returns details of a specific backup
func (s *BackupServiceServer) GetBackup(
	ctx context.Context,
	req *connect.Request[labv1.GetBackupRequest],
) (*connect.Response[labv1.GetBackupResponse], error) {
	if req.Msg.BackupId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("backup ID is required"))
	}

	backup, err := s.backupService.GetBackup(ctx, req.Msg.BackupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetBackupResponse{
		Backup: backup,
	}), nil
}

// CreateBackup creates a new backup
func (s *BackupServiceServer) CreateBackup(
	ctx context.Context,
	req *connect.Request[labv1.CreateBackupRequest],
) (*connect.Response[labv1.CreateBackupResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VM ID is required"))
	}

	if req.Msg.StoragePool == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("storage pool is required"))
	}

	backup, taskID, err := s.backupService.CreateBackup(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateBackupResponse{
		Backup: backup,
		TaskId: taskID,
	}), nil
}

// RestoreBackup restores a VM from a backup
func (s *BackupServiceServer) RestoreBackup(
	ctx context.Context,
	req *connect.Request[labv1.RestoreBackupRequest],
) (*connect.Response[labv1.RestoreBackupResponse], error) {
	if req.Msg.BackupId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("backup ID is required"))
	}

	taskID, targetVMID, err := s.backupService.RestoreBackup(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RestoreBackupResponse{
		TaskId:     taskID,
		TargetVmid: targetVMID,
	}), nil
}

// DeleteBackup deletes a backup
func (s *BackupServiceServer) DeleteBackup(
	ctx context.Context,
	req *connect.Request[labv1.DeleteBackupRequest],
) (*connect.Response[labv1.DeleteBackupResponse], error) {
	if req.Msg.BackupId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("backup ID is required"))
	}

	taskID, err := s.backupService.DeleteBackup(ctx, req.Msg.BackupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteBackupResponse{
		TaskId: taskID,
	}), nil
}

// VerifyBackup verifies a backup's integrity
func (s *BackupServiceServer) VerifyBackup(
	ctx context.Context,
	req *connect.Request[labv1.VerifyBackupRequest],
) (*connect.Response[labv1.VerifyBackupResponse], error) {
	if req.Msg.BackupId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("backup ID is required"))
	}

	success, verifiedAt, output, err := s.backupService.VerifyBackup(ctx, req.Msg.BackupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.VerifyBackupResponse{
		Success:      success,
		VerifiedAt:   verifiedAt,
		ErrorMessage: output, // Use output field for error message when failed
		Output:       output,
	}), nil
}

// ListBackupSchedules returns backup schedules
func (s *BackupServiceServer) ListBackupSchedules(
	ctx context.Context,
	req *connect.Request[labv1.ListBackupSchedulesRequest],
) (*connect.Response[labv1.ListBackupSchedulesResponse], error) {
	schedules, err := s.backupService.ListBackupSchedules(
		ctx,
		req.Msg.EntityType,
		req.Msg.EntityId,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListBackupSchedulesResponse{
		Schedules: schedules,
	}), nil
}

// CreateBackupSchedule creates a new backup schedule
func (s *BackupServiceServer) CreateBackupSchedule(
	ctx context.Context,
	req *connect.Request[labv1.CreateBackupScheduleRequest],
) (*connect.Response[labv1.CreateBackupScheduleResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("schedule name is required"))
	}

	if req.Msg.EntityType == "" || req.Msg.EntityId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("entity type and ID are required"))
	}

	if req.Msg.Schedule == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cron schedule is required"))
	}

	schedule, err := s.backupService.CreateBackupSchedule(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateBackupScheduleResponse{
		Schedule: schedule,
	}), nil
}

// UpdateBackupSchedule updates an existing schedule
func (s *BackupServiceServer) UpdateBackupSchedule(
	ctx context.Context,
	req *connect.Request[labv1.UpdateBackupScheduleRequest],
) (*connect.Response[labv1.UpdateBackupScheduleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("schedule ID is required"))
	}

	schedule, err := s.backupService.UpdateBackupSchedule(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateBackupScheduleResponse{
		Schedule: schedule,
	}), nil
}

// DeleteBackupSchedule deletes a schedule
func (s *BackupServiceServer) DeleteBackupSchedule(
	ctx context.Context,
	req *connect.Request[labv1.DeleteBackupScheduleRequest],
) (*connect.Response[labv1.DeleteBackupScheduleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("schedule ID is required"))
	}

	if err := s.backupService.DeleteBackupSchedule(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteBackupScheduleResponse{}), nil
}

// RunBackupSchedule manually runs a backup schedule
func (s *BackupServiceServer) RunBackupSchedule(
	ctx context.Context,
	req *connect.Request[labv1.RunBackupScheduleRequest],
) (*connect.Response[labv1.RunBackupScheduleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("schedule ID is required"))
	}

	taskID, backupID, err := s.backupService.RunBackupSchedule(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RunBackupScheduleResponse{
		TaskId:   taskID,
		BackupId: backupID,
	}), nil
}
