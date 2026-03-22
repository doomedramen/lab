package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the cost parameter for bcrypt hashing
	// Higher values are more secure but slower
	// 12 is a good balance for production use
	BcryptCost = 12
)

// Password provides password hashing and verification
type Password struct {
	cost int
}

// NewPassword creates a new Password handler
func NewPassword() *Password {
	return &Password{cost: BcryptCost}
}

// Hash hashes a password using bcrypt
func (p *Password) Hash(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// Verify compares a password with its hash
func (p *Password) Verify(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// NeedsRehash checks if a password hash needs to be rehashed
// This is useful when upgrading bcrypt cost over time
func (p *Password) NeedsRehash(hash string) bool {
	// bcrypt hashes start with $2a$, $2b$, or $2y$ followed by cost$
	// Format: $2a$12$... where 12 is the cost
	if len(hash) < 7 {
		return true
	}

	// Extract cost from hash
	costStr := hash[4:6]
	var cost int
	fmt.Sscanf(costStr, "%d", &cost)

	return cost < p.cost
}

// GenerateSecureToken generates a cryptographically secure random token
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidatePasswordStrength checks if a password meets security requirements
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?"

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case containsRune(specialChars, char):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower {
		return errors.New("password must contain both uppercase and lowercase letters")
	}
	if !hasDigit {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

func containsRune(s string, r rune) bool {
	for _, sr := range s {
		if sr == r {
			return true
		}
	}
	return false
}
