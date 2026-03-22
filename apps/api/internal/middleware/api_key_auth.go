package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	"github.com/doomedramen/lab/apps/api/pkg/response"
)

// APIKeyAuthMiddleware provides API key authentication for HTTP handlers
type APIKeyAuthMiddleware struct {
	apiKeyRepo *auth.APIKeyRepository
	userRepo   *auth.UserRepository
	auditRepo  *auth.AuditLogRepository
}

// NewAPIKeyAuthMiddleware creates a new API key auth middleware
func NewAPIKeyAuthMiddleware(
	apiKeyRepo *auth.APIKeyRepository,
	userRepo *auth.UserRepository,
	auditRepo *auth.AuditLogRepository,
) *APIKeyAuthMiddleware {
	return &APIKeyAuthMiddleware{
		apiKeyRepo: apiKeyRepo,
		userRepo:   userRepo,
		auditRepo:  auditRepo,
	}
}

// RequireAPIKey is a middleware that requires a valid API key
// It looks for the API key in:
// 1. Authorization header: "Authorization: labkey <key>" or "Authorization: Bearer <key>"
// 2. X-API-Key header: "X-API-Key: <key>"
//
// Usage:
//
//	r.Get("/protected", apiKeyAuthMiddleware.RequireAPIKey(handler))
func (m *APIKeyAuthMiddleware) RequireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := m.extractAPIKey(r)
		if apiKey == "" {
			response.Error(w, http.StatusUnauthorized, "missing API key")
			return
		}

		ctx := r.Context()
		user, apiKeyID, err := m.validateAPIKey(ctx, apiKey)
		if err != nil {
			response.Error(w, http.StatusUnauthorized, "invalid API key")
			return
		}

		// Add user and API key info to context
		ctx = context.WithValue(ctx, UserIDKey, user.ID)
		ctx = context.WithValue(ctx, UserEmailKey, user.Email)
		ctx = context.WithValue(ctx, UserRoleKey, string(user.Role))
		ctx = context.WithValue(ctx, UserKey, user)
		ctx = context.WithValue(ctx, APIKeyIDKey, apiKeyID)

		// Call next handler with enriched context
		next(w, r.WithContext(ctx))
	}
}

// extractAPIKey extracts the API key from the request headers
func (m *APIKeyAuthMiddleware) extractAPIKey(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Support "labkey <key>" format
		if strings.HasPrefix(authHeader, "labkey ") {
			return strings.TrimPrefix(authHeader, "labkey ")
		}
		// Support "Bearer <key>" format
		if strings.HasPrefix(authHeader, "Bearer ") {
			return strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	// Check X-API-Key header
	apiKeyHeader := r.Header.Get("X-API-Key")
	if apiKeyHeader != "" {
		return apiKeyHeader
	}

	return ""
}

// validateAPIKey validates an API key and returns the associated user
func (m *APIKeyAuthMiddleware) validateAPIKey(ctx context.Context, key string) (*auth.User, string, error) {
	apiKey, err := m.apiKeyRepo.GetByKey(ctx, key)
	if err != nil {
		return nil, "", err
	}

	// Get user from database
	user, err := m.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, "", err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, "", err
	}

	// Update last used timestamp (ignore errors)
	_ = m.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID)

	// Log API key usage (ignore errors)
	_ = m.auditRepo.LogAPIKeyUse(ctx, user.ID, apiKey.ID, "", "")

	return user, apiKey.ID, nil
}

// RequireAPIKeyFunc is a convenience function that returns a middleware function
// This is useful for passing to route registration functions
func RequireAPIKeyFunc(middleware *APIKeyAuthMiddleware) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.RequireAPIKey
}
