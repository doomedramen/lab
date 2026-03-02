package sqlite

import (
	"context"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// ContainerRepository implements repository.ContainerRepository using SQLite
// Note: Container support is not yet implemented
type ContainerRepository struct{}

// NewContainerRepository creates a new SQLite container repository
func NewContainerRepository() *ContainerRepository {
	return &ContainerRepository{}
}

// GetAll returns all containers (currently not implemented)
func (r *ContainerRepository) GetAll(_ context.Context) ([]*model.Container, error) {
	return []*model.Container{}, nil
}

// GetByNode returns containers for a specific node (currently not implemented)
func (r *ContainerRepository) GetByNode(_ context.Context, _ string) ([]*model.Container, error) {
	return []*model.Container{}, nil
}

// GetByID returns a container by ID (currently not implemented)
func (r *ContainerRepository) GetByID(_ context.Context, id string) (*model.Container, error) {
	return nil, fmt.Errorf("container %q not found", id)
}

// GetByCTID returns a container by CTID (currently not implemented)
func (r *ContainerRepository) GetByCTID(_ context.Context, ctid int) (*model.Container, error) {
	return nil, fmt.Errorf("container %d not found", ctid)
}

// Create creates a new container (currently not implemented)
func (r *ContainerRepository) Create(_ context.Context, _ *model.ContainerCreateRequest) (*model.Container, error) {
	return nil, fmt.Errorf("container creation not implemented")
}

// Update updates an existing container (currently not implemented)
func (r *ContainerRepository) Update(_ context.Context, ctid int, _ *model.ContainerUpdateRequest) (*model.Container, error) {
	return nil, fmt.Errorf("container update not implemented: ctid=%d", ctid)
}

// Delete deletes a container (currently not implemented)
func (r *ContainerRepository) Delete(_ context.Context, ctid int) error {
	return fmt.Errorf("container deletion not implemented: ctid=%d", ctid)
}

// Start starts a container (currently not implemented)
func (r *ContainerRepository) Start(_ context.Context, ctid int) error {
	return fmt.Errorf("container start not implemented: ctid=%d", ctid)
}

// Stop stops a container (currently not implemented)
func (r *ContainerRepository) Stop(_ context.Context, ctid int) error {
	return fmt.Errorf("container stop not implemented: ctid=%d", ctid)
}

// Shutdown shuts down a container (currently not implemented)
func (r *ContainerRepository) Shutdown(_ context.Context, ctid int) error {
	return fmt.Errorf("container shutdown not implemented: ctid=%d", ctid)
}

// Pause pauses a container (currently not implemented)
func (r *ContainerRepository) Pause(_ context.Context, ctid int) error {
	return fmt.Errorf("container pause not implemented: ctid=%d", ctid)
}

// Resume resumes a container (currently not implemented)
func (r *ContainerRepository) Resume(_ context.Context, ctid int) error {
	return fmt.Errorf("container resume not implemented: ctid=%d", ctid)
}

// Reboot reboots a container (currently not implemented)
func (r *ContainerRepository) Reboot(_ context.Context, ctid int) error {
	return fmt.Errorf("container reboot not implemented: ctid=%d", ctid)
}
