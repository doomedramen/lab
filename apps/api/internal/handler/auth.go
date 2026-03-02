package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/middleware"
	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// AuthServiceServer implements the AuthService Connect RPC server
type AuthServiceServer struct {
	authService *service.AuthService
}

// NewAuthServiceServer creates a new auth service server
func NewAuthServiceServer(authService *service.AuthService) *AuthServiceServer {
	return &AuthServiceServer{authService: authService}
}

// compile-time check that we implement the interface
var _ labv1connect.AuthServiceHandler = (*AuthServiceServer)(nil)

// Register registers a new user
func (s *AuthServiceServer) Register(ctx context.Context, req *connect.Request[labv1.RegisterRequest]) (*connect.Response[labv1.RegisterResponse], error) {
	msg := req.Msg

	// Validate input
	if msg.Email == "" || msg.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and password are required"))
	}

	// Determine role - first user is admin, others are viewer
	var role auth.Role
	hasAdmin, _ := s.authService.HasFirstAdmin(ctx)
	if !hasAdmin {
		role = auth.RoleAdmin
	} else {
		// Default to viewer for non-first users
		role = auth.RoleViewer
		if msg.Role != labv1.UserRole_USER_ROLE_UNSPECIFIED {
			role = protoToRole(msg.Role)
		}
	}

	// Get IP and user agent from headers
	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Register user
	result, err := s.authService.Register(ctx, service.RegisterInput{
		Email:     msg.Email,
		Password:  msg.Password,
		Role:      role,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	if err != nil {
		if err.Error() == "email already exists" {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RegisterResponse{
		User:         userToProto(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}), nil
}

// Login authenticates a user
func (s *AuthServiceServer) Login(ctx context.Context, req *connect.Request[labv1.LoginRequest]) (*connect.Response[labv1.LoginResponse], error) {
	msg := req.Msg

	// Validate input
	if msg.Email == "" || msg.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and password are required"))
	}

	// Get IP and user agent from headers
	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Login
	result, err := s.authService.Login(ctx, service.LoginInput{
		Email:     msg.Email,
		Password:  msg.Password,
		MFACode:   msg.MfaCode,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	if err != nil {
		if err.Error() == "invalid credentials" {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		if err.Error() == "invalid MFA code" {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// If MFA is required, return partial response
	if result.MFARequired {
		return connect.NewResponse(&labv1.LoginResponse{
			User:        userToProto(result.User),
			MfaRequired: true,
		}), nil
	}

	return connect.NewResponse(&labv1.LoginResponse{
		User:         userToProto(result.User),
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}), nil
}

// Logout logs out the current user
func (s *AuthServiceServer) Logout(ctx context.Context, req *connect.Request[labv1.LogoutRequest]) (*connect.Response[labv1.LogoutResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Logout
	if err := s.authService.Logout(ctx, user.ID, ipAddress, userAgent); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.LogoutResponse{
		Success: true,
	}), nil
}

// RefreshToken refreshes an access token
func (s *AuthServiceServer) RefreshToken(ctx context.Context, req *connect.Request[labv1.RefreshTokenRequest]) (*connect.Response[labv1.RefreshTokenResponse], error) {
	msg := req.Msg

	if msg.RefreshToken == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("refresh token is required"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Refresh token
	result, err := s.authService.RefreshToken(ctx, service.RefreshTokenInput{
		RefreshToken: msg.RefreshToken,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	return connect.NewResponse(&labv1.RefreshTokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
	}), nil
}

// SetupMFA sets up MFA for the current user
func (s *AuthServiceServer) SetupMFA(ctx context.Context, req *connect.Request[labv1.SetupMFARequest]) (*connect.Response[labv1.SetupMFAResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Setup MFA
	result, err := s.authService.SetupMFA(ctx, service.SetupMFAInput{
		UserID:    user.ID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.SetupMFAResponse{
		Secret:      result.Secret.Secret,
		QrCodeUrl:   result.Secret.QRCodeURL,
		ManualKey:   result.Secret.ManualKey,
		BackupCodes: result.BackupCodes,
	}), nil
}

// EnableMFA enables MFA for the current user
func (s *AuthServiceServer) EnableMFA(ctx context.Context, req *connect.Request[labv1.EnableMFARequest]) (*connect.Response[labv1.EnableMFAResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.MfaCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("MFA code is required"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Enable MFA
	if err := s.authService.EnableMFA(ctx, service.EnableMFAInput{
		UserID:    user.ID,
		MFACode:   req.Msg.MfaCode,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&labv1.EnableMFAResponse{
		Success: true,
	}), nil
}

// DisableMFA disables MFA for the current user
func (s *AuthServiceServer) DisableMFA(ctx context.Context, req *connect.Request[labv1.DisableMFARequest]) (*connect.Response[labv1.DisableMFAResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.MfaCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("MFA code is required"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Disable MFA
	if err := s.authService.DisableMFA(ctx, user.ID, req.Msg.MfaCode, ipAddress, userAgent); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&labv1.DisableMFAResponse{
		Success: true,
	}), nil
}

// VerifyMFACode verifies a TOTP code
func (s *AuthServiceServer) VerifyMFACode(ctx context.Context, req *connect.Request[labv1.VerifyMFACodeRequest]) (*connect.Response[labv1.VerifyMFACodeResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.MfaCode == "" {
		return connect.NewResponse(&labv1.VerifyMFACodeResponse{
			Valid: false,
		}), nil
	}

	// Get user's MFA secret
	if user.MFASecret == nil {
		return connect.NewResponse(&labv1.VerifyMFACodeResponse{
			Valid: false,
		}), nil
	}

	// Verify code (we need to access the MFA service - for now, return false)
	// This would need the MFA handler to be accessible
	return connect.NewResponse(&labv1.VerifyMFACodeResponse{
		Valid: false,
	}), nil
}

// CreateAPIKey creates a new API key
func (s *AuthServiceServer) CreateAPIKey(ctx context.Context, req *connect.Request[labv1.CreateAPIKeyRequest]) (*connect.Response[labv1.CreateAPIKeyResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	// Parse expiration time
	var expiresAt *time.Time
	if req.Msg.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, req.Msg.ExpiresAt)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid expires_at format, use ISO 8601"))
		}
		expiresAt = &t
	}

	// Create API key
	result, err := s.authService.CreateAPIKey(ctx, service.CreateAPIKeyInput{
		UserID:      user.ID,
		Name:        req.Msg.Name,
		Permissions: req.Msg.Permissions,
		ExpiresAt:   expiresAt,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	})

	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateAPIKeyResponse{
		ApiKey: apiKeyToProto(result.APIKey),
		RawKey: result.RawKey,
	}), nil
}

// ListAPIKeys lists all API keys for the current user
func (s *AuthServiceServer) ListAPIKeys(ctx context.Context, req *connect.Request[labv1.ListAPIKeysRequest]) (*connect.Response[labv1.ListAPIKeysResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	// List API keys
	keys, err := s.authService.ListAPIKeys(ctx, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var protoKeys []*labv1.APIKey
	for _, key := range keys {
		protoKeys = append(protoKeys, apiKeyToProto(key))
	}

	return connect.NewResponse(&labv1.ListAPIKeysResponse{
		ApiKeys: protoKeys,
	}), nil
}

// RevokeAPIKey revokes an API key
func (s *AuthServiceServer) RevokeAPIKey(ctx context.Context, req *connect.Request[labv1.RevokeAPIKeyRequest]) (*connect.Response[labv1.RevokeAPIKeyResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	// Revoke API key
	if err := s.authService.RevokeAPIKey(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RevokeAPIKeyResponse{
		Success: true,
	}), nil
}

// GetCurrentUser gets the current user
func (s *AuthServiceServer) GetCurrentUser(ctx context.Context, req *connect.Request[labv1.GetCurrentUserRequest]) (*connect.Response[labv1.GetCurrentUserResponse], error) {
	// Get user from context
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	return connect.NewResponse(&labv1.GetCurrentUserResponse{
		User: userToProto(user),
	}), nil
}

// UpdateCurrentUser updates the current user's email and/or password.
func (s *AuthServiceServer) UpdateCurrentUser(ctx context.Context, req *connect.Request[labv1.UpdateCurrentUserRequest]) (*connect.Response[labv1.UpdateCurrentUserResponse], error) {
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.Email == "" && req.Msg.NewPassword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one field (email or new_password) must be provided"))
	}

	ipAddress := extractIPAddress(req.Header())
	userAgent := req.Header().Get("User-Agent")

	updated, err := s.authService.UpdateCurrentUser(ctx, service.UpdateCurrentUserInput{
		UserID:          user.ID,
		Email:           req.Msg.Email,
		CurrentPassword: req.Msg.CurrentPassword,
		NewPassword:     req.Msg.NewPassword,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
	})
	if err != nil {
		switch {
		case err.Error() == "current password is incorrect":
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		case err.Error() == "current password is required to change password":
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case err.Error() == "email already exists":
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&labv1.UpdateCurrentUserResponse{
		User: userToProto(updated),
	}), nil
}

// Helper functions

// userToProto converts a domain User to protobuf User
func userToProto(u *auth.User) *labv1.User {
	pb := &labv1.User{
		Id:        u.ID,
		Email:     u.Email,
		Role:      roleToProto(u.Role),
		MfaEnabled: u.MFAEnabled,
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
		UpdatedAt: u.UpdatedAt.Format(time.RFC3339),
	}

	if u.LastLoginAt != nil {
		pb.LastLoginAt = u.LastLoginAt.Format(time.RFC3339)
	}

	return pb
}

// roleToProto converts a domain Role to protobuf UserRole
func roleToProto(role auth.Role) labv1.UserRole {
	switch role {
	case auth.RoleAdmin:
		return labv1.UserRole_USER_ROLE_ADMIN
	case auth.RoleOperator:
		return labv1.UserRole_USER_ROLE_OPERATOR
	case auth.RoleViewer:
		return labv1.UserRole_USER_ROLE_VIEWER
	default:
		return labv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

// protoToRole converts a protobuf UserRole to domain Role
func protoToRole(role labv1.UserRole) auth.Role {
	switch role {
	case labv1.UserRole_USER_ROLE_ADMIN:
		return auth.RoleAdmin
	case labv1.UserRole_USER_ROLE_OPERATOR:
		return auth.RoleOperator
	case labv1.UserRole_USER_ROLE_VIEWER:
		return auth.RoleViewer
	default:
		return auth.RoleViewer
	}
}

// apiKeyToProto converts a domain APIKey to protobuf APIKey
func apiKeyToProto(k *auth.APIKey) *labv1.APIKey {
	pb := &labv1.APIKey{
		Id:          k.ID,
		Name:        k.Name,
		Prefix:      k.Prefix,
		Permissions: k.Permissions,
		CreatedAt:   k.CreatedAt.Format(time.RFC3339),
	}

	if k.LastUsedAt != nil {
		pb.LastUsedAt = k.LastUsedAt.Format(time.RFC3339)
	}

	if k.ExpiresAt != nil {
		pb.ExpiresAt = k.ExpiresAt.Format(time.RFC3339)
	}

	return pb
}

// ListAuditLogs returns audit log entries. Admin only.
func (s *AuthServiceServer) ListAuditLogs(ctx context.Context, req *connect.Request[labv1.ListAuditLogsRequest]) (*connect.Response[labv1.ListAuditLogsResponse], error) {
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}
	if user.Role != auth.RoleAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("admin role required"))
	}

	out, err := s.authService.ListAuditLogs(ctx, service.ListAuditLogsInput{
		UserID: req.Msg.UserId,
		Action: req.Msg.Action,
		Limit:  int(req.Msg.Limit),
		Offset: int(req.Msg.Offset),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLogs := make([]*labv1.AuditLogEntry, 0, len(out.Logs))
	for _, l := range out.Logs {
		protoLogs = append(protoLogs, auditLogToProto(l))
	}

	return connect.NewResponse(&labv1.ListAuditLogsResponse{
		Logs:  protoLogs,
		Total: out.Total,
	}), nil
}

// auditLogToProto converts an AuditLog domain model to the protobuf representation.
func auditLogToProto(l *auth.AuditLog) *labv1.AuditLogEntry {
	entry := &labv1.AuditLogEntry{
		Id:           fmt.Sprintf("%d", l.ID),
		UserId:       l.UserID,
		Action:       l.Action,
		ResourceType: l.ResourceType,
		ResourceId:   l.ResourceID,
		IpAddress:    l.IPAddress,
		UserAgent:    l.UserAgent,
		Status:       string(l.Status),
		CreatedAt:    l.CreatedAt.Format(time.RFC3339),
	}
	if l.Details != nil {
		if b, err := json.Marshal(l.Details); err == nil {
			entry.Details = string(b)
		}
	}
	return entry
}

// extractIPAddress extracts the client IP address from headers
func extractIPAddress(headers map[string][]string) string {
	// Helper to get first value from header
	getHeader := func(name string) string {
		values := headers[name]
		if len(values) > 0 {
			return values[0]
		}
		return ""
	}

	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := getHeader("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := getHeader("X-Real-IP"); xri != "" {
		return xri
	}

	return ""
}

// ListSessions lists all active sessions for the current user
func (s *AuthServiceServer) ListSessions(ctx context.Context, req *connect.Request[labv1.ListSessionsRequest]) (*connect.Response[labv1.ListSessionsResponse], error) {
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	// Get current session JTI
	currentJTI := middleware.GetSessionJTIFromContext(ctx)

	sessions, err := s.authService.ListSessions(ctx, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var protoSessions []*labv1.Session
	for _, session := range sessions {
		protoSessions = append(protoSessions, &labv1.Session{
			Id:          session.ID,
			UserId:       session.UserID,
			IpAddress:    session.IPAddress,
			UserAgent:    session.UserAgent,
			DeviceName:   session.DeviceName,
			IssuedAt:     session.IssuedAt.Format(time.RFC3339),
			LastSeenAt:   session.LastSeenAt.Format(time.RFC3339),
			ExpiresAt:    session.ExpiresAt.Format(time.RFC3339),
			IsCurrent:    session.JTI == currentJTI,
		})
	}

	return connect.NewResponse(&labv1.ListSessionsResponse{
		Sessions: protoSessions,
	}), nil
}

// RevokeSession revokes a specific session
func (s *AuthServiceServer) RevokeSession(ctx context.Context, req *connect.Request[labv1.RevokeSessionRequest]) (*connect.Response[labv1.RevokeSessionResponse], error) {
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	if req.Msg.SessionId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("session_id is required"))
	}

	if err := s.authService.RevokeSession(ctx, user.ID, req.Msg.SessionId); err != nil {
		if err.Error() == "session not found" {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RevokeSessionResponse{
		Success: true,
	}), nil
}

// RevokeOtherSessions revokes all sessions except the current one
func (s *AuthServiceServer) RevokeOtherSessions(ctx context.Context, req *connect.Request[labv1.RevokeOtherSessionsRequest]) (*connect.Response[labv1.RevokeOtherSessionsResponse], error) {
	user := middleware.GetUserFromContext(ctx)
	if user == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("user not authenticated"))
	}

	currentJTI := middleware.GetSessionJTIFromContext(ctx)
	if currentJTI == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("no current session found"))
	}

	if err := s.authService.RevokeOtherSessions(ctx, user.ID, currentJTI); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RevokeOtherSessionsResponse{
		RevokedCount: 0, // We don't track the count for now
	}), nil
}
