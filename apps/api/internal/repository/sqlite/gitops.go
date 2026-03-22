package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// GitOpsRepository implements repository.GitOpsRepository using SQLite
type GitOpsRepository struct {
	db *sql.DB
}

// NewGitOpsRepository creates a new GitOps repository
func NewGitOpsRepository(db *sqlitePkg.DB) *GitOpsRepository {
	return &GitOpsRepository{db: db.DB}
}

// CreateConfig creates a new GitOps configuration
func (r *GitOpsRepository) CreateConfig(ctx context.Context, config *model.GitOpsConfig) error {
	query := `
		INSERT INTO gitops_configs (
			id, name, description, git_url, git_branch, git_path,
			sync_interval, status, status_message, enabled,
			created_at, updated_at, next_sync, sync_retries, max_sync_retries
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID, config.Name, config.Description,
		config.GitURL, config.GitBranch, config.GitPath,
		config.SyncInterval, config.Status, config.StatusMessage,
		config.Enabled, config.CreatedAt, config.UpdatedAt,
		config.NextSync, config.SyncRetries, config.MaxSyncRetries,
	)

	if err != nil {
		return fmt.Errorf("failed to create GitOps config: %w", err)
	}

	return nil
}

// GetConfig retrieves a GitOps configuration by ID
func (r *GitOpsRepository) GetConfig(ctx context.Context, id string) (*model.GitOpsConfig, error) {
	query := `
		SELECT id, name, description, git_url, git_branch, git_path,
		       sync_interval, last_sync, last_sync_hash, status,
		       status_message, enabled, created_at, updated_at,
		       next_sync, sync_retries, max_sync_retries
		FROM gitops_configs
		WHERE id = ?
	`

	config := &model.GitOpsConfig{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.Name, &config.Description,
		&config.GitURL, &config.GitBranch, &config.GitPath,
		&config.SyncInterval, &config.LastSync, &config.LastSyncHash,
		&config.Status, &config.StatusMessage, &config.Enabled,
		&config.CreatedAt, &config.UpdatedAt, &config.NextSync,
		&config.SyncRetries, &config.MaxSyncRetries,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps config: %w", err)
	}

	return config, nil
}

// ListConfigs lists all GitOps configurations
func (r *GitOpsRepository) ListConfigs(ctx context.Context) ([]*model.GitOpsConfig, error) {
	query := `
		SELECT id, name, description, git_url, git_branch, git_path,
		       sync_interval, last_sync, last_sync_hash, status,
		       status_message, enabled, created_at, updated_at,
		       next_sync, sync_retries, max_sync_retries
		FROM gitops_configs
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitOps configs: %w", err)
	}
	defer rows.Close()

	var configs []*model.GitOpsConfig
	for rows.Next() {
		config := &model.GitOpsConfig{}
		err := rows.Scan(
			&config.ID, &config.Name, &config.Description,
			&config.GitURL, &config.GitBranch, &config.GitPath,
			&config.SyncInterval, &config.LastSync, &config.LastSyncHash,
			&config.Status, &config.StatusMessage, &config.Enabled,
			&config.CreatedAt, &config.UpdatedAt, &config.NextSync,
			&config.SyncRetries, &config.MaxSyncRetries,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GitOps config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// UpdateConfig updates a GitOps configuration
func (r *GitOpsRepository) UpdateConfig(ctx context.Context, config *model.GitOpsConfig) error {
	query := `
		UPDATE gitops_configs
		SET name = ?, description = ?, git_url = ?, git_branch = ?, git_path = ?,
		    sync_interval = ?, status = ?, status_message = ?, enabled = ?,
		    updated_at = ?, next_sync = ?, sync_retries = ?, max_sync_retries = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		config.Name, config.Description, config.GitURL,
		config.GitBranch, config.GitPath, config.SyncInterval,
		config.Status, config.StatusMessage, config.Enabled,
		config.UpdatedAt, config.NextSync, config.SyncRetries,
		config.MaxSyncRetries, config.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update GitOps config: %w", err)
	}

	return nil
}

// DeleteConfig deletes a GitOps configuration
func (r *GitOpsRepository) DeleteConfig(ctx context.Context, id string) error {
	query := `DELETE FROM gitops_configs WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to delete GitOps config: %w", err)
	}

	return nil
}

// UpdateConfigSync updates sync tracking for a config
func (r *GitOpsRepository) UpdateConfigSync(ctx context.Context, id string, status model.GitOpsStatus, commitHash, message string) error {
	query := `
		UPDATE gitops_configs
		SET last_sync = ?, last_sync_hash = ?, status = ?, status_message = ?,
		    sync_retries = CASE WHEN ? != 'Healthy' THEN sync_retries + 1 ELSE 0 END,
		    next_sync = datetime('now', '+' || (SELECT sync_interval FROM gitops_configs WHERE id = ?) || ' seconds')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		time.Now(), commitHash, status, message,
		status, id, id,
	)

	if err != nil {
		return fmt.Errorf("failed to update GitOps config sync: %w", err)
	}

	return nil
}

// GetConfigsDueForSync returns configs that are due for synchronization
func (r *GitOpsRepository) GetConfigsDueForSync(ctx context.Context, now time.Time) ([]*model.GitOpsConfig, error) {
	query := `
		SELECT id, name, description, git_url, git_branch, git_path,
		       sync_interval, last_sync, last_sync_hash, status,
		       status_message, enabled, created_at, updated_at,
		       next_sync, sync_retries, max_sync_retries
		FROM gitops_configs
		WHERE enabled = 1 AND next_sync <= ?
		ORDER BY next_sync ASC
	`

	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to get configs due for sync: %w", err)
	}
	defer rows.Close()

	var configs []*model.GitOpsConfig
	for rows.Next() {
		config := &model.GitOpsConfig{}
		err := rows.Scan(
			&config.ID, &config.Name, &config.Description,
			&config.GitURL, &config.GitBranch, &config.GitPath,
			&config.SyncInterval, &config.LastSync, &config.LastSyncHash,
			&config.Status, &config.StatusMessage, &config.Enabled,
			&config.CreatedAt, &config.UpdatedAt, &config.NextSync,
			&config.SyncRetries, &config.MaxSyncRetries,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GitOps config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, nil
}

// CreateResource creates a new GitOps resource
func (r *GitOpsRepository) CreateResource(ctx context.Context, resource *model.GitOpsResource) error {
	specJSON, err := json.Marshal(resource.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal resource spec: %w", err)
	}

	query := `
		INSERT INTO gitops_resources (
			id, config_id, kind, name, namespace, manifest_path, manifest_hash,
			spec, status, status_message, last_applied, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		resource.ID, resource.ConfigID, resource.Kind, resource.Name,
		resource.Namespace, resource.ManifestPath, resource.ManifestHash,
		specJSON, resource.Status, resource.StatusMessage,
		resource.LastApplied, resource.CreatedAt, resource.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create GitOps resource: %w", err)
	}

	return nil
}

// GetResource retrieves a GitOps resource by ID
func (r *GitOpsRepository) GetResource(ctx context.Context, id string) (*model.GitOpsResource, error) {
	query := `
		SELECT id, config_id, kind, name, namespace, manifest_path, manifest_hash,
		       spec, status, status_message, last_applied, created_at, updated_at
		FROM gitops_resources
		WHERE id = ?
	`

	resource := &model.GitOpsResource{}
	var specJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&resource.ID, &resource.ConfigID, &resource.Kind, &resource.Name,
		&resource.Namespace, &resource.ManifestPath, &resource.ManifestHash,
		&specJSON, &resource.Status, &resource.StatusMessage,
		&resource.LastApplied, &resource.CreatedAt, &resource.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps resource: %w", err)
	}

	if err := json.Unmarshal(specJSON, &resource.Spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource spec: %w", err)
	}

	return resource, nil
}

// ListResourcesByConfig lists all resources for a GitOps configuration
func (r *GitOpsRepository) ListResourcesByConfig(ctx context.Context, configID string) ([]*model.GitOpsResource, error) {
	query := `
		SELECT id, config_id, kind, name, namespace, manifest_path, manifest_hash,
		       spec, status, status_message, last_applied, created_at, updated_at
		FROM gitops_resources
		WHERE config_id = ?
		ORDER BY kind, name
	`

	rows, err := r.db.QueryContext(ctx, query, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitOps resources: %w", err)
	}
	defer rows.Close()

	var resources []*model.GitOpsResource
	for rows.Next() {
		resource := &model.GitOpsResource{}
		var specJSON []byte

		err := rows.Scan(
			&resource.ID, &resource.ConfigID, &resource.Kind, &resource.Name,
			&resource.Namespace, &resource.ManifestPath, &resource.ManifestHash,
			&specJSON, &resource.Status, &resource.StatusMessage,
			&resource.LastApplied, &resource.CreatedAt, &resource.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GitOps resource: %w", err)
		}

		if err := json.Unmarshal(specJSON, &resource.Spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource spec: %w", err)
		}

		resources = append(resources, resource)
	}

	return resources, nil
}

// UpdateResource updates a GitOps resource
func (r *GitOpsRepository) UpdateResource(ctx context.Context, resource *model.GitOpsResource) error {
	specJSON, err := json.Marshal(resource.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal resource spec: %w", err)
	}

	query := `
		UPDATE gitops_resources
		SET manifest_hash = ?, spec = ?, status = ?, status_message = ?,
		    last_applied = ?, updated_at = ?, last_diff = ?
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query,
		resource.ManifestHash, specJSON, resource.Status,
		resource.StatusMessage, resource.LastApplied, resource.UpdatedAt,
		resource.LastDiff, resource.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update GitOps resource: %w", err)
	}

	return nil
}

// DeleteResource deletes a GitOps resource
func (r *GitOpsRepository) DeleteResource(ctx context.Context, id string) error {
	query := `DELETE FROM gitops_resources WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)

	if err != nil {
		return fmt.Errorf("failed to delete GitOps resource: %w", err)
	}

	return nil
}

// GetResourceByManifest retrieves a resource by its manifest path
func (r *GitOpsRepository) GetResourceByManifest(ctx context.Context, configID, manifestPath string) (*model.GitOpsResource, error) {
	query := `
		SELECT id, config_id, kind, name, namespace, manifest_path, manifest_hash,
		       spec, status, status_message, last_applied, created_at, updated_at
		FROM gitops_resources
		WHERE config_id = ? AND manifest_path = ?
	`

	resource := &model.GitOpsResource{}
	var specJSON []byte

	err := r.db.QueryRowContext(ctx, query, configID, manifestPath).Scan(
		&resource.ID, &resource.ConfigID, &resource.Kind, &resource.Name,
		&resource.Namespace, &resource.ManifestPath, &resource.ManifestHash,
		&specJSON, &resource.Status, &resource.StatusMessage,
		&resource.LastApplied, &resource.CreatedAt, &resource.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps resource by manifest: %w", err)
	}

	if err := json.Unmarshal(specJSON, &resource.Spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource spec: %w", err)
	}

	return resource, nil
}

// CreateSyncLog creates a new sync log entry
func (r *GitOpsRepository) CreateSyncLog(ctx context.Context, log *model.GitOpsSyncLog) error {
	query := `
		INSERT INTO gitops_sync_logs (
			id, config_id, start_time, end_time, duration, status, message,
			commit_hash, resources_scanned, resources_created,
			resources_updated, resources_deleted, resources_failed
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.ID, log.ConfigID, log.StartTime, log.EndTime, log.Duration,
		log.Status, log.Message, log.CommitHash, log.ResourcesScanned,
		log.ResourcesCreated, log.ResourcesUpdated, log.ResourcesDeleted,
		log.ResourcesFailed,
	)

	if err != nil {
		return fmt.Errorf("failed to create GitOps sync log: %w", err)
	}

	return nil
}

// GetSyncLogs retrieves sync logs for a configuration
func (r *GitOpsRepository) GetSyncLogs(ctx context.Context, configID string, limit int) ([]*model.GitOpsSyncLog, error) {
	query := `
		SELECT id, config_id, start_time, end_time, duration, status, message,
		       commit_hash, resources_scanned, resources_created,
		       resources_updated, resources_deleted, resources_failed
		FROM gitops_sync_logs
		WHERE config_id = ?
		ORDER BY start_time DESC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, configID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps sync logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.GitOpsSyncLog
	for rows.Next() {
		log := &model.GitOpsSyncLog{}
		err := rows.Scan(
			&log.ID, &log.ConfigID, &log.StartTime, &log.EndTime, &log.Duration,
			&log.Status, &log.Message, &log.CommitHash, &log.ResourcesScanned,
			&log.ResourcesCreated, &log.ResourcesUpdated, &log.ResourcesDeleted,
			&log.ResourcesFailed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GitOps sync log: %w", err)
		}
		logs = append(logs, log)
	}

	return logs, nil
}

// GetLastSyncLog retrieves the most recent sync log for a configuration
func (r *GitOpsRepository) GetLastSyncLog(ctx context.Context, configID string) (*model.GitOpsSyncLog, error) {
	query := `
		SELECT id, config_id, start_time, end_time, duration, status, message,
		       commit_hash, resources_scanned, resources_created,
		       resources_updated, resources_deleted, resources_failed
		FROM gitops_sync_logs
		WHERE config_id = ?
		ORDER BY start_time DESC
		LIMIT 1
	`

	log := &model.GitOpsSyncLog{}
	err := r.db.QueryRowContext(ctx, query, configID).Scan(
		&log.ID, &log.ConfigID, &log.StartTime, &log.EndTime, &log.Duration,
		&log.Status, &log.Message, &log.CommitHash, &log.ResourcesScanned,
		&log.ResourcesCreated, &log.ResourcesUpdated, &log.ResourcesDeleted,
		&log.ResourcesFailed,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last GitOps sync log: %w", err)
	}

	return log, nil
}
