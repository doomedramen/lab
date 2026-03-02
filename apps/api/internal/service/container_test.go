package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- mock container repository ---

type mockContainerRepo struct {
	containers map[int]*model.Container
	nextCTID   int
}

func newMockContainerRepo() *mockContainerRepo {
	return &mockContainerRepo{containers: make(map[int]*model.Container), nextCTID: 100}
}

func (m *mockContainerRepo) addContainer(c *model.Container) {
	m.containers[c.CTID] = c
}

func (m *mockContainerRepo) GetAll(_ context.Context) ([]*model.Container, error) {
	out := make([]*model.Container, 0, len(m.containers))
	for _, c := range m.containers {
		out = append(out, c)
	}
	return out, nil
}

func (m *mockContainerRepo) GetByNode(_ context.Context, node string) ([]*model.Container, error) {
	var out []*model.Container
	for _, c := range m.containers {
		if c.Node == node {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *mockContainerRepo) GetByID(_ context.Context, id string) (*model.Container, error) {
	for _, c := range m.containers {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, fmt.Errorf("container %s not found", id)
}

func (m *mockContainerRepo) GetByCTID(_ context.Context, ctid int) (*model.Container, error) {
	if c, ok := m.containers[ctid]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("container with CTID %d not found", ctid)
}

func (m *mockContainerRepo) Create(_ context.Context, req *model.ContainerCreateRequest) (*model.Container, error) {
	c := &model.Container{
		ID:     req.Name,
		CTID:   m.nextCTID,
		Name:   req.Name,
		Node:   req.Node,
		Status: model.ContainerStatusStopped,
	}
	m.nextCTID++
	m.containers[c.CTID] = c
	return c, nil
}

func (m *mockContainerRepo) Update(_ context.Context, ctid int, req *model.ContainerUpdateRequest) (*model.Container, error) {
	c, ok := m.containers[ctid]
	if !ok {
		return nil, fmt.Errorf("container with CTID %d not found", ctid)
	}
	_ = req
	return c, nil
}

func (m *mockContainerRepo) Delete(_ context.Context, ctid int) error {
	if _, ok := m.containers[ctid]; !ok {
		return fmt.Errorf("container with CTID %d not found", ctid)
	}
	delete(m.containers, ctid)
	return nil
}

func (m *mockContainerRepo) Start(_ context.Context, _ int) error    { return nil }
func (m *mockContainerRepo) Stop(_ context.Context, _ int) error     { return nil }
func (m *mockContainerRepo) Shutdown(_ context.Context, _ int) error { return nil }
func (m *mockContainerRepo) Pause(_ context.Context, _ int) error    { return nil }
func (m *mockContainerRepo) Resume(_ context.Context, _ int) error   { return nil }
func (m *mockContainerRepo) Reboot(_ context.Context, _ int) error   { return nil }

// --- tests ---

func TestContainerService_GetAll_Empty(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())
	containers, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 0 {
		t.Errorf("expected 0, got %d", len(containers))
	}
}

func TestContainerService_GetAll_ReturnsAll(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusRunning})
	repo.addContainer(&model.Container{ID: "c2", CTID: 101, Status: model.ContainerStatusStopped})
	svc := NewContainerService(repo)

	containers, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 2 {
		t.Errorf("expected 2, got %d", len(containers))
	}
}

func TestContainerService_GetByNode(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Node: "node-1"})
	repo.addContainer(&model.Container{ID: "c2", CTID: 101, Node: "node-2"})
	repo.addContainer(&model.Container{ID: "c3", CTID: 102, Node: "node-1"})
	svc := NewContainerService(repo)

	containers, err := svc.GetByNode(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(containers) != 2 {
		t.Errorf("expected 2 containers for node-1, got %d", len(containers))
	}
}

func TestContainerService_GetByID_Found(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Name: "web"})
	svc := NewContainerService(repo)

	c, err := svc.GetByID(context.Background(), "c1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "web" {
		t.Errorf("Name = %q, want web", c.Name)
	}
}

func TestContainerService_GetByID_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent container, got nil")
	}
}

func TestContainerService_GetByCTID_Found(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Name: "db"})
	svc := NewContainerService(repo)

	c, err := svc.GetByCTID(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "db" {
		t.Errorf("Name = %q, want db", c.Name)
	}
}

func TestContainerService_GetByCTID_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	_, err := svc.GetByCTID(context.Background(), 999)
	if err == nil {
		t.Error("expected error for nonexistent container CTID, got nil")
	}
}

func TestContainerService_Create(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	c, err := svc.Create(context.Background(), &model.ContainerCreateRequest{Name: "web", Node: "node-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("Create returned nil")
	}
	if c.Name != "web" {
		t.Errorf("Name = %q, want web", c.Name)
	}
}

func TestContainerService_Update_Found(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100})
	svc := NewContainerService(repo)

	c, err := svc.Update(context.Background(), 100, &model.ContainerUpdateRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Error("expected non-nil container")
	}
}

func TestContainerService_Update_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	_, err := svc.Update(context.Background(), 999, &model.ContainerUpdateRequest{})
	if err == nil {
		t.Error("expected error for nonexistent container, got nil")
	}
}

func TestContainerService_Delete_Found(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100})
	svc := NewContainerService(repo)

	if err := svc.Delete(context.Background(), 100); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerService_Delete_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	if err := svc.Delete(context.Background(), 999); err == nil {
		t.Error("expected error for nonexistent container, got nil")
	}
}

func TestContainerService_Start_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	if err := svc.Start(context.Background(), 999); err == nil {
		t.Error("expected error for nonexistent container, got nil")
	}
}

func TestContainerService_Start_AlreadyRunning(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusRunning})
	svc := NewContainerService(repo)

	if err := svc.Start(context.Background(), 100); !errors.Is(err, ErrContainerAlreadyRunning) {
		t.Errorf("expected ErrContainerAlreadyRunning, got %v", err)
	}
}

func TestContainerService_Start_Stopped(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusStopped})
	svc := NewContainerService(repo)

	if err := svc.Start(context.Background(), 100); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerService_Stop_NotFound(t *testing.T) {
	svc := NewContainerService(newMockContainerRepo())

	if err := svc.Stop(context.Background(), 999); err == nil {
		t.Error("expected error for nonexistent container, got nil")
	}
}

func TestContainerService_Stop_AlreadyStopped(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusStopped})
	svc := NewContainerService(repo)

	if err := svc.Stop(context.Background(), 100); !errors.Is(err, ErrContainerAlreadyStopped) {
		t.Errorf("expected ErrContainerAlreadyStopped, got %v", err)
	}
}

func TestContainerService_Stop_Running(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusRunning})
	svc := NewContainerService(repo)

	if err := svc.Stop(context.Background(), 100); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContainerService_Reboot_NotRunning(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusStopped})
	svc := NewContainerService(repo)

	if err := svc.Reboot(context.Background(), 100); !errors.Is(err, ErrContainerInvalidState) {
		t.Errorf("expected ErrContainerInvalidState, got %v", err)
	}
}

func TestContainerService_Reboot_Running(t *testing.T) {
	repo := newMockContainerRepo()
	repo.addContainer(&model.Container{ID: "c1", CTID: 100, Status: model.ContainerStatusRunning})
	svc := NewContainerService(repo)

	if err := svc.Reboot(context.Background(), 100); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
