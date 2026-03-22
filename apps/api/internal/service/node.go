package service

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

var (
	ErrNodeNotFound      = errors.New("node not found")
	ErrNodeOffline       = errors.New("node is offline")
	ErrNodeInMaintenance = errors.New("node is in maintenance mode")
)

// NodeService provides business logic for node operations
type NodeService struct {
	repo            repository.NodeRepository
	pciRepo         repository.PCIRepository
	tokensMu        sync.Mutex
	hostShellTokens map[string]model.HostShellToken
}

// NewNodeService creates a new node service
func NewNodeService(repo repository.NodeRepository, pciRepo repository.PCIRepository) *NodeService {
	svc := &NodeService{
		repo:            repo,
		pciRepo:         pciRepo,
		hostShellTokens: make(map[string]model.HostShellToken),
	}
	go svc.cleanupExpiredTokens()
	return svc
}

// cleanupExpiredTokens periodically removes expired shell tokens
func (s *NodeService) cleanupExpiredTokens() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.tokensMu.Lock()
		for token, ht := range s.hostShellTokens {
			if time.Since(ht.CreatedAt) > 30*time.Second {
				delete(s.hostShellTokens, token)
			}
		}
		s.tokensMu.Unlock()
	}
}

// GetAll returns all nodes
func (s *NodeService) GetAll(ctx context.Context) ([]*model.HostNode, error) {
	nodes, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	// Populate IOMMU/VFIO status from PCI repository for all nodes
	if s.pciRepo != nil {
		iommuAvailable := s.pciRepo.IsIOMMUAvailable()
		vfioAvailable := s.pciRepo.IsVFIOAvailable()
		for _, node := range nodes {
			node.IOMMUAvailable = iommuAvailable
			node.VFIOAvailable = vfioAvailable
		}
	}
	return nodes, nil
}

// GetByID returns a node by ID
func (s *NodeService) GetByID(ctx context.Context, id string) (*model.HostNode, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Populate IOMMU/VFIO status from PCI repository
	if s.pciRepo != nil {
		node.IOMMUAvailable = s.pciRepo.IsIOMMUAvailable()
		node.VFIOAvailable = s.pciRepo.IsVFIOAvailable()
	}
	return node, nil
}

// Reboot initiates a node reboot
func (s *NodeService) Reboot(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNodeNotFound
	}
	if node.Status == model.NodeStatusOffline {
		return ErrNodeOffline
	}
	return s.repo.Reboot(ctx, id)
}

// Shutdown initiates a node shutdown
func (s *NodeService) Shutdown(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrNodeNotFound
	}
	if node.Status == model.NodeStatusOffline {
		return ErrNodeOffline
	}
	return s.repo.Shutdown(ctx, id)
}

// GetHostShellToken generates a one-time token for WebSocket shell access to a host
func (s *NodeService) GetHostShellToken(nodeID string) (string, error) {
	token, err := generateNodeToken()
	if err != nil {
		return "", err
	}
	s.tokensMu.Lock()
	s.hostShellTokens[token] = model.HostShellToken{
		NodeID:    nodeID,
		CreatedAt: time.Now(),
	}
	s.tokensMu.Unlock()
	return token, nil
}

// ValidateHostShellToken validates and consumes a one-time host shell token
func (s *NodeService) ValidateHostShellToken(token string) (model.HostShellToken, bool) {
	s.tokensMu.Lock()
	defer s.tokensMu.Unlock()

	ht, ok := s.hostShellTokens[token]
	if !ok {
		return model.HostShellToken{}, false
	}
	if time.Since(ht.CreatedAt) > 30*time.Second {
		delete(s.hostShellTokens, token)
		return model.HostShellToken{}, false
	}
	delete(s.hostShellTokens, token)
	return ht, true
}

func generateNodeToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return string(b), nil
}
