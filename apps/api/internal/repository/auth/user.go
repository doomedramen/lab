package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Role represents user roles
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

// User represents a user in the system
type User struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"` // Never serialize
	Role          Role       `json:"role"`
	MFASecret     *string    `json:"-"` // Sensitive
	MFAEnabled    bool       `json:"mfa_enabled"`
	MFABackupCodes *string   `json:"-"` // Sensitive
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	IsActive      bool       `json:"is_active"`
}

// UserRepository provides database operations for users
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, email, passwordHash string, role Role) (*User, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, id, email, passwordHash, role, now.Unix(), now.Unix())
	if err != nil {
		// Check for unique constraint violation
		if isDuplicateEmail(err) {
			return nil, errors.New("email already exists")
		}
		return nil, err
	}

	return &User{
		ID:        id,
		Email:     email,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, mfa_secret, mfa_enabled, 
		       mfa_backup_codes, created_at, updated_at, last_login_at, is_active
		FROM users
		WHERE id = ? AND is_active = TRUE
	`

	user := &User{}
	var lastLoginAt sql.NullInt64
	var mfaSecret sql.NullString
	var mfaBackupCodes sql.NullString
	var createdAt, updatedAt int64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&mfaSecret,
		&user.MFAEnabled,
		&mfaBackupCodes,
		&createdAt,
		&updatedAt,
		&lastLoginAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if mfaSecret.Valid {
		user.MFASecret = &mfaSecret.String
	}
	if mfaBackupCodes.Valid {
		user.MFABackupCodes = &mfaBackupCodes.String
	}
	if lastLoginAt.Valid {
		t := time.Unix(lastLoginAt.Int64, 0)
		user.LastLoginAt = &t
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, mfa_secret, mfa_enabled, 
		       mfa_backup_codes, created_at, updated_at, last_login_at, is_active
		FROM users
		WHERE email = ? AND is_active = TRUE
	`

	user := &User{}
	var lastLoginAt sql.NullInt64
	var mfaSecret sql.NullString
	var mfaBackupCodes sql.NullString
	var createdAt, updatedAt int64

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&mfaSecret,
		&user.MFAEnabled,
		&mfaBackupCodes,
		&createdAt,
		&updatedAt,
		&lastLoginAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if mfaSecret.Valid {
		user.MFASecret = &mfaSecret.String
	}
	if mfaBackupCodes.Valid {
		user.MFABackupCodes = &mfaBackupCodes.String
	}
	if lastLoginAt.Valid {
		t := time.Unix(lastLoginAt.Int64, 0)
		user.LastLoginAt = &t
	}

	return user, nil
}

// Update updates a user's information
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users 
		SET email = ?, role = ?, mfa_secret = ?, mfa_enabled = ?, 
		    mfa_backup_codes = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now().Unix()

	var mfaSecret, mfaBackupCodes interface{}
	if user.MFASecret != nil {
		mfaSecret = *user.MFASecret
	} else {
		mfaSecret = sql.NullString{}
	}
	if user.MFABackupCodes != nil {
		mfaBackupCodes = *user.MFABackupCodes
	} else {
		mfaBackupCodes = sql.NullString{}
	}

	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Role,
		mfaSecret,
		user.MFAEnabled,
		mfaBackupCodes,
		now,
		user.ID,
	)

	if err != nil {
		if isDuplicateEmail(err) {
			return errors.New("email already exists")
		}
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("user not found")
	}

	user.UpdatedAt = time.Now()
	return nil
}

// UpdatePassword updates the hashed password for a user.
func (r *UserRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, passwordHash, time.Now().Unix(), id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

// UpdateEmail updates the email address for a user.
func (r *UserRepository) UpdateEmail(ctx context.Context, id, email string) error {
	query := `UPDATE users SET email = ?, updated_at = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, email, time.Now().Unix(), id)
	if err != nil {
		if isDuplicateEmail(err) {
			return errors.New("email already exists")
		}
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("user not found")
	}
	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, id string) error {
	query := `UPDATE users SET last_login_at = ? WHERE id = ?`
	now := time.Now().Unix()

	_, err := r.db.ExecContext(ctx, query, now, id)
	return err
}

// SetMFAEnabled enables or disables MFA for a user
func (r *UserRepository) SetMFAEnabled(ctx context.Context, id string, enabled bool, secret *string, backupCodes *string) error {
	query := `
		UPDATE users 
		SET mfa_enabled = ?, mfa_secret = ?, mfa_backup_codes = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now().Unix()

	var mfaSecret, mfaBackupCodes interface{}
	if secret != nil {
		mfaSecret = *secret
	} else {
		mfaSecret = sql.NullString{}
	}
	if backupCodes != nil {
		mfaBackupCodes = *backupCodes
	} else {
		mfaBackupCodes = sql.NullString{}
	}

	_, err := r.db.ExecContext(ctx, query, enabled, mfaSecret, mfaBackupCodes, now, id)
	return err
}

// Delete soft-deletes a user by setting is_active to false
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `UPDATE users SET is_active = FALSE, updated_at = ? WHERE id = ?`
	now := time.Now().Unix()

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("user not found")
	}

	return nil
}

// List retrieves all active users
func (r *UserRepository) List(ctx context.Context) ([]*User, error) {
	query := `
		SELECT id, email, password_hash, role, mfa_secret, mfa_enabled, 
		       mfa_backup_codes, created_at, updated_at, last_login_at, is_active
		FROM users
		WHERE is_active = TRUE
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var lastLoginAt sql.NullInt64
		var mfaSecret sql.NullString
		var mfaBackupCodes sql.NullString
		var createdAt, updatedAt int64

		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&mfaSecret,
			&user.MFAEnabled,
			&mfaBackupCodes,
			&createdAt,
			&updatedAt,
			&lastLoginAt,
			&user.IsActive,
		)

		if err != nil {
			return nil, err
		}

		user.CreatedAt = time.Unix(createdAt, 0)
		user.UpdatedAt = time.Unix(updatedAt, 0)
		if mfaSecret.Valid {
			user.MFASecret = &mfaSecret.String
		}
		if mfaBackupCodes.Valid {
			user.MFABackupCodes = &mfaBackupCodes.String
		}
		if lastLoginAt.Valid {
			t := time.Unix(lastLoginAt.Int64, 0)
			user.LastLoginAt = &t
		}

		users = append(users, user)
	}

	return users, rows.Err()
}

// Count returns the number of active users
func (r *UserRepository) Count(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM users WHERE is_active = TRUE`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// GetFirstAdmin returns the first admin user (for initial setup)
func (r *UserRepository) GetFirstAdmin(ctx context.Context) (*User, error) {
	query := `
		SELECT id, email, password_hash, role, mfa_secret, mfa_enabled, 
		       mfa_backup_codes, created_at, updated_at, last_login_at, is_active
		FROM users
		WHERE role = 'admin' AND is_active = TRUE
		ORDER BY created_at ASC
		LIMIT 1
	`

	user := &User{}
	var lastLoginAt sql.NullInt64
	var mfaSecret sql.NullString
	var mfaBackupCodes sql.NullString
	var createdAt, updatedAt int64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&mfaSecret,
		&user.MFAEnabled,
		&mfaBackupCodes,
		&createdAt,
		&updatedAt,
		&lastLoginAt,
		&user.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("no admin user found")
		}
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if mfaSecret.Valid {
		user.MFASecret = &mfaSecret.String
	}
	if mfaBackupCodes.Valid {
		user.MFABackupCodes = &mfaBackupCodes.String
	}
	if lastLoginAt.Valid {
		t := time.Unix(lastLoginAt.Int64, 0)
		user.LastLoginAt = &t
	}

	return user, nil
}

// isDuplicateEmail checks if the error is due to a duplicate email
func isDuplicateEmail(err error) bool {
	// SQLite error messages for unique constraint violations
	errStr := err.Error()
	return contains(errStr, "UNIQUE constraint failed") && contains(errStr, "email")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
