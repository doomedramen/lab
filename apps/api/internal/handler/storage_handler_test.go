package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

func TestStorageHandler_GetStoragePool_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.GetStoragePool(context.Background(), connect.NewRequest(&labv1.GetStoragePoolRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_CreateStoragePool_MissingName(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.CreateStoragePool(context.Background(), connect.NewRequest(&labv1.CreateStoragePoolRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_UpdateStoragePool_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.UpdateStoragePool(context.Background(), connect.NewRequest(&labv1.UpdateStoragePoolRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_DeleteStoragePool_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.DeleteStoragePool(context.Background(), connect.NewRequest(&labv1.DeleteStoragePoolRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_RefreshStoragePool_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.RefreshStoragePool(context.Background(), connect.NewRequest(&labv1.RefreshStoragePoolRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_CreateStorageDisk_MissingPoolId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.CreateStorageDisk(context.Background(), connect.NewRequest(&labv1.CreateStorageDiskRequest{
		SizeBytes: 10 * 1024 * 1024 * 1024,
		// PoolId intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_CreateStorageDisk_ZeroSize(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.CreateStorageDisk(context.Background(), connect.NewRequest(&labv1.CreateStorageDiskRequest{
		PoolId: "pool-1",
		// SizeBytes intentionally 0
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_ResizeStorageDisk_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.ResizeStorageDisk(context.Background(), connect.NewRequest(&labv1.ResizeStorageDiskRequest{
		NewSizeBytes: 20 * 1024 * 1024 * 1024,
		// DiskId intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_ResizeStorageDisk_ZeroSize(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.ResizeStorageDisk(context.Background(), connect.NewRequest(&labv1.ResizeStorageDiskRequest{
		DiskId: "disk-1",
		// NewSizeBytes intentionally 0
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_DeleteStorageDisk_MissingId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.DeleteStorageDisk(context.Background(), connect.NewRequest(&labv1.DeleteStorageDiskRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_MoveStorageDisk_MissingDiskId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.MoveStorageDisk(context.Background(), connect.NewRequest(&labv1.MoveStorageDiskRequest{
		TargetPoolId: "pool-2",
		// DiskId intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestStorageHandler_MoveStorageDisk_MissingTargetPoolId(t *testing.T) {
	h := &StorageServiceServer{}
	_, err := h.MoveStorageDisk(context.Background(), connect.NewRequest(&labv1.MoveStorageDiskRequest{
		DiskId: "disk-1",
		// TargetPoolId intentionally empty
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}
