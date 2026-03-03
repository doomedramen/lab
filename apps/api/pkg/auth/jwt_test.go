package auth

import (
	"strings"
	"testing"
	"time"
)

func testJWTConfig() Config {
	return Config{
		SecretKey:       []byte("test-secret-key-for-testing"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api-test",
	}
}

func TestJWT_GenerateAccessToken_ValidInput(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token, jti, err := jwt.GenerateAccessToken("user-123", "test@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateAccessToken error: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
	if jti == "" {
		t.Error("jti should not be empty")
	}

	// JWT has 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("JWT should have 3 parts, got %d", len(parts))
	}
}

func TestJWT_GenerateAccessToken_DifferentUsers(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token1, _, _ := jwt.GenerateAccessToken("user-1", "user1@example.com", "admin")
	token2, _, _ := jwt.GenerateAccessToken("user-2", "user2@example.com", "viewer")

	if token1 == token2 {
		t.Error("different users should get different tokens")
	}
}

func TestJWT_GenerateAccessToken_DifferentTimes(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token1, _, _ := jwt.GenerateAccessToken("user-1", "user1@example.com", "admin")
	time.Sleep(time.Second)
	token2, _, _ := jwt.GenerateAccessToken("user-1", "user1@example.com", "admin")

	// Tokens generated at different times should be different (due to iat/exp)
	if token1 == token2 {
		t.Error("tokens generated at different times should differ")
	}
}

func TestJWT_ValidateAccessToken_Valid(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token, jti, _ := jwt.GenerateAccessToken("user-123", "test@example.com", "admin")
	claims, err := jwt.ValidateAccessToken(token)

	if err != nil {
		t.Fatalf("ValidateAccessToken error: %v", err)
	}
	if claims.UserID != "user-123" {
		t.Errorf("UserID: got %q, want user-123", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email: got %q, want test@example.com", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("Role: got %q, want admin", claims.Role)
	}
	if claims.Issuer != "lab-api-test" {
		t.Errorf("Issuer: got %q, want lab-api-test", claims.Issuer)
	}
	if claims.ID != jti {
		t.Errorf("ID (JTI): got %q, want %q", claims.ID, jti)
	}
}

func TestJWT_ValidateAccessToken_Expired(t *testing.T) {
	config := Config{
		SecretKey:       []byte("test-secret-key-for-testing"),
		AccessTokenExp:  -1 * time.Hour, // Already expired
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api-test",
	}
	jwt := NewJWT(config)

	token, _, _ := jwt.GenerateAccessToken("user-123", "test@example.com", "admin")
	_, err := jwt.ValidateAccessToken(token)

	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestJWT_ValidateAccessToken_InvalidSignature(t *testing.T) {
	jwt1 := NewJWT(Config{
		SecretKey:       []byte("secret-key-1"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api-test",
	})
	jwt2 := NewJWT(Config{
		SecretKey:       []byte("secret-key-2"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "lab-api-test",
	})

	token, _, _ := jwt1.GenerateAccessToken("user-123", "test@example.com", "admin")
	_, err := jwt2.ValidateAccessToken(token)

	if err == nil {
		t.Error("expected error for invalid signature")
	}
}

func TestJWT_ValidateAccessToken_WrongIssuer(t *testing.T) {
	jwt1 := NewJWT(Config{
		SecretKey:       []byte("test-secret-key-for-testing"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "issuer-1",
	})
	jwt2 := NewJWT(Config{
		SecretKey:       []byte("test-secret-key-for-testing"),
		AccessTokenExp:  15 * time.Minute,
		RefreshTokenExp: 7 * 24 * time.Hour,
		Issuer:          "issuer-2",
	})

	token, _, _ := jwt1.GenerateAccessToken("user-123", "test@example.com", "admin")
	_, err := jwt2.ValidateAccessToken(token)

	if err == nil {
		t.Error("expected error for wrong issuer")
	}
	if !strings.Contains(err.Error(), "issuer") {
		t.Errorf("error should mention issuer: %v", err)
	}
}

func TestJWT_ValidateAccessToken_Malformed(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	tests := []string{
		"",
		"not-a-jwt",
		"a.b",
		"a.b.c.d",
		"invalid.base64.signature",
	}

	for _, token := range tests {
		_, err := jwt.ValidateAccessToken(token)
		if err == nil {
			t.Errorf("expected error for malformed token: %q", token)
		}
	}
}

func TestJWT_ValidateAccessToken_Tampered(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token, _, _ := jwt.GenerateAccessToken("user-123", "test@example.com", "admin")

	// Tamper with the token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(parts))
	}

	// Change a character in the payload
	tamperedPayload := parts[1]
	if len(tamperedPayload) > 0 {
		tamperedPayload = tamperedPayload[:len(tamperedPayload)-1] + "X"
	}
	tamperedToken := parts[0] + "." + tamperedPayload + "." + parts[2]

	_, err := jwt.ValidateAccessToken(tamperedToken)
	if err == nil {
		t.Error("expected error for tampered token")
	}
}

func TestJWT_GenerateRefreshToken_HasPrefix(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token, err := jwt.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}

	if !strings.HasPrefix(token, "lab_") {
		t.Errorf("refresh token should start with 'lab_', got: %s", token[:4])
	}
}

func TestJWT_GenerateRefreshToken_Uniqueness(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token, err := jwt.GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken error: %v", err)
		}
		if tokens[token] {
			t.Errorf("duplicate refresh token: %s", token)
		}
		tokens[token] = true
	}
}

func TestJWT_GenerateRefreshToken_Length(t *testing.T) {
	jwt := NewJWT(testJWTConfig())

	token, err := jwt.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken error: %v", err)
	}

	// 32 bytes -> 44 base64 chars + 4 prefix = 48+ chars
	if len(token) < 48 {
		t.Errorf("refresh token too short: %d", len(token))
	}
}

func TestJWT_GetTokenExpiry(t *testing.T) {
	config := testJWTConfig()
	jwt := NewJWT(config)

	exp := jwt.GetTokenExpiry()
	if exp != config.AccessTokenExp {
		t.Errorf("GetTokenExpiry: got %v, want %v", exp, config.AccessTokenExp)
	}
}

func TestJWT_GetRefreshTokenExpiry(t *testing.T) {
	config := testJWTConfig()
	jwt := NewJWT(config)

	exp := jwt.GetRefreshTokenExpiry()
	if exp != config.RefreshTokenExp {
		t.Errorf("GetRefreshTokenExpiry: got %v, want %v", exp, config.RefreshTokenExp)
	}
}

func TestGenerateAPIKey_Prefix(t *testing.T) {
	fullKey, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}

	if !strings.HasPrefix(fullKey, "labkey_") {
		t.Errorf("full key should start with 'labkey_', got: %s", fullKey[:7])
	}

	if !strings.HasPrefix(prefix, "labkey_") {
		t.Errorf("prefix should start with 'labkey_', got: %s", prefix[:7])
	}
}

func TestGenerateAPIKey_PrefixLength(t *testing.T) {
	_, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}

	// prefix is "labkey_" + 8 chars = 15 chars
	if len(prefix) != 15 {
		t.Errorf("prefix length: got %d, want 15", len(prefix))
	}
}

func TestGenerateAPIKey_FullKeyContainsPrefix(t *testing.T) {
	fullKey, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}

	if !strings.HasPrefix(fullKey, prefix) {
		t.Errorf("full key should start with prefix. full=%s, prefix=%s", fullKey, prefix)
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	keys := make(map[string]bool)

	for i := 0; i < 100; i++ {
		fullKey, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey error: %v", err)
		}
		if keys[fullKey] {
			t.Errorf("duplicate API key generated: %s", fullKey)
		}
		keys[fullKey] = true
	}
}

func TestGenerateAPIKey_ReturnsBothValues(t *testing.T) {
	fullKey, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey error: %v", err)
	}

	if fullKey == "" {
		t.Error("full key should not be empty")
	}
	if prefix == "" {
		t.Error("prefix should not be empty")
	}
	if fullKey == prefix {
		t.Error("full key and prefix should be different")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// SecretKey should be nil - must be set explicitly
	if config.SecretKey != nil {
		t.Error("default SecretKey should be nil, must be set explicitly")
	}
	if config.AccessTokenExp != 15*time.Minute {
		t.Errorf("AccessTokenExp: got %v, want 15m", config.AccessTokenExp)
	}
	if config.RefreshTokenExp != 7*24*time.Hour {
		t.Errorf("RefreshTokenExp: got %v, want 168h", config.RefreshTokenExp)
	}
	if config.Issuer != "lab-api" {
		t.Errorf("Issuer: got %q, want lab-api", config.Issuer)
	}
}

func TestNewJWT(t *testing.T) {
	config := testJWTConfig()
	jwt := NewJWT(config)

	if jwt == nil {
		t.Fatal("NewJWT returned nil")
	}
	if jwt.config.Issuer != config.Issuer {
		t.Errorf("config not set correctly")
	}
}
