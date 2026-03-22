// Package auth provides authentication utilities including password hashing,
// JWT token generation, and MFA support.
package auth

import (
	"errors"
)

// Authentication and authorization errors.
// These can be checked using errors.Is() for proper error handling.
var (
	// ErrEmailExists is returned when attempting to register with an email
	// that is already registered.
	ErrEmailExists = errors.New("email already exists")

	// ErrInvalidCredentials is returned when login fails due to incorrect
	// email or password.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidMFACode is returned when the provided MFA TOTP code is invalid.
	ErrInvalidMFACode = errors.New("invalid MFA code")

	// ErrInvalidMFAOrBackupCode is returned when neither MFA code nor backup code is valid.
	ErrInvalidMFAOrBackupCode = errors.New("invalid MFA or backup code")

	// ErrMFANotConfigured is returned when MFA operations are attempted
	// but MFA has not been set up for the user.
	ErrMFANotConfigured = errors.New("MFA not configured")

	// ErrMFANotSetUp is returned when attempting to enable MFA without setting it up first.
	ErrMFANotSetUp = errors.New("MFA not set up")

	// ErrMFAAlreadyEnabled is returned when attempting to enable MFA
	// when it's already enabled.
	ErrMFAAlreadyEnabled = errors.New("MFA already enabled")

	// ErrSessionNotFound is returned when attempting to revoke or access
	// a session that does not exist or does not belong to the user.
	ErrSessionNotFound = errors.New("session not found")

	// ErrUserNotFound is returned when a user lookup fails.
	ErrUserNotFound = errors.New("user not found")

	// ErrCurrentPasswordRequired is returned when attempting to change
	// password without providing the current password.
	ErrCurrentPasswordRequired = errors.New("current password is required to change password")

	// ErrCurrentPasswordIncorrect is returned when the provided current
	// password does not match the stored hash.
	ErrCurrentPasswordIncorrect = errors.New("current password is incorrect")

	// ErrTokenExpired is returned when attempting to use an expired token.
	ErrTokenExpired = errors.New("token expired")

	// ErrTokenInvalid is returned when a token fails validation.
	ErrTokenInvalid = errors.New("invalid or expired refresh token")

	// ErrAPIKeyNotFound is returned when attempting to revoke or access
	// an API key that does not exist.
	ErrAPIKeyNotFound = errors.New("API key not found")

	// ErrAPIKeyExpired is returned when attempting to use an expired API key.
	ErrAPIKeyExpired = errors.New("API key expired")

	// ErrAPIKeyRevoked is returned when attempting to use a revoked API key.
	ErrAPIKeyRevoked = errors.New("API key revoked")

	// ErrInvalidTokenFormat is returned when a token has an invalid format.
	ErrInvalidTokenFormat = errors.New("invalid token format")

	// ErrNoAdminFound is returned when looking up the first admin user fails.
	ErrNoAdminFound = errors.New("no admin user found")

	// ErrSessionManagementUnavailable is returned when session operations
	// are attempted but session management is not available.
	ErrSessionManagementUnavailable = errors.New("session management not available")
)
