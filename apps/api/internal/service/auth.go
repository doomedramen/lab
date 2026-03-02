package service

import (
	"context"
	"errors"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	authpkg "github.com/doomedramen/lab/apps/api/pkg/auth"
)

// AuthService provides authentication business logic
type AuthService struct {
	userRepo       *auth.UserRepository
	tokenRepo      *auth.RefreshTokenRepository
	apiKeyRepo     *auth.APIKeyRepository
	auditRepo      *auth.AuditLogRepository
	sessionRepo    *auth.SessionRepository
	jwt            *authpkg.JWT
	password       *authpkg.Password
	mfa            *authpkg.MFA
	issuer         string
}

// AuthServiceConfig holds auth service configuration
type AuthServiceConfig struct {
	JWTSecret       []byte
	AccessTokenExp  time.Duration
	RefreshTokenExp time.Duration
	Issuer          string
}

// DefaultAuthServiceConfig returns a default config (INSECURE - for development only)
func DefaultAuthServiceConfig() AuthServiceConfig {
	return AuthServiceConfig{
		JWTSecret:       []byte("change-me-in-production"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api",
	}
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *auth.UserRepository,
	tokenRepo *auth.RefreshTokenRepository,
	apiKeyRepo *auth.APIKeyRepository,
	auditRepo *auth.AuditLogRepository,
	sessionRepo *auth.SessionRepository,
	config AuthServiceConfig,
) *AuthService {
	jwtConfig := authpkg.Config{
		SecretKey:       config.JWTSecret,
		AccessTokenExp:  config.AccessTokenExp,
		RefreshTokenExp: config.RefreshTokenExp,
		Issuer:          config.Issuer,
	}

	return &AuthService{
		userRepo:    userRepo,
		tokenRepo:   tokenRepo,
		apiKeyRepo:  apiKeyRepo,
		auditRepo:   auditRepo,
		sessionRepo: sessionRepo,
		jwt:         authpkg.NewJWT(jwtConfig),
		password:    authpkg.NewPassword(),
		mfa:         authpkg.NewMFA(config.Issuer),
		issuer:      config.Issuer,
	}
}

// RegisterInput holds registration input
type RegisterInput struct {
	Email      string
	Password   string
	Role       auth.Role
	IPAddress  string
	UserAgent  string
	DeviceName string
}

// RegisterOutput holds registration output
type RegisterOutput struct {
	User         *auth.User
	AccessToken  string
	RefreshToken string
	SessionID    string // Session ID for tracking
}

// Register creates a new user and returns tokens
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*RegisterOutput, error) {
	// Validate password strength
	if err := authpkg.ValidatePasswordStrength(input.Password); err != nil {
		return nil, err
	}

	// Hash password
	passwordHash, err := s.password.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := s.userRepo.Create(ctx, input.Email, passwordHash, input.Role)
	if err != nil {
		return nil, err
	}

	// Generate tokens
	accessToken, jti, err := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token
	expiresAt := time.Now().Add(s.jwt.GetRefreshTokenExpiry())
	_, err = s.tokenRepo.Create(ctx, user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	// Create session record
	var sessionID string
	if s.sessionRepo != nil {
		session, err := s.sessionRepo.Create(ctx, user.ID, jti, input.IPAddress, input.UserAgent, input.DeviceName, expiresAt)
		if err == nil {
			sessionID = session.ID
		}
	}

	// Log registration
	_ = s.auditRepo.LogLogin(ctx, user.ID, user.Email, input.IPAddress, input.UserAgent, true)

	return &RegisterOutput{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		SessionID:    sessionID,
	}, nil
}

// LoginInput holds login input
type LoginInput struct {
	Email      string
	Password   string
	MFACode    string // Optional TOTP code
	IPAddress  string
	UserAgent  string
	DeviceName string
}

// LoginOutput holds login output
type LoginOutput struct {
	User         *auth.User
	AccessToken  string
	RefreshToken string
	MFARequired  bool
	SessionID    string // Session ID for tracking
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*LoginOutput, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		// Log failed login attempt (use empty userID if user not found)
		_ = s.auditRepo.LogLogin(ctx, "", input.Email, input.IPAddress, input.UserAgent, false)
		return nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := s.password.Verify(input.Password, user.PasswordHash); err != nil {
		_ = s.auditRepo.LogLogin(ctx, user.ID, user.Email, input.IPAddress, input.UserAgent, false)
		return nil, errors.New("invalid credentials")
	}

	// Check if MFA is required
	if user.MFAEnabled {
		if input.MFACode == "" {
			return &LoginOutput{
				User:        user,
				MFARequired: true,
			}, nil
		}

		// Verify MFA code
		if user.MFASecret == nil {
			return nil, errors.New("MFA is enabled but no secret found")
		}

		if err := s.mfa.VerifyCode(*user.MFASecret, input.MFACode); err != nil {
			_ = s.auditRepo.LogLogin(ctx, user.ID, user.Email, input.IPAddress, input.UserAgent, false)
			return nil, errors.New("invalid MFA code")
		}
	}

	// Generate access token
	accessToken, jti, err := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token
	expiresAt := time.Now().Add(s.jwt.GetRefreshTokenExpiry())
	_, err = s.tokenRepo.Create(ctx, user.ID, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	// Create session record
	var sessionID string
	if s.sessionRepo != nil {
		session, err := s.sessionRepo.Create(ctx, user.ID, jti, input.IPAddress, input.UserAgent, input.DeviceName, expiresAt)
		if err == nil {
			sessionID = session.ID
		}
	}

	// Update last login
	_ = s.userRepo.UpdateLastLogin(ctx, user.ID)

	// Log successful login
	_ = s.auditRepo.LogLogin(ctx, user.ID, user.Email, input.IPAddress, input.UserAgent, true)

	return &LoginOutput{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		MFARequired:  false,
		SessionID:    sessionID,
	}, nil
}

// Logout invalidates all user sessions
func (s *AuthService) Logout(ctx context.Context, userID, ipAddress, userAgent string) error {
	// Revoke all refresh tokens
	if err := s.tokenRepo.RevokeAllUserTokens(ctx, userID); err != nil {
		return err
	}

	// Revoke all sessions
	if s.sessionRepo != nil {
		_ = s.sessionRepo.RevokeAllUserSessions(ctx, userID, "")
	}

	// Log logout
	_ = s.auditRepo.LogLogout(ctx, userID, ipAddress, userAgent)

	return nil
}

// RefreshTokenInput holds refresh token input
type RefreshTokenInput struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
	DeviceName   string
}

// RefreshTokenOutput holds refresh token output
type RefreshTokenOutput struct {
	AccessToken  string
	RefreshToken string
}

// RefreshToken exchanges a refresh token for new tokens
func (s *AuthService) RefreshToken(ctx context.Context, input RefreshTokenInput) (*RefreshTokenOutput, error) {
	// Validate refresh token
	token, err := s.tokenRepo.GetByToken(ctx, input.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid or expired refresh token")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Revoke old refresh token (rotation)
	if err := s.tokenRepo.Revoke(ctx, token.ID); err != nil {
		return nil, err
	}

	// Generate new access token
	accessToken, jti, err := s.jwt.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, err
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	// Store new refresh token
	expiresAt := time.Now().Add(s.jwt.GetRefreshTokenExpiry())
	_, err = s.tokenRepo.Create(ctx, user.ID, newRefreshToken, expiresAt)
	if err != nil {
		return nil, err
	}

	// Create new session for refreshed token
	if s.sessionRepo != nil {
		_, _ = s.sessionRepo.Create(ctx, user.ID, jti, input.IPAddress, input.UserAgent, input.DeviceName, expiresAt)
	}

	return &RefreshTokenOutput{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// ValidateAccessToken validates an access token and returns claims
func (s *AuthService) ValidateAccessToken(ctx context.Context, tokenString string) (*authpkg.Claims, error) {
	return s.jwt.ValidateAccessToken(tokenString)
}

// ValidateAPIKey validates an API key and returns the user
func (s *AuthService) ValidateAPIKey(ctx context.Context, key string) (*auth.User, error) {
	apiKey, err := s.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	// Update last used
	_ = s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID)

	// Get user
	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// SetupMFAInput holds MFA setup input
type SetupMFAInput struct {
	UserID    string
	IPAddress string
	UserAgent string
}

// SetupMFAOutput holds MFA setup output
type SetupMFAOutput struct {
	Secret      *authpkg.MFASecret
	BackupCodes []string
}

// SetupMFA generates MFA secret and backup codes for a user
func (s *AuthService) SetupMFA(ctx context.Context, input SetupMFAInput) (*SetupMFAOutput, error) {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Generate MFA secret
	secret, err := s.mfa.GenerateSecret(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	// Generate backup codes
	backupCodes, err := s.mfa.GenerateBackupCodes()
	if err != nil {
		return nil, err
	}

	// Hash backup codes for storage
	hashedCodes, err := s.mfa.HashBackupCodes(backupCodes)
	if err != nil {
		return nil, err
	}

	// Store MFA secret and backup codes (but don't enable MFA yet)
	if err := s.userRepo.SetMFAEnabled(ctx, user.ID, false, &secret.Secret, &hashedCodes); err != nil {
		return nil, err
	}

	return &SetupMFAOutput{
		Secret:      secret,
		BackupCodes: backupCodes,
	}, nil
}

// EnableMFAInput holds MFA enable input
type EnableMFAInput struct {
	UserID    string
	MFACode   string
	IPAddress string
	UserAgent string
}

// EnableMFA enables MFA for a user after verifying a code
func (s *AuthService) EnableMFA(ctx context.Context, input EnableMFAInput) error {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.MFASecret == nil {
		return errors.New("MFA not set up")
	}

	// Verify the code
	if err := s.mfa.VerifyCode(*user.MFASecret, input.MFACode); err != nil {
		return errors.New("invalid MFA code")
	}

	// Enable MFA
	if err := s.userRepo.SetMFAEnabled(ctx, user.ID, true, user.MFASecret, user.MFABackupCodes); err != nil {
		return err
	}

	return nil
}

// DisableMFA disables MFA for a user
func (s *AuthService) DisableMFA(ctx context.Context, userID, mfaCode, ipAddress, userAgent string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return errors.New("user not found")
	}

	if !user.MFAEnabled {
		return nil // Already disabled
	}

	if user.MFASecret == nil {
		return errors.New("MFA not configured")
	}

	// Verify the code (or use backup code)
	if err := s.mfa.VerifyCode(*user.MFASecret, mfaCode); err != nil {
		// Try backup codes
		if user.MFABackupCodes != nil {
			validBackup, _ := s.mfa.VerifyBackupCode(*user.MFABackupCodes, mfaCode)
			if validBackup {
				// Remove used backup code
				newCodes, _ := s.mfa.RemoveBackupCode(*user.MFABackupCodes, mfaCode)
				user.MFABackupCodes = &newCodes
			} else {
				return errors.New("invalid MFA or backup code")
			}
		} else {
			return errors.New("invalid MFA code")
		}
	}

	// Disable MFA
	var emptySecret string
	if err := s.userRepo.SetMFAEnabled(ctx, user.ID, false, &emptySecret, user.MFABackupCodes); err != nil {
		return err
	}

	return nil
}

// CreateAPIKeyInput holds API key creation input
type CreateAPIKeyInput struct {
	UserID      string
	Name        string
	Permissions []string
	ExpiresAt   *time.Time
	IPAddress   string
	UserAgent   string
}

// CreateAPIKeyOutput holds API key creation output
type CreateAPIKeyOutput struct {
	APIKey      *auth.APIKey
	RawKey      string // Shown only once!
}

// CreateAPIKey creates a new API key
func (s *AuthService) CreateAPIKey(ctx context.Context, input CreateAPIKeyInput) (*CreateAPIKeyOutput, error) {
	// Generate API key
	rawKey, prefix, err := authpkg.GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	// Store API key
	apiKey, err := s.apiKeyRepo.Create(ctx, input.UserID, input.Name, rawKey, prefix, input.Permissions, input.ExpiresAt)
	if err != nil {
		return nil, err
	}

	// Log API key creation
	_ = s.auditRepo.LogAPIKeyCreate(ctx, input.UserID, apiKey.ID, input.Name, input.IPAddress, input.UserAgent)

	return &CreateAPIKeyOutput{
		APIKey: apiKey,
		RawKey: rawKey,
	}, nil
}

// RevokeAPIKey revokes an API key
func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID string) error {
	return s.apiKeyRepo.Revoke(ctx, keyID)
}

// ListAPIKeys lists all API keys for a user
func (s *AuthService) ListAPIKeys(ctx context.Context, userID string) ([]*auth.APIKey, error) {
	return s.apiKeyRepo.ListByUser(ctx, userID)
}

// GetCurrentUser gets the current user by ID
func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (*auth.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// UpdateCurrentUserInput holds the fields a user may update on their own account.
type UpdateCurrentUserInput struct {
	UserID          string
	Email           string // empty = no change
	CurrentPassword string // required when NewPassword is non-empty
	NewPassword     string // empty = no change
	IPAddress       string
	UserAgent       string
}

// UpdateCurrentUser updates the calling user's email and/or password.
// A password change requires the current password for verification.
func (s *AuthService) UpdateCurrentUser(ctx context.Context, input UpdateCurrentUserInput) (*auth.User, error) {
	user, err := s.userRepo.GetByID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	// Password change
	if input.NewPassword != "" {
		if input.CurrentPassword == "" {
			return nil, errors.New("current password is required to change password")
		}
		if err := s.password.Verify(input.CurrentPassword, user.PasswordHash); err != nil {
			return nil, errors.New("current password is incorrect")
		}
		if err := authpkg.ValidatePasswordStrength(input.NewPassword); err != nil {
			return nil, err
		}
		newHash, err := s.password.Hash(input.NewPassword)
		if err != nil {
			return nil, err
		}
		if err := s.userRepo.UpdatePassword(ctx, input.UserID, newHash); err != nil {
			return nil, err
		}
		// Audit password change
		if s.auditRepo != nil {
			_ = s.auditRepo.LogResourceAction(ctx, input.UserID, "password_change", "user", input.UserID, input.IPAddress, input.UserAgent, nil, true)
		}
	}

	// Email change
	if input.Email != "" && input.Email != user.Email {
		if err := s.userRepo.UpdateEmail(ctx, input.UserID, input.Email); err != nil {
			return nil, err
		}
		// Audit email change
		if s.auditRepo != nil {
			_ = s.auditRepo.LogResourceAction(ctx, input.UserID, "email_change", "user", input.UserID, input.IPAddress, input.UserAgent, map[string]any{"new_email": input.Email}, true)
		}
	}

	return s.userRepo.GetByID(ctx, input.UserID)
}

// ListAuditLogsInput holds filters for querying audit logs.
type ListAuditLogsInput struct {
	UserID string
	Action string
	Limit  int
	Offset int
}

// ListAuditLogsOutput holds the result of a ListAuditLogs call.
type ListAuditLogsOutput struct {
	Logs  []*auth.AuditLog
	Total int64
}

// ListAuditLogs returns audit log entries with optional filtering.
func (s *AuthService) ListAuditLogs(ctx context.Context, input ListAuditLogsInput) (*ListAuditLogsOutput, error) {
	if s.auditRepo == nil {
		return &ListAuditLogsOutput{}, nil
	}

	limit := input.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	var (
		logs  []*auth.AuditLog
		total int64
		err   error
	)

	switch {
	case input.UserID != "":
		logs, err = s.auditRepo.ListByUser(ctx, input.UserID, limit, input.Offset)
	case input.Action != "":
		logs, err = s.auditRepo.ListByAction(ctx, input.Action, limit, input.Offset)
	default:
		logs, err = s.auditRepo.List(ctx, limit, input.Offset)
	}
	if err != nil {
		return nil, err
	}

	total, err = s.auditRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	return &ListAuditLogsOutput{Logs: logs, Total: total}, nil
}

// HasFirstAdmin checks if there's already an admin user
func (s *AuthService) HasFirstAdmin(ctx context.Context) (bool, error) {
	_, err := s.userRepo.GetFirstAdmin(ctx)
	if err != nil {
		if errors.Is(err, errors.New("no admin user found")) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListSessions returns all active sessions for a user
func (s *AuthService) ListSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	if s.sessionRepo == nil {
		return nil, nil
	}
	return s.sessionRepo.ListByUser(ctx, userID)
}

// RevokeSession revokes a specific session
func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	if s.sessionRepo == nil {
		return errors.New("session management not available")
	}

	// Get the session to verify ownership
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return errors.New("session not found")
	}

	// Verify the session belongs to the user
	if session.UserID != userID {
		return errors.New("session not found")
	}

	return s.sessionRepo.Revoke(ctx, sessionID)
}

// RevokeOtherSessions revokes all sessions except the current one
func (s *AuthService) RevokeOtherSessions(ctx context.Context, userID, currentJTI string) error {
	if s.sessionRepo == nil {
		return errors.New("session management not available")
	}
	return s.sessionRepo.RevokeAllUserSessions(ctx, userID, currentJTI)
}

// UpdateSessionLastSeen updates the last_seen_at timestamp for a session
func (s *AuthService) UpdateSessionLastSeen(ctx context.Context, jti string) error {
	if s.sessionRepo == nil {
		return nil
	}
	return s.sessionRepo.UpdateLastSeen(ctx, jti)
}

// IsSessionRevoked checks if a session is revoked
func (s *AuthService) IsSessionRevoked(ctx context.Context, jti string) (bool, error) {
	if s.sessionRepo == nil {
		return false, nil
	}
	return s.sessionRepo.IsRevoked(ctx, jti)
}
