package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// APIKey represents an API key for automation/CLI access
type APIKey struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"-"` // Never serialize
	Prefix      string     `json:"prefix"`
	Permissions []string   `json:"permissions"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// APIKeyRepository provides database operations for API keys
type APIKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create stores a new API key
func (r *APIKeyRepository) Create(ctx context.Context, userID, name, key, prefix string, permissions []string, expiresAt *time.Time) (*APIKey, error) {
	id := uuid.New().String()
	now := time.Now()

	// Hash the API key
	keyHash, err := bcrypt.GenerateFromPassword([]byte(key), 12)
	if err != nil {
		return nil, err
	}

	// Serialize permissions to JSON
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO api_keys (id, user_id, name, key_hash, prefix, permissions, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	var expiresAtUnix interface{}
	if expiresAt != nil {
		expiresAtUnix = expiresAt.Unix()
	} else {
		expiresAtUnix = nil
	}

	_, err = r.db.ExecContext(ctx, query, id, userID, name, string(keyHash), prefix, string(permissionsJSON), expiresAtUnix, now.Unix())
	if err != nil {
		return nil, err
	}

	apiKey := &APIKey{
		ID:          id,
		UserID:      userID,
		Name:        name,
		KeyHash:     string(keyHash),
		Prefix:      prefix,
		Permissions: permissions,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	return apiKey, nil
}

// GetByKey finds an API key by its raw key value
func (r *APIKeyRepository) GetByKey(ctx context.Context, key string) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, prefix, permissions, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE (expires_at IS NULL OR expires_at > ?)
	`

	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ak := &APIKey{}
		var lastUsedAt, expiresAt, createdAt sql.NullInt64
		var permissionsJSON sql.NullString

		err := rows.Scan(
			&ak.ID,
			&ak.UserID,
			&ak.Name,
			&ak.KeyHash,
			&ak.Prefix,
			&permissionsJSON,
			&lastUsedAt,
			&expiresAt,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Compare the key with the hash
		err = bcrypt.CompareHashAndPassword([]byte(ak.KeyHash), []byte(key))
		if err == nil {
			// Parse permissions
			if permissionsJSON.Valid {
				if err := json.Unmarshal([]byte(permissionsJSON.String), &ak.Permissions); err != nil {
					return nil, err
				}
			} else {
				ak.Permissions = []string{}
			}

			// Parse timestamps
			if lastUsedAt.Valid {
				t := time.Unix(lastUsedAt.Int64, 0)
				ak.LastUsedAt = &t
			}
			if expiresAt.Valid {
				t := time.Unix(expiresAt.Int64, 0)
				ak.ExpiresAt = &t
			}
			if createdAt.Valid {
				ak.CreatedAt = time.Unix(createdAt.Int64, 0)
			}

			return ak, nil
		}
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return nil, errors.New("invalid or expired API key")
}

// GetByID retrieves an API key by ID
func (r *APIKeyRepository) GetByID(ctx context.Context, id string) (*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, prefix, permissions, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE id = ?
	`

	ak := &APIKey{}
	var lastUsedAt, expiresAt, createdAt sql.NullInt64
	var permissionsJSON sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&ak.ID,
		&ak.UserID,
		&ak.Name,
		&ak.KeyHash,
		&ak.Prefix,
		&permissionsJSON,
		&lastUsedAt,
		&expiresAt,
		&createdAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("API key not found")
		}
		return nil, err
	}

	// Parse permissions
	if permissionsJSON.Valid {
		if err := json.Unmarshal([]byte(permissionsJSON.String), &ak.Permissions); err != nil {
			return nil, err
		}
	} else {
		ak.Permissions = []string{}
	}

	// Parse timestamps
	if lastUsedAt.Valid {
		t := time.Unix(lastUsedAt.Int64, 0)
		ak.LastUsedAt = &t
	}
	if expiresAt.Valid {
		t := time.Unix(expiresAt.Int64, 0)
		ak.ExpiresAt = &t
	}
	if createdAt.Valid {
		ak.CreatedAt = time.Unix(createdAt.Int64, 0)
	}

	return ak, nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at = ? WHERE id = ?`
	now := time.Now().Unix()

	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

// Revoke deletes an API key
func (r *APIKeyRepository) Revoke(ctx context.Context, id string) error {
	query := `DELETE FROM api_keys WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("API key not found")
	}

	return nil
}

// ListByUser retrieves all API keys for a user
func (r *APIKeyRepository) ListByUser(ctx context.Context, userID string) ([]*APIKey, error) {
	query := `
		SELECT id, user_id, name, key_hash, prefix, permissions, last_used_at, expires_at, created_at
		FROM api_keys
		WHERE user_id = ? AND (expires_at IS NULL OR expires_at > ?)
		ORDER BY created_at DESC
	`

	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, query, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*APIKey
	for rows.Next() {
		ak := &APIKey{}
		var lastUsedAt, expiresAt, createdAt sql.NullInt64
		var permissionsJSON sql.NullString

		err := rows.Scan(
			&ak.ID,
			&ak.UserID,
			&ak.Name,
			&ak.KeyHash,
			&ak.Prefix,
			&permissionsJSON,
			&lastUsedAt,
			&expiresAt,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse permissions
		if permissionsJSON.Valid {
			if err := json.Unmarshal([]byte(permissionsJSON.String), &ak.Permissions); err != nil {
				return nil, err
			}
		} else {
			ak.Permissions = []string{}
		}

		// Parse timestamps
		if lastUsedAt.Valid {
			t := time.Unix(lastUsedAt.Int64, 0)
			ak.LastUsedAt = &t
		}
		if expiresAt.Valid {
			t := time.Unix(expiresAt.Int64, 0)
			ak.ExpiresAt = &t
		}
		if createdAt.Valid {
			ak.CreatedAt = time.Unix(createdAt.Int64, 0)
		}

		keys = append(keys, ak)
	}

	return keys, rows.Err()
}

// HasPermission checks if an API key has a specific permission
func (r *APIKeyRepository) HasPermission(ctx context.Context, id, permission string) (bool, error) {
	ak, err := r.GetByID(ctx, id)
	if err != nil {
		return false, err
	}

	for _, p := range ak.Permissions {
		if p == permission || p == "*" {
			return true, nil
		}
	}

	return false, nil
}

// CountByUser returns the number of active API keys for a user
func (r *APIKeyRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM api_keys 
		WHERE user_id = ? AND (expires_at IS NULL OR expires_at > ?)
	`

	now := time.Now().Unix()
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, now).Scan(&count)
	return count, err
}
