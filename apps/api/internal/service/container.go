package service

import (
	"context"
	"errors"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

var (
	ErrContainerNotFound       = errors.New("container not found")
	ErrContainerAlreadyRunning = errors.New("container is already running")
	ErrContainerAlreadyStopped = errors.New("container is already stopped")
	ErrContainerInvalidState   = errors.New("container is in an invalid state for this action")
)

// ContainerService provides business logic for container operations
type ContainerService struct {
	repo repository.ContainerRepository
}

// NewContainerService creates a new container service
func NewContainerService(repo repository.ContainerRepository) *ContainerService {
	return &ContainerService{repo: repo}
}

// GetAll returns all containers
func (s *ContainerService) GetAll(ctx context.Context) ([]*model.Container, error) {
	return s.repo.GetAll(ctx)
}

// GetByNode returns containers filtered by node
func (s *ContainerService) GetByNode(ctx context.Context, node string) ([]*model.Container, error) {
	return s.repo.GetByNode(ctx, node)
}

// GetByID returns a container by ID
func (s *ContainerService) GetByID(ctx context.Context, id string) (*model.Container, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByCTID returns a container by numeric CTID
func (s *ContainerService) GetByCTID(ctx context.Context, ctid int) (*model.Container, error) {
	return s.repo.GetByCTID(ctx, ctid)
}

// Create creates a new container
func (s *ContainerService) Create(ctx context.Context, req *model.ContainerCreateRequest) (*model.Container, error) {
	return s.repo.Create(ctx, req)
}

// Update updates an existing container
func (s *ContainerService) Update(ctx context.Context, ctid int, req *model.ContainerUpdateRequest) (*model.Container, error) {
	return s.repo.Update(ctx, ctid, req)
}

// Delete removes a container
func (s *ContainerService) Delete(ctx context.Context, ctid int) error {
	return s.repo.Delete(ctx, ctid)
}

// Start starts a container
func (s *ContainerService) Start(ctx context.Context, ctid int) error {
	ctr, err := s.GetByCTID(ctx, ctid)
	if err != nil {
		return err
	}
	if ctr.Status == model.ContainerStatusRunning {
		return ErrContainerAlreadyRunning
	}
	return s.repo.Start(ctx, ctid)
}

// Stop stops a container
func (s *ContainerService) Stop(ctx context.Context, ctid int) error {
	ctr, err := s.GetByCTID(ctx, ctid)
	if err != nil {
		return err
	}
	if ctr.Status == model.ContainerStatusStopped {
		return ErrContainerAlreadyStopped
	}
	return s.repo.Stop(ctx, ctid)
}

// Reboot reboots a container
func (s *ContainerService) Reboot(ctx context.Context, ctid int) error {
	ctr, err := s.GetByCTID(ctx, ctid)
	if err != nil {
		return err
	}
	if ctr.Status != model.ContainerStatusRunning {
		return ErrContainerInvalidState
	}
	return s.repo.Reboot(ctx, ctid)
}
