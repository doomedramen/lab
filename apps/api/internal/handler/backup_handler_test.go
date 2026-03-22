package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

func TestBackupHandler_GetBackup_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.GetBackup(context.Background(), connect.NewRequest(&labv1.GetBackupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_CreateBackup_MissingVmid(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.CreateBackup(context.Background(), connect.NewRequest(&labv1.CreateBackupRequest{
		StoragePool: "pool-1",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_CreateBackup_MissingStoragePool(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.CreateBackup(context.Background(), connect.NewRequest(&labv1.CreateBackupRequest{
		Vmid: 100,
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_RestoreBackup_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.RestoreBackup(context.Background(), connect.NewRequest(&labv1.RestoreBackupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_DeleteBackup_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.DeleteBackup(context.Background(), connect.NewRequest(&labv1.DeleteBackupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_CreateBackupSchedule_MissingName(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.CreateBackupSchedule(context.Background(), connect.NewRequest(&labv1.CreateBackupScheduleRequest{
		EntityType: "vm",
		EntityId:   100,
		Schedule:   "0 2 * * *",
		// Name intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_CreateBackupSchedule_MissingEntity(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.CreateBackupSchedule(context.Background(), connect.NewRequest(&labv1.CreateBackupScheduleRequest{
		Name:     "daily",
		Schedule: "0 2 * * *",
		// EntityType and EntityId intentionally missing
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_CreateBackupSchedule_MissingSchedule(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.CreateBackupSchedule(context.Background(), connect.NewRequest(&labv1.CreateBackupScheduleRequest{
		Name:       "daily",
		EntityType: "vm",
		EntityId:   100,
		// Schedule intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_UpdateBackupSchedule_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.UpdateBackupSchedule(context.Background(), connect.NewRequest(&labv1.UpdateBackupScheduleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_DeleteBackupSchedule_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.DeleteBackupSchedule(context.Background(), connect.NewRequest(&labv1.DeleteBackupScheduleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestBackupHandler_RunBackupSchedule_MissingId(t *testing.T) {
	h := &BackupServiceServer{}
	_, err := h.RunBackupSchedule(context.Background(), connect.NewRequest(&labv1.RunBackupScheduleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}
