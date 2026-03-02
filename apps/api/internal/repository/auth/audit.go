package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// AuditLogStatus represents the status of an audit log entry
type AuditLogStatus string

const (
	StatusSuccess AuditLogStatus = "success"
	StatusFailure AuditLogStatus = "failure"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID           int64          `json:"id"`
	UserID       string         `json:"user_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Details      map[string]any `json:"details"`
	IPAddress    string         `json:"ip_address"`
	UserAgent    string         `json:"user_agent"`
	Status       AuditLogStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
}

// AuditLogRepository provides database operations for audit logs
type AuditLogRepository struct {
	db *sql.DB
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(ctx context.Context, log *AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, action, resource_type, resource_id, details, ip_address, user_agent, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Serialize details to JSON
	var detailsJSON interface{}
	if log.Details != nil {
		data, err := json.Marshal(log.Details)
		if err != nil {
			return err
		}
		detailsJSON = string(data)
	}

	_, err := r.db.ExecContext(ctx, query,
		log.UserID,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		detailsJSON,
		log.IPAddress,
		log.UserAgent,
		log.Status,
		time.Now().Unix(),
	)

	return err
}

// LogLogin logs a login attempt
func (r *AuditLogRepository) LogLogin(ctx context.Context, userID, email, ipAddress, userAgent string, success bool) error {
	status := StatusSuccess
	if !success {
		status = StatusFailure
	}

	log := &AuditLog{
		UserID:    userID,
		Action:    "user.login",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Status:    status,
		Details: map[string]any{
			"email": email,
		},
	}

	return r.Create(ctx, log)
}

// LogLogout logs a logout event
func (r *AuditLogRepository) LogLogout(ctx context.Context, userID, ipAddress, userAgent string) error {
	log := &AuditLog{
		UserID:    userID,
		Action:    "user.logout",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Status:    StatusSuccess,
	}

	return r.Create(ctx, log)
}

// LogAPIKeyCreate logs API key creation
func (r *AuditLogRepository) LogAPIKeyCreate(ctx context.Context, userID, keyID, keyName, ipAddress, userAgent string) error {
	log := &AuditLog{
		UserID:       userID,
		Action:       "api_key.create",
		ResourceType: "api_key",
		ResourceID:   keyID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       StatusSuccess,
		Details: map[string]any{
			"key_name": keyName,
		},
	}

	return r.Create(ctx, log)
}

// LogAPIKeyUse logs API key usage
func (r *AuditLogRepository) LogAPIKeyUse(ctx context.Context, userID, keyID, ipAddress, userAgent string) error {
	log := &AuditLog{
		UserID:       userID,
		Action:       "api_key.use",
		ResourceType: "api_key",
		ResourceID:   keyID,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       StatusSuccess,
	}

	return r.Create(ctx, log)
}

// LogResourceAction logs a resource action (e.g., VM start/stop)
func (r *AuditLogRepository) LogResourceAction(ctx context.Context, userID, action, resourceType, resourceID, ipAddress, userAgent string, details map[string]any, success bool) error {
	status := StatusSuccess
	if !success {
		status = StatusFailure
	}

	log := &AuditLog{
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		Status:       status,
	}

	return r.Create(ctx, log)
}

// List retrieves audit logs with pagination
func (r *AuditLogRepository) List(ctx context.Context, limit, offset int) ([]*AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, status, created_at
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var detailsJSON sql.NullString
		var createdAt int64

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.Status,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse details JSON
		if detailsJSON.Valid {
			if err := json.Unmarshal([]byte(detailsJSON.String), &log.Details); err != nil {
				return nil, err
			}
		} else {
			log.Details = make(map[string]any)
		}

		log.CreatedAt = time.Unix(createdAt, 0)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ListByUser retrieves audit logs for a specific user
func (r *AuditLogRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, status, created_at
		FROM audit_logs
		WHERE user_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var detailsJSON sql.NullString
		var createdAt int64

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.Status,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse details JSON
		if detailsJSON.Valid {
			if err := json.Unmarshal([]byte(detailsJSON.String), &log.Details); err != nil {
				return nil, err
			}
		} else {
			log.Details = make(map[string]any)
		}

		log.CreatedAt = time.Unix(createdAt, 0)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ListByAction retrieves audit logs for a specific action
func (r *AuditLogRepository) ListByAction(ctx context.Context, action string, limit, offset int) ([]*AuditLog, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id, details, ip_address, user_agent, status, created_at
		FROM audit_logs
		WHERE action = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := r.db.QueryContext(ctx, query, action, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var detailsJSON sql.NullString
		var createdAt int64

		err := rows.Scan(
			&log.ID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&log.ResourceID,
			&detailsJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.Status,
			&createdAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse details JSON
		if detailsJSON.Valid {
			if err := json.Unmarshal([]byte(detailsJSON.String), &log.Details); err != nil {
				return nil, err
			}
		} else {
			log.Details = make(map[string]any)
		}

		log.CreatedAt = time.Unix(createdAt, 0)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// DeleteOld removes audit logs older than the specified date
func (r *AuditLogRepository) DeleteOld(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE created_at < ?`

	result, err := r.db.ExecContext(ctx, query, before.Unix())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Count returns the total number of audit logs
func (r *AuditLogRepository) Count(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs`

	var count int64
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}
