// Package validator provides input validation for API requests.
// All validation errors implement the error interface and can be returned directly
// from handlers, where they will be converted to INVALID_ARGUMENT Connect errors.
package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError represents a field validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation failures
type ValidationErrors []*ValidationError

func (es ValidationErrors) Error() string {
	if len(es) == 0 {
		return ""
	}
	if len(es) == 1 {
		return es[0].Error()
	}
	
	var msgs []string
	for _, e := range es {
		msgs = append(msgs, e.Message)
	}
	return fmt.Sprintf("%d validation errors: %s", len(es), strings.Join(msgs, "; "))
}

// appendError appends a validation error to a slice
func appendError(errs ValidationErrors, field, message string) ValidationErrors {
	return append(errs, &ValidationError{Field: field, Message: message})
}

// Common validation patterns
var (
	// Alphanumeric with underscores and hyphens (for names, identifiers)
	alphanumericPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	
	// DNS label pattern (for hostnames, domain names)
	dnsLabelPattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
	
	// Path safety: no path traversal
	pathTraversalPattern = regexp.MustCompile(`\.\.`)
)

// ValidateVMName validates a VM name
// Rules: 1-64 characters, alphanumeric + underscore/hyphen, must start with letter
func ValidateVMName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 64 {
		errs = appendError(errs, "name", "must be 64 characters or less")
	}
	
	if len(name) < 1 {
		return errs
	}
	
	// Must start with a letter
	if !unicode.IsLetter(rune(name[0])) {
		errs = appendError(errs, "name", "must start with a letter")
	}
	
	// Only alphanumeric, underscore, hyphen
	if !alphanumericPattern.MatchString(name) {
		errs = appendError(errs, "name", "contains invalid characters (only letters, numbers, underscore, hyphen allowed)")
	}
	
	return errs
}

// ValidateVMDescription validates a VM description
func ValidateVMDescription(description string) ValidationErrors {
	var errs ValidationErrors
	
	if description == "" {
		return nil // Optional field
	}
	
	if len(description) > 1024 {
		errs = appendError(errs, "description", "must be 1024 characters or less")
	}
	
	return errs
}

// ValidateMemoryGB validates memory size in GB
func ValidateMemoryGB(memory float64, min, max float64) ValidationErrors {
	var errs ValidationErrors
	
	if memory <= 0 {
		return appendError(errs, "memory", "must be greater than 0")
	}
	
	if memory < min {
		errs = appendError(errs, "memory", fmt.Sprintf("must be at least %.1f GB", min))
	}
	
	if memory > max {
		errs = appendError(errs, "memory", fmt.Sprintf("must be at most %.1f GB", max))
	}
	
	return errs
}

// ValidateCPUCores validates CPU core count
func ValidateCPUCores(cores int, min, max int) ValidationErrors {
	var errs ValidationErrors
	
	if cores <= 0 {
		return appendError(errs, "cpu_cores", "must be greater than 0")
	}
	
	if cores < min {
		errs = appendError(errs, "cpu_cores", fmt.Sprintf("must be at least %d", min))
	}
	
	if cores > max {
		errs = appendError(errs, "cpu_cores", fmt.Sprintf("must be at most %d", max))
	}
	
	return errs
}

// ValidateDiskSizeGB validates disk size in GB
func ValidateDiskSizeGB(size float64, min, max float64) ValidationErrors {
	var errs ValidationErrors
	
	if size <= 0 {
		return appendError(errs, "disk_size", "must be greater than 0")
	}
	
	if size < min {
		errs = appendError(errs, "disk_size", fmt.Sprintf("must be at least %.1f GB", min))
	}
	
	if size > max {
		errs = appendError(errs, "disk_size", fmt.Sprintf("must be at most %.1f GB", max))
	}
	
	return errs
}

// ValidateEmail validates email format
func ValidateEmail(email string) ValidationErrors {
	var errs ValidationErrors
	
	if email == "" {
		return appendError(errs, "email", "is required")
	}
	
	_, err := mail.ParseAddress(email)
	if err != nil {
		errs = appendError(errs, "email", "is not a valid email address")
	}
	
	// Additional length check
	if len(email) > 254 {
		errs = appendError(errs, "email", "must be 254 characters or less")
	}
	
	return errs
}

// ValidatePassword validates password strength
func ValidatePassword(password string) ValidationErrors {
	var errs ValidationErrors
	
	if password == "" {
		return appendError(errs, "password", "is required")
	}
	
	if len(password) < 8 {
		errs = appendError(errs, "password", "must be at least 8 characters")
	}
	
	if len(password) > 128 {
		errs = appendError(errs, "password", "must be 128 characters or less")
	}
	
	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false
	
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		errs = appendError(errs, "password", "must contain at least one uppercase letter")
	}
	
	if !hasLower {
		errs = appendError(errs, "password", "must contain at least one lowercase letter")
	}
	
	if !hasDigit {
		errs = appendError(errs, "password", "must contain at least one number")
	}
	
	if !hasSpecial {
		errs = appendError(errs, "password", "must contain at least one special character")
	}
	
	return errs
}

// ValidateTagName validates a tag name
func ValidateTagName(tag string) ValidationErrors {
	var errs ValidationErrors
	
	if tag == "" {
		return appendError(errs, "tag", "cannot be empty")
	}
	
	if len(tag) > 64 {
		errs = appendError(errs, "tag", "must be 64 characters or less")
	}
	
	// Tags should be lowercase alphanumeric with hyphens
	tagPattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !tagPattern.MatchString(tag) {
		errs = appendError(errs, "tag", "must be lowercase alphanumeric with hyphens only")
	}
	
	return errs
}

// ValidateTags validates a list of tags
func ValidateTags(tags []string) ValidationErrors {
	var errs ValidationErrors
	
	if len(tags) > 20 {
		errs = appendError(errs, "tags", "cannot contain more than 20 tags")
	}
	
	seen := make(map[string]bool)
	for i, tag := range tags {
		tagErrs := ValidateTagName(tag)
		for _, e := range tagErrs {
			e.Field = fmt.Sprintf("tags[%d]", i)
			errs = append(errs, e)
		}
		
		if seen[tag] {
			errs = appendError(errs, fmt.Sprintf("tags[%d]", i), "duplicate tag")
		}
		seen[tag] = true
	}
	
	return errs
}

// ValidateBridgeName validates a network bridge name
func ValidateBridgeName(bridge string) ValidationErrors {
	var errs ValidationErrors
	
	if bridge == "" {
		return appendError(errs, "bridge", "is required")
	}
	
	if len(bridge) > 32 {
		errs = appendError(errs, "bridge", "must be 32 characters or less")
	}
	
	// Bridge names should be alphanumeric
	if !alphanumericPattern.MatchString(bridge) {
		errs = appendError(errs, "bridge", "contains invalid characters")
	}
	
	return errs
}

// ValidateStoragePoolName validates a storage pool name
func ValidateStoragePoolName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 64 {
		errs = appendError(errs, "name", "must be 64 characters or less")
	}
	
	if !alphanumericPattern.MatchString(name) {
		errs = appendError(errs, "name", "contains invalid characters (only letters, numbers, underscore, hyphen allowed)")
	}
	
	return errs
}

// ValidatePath validates a file system path for safety
func ValidatePath(path string, fieldName string) ValidationErrors {
	var errs ValidationErrors
	
	if path == "" {
		return appendError(errs, fieldName, "is required")
	}
	
	// Check for path traversal
	if pathTraversalPattern.MatchString(path) {
		errs = appendError(errs, fieldName, "cannot contain path traversal (..)")
	}
	
	// Must be absolute path
	if !strings.HasPrefix(path, "/") {
		errs = appendError(errs, fieldName, "must be an absolute path")
	}
	
	return errs
}

// ValidateISOName validates an ISO file name
func ValidateISOName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 255 {
		errs = appendError(errs, "name", "must be 255 characters or less")
	}
	
	// Must end with .iso
	if !strings.HasSuffix(strings.ToLower(name), ".iso") {
		errs = appendError(errs, "name", "must have .iso extension")
	}
	
	// Check for path traversal
	if pathTraversalPattern.MatchString(name) {
		errs = appendError(errs, "name", "cannot contain path traversal (..)")
	}
	
	return errs
}

// ValidateBackupName validates a backup name
func ValidateBackupName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 128 {
		errs = appendError(errs, "name", "must be 128 characters or less")
	}
	
	return errs
}

// ValidateSnapshotName validates a snapshot name
func ValidateSnapshotName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 64 {
		errs = appendError(errs, "name", "must be 64 characters or less")
	}
	
	if !alphanumericPattern.MatchString(name) {
		errs = appendError(errs, "name", "contains invalid characters")
	}
	
	return errs
}

// ValidateAlertRuleName validates an alert rule name
func ValidateAlertRuleName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 128 {
		errs = appendError(errs, "name", "must be 128 characters or less")
	}
	
	return errs
}

// ValidateNotificationChannelName validates a notification channel name
func ValidateNotificationChannelName(name string) ValidationErrors {
	var errs ValidationErrors
	
	if name == "" {
		return appendError(errs, "name", "is required")
	}
	
	if len(name) > 128 {
		errs = appendError(errs, "name", "must be 128 characters or less")
	}
	
	return errs
}

// ValidateWebhookURL validates a webhook URL
func ValidateWebhookURL(url string) ValidationErrors {
	var errs ValidationErrors
	
	if url == "" {
		return appendError(errs, "url", "is required")
	}
	
	if len(url) > 2048 {
		errs = appendError(errs, "url", "must be 2048 characters or less")
	}
	
	// Must start with http:// or https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		errs = appendError(errs, "url", "must start with http:// or https://")
	}
	
	return errs
}

// ValidateEmailRecipient validates an email recipient address
func ValidateEmailRecipient(email string) ValidationErrors {
	return ValidateEmail(email)
}

// ValidateSMTPServer validates an SMTP server address
func ValidateSMTPServer(server string) ValidationErrors {
	var errs ValidationErrors
	
	if server == "" {
		return appendError(errs, "smtp_server", "is required")
	}
	
	// Can be hostname:port or just hostname
	parts := strings.Split(server, ":")
	hostname := parts[0]
	
	if !dnsLabelPattern.MatchString(hostname) && !strings.Contains(hostname, ".") {
		errs = appendError(errs, "smtp_server", "must be a valid hostname or IP address")
	}
	
	if len(server) > 255 {
		errs = appendError(errs, "smtp_server", "must be 255 characters or less")
	}
	
	return errs
}

// ValidateCronExpression validates a cron expression (basic validation)
func ValidateCronExpression(expr string) ValidationErrors {
	var errs ValidationErrors
	
	if expr == "" {
		return appendError(errs, "cron_expression", "is required")
	}
	
	// Basic format check: should have 5 or 6 space-separated fields
	fields := strings.Fields(expr)
	if len(fields) < 5 || len(fields) > 6 {
		errs = appendError(errs, "cron_expression", "must have 5 or 6 space-separated fields")
	}
	
	if len(expr) > 256 {
		errs = appendError(errs, "cron_expression", "must be 256 characters or less")
	}
	
	return errs
}

// ValidateRetentionDays validates retention period
func ValidateRetentionDays(days int, min, max int) ValidationErrors {
	var errs ValidationErrors

	if days < min {
		errs = appendError(errs, "retention_days", fmt.Sprintf("must be at least %d days", min))
	}

	if days > max {
		errs = appendError(errs, "retention_days", fmt.Sprintf("must be at most %d days", max))
	}

	return errs
}

// uuidPattern matches UUID format: 8-4-4-4-12 hex digits
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateUUID validates that a string is a valid UUID format.
// Accepts both uppercase and lowercase hex digits.
func ValidateUUID(id, fieldName string) ValidationErrors {
	var errs ValidationErrors

	if id == "" {
		return appendError(errs, fieldName, "is required")
	}

	if !uuidPattern.MatchString(id) {
		errs = appendError(errs, fieldName, "must be a valid UUID")
	}

	return errs
}

// ValidateUUIDOptional validates UUID format but allows empty strings.
// Use this for optional UUID fields.
func ValidateUUIDOptional(id, fieldName string) ValidationErrors {
	if id == "" {
		return nil
	}
	return ValidateUUID(id, fieldName)
}
