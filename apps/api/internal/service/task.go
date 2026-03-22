package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/google/uuid"
)

// TaskService manages task tracking for operations
type TaskService struct {
	repo        repository.TaskRepository
	mu          sync.RWMutex
	tasks       map[string]*model.Task
	taskCounter int
}

// NewTaskService creates a new task service
func NewTaskService(repo repository.TaskRepository) *TaskService {
	return &TaskService{
		repo:        repo,
		mu:          sync.RWMutex{},
		tasks:       make(map[string]*model.Task),
		taskCounter: 0,
	}
}

// Start creates a new task and returns it
func (s *TaskService) Start(ctx context.Context, taskType model.TaskType, resourceType model.ResourceType, resourceID string, message string) (*model.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	task := &model.Task{
		ID:           uuid.New().String(),
		Type:         taskType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       model.TaskStatusPending,
		Message:      message,
		Progress:     0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		slog.Error("Failed to create task", "task_id", task.ID, "error", err)
		return nil, err
	}
	s.tasks[task.ID] = task
	s.taskCounter++

	return task, nil
}

// Progress updates the progress and message of a running task
func (s *TaskService) Progress(ctx context.Context, taskID string, progress int, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		return fmt.Errorf("cannot update progress on a task with status: %s", task.Status)
	}

	task.Progress = progress
	task.Message = message
	task.UpdatedAt = time.Now()

	if err := s.repo.UpdateProgress(ctx, taskID, progress, message); err != nil {
		slog.Error("Failed to update task progress", "task_id", taskID, "error", err)
	}
	return nil
}

// Complete marks a task as completed successfully
func (s *TaskService) Complete(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		return fmt.Errorf("cannot complete a task with status: %s", task.Status)
	}

	task.Status = model.TaskStatusCompleted
	task.Message = "Task completed"
	task.CompletedAt = timePtr(time.Now())
	task.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, task); err != nil {
		slog.Error("Failed to update task", "task_id", taskID, "error", err)
	}
	return nil
}

// Fail marks a task as failed
func (s *TaskService) Fail(ctx context.Context, taskID string, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		return fmt.Errorf("cannot fail a task with status: %s", task.Status)
	}

	task.Status = model.TaskStatusFailed
	task.Error = errMsg
	task.Message = fmt.Sprintf("Task failed: %s", errMsg)
	task.UpdatedAt = time.Now()
	task.CompletedAt = timePtr(time.Now())

	if err := s.repo.UpdateStatus(ctx, task.ID, model.TaskStatusFailed, task.Message); err != nil {
		slog.Error("Failed to update task status", "task_id", taskID, "error", err)
	}
	return nil
}

// Cancel marks a task as cancelled
func (s *TaskService) Cancel(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}
	if task.IsTerminal() {
		return fmt.Errorf("cannot cancel a task with status: %s", task.Status)
	}

	task.Status = model.TaskStatusCancelled
	task.Message = "Task cancelled"
	task.UpdatedAt = time.Now()
	task.CompletedAt = timePtr(time.Now())

	if err := s.repo.UpdateStatus(ctx, task.ID, model.TaskStatusCancelled, task.Message); err != nil {
		slog.Error("Failed to update task status", "task_id", taskID, "error", err)
	}
	return nil
}

// Get retrieves a task by ID
func (s *TaskService) Get(ctx context.Context, taskID string) (*model.Task, error) {
	return s.repo.GetByID(ctx, taskID)
}

// List retrieves tasks with optional filters
func (s *TaskService) List(ctx context.Context, filter model.TaskFilter) ([]*model.Task, error) {
	return s.repo.List(ctx, filter)
}

// Delete removes a task
func (s *TaskService) Delete(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tasks, taskID)
	return s.repo.Delete(ctx, taskID)
}

// Cleanup removes completed tasks older than the specified duration
func (s *TaskService) Cleanup(ctx context.Context, olderThan time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	count, err := s.repo.DeleteCompleted(ctx, olderThan)
	if err != nil {
		slog.Error("Failed to cleanup completed tasks", "error", err)
		return 0, err
	}
	return count, nil
}

// timePtr returns a pointer to a time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
