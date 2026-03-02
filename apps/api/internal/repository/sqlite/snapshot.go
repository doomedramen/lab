package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// SnapshotRepository handles snapshot metadata storage and retrieval
type SnapshotRepository struct {
	db *sqlitePkg.DB
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(db *sqlitePkg.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

// Create saves a new snapshot to the database
func (r *SnapshotRepository) Create(ctx context.Context, snapshot *model.Snapshot) error {
	query := `
		INSERT INTO vm_snapshots (
			id, vmid, name, description, created_at, parent_id,
			size_bytes, status, vm_state, has_children, snapshot_path
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		snapshot.ID,
		snapshot.VMID,
		snapshot.Name,
		snapshot.Description,
		snapshot.CreatedAt,
		nullString(snapshot.ParentID),
		snapshot.SizeBytes,
		string(snapshot.Status),
		string(snapshot.VMState),
		boolToInt(snapshot.HasChildren),
		nullString(snapshot.SnapshotPath),
	)

	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// GetByID retrieves a snapshot by its ID
func (r *SnapshotRepository) GetByID(ctx context.Context, vmid int, id string) (*model.Snapshot, error) {
	query := `
		SELECT id, vmid, name, description, created_at, parent_id,
		       size_bytes, status, vm_state, has_children, snapshot_path
		FROM vm_snapshots
		WHERE vmid = ? AND id = ?
	`

	row := r.db.QueryRowContext(ctx, query, vmid, id)
	return scanSnapshot(row)
}

// ListByVMID retrieves all snapshots for a VM
func (r *SnapshotRepository) ListByVMID(ctx context.Context, vmid int) ([]*model.Snapshot, error) {
	query := `
		SELECT id, vmid, name, description, created_at, parent_id,
		       size_bytes, status, vm_state, has_children, snapshot_path
		FROM vm_snapshots
		WHERE vmid = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, vmid)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*model.Snapshot
	for rows.Next() {
		snapshot, err := scanSnapshotRow(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating snapshots: %w", err)
	}

	return snapshots, nil
}

// Update updates an existing snapshot
func (r *SnapshotRepository) Update(ctx context.Context, snapshot *model.Snapshot) error {
	query := `
		UPDATE vm_snapshots
		SET name = ?, description = ?, size_bytes = ?, status = ?,
		    vm_state = ?, has_children = ?, snapshot_path = ?
		WHERE id = ? AND vmid = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		snapshot.Name,
		snapshot.Description,
		snapshot.SizeBytes,
		string(snapshot.Status),
		string(snapshot.VMState),
		boolToInt(snapshot.HasChildren),
		nullString(snapshot.SnapshotPath),
		snapshot.ID,
		snapshot.VMID,
	)

	if err != nil {
		return fmt.Errorf("failed to update snapshot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("snapshot not found: %s", snapshot.ID)
	}

	return nil
}

// Delete removes a snapshot from the database
func (r *SnapshotRepository) Delete(ctx context.Context, vmid int, id string) error {
	query := `DELETE FROM vm_snapshots WHERE vmid = ? AND id = ?`

	result, err := r.db.ExecContext(ctx, query, vmid, id)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("snapshot not found: %s", id)
	}

	return nil
}

// DeleteWithChildren removes a snapshot and all its children
func (r *SnapshotRepository) DeleteWithChildren(ctx context.Context, vmid int, id string) error {
	// First, get all children recursively
	children, err := r.getChildrenRecursive(ctx, vmid, id)
	if err != nil {
		return err
	}

	// Delete children first (in reverse order to delete leaves first)
	for i := len(children) - 1; i >= 0; i-- {
		if err := r.Delete(ctx, vmid, children[i]); err != nil {
			return fmt.Errorf("failed to delete child snapshot %s: %w", children[i], err)
		}
	}

	// Then delete the parent
	return r.Delete(ctx, vmid, id)
}

// getChildrenRecursive returns all child snapshot IDs recursively
func (r *SnapshotRepository) getChildrenRecursive(ctx context.Context, vmid int, parentID string) ([]string, error) {
	var children []string

	query := `SELECT id FROM vm_snapshots WHERE vmid = ? AND parent_id = ?`
	rows, err := r.db.QueryContext(ctx, query, vmid, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		children = append(children, id)

		// Recursively get grandchildren
		grandchildren, err := r.getChildrenRecursive(ctx, vmid, id)
		if err != nil {
			return nil, err
		}
		children = append(children, grandchildren...)
	}

	return children, rows.Err()
}

// UpdateStatus updates the status of a snapshot
func (r *SnapshotRepository) UpdateStatus(ctx context.Context, id string, vmid int, status model.SnapshotStatus) error {
	query := `UPDATE vm_snapshots SET status = ? WHERE id = ? AND vmid = ?`

	_, err := r.db.ExecContext(ctx, query, string(status), id, vmid)
	if err != nil {
		return fmt.Errorf("failed to update snapshot status: %w", err)
	}

	return nil
}

// UpdateSize updates the size of a snapshot
func (r *SnapshotRepository) UpdateSize(ctx context.Context, id string, vmid int, sizeBytes int64) error {
	query := `UPDATE vm_snapshots SET size_bytes = ? WHERE id = ? AND vmid = ?`

	_, err := r.db.ExecContext(ctx, query, sizeBytes, id, vmid)
	if err != nil {
		return fmt.Errorf("failed to update snapshot size: %w", err)
	}

	return nil
}

// Exists checks if a snapshot exists
func (r *SnapshotRepository) Exists(ctx context.Context, vmid int, id string) bool {
	query := `SELECT COUNT(*) FROM vm_snapshots WHERE vmid = ? AND id = ?`

	var count int
	err := r.db.QueryRowContext(ctx, query, vmid, id).Scan(&count)
	return err == nil && count > 0
}

// GetTree builds a snapshot tree for a VM
func (r *SnapshotRepository) GetTree(ctx context.Context, vmid int) (*model.SnapshotTree, error) {
	snapshots, err := r.ListByVMID(ctx, vmid)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	// Build a map for quick lookup
	snapshotMap := make(map[string]*model.SnapshotTree)
	for _, s := range snapshots {
		snapshotMap[s.ID] = &model.SnapshotTree{
			Snapshot: s,
			Children: []*model.SnapshotTree{},
		}
	}

	var root *model.SnapshotTree

	// Build the tree
	for _, s := range snapshots {
		if s.ParentID == "" {
			// This is a root node
			if root == nil {
				root = snapshotMap[s.ID]
			}
		} else {
			// Add as child to parent
			if parent, ok := snapshotMap[s.ParentID]; ok {
				parent.Children = append(parent.Children, snapshotMap[s.ID])
			}
		}
	}

	return root, nil
}

// scanSnapshot scans a single row into a Snapshot struct
func scanSnapshot(row scanner) (*model.Snapshot, error) {
	var s model.Snapshot
	var parentID sql.NullString
	var snapshotPath sql.NullString
	var hasChildren int

	err := row.Scan(
		&s.ID,
		&s.VMID,
		&s.Name,
		&s.Description,
		&s.CreatedAt,
		&parentID,
		&s.SizeBytes,
		&s.Status,
		&s.VMState,
		&hasChildren,
		&snapshotPath,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan snapshot: %w", err)
	}

	s.ParentID = parentID.String
	s.SnapshotPath = snapshotPath.String
	s.HasChildren = hasChildren == 1

	return &s, nil
}

// scanSnapshotRow scans a row from Rows into a Snapshot struct
func scanSnapshotRow(rows rowsScanner) (*model.Snapshot, error) {
	var s model.Snapshot
	var parentID sql.NullString
	var snapshotPath sql.NullString
	var hasChildren int

	err := rows.Scan(
		&s.ID,
		&s.VMID,
		&s.Name,
		&s.Description,
		&s.CreatedAt,
		&parentID,
		&s.SizeBytes,
		&s.Status,
		&s.VMState,
		&hasChildren,
		&snapshotPath,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan snapshot row: %w", err)
	}

	s.ParentID = parentID.String
	s.SnapshotPath = snapshotPath.String
	s.HasChildren = hasChildren == 1

	return &s, nil
}

// Helper types for scanning
type scanner interface {
	Scan(dest ...interface{}) error
}

type rowsScanner interface {
	Scan(dest ...interface{}) error
}

// Helper functions
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
