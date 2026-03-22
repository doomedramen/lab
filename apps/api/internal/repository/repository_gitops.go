package repository

import (
	"context"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// GitOpsRepository defines operations for GitOps configuration and state storage
type GitOpsRepository interface {
	// Config CRUD
	CreateConfig(ctx context.Context, config *model.GitOpsConfig) error
	GetConfig(ctx context.Context, id string) (*model.GitOpsConfig, error)
	ListConfigs(ctx context.Context) ([]*model.GitOpsConfig, error)
	UpdateConfig(ctx context.Context, config *model.GitOpsConfig) error
	DeleteConfig(ctx context.Context, id string) error
	
	// Config sync tracking
	UpdateConfigSync(ctx context.Context, id string, status model.GitOpsStatus, commitHash, message string) error
	GetConfigsDueForSync(ctx context.Context, now time.Time) ([]*model.GitOpsConfig, error)
	
	// Resource CRUD
	CreateResource(ctx context.Context, resource *model.GitOpsResource) error
	GetResource(ctx context.Context, id string) (*model.GitOpsResource, error)
	ListResourcesByConfig(ctx context.Context, configID string) ([]*model.GitOpsResource, error)
	UpdateResource(ctx context.Context, resource *model.GitOpsResource) error
	DeleteResource(ctx context.Context, id string) error
	GetResourceByManifest(ctx context.Context, configID, manifestPath string) (*model.GitOpsResource, error)
	
	// Sync log operations
	CreateSyncLog(ctx context.Context, log *model.GitOpsSyncLog) error
	GetSyncLogs(ctx context.Context, configID string, limit int) ([]*model.GitOpsSyncLog, error)
	GetLastSyncLog(ctx context.Context, configID string) (*model.GitOpsSyncLog, error)
}
