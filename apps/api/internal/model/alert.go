package model

import "time"

// AlertRuleType represents the type of alert rule
type AlertRuleType string

const (
	AlertTypeStoragePoolUsage  AlertRuleType = "storage_pool_usage"
	AlertTypeVMStopped         AlertRuleType = "vm_stopped"
	AlertTypeBackupFailed      AlertRuleType = "backup_failed"
	AlertTypeNodeOffline       AlertRuleType = "node_offline"
	AlertTypeCPUUsage          AlertRuleType = "cpu_usage"
	AlertTypeMemoryUsage       AlertRuleType = "memory_usage"
	AlertTypeUptimeCheckFailed AlertRuleType = "uptime_check_failed"
)

// NotificationChannelType represents the type of notification channel
type NotificationChannelType string

const (
	ChannelTypeEmail   NotificationChannelType = "email"
	ChannelTypeWebhook NotificationChannelType = "webhook"
)

// AlertSeverity represents the severity level of a fired alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of a fired alert
type AlertStatus string

const (
	AlertStatusOpen          AlertStatus = "open"
	AlertStatusAcknowledged  AlertStatus = "acknowledged"
	AlertStatusResolved      AlertStatus = "resolved"
)

// NotificationChannel represents a configured notification destination
type NotificationChannel struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      NotificationChannelType `json:"type"`
	Config    map[string]string      `json:"config"` // JSON stored as map
	Enabled   bool                   `json:"enabled"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// EmailChannelConfig contains email-specific configuration
type EmailChannelConfig struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUser     string `json:"smtp_user"`
	SMTPPassword string `json:"smtp_pass"`
	FromAddress  string `json:"from_address"`
	ToAddresses  string `json:"to_addresses"` // comma-separated
}

// WebhookChannelConfig contains webhook-specific configuration
type WebhookChannelConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`  // POST, PUT, etc.
	Headers map[string]string `json:"headers"` // Custom headers
}

// AlertRule represents a configured alert rule
type AlertRule struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	Type              AlertRuleType `json:"type"`
	Threshold         *float64      `json:"threshold,omitempty"`         // Percentage for usage alerts
	DurationMinutes   int           `json:"duration_minutes"`            // Sustained duration before firing
	EntityType        string        `json:"entity_type,omitempty"`       // "vm", "node", "storage_pool", "backup", or empty for all
	EntityID          string        `json:"entity_id,omitempty"`         // Specific entity ID or empty for all
	ChannelID         string        `json:"channel_id"`
	Enabled           bool          `json:"enabled"`
	LastTriggeredAt   *time.Time    `json:"last_triggered_at,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// Alert represents a fired alert instance
type Alert struct {
	ID             string        `json:"id"`
	RuleID         string        `json:"rule_id"`
	RuleName       string        `json:"rule_name"`
	EntityType     string        `json:"entity_type,omitempty"`
	EntityID       string        `json:"entity_id,omitempty"`
	EntityName     string        `json:"entity_name,omitempty"`
	Message        string        `json:"message"`
	Severity       AlertSeverity `json:"severity"`
	Status         AlertStatus   `json:"status"`
	FiredAt        time.Time     `json:"fired_at"`
	AcknowledgedAt *time.Time    `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string        `json:"acknowledged_by,omitempty"`
	ResolvedAt     *time.Time    `json:"resolved_at,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"` // Additional context
}

// NotificationChannelCreateRequest represents a request to create a notification channel
type NotificationChannelCreateRequest struct {
	Name   string                  `json:"name"`
	Type   NotificationChannelType `json:"type"`
	Config map[string]string       `json:"config"`
}

// NotificationChannelUpdateRequest represents a request to update a notification channel
type NotificationChannelUpdateRequest struct {
	Name   *string             `json:"name,omitempty"`
	Config map[string]string   `json:"config,omitempty"`
	Enabled *bool              `json:"enabled,omitempty"`
}

// AlertRuleCreateRequest represents a request to create an alert rule
type AlertRuleCreateRequest struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Type            AlertRuleType `json:"type"`
	Threshold       *float64      `json:"threshold,omitempty"`
	DurationMinutes int           `json:"duration_minutes"`
	EntityType      string        `json:"entity_type,omitempty"`
	EntityID        string        `json:"entity_id,omitempty"`
	ChannelID       string        `json:"channel_id"`
	Enabled         bool          `json:"enabled"`
}

// AlertRuleUpdateRequest represents a request to update an alert rule
type AlertRuleUpdateRequest struct {
	Name            *string       `json:"name,omitempty"`
	Description     *string       `json:"description,omitempty"`
	Threshold       *float64      `json:"threshold,omitempty"`
	DurationMinutes *int          `json:"duration_minutes,omitempty"`
	EntityType      *string       `json:"entity_type,omitempty"`
	EntityID        *string       `json:"entity_id,omitempty"`
	ChannelID       *string       `json:"channel_id,omitempty"`
	Enabled         *bool         `json:"enabled,omitempty"`
}

// AlertFilter represents filters for listing alerts
type AlertFilter struct {
	Status     AlertStatus   `json:"status,omitempty"`
	Severity   AlertSeverity `json:"severity,omitempty"`
	RuleID     string        `json:"rule_id,omitempty"`
	EntityType string        `json:"entity_type,omitempty"`
	EntityID   string        `json:"entity_id,omitempty"`
	OpenOnly   bool          `json:"open_only,omitempty"`
}

// AlertContext provides context for evaluating an alert condition
type AlertContext struct {
	EntityType string
	EntityID   string
	EntityName string
	Value      float64 // Current value (usage %, etc.)
	Timestamp  time.Time
	Metadata   map[string]string
}

// ShouldFire determines if an alert should fire based on the rule and context
func (r *AlertRule) ShouldFire(ctx AlertContext) bool {
	if !r.Enabled {
		return false
	}

	// Check entity filter
	if r.EntityType != "" && r.EntityType != ctx.EntityType {
		return false
	}
	if r.EntityID != "" && r.EntityID != ctx.EntityID {
		return false
	}

	// Check threshold (for usage-based alerts)
	if r.Threshold != nil {
		return ctx.Value >= *r.Threshold
	}

	// For non-threshold alerts (vm_stopped, backup_failed, node_offline), value > 0 means condition is true
	return ctx.Value > 0
}

// GetSeverity determines the severity based on threshold and rule type
func (r *AlertRule) GetSeverity(ctx AlertContext) AlertSeverity {
	if r.Threshold == nil {
		return AlertSeverityWarning
	}

	// Calculate severity based on how much threshold is exceeded
	exceededBy := ctx.Value - *r.Threshold
	if exceededBy >= 10 {
		return AlertSeverityCritical
	} else if exceededBy >= 5 {
		return AlertSeverityWarning
	}
	return AlertSeverityInfo
}
