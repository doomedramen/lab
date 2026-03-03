package validator

import (
	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
)

// AuthValidator validates authentication-related requests
type AuthValidator struct{}

// DefaultAuthValidator returns a new auth validator
func DefaultAuthValidator() *AuthValidator {
	return &AuthValidator{}
}

// ValidateRegisterRequest validates a user registration request
func (v *AuthValidator) ValidateRegisterRequest(email, password string, role auth.Role) ValidationErrors {
	var errs ValidationErrors
	
	// Validate email
	errs = append(errs, ValidateEmail(email)...)
	
	// Validate password strength
	errs = append(errs, ValidatePassword(password)...)
	
	// Validate role if provided
	if role != "" {
		switch role {
		case auth.RoleAdmin, auth.RoleOperator, auth.RoleViewer:
			// Valid
		default:
			errs = appendError(errs, "role", "must be one of: admin, operator, viewer")
		}
	}
	
	return errs
}

// ValidateLoginRequest validates a login request
func (v *AuthValidator) ValidateLoginRequest(email, password, mfaCode string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate email
	errs = append(errs, ValidateEmail(email)...)
	
	// Validate password (required for login)
	if password == "" {
		errs = appendError(errs, "password", "is required")
	}
	
	// Validate MFA code if provided (must be 6 digits)
	if mfaCode != "" {
		errs = append(errs, ValidateMFACode(mfaCode)...)
	}
	
	return errs
}

// ValidateRefreshTokenRequest validates a refresh token request
func (v *AuthValidator) ValidateRefreshTokenRequest(refreshToken string) ValidationErrors {
	var errs ValidationErrors
	
	if refreshToken == "" {
		errs = appendError(errs, "refresh_token", "is required")
	}
	
	// Basic format check: should start with "lab_"
	if refreshToken != "" && len(refreshToken) > 4 && refreshToken[:4] != "lab_" {
		errs = appendError(errs, "refresh_token", "has invalid format")
	}
	
	return errs
}

// ValidateMFACode validates a TOTP MFA code
func ValidateMFACode(code string) ValidationErrors {
	var errs ValidationErrors
	
	if code == "" {
		return appendError(errs, "mfa_code", "is required")
	}
	
	// Must be exactly 6 digits
	if len(code) != 6 {
		return appendError(errs, "mfa_code", "must be exactly 6 digits")
	}
	
	for _, r := range code {
		if r < '0' || r > '9' {
			return appendError(errs, "mfa_code", "must contain only digits")
		}
	}
	
	return errs
}

// ValidateAPIKeyRequest validates an API key creation request
func (v *AuthValidator) ValidateAPIKeyRequest(name string, permissions []string, expiresAt *string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	if name == "" {
		errs = appendError(errs, "name", "is required")
	}
	
	if len(name) > 128 {
		errs = appendError(errs, "name", "must be 128 characters or less")
	}
	
	// Validate permissions
	if len(permissions) > 50 {
		errs = appendError(errs, "permissions", "cannot have more than 50 permissions")
	}
	
	validPermissions := map[string]bool{
		"vm:read": true, "vm:create": true, "vm:update": true, "vm:delete": true,
		"vm:start": true, "vm:stop": true, "vm:reboot": true, "vm:snapshot": true,
		"container:read": true, "container:create": true, "container:update": true, "container:delete": true,
		"storage:read": true, "storage:create": true, "storage:update": true, "storage:delete": true,
		"network:read": true, "network:create": true, "network:update": true, "network:delete": true,
		"backup:read": true, "backup:create": true, "backup:restore": true, "backup:delete": true,
		"admin:read": true, "admin:write": true,
	}
	
	seen := make(map[string]bool)
	for _, perm := range permissions {
		if !validPermissions[perm] {
			errs = append(errs, &ValidationError{
				Field:   "permissions",
				Message: "invalid permission: " + perm,
			})
		}

		if seen[perm] {
			errs = append(errs, &ValidationError{
				Field:   "permissions",
				Message: "duplicate permission: " + perm,
			})
		}
		seen[perm] = true

		// Check individual permission format
		if len(perm) > 64 {
			errs = append(errs, &ValidationError{
				Field:   "permissions",
				Message: "permission too long",
			})
		}
	}
	
	// Validate expiresAt if provided
	if expiresAt != nil && *expiresAt != "" {
		// Basic RFC3339 format check
		if len(*expiresAt) < 10 {
			errs = appendError(errs, "expires_at", "must be a valid RFC3339 timestamp")
		}
	}
	
	return errs
}

// ValidateUpdateUserRequest validates a user update request
func (v *AuthValidator) ValidateUpdateUserRequest(email *string, password *string, role *auth.Role, isActive *bool) ValidationErrors {
	var errs ValidationErrors
	
	// Validate email if provided
	if email != nil {
		errs = append(errs, ValidateEmail(*email)...)
	}
	
	// Validate password if provided
	if password != nil {
		errs = append(errs, ValidatePassword(*password)...)
	}
	
	// Validate role if provided
	if role != nil {
		switch *role {
		case auth.RoleAdmin, auth.RoleOperator, auth.RoleViewer:
			// Valid
		default:
			errs = appendError(errs, "role", "must be one of: admin, operator, viewer")
		}
	}
	
	// isActive is a boolean, no validation needed beyond type check
	
	return errs
}

// ValidateNotificationChannelRequest validates a notification channel request
func (v *AuthValidator) ValidateNotificationChannelRequest(name, channelType string, config map[string]interface{}) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	errs = append(errs, ValidateNotificationChannelName(name)...)
	
	// Validate type
	if channelType == "" {
		errs = appendError(errs, "type", "is required")
	}
	
	validTypes := map[string]bool{
		"email":   true,
		"webhook": true,
	}
	
	if channelType != "" && !validTypes[channelType] {
		errs = appendError(errs, "type", "must be one of: email, webhook")
	}
	
	// Validate config based on type
	if channelType == "email" {
		errs = append(errs, validateEmailChannelConfig(config)...)
	} else if channelType == "webhook" {
		errs = append(errs, validateWebhookChannelConfig(config)...)
	}
	
	return errs
}

func validateEmailChannelConfig(config map[string]interface{}) ValidationErrors {
	var errs ValidationErrors
	
	// SMTP server is required
	if smtpServer, ok := config["smtp_server"].(string); ok {
		errs = append(errs, ValidateSMTPServer(smtpServer)...)
	} else {
		errs = appendError(errs, "config.smtp_server", "is required")
	}
	
	// SMTP port is required
	if smtpPort, ok := config["smtp_port"].(int); ok {
		if smtpPort < 1 || smtpPort > 65535 {
			errs = appendError(errs, "config.smtp_port", "must be between 1 and 65535")
		}
	} else {
		errs = appendError(errs, "config.smtp_port", "is required")
	}
	
	// From email is required
	if fromEmail, ok := config["from_email"].(string); ok {
		errs = append(errs, ValidateEmailRecipient(fromEmail)...)
	} else {
		errs = appendError(errs, "config.from_email", "is required")
	}
	
	return errs
}

func validateWebhookChannelConfig(config map[string]interface{}) ValidationErrors {
	var errs ValidationErrors
	
	// URL is required
	if url, ok := config["url"].(string); ok {
		errs = append(errs, ValidateWebhookURL(url)...)
	} else {
		errs = appendError(errs, "config.url", "is required")
	}
	
	// Method is optional but must be valid if provided
	if method, ok := config["method"].(string); ok {
		validMethods := map[string]bool{
			"GET": true, "POST": true, "PUT": true, "DELETE": true, "PATCH": true,
		}
		if !validMethods[method] {
			errs = appendError(errs, "config.method", "must be one of: GET, POST, PUT, DELETE, PATCH")
		}
	}
	
	return errs
}

// ValidateAlertRuleRequest validates an alert rule request
func (v *AuthValidator) ValidateAlertRuleRequest(name, ruleType, severity string, threshold float64, channelIDs []string) ValidationErrors {
	var errs ValidationErrors
	
	// Validate name
	errs = append(errs, ValidateAlertRuleName(name)...)
	
	// Validate type
	validTypes := map[string]bool{
		"storage_usage":    true,
		"vm_stopped":       true,
		"backup_failed":    true,
		"node_offline":     true,
		"cpu_usage":        true,
		"memory_usage":     true,
		"uptime_check":     true,
	}
	
	if ruleType == "" {
		errs = appendError(errs, "type", "is required")
	} else if !validTypes[ruleType] {
		errs = appendError(errs, "type", "invalid rule type")
	}
	
	// Validate severity
	validSeverities := map[string]bool{
		"info":     true,
		"warning":  true,
		"critical": true,
	}
	
	if severity == "" {
		errs = appendError(errs, "severity", "is required")
	} else if !validSeverities[severity] {
		errs = appendError(errs, "severity", "must be one of: info, warning, critical")
	}
	
	// Validate threshold (must be positive for percentage-based rules)
	if threshold < 0 {
		errs = appendError(errs, "threshold", "must be non-negative")
	}
	
	// Validate channel IDs
	if len(channelIDs) == 0 {
		errs = appendError(errs, "channel_ids", "at least one notification channel is required")
	}
	
	for i, id := range channelIDs {
		if id == "" {
			errs = append(errs, &ValidationError{
				Field:   "channel_ids",
				Message: "cannot be empty",
			})
		}
		if len(id) > 64 {
			errs = append(errs, &ValidationError{
				Field:   "channel_ids",
				Message: "must be 64 characters or less",
			})
		}
		_ = i // suppress unused variable warning
	}
	
	return errs
}
