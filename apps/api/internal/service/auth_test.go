package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	authrepo "github.com/doomedramen/lab/apps/api/internal/repository/auth"
	_ "modernc.org/sqlite"
	"github.com/pquerna/otp/totp"
)

// Test helper functions

func setupAuthTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	// Create tables
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

	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		jti TEXT UNIQUE NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		device_name TEXT,
		issued_at INTEGER NOT NULL,
		last_seen_at INTEGER NOT NULL,
		expires_at INTEGER NOT NULL,
		revoked INTEGER DEFAULT 0
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

func setupAuthService(t *testing.T) (*AuthService, *sql.DB) {
	db := setupAuthTestDB(t)

	userRepo := authrepo.NewUserRepository(db)
	tokenRepo := authrepo.NewRefreshTokenRepository(db)
	apiKeyRepo := authrepo.NewAPIKeyRepository(db)
	auditRepo := authrepo.NewAuditLogRepository(db)
	sessionRepo := authrepo.NewSessionRepository(db)

	config := AuthServiceConfig{
		JWTSecret:       []byte("test-secret-key-for-auth-service"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api-test",
	}

	svc := NewAuthService(userRepo, tokenRepo, apiKeyRepo, auditRepo, sessionRepo, config)
	return svc, db
}

// Tests

func TestAuthService_Register_ValidInput(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	output, err := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if output.User == nil {
		t.Fatal("User should not be nil")
	}
	if output.User.Email != "test@example.com" {
		t.Errorf("Email: got %q, want test@example.com", output.User.Email)
	}
	if output.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if output.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
}

func TestAuthService_Register_WeakPassword(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	_, err := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "weak",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	if err == nil {
		t.Error("expected error for weak password")
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	input := RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	}

	// First registration should succeed
	_, err := svc.Register(ctx, input)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration with same email should fail
	_, err = svc.Register(ctx, input)
	if err == nil {
		t.Error("expected error for duplicate email")
	}
}

func TestAuthService_Login_ValidCredentials(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user first
	_, _ = svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Login
	output, err := svc.Login(ctx, LoginInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if output.User.Email != "test@example.com" {
		t.Errorf("Email: got %q, want test@example.com", output.User.Email)
	}
	if output.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if output.MFARequired {
		t.Error("MFARequired should be false")
	}
}

func TestAuthService_Login_InvalidEmail(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Login(ctx, LoginInput{
		Email:     "nonexistent@example.com",
		Password:  "ValidPass123!",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	if err == nil {
		t.Error("expected error for invalid email")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("error message: got %q, want 'invalid credentials'", err.Error())
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user first
	_, _ = svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Login with wrong password
	_, err := svc.Login(ctx, LoginInput{
		Email:     "test@example.com",
		Password:  "WrongPassword123!",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	if err == nil {
		t.Error("expected error for invalid password")
	}
	if err.Error() != "invalid credentials" {
		t.Errorf("error message: got %q, want 'invalid credentials'", err.Error())
	}
}

func TestAuthService_Logout(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Logout
	err := svc.Logout(ctx, regOutput.User.ID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Logout failed: %v", err)
	}

	// Refresh token should no longer work
	_, err = svc.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: regOutput.RefreshToken,
	})
	if err == nil {
		t.Error("expected error when using revoked refresh token")
	}
}

func TestAuthService_RefreshToken_Valid(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Refresh token
	refreshOutput, err := svc.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: regOutput.RefreshToken,
	})
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}

	if refreshOutput.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	if refreshOutput.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	if refreshOutput.RefreshToken == regOutput.RefreshToken {
		t.Error("new refresh token should be different (rotation)")
	}
}

func TestAuthService_RefreshToken_Invalid(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: "invalid-token",
	})
	if err == nil {
		t.Error("expected error for invalid refresh token")
	}
}

func TestAuthService_SetupMFA(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Setup MFA
	mfaOutput, err := svc.SetupMFA(ctx, SetupMFAInput{
		UserID:    regOutput.User.ID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("SetupMFA failed: %v", err)
	}

	if mfaOutput.Secret == nil {
		t.Error("Secret should not be nil")
	}
	if mfaOutput.Secret.Secret == "" {
		t.Error("Secret.Secret should not be empty")
	}
	if len(mfaOutput.BackupCodes) != 10 {
		t.Errorf("expected 10 backup codes, got %d", len(mfaOutput.BackupCodes))
	}
}

func TestAuthService_SetupMFA_UserNotFound(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.SetupMFA(ctx, SetupMFAInput{
		UserID:    "nonexistent-id",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestAuthService_EnableMFA(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Setup MFA
	mfaOutput, _ := svc.SetupMFA(ctx, SetupMFAInput{
		UserID:    regOutput.User.ID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Generate a valid TOTP code
	validCode, _ := generateTestTOTPCode(mfaOutput.Secret.Secret)

	// Enable MFA
	err := svc.EnableMFA(ctx, EnableMFAInput{
		UserID:    regOutput.User.ID,
		MFACode:   validCode,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("EnableMFA failed: %v", err)
	}

	// Verify user now has MFA enabled
	user, _ := svc.GetCurrentUser(ctx, regOutput.User.ID)
	if !user.MFAEnabled {
		t.Error("MFA should be enabled")
	}
}

func TestAuthService_EnableMFA_InvalidCode(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Setup MFA
	_, _ = svc.SetupMFA(ctx, SetupMFAInput{
		UserID:    regOutput.User.ID,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Try to enable with invalid code
	err := svc.EnableMFA(ctx, EnableMFAInput{
		UserID:    regOutput.User.ID,
		MFACode:   "000000",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err == nil {
		t.Error("expected error for invalid MFA code")
	}
}

func TestAuthService_CreateAPIKey(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Create API key
	keyOutput, err := svc.CreateAPIKey(ctx, CreateAPIKeyInput{
		UserID:      regOutput.User.ID,
		Name:        "test-key",
		Permissions: []string{"vm:read", "vm:write"},
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	})
	if err != nil {
		t.Fatalf("CreateAPIKey failed: %v", err)
	}

	if keyOutput.APIKey == nil {
		t.Fatal("APIKey should not be nil")
	}
	if keyOutput.RawKey == "" {
		t.Error("RawKey should not be empty")
	}
	if keyOutput.APIKey.Name != "test-key" {
		t.Errorf("Name: got %q, want test-key", keyOutput.APIKey.Name)
	}
}

func TestAuthService_ValidateAPIKey(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Create API key
	keyOutput, _ := svc.CreateAPIKey(ctx, CreateAPIKeyInput{
		UserID:      regOutput.User.ID,
		Name:        "test-key",
		Permissions: []string{"vm:read"},
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	})

	// Validate API key
	user, err := svc.ValidateAPIKey(ctx, keyOutput.RawKey)
	if err != nil {
		t.Fatalf("ValidateAPIKey failed: %v", err)
	}

	if user.ID != regOutput.User.ID {
		t.Errorf("User ID: got %q, want %q", user.ID, regOutput.User.ID)
	}
}

func TestAuthService_ValidateAPIKey_Invalid(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.ValidateAPIKey(ctx, "labkey_invalidkey")
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestAuthService_RevokeAPIKey(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Create API key
	keyOutput, _ := svc.CreateAPIKey(ctx, CreateAPIKeyInput{
		UserID:      regOutput.User.ID,
		Name:        "test-key",
		Permissions: []string{"vm:read"},
		IPAddress:   "127.0.0.1",
		UserAgent:   "test-agent",
	})

	// Revoke API key
	err := svc.RevokeAPIKey(ctx, keyOutput.APIKey.ID)
	if err != nil {
		t.Fatalf("RevokeAPIKey failed: %v", err)
	}

	// Key should no longer validate
	_, err = svc.ValidateAPIKey(ctx, keyOutput.RawKey)
	if err == nil {
		t.Error("expected error for revoked API key")
	}
}

func TestAuthService_ListAPIKeys(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Create multiple API keys
	svc.CreateAPIKey(ctx, CreateAPIKeyInput{UserID: regOutput.User.ID, Name: "key1"})
	svc.CreateAPIKey(ctx, CreateAPIKeyInput{UserID: regOutput.User.ID, Name: "key2"})

	// List API keys
	keys, err := svc.ListAPIKeys(ctx, regOutput.User.ID)
	if err != nil {
		t.Fatalf("ListAPIKeys failed: %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestAuthService_GetCurrentUser(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Get current user
	user, err := svc.GetCurrentUser(ctx, regOutput.User.ID)
	if err != nil {
		t.Fatalf("GetCurrentUser failed: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Email: got %q, want test@example.com", user.Email)
	}
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Validate access token
	claims, err := svc.ValidateAccessToken(ctx, regOutput.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if claims.Email != "test@example.com" {
		t.Errorf("Email: got %q, want test@example.com", claims.Email)
	}
}

func TestAuthService_Login_MFARequired(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Setup and enable MFA
	mfaOutput, _ := svc.SetupMFA(ctx, SetupMFAInput{UserID: regOutput.User.ID})
	validCode, _ := generateTestTOTPCode(mfaOutput.Secret.Secret)
	svc.EnableMFA(ctx, EnableMFAInput{UserID: regOutput.User.ID, MFACode: validCode})

	// Login without MFA code
	output, err := svc.Login(ctx, LoginInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if !output.MFARequired {
		t.Error("MFARequired should be true")
	}
	if output.AccessToken != "" {
		t.Error("AccessToken should be empty when MFA is required")
	}
}

func TestAuthService_Login_MFAFlow(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()

	// Register user
	regOutput, _ := svc.Register(ctx, RegisterInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		Role:      authrepo.RoleAdmin,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})

	// Setup and enable MFA
	mfaOutput, _ := svc.SetupMFA(ctx, SetupMFAInput{UserID: regOutput.User.ID})
	validCode, _ := generateTestTOTPCode(mfaOutput.Secret.Secret)
	svc.EnableMFA(ctx, EnableMFAInput{UserID: regOutput.User.ID, MFACode: validCode})

	// Login with MFA code
	output, err := svc.Login(ctx, LoginInput{
		Email:     "test@example.com",
		Password:  "ValidPass123!",
		MFACode:   validCode,
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("Login with MFA failed: %v", err)
	}

	if output.MFARequired {
		t.Error("MFARequired should be false")
	}
	if output.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
}

func TestDefaultAuthServiceConfig(t *testing.T) {
	config := DefaultAuthServiceConfig()

	// JWTSecret should be nil - must be set explicitly
	if config.JWTSecret != nil {
		t.Error("default JWTSecret should be nil, must be set explicitly")
	}
	if config.AccessTokenExp != 15*time.Minute {
		t.Errorf("AccessTokenExp: got %v, want 15m", config.AccessTokenExp)
	}
	if config.RefreshTokenExp != 7*24*time.Hour {
		t.Errorf("RefreshTokenExp: got %v, want 168h", config.RefreshTokenExp)
	}
	if config.Issuer != "lab-api" {
		t.Errorf("Issuer: got %q, want lab-api", config.Issuer)
	}
}

func TestAuthService_UpdateCurrentUser_ChangePassword(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterInput{
		Email:     "pwchange@example.com",
		Password:  "OldPass123!",
		Role:      authrepo.RoleOperator,
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})

	// Correct current password → should succeed
	updated, err := svc.UpdateCurrentUser(ctx, UpdateCurrentUserInput{
		UserID:          reg.User.ID,
		CurrentPassword: "OldPass123!",
		NewPassword:     "NewPass456!",
		IPAddress:       "127.0.0.1",
		UserAgent:       "test",
	})
	if err != nil {
		t.Fatalf("UpdateCurrentUser: %v", err)
	}
	if updated.Email != "pwchange@example.com" {
		t.Errorf("email changed unexpectedly: %q", updated.Email)
	}

	// Old password must no longer work for login
	_, err = svc.Login(ctx, LoginInput{
		Email:     "pwchange@example.com",
		Password:  "OldPass123!",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})
	if err == nil {
		t.Error("expected login with old password to fail")
	}

	// New password should work
	_, err = svc.Login(ctx, LoginInput{
		Email:     "pwchange@example.com",
		Password:  "NewPass456!",
		IPAddress: "127.0.0.1",
		UserAgent: "test",
	})
	if err != nil {
		t.Errorf("login with new password: %v", err)
	}
}

func TestAuthService_UpdateCurrentUser_WrongCurrentPassword(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterInput{
		Email:    "wrongpw@example.com",
		Password: "Correct123!",
		Role:     authrepo.RoleViewer,
	})

	_, err := svc.UpdateCurrentUser(ctx, UpdateCurrentUserInput{
		UserID:          reg.User.ID,
		CurrentPassword: "WrongPassword!",
		NewPassword:     "NewPass456!",
	})
	if err == nil {
		t.Fatal("expected error with wrong current password")
	}
	if err.Error() != "current password is incorrect" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthService_UpdateCurrentUser_MissingCurrentPassword(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterInput{
		Email:    "nopw@example.com",
		Password: "Valid123!",
		Role:     authrepo.RoleViewer,
	})

	_, err := svc.UpdateCurrentUser(ctx, UpdateCurrentUserInput{
		UserID:      reg.User.ID,
		NewPassword: "NewPass456!",
		// CurrentPassword intentionally empty
	})
	if err == nil {
		t.Fatal("expected error when current password missing")
	}
	if err.Error() != "current password is required to change password" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAuthService_UpdateCurrentUser_ChangeEmail(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	reg, _ := svc.Register(ctx, RegisterInput{
		Email:    "old@example.com",
		Password: "Valid123!",
		Role:     authrepo.RoleOperator,
	})

	updated, err := svc.UpdateCurrentUser(ctx, UpdateCurrentUserInput{
		UserID: reg.User.ID,
		Email:  "new@example.com",
	})
	if err != nil {
		t.Fatalf("UpdateCurrentUser email change: %v", err)
	}
	if updated.Email != "new@example.com" {
		t.Errorf("email = %q, want new@example.com", updated.Email)
	}
}

func TestAuthService_UpdateCurrentUser_DuplicateEmail(t *testing.T) {
	svc, db := setupAuthService(t)
	defer db.Close()

	ctx := context.Background()
	svc.Register(ctx, RegisterInput{Email: "taken@example.com", Password: "Valid123!", Role: authrepo.RoleViewer})
	reg2, _ := svc.Register(ctx, RegisterInput{Email: "other@example.com", Password: "Valid123!", Role: authrepo.RoleViewer})

	_, err := svc.UpdateCurrentUser(ctx, UpdateCurrentUserInput{
		UserID: reg2.User.ID,
		Email:  "taken@example.com",
	})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
	if err.Error() != "email already exists" {
		t.Errorf("unexpected error: %v", err)
	}
}

// Helper to generate a valid TOTP code for testing
func generateTestTOTPCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}
