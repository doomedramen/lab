package model

import "time"

// TaskType represents the type of async operation
type TaskType string

const (
	TaskTypeBackup           TaskType = "backup"
	TaskTypeRestore          TaskType = "restore"
	TaskTypeSnapshotCreate   TaskType = "snapshot_create"
	TaskTypeSnapshotDelete   TaskType = "snapshot_delete"
	TaskTypeSnapshotRestore  TaskType = "snapshot_restore"
	TaskTypeClone            TaskType = "clone"
	TaskTypeMigration        TaskType = "migration"
	TaskTypeImport           TaskType = "import"
	TaskTypeExport           TaskType = "export"
)

// TaskStatus represents the current status of a task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// ResourceType represents the type of resource a task operates on
type ResourceType string

const (
	ResourceTypeVM         ResourceType = "vm"
	ResourceTypeContainer  ResourceType = "container"
	ResourceTypeStack      ResourceType = "stack"
	ResourceTypeBackup     ResourceType = "backup"
	ResourceTypeSnapshot   ResourceType = "snapshot"
	ResourceTypeISO        ResourceType = "iso"
	ResourceTypeNetwork    ResourceType = "network"
	ResourceTypeStorage    ResourceType = "storage"
)

// Task represents an async operation that can be tracked
type Task struct {
	ID           string    `json:"id"`
	Type         TaskType `json:"type"`
	Status       TaskStatus `json:"status"`
	Progress     int       `json:"progress"` // 0-100
	Message      string       `json:"message"` // Human-readable status
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Error        string       `json:"error,omitempty"`
}


// TaskCreateRequest represents a request to create a new task
type TaskCreateRequest struct {
	Type         TaskType     `json:"type"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
	Message      string       `json:"message,omitempty"`
}

// TaskFilter represents filters for listing tasks
type TaskFilter struct {
	Status       TaskStatus   `json:"status,omitempty"`
	Type         TaskType     `json:"type,omitempty"`
	ResourceType ResourceType `json:"resource_type,omitempty"`
	ResourceID   string       `json:"resource_id,omitempty"`
	ActiveOnly   bool         `json:"active_only,omitempty"`
}

// IsTerminal returns true if the task is in a terminal state (completed, failed, or cancelled)
func (t *Task) IsTerminal() bool {
	return t.Status == TaskStatusCompleted || t.Status == TaskStatusFailed || t.Status == TaskStatusCancelled
}
