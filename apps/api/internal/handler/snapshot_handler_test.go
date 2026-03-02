package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

func TestSnapshotHandler_CreateSnapshot_MissingVmid(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.CreateSnapshotRequest{
		Name: "snap1",
		// Vmid intentionally 0
	})
	_, err := h.CreateSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_CreateSnapshot_MissingName(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.CreateSnapshotRequest{
		Vmid: 100,
		// Name intentionally empty
	})
	_, err := h.CreateSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_DeleteSnapshot_MissingVmid(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.DeleteSnapshotRequest{
		SnapshotId: "snap-1",
		// Vmid intentionally 0
	})
	_, err := h.DeleteSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_DeleteSnapshot_MissingSnapshotId(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.DeleteSnapshotRequest{
		Vmid: 100,
		// SnapshotId intentionally empty
	})
	_, err := h.DeleteSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_RestoreSnapshot_MissingVmid(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.RestoreSnapshotRequest{
		SnapshotId: "snap-1",
		// Vmid intentionally 0
	})
	_, err := h.RestoreSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_RestoreSnapshot_MissingSnapshotId(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.RestoreSnapshotRequest{
		Vmid: 100,
		// SnapshotId intentionally empty
	})
	_, err := h.RestoreSnapshot(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_GetSnapshotInfo_MissingVmid(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.GetSnapshotInfoRequest{
		SnapshotId: "snap-1",
		// Vmid intentionally 0
	})
	_, err := h.GetSnapshotInfo(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestSnapshotHandler_GetSnapshotInfo_MissingSnapshotId(t *testing.T) {
	h := &SnapshotServiceServer{}
	req := connect.NewRequest(&labv1.GetSnapshotInfoRequest{
		Vmid: 100,
		// SnapshotId intentionally empty
	})
	_, err := h.GetSnapshotInfo(context.Background(), req)
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

