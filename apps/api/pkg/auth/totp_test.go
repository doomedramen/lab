package auth

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

func testMFA() *MFA {
	return NewMFA("test-issuer")
}

func TestMFA_GenerateSecret_ReturnsSecret(t *testing.T) {
	mfa := testMFA()

	secret, err := mfa.GenerateSecret("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	if secret.Secret == "" {
		t.Error("Secret should not be empty")
	}
}

func TestMFA_GenerateSecret_ReturnsQRCodeURL(t *testing.T) {
	mfa := testMFA()

	secret, err := mfa.GenerateSecret("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	if secret.QRCodeURL == "" {
		t.Error("QRCodeURL should not be empty")
	}
	if !strings.HasPrefix(secret.QRCodeURL, "otpauth://totp/") {
		t.Errorf("QRCodeURL should start with otpauth://totp/, got: %s", secret.QRCodeURL[:20])
	}
}

func TestMFA_GenerateSecret_ReturnsManualKey(t *testing.T) {
	mfa := testMFA()

	secret, err := mfa.GenerateSecret("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	if secret.ManualKey == "" {
		t.Error("ManualKey should not be empty")
	}
	// Manual key should contain the same secret (possibly with spaces)
	if strings.ReplaceAll(secret.ManualKey, " ", "") != secret.Secret {
		t.Errorf("ManualKey should match Secret (ignoring spaces). manual=%s, secret=%s",
			secret.ManualKey, secret.Secret)
	}
}

func TestMFA_GenerateSecret_DifferentUsers(t *testing.T) {
	mfa := testMFA()

	secret1, _ := mfa.GenerateSecret("user-1", "user1@example.com")
	secret2, _ := mfa.GenerateSecret("user-2", "user2@example.com")

	if secret1.Secret == secret2.Secret {
		t.Error("different users should get different secrets")
	}
}

func TestMFA_GenerateSecret_QRCodeContainsIssuer(t *testing.T) {
	mfa := testMFA()

	secret, err := mfa.GenerateSecret("user-123", "test@example.com")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	if !strings.Contains(secret.QRCodeURL, "issuer=test-issuer") {
		t.Errorf("QRCodeURL should contain issuer, got: %s", secret.QRCodeURL)
	}
}

func TestMFA_VerifyCode_Valid(t *testing.T) {
	mfa := testMFA()

	secret, _ := mfa.GenerateSecret("user-123", "test@example.com")

	// Generate a valid code using the same library
	validCode, err := totp.GenerateCode(secret.Secret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode error: %v", err)
	}

	err = mfa.VerifyCode(secret.Secret, validCode)
	if err != nil {
		t.Errorf("VerifyCode should succeed for valid code: %v", err)
	}
}

func TestMFA_VerifyCode_Invalid(t *testing.T) {
	mfa := testMFA()

	secret, _ := mfa.GenerateSecret("user-123", "test@example.com")

	err := mfa.VerifyCode(secret.Secret, "000000")
	if err == nil {
		t.Error("VerifyCode should fail for invalid code")
	}
}

func TestMFA_VerifyCode_WrongLength(t *testing.T) {
	mfa := testMFA()

	tests := []string{
		"12345",   // too short
		"1234567", // too long
		"abcdef",  // letters
		"",        // empty
	}

	for _, code := range tests {
		err := mfa.VerifyCode("any-secret", code)
		if err == nil {
			t.Errorf("expected error for code %q", code)
		}
	}
}

func TestMFA_VerifyCode_NonNumeric(t *testing.T) {
	mfa := testMFA()

	err := mfa.VerifyCode("secret", "abcdef")
	if err == nil {
		t.Error("expected error for non-numeric code")
	}
}

func TestMFA_GenerateBackupCodes_Count(t *testing.T) {
	mfa := testMFA()

	codes, err := mfa.GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes error: %v", err)
	}

	if len(codes) != MFABackupCodeCount {
		t.Errorf("expected %d codes, got %d", MFABackupCodeCount, len(codes))
	}
}

func TestMFA_GenerateBackupCodes_Length(t *testing.T) {
	mfa := testMFA()

	codes, err := mfa.GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes error: %v", err)
	}

	for i, code := range codes {
		if len(code) != MFABackupCodeLength {
			t.Errorf("code[%d] length: got %d, want %d", i, len(code), MFABackupCodeLength)
		}
	}
}

func TestMFA_GenerateBackupCodes_AllDifferent(t *testing.T) {
	mfa := testMFA()

	codes, err := mfa.GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes error: %v", err)
	}

	seen := make(map[string]bool)
	for i, code := range codes {
		if seen[code] {
			t.Errorf("duplicate code at index %d: %s", i, code)
		}
		seen[code] = true
	}
}

func TestMFA_GenerateBackupCodes_NumericOnly(t *testing.T) {
	mfa := testMFA()

	codes, err := mfa.GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes error: %v", err)
	}

	for i, code := range codes {
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Errorf("code[%d] contains non-numeric character: %c", i, c)
			}
		}
	}
}

func TestMFA_HashBackupCodes_ReturnsValidJSON(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321"}
	hashedJSON, err := mfa.HashBackupCodes(codes)
	if err != nil {
		t.Fatalf("HashBackupCodes error: %v", err)
	}

	// Should be valid JSON array
	var parsed []string
	if err := json.Unmarshal([]byte(hashedJSON), &parsed); err != nil {
		t.Errorf("HashBackupCodes should return valid JSON: %v", err)
	}
}

func TestMFA_HashBackupCodes_AllCodesHashed(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321", "1111111111"}
	hashedJSON, err := mfa.HashBackupCodes(codes)
	if err != nil {
		t.Fatalf("HashBackupCodes error: %v", err)
	}

	var hashed []string
	json.Unmarshal([]byte(hashedJSON), &hashed)

	if len(hashed) != len(codes) {
		t.Errorf("expected %d hashed codes, got %d", len(codes), len(hashed))
	}

	// Each should be a bcrypt hash
	for i, h := range hashed {
		if !strings.HasPrefix(h, "$2a$") {
			t.Errorf("hashed[%d] should be bcrypt hash, got: %s", i, h[:4])
		}
	}
}

func TestMFA_VerifyBackupCode_Valid(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321"}
	hashedJSON, _ := mfa.HashBackupCodes(codes)

	valid, err := mfa.VerifyBackupCode(hashedJSON, "1234567890")
	if err != nil {
		t.Fatalf("VerifyBackupCode error: %v", err)
	}
	if !valid {
		t.Error("VerifyBackupCode should return true for valid code")
	}
}

func TestMFA_VerifyBackupCode_Invalid(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321"}
	hashedJSON, _ := mfa.HashBackupCodes(codes)

	valid, err := mfa.VerifyBackupCode(hashedJSON, "9999999999")
	if err != nil {
		t.Fatalf("VerifyBackupCode error: %v", err)
	}
	if valid {
		t.Error("VerifyBackupCode should return false for invalid code")
	}
}

func TestMFA_VerifyBackupCode_InvalidJSON(t *testing.T) {
	mfa := testMFA()

	_, err := mfa.VerifyBackupCode("not-valid-json", "1234567890")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMFA_RemoveBackupCode_RemovesUsedCode(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321", "1111111111"}
	hashedJSON, _ := mfa.HashBackupCodes(codes)

	// Use one code
	newJSON, err := mfa.RemoveBackupCode(hashedJSON, "1234567890")
	if err != nil {
		t.Fatalf("RemoveBackupCode error: %v", err)
	}

	var newCodes []string
	json.Unmarshal([]byte(newJSON), &newCodes)

	if len(newCodes) != 2 {
		t.Errorf("expected 2 codes after removal, got %d", len(newCodes))
	}

	// The used code should no longer verify
	valid, _ := mfa.VerifyBackupCode(newJSON, "1234567890")
	if valid {
		t.Error("used code should not verify after removal")
	}

	// Other codes should still work
	valid, _ = mfa.VerifyBackupCode(newJSON, "0987654321")
	if !valid {
		t.Error("other codes should still verify")
	}
}

func TestMFA_RemoveBackupCode_OnlyRemovesOne(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321"}
	hashedJSON, _ := mfa.HashBackupCodes(codes)

	newJSON, _ := mfa.RemoveBackupCode(hashedJSON, "1234567890")

	var newCodes []string
	json.Unmarshal([]byte(newJSON), &newCodes)

	if len(newCodes) != 1 {
		t.Errorf("expected 1 code remaining, got %d", len(newCodes))
	}
}

func TestMFA_RemoveBackupCode_InvalidJSON(t *testing.T) {
	mfa := testMFA()

	_, err := mfa.RemoveBackupCode("not-valid-json", "1234567890")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestMFA_RemoveBackupCode_CodeNotFound(t *testing.T) {
	mfa := testMFA()

	codes := []string{"1234567890", "0987654321"}
	hashedJSON, _ := mfa.HashBackupCodes(codes)

	// Try to remove a code that doesn't exist
	newJSON, err := mfa.RemoveBackupCode(hashedJSON, "9999999999")
	if err != nil {
		t.Fatalf("RemoveBackupCode should not error: %v", err)
	}

	var newCodes []string
	json.Unmarshal([]byte(newJSON), &newCodes)

	// All original codes should remain
	if len(newCodes) != 2 {
		t.Errorf("expected 2 codes (none removed), got %d", len(newCodes))
	}
}

func TestNewMFA(t *testing.T) {
	issuer := "my-app"
	mfa := NewMFA(issuer)

	if mfa == nil {
		t.Fatal("NewMFA returned nil")
	}
	if mfa.issuer != issuer {
		t.Errorf("issuer: got %q, want %q", mfa.issuer, issuer)
	}
}

func TestFormatSecret(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABCD", "ABCD"},
		{"ABCDEFGH", "ABCD EFGH"},
		{"ABCDEFGHIJKL", "ABCD EFGH IJKL"},
	}

	for _, tt := range tests {
		result := formatSecret(tt.input)
		if result != tt.expected {
			t.Errorf("formatSecret(%q): got %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestMFABackupCodeConstants(t *testing.T) {
	if MFABackupCodeCount != 10 {
		t.Errorf("MFABackupCodeCount: got %d, want 10", MFABackupCodeCount)
	}
	if MFABackupCodeLength != 10 {
		t.Errorf("MFABackupCodeLength: got %d, want 10", MFABackupCodeLength)
	}
}
