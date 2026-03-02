package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestStackRepository_DisabledWhenNoDir(t *testing.T) {
	repo := NewStackRepository("")
	ctx := context.Background()

	_, err := repo.GetAll(ctx)
	if err != ErrStacksDisabled {
		t.Fatalf("GetAll with empty dir should return ErrStacksDisabled, got: %v", err)
	}

	_, err = repo.GetByID(ctx, "test")
	if err != ErrStacksDisabled {
		t.Fatalf("GetByID with empty dir should return ErrStacksDisabled, got: %v", err)
	}

	_, err = repo.Create(ctx, &model.StackCreateRequest{Name: "test", Compose: "version: '3'"})
	if err != ErrStacksDisabled {
		t.Fatalf("Create with empty dir should return ErrStacksDisabled, got: %v", err)
	}
}

func TestStackRepository_CreateAndRead(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	compose := "services:\n  web:\n    image: nginx\n"
	env := "PORT=8080\n"

	stack, err := repo.Create(ctx, &model.StackCreateRequest{
		Name:    "test-stack",
		Compose: compose,
		Env:     env,
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if stack.ID != "test-stack" {
		t.Errorf("expected ID=test-stack, got %q", stack.ID)
	}
	if stack.Compose != compose {
		t.Errorf("compose mismatch")
	}
	if stack.Env != env {
		t.Errorf("env mismatch")
	}
	if stack.Status != model.StackStatusStopped {
		t.Errorf("expected stopped status, got %s", stack.Status)
	}

	// Verify files on disk
	composeOnDisk, err := os.ReadFile(filepath.Join(dir, "test-stack", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("compose file not written: %v", err)
	}
	if string(composeOnDisk) != compose {
		t.Errorf("compose file content mismatch")
	}

	envOnDisk, err := os.ReadFile(filepath.Join(dir, "test-stack", ".env"))
	if err != nil {
		t.Fatalf(".env file not written: %v", err)
	}
	if string(envOnDisk) != env {
		t.Errorf(".env file content mismatch")
	}
}

func TestStackRepository_GetAll(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	names := []string{"alpha", "beta", "gamma"}
	for _, n := range names {
		_, err := repo.Create(ctx, &model.StackCreateRequest{Name: n, Compose: "services: {}"})
		if err != nil {
			t.Fatalf("Create %s failed: %v", n, err)
		}
	}

	stacks, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(stacks) != len(names) {
		t.Fatalf("expected %d stacks, got %d", len(names), len(stacks))
	}
}

func TestStackRepository_Update(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	_, err := repo.Create(ctx, &model.StackCreateRequest{Name: "mystack", Compose: "old compose"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := repo.Update(ctx, "mystack", &model.StackUpdateRequest{
		Compose: "new compose",
		Env:     "NEW_VAR=1",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Compose != "new compose" {
		t.Errorf("compose not updated")
	}
	if updated.Env != "NEW_VAR=1" {
		t.Errorf("env not updated")
	}
}

func TestStackRepository_UpdateNotFound(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	_, err := repo.Update(ctx, "nonexistent", &model.StackUpdateRequest{Compose: "x"})
	if err != ErrStackNotFound {
		t.Fatalf("expected ErrStackNotFound, got: %v", err)
	}
}

func TestStackRepository_Delete(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	_, err := repo.Create(ctx, &model.StackCreateRequest{Name: "todelete", Compose: "x"})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete without running docker (it won't be running in tests), Down will fail silently
	// The directory removal is what matters
	if err := repo.Delete(ctx, "todelete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(dir, "todelete")); !os.IsNotExist(statErr) {
		t.Errorf("stack directory still exists after delete")
	}
}

func TestStackRepository_InvalidName(t *testing.T) {
	dir := t.TempDir()
	repo := NewStackRepository(dir)
	ctx := context.Background()

	invalidNames := []string{"my stack", "my/stack", "my.stack", "my!stack"}
	for _, name := range invalidNames {
		_, err := repo.Create(ctx, &model.StackCreateRequest{Name: name, Compose: "x"})
		if err != ErrInvalidName {
			t.Errorf("expected ErrInvalidName for %q, got: %v", name, err)
		}
	}
}

func TestDeriveStatus(t *testing.T) {
	cases := []struct {
		containers []model.DockerContainer
		expected   model.StackStatus
	}{
		{nil, model.StackStatusStopped},
		{[]model.DockerContainer{}, model.StackStatusStopped},
		{
			[]model.DockerContainer{{State: "running"}, {State: "running"}},
			model.StackStatusRunning,
		},
		{
			[]model.DockerContainer{{State: "exited"}, {State: "exited"}},
			model.StackStatusStopped,
		},
		{
			[]model.DockerContainer{{State: "running"}, {State: "exited"}},
			model.StackStatusPartiallyRunning,
		},
	}

	for _, tc := range cases {
		got := deriveStatus(tc.containers)
		if got != tc.expected {
			t.Errorf("deriveStatus(%v) = %v, want %v", tc.containers, got, tc.expected)
		}
	}
}
