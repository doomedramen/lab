package handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TaskServiceServer implements the TaskService Connect RPC server
type TaskServiceServer struct {
	taskSvc *service.TaskService
}

// NewTaskServiceServer creates a new task service server
func NewTaskServiceServer(taskSvc *service.TaskService) *TaskServiceServer {
	return &TaskServiceServer{taskSvc: taskSvc}
}

// compile-time check that we implement the interface
var _ labv1connect.TaskServiceHandler = (*TaskServiceServer)(nil)

// ListTasks returns all tasks with optional filters
func (s *TaskServiceServer) ListTasks(ctx context.Context, req *connect.Request[labv1.ListTasksRequest]) (*connect.Response[labv1.ListTasksResponse], error) {
	filter := model.TaskFilter{
		Status:       protoToTaskStatus(req.Msg.Status),
		Type:         protoToTaskType(req.Msg.Type),
		ResourceType: protoToResourceType(req.Msg.ResourceType),
		ResourceID:   req.Msg.ResourceId,
		ActiveOnly:   req.Msg.ActiveOnly,
	}

	tasks, err := s.taskSvc.List(ctx, filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list tasks: %w", err))
	}

	var protoTasks []*labv1.Task
	for _, task := range tasks {
		protoTasks = append(protoTasks, modelTaskToProto(task))
	}

	return connect.NewResponse(&labv1.ListTasksResponse{
		Tasks: protoTasks,
	}), nil
}

// GetTask returns details of a specific task
func (s *TaskServiceServer) GetTask(ctx context.Context, req *connect.Request[labv1.GetTaskRequest]) (*connect.Response[labv1.GetTaskResponse], error) {
	task, err := s.taskSvc.Get(ctx, req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get task: %w", err))
	}
	if task == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found: %s", req.Msg.TaskId))
	}

	return connect.NewResponse(&labv1.GetTaskResponse{
		Task: modelTaskToProto(task),
	}), nil
}

// CancelTask cancels a running task
func (s *TaskServiceServer) CancelTask(ctx context.Context, req *connect.Request[labv1.CancelTaskRequest]) (*connect.Response[labv1.CancelTaskResponse], error) {
	if err := s.taskSvc.Cancel(ctx, req.Msg.TaskId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel task: %w", err))
	}

	task, err := s.taskSvc.Get(ctx, req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get task after cancel: %w", err))
	}

	return connect.NewResponse(&labv1.CancelTaskResponse{
		Task: modelTaskToProto(task),
	}), nil
}

// modelTaskToProto converts model.Task to labv1.Task
func modelTaskToProto(task *model.Task) *labv1.Task {
	if task == nil {
		return nil
	}

	protoTask := &labv1.Task{
		Id:           task.ID,
		Type:         modelTaskTypeToProto(task.Type),
		Status:       modelTaskStatusToProto(task.Status),
		Progress:     int32(task.Progress),
		Message:      task.Message,
		ResourceType: modelResourceTypeToProto(task.ResourceType),
		ResourceId:   task.ResourceID,
		CreatedAt:    timestamppb.New(task.CreatedAt),
		UpdatedAt:    timestamppb.New(task.UpdatedAt),
		Error:        task.Error,
	}
	if task.CompletedAt != nil {
		protoTask.CompletedAt = timestamppb.New(*task.CompletedAt)
	}
	return protoTask
}

// protoToTaskStatus converts labv1.TaskStatus to model.TaskStatus
func protoToTaskStatus(s labv1.TaskStatus) model.TaskStatus {
	switch s {
	case labv1.TaskStatus_TASK_STATUS_PENDING:
		return model.TaskStatusPending
	case labv1.TaskStatus_TASK_STATUS_RUNNING:
		return model.TaskStatusRunning
	case labv1.TaskStatus_TASK_STATUS_COMPLETED:
		return model.TaskStatusCompleted
	case labv1.TaskStatus_TASK_STATUS_FAILED:
		return model.TaskStatusFailed
	case labv1.TaskStatus_TASK_STATUS_CANCELLED:
		return model.TaskStatusCancelled
	default:
		return ""
	}
}

// modelTaskStatusToProto converts model.TaskStatus to labv1.TaskStatus
func modelTaskStatusToProto(s model.TaskStatus) labv1.TaskStatus {
	switch s {
	case model.TaskStatusPending:
		return labv1.TaskStatus_TASK_STATUS_PENDING
	case model.TaskStatusRunning:
		return labv1.TaskStatus_TASK_STATUS_RUNNING
	case model.TaskStatusCompleted:
		return labv1.TaskStatus_TASK_STATUS_COMPLETED
	case model.TaskStatusFailed:
		return labv1.TaskStatus_TASK_STATUS_FAILED
	case model.TaskStatusCancelled:
		return labv1.TaskStatus_TASK_STATUS_CANCELLED
	default:
		return labv1.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// protoToTaskType converts labv1.TaskType to model.TaskType
func protoToTaskType(t labv1.TaskType) model.TaskType {
	switch t {
	case labv1.TaskType_TASK_TYPE_BACKUP:
		return model.TaskTypeBackup
	case labv1.TaskType_TASK_TYPE_RESTORE:
		return model.TaskTypeRestore
	case labv1.TaskType_TASK_TYPE_SNAPSHOT_CREATE:
		return model.TaskTypeSnapshotCreate
	case labv1.TaskType_TASK_TYPE_SNAPSHOT_DELETE:
		return model.TaskTypeSnapshotDelete
	case labv1.TaskType_TASK_TYPE_SNAPSHOT_RESTORE:
		return model.TaskTypeSnapshotRestore
	case labv1.TaskType_TASK_TYPE_CLONE:
		return model.TaskTypeClone
	case labv1.TaskType_TASK_TYPE_MIGRATION:
		return model.TaskTypeMigration
	case labv1.TaskType_TASK_TYPE_IMPORT:
		return model.TaskTypeImport
	case labv1.TaskType_TASK_TYPE_EXPORT:
		return model.TaskTypeExport
	default:
		return ""
	}
}

// modelTaskTypeToProto converts model.TaskType to labv1.TaskType
func modelTaskTypeToProto(t model.TaskType) labv1.TaskType {
	switch t {
	case model.TaskTypeBackup:
		return labv1.TaskType_TASK_TYPE_BACKUP
	case model.TaskTypeRestore:
		return labv1.TaskType_TASK_TYPE_RESTORE
	case model.TaskTypeSnapshotCreate:
		return labv1.TaskType_TASK_TYPE_SNAPSHOT_CREATE
	case model.TaskTypeSnapshotDelete:
		return labv1.TaskType_TASK_TYPE_SNAPSHOT_DELETE
	case model.TaskTypeSnapshotRestore:
		return labv1.TaskType_TASK_TYPE_SNAPSHOT_RESTORE
	case model.TaskTypeClone:
		return labv1.TaskType_TASK_TYPE_CLONE
	case model.TaskTypeMigration:
		return labv1.TaskType_TASK_TYPE_MIGRATION
	case model.TaskTypeImport:
		return labv1.TaskType_TASK_TYPE_IMPORT
	case model.TaskTypeExport:
		return labv1.TaskType_TASK_TYPE_EXPORT
	default:
		return labv1.TaskType_TASK_TYPE_UNSPECIFIED
	}
}

// protoToResourceType converts labv1.ResourceType to model.ResourceType
func protoToResourceType(t labv1.ResourceType) model.ResourceType {
	switch t {
	case labv1.ResourceType_RESOURCE_TYPE_VM:
		return model.ResourceTypeVM
	case labv1.ResourceType_RESOURCE_TYPE_CONTAINER:
		return model.ResourceTypeContainer
	case labv1.ResourceType_RESOURCE_TYPE_STACK:
		return model.ResourceTypeStack
	case labv1.ResourceType_RESOURCE_TYPE_BACKUP:
		return model.ResourceTypeBackup
	case labv1.ResourceType_RESOURCE_TYPE_SNAPSHOT:
		return model.ResourceTypeSnapshot
	case labv1.ResourceType_RESOURCE_TYPE_ISO:
		return model.ResourceTypeISO
	case labv1.ResourceType_RESOURCE_TYPE_NETWORK:
		return model.ResourceTypeNetwork
	case labv1.ResourceType_RESOURCE_TYPE_STORAGE:
		return model.ResourceTypeStorage
	default:
		return ""
	}
}

// modelResourceTypeToProto converts model.ResourceType to labv1.ResourceType
func modelResourceTypeToProto(t model.ResourceType) labv1.ResourceType {
	switch t {
	case model.ResourceTypeVM:
		return labv1.ResourceType_RESOURCE_TYPE_VM
	case model.ResourceTypeContainer:
		return labv1.ResourceType_RESOURCE_TYPE_CONTAINER
	case model.ResourceTypeStack:
		return labv1.ResourceType_RESOURCE_TYPE_STACK
	case model.ResourceTypeBackup:
		return labv1.ResourceType_RESOURCE_TYPE_BACKUP
	case model.ResourceTypeSnapshot:
		return labv1.ResourceType_RESOURCE_TYPE_SNAPSHOT
	case model.ResourceTypeISO:
		return labv1.ResourceType_RESOURCE_TYPE_ISO
	case model.ResourceTypeNetwork:
		return labv1.ResourceType_RESOURCE_TYPE_NETWORK
	case model.ResourceTypeStorage:
		return labv1.ResourceType_RESOURCE_TYPE_STORAGE
	default:
		return labv1.ResourceType_RESOURCE_TYPE_UNSPECIFIED
	}
}