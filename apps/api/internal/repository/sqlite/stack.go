package sqlite

import (
	"context"
	"errors"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

var errNotImplemented = errors.New("stack operations not available: stacks_dir not configured")

// StackRepository is a no-op implementation of repository.StackRepository.
// It is used when stacks_dir is not configured and Docker stacks are disabled.
type StackRepository struct{}

// NewStackRepository creates a new no-op stack repository.
func NewStackRepository() *StackRepository {
	return &StackRepository{}
}

func (r *StackRepository) GetAll(_ context.Context) ([]*model.DockerStack, error) {
	return nil, nil
}

func (r *StackRepository) GetByID(_ context.Context, _ string) (*model.DockerStack, error) {
	return nil, errNotImplemented
}

func (r *StackRepository) Create(_ context.Context, _ *model.StackCreateRequest) (*model.DockerStack, error) {
	return nil, errNotImplemented
}

func (r *StackRepository) Update(_ context.Context, _ string, _ *model.StackUpdateRequest) (*model.DockerStack, error) {
	return nil, errNotImplemented
}

func (r *StackRepository) Delete(_ context.Context, _ string) error {
	return errNotImplemented
}

func (r *StackRepository) Start(_ context.Context, _ string) error {
	return errNotImplemented
}

func (r *StackRepository) Stop(_ context.Context, _ string) error {
	return errNotImplemented
}

func (r *StackRepository) Restart(_ context.Context, _ string) error {
	return errNotImplemented
}

func (r *StackRepository) UpdateImages(_ context.Context, _ string) error {
	return errNotImplemented
}

func (r *StackRepository) Down(_ context.Context, _ string) error {
	return errNotImplemented
}
