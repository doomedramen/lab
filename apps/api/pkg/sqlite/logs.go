package sqlite

import (
	"context"
	"encoding/json"
	"time"
)

// VMLogEntry represents a VM log entry in the database
type VMLogEntry struct {
	ID        int64             `json:"id"`
	VMID      int               `json:"vmid"`
	Level     string            `json:"level"`
	Source    string            `json:"source"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt int64             `json:"created_at"`
}

// VMLogQuery holds query parameters for filtering VM logs
type VMLogQuery struct {
	VMID      int
	Level     string
	Source    string
	StartTime int64
	EndTime   int64
	Limit     int
}

// VMLogRepository provides methods for storing and querying VM logs
type VMLogRepository struct {
	db *DB
}

// NewVMLogRepository creates a new VM log repository
func NewVMLogRepository(db *DB) *VMLogRepository {
	return &VMLogRepository{db: db}
}

// Insert saves a VM log entry
func (r *VMLogRepository) Insert(ctx context.Context, entry *VMLogEntry) error {
	var metadataJSON *string
	if entry.Metadata != nil && len(entry.Metadata) > 0 {
		data, err := json.Marshal(entry.Metadata)
		if err != nil {
			return err
		}
		s := string(data)
		metadataJSON = &s
	}

	// Use database DEFAULT for created_at if not set
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vm_logs (vmid, level, source, message, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, COALESCE(NULLIF(?, 0), (strftime('%s', 'now'))))
	`, entry.VMID, entry.Level, entry.Source, entry.Message, metadataJSON, entry.CreatedAt)
	return err
}

// InsertBatch saves multiple VM log entries in a single transaction
func (r *VMLogRepository) InsertBatch(ctx context.Context, entries []*VMLogEntry) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO vm_logs (vmid, level, source, message, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, COALESCE(NULLIF(?, 0), (strftime('%s', 'now'))))
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		var metadataJSON *string
		if e.Metadata != nil && len(e.Metadata) > 0 {
			data, err := json.Marshal(e.Metadata)
			if err != nil {
				return err
			}
			s := string(data)
			metadataJSON = &s
		}

		_, err := stmt.ExecContext(ctx, e.VMID, e.Level, e.Source, e.Message, metadataJSON, e.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query retrieves VM logs based on query parameters
func (r *VMLogRepository) Query(ctx context.Context, q VMLogQuery) ([]*VMLogEntry, error) {
	query := `SELECT id, vmid, level, source, message, metadata, created_at FROM vm_logs WHERE 1=1`
	args := []interface{}{}

	if q.VMID > 0 {
		query += " AND vmid = ?"
		args = append(args, q.VMID)
	}

	if q.Level != "" {
		query += " AND level = ?"
		args = append(args, q.Level)
	}

	if q.Source != "" {
		query += " AND source = ?"
		args = append(args, q.Source)
	}

	if q.StartTime > 0 {
		query += " AND created_at >= ?"
		args = append(args, q.StartTime)
	}

	if q.EndTime > 0 {
		query += " AND created_at <= ?"
		args = append(args, q.EndTime)
	}

	query += " ORDER BY created_at DESC"

	if q.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, q.Limit)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*VMLogEntry
	for rows.Next() {
		var e VMLogEntry
		var metadataJSON *string

		err := rows.Scan(&e.ID, &e.VMID, &e.Level, &e.Source, &e.Message, &metadataJSON, &e.CreatedAt)
		if err != nil {
			return nil, err
		}

		if metadataJSON != nil {
			var metadata map[string]string
			if err := json.Unmarshal([]byte(*metadataJSON), &metadata); err == nil {
				e.Metadata = metadata
			}
		}

		entries = append(entries, &e)
	}

	return entries, rows.Err()
}

// DeleteOld deletes VM logs older than the specified number of days
func (r *VMLogRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -days).Unix()

	result, err := r.db.ExecContext(ctx, `
		DELETE FROM vm_logs WHERE created_at < ?
	`, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Count returns the total number of VM logs
func (r *VMLogRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vm_logs").Scan(&count)
	return count, err
}

// CountByVMID returns the number of VM logs for a specific VM
func (r *VMLogRepository) CountByVMID(ctx context.Context, vmid int) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vm_logs WHERE vmid = ?", vmid).Scan(&count)
	return count, err
}

// DeleteByMessage deletes VM logs with a specific message
// Used for cleaning up spam logs
func (r *VMLogRepository) DeleteByMessage(ctx context.Context, message string) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM vm_logs WHERE message = ?
	`, message)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DeleteOldEpoch deletes VM logs with epoch timestamps (created_at = 0 or before year 2000)
// These are logs created before the timestamp fix
func (r *VMLogRepository) DeleteOldEpoch(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM vm_logs WHERE created_at = 0 OR created_at < 946684800
	`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
