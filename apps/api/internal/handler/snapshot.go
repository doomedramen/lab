package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// SnapshotServiceServer implements the SnapshotService Connect RPC server
type SnapshotServiceServer struct {
	snapshotService *service.SnapshotService
}

// NewSnapshotServiceServer creates a new snapshot service server
func NewSnapshotServiceServer(snapshotService *service.SnapshotService) *SnapshotServiceServer {
	return &SnapshotServiceServer{snapshotService: snapshotService}
}

// compile-time check that we implement the interface
var _ labv1connect.SnapshotServiceHandler = (*SnapshotServiceServer)(nil)

// ListSnapshots returns all snapshots for a VM
func (s *SnapshotServiceServer) ListSnapshots(
	ctx context.Context,
	req *connect.Request[labv1.ListSnapshotsRequest],
) (*connect.Response[labv1.ListSnapshotsResponse], error) {
	snapshots, tree, err := s.snapshotService.ListSnapshots(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListSnapshotsResponse{
		Snapshots: snapshots,
		Tree:      tree,
	}), nil
}

// CreateSnapshot creates a new snapshot of a VM
func (s *SnapshotServiceServer) CreateSnapshot(
	ctx context.Context,
	req *connect.Request[labv1.CreateSnapshotRequest],
) (*connect.Response[labv1.CreateSnapshotResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("vmid is required"))
	}

	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot name is required"))
	}

	snapshot, taskID, err := s.snapshotService.CreateSnapshot(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateSnapshotResponse{
		Snapshot: snapshot,
		TaskId:   taskID,
	}), nil
}

// DeleteSnapshot deletes a snapshot
func (s *SnapshotServiceServer) DeleteSnapshot(
	ctx context.Context,
	req *connect.Request[labv1.DeleteSnapshotRequest],
) (*connect.Response[labv1.DeleteSnapshotResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("vmid is required"))
	}

	if req.Msg.SnapshotId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot ID is required"))
	}

	taskID, err := s.snapshotService.DeleteSnapshot(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteSnapshotResponse{
		TaskId: taskID,
	}), nil
}

// RestoreSnapshot restores a VM to a snapshot state
func (s *SnapshotServiceServer) RestoreSnapshot(
	ctx context.Context,
	req *connect.Request[labv1.RestoreSnapshotRequest],
) (*connect.Response[labv1.RestoreSnapshotResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("vmid is required"))
	}

	if req.Msg.SnapshotId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot ID is required"))
	}

	taskID, err := s.snapshotService.RestoreSnapshot(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RestoreSnapshotResponse{
		TaskId: taskID,
	}), nil
}

// GetSnapshotInfo returns detailed information about a snapshot
func (s *SnapshotServiceServer) GetSnapshotInfo(
	ctx context.Context,
	req *connect.Request[labv1.GetSnapshotInfoRequest],
) (*connect.Response[labv1.GetSnapshotInfoResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("vmid is required"))
	}

	if req.Msg.SnapshotId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("snapshot ID is required"))
	}

	snapshot, tree, err := s.snapshotService.GetSnapshotInfo(ctx, int(req.Msg.Vmid), req.Msg.SnapshotId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetSnapshotInfoResponse{
		Snapshot: snapshot,
		Tree:     tree,
	}), nil
}
