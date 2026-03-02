package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// TaskRepository handles task storage and retrieval
type TaskRepository struct {
	db *sqlitePkg.DB
}

// NewTaskRepository creates a new task repository
func NewTaskRepository(db *sqlitePkg.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create saves a new task to the database
func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	query := `
		INSERT INTO tasks (
			id, type, status, progress, message, resource_type,
			resource_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		task.ID,
		string(task.Type),
		string(task.Status),
		task.Progress,
		task.Message,
		string(task.ResourceType),
		task.ResourceID,
		task.CreatedAt.Format(time.RFC3339),
		task.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*model.Task, error) {
	query := `
		SELECT id, type, status, progress, message, resource_type,
		       resource_id, created_at, updated_at, completed_at, error
		FROM tasks
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanTask(row)
}

// List retrieves tasks with optional filters
func (r *TaskRepository) List(ctx context.Context, filter model.TaskFilter) ([]*model.Task, error) {
	query := `
		SELECT id, type, status, progress, message, resource_type,
		       resource_id, created_at, updated_at, completed_at, error
		FROM tasks
		WHERE 1=1
	`

	args := []interface{}{}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, string(filter.Type))
	}
	if filter.ResourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, string(filter.ResourceType))
	}
	if filter.ResourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, filter.ResourceID)
	}
	if filter.ActiveOnly {
		query += " AND status IN ('pending', 'running')"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*model.Task
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tasks: %w", err)
	}

	return tasks, nil
}

// Update updates an existing task
func (r *TaskRepository) Update(ctx context.Context, task *model.Task) error {
	query := `
		UPDATE tasks
		SET status = ?, progress = ?, message = ?, updated_at = ?,
		    completed_at = ?, error = ?
		WHERE id = ?
	`

	var completedAt, errorMsg sql.NullString
	if task.CompletedAt != nil {
		completedAt = sql.NullString{String: task.CompletedAt.Format(time.RFC3339), Valid: true}
	}
	if task.Error != "" {
		errorMsg = sql.NullString{String: task.Error, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		string(task.Status),
		task.Progress,
		task.Message,
		task.UpdatedAt.Format(time.RFC3339),
		completedAt,
		errorMsg,
		task.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// UpdateProgress updates the progress and message of a task
func (r *TaskRepository) UpdateProgress(ctx context.Context, id string, progress int, message string) error {
	query := `UPDATE tasks SET progress = ?, message = ?, updated_at = datetime('now') WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, progress, message, id)
	if err != nil {
		return fmt.Errorf("failed to update task progress: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of a task
func (r *TaskRepository) UpdateStatus(ctx context.Context, id string, status model.TaskStatus, message string) error {
	var completedAt string
	if status == model.TaskStatusCompleted || status == model.TaskStatusFailed || status == model.TaskStatusCancelled {
		completedAt = "datetime('now')"
	}

	query := fmt.Sprintf(`
		UPDATE tasks
		SET status = ?, message = ?, updated_at = datetime('now')%s
		WHERE id = ?
	`, ternary(completedAt != "", ", completed_at = "+completedAt, ""))

	_, err := r.db.ExecContext(ctx, query, string(status), message, id)
	if err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// Delete removes a task from the database
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	return nil
}

// DeleteCompleted removes completed tasks older than the specified duration
func (r *TaskRepository) DeleteCompleted(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
	query := `
		DELETE FROM tasks
		WHERE status IN ('completed', 'failed', 'cancelled')
		  AND completed_at IS NOT NULL
		  AND datetime(completed_at) < datetime(?)
	`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete completed tasks: %w", err)
	}

	return result.RowsAffected()
}

// scanTask scans a single row into a Task struct
func scanTask(row scanner) (*model.Task, error) {
	var t model.Task
	var completedAt, errorMsg sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&t.ID,
		&t.Type,
		&t.Status,
		&t.Progress,
		&t.Message,
		&t.ResourceType,
		&t.ResourceID,
		&createdAtStr,
		&updatedAtStr,
		&completedAt,
		&errorMsg,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan task: %w", err)
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if completedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			t.CompletedAt = &parsedTime
		}
	}
	t.Error = errorMsg.String

	return &t, nil
}

// scanTaskRow scans a row from Rows into a Task struct
func scanTaskRow(rows rowsScanner) (*model.Task, error) {
	var t model.Task
	var completedAt, errorMsg sql.NullString
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&t.ID,
		&t.Type,
		&t.Status,
		&t.Progress,
		&t.Message,
		&t.ResourceType,
		&t.ResourceID,
		&createdAtStr,
		&updatedAtStr,
		&completedAt,
		&errorMsg,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan task row: %w", err)
	}

	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if completedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, completedAt.String); err == nil {
			t.CompletedAt = &parsedTime
		}
	}
	t.Error = errorMsg.String

	return &t, nil
}

// ternary returns a if cond is true, otherwise b
func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
