package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// AlertRepository defines the interface for alert persistence
type AlertRepository interface {
	// Notification channels
	CreateChannel(ctx context.Context, channel *model.NotificationChannel) error
	GetChannelByID(ctx context.Context, id string) (*model.NotificationChannel, error)
	ListChannels(ctx context.Context) ([]*model.NotificationChannel, error)
	UpdateChannel(ctx context.Context, channel *model.NotificationChannel) error
	DeleteChannel(ctx context.Context, id string) error

	// Alert rules
	CreateRule(ctx context.Context, rule *model.AlertRule) error
	GetRuleByID(ctx context.Context, id string) (*model.AlertRule, error)
	ListRules(ctx context.Context, enabledOnly bool) ([]*model.AlertRule, error)
	UpdateRule(ctx context.Context, rule *model.AlertRule) error
	UpdateRuleLastTriggered(ctx context.Context, id string) error
	DeleteRule(ctx context.Context, id string) error

	// Fired alerts
	CreateAlert(ctx context.Context, alert *model.Alert) error
	GetAlertByID(ctx context.Context, id string) (*model.Alert, error)
	ListAlerts(ctx context.Context, filter model.AlertFilter) ([]*model.Alert, error)
	AcknowledgeAlert(ctx context.Context, id string, acknowledgedBy string) error
	ResolveAlert(ctx context.Context, id string) error
	HasOpenAlert(ctx context.Context, ruleID string, entityType string, entityID string) (bool, error)
	DeleteOldAlerts(ctx context.Context, olderThan time.Duration) (int64, error)
}

// AlertServiceConfig holds configuration for the alert service
type AlertServiceConfig struct {
	EvaluationInterval time.Duration // How often to evaluate rules (default: 60s)
	RetentionDays      int           // How long to keep resolved alerts (default: 30)
}

// AlertService manages alert rules and notifications
type AlertService struct {
	repo          AlertRepository
	config        AlertServiceConfig
	emailNotifier *EmailNotifier
	webhookNotifier *WebhookNotifier

	// Metric providers - injected dependencies
	nodeRepo      NodeMetricProvider
	vmRepo        VMMetricProvider
	storageRepo   StorageMetricProvider
	backupRepo    BackupMetricProvider
	uptimeRepo    UptimeProvider

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// NodeMetricProvider provides node metrics for alert evaluation
type NodeMetricProvider interface {
	GetAll(ctx context.Context) ([]*model.HostNode, error)
	GetByID(ctx context.Context, id string) (*model.HostNode, error)
}

// VMMetricProvider provides VM metrics for alert evaluation
type VMMetricProvider interface {
	GetAll(ctx context.Context) ([]*model.VM, error)
	GetByVMID(ctx context.Context, vmid int) (*model.VM, error)
}

// StorageMetricProvider provides storage pool metrics for alert evaluation
type StorageMetricProvider interface {
	ListStoragePoolsForAlerts(ctx context.Context) ([]*model.StoragePool, error)
}

// BackupMetricProvider provides backup status for alert evaluation
type BackupMetricProvider interface {
	ListBackupsForAlerts(ctx context.Context, status model.BackupStatus) ([]*model.Backup, error)
}

// UptimeProvider provides uptime check failure data for alert evaluation.
type UptimeProvider interface {
	GetUptimeMonitorFailures(ctx context.Context) ([]*UptimeMonitorFailure, error)
}

// NewAlertService creates a new alert service
func NewAlertService(repo AlertRepository, config AlertServiceConfig) *AlertService {
	if config.EvaluationInterval == 0 {
		config.EvaluationInterval = 60 * time.Second
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = 30
	}

	return &AlertService{
		repo:            repo,
		config:          config,
		emailNotifier:   NewEmailNotifier(),
		webhookNotifier: NewWebhookNotifier(),
	}
}

// WithNodeProvider sets the node metric provider
func (s *AlertService) WithNodeProvider(provider NodeMetricProvider) *AlertService {
	s.nodeRepo = provider
	return s
}

// WithVMProvider sets the VM metric provider
func (s *AlertService) WithVMProvider(provider VMMetricProvider) *AlertService {
	s.vmRepo = provider
	return s
}

// WithStorageProvider sets the storage metric provider
func (s *AlertService) WithStorageProvider(provider StorageMetricProvider) *AlertService {
	s.storageRepo = provider
	return s
}

// WithBackupProvider sets the backup metric provider
func (s *AlertService) WithBackupProvider(provider BackupMetricProvider) *AlertService {
	s.backupRepo = provider
	return s
}

// WithUptimeProvider sets the uptime provider for uptime_check_failed alerts
func (s *AlertService) WithUptimeProvider(provider UptimeProvider) *AlertService {
	s.uptimeRepo = provider
	return s
}

// Start begins the alert evaluation loop
func (s *AlertService) Start() {
	s.mu.Lock()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.mu.Unlock()

	s.wg.Add(1)
	go s.evaluationLoop()

	slog.Info("Alert service started",
		"evaluation_interval", s.config.EvaluationInterval,
		"retention_days", s.config.RetentionDays)
}

// Stop stops the alert evaluation loop
func (s *AlertService) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	s.wg.Wait()
	slog.Info("Alert service stopped")
}

// evaluationLoop periodically evaluates all enabled alert rules
func (s *AlertService) evaluationLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.EvaluationInterval)
	defer ticker.Stop()

	// Run initial evaluation
	s.evaluateAllRules()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.evaluateAllRules()
			s.cleanupOldAlerts()
		}
	}
}

// evaluateAllRules evaluates all enabled alert rules
func (s *AlertService) evaluateAllRules() {
	ctx := s.ctx

	rules, err := s.repo.ListRules(ctx, true) // enabled only
	if err != nil {
		slog.Error("Failed to list alert rules", "error", err)
		return
	}

	for _, rule := range rules {
		if err := s.evaluateRule(ctx, rule); err != nil {
			slog.Error("Failed to evaluate rule",
				"rule_id", rule.ID,
				"rule_name", rule.Name,
				"error", err)
		}
	}
}

// evaluateRule evaluates a single alert rule
func (s *AlertService) evaluateRule(ctx context.Context, rule *model.AlertRule) error {
	contexts, err := s.gatherAlertContexts(ctx, rule)
	if err != nil {
		return fmt.Errorf("failed to gather contexts: %w", err)
	}

	for _, alertCtx := range contexts {
		if rule.ShouldFire(alertCtx) {
			// Check if there's already an open alert for this rule/entity
			hasOpen, err := s.repo.HasOpenAlert(ctx, rule.ID, alertCtx.EntityType, alertCtx.EntityID)
			if err != nil {
				slog.Error("Failed to check for open alert", "error", err)
				continue
			}

			if hasOpen {
				// Skip - already have an open alert for this
				continue
			}

			// Fire the alert
			if err := s.fireAlert(ctx, rule, alertCtx); err != nil {
				slog.Error("Failed to fire alert",
					"rule_id", rule.ID,
					"entity", alertCtx.EntityName,
					"error", err)
			}
		}
	}

	return nil
}

// gatherAlertContexts gathers relevant contexts for a rule based on its type
func (s *AlertService) gatherAlertContexts(ctx context.Context, rule *model.AlertRule) ([]model.AlertContext, error) {
	var contexts []model.AlertContext

	switch rule.Type {
	case model.AlertTypeStoragePoolUsage:
		contexts = s.gatherStoragePoolUsage(ctx, rule)
	case model.AlertTypeVMStopped:
		contexts = s.gatherVMStopped(ctx, rule)
	case model.AlertTypeBackupFailed:
		contexts = s.gatherBackupFailed(ctx, rule)
	case model.AlertTypeNodeOffline:
		contexts = s.gatherNodeOffline(ctx, rule)
	case model.AlertTypeCPUUsage:
		contexts = s.gatherCPUUsage(ctx, rule)
	case model.AlertTypeMemoryUsage:
		contexts = s.gatherMemoryUsage(ctx, rule)
	case model.AlertTypeUptimeCheckFailed:
		contexts = s.gatherUptimeCheckFailed(ctx, rule)
	}

	return contexts, nil
}

func (s *AlertService) gatherStoragePoolUsage(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.storageRepo == nil {
		return contexts
	}

	pools, err := s.storageRepo.ListStoragePoolsForAlerts(ctx)
	if err != nil {
		slog.Error("Failed to list storage pools", "error", err)
		return contexts
	}

	for _, pool := range pools {
		// Filter by entity if specified
		if rule.EntityID != "" && rule.EntityID != pool.ID {
			continue
		}

		usagePercent := pool.UsagePercent
		if usagePercent == 0 && pool.CapacityBytes > 0 {
			usagePercent = float64(pool.UsedBytes) / float64(pool.CapacityBytes) * 100
		}

		contexts = append(contexts, model.AlertContext{
			EntityType: "storage_pool",
			EntityID:   pool.ID,
			EntityName: pool.Name,
			Value:      usagePercent,
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"capacity_bytes": fmt.Sprintf("%d", pool.CapacityBytes),
				"used_bytes":     fmt.Sprintf("%d", pool.UsedBytes),
			},
		})
	}

	return contexts
}

func (s *AlertService) gatherVMStopped(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.vmRepo == nil {
		return contexts
	}

	vms, err := s.vmRepo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to list VMs", "error", err)
		return contexts
	}

	for _, vm := range vms {
		// Filter by entity if specified
		if rule.EntityID != "" && fmt.Sprintf("%d", vm.VMID) != rule.EntityID {
			continue
		}

		// Value of 1 means VM is stopped when it shouldn't be
		// For now, we report all stopped VMs
		if vm.Status == model.VMStatusStopped {
			contexts = append(contexts, model.AlertContext{
				EntityType: "vm",
				EntityID:   fmt.Sprintf("%d", vm.VMID),
				EntityName: vm.Name,
				Value:      1, // Condition is true
				Timestamp:  time.Now(),
				Metadata: map[string]string{
					"status": string(vm.Status),
				},
			})
		}
	}

	return contexts
}

func (s *AlertService) gatherBackupFailed(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.backupRepo == nil {
		return contexts
	}

	// Get all failed backups
	backups, err := s.backupRepo.ListBackupsForAlerts(ctx, model.BackupStatusFailed)
	if err != nil {
		slog.Error("Failed to list backups", "error", err)
		return contexts
	}

	for _, backup := range backups {
		// Only alert on recent failures (within last hour)
		backupTime, err := time.Parse(time.RFC3339, backup.CreatedAt)
		if err != nil || time.Since(backupTime) > time.Hour {
			continue
		}

		contexts = append(contexts, model.AlertContext{
			EntityType: "backup",
			EntityID:   backup.ID,
			EntityName: backup.Name,
			Value:      1, // Condition is true
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"vm_id":        fmt.Sprintf("%d", backup.VMID),
				"vm_name":      backup.VMName,
				"error":        backup.ErrorMessage,
				"backup_type":  string(backup.Type),
			},
		})
	}

	return contexts
}

func (s *AlertService) gatherNodeOffline(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.nodeRepo == nil {
		return contexts
	}

	nodes, err := s.nodeRepo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to list nodes", "error", err)
		return contexts
	}

	for _, node := range nodes {
		// Filter by entity if specified
		if rule.EntityID != "" && rule.EntityID != node.ID {
			continue
		}

		// Value of 1 means node is offline
		if node.Status == model.NodeStatusOffline {
			contexts = append(contexts, model.AlertContext{
				EntityType: "node",
				EntityID:   node.ID,
				EntityName: node.ID, // Node ID is typically the hostname
				Value:      1,       // Condition is true
				Timestamp:  time.Now(),
				Metadata: map[string]string{
					"status": string(node.Status),
				},
			})
		}
	}

	return contexts
}

func (s *AlertService) gatherCPUUsage(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.nodeRepo == nil {
		return contexts
	}

	nodes, err := s.nodeRepo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to list nodes", "error", err)
		return contexts
	}

	for _, node := range nodes {
		// Filter by entity if specified
		if rule.EntityID != "" && rule.EntityID != node.ID {
			continue
		}

		// Skip offline nodes
		if node.Status != model.NodeStatusOnline {
			continue
		}

		contexts = append(contexts, model.AlertContext{
			EntityType: "node",
			EntityID:   node.ID,
			EntityName: node.ID,
			Value:      node.CPU.Used, // CPU usage percentage
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"cores": fmt.Sprintf("%d", node.CPU.Cores),
			},
		})
	}

	return contexts
}

func (s *AlertService) gatherMemoryUsage(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.nodeRepo == nil {
		return contexts
	}

	nodes, err := s.nodeRepo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to list nodes", "error", err)
		return contexts
	}

	for _, node := range nodes {
		// Filter by entity if specified
		if rule.EntityID != "" && rule.EntityID != node.ID {
			continue
		}

		// Skip offline nodes
		if node.Status != model.NodeStatusOnline {
			continue
		}

		// Calculate memory usage percentage
		usagePercent := 0.0
		if node.Memory.Total > 0 {
			usagePercent = (node.Memory.Used / node.Memory.Total) * 100
		}

		contexts = append(contexts, model.AlertContext{
			EntityType: "node",
			EntityID:   node.ID,
			EntityName: node.ID,
			Value:      usagePercent,
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"used_gb":  fmt.Sprintf("%.2f", node.Memory.Used),
				"total_gb": fmt.Sprintf("%.2f", node.Memory.Total),
			},
		})
	}

	return contexts
}

func (s *AlertService) gatherUptimeCheckFailed(ctx context.Context, rule *model.AlertRule) []model.AlertContext {
	var contexts []model.AlertContext

	if s.uptimeRepo == nil {
		return contexts
	}

	failures, err := s.uptimeRepo.GetUptimeMonitorFailures(ctx)
	if err != nil {
		slog.Error("Failed to list uptime failures", "error", err)
		return contexts
	}

	for _, f := range failures {
		// Filter by entity if specified
		if rule.EntityID != "" && rule.EntityID != f.MonitorID {
			continue
		}

		contexts = append(contexts, model.AlertContext{
			EntityType: "uptime_monitor",
			EntityID:   f.MonitorID,
			EntityName: f.MonitorName,
			Value:      float64(f.ConsecutiveFailures),
			Timestamp:  time.Now(),
			Metadata: map[string]string{
				"url":                  f.URL,
				"consecutive_failures": fmt.Sprintf("%d", f.ConsecutiveFailures),
				"last_error":           f.LastError,
			},
		})
	}

	return contexts
}

// FireUptimeAlert directly fires an alert for an uptime check failure.
// Called by ProxyService when a monitor has consecutive failures, bypassing
// the rule evaluation loop for immediate notification.
func (s *AlertService) FireUptimeAlert(ctx context.Context, monitorID, monitorName, url string, consecutiveFailures int, lastError string) {
	alertCtx := model.AlertContext{
		EntityType: "uptime_monitor",
		EntityID:   monitorID,
		EntityName: monitorName,
		Value:      float64(consecutiveFailures),
		Timestamp:  time.Now(),
		Metadata: map[string]string{
			"url":                  url,
			"consecutive_failures": fmt.Sprintf("%d", consecutiveFailures),
			"last_error":           lastError,
		},
	}

	// Get all enabled channels and fire to each one that has an uptime_check_failed rule.
	s.mu.RLock()
	evalCtx := s.ctx
	s.mu.RUnlock()

	if evalCtx == nil {
		return
	}

	rules, err := s.repo.ListRules(evalCtx, true)
	if err != nil {
		slog.Error("Failed to list rules for uptime alert", "error", err)
		return
	}

	for _, rule := range rules {
		if rule.Type != model.AlertTypeUptimeCheckFailed {
			continue
		}
		if rule.EntityID != "" && rule.EntityID != monitorID {
			continue
		}

		// Check if there's already an open alert for this monitor.
		hasOpen, err := s.repo.HasOpenAlert(evalCtx, rule.ID, "uptime_monitor", monitorID)
		if err != nil || hasOpen {
			continue
		}

		if err := s.fireAlert(evalCtx, rule, alertCtx); err != nil {
			slog.Error("Failed to fire uptime alert",
				"rule_id", rule.ID,
				"monitor", monitorName,
				"error", err)
		}
	}
}

// fireAlert creates a fired alert and sends a notification
func (s *AlertService) fireAlert(ctx context.Context, rule *model.AlertRule, alertCtx model.AlertContext) error {
	// Get the notification channel
	channel, err := s.repo.GetChannelByID(ctx, rule.ChannelID)
	if err != nil {
		return fmt.Errorf("failed to get notification channel: %w", err)
	}

	if !channel.Enabled {
		return fmt.Errorf("notification channel %s is disabled", channel.ID)
	}

	// Create the alert record
	alert := &model.Alert{
		ID:         uuid.New().String(),
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		EntityType: alertCtx.EntityType,
		EntityID:   alertCtx.EntityID,
		EntityName: alertCtx.EntityName,
		Message:    s.formatAlertMessage(rule, alertCtx),
		Severity:   rule.GetSeverity(alertCtx),
		Status:     model.AlertStatusOpen,
		FiredAt:    time.Now(),
		Metadata:   alertCtx.Metadata,
	}

	if err := s.repo.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	// Update rule last triggered
	if err := s.repo.UpdateRuleLastTriggered(ctx, rule.ID); err != nil {
		slog.Error("Failed to update rule last triggered", "error", err)
	}

	// Send notification
	if err := s.sendNotification(alert, channel); err != nil {
		slog.Error("Failed to send notification",
			"alert_id", alert.ID,
			"channel_id", channel.ID,
			"error", err)
		// Don't fail the whole operation if notification fails
	}

	slog.Info("Alert fired",
		"alert_id", alert.ID,
		"rule_id", rule.ID,
		"rule_name", rule.Name,
		"entity", alertCtx.EntityName,
		"severity", alert.Severity)

	return nil
}

func (s *AlertService) formatAlertMessage(rule *model.AlertRule, ctx model.AlertContext) string {
	switch rule.Type {
	case model.AlertTypeStoragePoolUsage:
		return fmt.Sprintf("Storage pool '%s' usage is %.1f%% (threshold: %.1f%%)",
			ctx.EntityName, ctx.Value, *rule.Threshold)
	case model.AlertTypeVMStopped:
		return fmt.Sprintf("VM '%s' has stopped unexpectedly", ctx.EntityName)
	case model.AlertTypeBackupFailed:
		return fmt.Sprintf("Backup '%s' failed", ctx.EntityName)
	case model.AlertTypeNodeOffline:
		return fmt.Sprintf("Node '%s' is offline", ctx.EntityName)
	case model.AlertTypeCPUUsage:
		return fmt.Sprintf("Node '%s' CPU usage is %.1f%% (threshold: %.1f%%)",
			ctx.EntityName, ctx.Value, *rule.Threshold)
	case model.AlertTypeMemoryUsage:
		return fmt.Sprintf("Node '%s' memory usage is %.1f%% (threshold: %.1f%%)",
			ctx.EntityName, ctx.Value, *rule.Threshold)
	case model.AlertTypeUptimeCheckFailed:
		url := ctx.Metadata["url"]
		fails := ctx.Metadata["consecutive_failures"]
		return fmt.Sprintf("Uptime monitor '%s' (%s) has failed %s consecutive check(s)", ctx.EntityName, url, fails)
	default:
		return fmt.Sprintf("Alert '%s' triggered for %s", rule.Name, ctx.EntityName)
	}
}

func (s *AlertService) sendNotification(alert *model.Alert, channel *model.NotificationChannel) error {
	switch channel.Type {
	case model.ChannelTypeEmail:
		return s.emailNotifier.Send(alert, channel)
	case model.ChannelTypeWebhook:
		return s.webhookNotifier.Send(alert, channel)
	default:
		return fmt.Errorf("unknown channel type: %s", channel.Type)
	}
}

func (s *AlertService) cleanupOldAlerts() {
	ctx := s.ctx
	retention := time.Duration(s.config.RetentionDays) * 24 * time.Hour

	deleted, err := s.repo.DeleteOldAlerts(ctx, retention)
	if err != nil {
		slog.Error("Failed to cleanup old alerts", "error", err)
		return
	}

	if deleted > 0 {
		slog.Info("Cleaned up old resolved alerts", "count", deleted)
	}
}

// --- Public API methods ---

// CreateChannel creates a new notification channel
func (s *AlertService) CreateChannel(ctx context.Context, req *model.NotificationChannelCreateRequest) (*model.NotificationChannel, error) {
	channel := &model.NotificationChannel{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Type:      req.Type,
		Config:    req.Config,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// GetChannel retrieves a notification channel by ID
func (s *AlertService) GetChannel(ctx context.Context, id string) (*model.NotificationChannel, error) {
	return s.repo.GetChannelByID(ctx, id)
}

// ListChannels lists all notification channels
func (s *AlertService) ListChannels(ctx context.Context) ([]*model.NotificationChannel, error) {
	return s.repo.ListChannels(ctx)
}

// UpdateChannel updates a notification channel
func (s *AlertService) UpdateChannel(ctx context.Context, id string, req *model.NotificationChannelUpdateRequest) (*model.NotificationChannel, error) {
	channel, err := s.repo.GetChannelByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Config != nil {
		channel.Config = req.Config
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}
	channel.UpdatedAt = time.Now()

	if err := s.repo.UpdateChannel(ctx, channel); err != nil {
		return nil, err
	}

	return channel, nil
}

// DeleteChannel deletes a notification channel
func (s *AlertService) DeleteChannel(ctx context.Context, id string) error {
	return s.repo.DeleteChannel(ctx, id)
}

// CreateRule creates a new alert rule
func (s *AlertService) CreateRule(ctx context.Context, req *model.AlertRuleCreateRequest) (*model.AlertRule, error) {
	rule := &model.AlertRule{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Threshold:       req.Threshold,
		DurationMinutes: req.DurationMinutes,
		EntityType:      req.EntityType,
		EntityID:        req.EntityID,
		ChannelID:       req.ChannelID,
		Enabled:         req.Enabled,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// GetRule retrieves an alert rule by ID
func (s *AlertService) GetRule(ctx context.Context, id string) (*model.AlertRule, error) {
	return s.repo.GetRuleByID(ctx, id)
}

// ListRules lists all alert rules
func (s *AlertService) ListRules(ctx context.Context, enabledOnly bool) ([]*model.AlertRule, error) {
	return s.repo.ListRules(ctx, enabledOnly)
}

// UpdateRule updates an alert rule
func (s *AlertService) UpdateRule(ctx context.Context, id string, req *model.AlertRuleUpdateRequest) (*model.AlertRule, error) {
	rule, err := s.repo.GetRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if req.Threshold != nil {
		rule.Threshold = req.Threshold
	}
	if req.DurationMinutes != nil {
		rule.DurationMinutes = *req.DurationMinutes
	}
	if req.EntityType != nil {
		rule.EntityType = *req.EntityType
	}
	if req.EntityID != nil {
		rule.EntityID = *req.EntityID
	}
	if req.ChannelID != nil {
		rule.ChannelID = *req.ChannelID
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	rule.UpdatedAt = time.Now()

	if err := s.repo.UpdateRule(ctx, rule); err != nil {
		return nil, err
	}

	return rule, nil
}

// DeleteRule deletes an alert rule
func (s *AlertService) DeleteRule(ctx context.Context, id string) error {
	return s.repo.DeleteRule(ctx, id)
}

// GetAlert retrieves a fired alert by ID
func (s *AlertService) GetAlert(ctx context.Context, id string) (*model.Alert, error) {
	return s.repo.GetAlertByID(ctx, id)
}

// ListAlerts lists fired alerts with optional filters
func (s *AlertService) ListAlerts(ctx context.Context, filter model.AlertFilter) ([]*model.Alert, error) {
	return s.repo.ListAlerts(ctx, filter)
}

// AcknowledgeAlert acknowledges a fired alert
func (s *AlertService) AcknowledgeAlert(ctx context.Context, id string, acknowledgedBy string) error {
	return s.repo.AcknowledgeAlert(ctx, id, acknowledgedBy)
}

// ResolveAlert resolves a fired alert
func (s *AlertService) ResolveAlert(ctx context.Context, id string) error {
	return s.repo.ResolveAlert(ctx, id)
}
