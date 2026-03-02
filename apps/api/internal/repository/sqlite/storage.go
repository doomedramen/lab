package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// StoragePoolRepository handles storage pool storage and retrieval
type StoragePoolRepository struct {
	db *sqlitePkg.DB
}

// NewStoragePoolRepository creates a new storage pool repository
func NewStoragePoolRepository(db *sqlitePkg.DB) *StoragePoolRepository {
	return &StoragePoolRepository{db: db}
}

// Create saves a new storage pool to the database
func (r *StoragePoolRepository) Create(ctx context.Context, pool *model.StoragePool) error {
	optionsJSON, err := json.Marshal(pool.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	query := `
		INSERT INTO storage_pools (
			id, name, type, status, path, capacity_bytes, used_bytes,
			available_bytes, options, enabled, disk_count, description
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		pool.ID,
		pool.Name,
		string(pool.Type),
		string(pool.Status),
		pool.Path,
		pool.CapacityBytes,
		pool.UsedBytes,
		pool.AvailableBytes,
		string(optionsJSON),
		boolToInt(pool.Enabled),
		pool.DiskCount,
		pool.Description,
	)

	if err != nil {
		return fmt.Errorf("failed to create storage pool: %w", err)
	}

	return nil
}

// GetByID retrieves a storage pool by its ID
func (r *StoragePoolRepository) GetByID(ctx context.Context, id string) (*model.StoragePool, error) {
	query := `
		SELECT id, name, type, status, path, capacity_bytes, used_bytes,
		       available_bytes, options, created_at, updated_at, enabled,
		       disk_count, description
		FROM storage_pools
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanStoragePool(row)
}

// List retrieves storage pools with optional filters
func (r *StoragePoolRepository) List(ctx context.Context, poolType model.StorageType, status model.StorageStatus, enabledOnly bool) ([]*model.StoragePool, error) {
	query := `
		SELECT id, name, type, status, path, capacity_bytes, used_bytes,
		       available_bytes, options, created_at, updated_at, enabled,
		       disk_count, description
		FROM storage_pools
		WHERE 1=1
	`

	args := []interface{}{}
	if poolType != "" {
		query += " AND type = ?"
		args = append(args, string(poolType))
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}
	if enabledOnly {
		query += " AND enabled = 1"
		args = append(args, 1)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage pools: %w", err)
	}
	defer rows.Close()

	var pools []*model.StoragePool
	for rows.Next() {
		pool, err := scanStoragePoolRow(rows)
		if err != nil {
			return nil, err
		}
		pools = append(pools, pool)
	}

	return pools, rows.Err()
}

// Update updates an existing storage pool
func (r *StoragePoolRepository) Update(ctx context.Context, pool *model.StoragePool) error {
	optionsJSON, err := json.Marshal(pool.Options)
	if err != nil {
		return fmt.Errorf("failed to marshal options: %w", err)
	}

	query := `
		UPDATE storage_pools
		SET name = ?, status = ?, path = ?, capacity_bytes = ?,
		    used_bytes = ?, available_bytes = ?, options = ?,
		    updated_at = datetime('now'), enabled = ?,
		    disk_count = ?, description = ?
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query,
		pool.Name,
		string(pool.Status),
		pool.Path,
		pool.CapacityBytes,
		pool.UsedBytes,
		pool.AvailableBytes,
		string(optionsJSON),
		boolToInt(pool.Enabled),
		pool.DiskCount,
		pool.Description,
		pool.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update storage pool: %w", err)
	}

	return nil
}

// UpdateStats updates the capacity/usage statistics for a pool
func (r *StoragePoolRepository) UpdateStats(ctx context.Context, id string, capacity, used, available int64, diskCount int) error {
	query := `
		UPDATE storage_pools
		SET capacity_bytes = ?, used_bytes = ?, available_bytes = ?,
		    disk_count = ?, updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, capacity, used, available, diskCount, id)
	if err != nil {
		return fmt.Errorf("failed to update pool stats: %w", err)
	}

	return nil
}

// Delete removes a storage pool from the database
func (r *StoragePoolRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM storage_pools WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete storage pool: %w", err)
	}

	return nil
}

// scanStoragePool scans a single row into a StoragePool struct
func scanStoragePool(row scanner) (*model.StoragePool, error) {
	var p model.StoragePool
	var optionsJSON sql.NullString

	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.Type,
		&p.Status,
		&p.Path,
		&p.CapacityBytes,
		&p.UsedBytes,
		&p.AvailableBytes,
		&optionsJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.Enabled,
		&p.DiskCount,
		&p.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan storage pool: %w", err)
	}

	if optionsJSON.Valid && optionsJSON.String != "" {
		if err := json.Unmarshal([]byte(optionsJSON.String), &p.Options); err != nil {
			p.Options = make(map[string]string)
		}
	} else {
		p.Options = make(map[string]string)
	}

	// Calculate usage percent
	if p.CapacityBytes > 0 {
		p.UsagePercent = float64(p.UsedBytes) / float64(p.CapacityBytes) * 100
	}

	return &p, nil
}

// scanStoragePoolRow scans a row from Rows into a StoragePool struct
func scanStoragePoolRow(rows rowsScanner) (*model.StoragePool, error) {
	var p model.StoragePool
	var optionsJSON sql.NullString

	err := rows.Scan(
		&p.ID,
		&p.Name,
		&p.Type,
		&p.Status,
		&p.Path,
		&p.CapacityBytes,
		&p.UsedBytes,
		&p.AvailableBytes,
		&optionsJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.Enabled,
		&p.DiskCount,
		&p.Description,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan storage pool row: %w", err)
	}

	if optionsJSON.Valid && optionsJSON.String != "" {
		if err := json.Unmarshal([]byte(optionsJSON.String), &p.Options); err != nil {
			p.Options = make(map[string]string)
		}
	} else {
		p.Options = make(map[string]string)
	}

	// Calculate usage percent
	if p.CapacityBytes > 0 {
		p.UsagePercent = float64(p.UsedBytes) / float64(p.CapacityBytes) * 100
	}

	return &p, nil
}

// StorageDiskRepository handles storage disk storage and retrieval
type StorageDiskRepository struct {
	db *sqlitePkg.DB
}

// NewStorageDiskRepository creates a new storage disk repository
func NewStorageDiskRepository(db *sqlitePkg.DB) *StorageDiskRepository {
	return &StorageDiskRepository{db: db}
}

// Create saves a new storage disk to the database
func (r *StorageDiskRepository) Create(ctx context.Context, disk *model.StorageDisk) error {
	query := `
		INSERT INTO storage_disks (
			id, pool_id, name, size_bytes, format, bus, vmid,
			path, used_bytes, sparse, description
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		disk.ID,
		disk.PoolID,
		disk.Name,
		disk.SizeBytes,
		string(disk.Format),
		string(disk.Bus),
		disk.VMID,
		disk.Path,
		disk.UsedBytes,
		boolToInt(disk.Sparse),
		disk.Description,
	)

	if err != nil {
		return fmt.Errorf("failed to create storage disk: %w", err)
	}

	return nil
}

// GetByID retrieves a storage disk by its ID
func (r *StorageDiskRepository) GetByID(ctx context.Context, id string) (*model.StorageDisk, error) {
	query := `
		SELECT id, pool_id, name, size_bytes, format, bus, vmid,
		       path, used_bytes, sparse, created_at, description
		FROM storage_disks
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanStorageDisk(row)
}

// List retrieves storage disks with optional filters
func (r *StorageDiskRepository) List(ctx context.Context, poolID string, vmid int, unassignedOnly bool) ([]*model.StorageDisk, error) {
	query := `
		SELECT id, pool_id, name, size_bytes, format, bus, vmid,
		       path, used_bytes, sparse, created_at, description
		FROM storage_disks
		WHERE 1=1
	`

	args := []interface{}{}
	if poolID != "" {
		query += " AND pool_id = ?"
		args = append(args, poolID)
	}
	if vmid > 0 {
		query += " AND vmid = ?"
		args = append(args, vmid)
	}
	if unassignedOnly {
		query += " AND (vmid = 0 OR vmid IS NULL)"
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list storage disks: %w", err)
	}
	defer rows.Close()

	var disks []*model.StorageDisk
	for rows.Next() {
		disk, err := scanStorageDiskRow(rows)
		if err != nil {
			return nil, err
		}
		disks = append(disks, disk)
	}

	return disks, rows.Err()
}

// Update updates an existing storage disk
func (r *StorageDiskRepository) Update(ctx context.Context, disk *model.StorageDisk) error {
	query := `
		UPDATE storage_disks
		SET name = ?, size_bytes = ?, format = ?, bus = ?, vmid = ?,
		    path = ?, used_bytes = ?, sparse = ?, description = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		disk.Name,
		disk.SizeBytes,
		string(disk.Format),
		string(disk.Bus),
		disk.VMID,
		disk.Path,
		disk.UsedBytes,
		boolToInt(disk.Sparse),
		disk.Description,
		disk.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update storage disk: %w", err)
	}

	return nil
}

// Delete removes a storage disk from the database
func (r *StorageDiskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM storage_disks WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete storage disk: %w", err)
	}

	return nil
}

// scanStorageDisk scans a single row into a StorageDisk struct
func scanStorageDisk(row scanner) (*model.StorageDisk, error) {
	var d model.StorageDisk
	var sparse int

	err := row.Scan(
		&d.ID,
		&d.PoolID,
		&d.Name,
		&d.SizeBytes,
		&d.Format,
		&d.Bus,
		&d.VMID,
		&d.Path,
		&d.UsedBytes,
		&sparse,
		&d.CreatedAt,
		&d.Description,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan storage disk: %w", err)
	}

	d.Sparse = sparse == 1
	return &d, nil
}

// scanStorageDiskRow scans a row from Rows into a StorageDisk struct
func scanStorageDiskRow(rows rowsScanner) (*model.StorageDisk, error) {
	var d model.StorageDisk
	var sparse int

	err := rows.Scan(
		&d.ID,
		&d.PoolID,
		&d.Name,
		&d.SizeBytes,
		&d.Format,
		&d.Bus,
		&d.VMID,
		&d.Path,
		&d.UsedBytes,
		&sparse,
		&d.CreatedAt,
		&d.Description,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan storage disk row: %w", err)
	}

	d.Sparse = sparse == 1
	return &d, nil
}
