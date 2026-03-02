package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- mock node repository ---

type mockNodeRepo struct {
	nodes       map[string]*model.HostNode
	rebootErr   error
	shutdownErr error
}

func newMockNodeRepo() *mockNodeRepo {
	return &mockNodeRepo{nodes: make(map[string]*model.HostNode)}
}

func (m *mockNodeRepo) addNode(n *model.HostNode) {
	m.nodes[n.ID] = n
}

func (m *mockNodeRepo) GetAll(_ context.Context) ([]*model.HostNode, error) {
	out := make([]*model.HostNode, 0, len(m.nodes))
	for _, n := range m.nodes {
		out = append(out, n)
	}
	return out, nil
}

func (m *mockNodeRepo) GetByID(_ context.Context, id string) (*model.HostNode, error) {
	if n, ok := m.nodes[id]; ok {
		return n, nil
	}
	return nil, fmt.Errorf("node %s not found", id)
}

func (m *mockNodeRepo) GetByName(_ context.Context, name string) (*model.HostNode, error) {
	for _, n := range m.nodes {
		if n.Name == name {
			return n, nil
		}
	}
	return nil, fmt.Errorf("node with name %s not found", name)
}

func (m *mockNodeRepo) Reboot(_ context.Context, _ string) error  { return m.rebootErr }
func (m *mockNodeRepo) Shutdown(_ context.Context, _ string) error { return m.shutdownErr }

// --- tests ---

func TestNodeService_GetAll_Empty(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())
	nodes, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestNodeService_GetAll_ReturnsAllNodes(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Name: "node-1", Status: model.NodeStatusOnline})
	repo.addNode(&model.HostNode{ID: "n2", Name: "node-2", Status: model.NodeStatusOffline})
	svc := NewNodeService(repo)

	nodes, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

func TestNodeService_GetByID_Found(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Name: "node-1"})
	svc := NewNodeService(repo)

	node, err := svc.GetByID(context.Background(), "n1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if node.Name != "node-1" {
		t.Errorf("Name = %q, want node-1", node.Name)
	}
}

func TestNodeService_GetByID_NotFound(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent node, got nil")
	}
}

func TestNodeService_Reboot_Success(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusOnline})
	svc := NewNodeService(repo)

	if err := svc.Reboot(context.Background(), "n1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNodeService_Reboot_NotFound(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	if err := svc.Reboot(context.Background(), "nonexistent"); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestNodeService_Reboot_Offline(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusOffline})
	svc := NewNodeService(repo)

	if err := svc.Reboot(context.Background(), "n1"); !errors.Is(err, ErrNodeOffline) {
		t.Errorf("expected ErrNodeOffline, got %v", err)
	}
}

func TestNodeService_Reboot_RepoError(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusOnline})
	repo.rebootErr = errors.New("reboot failed")
	svc := NewNodeService(repo)

	if err := svc.Reboot(context.Background(), "n1"); err == nil {
		t.Error("expected error from repo")
	}
}

func TestNodeService_Shutdown_Success(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusOnline})
	svc := NewNodeService(repo)

	if err := svc.Shutdown(context.Background(), "n1"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNodeService_Shutdown_NotFound(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	if err := svc.Shutdown(context.Background(), "nonexistent"); !errors.Is(err, ErrNodeNotFound) {
		t.Errorf("expected ErrNodeNotFound, got %v", err)
	}
}

func TestNodeService_Shutdown_Offline(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusOffline})
	svc := NewNodeService(repo)

	if err := svc.Shutdown(context.Background(), "n1"); !errors.Is(err, ErrNodeOffline) {
		t.Errorf("expected ErrNodeOffline, got %v", err)
	}
}

func TestNodeService_Reboot_MaintenanceMode(t *testing.T) {
	repo := newMockNodeRepo()
	repo.addNode(&model.HostNode{ID: "n1", Status: model.NodeStatusMaintenance})
	svc := NewNodeService(repo)

	// Maintenance mode node is not offline, so reboot should proceed
	if err := svc.Reboot(context.Background(), "n1"); err != nil {
		t.Errorf("expected no error for maintenance node, got %v", err)
	}
}

// --- host shell token tests ---

func TestNodeService_GetHostShellToken(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	token, err := svc.GetHostShellToken("node-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if len(token) != 32 {
		t.Errorf("token length = %d, want 32", len(token))
	}
}

func TestNodeService_ValidateHostShellToken_Valid(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	token, _ := svc.GetHostShellToken("node-1")
	ht, ok := svc.ValidateHostShellToken(token)
	if !ok {
		t.Error("expected token to be valid")
	}
	if ht.NodeID != "node-1" {
		t.Errorf("NodeID = %q, want node-1", ht.NodeID)
	}
}

func TestNodeService_ValidateHostShellToken_Invalid(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	_, ok := svc.ValidateHostShellToken("nonexistent-token")
	if ok {
		t.Error("expected token to be invalid")
	}
}

func TestNodeService_ValidateHostShellToken_Consumed(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	token, _ := svc.GetHostShellToken("node-1")

	// First validation should succeed
	_, ok := svc.ValidateHostShellToken(token)
	if !ok {
		t.Error("first validation: expected token to be valid")
	}

	// Second validation should fail (token is consumed)
	_, ok = svc.ValidateHostShellToken(token)
	if ok {
		t.Error("second validation: expected token to be invalid (consumed)")
	}
}

func TestNodeService_GetHostShellToken_UniqueTokens(t *testing.T) {
	svc := NewNodeService(newMockNodeRepo())

	token1, _ := svc.GetHostShellToken("node-1")
	token2, _ := svc.GetHostShellToken("node-2")

	if token1 == token2 {
		t.Error("expected different tokens for different calls")
	}
}
