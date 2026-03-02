package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

var (
	ErrStackNotFound      = errors.New("stack not found")
	ErrStackInvalidToken  = errors.New("invalid or expired token")
)

// StackService provides business logic for Docker Compose stack operations.
type StackService struct {
	repo repository.StackRepository

	tokensMu        sync.Mutex
	containerTokens map[string]model.ContainerToken
	logsTokens      map[string]model.LogsToken
}

// NewStackService creates a new StackService.
func NewStackService(repo repository.StackRepository) *StackService {
	svc := &StackService{
		repo:            repo,
		containerTokens: make(map[string]model.ContainerToken),
		logsTokens:      make(map[string]model.LogsToken),
	}
	go svc.cleanupExpiredTokens()
	return svc
}

func (s *StackService) cleanupExpiredTokens() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.tokensMu.Lock()
		for token, ct := range s.containerTokens {
			if time.Since(ct.CreatedAt) > 30*time.Second {
				delete(s.containerTokens, token)
			}
		}
		for token, lt := range s.logsTokens {
			if time.Since(lt.CreatedAt) > 30*time.Second {
				delete(s.logsTokens, token)
			}
		}
		s.tokensMu.Unlock()
	}
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetAll returns all stacks.
func (s *StackService) GetAll(ctx context.Context) ([]*model.DockerStack, error) {
	return s.repo.GetAll(ctx)
}

// GetByID returns a stack by ID.
func (s *StackService) GetByID(ctx context.Context, id string) (*model.DockerStack, error) {
	st, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if st == nil {
		return nil, ErrStackNotFound
	}
	return st, nil
}

// Create creates a new stack.
func (s *StackService) Create(ctx context.Context, req *model.StackCreateRequest) (*model.DockerStack, error) {
	return s.repo.Create(ctx, req)
}

// Update updates an existing stack's compose/env files.
func (s *StackService) Update(ctx context.Context, id string, req *model.StackUpdateRequest) (*model.DockerStack, error) {
	return s.repo.Update(ctx, id, req)
}

// Delete removes a stack.
func (s *StackService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Start brings up the stack (docker compose up -d).
func (s *StackService) Start(ctx context.Context, id string) error {
	return s.repo.Start(ctx, id)
}

// Stop stops the stack containers (docker compose stop).
func (s *StackService) Stop(ctx context.Context, id string) error {
	return s.repo.Stop(ctx, id)
}

// Restart restarts the stack (docker compose restart).
func (s *StackService) Restart(ctx context.Context, id string) error {
	return s.repo.Restart(ctx, id)
}

// UpdateImages pulls new images and recreates containers.
func (s *StackService) UpdateImages(ctx context.Context, id string) error {
	return s.repo.UpdateImages(ctx, id)
}

// Down brings down the stack and removes containers (docker compose down).
func (s *StackService) Down(ctx context.Context, id string) error {
	return s.repo.Down(ctx, id)
}

// GetContainerToken generates a one-time token for WebSocket PTY access to a container.
func (s *StackService) GetContainerToken(stackID, containerName string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	s.tokensMu.Lock()
	s.containerTokens[token] = model.ContainerToken{
		StackID:       stackID,
		ContainerName: containerName,
		CreatedAt:     time.Now(),
	}
	s.tokensMu.Unlock()
	return token, nil
}

// ValidateContainerToken validates and consumes a one-time container token.
func (s *StackService) ValidateContainerToken(token string) (model.ContainerToken, bool) {
	s.tokensMu.Lock()
	defer s.tokensMu.Unlock()

	ct, ok := s.containerTokens[token]
	if !ok {
		return model.ContainerToken{}, false
	}
	if time.Since(ct.CreatedAt) > 30*time.Second {
		delete(s.containerTokens, token)
		return model.ContainerToken{}, false
	}
	delete(s.containerTokens, token)
	return ct, true
}

// GetStackLogsToken generates a one-time token for WebSocket log streaming.
func (s *StackService) GetStackLogsToken(stackID string) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}
	s.tokensMu.Lock()
	s.logsTokens[token] = model.LogsToken{
		StackID:   stackID,
		CreatedAt: time.Now(),
	}
	s.tokensMu.Unlock()
	return token, nil
}

// ValidateStackLogsToken validates and consumes a one-time logs token.
func (s *StackService) ValidateStackLogsToken(token string) (model.LogsToken, bool) {
	s.tokensMu.Lock()
	defer s.tokensMu.Unlock()

	lt, ok := s.logsTokens[token]
	if !ok {
		return model.LogsToken{}, false
	}
	if time.Since(lt.CreatedAt) > 30*time.Second {
		delete(s.logsTokens, token)
		return model.LogsToken{}, false
	}
	delete(s.logsTokens, token)
	return lt, true
}
