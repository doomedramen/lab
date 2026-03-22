package validator

import (
	"errors"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	authpkg "github.com/doomedramen/lab/apps/api/pkg/auth"
)

func TestDefaultAuthValidator(t *testing.T) {
	v := DefaultAuthValidator()
	if v == nil {
		t.Error("DefaultAuthValidator returned nil")
	}
}

func TestValidateRegisterRequest(t *testing.T) {
	v := DefaultAuthValidator()

	// Test valid request
	t.Run("valid request", func(t *testing.T) {
		errs := v.ValidateRegisterRequest("test@example.com", "SecurePass123!", auth.RoleViewer)
		if len(errs) > 0 {
			t.Errorf("ValidateRegisterRequest() should not return errors for valid input")
		}
	})

	// Test empty email
	t.Run("empty email", func(t *testing.T) {
		errs := v.ValidateRegisterRequest("", "SecurePass123!", auth.RoleViewer)
		if len(errs) == 0 {
			t.Errorf("ValidateRegisterRequest() should return error for empty email")
		}
	})

	// Test empty password
	t.Run("empty password", func(t *testing.T) {
		errs := v.ValidateRegisterRequest("test@example.com", "", auth.RoleViewer)
		if len(errs) == 0 {
			t.Errorf("ValidateRegisterRequest() should return error for empty password")
		}
		if len(errs) > 0 && errs[0].Field != "password" {
			t.Errorf("ValidateRegisterRequest() should return password field error")
		}
	})

	// Test invalid email format
	t.Run("invalid email", func(t *testing.T) {
		errs := v.ValidateRegisterRequest("invalid-email", "SecurePass123!", auth.RoleViewer)
		if len(errs) == 0 {
			t.Errorf("ValidateRegisterRequest() should return error for invalid email")
		}
		if len(errs) > 0 && errs[0].Field != "email" {
			t.Errorf("ValidateRegisterRequest() should return email field error")
		}
	})

	// Test weak password
	t.Run("weak password", func(t *testing.T) {
		errs := v.ValidateRegisterRequest("test@example.com", "weak", auth.RoleViewer)
		if len(errs) == 0 {
			t.Errorf("ValidateRegisterRequest() should return error for weak password")
		}
		if len(errs) > 0 && errs[0].Field != "password" {
			t.Errorf("ValidateRegisterRequest() should return password field error")
		}
	})
}

func TestValidateLoginRequest(t *testing.T) {
	v := DefaultAuthValidator()

	// Test valid request
	t.Run("valid request", func(t *testing.T) {
		errs := v.ValidateLoginRequest("test@example.com", "SecurePass123!", "")
		if len(errs) > 0 {
			t.Errorf("ValidateLoginRequest() should not return errors for valid input")
		}
	})

	// Test empty email
	t.Run("empty email", func(t *testing.T) {
		errs := v.ValidateLoginRequest("", "SecurePass123!", "")
		if len(errs) == 0 {
			t.Errorf("ValidateLoginRequest() should return error for empty email")
		}
		if len(errs) > 0 && errs[0].Field != "email" {
			t.Errorf("ValidateLoginRequest() should return email field error")
		}
	})

	// Test empty password
	t.Run("empty password", func(t *testing.T) {
		errs := v.ValidateLoginRequest("test@example.com", "", "")
		if len(errs) == 0 {
			t.Errorf("ValidateLoginRequest() should return error for empty password")
		}
		if len(errs) > 0 && errs[0].Field != "password" {
			t.Errorf("ValidateLoginRequest() should return password field error")
		}
	})

	// Test invalid MFA code format (non-digits)
	t.Run("invalid MFA code format", func(t *testing.T) {
		errs := v.ValidateLoginRequest("test@example.com", "SecurePass123!", "abc123")
		if len(errs) == 0 {
			t.Errorf("ValidateLoginRequest() should return error for non-digit MFA code")
		}
		if len(errs) > 0 && errs[0].Field != "mfa_code" {
			t.Errorf("ValidateLoginRequest() should return mfa_code field error")
		}
	})

	// Test MFA code too short
	t.Run("MFA code too short", func(t *testing.T) {
		errs := v.ValidateLoginRequest("test@example.com", "SecurePass123!", "123")
		if len(errs) == 0 {
			t.Errorf("ValidateLoginRequest() should return error for short MFA code")
		}
		if len(errs) > 0 && errs[0].Field != "mfa_code" {
			t.Errorf("ValidateLoginRequest() should return mfa_code field error")
		}
	})

	// Test MFA code too long
	t.Run("MFA code too long", func(t *testing.T) {
		errs := v.ValidateLoginRequest("test@example.com", "SecurePass123!", "1234567")
		if len(errs) == 0 {
			t.Errorf("ValidateLoginRequest() should return error for long MFA code")
		}
		if len(errs) > 0 && errs[0].Field != "mfa_code" {
			t.Errorf("ValidateLoginRequest() should return mfa_code field error")
		}
	})
}

func TestValidateRefreshTokenRequest(t *testing.T) {
	v := DefaultAuthValidator()

	// Test valid token
	t.Run("valid token", func(t *testing.T) {
		errs := v.ValidateRefreshTokenRequest("lab_abc123456789012345678901234567890")
		if len(errs) > 0 {
			t.Errorf("ValidateRefreshTokenRequest() should not return errors for valid token")
		}
	})

	// Test empty token
	t.Run("empty token", func(t *testing.T) {
		errs := v.ValidateRefreshTokenRequest("")
		if len(errs) == 0 {
			t.Errorf("ValidateRefreshTokenRequest() should return error for empty token")
		}
		if len(errs) > 0 && errs[0].Field != "refresh_token" {
			t.Errorf("ValidateRefreshTokenRequest() should return refresh_token field error")
		}
	})

	// Test invalid format (no prefix)
	t.Run("invalid format - no prefix", func(t *testing.T) {
		errs := v.ValidateRefreshTokenRequest("invalid-token-no-prefix")
		if len(errs) == 0 {
			t.Errorf("ValidateRefreshTokenRequest() should return error for invalid format")
		}
		if len(errs) > 0 && errs[0].Field != "refresh_token" {
			t.Errorf("ValidateRefreshTokenRequest() should return refresh_token field error")
		}
	})
}

func TestValidateAPIKeyRequest(t *testing.T) {
	v := DefaultAuthValidator()

	// Test valid request
	t.Run("valid request", func(t *testing.T) {
		errs := v.ValidateAPIKeyRequest("my-key", []string{"vm:read", "vm:update"}, nil)
		if len(errs) > 0 {
			t.Errorf("ValidateAPIKeyRequest() should not return errors for valid input")
		}
	})

	// Test empty name
	t.Run("empty name", func(t *testing.T) {
		errs := v.ValidateAPIKeyRequest("", []string{"vm:read"}, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateAPIKeyRequest() should return error for empty name")
		}
		if len(errs) > 0 && errs[0].Field != "name" {
			t.Errorf("ValidateAPIKeyRequest() should return name field error")
		}
	})

	// Test name too long
	t.Run("name too long", func(t *testing.T) {
		longName := "this-is-a-very-long-api-key-name-that-clearly-exceeds-the-128-character-limit-which-is-enforced-for-security-reasons-and-this-string-definitely-is-longer-than-128-characters"
		errs := v.ValidateAPIKeyRequest(longName, []string{"vm:read"}, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateAPIKeyRequest() should return error for long name")
		}
		if len(errs) > 0 && errs[0].Field != "name" {
			t.Errorf("ValidateAPIKeyRequest() should return name field error")
		}
	})

	// Test too many permissions
	t.Run("too many permissions", func(t *testing.T) {
		tooMany := make([]string, 51)
		errs := v.ValidateAPIKeyRequest("my-key", tooMany, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateAPIKeyRequest() should return error for too many permissions")
		}
		if len(errs) > 0 && errs[0].Field != "permissions" {
			t.Errorf("ValidateAPIKeyRequest() should return permissions field error")
		}
	})

	// Test invalid permission format
	t.Run("invalid permission format", func(t *testing.T) {
		errs := v.ValidateAPIKeyRequest("my-key", []string{"vm:read", "this-is-invalid!@#"}, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateAPIKeyRequest() should return error for invalid permission")
		}
	})

	// Test duplicate permission
	t.Run("duplicate permission", func(t *testing.T) {
		errs := v.ValidateAPIKeyRequest("my-key", []string{"vm:read", "vm:read"}, nil)
		// Should get one error for duplicate permission
		if len(errs) != 1 {
			t.Errorf("ValidateAPIKeyRequest() should return 1 error for duplicate permission, got %d", len(errs))
		}
	})

	// Test permission too long
	t.Run("permission too long", func(t *testing.T) {
		longPerm := string(make([]byte, 65))
		errs := v.ValidateAPIKeyRequest("my-key", []string{longPerm}, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateAPIKeyRequest() should return error for long permission")
		}
		if len(errs) > 0 && errs[0].Field != "permissions" {
			t.Errorf("ValidateAPIKeyRequest() should return permissions field error")
		}
	})
}

func TestValidateUpdateUserRequest(t *testing.T) {
	v := DefaultAuthValidator()

	// Test valid request
	t.Run("valid request", func(t *testing.T) {
		email := "test@example.com"
		password := "SecurePass123!"
		role := auth.RoleViewer
		errs := v.ValidateUpdateUserRequest(&email, &password, &role, nil)
		if len(errs) > 0 {
			t.Errorf("ValidateUpdateUserRequest() should not return errors for valid input")
		}
	})

	// Test with invalid role
	t.Run("invalid role", func(t *testing.T) {
		invalidRole := auth.Role("invalid")
		errs := v.ValidateUpdateUserRequest(nil, nil, &invalidRole, nil)
		if len(errs) == 0 {
			t.Errorf("ValidateUpdateUserRequest() should return error for invalid role")
		}
		if len(errs) > 0 && errs[0].Field != "role" {
			t.Errorf("ValidateUpdateUserRequest() should return role field error")
		}
	})
}

// TestTypedErrors verifies that the typed auth errors can be compared with errors.Is()
func TestTypedErrors(t *testing.T) {
	// Test that our typed errors can be checked with errors.Is()
	if !errors.Is(authpkg.ErrEmailExists, authpkg.ErrEmailExists) {
		t.Error("errors.Is() should return true for ErrEmailExists")
	}
	if !errors.Is(authpkg.ErrInvalidCredentials, authpkg.ErrInvalidCredentials) {
		t.Error("errors.Is() should return true for ErrInvalidCredentials")
	}
	if !errors.Is(authpkg.ErrInvalidMFACode, authpkg.ErrInvalidMFACode) {
		t.Error("errors.Is() should return true for ErrInvalidMFACode")
	}
	if !errors.Is(authpkg.ErrMFANotConfigured, authpkg.ErrMFANotConfigured) {
		t.Error("errors.Is() should return true for ErrMFANotConfigured")
	}
	if !errors.Is(authpkg.ErrCurrentPasswordIncorrect, authpkg.ErrCurrentPasswordIncorrect) {
		t.Error("errors.Is() should return true for ErrCurrentPasswordIncorrect")
	}
	if !errors.Is(authpkg.ErrCurrentPasswordRequired, authpkg.ErrCurrentPasswordRequired) {
		t.Error("errors.Is() should return true for ErrCurrentPasswordRequired")
	}
	if !errors.Is(authpkg.ErrTokenInvalid, authpkg.ErrTokenInvalid) {
		t.Error("errors.Is() should return true for ErrTokenInvalid")
	}
	if !errors.Is(authpkg.ErrUserNotFound, authpkg.ErrUserNotFound) {
		t.Error("errors.Is() should return true for ErrUserNotFound")
	}
}
