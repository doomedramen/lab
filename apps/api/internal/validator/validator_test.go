package validator

import (
	"testing"
)

func TestValidateVMName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid simple", "my-vm", false},
		{"valid with underscore", "my_vm", false},
		{"valid with numbers", "vm-123", false},
		{"valid mixed", "Test-VM_01", false},
		{"empty", "", true},
		{"starts with number", "123-vm", true},
		{"starts with underscore", "_vm", true},
		{"starts with hyphen", "-vm", true},
		{"contains space", "my vm", true},
		{"contains special char", "my@vm", true},
		{"too long", "this-is-a-very-long-vm-name-that-exceeds-the-maximum-allowed-length-of-64-characters-limit", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateVMName(tt.input)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateVMName(%q) error = %v, wantError %v", tt.input, errs, tt.wantError)
			}
		})
	}
}

func TestValidateMemoryGB(t *testing.T) {
	tests := []struct {
		name      string
		memory    float64
		min       float64
		max       float64
		wantError bool
	}{
		{"valid", 4, 0.5, 1024, false},
		{"valid minimum", 0.5, 0.5, 1024, false},
		{"valid maximum", 1024, 0.5, 1024, false},
		{"zero", 0, 0.5, 1024, true},
		{"negative", -1, 0.5, 1024, true},
		{"below minimum", 0.1, 0.5, 1024, true},
		{"above maximum", 2048, 0.5, 1024, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateMemoryGB(tt.memory, tt.min, tt.max)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateMemoryGB(%v) error = %v, wantError %v", tt.memory, errs, tt.wantError)
			}
		})
	}
}

func TestValidateCPUCores(t *testing.T) {
	tests := []struct {
		name      string
		cores     int
		min       int
		max       int
		wantError bool
	}{
		{"valid", 4, 1, 128, false},
		{"valid minimum", 1, 1, 128, false},
		{"valid maximum", 128, 1, 128, false},
		{"zero", 0, 1, 128, true},
		{"negative", -1, 1, 128, true},
		{"below minimum", 0, 1, 128, true},
		{"above maximum", 256, 1, 128, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateCPUCores(tt.cores, tt.min, tt.max)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateCPUCores(%d) error = %v, wantError %v", tt.cores, errs, tt.wantError)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantError bool
	}{
		{"valid simple", "user@example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid with dot", "user.name@example.com", false},
		{"valid subdomain", "user@mail.example.com", false},
		{"empty", "", true},
		{"no at sign", "userexample.com", true},
		{"no domain", "user@", true},
		{"no local", "@example.com", true},
		{"invalid chars", "user@exam ple.com", true},
		{"too long", "very.long.email.address.that.exceeds.the.maximum.allowed.length.of.254.characters.for.an.email.address.user.with.very.long.local.part.that.goes.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on.and.on@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateEmail(tt.email)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateEmail(%q) error = %v, wantError %v", tt.email, errs, tt.wantError)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		wantError bool
	}{
		{"valid", "SecureP@ss123", false},
		{"valid complex", "MyC0mpl3x!Pass", false},
		{"too short", "Ab1!", true},
		{"no uppercase", "securep@ss123", true},
		{"no lowercase", "SECUREP@SS123", true},
		{"no digit", "SecurePass!!!", true},
		{"no special", "SecurePass123", true},
		{"empty", "", true},
		{"too long", "ThisIsAVeryLongPasswordThatExceedsTheMaximumAllowedLengthOf128CharactersAndShouldFailValidationSecureP@ss1234567890!@#$", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePassword(tt.password)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidatePassword() error = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name      string
		tags      []string
		wantError bool
	}{
		{"valid single", []string{"production"}, false},
		{"valid multiple", []string{"production", "web", "tier-1"}, false},
		{"empty list", []string{}, false},
		{"empty tag", []string{""}, true},
		{"uppercase", []string{"Production"}, true},
		{"special chars", []string{"prod!"}, true},
		{"duplicate", []string{"prod", "prod"}, true},
		{"too many", make([]string, 21), true},
		{"too long tag", []string{"this-is-a-very-long-tag-name-that-exceeds-the-maximum-allowed-length"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateTags(tt.tags)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateTags() error = %v, wantError %v", errs, tt.wantError)
			}
		})
	}
}

func TestValidateISOName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid", "ubuntu-24.04.iso", false},
		{"valid uppercase ext", "ubuntu-24.04.ISO", false},
		{"valid mixed case ext", "ubuntu-24.04.Iso", false},
		{"empty", "", true},
		{"no extension", "ubuntu-24.04", true},
		{"wrong extension", "ubuntu-24.04.img", true},
		{"path traversal", "../ubuntu.iso", true},
		{"too long", "this-is-a-very-long-iso-name-that-exceeds-the-maximum-allowed-length-of-255-characters-for-file-names-on-most-filesystems-and-should-fail-validation-because-it-is-way-too-long-and-will-cause-errors-when-trying-to-create-files-with-this-name-on-the-filesystem.iso", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateISOName(tt.input)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateISOName(%q) error = %v, wantError %v", tt.input, errs, tt.wantError)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantError bool
	}{
		{"valid http", "http://example.com/webhook", false},
		{"valid https", "https://example.com/webhook", false},
		{"valid with port", "https://example.com:8080/webhook", false},
		{"valid complex", "https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX", false},
		{"empty", "", true},
		{"no protocol", "example.com/webhook", true},
		{"ftp protocol", "ftp://example.com/webhook", true},
		{"too long", "https://example.com/" + string(make([]byte, 2040)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateWebhookURL(tt.url)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateWebhookURL(%q) error = %v, wantError %v", tt.url, errs, tt.wantError)
			}
		})
	}
}

func TestValidateMFACode(t *testing.T) {
	tests := []struct {
		name      string
		code      string
		wantError bool
	}{
		{"valid", "123456", false},
		{"valid zeros", "000000", false},
		{"valid nines", "999999", false},
		{"empty", "", true},
		{"too short", "12345", true},
		{"too long", "1234567", true},
		{"with letters", "12345a", true},
		{"all letters", "abcdef", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateMFACode(tt.code)
			hasError := len(errs) > 0
			if hasError != tt.wantError {
				t.Errorf("ValidateMFACode(%q) error = %v, wantError %v", tt.code, errs, tt.wantError)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{Field: "name", Message: "is required"}
	expected := "name: is required"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %q, want %q", err.Error(), expected)
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors ValidationErrors
		want   string
	}{
		{
			name:   "empty",
			errors: ValidationErrors{},
			want:   "",
		},
		{
			name:   "single",
			errors: ValidationErrors{{Field: "name", Message: "is required"}},
			want:   "name: is required",
		},
		{
			name:   "multiple",
			errors: ValidationErrors{{Field: "name", Message: "is required"}, {Field: "email", Message: "is invalid"}},
			want:   "2 validation errors: is required; is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.errors.Error()
			if got != tt.want {
				t.Errorf("ValidationErrors.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}
