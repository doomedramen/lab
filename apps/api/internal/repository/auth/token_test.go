package auth

import (
	"context"
	"testing"
	"time"
)

func TestRefreshTokenRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	token, err := repo.Create(ctx, "user-1", "my-raw-token", time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token.ID == "" {
		t.Error("expected non-empty ID")
	}
	if token.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", token.UserID)
	}
	if token.TokenHash == "" {
		t.Error("expected non-empty TokenHash")
	}
	if token.TokenHash == "my-raw-token" {
		t.Error("TokenHash should be bcrypt hash, not raw token")
	}
}

func TestRefreshTokenRepository_GetByToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	rawToken := "secret-refresh-token"
	created, err := repo.Create(ctx, "user-1", rawToken, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	found, err := repo.GetByToken(ctx, rawToken)
	if err != nil {
		t.Fatalf("GetByToken: %v", err)
	}
	if found.ID != created.ID {
		t.Errorf("ID = %q, want %q", found.ID, created.ID)
	}
	if found.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", found.UserID)
	}
}

func TestRefreshTokenRepository_GetByToken_WrongToken(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "correct-token", time.Now().Add(24*time.Hour))

	_, err := repo.GetByToken(ctx, "wrong-token")
	if err == nil {
		t.Error("expected error for wrong token")
	}
	if err.Error() != "invalid or expired refresh token" {
		t.Errorf("error = %q, want 'invalid or expired refresh token'", err.Error())
	}
}

func TestRefreshTokenRepository_GetByToken_Expired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create token that already expired
	repo.Create(ctx, "user-1", "expired-token", time.Now().Add(-1*time.Hour))

	_, err := repo.GetByToken(ctx, "expired-token")
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestRefreshTokenRepository_GetByToken_Revoked(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	rawToken := "revokable-token"
	created, _ := repo.Create(ctx, "user-1", rawToken, time.Now().Add(24*time.Hour))
	repo.Revoke(ctx, created.ID)

	_, err := repo.GetByToken(ctx, rawToken)
	if err == nil {
		t.Error("expected error for revoked token")
	}
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "user-1", "some-token", time.Now().Add(24*time.Hour))

	if err := repo.Revoke(ctx, created.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}

func TestRefreshTokenRepository_Revoke_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	err := repo.Revoke(ctx, "nonexistent")
	if err == nil || err.Error() != "refresh token not found" {
		t.Errorf("expected 'refresh token not found', got %v", err)
	}
}

func TestRefreshTokenRepository_RevokeAllUserTokens(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "token-a", time.Now().Add(24*time.Hour))
	repo.Create(ctx, "user-1", "token-b", time.Now().Add(24*time.Hour))
	repo.Create(ctx, "user-2", "token-c", time.Now().Add(24*time.Hour))

	if err := repo.RevokeAllUserTokens(ctx, "user-1"); err != nil {
		t.Fatalf("RevokeAllUserTokens: %v", err)
	}

	// user-1 tokens should be gone
	count, _ := repo.CountByUser(ctx, "user-1")
	if count != 0 {
		t.Errorf("user-1 count = %d, want 0", count)
	}

	// user-2 token should remain
	count, _ = repo.CountByUser(ctx, "user-2")
	if count != 1 {
		t.Errorf("user-2 count = %d, want 1", count)
	}
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "expired-1", time.Now().Add(-2*time.Hour))
	repo.Create(ctx, "user-1", "expired-2", time.Now().Add(-1*time.Hour))
	repo.Create(ctx, "user-1", "valid-token", time.Now().Add(24*time.Hour))

	deleted, err := repo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}
}

func TestRefreshTokenRepository_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "token-a", time.Now().Add(24*time.Hour))
	repo.Create(ctx, "user-1", "token-b", time.Now().Add(24*time.Hour))
	repo.Create(ctx, "user-2", "token-c", time.Now().Add(24*time.Hour))

	tokens, err := repo.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens for user-1, got %d", len(tokens))
	}
}

func TestRefreshTokenRepository_ListByUser_ExcludesExpired(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "user-1", "expired", time.Now().Add(-1*time.Hour))
	repo.Create(ctx, "user-1", "valid", time.Now().Add(24*time.Hour))

	tokens, _ := repo.ListByUser(ctx, "user-1")
	if len(tokens) != 1 {
		t.Errorf("expected 1 active token, got %d", len(tokens))
	}
}

func TestRefreshTokenRepository_CountByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	count, _ := repo.CountByUser(ctx, "user-1")
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	repo.Create(ctx, "user-1", "token-a", time.Now().Add(24*time.Hour))
	repo.Create(ctx, "user-1", "token-b", time.Now().Add(24*time.Hour))

	count, _ = repo.CountByUser(ctx, "user-1")
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}
