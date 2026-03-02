package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	// MFABackupCodeCount is the number of backup codes to generate
	MFABackupCodeCount = 10
	// MFABackupCodeLength is the length of each backup code
	MFABackupCodeLength = 10
)

// MFA provides TOTP-based multi-factor authentication
type MFA struct {
	issuer string
}

// MFASecret holds the TOTP secret and QR code information
type MFASecret struct {
	Secret     string `json:"secret"`
	QRCodeURL  string `json:"qr_code_url"`
	ManualKey  string `json:"manual_key"` // For manual entry in authenticator apps
}

// NewMFA creates a new MFA handler
func NewMFA(issuer string) *MFA {
	return &MFA{issuer: issuer}
}

// GenerateSecret creates a new TOTP secret for a user
func (m *MFA) GenerateSecret(userID, email string) (*MFASecret, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: email,
		SecretSize:  20, // 160 bits
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Extract the raw secret (without spaces)
	secret := key.Secret()

	// Create a manual key format (with spaces for readability)
	manualKey := formatSecret(secret)

	return &MFASecret{
		Secret:     secret,
		QRCodeURL:  key.URL(),
		ManualKey:  manualKey,
	}, nil
}

// VerifyCode validates a TOTP code
func (m *MFA) VerifyCode(secret, code string) error {
	// Validate code format (should be 6 digits)
	if len(code) != 6 {
		return errors.New("code must be 6 digits")
	}

	valid := totp.Validate(code, secret)
	if !valid {
		return errors.New("invalid TOTP code")
	}

	return nil
}

// GenerateBackupCodes creates backup codes for account recovery
func (m *MFA) GenerateBackupCodes() ([]string, error) {
	codes := make([]string, MFABackupCodeCount)

	for i := 0; i < MFABackupCodeCount; i++ {
		code, err := generateBackupCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}

	return codes, nil
}

// HashBackupCodes hashes backup codes for storage
func (m *MFA) HashBackupCodes(codes []string) (string, error) {
	// Hash each code
	hashedCodes := make([]string, len(codes))
	password := NewPassword()

	for i, code := range codes {
		hash, err := password.Hash(code)
		if err != nil {
			return "", err
		}
		hashedCodes[i] = hash
	}

	// Serialize to JSON
	data, err := json.Marshal(hashedCodes)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// VerifyBackupCode checks if a backup code matches any of the hashed codes
func (m *MFA) VerifyBackupCode(hashedCodesJSON, code string) (bool, error) {
	// Parse hashed codes
	var hashedCodes []string
	if err := json.Unmarshal([]byte(hashedCodesJSON), &hashedCodes); err != nil {
		return false, fmt.Errorf("failed to parse backup codes: %w", err)
	}

	// Check each code
	password := NewPassword()
	for _, hashed := range hashedCodes {
		if err := password.Verify(code, hashed); err == nil {
			return true, nil
		}
	}

	return false, nil
}

// RemoveBackupCode removes a used backup code from the list
func (m *MFA) RemoveBackupCode(hashedCodesJSON, usedCode string) (string, error) {
	// Parse hashed codes
	var hashedCodes []string
	if err := json.Unmarshal([]byte(hashedCodesJSON), &hashedCodes); err != nil {
		return "", fmt.Errorf("failed to parse backup codes: %w", err)
	}

	// Find and remove the used code
	password := NewPassword()
	newCodes := make([]string, 0, len(hashedCodes))

	for _, hashed := range hashedCodes {
		if err := password.Verify(usedCode, hashed); err != nil {
			// Keep unused codes
			newCodes = append(newCodes, hashed)
		}
	}

	// Serialize back to JSON
	data, err := json.Marshal(newCodes)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// generateBackupCode creates a single backup code
func generateBackupCode() (string, error) {
	bytes := make([]byte, MFABackupCodeLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Convert to numeric string
	code := hex.EncodeToString(bytes)
	// Keep only digits
	var numeric strings.Builder
	for _, c := range code {
		if c >= '0' && c <= '9' {
			numeric.WriteRune(c)
			if numeric.Len() >= MFABackupCodeLength {
				break
			}
		}
	}

	// If we don't have enough digits, pad with random digits
	for numeric.Len() < MFABackupCodeLength {
		digit := byte('0' + (bytes[len(bytes)-1] % 10))
		numeric.WriteByte(digit)
	}

	return numeric.String()[:MFABackupCodeLength], nil
}

// formatSecret formats a secret with spaces for readability
func formatSecret(secret string) string {
	// Group into 4-character chunks
	var builder strings.Builder
	for i, c := range secret {
		if i > 0 && i%4 == 0 {
			builder.WriteByte(' ')
		}
		builder.WriteRune(c)
	}
	return builder.String()
}
