package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// RefreshToken represents a refresh token session
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"` // Never serialize
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshTokenRepository provides database operations for refresh tokens
type RefreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create stores a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, userID, token string, expiresAt time.Time) (*RefreshToken, error) {
	id := uuid.New().String()

	// Hash the token before storing
	tokenHash, err := bcrypt.GenerateFromPassword([]byte(token), 12)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query, id, userID, string(tokenHash), expiresAt.Unix(), time.Now().Unix())
	if err != nil {
		return nil, err
	}

	return &RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: string(tokenHash),
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}, nil
}

// GetByToken finds a refresh token by its raw token value
func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE revoked = FALSE AND expires_at > ?
	`

	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		rt := &RefreshToken{}
		var expiresAt, createdAt int64

		err := rows.Scan(
			&rt.ID,
			&rt.UserID,
			&rt.TokenHash,
			&expiresAt,
			&rt.Revoked,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Compare the token with the hash
		err = bcrypt.CompareHashAndPassword([]byte(rt.TokenHash), []byte(token))
		if err == nil {
			rt.ExpiresAt = time.Unix(expiresAt, 0)
			rt.CreatedAt = time.Unix(createdAt, 0)
			return rt, nil
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return nil, errors.New("invalid or expired refresh token")
}

// Revoke marks a refresh token as revoked
func (r *RefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("refresh token not found")
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = ?`

	_, err := r.db.ExecContext(ctx, query, userID)
	return err
}

// DeleteExpired removes expired refresh tokens from the database
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < ?`

	now := time.Now().Unix()
	result, err := r.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ListByUser retrieves all active refresh tokens for a user
func (r *RefreshTokenRepository) ListByUser(ctx context.Context, userID string) ([]*RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
		ORDER BY created_at DESC
	`

	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, query, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*RefreshToken
	for rows.Next() {
		rt := &RefreshToken{}
		var expiresAt, createdAt int64

		err := rows.Scan(
			&rt.ID,
			&rt.UserID,
			&rt.TokenHash,
			&expiresAt,
			&rt.Revoked,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		rt.ExpiresAt = time.Unix(expiresAt, 0)
		rt.CreatedAt = time.Unix(createdAt, 0)
		tokens = append(tokens, rt)
	}

	return tokens, rows.Err()
}

// CountByUser returns the number of active refresh tokens for a user
func (r *RefreshTokenRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM refresh_tokens 
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
	`

	now := time.Now().Unix()
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, now).Scan(&count)
	return count, err
}
