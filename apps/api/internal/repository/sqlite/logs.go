package sqlite

import (
	"context"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// VMLogRepository handles VM log storage and retrieval
type VMLogRepository struct {
	repo *sqlitePkg.VMLogRepository
}

// NewVMLogRepository creates a new VM log repository
func NewVMLogRepository(db *sqlitePkg.DB) *VMLogRepository {
	return &VMLogRepository{repo: sqlitePkg.NewVMLogRepository(db)}
}

// Record saves a VM log entry
func (r *VMLogRepository) Record(ctx context.Context, vmid int, level, source, message string, metadata map[string]string) error {
	entry := &sqlitePkg.VMLogEntry{
		VMID:      vmid,
		Level:     level,
		Source:    source,
		Message:   message,
		Metadata:  metadata,
		CreatedAt: 0, // Will be set by the repository
	}
	return r.repo.Insert(ctx, entry)
}

// RecordBatch saves multiple VM log entries
func (r *VMLogRepository) RecordBatch(ctx context.Context, vmid int, entries []*model.VMLogEntry) error {
	pkgEntries := make([]*sqlitePkg.VMLogEntry, len(entries))
	for i, e := range entries {
		pkgEntries[i] = &sqlitePkg.VMLogEntry{
			VMID:      vmid,
			Level:     e.Level,
			Source:    e.Source,
			Message:   e.Message,
			Metadata:  e.Metadata,
			CreatedAt: 0,
		}
	}
	return r.repo.InsertBatch(ctx, pkgEntries)
}

// Query retrieves VM logs based on query parameters
func (r *VMLogRepository) Query(ctx context.Context, vmid int, limit int) ([]*model.VMLogEntry, error) {
	entries, err := r.repo.Query(ctx, sqlitePkg.VMLogQuery{
		VMID:  vmid,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	var result []*model.VMLogEntry
	for _, e := range entries {
		result = append(result, &model.VMLogEntry{
			ID:        e.ID,
			VMID:      e.VMID,
			Level:     e.Level,
			Source:    e.Source,
			Message:   e.Message,
			Metadata:  e.Metadata,
			CreatedAt: e.CreatedAt,
		})
	}

	return result, nil
}

// DeleteOld deletes VM logs older than the specified number of days
func (r *VMLogRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	return r.repo.DeleteOld(ctx, days)
}

// Count returns the total number of VM logs
func (r *VMLogRepository) Count(ctx context.Context) (int64, error) {
	return r.repo.Count(ctx)
}

// DeleteByMessage deletes VM logs with a specific message
func (r *VMLogRepository) DeleteByMessage(ctx context.Context, message string) (int64, error) {
	return r.repo.DeleteByMessage(ctx, message)
}

// DeleteOldEpoch deletes VM logs with epoch timestamps (created_at = 0 or before year 2000)
// These are logs created before the timestamp fix
func (r *VMLogRepository) DeleteOldEpoch(ctx context.Context) (int64, error) {
	return r.repo.DeleteOldEpoch(ctx)
}
