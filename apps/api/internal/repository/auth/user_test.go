package auth

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL,
		mfa_secret TEXT,
		mfa_enabled INTEGER DEFAULT 0,
		mfa_backup_codes TEXT,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL,
		last_login_at INTEGER,
		is_active INTEGER DEFAULT 1
	);
	CREATE TABLE refresh_tokens (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		token_hash TEXT NOT NULL,
		expires_at INTEGER NOT NULL,
		revoked INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL
	);
	CREATE TABLE api_keys (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		key_hash TEXT NOT NULL,
		prefix TEXT NOT NULL,
		permissions TEXT,
		last_used_at INTEGER,
		expires_at INTEGER,
		created_at INTEGER NOT NULL
	);
	CREATE TABLE audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id TEXT,
		action TEXT NOT NULL,
		resource_type TEXT,
		resource_id TEXT,
		details TEXT,
		ip_address TEXT,
		user_agent TEXT,
		status TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}

func TestUserRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	user, err := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", user.Email)
	}
	if user.Role != RoleAdmin {
		t.Errorf("Role = %q, want admin", user.Role)
	}
	if !user.IsActive {
		t.Error("expected IsActive=true")
	}
}

func TestUserRepository_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}

	_, err = repo.Create(ctx, "alice@example.com", "hashedpw2", RoleViewer)
	if err == nil {
		t.Error("expected error for duplicate email")
	}
	if err.Error() != "email already exists" {
		t.Errorf("error = %q, want 'email already exists'", err.Error())
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	user, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Email = %q, want alice@example.com", user.Email)
	}
	if user.PasswordHash != "hashedpw" {
		t.Errorf("PasswordHash = %q, want hashedpw", user.PasswordHash)
	}
}

func TestUserRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
	if err.Error() != "user not found" {
		t.Errorf("error = %q, want 'user not found'", err.Error())
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "bob@example.com", "hashedpw", RoleViewer)

	user, err := repo.GetByEmail(ctx, "bob@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if user.Role != RoleViewer {
		t.Errorf("Role = %q, want viewer", user.Role)
	}
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nobody@example.com")
	if err == nil {
		t.Error("expected error for nonexistent email")
	}
}

func TestUserRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	created.Role = RoleOperator
	created.Email = "alice-updated@example.com"
	if err := repo.Update(ctx, created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	fetched, _ := repo.GetByEmail(ctx, "alice-updated@example.com")
	if fetched.Role != RoleOperator {
		t.Errorf("Role = %q, want operator", fetched.Role)
	}
}

func TestUserRepository_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	err := repo.Update(ctx, &User{ID: "nonexistent", Email: "x@x.com"})
	if err == nil || err.Error() != "user not found" {
		t.Errorf("expected 'user not found', got %v", err)
	}
}

func TestUserRepository_UpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	if err := repo.UpdateLastLogin(ctx, created.ID); err != nil {
		t.Fatalf("UpdateLastLogin: %v", err)
	}

	user, _ := repo.GetByID(ctx, created.ID)
	if user.LastLoginAt == nil {
		t.Error("expected LastLoginAt to be set")
	}
}

func TestUserRepository_SetMFAEnabled(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	secret := "JBSWY3DPEHPK3PXP"
	codes := `["abc","def"]`
	if err := repo.SetMFAEnabled(ctx, created.ID, true, &secret, &codes); err != nil {
		t.Fatalf("SetMFAEnabled: %v", err)
	}

	user, _ := repo.GetByID(ctx, created.ID)
	if !user.MFAEnabled {
		t.Error("expected MFAEnabled=true")
	}
	if user.MFASecret == nil || *user.MFASecret != secret {
		t.Error("expected MFASecret to be set")
	}
}

func TestUserRepository_SetMFAEnabled_Disable(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	// Enable first
	secret := "JBSWY3DPEHPK3PXP"
	repo.SetMFAEnabled(ctx, created.ID, true, &secret, nil)

	// Disable
	if err := repo.SetMFAEnabled(ctx, created.ID, false, nil, nil); err != nil {
		t.Fatalf("SetMFAEnabled(disable): %v", err)
	}

	user, _ := repo.GetByID(ctx, created.ID)
	if user.MFAEnabled {
		t.Error("expected MFAEnabled=false after disable")
	}
}

func TestUserRepository_Delete_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	created, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Should not be findable via GetByID (which filters is_active=TRUE)
	_, err := repo.GetByID(ctx, created.ID)
	if err == nil {
		t.Error("expected error after soft delete")
	}
}

func TestUserRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	if err == nil || err.Error() != "user not found" {
		t.Errorf("expected 'user not found', got %v", err)
	}
}

func TestUserRepository_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	repo.Create(ctx, "bob@example.com", "hashedpw", RoleViewer)
	repo.Create(ctx, "charlie@example.com", "hashedpw", RoleOperator)

	users, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestUserRepository_List_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	alice, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	repo.Create(ctx, "bob@example.com", "hashedpw", RoleViewer)

	repo.Delete(ctx, alice.ID)

	users, _ := repo.List(ctx)
	if len(users) != 1 {
		t.Errorf("expected 1 user after delete, got %d", len(users))
	}
}

func TestUserRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	count, _ := repo.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	repo.Create(ctx, "bob@example.com", "hashedpw", RoleViewer)

	count, _ = repo.Count(ctx)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestUserRepository_Count_ExcludesDeleted(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	alice, _ := repo.Create(ctx, "alice@example.com", "hashedpw", RoleAdmin)
	repo.Create(ctx, "bob@example.com", "hashedpw", RoleViewer)

	repo.Delete(ctx, alice.ID)

	count, _ := repo.Count(ctx)
	if count != 1 {
		t.Errorf("expected 1 after delete, got %d", count)
	}
}

func TestUserRepository_GetFirstAdmin(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "viewer@example.com", "hashedpw", RoleViewer)
	repo.Create(ctx, "admin@example.com", "hashedpw", RoleAdmin)

	admin, err := repo.GetFirstAdmin(ctx)
	if err != nil {
		t.Fatalf("GetFirstAdmin: %v", err)
	}
	if admin.Email != "admin@example.com" {
		t.Errorf("Email = %q, want admin@example.com", admin.Email)
	}
}

func TestUserRepository_GetFirstAdmin_NoneExists(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)
	ctx := context.Background()

	repo.Create(ctx, "viewer@example.com", "hashedpw", RoleViewer)

	_, err := repo.GetFirstAdmin(ctx)
	if err == nil {
		t.Error("expected error when no admin exists")
	}
}

func TestIsDuplicateEmail(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"UNIQUE constraint failed: users.email", true},
		{"UNIQUE constraint failed: users.id", false},
		{"some other error", false},
		{"", false},
	}

	for _, tt := range tests {
		got := isDuplicateEmail(java_error(tt.msg))
		if got != tt.want {
			t.Errorf("isDuplicateEmail(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

// java_error is a simple error type for testing
type java_error string

func (e java_error) Error() string { return string(e) }
