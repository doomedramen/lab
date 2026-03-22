package handler

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
	"github.com/google/uuid"
)

// GitOpsServiceServer implements labv1connect.GitOpsServiceHandler
type GitOpsServiceServer struct {
	labv1connect.UnimplementedGitOpsServiceHandler
	gitopsSvc *service.GitOpsService
}

// NewGitOpsServiceServer creates a new GitOps service server
func NewGitOpsServiceServer(gitopsSvc *service.GitOpsService) *GitOpsServiceServer {
	return &GitOpsServiceServer{
		gitopsSvc: gitopsSvc,
	}
}

var _ labv1connect.GitOpsServiceHandler = (*GitOpsServiceServer)(nil)

// ListGitOpsConfigs returns all GitOps configurations
func (s *GitOpsServiceServer) ListGitOpsConfigs(
	ctx context.Context,
	req *connect.Request[labv1.ListGitOpsConfigsRequest],
) (*connect.Response[labv1.ListGitOpsConfigsResponse], error) {
	// TODO: Implement when repository is wired
	return connect.NewResponse(&labv1.ListGitOpsConfigsResponse{}), nil
}

// GetGitOpsConfig returns a single GitOps configuration
func (s *GitOpsServiceServer) GetGitOpsConfig(
	ctx context.Context,
	req *connect.Request[labv1.GetGitOpsConfigRequest],
) (*connect.Response[labv1.GetGitOpsConfigResponse], error) {
	// TODO: Implement when repository is wired
	return connect.NewResponse(&labv1.GetGitOpsConfigResponse{}), nil
}

// CreateGitOpsConfig creates a new GitOps configuration
func (s *GitOpsServiceServer) CreateGitOpsConfig(
	ctx context.Context,
	req *connect.Request[labv1.CreateGitOpsConfigRequest],
) (*connect.Response[labv1.CreateGitOpsConfigResponse], error) {
	// Validate input
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if req.Msg.GitUrl == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("git_url is required"))
	}

	// Create config
	config := &model.GitOpsConfig{
		ID:               uuid.New().String(),
		Name:             req.Msg.Name,
		Description:      req.Msg.Description,
		GitURL:           req.Msg.GitUrl,
		GitBranch:        req.Msg.GitBranch,
		GitPath:          req.Msg.GitPath,
		SyncInterval:     time.Duration(req.Msg.SyncInterval) * time.Second,
		Enabled:          req.Msg.Enabled,
		MaxSyncRetries:   int(req.Msg.MaxSyncRetries),
		Status:           model.GitOpsStatusPending,
		StatusMessage:    "Configuration created, awaiting first sync",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		NextSync:         time.Now(),
	}

	// TODO: Save to repository when wired

	return connect.NewResponse(&labv1.CreateGitOpsConfigResponse{
		Config: modelToProtoGitOpsConfig(config),
	}), nil
}

// UpdateGitOpsConfig updates a GitOps configuration
func (s *GitOpsServiceServer) UpdateGitOpsConfig(
	ctx context.Context,
	req *connect.Request[labv1.UpdateGitOpsConfigRequest],
) (*connect.Response[labv1.UpdateGitOpsConfigResponse], error) {
	// Validate input
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	// TODO: Implement when repository is wired

	return connect.NewResponse(&labv1.UpdateGitOpsConfigResponse{}), nil
}

// DeleteGitOpsConfig deletes a GitOps configuration
func (s *GitOpsServiceServer) DeleteGitOpsConfig(
	ctx context.Context,
	req *connect.Request[labv1.DeleteGitOpsConfigRequest],
) (*connect.Response[labv1.DeleteGitOpsConfigResponse], error) {
	// Validate input
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	// TODO: Implement when repository is wired

	return connect.NewResponse(&labv1.DeleteGitOpsConfigResponse{}), nil
}

// SyncGitOpsConfig triggers an immediate synchronization
func (s *GitOpsServiceServer) SyncGitOpsConfig(
	ctx context.Context,
	req *connect.Request[labv1.SyncGitOpsConfigRequest],
) (*connect.Response[labv1.SyncGitOpsConfigResponse], error) {
	// Validate input
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	// TODO: Implement when repository is wired

	return connect.NewResponse(&labv1.SyncGitOpsConfigResponse{}), nil
}

// ListGitOpsResources returns resources for a GitOps configuration
func (s *GitOpsServiceServer) ListGitOpsResources(
	ctx context.Context,
	req *connect.Request[labv1.ListGitOpsResourcesRequest],
) (*connect.Response[labv1.ListGitOpsResourcesResponse], error) {
	// TODO: Implement when repository is wired
	return connect.NewResponse(&labv1.ListGitOpsResourcesResponse{}), nil
}

// GetGitOpsResource returns a single GitOps resource
func (s *GitOpsServiceServer) GetGitOpsResource(
	ctx context.Context,
	req *connect.Request[labv1.GetGitOpsResourceRequest],
) (*connect.Response[labv1.GetGitOpsResourceResponse], error) {
	// TODO: Implement when repository is wired
	return connect.NewResponse(&labv1.GetGitOpsResourceResponse{}), nil
}

// GetGitOpsSyncLogs returns sync logs for a GitOps configuration
func (s *GitOpsServiceServer) GetGitOpsSyncLogs(
	ctx context.Context,
	req *connect.Request[labv1.GetGitOpsSyncLogsRequest],
) (*connect.Response[labv1.GetGitOpsSyncLogsResponse], error) {
	// TODO: Implement when repository is wired
	return connect.NewResponse(&labv1.GetGitOpsSyncLogsResponse{}), nil
}

// modelToProtoGitOpsConfig converts model.GitOpsConfig to proto
func modelToProtoGitOpsConfig(config *model.GitOpsConfig) *labv1.GitOpsConfig {
	return &labv1.GitOpsConfig{
		Id:             config.ID,
		Name:           config.Name,
		Description:    config.Description,
		GitUrl:         config.GitURL,
		GitBranch:      config.GitBranch,
		GitPath:        config.GitPath,
		SyncInterval:   int32(config.SyncInterval),
		LastSync:       config.LastSync.Format("2006-01-02T15:04:05Z"),
		LastSyncHash:   config.LastSyncHash,
		Status:         string(config.Status),
		StatusMessage:  config.StatusMessage,
		Enabled:        config.Enabled,
		CreatedAt:      config.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      config.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		NextSync:       config.NextSync.Format("2006-01-02T15:04:05Z"),
		SyncRetries:    int32(config.SyncRetries),
		MaxSyncRetries: int32(config.MaxSyncRetries),
	}
}
