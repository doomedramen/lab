package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// BackupRepository handles backup metadata storage and retrieval
type BackupRepository struct {
	db *sqlitePkg.DB
}

// NewBackupRepository creates a new backup repository
func NewBackupRepository(db *sqlitePkg.DB) *BackupRepository {
	return &BackupRepository{db: db}
}

// Create saves a new backup to the database
func (r *BackupRepository) Create(ctx context.Context, backup *model.Backup) error {
	query := `
		INSERT INTO backups (
			id, vmid, vm_name, name, type, status, size_bytes,
			storage_pool, backup_path, created_at, retention_days
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		backup.ID,
		backup.VMID,
		backup.VMName,
		backup.Name,
		string(backup.Type),
		string(backup.Status),
		backup.SizeBytes,
		backup.StoragePool,
		nullString(backup.BackupPath),
		backup.CreatedAt,
		backup.RetentionDays,
	)

	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// GetByID retrieves a backup by its ID
func (r *BackupRepository) GetByID(ctx context.Context, id string) (*model.Backup, error) {
	query := `
		SELECT id, vmid, vm_name, name, type, status, size_bytes,
		       storage_pool, backup_path, created_at, completed_at,
		       expires_at, error_message, retention_days
		FROM backups
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanBackup(row)
}

// List retrieves backups with optional filters
func (r *BackupRepository) List(ctx context.Context, vmid int, status model.BackupStatus, storagePool string) ([]*model.Backup, error) {
	query := `
		SELECT id, vmid, vm_name, name, type, status, size_bytes,
		       storage_pool, backup_path, created_at, completed_at,
		       expires_at, error_message, retention_days
		FROM backups
		WHERE 1=1
	`

	args := []interface{}{}
	if vmid > 0 {
		query += " AND vmid = ?"
		args = append(args, vmid)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}
	if storagePool != "" {
		query += " AND storage_pool = ?"
		args = append(args, storagePool)
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	defer rows.Close()

	var backups []*model.Backup
	for rows.Next() {
		backup, err := scanBackupRow(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating backups: %w", err)
	}

	return backups, nil
}

// Update updates an existing backup
func (r *BackupRepository) Update(ctx context.Context, backup *model.Backup) error {
	query := `
		UPDATE backups
		SET status = ?, size_bytes = ?, completed_at = ?,
		    expires_at = ?, error_message = ?, backup_path = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		string(backup.Status),
		backup.SizeBytes,
		nullString(backup.CompletedAt),
		nullString(backup.ExpiresAt),
		nullString(backup.ErrorMessage),
		nullString(backup.BackupPath),
		backup.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update backup: %w", err)
	}

	return nil
}

// UpdateStatus updates the status of a backup
func (r *BackupRepository) UpdateStatus(ctx context.Context, id string, status model.BackupStatus, errorMessage string) error {
	query := `UPDATE backups SET status = ?, error_message = ? WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, string(status), nullString(errorMessage), id)
	if err != nil {
		return fmt.Errorf("failed to update backup status: %w", err)
	}

	return nil
}

// Delete removes a backup from the database
func (r *BackupRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM backups WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	return nil
}

// GetExpired returns backups that have passed their retention period
func (r *BackupRepository) GetExpired(ctx context.Context) ([]*model.Backup, error) {
	query := `
		SELECT id, vmid, vm_name, name, type, status, size_bytes,
		       storage_pool, backup_path, created_at, completed_at,
		       expires_at, error_message, retention_days
		FROM backups
		WHERE expires_at IS NOT NULL 
		  AND expires_at != ''
		  AND datetime(expires_at) < datetime('now')
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired backups: %w", err)
	}
	defer rows.Close()

	var backups []*model.Backup
	for rows.Next() {
		backup, err := scanBackupRow(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}

	return backups, rows.Err()
}

// BackupScheduleRepository handles backup schedule storage and retrieval
type BackupScheduleRepository struct {
	db *sqlitePkg.DB
}

// NewBackupScheduleRepository creates a new backup schedule repository
func NewBackupScheduleRepository(db *sqlitePkg.DB) *BackupScheduleRepository {
	return &BackupScheduleRepository{db: db}
}

// Create saves a new backup schedule to the database
func (r *BackupScheduleRepository) Create(ctx context.Context, schedule *model.BackupSchedule) error {
	query := `
		INSERT INTO backup_schedules (
			id, name, entity_type, entity_id, storage_pool,
			schedule, backup_type, retention_days, enabled,
			created_at, updated_at, next_run_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.Name,
		schedule.EntityType,
		schedule.EntityID,
		schedule.StoragePool,
		schedule.Schedule,
		string(schedule.BackupType),
		schedule.RetentionDays,
		boolToInt(schedule.Enabled),
		schedule.CreatedAt,
		schedule.UpdatedAt,
		nullString(schedule.NextRunAt),
	)

	if err != nil {
		return fmt.Errorf("failed to create backup schedule: %w", err)
	}

	return nil
}

// GetByID retrieves a backup schedule by its ID
func (r *BackupScheduleRepository) GetByID(ctx context.Context, id string) (*model.BackupSchedule, error) {
	query := `
		SELECT id, name, entity_type, entity_id, storage_pool,
		       schedule, backup_type, retention_days, enabled,
		       created_at, updated_at, last_run_at, next_run_at, total_backups
		FROM backup_schedules
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanBackupSchedule(row)
}

// List retrieves backup schedules with optional filters
func (r *BackupScheduleRepository) List(ctx context.Context, entityType string, entityID int) ([]*model.BackupSchedule, error) {
	query := `
		SELECT id, name, entity_type, entity_id, storage_pool,
		       schedule, backup_type, retention_days, enabled,
		       created_at, updated_at, last_run_at, next_run_at, total_backups
		FROM backup_schedules
		WHERE 1=1
	`

	args := []interface{}{}
	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}
	if entityID > 0 {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list backup schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*model.BackupSchedule
	for rows.Next() {
		schedule, err := scanBackupScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	return schedules, rows.Err()
}

// Update updates an existing backup schedule
func (r *BackupScheduleRepository) Update(ctx context.Context, schedule *model.BackupSchedule) error {
	query := `
		UPDATE backup_schedules
		SET name = ?, storage_pool = ?, schedule = ?, backup_type = ?,
		    retention_days = ?, enabled = ?, updated_at = ?,
		    last_run_at = ?, next_run_at = ?, total_backups = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		schedule.Name,
		schedule.StoragePool,
		schedule.Schedule,
		string(schedule.BackupType),
		schedule.RetentionDays,
		boolToInt(schedule.Enabled),
		schedule.UpdatedAt,
		nullString(schedule.LastRunAt),
		nullString(schedule.NextRunAt),
		schedule.TotalBackups,
		schedule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update backup schedule: %w", err)
	}

	return nil
}

// UpdateRunInfo updates the last run and next run times for a schedule
func (r *BackupScheduleRepository) UpdateRunInfo(ctx context.Context, id string, lastRunAt, nextRunAt string, incrementBackups bool) error {
	increment := ""
	if incrementBackups {
		increment = ", total_backups = total_backups + 1"
	}

	query := fmt.Sprintf(`
		UPDATE backup_schedules
		SET last_run_at = ?, next_run_at = ?, updated_at = datetime('now')%s
		WHERE id = ?
	`, increment)

	_, err := r.db.ExecContext(ctx, query,
		nullString(lastRunAt),
		nullString(nextRunAt),
		id,
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule run info: %w", err)
	}

	return nil
}

// Delete removes a backup schedule from the database
func (r *BackupScheduleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM backup_schedules WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete backup schedule: %w", err)
	}

	return nil
}

// GetDueSchedules returns schedules that are due to run
func (r *BackupScheduleRepository) GetDueSchedules(ctx context.Context) ([]*model.BackupSchedule, error) {
	query := `
		SELECT id, name, entity_type, entity_id, storage_pool,
		       schedule, backup_type, retention_days, enabled,
		       created_at, updated_at, last_run_at, next_run_at, total_backups
		FROM backup_schedules
		WHERE enabled = 1
		  AND next_run_at IS NOT NULL
		  AND datetime(next_run_at) <= datetime('now')
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []*model.BackupSchedule
	for rows.Next() {
		schedule, err := scanBackupScheduleRow(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	return schedules, rows.Err()
}

// scanBackup scans a single row into a Backup struct
func scanBackup(row scanner) (*model.Backup, error) {
	var b model.Backup
	var completedAt, expiresAt, backupPath, errorMessage sql.NullString

	err := row.Scan(
		&b.ID,
		&b.VMID,
		&b.VMName,
		&b.Name,
		&b.Type,
		&b.Status,
		&b.SizeBytes,
		&b.StoragePool,
		&backupPath,
		&b.CreatedAt,
		&completedAt,
		&expiresAt,
		&errorMessage,
		&b.RetentionDays,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan backup: %w", err)
	}

	b.CompletedAt = completedAt.String
	b.ExpiresAt = expiresAt.String
	b.BackupPath = backupPath.String
	b.ErrorMessage = errorMessage.String

	return &b, nil
}

// scanBackupRow scans a row from Rows into a Backup struct
func scanBackupRow(rows rowsScanner) (*model.Backup, error) {
	var b model.Backup
	var completedAt, expiresAt, backupPath, errorMessage sql.NullString

	err := rows.Scan(
		&b.ID,
		&b.VMID,
		&b.VMName,
		&b.Name,
		&b.Type,
		&b.Status,
		&b.SizeBytes,
		&b.StoragePool,
		&backupPath,
		&b.CreatedAt,
		&completedAt,
		&expiresAt,
		&errorMessage,
		&b.RetentionDays,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan backup row: %w", err)
	}

	b.CompletedAt = completedAt.String
	b.ExpiresAt = expiresAt.String
	b.BackupPath = backupPath.String
	b.ErrorMessage = errorMessage.String

	return &b, nil
}

// scanBackupSchedule scans a single row into a BackupSchedule struct
func scanBackupSchedule(row scanner) (*model.BackupSchedule, error) {
	var s model.BackupSchedule
	var lastRunAt, nextRunAt sql.NullString

	err := row.Scan(
		&s.ID,
		&s.Name,
		&s.EntityType,
		&s.EntityID,
		&s.StoragePool,
		&s.Schedule,
		&s.BackupType,
		&s.RetentionDays,
		&s.Enabled,
		&s.CreatedAt,
		&s.UpdatedAt,
		&lastRunAt,
		&nextRunAt,
		&s.TotalBackups,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan backup schedule: %w", err)
	}

	s.LastRunAt = lastRunAt.String
	s.NextRunAt = nextRunAt.String

	return &s, nil
}

// scanBackupScheduleRow scans a row from Rows into a BackupSchedule struct
func scanBackupScheduleRow(rows rowsScanner) (*model.BackupSchedule, error) {
	var s model.BackupSchedule
	var lastRunAt, nextRunAt sql.NullString

	err := rows.Scan(
		&s.ID,
		&s.Name,
		&s.EntityType,
		&s.EntityID,
		&s.StoragePool,
		&s.Schedule,
		&s.BackupType,
		&s.RetentionDays,
		&s.Enabled,
		&s.CreatedAt,
		&s.UpdatedAt,
		&lastRunAt,
		&nextRunAt,
		&s.TotalBackups,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan backup schedule row: %w", err)
	}

	s.LastRunAt = lastRunAt.String
	s.NextRunAt = nextRunAt.String

	return &s, nil
}
