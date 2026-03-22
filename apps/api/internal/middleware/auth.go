package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	authpkg "github.com/doomedramen/lab/apps/api/pkg/auth"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UserEmailKey is the context key for user email
	UserEmailKey contextKey = "user_email"
	// UserRoleKey is the context key for user role
	UserRoleKey contextKey = "user_role"
	// UserKey is the context key for the full user object
	UserKey contextKey = "user"
	// APIKeyIDKey is the context key for API key ID (if using API key auth)
	APIKeyIDKey contextKey = "api_key_id"
	// SessionJTIKey is the context key for session JTI
	SessionJTIKey contextKey = "session_jti"
)

// AuthInterceptorConfig holds configuration for the auth interceptor
type AuthInterceptorConfig struct {
	JWTSecret       []byte
	AccessTokenExp  int64 // in seconds
	RefreshTokenExp int64 // in seconds
	Issuer          string
}

// AuthInterceptor provides authentication middleware for Connect RPC
type AuthInterceptor struct {
	config      AuthInterceptorConfig
	userRepo    *auth.UserRepository
	apiKeyRepo  *auth.APIKeyRepository
	auditRepo   *auth.AuditLogRepository
	sessionRepo *auth.SessionRepository
	publicPaths map[string]bool
}

// NewAuthInterceptor creates a new auth interceptor
func NewAuthInterceptor(
	config AuthInterceptorConfig,
	userRepo *auth.UserRepository,
	apiKeyRepo *auth.APIKeyRepository,
	auditRepo *auth.AuditLogRepository,
	sessionRepo *auth.SessionRepository,
) *AuthInterceptor {
	// Define public paths that don't require authentication
	publicPaths := map[string]bool{
		"/lab.v1.AuthService/Login":    true,
		"/lab.v1.AuthService/Register": true,
		// Health checks are typically public
		"/health": true,
	}

	return &AuthInterceptor{
		config:      config,
		userRepo:    userRepo,
		apiKeyRepo:  apiKeyRepo,
		auditRepo:   auditRepo,
		sessionRepo: sessionRepo,
		publicPaths: publicPaths,
	}
}

// WrapUnary wraps unary Connect RPC calls with authentication
func (a *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// Check if path is public
		if a.isPublicPath(req.Spec().Procedure) {
			return next(ctx, req)
		}

		// Extract and validate token
		user, apiKeyID, jti, err := a.authenticate(ctx, req.Header())
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		// Add user to context
		ctx = context.WithValue(ctx, UserIDKey, user.ID)
		ctx = context.WithValue(ctx, UserEmailKey, user.Email)
		ctx = context.WithValue(ctx, UserRoleKey, string(user.Role))
		ctx = context.WithValue(ctx, UserKey, user)

		// Add API key ID to context if using API key auth
		if apiKeyID != "" {
			ctx = context.WithValue(ctx, APIKeyIDKey, apiKeyID)
		}

		// Add session JTI to context
		if jti != "" {
			ctx = context.WithValue(ctx, SessionJTIKey, jti)
		}

		// Call the handler with authenticated context
		return next(ctx, req)
	}
}

// WrapStreamingClient wraps streaming client-side Connect RPC calls
func (a *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler wraps streaming handler-side Connect RPC calls
func (a *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// isPublicPath checks if a path is publicly accessible
func (a *AuthInterceptor) isPublicPath(path string) bool {
	return a.publicPaths[path]
}

// authenticate extracts and validates the authentication token from headers
func (a *AuthInterceptor) authenticate(ctx context.Context, header http.Header) (*auth.User, string, string, error) {
	authHeader := header.Get("Authorization")
	if authHeader == "" {
		return nil, "", "", errors.New("missing authorization header")
	}

	// Check if it's a Bearer token (JWT)
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		return a.validateJWT(ctx, token)
	}

	// Check if it's an API key
	if strings.HasPrefix(authHeader, "labkey_") {
		user, apiKeyID, err := a.validateAPIKey(ctx, authHeader)
		return user, apiKeyID, "", err
	}

	return nil, "", "", errors.New("invalid authorization header format")
}

// validateJWT validates a JWT access token
func (a *AuthInterceptor) validateJWT(ctx context.Context, tokenString string) (*auth.User, string, string, error) {
	// Parse and validate JWT
	jwtHandler := authpkg.NewJWT(authpkg.Config{
		SecretKey: a.config.JWTSecret,
		Issuer:    a.config.Issuer,
	})

	claims, err := jwtHandler.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, "", "", errors.New("invalid or expired token")
	}

	// Check if session is revoked
	if a.sessionRepo != nil && claims.ID != "" {
		revoked, err := a.sessionRepo.IsRevoked(ctx, claims.ID)
		if err != nil {
			// Session not found - since this is new code not in production yet,
			// we can be strict and require sessions
			return nil, "", "", errors.New("session not found")
		}
		if revoked {
			return nil, "", "", errors.New("session has been revoked")
		}

		// Update last seen timestamp (async, ignore errors)
		_ = a.sessionRepo.UpdateLastSeen(ctx, claims.ID)
	}

	// Get user from database
	user, err := a.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, "", "", errors.New("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, "", "", errors.New("user account is disabled")
	}

	return user, "", claims.ID, nil
}

// validateAPIKey validates an API key
func (a *AuthInterceptor) validateAPIKey(ctx context.Context, key string) (*auth.User, string, error) {
	apiKey, err := a.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, "", errors.New("invalid API key")
	}

	// Get user from database
	user, err := a.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, "", errors.New("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, "", errors.New("user account is disabled")
	}

	// Update last used timestamp (ignore errors)
	_ = a.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID)

	// Log API key usage (ignore errors)
	_ = a.auditRepo.LogAPIKeyUse(ctx, user.ID, apiKey.ID, "", "")

	return user, apiKey.ID, nil
}

// GetUserFromContext extracts the user from context
func GetUserFromContext(ctx context.Context) *auth.User {
	user, ok := ctx.Value(UserKey).(*auth.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserIDFromContext extracts the user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// GetSessionJTIFromContext extracts the session JTI from context
func GetSessionJTIFromContext(ctx context.Context) string {
	jti, ok := ctx.Value(SessionJTIKey).(string)
	if !ok {
		return ""
	}
	return jti
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(requiredRole auth.Role) connect.Interceptor {
	return &roleInterceptor{requiredRole: requiredRole}
}

type roleInterceptor struct {
	requiredRole auth.Role
}

func (r *roleInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		user := GetUserFromContext(ctx)
		if user == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
		}

		// Check if user has required role
		if !hasRole(user.Role, r.requiredRole) {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("insufficient permissions"))
		}

		return next(ctx, req)
	}
}

func (r *roleInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (r *roleInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// hasRole checks if a user's role has at least the required permissions
// Role hierarchy: admin > operator > viewer
func hasRole(userRole, requiredRole auth.Role) bool {
	roleHierarchy := map[auth.Role]int{
		auth.RoleViewer:   1,
		auth.RoleOperator: 2,
		auth.RoleAdmin:    3,
	}

	userLevel, ok := roleHierarchy[userRole]
	if !ok {
		return false
	}

	requiredLevel, ok := roleHierarchy[requiredRole]
	if !ok {
		return false
	}

	return userLevel >= requiredLevel
}

// HasPermission creates a middleware that checks for specific permissions
// This is useful for API key-based authorization
func HasPermission(permission string) connect.Interceptor {
	return &permissionInterceptor{permission: permission}
}

type permissionInterceptor struct {
	permission string
}

func (p *permissionInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// For now, we just check if the user is authenticated
		// API key permission checking can be added here
		user := GetUserFromContext(ctx)
		if user == nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
		}

		return next(ctx, req)
	}
}

func (p *permissionInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (p *permissionInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
