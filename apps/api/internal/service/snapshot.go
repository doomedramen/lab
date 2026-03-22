package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/google/uuid"
)

// SnapshotService handles VM snapshot operations
type SnapshotService struct {
	snapshotRepo repository.SnapshotRepository
	snapshotLib  libvirtSnapshotRepository
	vmRepo       repository.VMRepository
	taskSvc      *TaskService
}

// libvirtSnapshotRepository defines the interface for libvirt snapshot operations
type libvirtSnapshotRepository interface {
	Create(vmid int, name, description string, live, includeMemory bool) (*model.Snapshot, error)
	Delete(vmid int, snapshotID string) error
	Restore(vmid int, snapshotID string) error
	List(vmid int) ([]model.Snapshot, error)
	GetInfo(vmid int, snapshotID string) (*model.Snapshot, error)
	GetSnapshotTree(vmid int) (*model.SnapshotTree, error)
}

// NewSnapshotService creates a new snapshot service
func NewSnapshotService(
	snapshotRepo repository.SnapshotRepository,
	snapshotLib libvirtSnapshotRepository,
	vmRepo repository.VMRepository,
	taskSvc *TaskService,
) *SnapshotService {
	return &SnapshotService{
		snapshotRepo: snapshotRepo,
		snapshotLib:  snapshotLib,
		vmRepo:       vmRepo,
		taskSvc:      taskSvc,
	}
}

// ListSnapshots returns all snapshots for a VM
func (s *SnapshotService) ListSnapshots(ctx context.Context, vmid int) ([]*labv1.Snapshot, *labv1.SnapshotTree, error) {
	// Get snapshots from libvirt
	libvirtSnapshots, err := s.snapshotLib.List(vmid)
	if err != nil {
		slog.Error("Failed to list snapshots from libvirt", "vmid", vmid, "error", err)
		return nil, nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Convert to proto format
	var snapshots []*labv1.Snapshot
	for _, snap := range libvirtSnapshots {
		snapshots = append(snapshots, s.modelToProto(&snap))
	}

	// Build snapshot tree
	tree, err := s.buildSnapshotTree(ctx, vmid)
	if err != nil {
		slog.Warn("Failed to build snapshot tree", "vmid", vmid, "error", err)
	}

	return snapshots, tree, nil
}

// CreateSnapshot creates a new snapshot of a VM
func (s *SnapshotService) CreateSnapshot(ctx context.Context, req *labv1.CreateSnapshotRequest) (*labv1.Snapshot, string, error) {
	// Verify VM exists
	_, err := s.vmRepo.GetByVMID(ctx, int(req.Vmid))
	if err != nil {
		return nil, "", fmt.Errorf("VM %d not found: %w", req.Vmid, err)
	}

	// Generate snapshot ID
	snapshotID := uuid.New().String()

	// Create snapshot in libvirt
	libvirtSnapshot, err := s.snapshotLib.Create(
		int(req.Vmid),
		req.Name,
		req.Description,
		req.Live,
		req.IncludeMemory,
	)
	if err != nil {
		slog.Error("Failed to create libvirt snapshot", "vmid", req.Vmid, "error", err)
		return nil, "", fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Use the actual snapshot name from libvirt
	snapshotID = libvirtSnapshot.ID

	// Save metadata to SQLite
	dbSnapshot := &model.Snapshot{
		ID:           snapshotID,
		VMID:         int(req.Vmid),
		Name:         req.Name,
		Description:  req.Description,
		CreatedAt:    time.Now().Format(time.RFC3339),
		SizeBytes:    libvirtSnapshot.SizeBytes,
		Status:       model.SnapshotStatusReady,
		VMState:      libvirtSnapshot.VMState,
		HasChildren:  false,
		SnapshotPath: libvirtSnapshot.SnapshotPath,
	}

	if err := s.snapshotRepo.Create(ctx, dbSnapshot); err != nil {
		slog.Error("Failed to save snapshot metadata", "vmid", req.Vmid, "error", err)
		// Note: We don't fail the operation - libvirt snapshot was created successfully
	}

	// Create task for tracking
	var taskID string
	if s.taskSvc != nil {
		resourceType := model.ResourceTypeVM
		task, err := s.taskSvc.Start(ctx, model.TaskTypeSnapshotCreate, resourceType, fmt.Sprintf("vm/%d", req.Vmid), "Creating snapshot")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			taskID = task.ID
			// Snapshot creation is instant, mark as complete
			s.taskSvc.Progress(ctx, taskID, 100, "Snapshot created")
			s.taskSvc.Complete(ctx, taskID)
		}
	} else {
		// Fallback without task tracking
		taskID = uuid.New().String()
	}

	return s.modelToProto(libvirtSnapshot), taskID, nil
}

// DeleteSnapshot deletes a snapshot
func (s *SnapshotService) DeleteSnapshot(ctx context.Context, req *labv1.DeleteSnapshotRequest) (string, error) {
	// Verify VM exists
	_, err := s.vmRepo.GetByVMID(ctx, int(req.Vmid))
	if err != nil {
		return "", fmt.Errorf("VM %d not found: %w", req.Vmid, err)
	}

	// Check if snapshot exists
	exists := s.snapshotRepo.Exists(ctx, int(req.Vmid), req.SnapshotId)
	if !exists {
		return "", fmt.Errorf("snapshot %s not found", req.SnapshotId)
	}

	// Delete from libvirt
	if err := s.snapshotLib.Delete(int(req.Vmid), req.SnapshotId); err != nil {
		slog.Error("Failed to delete libvirt snapshot", "vmid", req.Vmid, "snapshot", req.SnapshotId, "error", err)
		return "", fmt.Errorf("failed to delete snapshot: %w", err)
	}

	// Delete from SQLite
	if req.IncludeChildren {
		if err := s.snapshotRepo.DeleteWithChildren(ctx, int(req.Vmid), req.SnapshotId); err != nil {
			slog.Error("Failed to delete snapshot metadata with children", "vmid", req.Vmid, "error", err)
		}
	} else {
		if err := s.snapshotRepo.Delete(ctx, int(req.Vmid), req.SnapshotId); err != nil {
			slog.Error("Failed to delete snapshot metadata", "vmid", req.Vmid, "error", err)
		}
	}

	// Create task for tracking
	var taskID string
	if s.taskSvc != nil {
		resourceType := model.ResourceTypeSnapshot
		task, err := s.taskSvc.Start(ctx, model.TaskTypeSnapshotDelete, resourceType, req.SnapshotId, "Deleting snapshot")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			taskID = task.ID
			// Snapshot deletion is instant, mark as complete
			s.taskSvc.Progress(ctx, taskID, 100, "Snapshot deleted")
			s.taskSvc.Complete(ctx, taskID)
		}
	} else {
		// Fallback without task tracking
		taskID = uuid.New().String()
	}

	return taskID, nil
}

// RestoreSnapshot restores a VM to a snapshot state
func (s *SnapshotService) RestoreSnapshot(ctx context.Context, req *labv1.RestoreSnapshotRequest) (string, error) {
	// Verify VM exists
	_, err := s.vmRepo.GetByVMID(ctx, int(req.Vmid))
	if err != nil {
		return "", fmt.Errorf("VM %d not found: %w", req.Vmid, err)
	}

	// Check if snapshot exists
	exists := s.snapshotRepo.Exists(ctx, int(req.Vmid), req.SnapshotId)
	if !exists {
		return "", fmt.Errorf("snapshot %s not found", req.SnapshotId)
	}

	// Restore from libvirt
	if err := s.snapshotLib.Restore(int(req.Vmid), req.SnapshotId); err != nil {
		slog.Error("Failed to restore libvirt snapshot", "vmid", req.Vmid, "snapshot", req.SnapshotId, "error", err)
		return "", fmt.Errorf("failed to restore snapshot: %w", err)
	}

	// Create task for tracking
	var taskID string
	if s.taskSvc != nil {
		resourceType := model.ResourceTypeVM
		task, err := s.taskSvc.Start(ctx, model.TaskTypeSnapshotRestore, resourceType, fmt.Sprintf("vm/%d", req.Vmid), "Restoring snapshot")
		if err != nil {
			slog.Error("Failed to create task", "error", err)
		} else {
			taskID = task.ID
			// Snapshot restore is instant, mark as complete
			s.taskSvc.Progress(ctx, taskID, 100, "Snapshot restored")
			s.taskSvc.Complete(ctx, taskID)
		}
	} else {
		// Fallback without task tracking
		taskID = uuid.New().String()
	}

	return taskID, nil
}

// GetSnapshotInfo returns detailed information about a snapshot
func (s *SnapshotService) GetSnapshotInfo(ctx context.Context, vmid int, snapshotID string) (*labv1.Snapshot, *labv1.SnapshotTree, error) {
	// Verify VM exists
	_, err := s.vmRepo.GetByVMID(ctx, vmid)
	if err != nil {
		return nil, nil, fmt.Errorf("VM %d not found: %w", vmid, err)
	}

	// Get snapshot info from libvirt
	libvirtSnapshot, err := s.snapshotLib.GetInfo(vmid, snapshotID)
	if err != nil {
		slog.Error("Failed to get snapshot info", "vmid", vmid, "snapshot", snapshotID, "error", err)
		return nil, nil, fmt.Errorf("failed to get snapshot info: %w", err)
	}

	// Build snapshot tree for context
	tree, err := s.buildSnapshotTree(ctx, vmid)
	if err != nil {
		slog.Warn("Failed to build snapshot tree", "vmid", vmid, "error", err)
	}

	return s.modelToProto(libvirtSnapshot), tree, nil
}

// modelToProto converts a model.Snapshot to labv1.Snapshot
func (s *SnapshotService) modelToProto(snap *model.Snapshot) *labv1.Snapshot {
	if snap == nil {
		return nil
	}

	status := labv1.SnapshotStatus_SNAPSHOT_STATUS_READY
	switch snap.Status {
	case model.SnapshotStatusCreating:
		status = labv1.SnapshotStatus_SNAPSHOT_STATUS_CREATING
	case model.SnapshotStatusReady:
		status = labv1.SnapshotStatus_SNAPSHOT_STATUS_READY
	case model.SnapshotStatusDeleting:
		status = labv1.SnapshotStatus_SNAPSHOT_STATUS_DELETING
	case model.SnapshotStatusError:
		status = labv1.SnapshotStatus_SNAPSHOT_STATUS_ERROR
	}

	return &labv1.Snapshot{
		Id:          snap.ID,
		Vmid:        int32(snap.VMID),
		Name:        snap.Name,
		Description: snap.Description,
		CreatedAt:   snap.CreatedAt,
		ParentId:    snap.ParentID,
		SizeBytes:   snap.SizeBytes,
		Status:      status,
		VmState:     string(snap.VMState),
		HasChildren: snap.HasChildren,
	}
}

// buildSnapshotTree builds a hierarchical snapshot tree
func (s *SnapshotService) buildSnapshotTree(ctx context.Context, vmid int) (*labv1.SnapshotTree, error) {
	libvirtTree, err := s.snapshotLib.GetSnapshotTree(vmid)
	if err != nil {
		return nil, err
	}

	if libvirtTree == nil {
		return nil, nil
	}

	return s.libvirtTreeToProto(libvirtTree), nil
}

// libvirtTreeToProto converts libvirt snapshot tree to proto
func (s *SnapshotService) libvirtTreeToProto(tree *model.SnapshotTree) *labv1.SnapshotTree {
	if tree == nil {
		return nil
	}

	protoTree := &labv1.SnapshotTree{
		Snapshot: s.modelToProto(tree.Snapshot),
		Children: make([]*labv1.SnapshotTree, len(tree.Children)),
	}

	for i, child := range tree.Children {
		protoTree.Children[i] = s.libvirtTreeToProto(child)
	}

	return protoTree
}
