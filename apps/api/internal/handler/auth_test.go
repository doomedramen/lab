package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
)

func TestHealthCheck(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	HealthCheck(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestRoleToProto(t *testing.T) {
	tests := []struct {
		role auth.Role
		name string
	}{
		{auth.RoleAdmin, "admin"},
		{auth.RoleOperator, "operator"},
		{auth.RoleViewer, "viewer"},
		{auth.Role("unknown"), "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto := roleToProto(tt.role)
			// Just verify it doesn't panic and returns a valid enum
			if proto < 0 {
				t.Errorf("unexpected negative proto value: %d", proto)
			}
		})
	}
}

func TestProtoToRole(t *testing.T) {
	// Round-trip: role -> proto -> role
	for _, role := range []auth.Role{auth.RoleAdmin, auth.RoleOperator, auth.RoleViewer} {
		proto := roleToProto(role)
		back := protoToRole(proto)
		if back != role {
			t.Errorf("roundtrip %q: got %q", role, back)
		}
	}
}

func TestUserToProto(t *testing.T) {
	now := time.Now()
	lastLogin := now.Add(-1 * time.Hour)

	user := &auth.User{
		ID:          "user-123",
		Email:       "test@example.com",
		Role:        auth.RoleAdmin,
		MFAEnabled:  true,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: &lastLogin,
	}

	proto := userToProto(user)

	if proto.Id != "user-123" {
		t.Errorf("Id = %q, want user-123", proto.Id)
	}
	if proto.Email != "test@example.com" {
		t.Errorf("Email = %q, want test@example.com", proto.Email)
	}
	if !proto.MfaEnabled {
		t.Error("expected MfaEnabled=true")
	}
	if !proto.IsActive {
		t.Error("expected IsActive=true")
	}
	if proto.LastLoginAt == "" {
		t.Error("expected LastLoginAt to be set")
	}
}

func TestUserToProto_NilLastLogin(t *testing.T) {
	user := &auth.User{
		ID:        "user-123",
		Email:     "test@example.com",
		Role:      auth.RoleViewer,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	proto := userToProto(user)

	if proto.LastLoginAt != "" {
		t.Errorf("LastLoginAt = %q, want empty", proto.LastLoginAt)
	}
}

func TestApiKeyToProto(t *testing.T) {
	now := time.Now()
	lastUsed := now.Add(-30 * time.Minute)
	expires := now.Add(24 * time.Hour)

	key := &auth.APIKey{
		ID:          "key-123",
		Name:        "my-key",
		Prefix:      "lab_abc",
		Permissions: []string{"read", "write"},
		CreatedAt:   now,
		LastUsedAt:  &lastUsed,
		ExpiresAt:   &expires,
	}

	proto := apiKeyToProto(key)

	if proto.Id != "key-123" {
		t.Errorf("Id = %q, want key-123", proto.Id)
	}
	if proto.Name != "my-key" {
		t.Errorf("Name = %q, want my-key", proto.Name)
	}
	if proto.Prefix != "lab_abc" {
		t.Errorf("Prefix = %q, want lab_abc", proto.Prefix)
	}
	if proto.LastUsedAt == "" {
		t.Error("expected LastUsedAt to be set")
	}
	if proto.ExpiresAt == "" {
		t.Error("expected ExpiresAt to be set")
	}
}

func TestApiKeyToProto_NilOptionalFields(t *testing.T) {
	key := &auth.APIKey{
		ID:        "key-123",
		Name:      "my-key",
		CreatedAt: time.Now(),
	}

	proto := apiKeyToProto(key)

	if proto.LastUsedAt != "" {
		t.Errorf("LastUsedAt = %q, want empty", proto.LastUsedAt)
	}
	if proto.ExpiresAt != "" {
		t.Errorf("ExpiresAt = %q, want empty", proto.ExpiresAt)
	}
}

func TestExtractIPAddress(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string][]string
		want    string
	}{
		{
			"X-Forwarded-For",
			map[string][]string{"X-Forwarded-For": {"1.2.3.4"}},
			"1.2.3.4",
		},
		{
			"X-Real-IP",
			map[string][]string{"X-Real-IP": {"5.6.7.8"}},
			"5.6.7.8",
		},
		{
			"X-Forwarded-For takes precedence",
			map[string][]string{
				"X-Forwarded-For": {"1.2.3.4"},
				"X-Real-IP":       {"5.6.7.8"},
			},
			"1.2.3.4",
		},
		{
			"no headers",
			map[string][]string{},
			"",
		},
		{
			"empty header value",
			map[string][]string{"X-Forwarded-For": {""}},
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIPAddress(tt.headers)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
