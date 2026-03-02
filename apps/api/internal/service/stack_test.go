package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- mock repository ---

type mockStackRepo struct {
	stacks          map[string]*model.DockerStack
	startErr        error
	stopErr         error
	restartErr      error
	downErr         error
	updateImagesErr error
}

func newMockStackRepo() *mockStackRepo {
	return &mockStackRepo{stacks: make(map[string]*model.DockerStack)}
}

func (m *mockStackRepo) GetAll(_ context.Context) ([]*model.DockerStack, error) {
	out := make([]*model.DockerStack, 0, len(m.stacks))
	for _, s := range m.stacks {
		out = append(out, s)
	}
	return out, nil
}

func (m *mockStackRepo) GetByID(_ context.Context, id string) (*model.DockerStack, error) {
	s, ok := m.stacks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

func (m *mockStackRepo) Create(_ context.Context, req *model.StackCreateRequest) (*model.DockerStack, error) {
	s := &model.DockerStack{ID: req.Name, Name: req.Name, Compose: req.Compose, Env: req.Env, CreatedAt: time.Now()}
	m.stacks[req.Name] = s
	return s, nil
}

func (m *mockStackRepo) Update(_ context.Context, id string, req *model.StackUpdateRequest) (*model.DockerStack, error) {
	s, ok := m.stacks[id]
	if !ok {
		return nil, errors.New("not found")
	}
	s.Compose = req.Compose
	s.Env = req.Env
	return s, nil
}

func (m *mockStackRepo) Delete(_ context.Context, id string) error {
	if _, ok := m.stacks[id]; !ok {
		return errors.New("not found")
	}
	delete(m.stacks, id)
	return nil
}

func (m *mockStackRepo) Start(_ context.Context, _ string) error        { return m.startErr }
func (m *mockStackRepo) Stop(_ context.Context, _ string) error         { return m.stopErr }
func (m *mockStackRepo) Restart(_ context.Context, _ string) error      { return m.restartErr }
func (m *mockStackRepo) UpdateImages(_ context.Context, _ string) error { return m.updateImagesErr }
func (m *mockStackRepo) Down(_ context.Context, _ string) error         { return m.downErr }

// --- helpers ---

func newTestStackService() *StackService {
	return NewStackService(newMockStackRepo())
}

// --- tests ---

func TestStackService_GetContainerToken_GeneratesUniqueTokens(t *testing.T) {
	svc := newTestStackService()

	t1, err := svc.GetContainerToken("mystack", "web-1")
	if err != nil {
		t.Fatalf("GetContainerToken: %v", err)
	}
	t2, err := svc.GetContainerToken("mystack", "web-1")
	if err != nil {
		t.Fatalf("GetContainerToken: %v", err)
	}
	if t1 == t2 {
		t.Error("expected unique tokens, got identical tokens")
	}
	if len(t1) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d", len(t1))
	}
}

func TestStackService_ValidateContainerToken_Valid(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetContainerToken("mystack", "web-1")
	ct, ok := svc.ValidateContainerToken(token)

	if !ok {
		t.Fatal("expected token to be valid")
	}
	if ct.StackID != "mystack" {
		t.Errorf("StackID: got %q, want %q", ct.StackID, "mystack")
	}
	if ct.ContainerName != "web-1" {
		t.Errorf("ContainerName: got %q, want %q", ct.ContainerName, "web-1")
	}
}

func TestStackService_ValidateContainerToken_OneTimeUse(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetContainerToken("mystack", "web-1")

	_, ok1 := svc.ValidateContainerToken(token)
	if !ok1 {
		t.Fatal("first validation should succeed")
	}

	_, ok2 := svc.ValidateContainerToken(token)
	if ok2 {
		t.Error("second validation should fail — token is one-time use")
	}
}

func TestStackService_ValidateContainerToken_InvalidToken(t *testing.T) {
	svc := newTestStackService()

	_, ok := svc.ValidateContainerToken("notarealtoken")
	if ok {
		t.Error("expected invalid token to fail validation")
	}
}

func TestStackService_ValidateContainerToken_Expired(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetContainerToken("mystack", "web-1")

	// Manually back-date the token so it looks expired
	svc.tokensMu.Lock()
	ct := svc.containerTokens[token]
	ct.CreatedAt = time.Now().Add(-60 * time.Second) // 60s ago
	svc.containerTokens[token] = ct
	svc.tokensMu.Unlock()

	_, ok := svc.ValidateContainerToken(token)
	if ok {
		t.Error("expired token should fail validation")
	}
}

func TestStackService_GetStackLogsToken_GeneratesUniqueTokens(t *testing.T) {
	svc := newTestStackService()

	t1, err := svc.GetStackLogsToken("mystack")
	if err != nil {
		t.Fatalf("GetStackLogsToken: %v", err)
	}
	t2, err := svc.GetStackLogsToken("mystack")
	if err != nil {
		t.Fatalf("GetStackLogsToken: %v", err)
	}
	if t1 == t2 {
		t.Error("expected unique tokens, got identical tokens")
	}
	if len(t1) != 64 {
		t.Errorf("expected 64-char hex token, got len=%d", len(t1))
	}
}

func TestStackService_ValidateStackLogsToken_Valid(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetStackLogsToken("mystack")
	lt, ok := svc.ValidateStackLogsToken(token)

	if !ok {
		t.Fatal("expected token to be valid")
	}
	if lt.StackID != "mystack" {
		t.Errorf("StackID: got %q, want %q", lt.StackID, "mystack")
	}
}

func TestStackService_ValidateStackLogsToken_OneTimeUse(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetStackLogsToken("mystack")

	_, ok1 := svc.ValidateStackLogsToken(token)
	if !ok1 {
		t.Fatal("first validation should succeed")
	}
	_, ok2 := svc.ValidateStackLogsToken(token)
	if ok2 {
		t.Error("second validation should fail — token is one-time use")
	}
}

func TestStackService_ValidateStackLogsToken_Expired(t *testing.T) {
	svc := newTestStackService()

	token, _ := svc.GetStackLogsToken("mystack")

	svc.tokensMu.Lock()
	lt := svc.logsTokens[token]
	lt.CreatedAt = time.Now().Add(-60 * time.Second)
	svc.logsTokens[token] = lt
	svc.tokensMu.Unlock()

	_, ok := svc.ValidateStackLogsToken(token)
	if ok {
		t.Error("expired token should fail validation")
	}
}

func TestStackService_MultipleTokensForDifferentStacks(t *testing.T) {
	svc := newTestStackService()

	tok1, _ := svc.GetContainerToken("stack-a", "container-1")
	tok2, _ := svc.GetContainerToken("stack-b", "container-2")

	ct1, ok1 := svc.ValidateContainerToken(tok1)
	if !ok1 {
		t.Fatal("tok1 should be valid")
	}
	ct2, ok2 := svc.ValidateContainerToken(tok2)
	if !ok2 {
		t.Fatal("tok2 should be valid")
	}

	if ct1.StackID != "stack-a" {
		t.Errorf("ct1 StackID: got %q, want stack-a", ct1.StackID)
	}
	if ct2.StackID != "stack-b" {
		t.Errorf("ct2 StackID: got %q, want stack-b", ct2.StackID)
	}
}

func TestStackService_GetAll_DelegatestoRepo(t *testing.T) {
	svc := newTestStackService()

	// Add two stacks via Create
	_, _ = svc.Create(context.Background(), &model.StackCreateRequest{Name: "a", Compose: "x"})
	_, _ = svc.Create(context.Background(), &model.StackCreateRequest{Name: "b", Compose: "x"})

	stacks, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(stacks) != 2 {
		t.Errorf("expected 2 stacks, got %d", len(stacks))
	}
}

func TestStackService_GetByID_NotFound(t *testing.T) {
	svc := newTestStackService()

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent stack")
	}
}

func TestStackService_CreateAndUpdate(t *testing.T) {
	svc := newTestStackService()

	st, err := svc.Create(context.Background(), &model.StackCreateRequest{Name: "mystack", Compose: "v1", Env: "A=1"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if st.ID != "mystack" {
		t.Errorf("ID: got %q, want mystack", st.ID)
	}

	updated, err := svc.Update(context.Background(), "mystack", &model.StackUpdateRequest{Compose: "v2", Env: "B=2"})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Compose != "v2" {
		t.Errorf("Compose after update: got %q, want v2", updated.Compose)
	}
	if updated.Env != "B=2" {
		t.Errorf("Env after update: got %q, want B=2", updated.Env)
	}
}

func TestStackService_Delete(t *testing.T) {
	svc := newTestStackService()

	_, _ = svc.Create(context.Background(), &model.StackCreateRequest{Name: "todelete", Compose: "x"})

	if err := svc.Delete(context.Background(), "todelete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := svc.GetByID(context.Background(), "todelete")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestStackService_ActionDelegation(t *testing.T) {
	repo := newMockStackRepo()
	svc := NewStackService(repo)

	// Start delegates
	if err := svc.Start(context.Background(), "any"); err != nil {
		t.Errorf("Start: unexpected error: %v", err)
	}

	// Repo-level error propagates
	sentinel := errors.New("start failed")
	repo.startErr = sentinel
	if err := svc.Start(context.Background(), "any"); !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got: %v", err)
	}

	repo.startErr = nil
	repo.stopErr = sentinel
	if err := svc.Stop(context.Background(), "any"); !errors.Is(err, sentinel) {
		t.Errorf("Stop: expected sentinel error, got: %v", err)
	}

	repo.stopErr = nil
	repo.restartErr = sentinel
	if err := svc.Restart(context.Background(), "any"); !errors.Is(err, sentinel) {
		t.Errorf("Restart: expected sentinel error, got: %v", err)
	}

	repo.restartErr = nil
	repo.updateImagesErr = sentinel
	if err := svc.UpdateImages(context.Background(), "any"); !errors.Is(err, sentinel) {
		t.Errorf("UpdateImages: expected sentinel error, got: %v", err)
	}

	repo.updateImagesErr = nil
	repo.downErr = sentinel
	if err := svc.Down(context.Background(), "any"); !errors.Is(err, sentinel) {
		t.Errorf("Down: expected sentinel error, got: %v", err)
	}
}
