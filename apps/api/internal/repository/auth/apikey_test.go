package auth

import (
	"context"
	"testing"
	"time"
)

func TestAPIKeyRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	key, err := repo.Create(ctx, "user-1", "my-key", "raw-api-key-value", "lab_abc", []string{"read", "write"}, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if key.ID == "" {
		t.Error("expected non-empty ID")
	}
	if key.Name != "my-key" {
		t.Errorf("Name = %q, want my-key", key.Name)
	}
	if key.Prefix != "lab_abc" {
		t.Errorf("Prefix = %q, want lab_abc", key.Prefix)
	}
	if len(key.Permissions) != 2 {
		t.Errorf("Permissions count = %d, want 2", len(key.Permissions))
	}
	if key.KeyHash == "raw-api-key-value" {
		t.Error("KeyHash should be bcrypt hash, not raw key")
	}
}

func TestAPIKeyRepository_Create_WithExpiry(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	expires := time.Now().Add(30 * 24 * time.Hour)
	key, err := repo.Create(ctx, "user-1", "expiring-key", "raw-key", "lab_xyz", []string{"read"}, &expires)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if key.ExpiresAt == nil {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestAPIKeyRepository_GetByKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	rawKey := "my-secret-api-key"
	created, _ := repo.Create(ctx, "user-1", "test-key", rawKey, "lab_abc", []string{"read", "write"}, nil)

	found, err := repo.GetByKey(ctx, rawKey)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.Name != "test-key" {
		t.Errorf("Name = %q, want test-key", found.Name)
	}
	if len(found.Permissions) != 2 || found.Permissions[0] != "read" {
		t.Errorf("Permissions = %v, want [read write]", found.Permissions)
	}
}

func TestAPIKeyRepository_GetByKey_WrongKey(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "test-key", "correct-key", "lab_abc", nil, nil)

	_, err := repo.GetByKey(ctx, "wrong-key")
	if err == nil {
		t.Error("expected error for wrong key")
	}
	if err.Error() != "invalid or expired API key" {
		t.Errorf("error = %q, want 'invalid or expired API key'", err.Error())
	}
}

func TestAPIKeyRepository_GetByKey_Expired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	repo.Create(ctx, "user-1", "expired-key", "the-key", "lab_abc", nil, &expired)

	_, err := repo.GetByKey(ctx, "the-key")
	if err == nil {
		t.Error("expected error for expired key")
	}
}

func TestAPIKeyRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "my-key", "raw-key", "lab_abc", []string{"admin"}, nil)

	found, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if found.Name != "my-key" {
		t.Errorf("Name = %q, want my-key", found.Name)
	}
	if len(found.Permissions) != 1 || found.Permissions[0] != "admin" {
		t.Errorf("Permissions = %v, want [admin]", found.Permissions)
	}
}

func TestAPIKeyRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil || err.Error() != "API key not found" {
		t.Errorf("expected 'API key not found', got %v", err)
	}
}

func TestAPIKeyRepository_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "my-key", "raw-key", "lab_abc", nil, nil)

	if err := repo.UpdateLastUsed(ctx, created.ID); err != nil {
		t.Fatalf("UpdateLastUsed: %v", err)
	}

	found, _ := repo.GetByID(ctx, created.ID)
	if found.LastUsedAt == nil {
		t.Error("expected LastUsedAt to be set")
	}
}

func TestAPIKeyRepository_Revoke(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "my-key", "raw-key", "lab_abc", nil, nil)

	if err := repo.Revoke(ctx, created.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	// Should be gone
	_, err := repo.GetByID(ctx, created.ID)
	if err == nil {
		t.Error("expected error after revoke")
	}
}

func TestAPIKeyRepository_Revoke_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	err := repo.Revoke(ctx, "nonexistent")
	if err == nil || err.Error() != "API key not found" {
		t.Errorf("expected 'API key not found', got %v", err)
	}
}

func TestAPIKeyRepository_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "key-a", "raw-a", "lab_a", []string{"read"}, nil)
	repo.Create(ctx, "user-1", "key-b", "raw-b", "lab_b", []string{"write"}, nil)
	repo.Create(ctx, "user-2", "key-c", "raw-c", "lab_c", nil, nil)

	keys, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys for user-1, got %d", len(keys))
	}
}

func TestAPIKeyRepository_ListByUser_ExcludesExpired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	expired := time.Now().Add(-1 * time.Hour)
	repo.Create(ctx, "user-1", "expired-key", "raw-exp", "lab_e", nil, &expired)
	repo.Create(ctx, "user-1", "valid-key", "raw-val", "lab_v", nil, nil)

	keys, _ := repo.ListByUser(ctx, "user-1")
	if len(keys) != 1 {
		t.Errorf("expected 1 active key, got %d", len(keys))
	}
}

func TestAPIKeyRepository_HasPermission(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "my-key", "raw-key", "lab_abc", []string{"read", "write"}, nil)

	has, err := repo.HasPermission(ctx, created.ID, "read")
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !has {
		t.Error("expected to have 'read' permission")
	}

	has, _ = repo.HasPermission(ctx, created.ID, "delete")
	if has {
		t.Error("expected NOT to have 'delete' permission")
	}
}

func TestAPIKeyRepository_HasPermission_Wildcard(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "admin-key", "raw-key", "lab_abc", []string{"*"}, nil)

	has, err := repo.HasPermission(ctx, created.ID, "anything")
	if err != nil {
		t.Fatalf("HasPermission: %v", err)
	}
	if !has {
		t.Error("expected wildcard to grant any permission")
	}
}

func TestAPIKeyRepository_HasPermission_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	_, err := repo.HasPermission(ctx, "nonexistent", "read")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestAPIKeyRepository_CountByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAPIKeyRepository(db)
	ctx := context.Background()

	count, _ := repo.CountByUser(ctx, "user-1")
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	repo.Create(ctx, "user-1", "key-a", "raw-a", "lab_a", nil, nil)
	repo.Create(ctx, "user-1", "key-b", "raw-b", "lab_b", nil, nil)

	count, _ = repo.CountByUser(ctx, "user-1")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}
