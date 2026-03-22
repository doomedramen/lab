package auth

import (
	"strings"
	"testing"
)

func TestPassword_Hash_ValidPassword(t *testing.T) {
	p := NewPassword()
	hash, err := p.Hash("ValidPass123!")

	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if hash == "" {
		t.Error("Hash returned empty string")
	}
	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("Hash should start with $2a$, got: %s", hash[:4])
	}
}

func TestPassword_Hash_ShortPassword(t *testing.T) {
	p := NewPassword()
	_, err := p.Hash("short")

	if err == nil {
		t.Error("expected error for short password")
	}
	if err.Error() != "password must be at least 8 characters" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPassword_Hash_MinLengthPassword(t *testing.T) {
	p := NewPassword()
	// Exactly 8 characters
	hash, err := p.Hash("12345678")

	if err != nil {
		t.Fatalf("Hash should accept 8-char password: %v", err)
	}
	if hash == "" {
		t.Error("Hash returned empty string")
	}
}

func TestPassword_Hash_DifferentPasswords(t *testing.T) {
	p := NewPassword()
	hash1, _ := p.Hash("Password123!a")
	hash2, _ := p.Hash("Password123!b")

	if hash1 == hash2 {
		t.Error("different passwords should produce different hashes")
	}
}

func TestPassword_Hash_SamePasswordDifferentHashes(t *testing.T) {
	p := NewPassword()
	hash1, _ := p.Hash("SamePassword123!")
	hash2, _ := p.Hash("SamePassword123!")

	// Due to salt, same password should produce different hashes
	if hash1 == hash2 {
		t.Error("same password should produce different hashes due to salt")
	}
}

func TestPassword_Verify_CorrectPassword(t *testing.T) {
	p := NewPassword()
	password := "CorrectPassword123!"
	hash, _ := p.Hash(password)

	err := p.Verify(password, hash)
	if err != nil {
		t.Errorf("Verify should succeed for correct password: %v", err)
	}
}

func TestPassword_Verify_WrongPassword(t *testing.T) {
	p := NewPassword()
	hash, _ := p.Hash("CorrectPassword123!")

	err := p.Verify("WrongPassword123!", hash)
	if err == nil {
		t.Error("Verify should fail for wrong password")
	}
	if err.Error() != "invalid password" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPassword_Verify_InvalidHash(t *testing.T) {
	p := NewPassword()

	err := p.Verify("anypassword", "invalid-hash-format")
	if err == nil {
		t.Error("Verify should fail for invalid hash format")
	}
}

func TestPassword_Verify_EmptyHash(t *testing.T) {
	p := NewPassword()

	err := p.Verify("anypassword", "")
	if err == nil {
		t.Error("Verify should fail for empty hash")
	}
}

func TestPassword_NeedsRehash_OldCost(t *testing.T) {
	p := NewPassword()

	// Hash with cost 10 (less than default 12)
	lowerCostHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3/ItB/XBG/eCknfIrqS6"

	if !p.NeedsRehash(lowerCostHash) {
		t.Error("hash with cost 10 should need rehash when default is 12")
	}
}

func TestPassword_NeedsRehash_SameCost(t *testing.T) {
	p := NewPassword()

	// Hash with cost 12 (same as default)
	sameCostHash := "$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3/ItB/XBG/eCknfIrqS6"

	if p.NeedsRehash(sameCostHash) {
		t.Error("hash with same cost should not need rehash")
	}
}

func TestPassword_NeedsRehash_HigherCost(t *testing.T) {
	p := NewPassword()

	// Hash with cost 14 (higher than default 12)
	higherCostHash := "$2a$14$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3/ItB/XBG/eCknfIrqS6"

	if p.NeedsRehash(higherCostHash) {
		t.Error("hash with higher cost should not need rehash")
	}
}

func TestPassword_NeedsRehash_TooShort(t *testing.T) {
	p := NewPassword()

	if !p.NeedsRehash("short") {
		t.Error("short hash should need rehash")
	}
}

func TestPassword_NeedsRehash_Empty(t *testing.T) {
	p := NewPassword()

	if !p.NeedsRehash("") {
		t.Error("empty hash should need rehash")
	}
}

func TestValidatePasswordStrength_Valid(t *testing.T) {
	validPasswords := []string{
		"Password123!",
		"Abcdefg1@",
		"MyP@ssw0rd",
		"Test!ng123",
		"UPPERlower123!",
	}

	for _, pw := range validPasswords {
		err := ValidatePasswordStrength(pw)
		if err != nil {
			t.Errorf("password %q should be valid: %v", pw, err)
		}
	}
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	err := ValidatePasswordStrength("Sh0rt!")
	if err == nil {
		t.Error("expected error for short password")
	}
	if err.Error() != "password must be at least 8 characters" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidatePasswordStrength_NoUppercase(t *testing.T) {
	err := ValidatePasswordStrength("lowercase123!")
	if err == nil {
		t.Error("expected error for password without uppercase")
	}
	if !strings.Contains(err.Error(), "uppercase") {
		t.Errorf("error should mention uppercase: %v", err)
	}
}

func TestValidatePasswordStrength_NoLowercase(t *testing.T) {
	err := ValidatePasswordStrength("UPPERCASE123!")
	if err == nil {
		t.Error("expected error for password without lowercase")
	}
	if !strings.Contains(err.Error(), "lowercase") {
		t.Errorf("error should mention lowercase: %v", err)
	}
}

func TestValidatePasswordStrength_NoDigit(t *testing.T) {
	err := ValidatePasswordStrength("NoDigitsHere!")
	if err == nil {
		t.Error("expected error for password without digit")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("error should mention number: %v", err)
	}
}

func TestValidatePasswordStrength_NoSpecialChar(t *testing.T) {
	err := ValidatePasswordStrength("NoSpecialChars123")
	if err == nil {
		t.Error("expected error for password without special character")
	}
	if !strings.Contains(err.Error(), "special") {
		t.Errorf("error should mention special character: %v", err)
	}
}

func TestValidatePasswordStrength_AllSpecialChars(t *testing.T) {
	// Test that all defined special characters work
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	for _, c := range specialChars {
		password := "Password1" + string(c)
		err := ValidatePasswordStrength(password)
		if err != nil {
			t.Errorf("password with special char %q should be valid: %v", c, err)
		}
	}
}

func TestGenerateSecureToken_Length(t *testing.T) {
	tests := []int{16, 32, 64}

	for _, length := range tests {
		token, err := GenerateSecureToken(length)
		if err != nil {
			t.Fatalf("GenerateSecureToken(%d) error: %v", length, err)
		}
		// hex encoding doubles the length
		if len(token) != length*2 {
			t.Errorf("token length: got %d, want %d", len(token), length*2)
		}
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)

	// Generate 100 tokens and check they're all unique
	for i := 0; i < 100; i++ {
		token, err := GenerateSecureToken(16)
		if err != nil {
			t.Fatalf("GenerateSecureToken error: %v", err)
		}
		if tokens[token] {
			t.Errorf("duplicate token generated: %s", token)
		}
		tokens[token] = true
	}
}

func TestGenerateSecureToken_IsHex(t *testing.T) {
	token, err := GenerateSecureToken(16)
	if err != nil {
		t.Fatalf("GenerateSecureToken error: %v", err)
	}

	for _, c := range token {
		if !isHexChar(c) {
			t.Errorf("token contains non-hex character: %c", c)
		}
	}
}

func isHexChar(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}
