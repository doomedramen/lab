package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Custom claims for our JWT tokens
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT configuration
type Config struct {
	SecretKey       []byte
	AccessTokenExp  time.Duration // e.g., 15 minutes
	RefreshTokenExp time.Duration // e.g., 7 days
	Issuer          string
}

// DefaultConfig returns a default config with sensible expiry defaults.
// NOTE: SecretKey must be set explicitly. There is no secure default.
// In production, always use a cryptographically secure random secret:
//   openssl rand -base64 32
func DefaultConfig() Config {
	return Config{
		SecretKey:       nil, // Must be set explicitly
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api",
	}
}

// JWT handles JWT token operations
type JWT struct {
	config Config
}

// NewJWT creates a new JWT handler
func NewJWT(config Config) *JWT {
	return &JWT{config: config}
}

// GenerateAccessToken creates a short-lived access token
// Returns the token string and the JTI (JWT ID) for session tracking
func (j *JWT) GenerateAccessToken(userID, email, role string) (string, string, error) {
	jti := generateTokenID()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.config.AccessTokenExp)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.config.Issuer,
			Subject:   userID,
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.config.SecretKey)
	if err != nil {
		return "", "", err
	}
	return tokenString, jti, nil
}

// GenerateRefreshToken creates a long-lived refresh token
// Note: Refresh tokens are stored in DB and validated there
// This just creates the raw token string
func (j *JWT) GenerateRefreshToken() (string, error) {
	return generateSecureToken()
}

// ValidateAccessToken validates an access token and returns claims
func (j *JWT) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.config.SecretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate issuer
	if claims.Issuer != j.config.Issuer {
		return nil, errors.New("invalid token issuer")
	}

	return claims, nil
}

// GetTokenExpiry returns the access token expiry duration
func (j *JWT) GetTokenExpiry() time.Duration {
	return j.config.AccessTokenExp
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func (j *JWT) GetRefreshTokenExpiry() time.Duration {
	return j.config.RefreshTokenExp
}

// generateTokenID creates a unique token ID
func generateTokenID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp
		return fmt.Sprintf("tok_%d", time.Now().UnixNano())
	}
	return "tok_" + hex.EncodeToString(bytes)
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding
	token := base64.URLEncoding.EncodeToString(bytes)
	return "lab_" + token, nil
}

// GenerateAPIKey creates a new API key with prefix for identification
func GenerateAPIKey() (string, string, error) {
	// Generate 32 bytes of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}

	// Full key (shown only once to user)
	fullKey := "labkey_" + base64.URLEncoding.EncodeToString(bytes)

	// Prefix for identification (first 8 chars after prefix)
	prefix := fullKey[:15] // "labkey_" + 8 chars

	return fullKey, prefix, nil
}
