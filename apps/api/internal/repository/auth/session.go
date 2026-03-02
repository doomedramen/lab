package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session represents an active user session
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	JTI        string    `json:"jti"` // JWT ID claim
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	DeviceName string    `json:"device_name"`
	IssuedAt   time.Time `json:"issued_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Revoked    bool      `json:"revoked"`
}

// SessionRepository provides database operations for sessions
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(ctx context.Context, userID, jti, ipAddress, userAgent, deviceName string, expiresAt time.Time) (*Session, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO sessions (id, user_id, jti, ip_address, user_agent, device_name, issued_at, last_seen_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		id,
		userID,
		jti,
		ipAddress,
		userAgent,
		deviceName,
		now.Unix(),
		now.Unix(),
		expiresAt.Unix(),
	)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:         id,
		UserID:     userID,
		JTI:        jti,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		DeviceName: deviceName,
		IssuedAt:   now,
		LastSeenAt: now,
		ExpiresAt:  expiresAt,
		Revoked:    false,
	}, nil
}

// GetByJTI retrieves a session by JWT ID
func (r *SessionRepository) GetByJTI(ctx context.Context, jti string) (*Session, error) {
	query := `
		SELECT id, user_id, jti, ip_address, user_agent, device_name, issued_at, last_seen_at, expires_at, revoked
		FROM sessions
		WHERE jti = ?
	`

	session := &Session{}
	var issuedAt, lastSeenAt, expiresAt int64

	err := r.db.QueryRowContext(ctx, query, jti).Scan(
		&session.ID,
		&session.UserID,
		&session.JTI,
		&session.IPAddress,
		&session.UserAgent,
		&session.DeviceName,
		&issuedAt,
		&lastSeenAt,
		&expiresAt,
		&session.Revoked,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	session.IssuedAt = time.Unix(issuedAt, 0)
	session.LastSeenAt = time.Unix(lastSeenAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)

	return session, nil
}

// GetByID retrieves a session by ID
func (r *SessionRepository) GetByID(ctx context.Context, id string) (*Session, error) {
	query := `
		SELECT id, user_id, jti, ip_address, user_agent, device_name, issued_at, last_seen_at, expires_at, revoked
		FROM sessions
		WHERE id = ?
	`

	session := &Session{}
	var issuedAt, lastSeenAt, expiresAt int64

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.JTI,
		&session.IPAddress,
		&session.UserAgent,
		&session.DeviceName,
		&issuedAt,
		&lastSeenAt,
		&expiresAt,
		&session.Revoked,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	session.IssuedAt = time.Unix(issuedAt, 0)
	session.LastSeenAt = time.Unix(lastSeenAt, 0)
	session.ExpiresAt = time.Unix(expiresAt, 0)

	return session, nil
}

// UpdateLastSeen updates the last_seen_at timestamp for a session
func (r *SessionRepository) UpdateLastSeen(ctx context.Context, jti string) error {
	query := `UPDATE sessions SET last_seen_at = ? WHERE jti = ?`
	_, err := r.db.ExecContext(ctx, query, time.Now().Unix(), jti)
	return err
}

// Revoke marks a session as revoked
func (r *SessionRepository) Revoke(ctx context.Context, id string) error {
	query := `UPDATE sessions SET revoked = TRUE WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("session not found")
	}

	return nil
}

// RevokeByJTI marks a session as revoked by JTI
func (r *SessionRepository) RevokeByJTI(ctx context.Context, jti string) error {
	query := `UPDATE sessions SET revoked = TRUE WHERE jti = ?`

	result, err := r.db.ExecContext(ctx, query, jti)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("session not found")
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user except the current one
func (r *SessionRepository) RevokeAllUserSessions(ctx context.Context, userID string, excludeJTI string) error {
	query := `UPDATE sessions SET revoked = TRUE WHERE user_id = ? AND jti != ?`
	_, err := r.db.ExecContext(ctx, query, userID, excludeJTI)
	return err
}

// ListByUser retrieves all active sessions for a user
func (r *SessionRepository) ListByUser(ctx context.Context, userID string) ([]*Session, error) {
	query := `
		SELECT id, user_id, jti, ip_address, user_agent, device_name, issued_at, last_seen_at, expires_at, revoked
		FROM sessions
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
		ORDER BY last_seen_at DESC
	`

	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, query, userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		var issuedAt, lastSeenAt, expiresAt int64

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.JTI,
			&session.IPAddress,
			&session.UserAgent,
			&session.DeviceName,
			&issuedAt,
			&lastSeenAt,
			&expiresAt,
			&session.Revoked,
		)
		if err != nil {
			return nil, err
		}

		session.IssuedAt = time.Unix(issuedAt, 0)
		session.LastSeenAt = time.Unix(lastSeenAt, 0)
		session.ExpiresAt = time.Unix(expiresAt, 0)
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// IsRevoked checks if a session is revoked
func (r *SessionRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	query := `
		SELECT revoked FROM sessions
		WHERE jti = ? AND expires_at > ?
	`

	var revoked bool
	err := r.db.QueryRowContext(ctx, query, jti, time.Now().Unix()).Scan(&revoked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Session not found in DB - could be old token format
			// We'll allow it for backwards compatibility
			return false, nil
		}
		return false, err
	}

	return revoked, nil
}

// DeleteExpired removes expired sessions from the database
func (r *SessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < ?`

	now := time.Now().Unix()
	result, err := r.db.ExecContext(ctx, query, now)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// CountByUser returns the number of active sessions for a user
func (r *SessionRepository) CountByUser(ctx context.Context, userID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM sessions
		WHERE user_id = ? AND revoked = FALSE AND expires_at > ?
	`

	now := time.Now().Unix()
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, now).Scan(&count)
	return count, err
}
