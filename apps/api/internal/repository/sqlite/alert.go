package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// AlertRepository handles alert storage and retrieval
type AlertRepository struct {
	db *sqlitePkg.DB
}

// NewAlertRepository creates a new alert repository
func NewAlertRepository(db *sqlitePkg.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

// --- Notification Channels ---

// CreateChannel creates a new notification channel
func (r *AlertRepository) CreateChannel(ctx context.Context, channel *model.NotificationChannel) error {
	configJSON, err := json.Marshal(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal channel config: %w", err)
	}

	query := `
		INSERT INTO notification_channels (id, name, type, config, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		channel.ID,
		channel.Name,
		string(channel.Type),
		string(configJSON),
		channel.Enabled,
		channel.CreatedAt.Format(time.RFC3339),
		channel.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create notification channel: %w", err)
	}

	return nil
}

// GetChannelByID retrieves a notification channel by ID
func (r *AlertRepository) GetChannelByID(ctx context.Context, id string) (*model.NotificationChannel, error) {
	query := `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM notification_channels
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanChannel(row)
}

// ListChannels retrieves all notification channels
func (r *AlertRepository) ListChannels(ctx context.Context) ([]*model.NotificationChannel, error) {
	query := `
		SELECT id, name, type, config, enabled, created_at, updated_at
		FROM notification_channels
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []*model.NotificationChannel
	for rows.Next() {
		channel, err := scanChannelRow(rows)
		if err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating channels: %w", err)
	}

	return channels, nil
}

// UpdateChannel updates a notification channel
func (r *AlertRepository) UpdateChannel(ctx context.Context, channel *model.NotificationChannel) error {
	configJSON, err := json.Marshal(channel.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal channel config: %w", err)
	}

	query := `
		UPDATE notification_channels
		SET name = ?, type = ?, config = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, query,
		channel.Name,
		string(channel.Type),
		string(configJSON),
		channel.Enabled,
		channel.UpdatedAt.Format(time.RFC3339),
		channel.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update notification channel: %w", err)
	}

	return nil
}

// DeleteChannel deletes a notification channel
func (r *AlertRepository) DeleteChannel(ctx context.Context, id string) error {
	query := `DELETE FROM notification_channels WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	return nil
}

// --- Alert Rules ---

// CreateRule creates a new alert rule
func (r *AlertRepository) CreateRule(ctx context.Context, rule *model.AlertRule) error {
	query := `
		INSERT INTO alert_rules (
			id, name, description, type, threshold, duration_minutes,
			entity_type, entity_id, channel_id, enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var threshold sql.NullFloat64
	if rule.Threshold != nil {
		threshold = sql.NullFloat64{Float64: *rule.Threshold, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		string(rule.Type),
		threshold,
		rule.DurationMinutes,
		rule.EntityType,
		rule.EntityID,
		rule.ChannelID,
		rule.Enabled,
		rule.CreatedAt.Format(time.RFC3339),
		rule.UpdatedAt.Format(time.RFC3339),
	)

	if err != nil {
		return fmt.Errorf("failed to create alert rule: %w", err)
	}

	return nil
}

// GetRuleByID retrieves an alert rule by ID
func (r *AlertRepository) GetRuleByID(ctx context.Context, id string) (*model.AlertRule, error) {
	query := `
		SELECT id, name, description, type, threshold, duration_minutes,
		       entity_type, entity_id, channel_id, enabled, last_triggered_at, created_at, updated_at
		FROM alert_rules
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanRule(row)
}

// ListRules retrieves alert rules with optional filters
func (r *AlertRepository) ListRules(ctx context.Context, enabledOnly bool) ([]*model.AlertRule, error) {
	query := `
		SELECT id, name, description, type, threshold, duration_minutes,
		       entity_type, entity_id, channel_id, enabled, last_triggered_at, created_at, updated_at
		FROM alert_rules
		WHERE 1=1
	`

	args := []interface{}{}

	if enabledOnly {
		query += " AND enabled = 1"
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list alert rules: %w", err)
	}
	defer rows.Close()

	var rules []*model.AlertRule
	for rows.Next() {
		rule, err := scanRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

// UpdateRule updates an alert rule
func (r *AlertRepository) UpdateRule(ctx context.Context, rule *model.AlertRule) error {
	query := `
		UPDATE alert_rules
		SET name = ?, description = ?, type = ?, threshold = ?, duration_minutes = ?,
		    entity_type = ?, entity_id = ?, channel_id = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`

	var threshold sql.NullFloat64
	if rule.Threshold != nil {
		threshold = sql.NullFloat64{Float64: *rule.Threshold, Valid: true}
	}

	_, err := r.db.ExecContext(ctx, query,
		rule.Name,
		rule.Description,
		string(rule.Type),
		threshold,
		rule.DurationMinutes,
		rule.EntityType,
		rule.EntityID,
		rule.ChannelID,
		rule.Enabled,
		rule.UpdatedAt.Format(time.RFC3339),
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update alert rule: %w", err)
	}

	return nil
}

// UpdateRuleLastTriggered updates the last_triggered_at timestamp
func (r *AlertRepository) UpdateRuleLastTriggered(ctx context.Context, id string) error {
	query := `UPDATE alert_rules SET last_triggered_at = datetime('now'), updated_at = datetime('now') WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update rule last triggered: %w", err)
	}

	return nil
}

// DeleteRule deletes an alert rule
func (r *AlertRepository) DeleteRule(ctx context.Context, id string) error {
	query := `DELETE FROM alert_rules WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete alert rule: %w", err)
	}

	return nil
}

// --- Fired Alerts ---

// CreateAlert creates a new fired alert
func (r *AlertRepository) CreateAlert(ctx context.Context, alert *model.Alert) error {
	metadataJSON, err := json.Marshal(alert.Metadata)
	if err != nil {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO fired_alerts (
			id, rule_id, rule_name, entity_type, entity_id, entity_name,
			message, severity, status, fired_at, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		alert.ID,
		alert.RuleID,
		alert.RuleName,
		alert.EntityType,
		alert.EntityID,
		alert.EntityName,
		alert.Message,
		string(alert.Severity),
		string(alert.Status),
		alert.FiredAt.Format(time.RFC3339),
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to create fired alert: %w", err)
	}

	return nil
}

// GetAlertByID retrieves a fired alert by ID
func (r *AlertRepository) GetAlertByID(ctx context.Context, id string) (*model.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, entity_type, entity_id, entity_name,
		       message, severity, status, fired_at, acknowledged_at, acknowledged_by,
		       resolved_at, metadata
		FROM fired_alerts
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanAlert(row)
}

// ListAlerts retrieves fired alerts with optional filters
func (r *AlertRepository) ListAlerts(ctx context.Context, filter model.AlertFilter) ([]*model.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, entity_type, entity_id, entity_name,
		       message, severity, status, fired_at, acknowledged_at, acknowledged_by,
		       resolved_at, metadata
		FROM fired_alerts
		WHERE 1=1
	`

	args := []interface{}{}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if filter.Severity != "" {
		query += " AND severity = ?"
		args = append(args, string(filter.Severity))
	}
	if filter.RuleID != "" {
		query += " AND rule_id = ?"
		args = append(args, filter.RuleID)
	}
	if filter.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, filter.EntityType)
	}
	if filter.EntityID != "" {
		query += " AND entity_id = ?"
		args = append(args, filter.EntityID)
	}
	if filter.OpenOnly {
		query += " AND status = 'open'"
	}

	query += " ORDER BY fired_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list fired alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*model.Alert
	for rows.Next() {
		alert, err := scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating alerts: %w", err)
	}

	return alerts, nil
}

// AcknowledgeAlert acknowledges a fired alert
func (r *AlertRepository) AcknowledgeAlert(ctx context.Context, id string, acknowledgedBy string) error {
	query := `
		UPDATE fired_alerts
		SET status = 'acknowledged', acknowledged_at = datetime('now'), acknowledged_by = ?
		WHERE id = ? AND status = 'open'
	`

	_, err := r.db.ExecContext(ctx, query, acknowledgedBy, id)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return nil
}

// ResolveAlert resolves a fired alert
func (r *AlertRepository) ResolveAlert(ctx context.Context, id string) error {
	query := `
		UPDATE fired_alerts
		SET status = 'resolved', resolved_at = datetime('now')
		WHERE id = ? AND status IN ('open', 'acknowledged')
	`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	return nil
}

// HasOpenAlert checks if there's an open alert for a rule/entity combination
func (r *AlertRepository) HasOpenAlert(ctx context.Context, ruleID string, entityType string, entityID string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM fired_alerts
		WHERE rule_id = ? AND entity_type = ? AND entity_id = ? AND status = 'open'
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, ruleID, entityType, entityID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check for open alert: %w", err)
	}

	return count > 0, nil
}

// DeleteOldAlerts deletes resolved alerts older than the specified duration
func (r *AlertRepository) DeleteOldAlerts(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Format(time.RFC3339)
	query := `
		DELETE FROM fired_alerts
		WHERE status = 'resolved'
		  AND datetime(resolved_at) < datetime(?)
	`

	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old alerts: %w", err)
	}

	return result.RowsAffected()
}

// --- Scanners ---

func scanChannel(row scanner) (*model.NotificationChannel, error) {
	var c model.NotificationChannel
	var configJSON string
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&c.ID,
		&c.Name,
		&c.Type,
		&configJSON,
		&c.Enabled,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan channel: %w", err)
	}

	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if err := json.Unmarshal([]byte(configJSON), &c.Config); err != nil {
		c.Config = make(map[string]string)
	}

	return &c, nil
}

func scanChannelRow(rows rowsScanner) (*model.NotificationChannel, error) {
	var c model.NotificationChannel
	var configJSON string
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&c.ID,
		&c.Name,
		&c.Type,
		&configJSON,
		&c.Enabled,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan channel row: %w", err)
	}

	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	if err := json.Unmarshal([]byte(configJSON), &c.Config); err != nil {
		c.Config = make(map[string]string)
	}

	return &c, nil
}

func scanRule(row scanner) (*model.AlertRule, error) {
	var r model.AlertRule
	var threshold sql.NullFloat64
	var description, entityType, entityID, lastTriggered sql.NullString
	var createdAtStr, updatedAtStr string

	err := row.Scan(
		&r.ID,
		&r.Name,
		&description,
		&r.Type,
		&threshold,
		&r.DurationMinutes,
		&entityType,
		&entityID,
		&r.ChannelID,
		&r.Enabled,
		&lastTriggered,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan rule: %w", err)
	}

	r.Description = description.String
	r.EntityType = entityType.String
	r.EntityID = entityID.String
	if threshold.Valid {
		r.Threshold = &threshold.Float64
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if lastTriggered.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, lastTriggered.String); err == nil {
			r.LastTriggeredAt = &parsedTime
		}
	}

	return &r, nil
}

func scanRuleRow(rows rowsScanner) (*model.AlertRule, error) {
	var r model.AlertRule
	var threshold sql.NullFloat64
	var description, entityType, entityID, lastTriggered sql.NullString
	var createdAtStr, updatedAtStr string

	err := rows.Scan(
		&r.ID,
		&r.Name,
		&description,
		&r.Type,
		&threshold,
		&r.DurationMinutes,
		&entityType,
		&entityID,
		&r.ChannelID,
		&r.Enabled,
		&lastTriggered,
		&createdAtStr,
		&updatedAtStr,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan rule row: %w", err)
	}

	r.Description = description.String
	r.EntityType = entityType.String
	r.EntityID = entityID.String
	if threshold.Valid {
		r.Threshold = &threshold.Float64
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if lastTriggered.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, lastTriggered.String); err == nil {
			r.LastTriggeredAt = &parsedTime
		}
	}

	return &r, nil
}

func scanAlert(row scanner) (*model.Alert, error) {
	var a model.Alert
	var metadataJSON string
	var entityType, entityID, entityName, acknowledgedAt, acknowledgedBy, resolvedAt sql.NullString
	var firedAtStr string

	err := row.Scan(
		&a.ID,
		&a.RuleID,
		&a.RuleName,
		&entityType,
		&entityID,
		&entityName,
		&a.Message,
		&a.Severity,
		&a.Status,
		&firedAtStr,
		&acknowledgedAt,
		&acknowledgedBy,
		&resolvedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan alert: %w", err)
	}

	a.EntityType = entityType.String
	a.EntityID = entityID.String
	a.EntityName = entityName.String
	a.AcknowledgedBy = acknowledgedBy.String
	a.FiredAt, _ = time.Parse(time.RFC3339, firedAtStr)
	if acknowledgedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, acknowledgedAt.String); err == nil {
			a.AcknowledgedAt = &parsedTime
		}
	}
	if resolvedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, resolvedAt.String); err == nil {
			a.ResolvedAt = &parsedTime
		}
	}
	if err := json.Unmarshal([]byte(metadataJSON), &a.Metadata); err != nil {
		a.Metadata = make(map[string]string)
	}

	return &a, nil
}

func scanAlertRow(rows rowsScanner) (*model.Alert, error) {
	var a model.Alert
	var metadataJSON string
	var entityType, entityID, entityName, acknowledgedAt, acknowledgedBy, resolvedAt sql.NullString
	var firedAtStr string

	err := rows.Scan(
		&a.ID,
		&a.RuleID,
		&a.RuleName,
		&entityType,
		&entityID,
		&entityName,
		&a.Message,
		&a.Severity,
		&a.Status,
		&firedAtStr,
		&acknowledgedAt,
		&acknowledgedBy,
		&resolvedAt,
		&metadataJSON,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan alert row: %w", err)
	}

	a.EntityType = entityType.String
	a.EntityID = entityID.String
	a.EntityName = entityName.String
	a.AcknowledgedBy = acknowledgedBy.String
	a.FiredAt, _ = time.Parse(time.RFC3339, firedAtStr)
	if acknowledgedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, acknowledgedAt.String); err == nil {
			a.AcknowledgedAt = &parsedTime
		}
	}
	if resolvedAt.Valid {
		if parsedTime, err := time.Parse(time.RFC3339, resolvedAt.String); err == nil {
			a.ResolvedAt = &parsedTime
		}
	}
	if err := json.Unmarshal([]byte(metadataJSON), &a.Metadata); err != nil {
		a.Metadata = make(map[string]string)
	}

	return &a, nil
}
