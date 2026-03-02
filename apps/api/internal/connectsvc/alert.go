package connectsvc

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// AlertServiceServer implements labv1connect.AlertServiceHandler.
type AlertServiceServer struct {
	labv1connect.UnimplementedAlertServiceHandler
	svc *service.AlertService
}

// NewAlertServiceServer creates a new AlertServiceServer.
func NewAlertServiceServer(svc *service.AlertService) *AlertServiceServer {
	return &AlertServiceServer{svc: svc}
}

// --- Notification Channels ---

// CreateNotificationChannel creates a new notification channel.
func (s *AlertServiceServer) CreateNotificationChannel(
	ctx context.Context,
	req *connect.Request[labv1.CreateNotificationChannelRequest],
) (*connect.Response[labv1.CreateNotificationChannelResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if req.Msg.Type == labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("type is required"))
	}

	channel, err := s.svc.CreateChannel(ctx, &model.NotificationChannelCreateRequest{
		Name:   req.Msg.Name,
		Type:   protoChannelTypeToModel(req.Msg.Type),
		Config: req.Msg.Config,
	})
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.CreateNotificationChannelResponse{
		Channel: modelChannelToProto(channel),
	}), nil
}

// GetNotificationChannel retrieves a notification channel by ID.
func (s *AlertServiceServer) GetNotificationChannel(
	ctx context.Context,
	req *connect.Request[labv1.GetNotificationChannelRequest],
) (*connect.Response[labv1.GetNotificationChannelResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	channel, err := s.svc.GetChannel(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.GetNotificationChannelResponse{
		Channel: modelChannelToProto(channel),
	}), nil
}

// ListNotificationChannels lists all notification channels.
func (s *AlertServiceServer) ListNotificationChannels(
	ctx context.Context,
	_ *connect.Request[labv1.ListNotificationChannelsRequest],
) (*connect.Response[labv1.ListNotificationChannelsResponse], error) {
	channels, err := s.svc.ListChannels(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	protoChannels := make([]*labv1.NotificationChannel, len(channels))
	for i, c := range channels {
		protoChannels[i] = modelChannelToProto(c)
	}

	return connect.NewResponse(&labv1.ListNotificationChannelsResponse{
		Channels: protoChannels,
	}), nil
}

// UpdateNotificationChannel updates a notification channel.
func (s *AlertServiceServer) UpdateNotificationChannel(
	ctx context.Context,
	req *connect.Request[labv1.UpdateNotificationChannelRequest],
) (*connect.Response[labv1.UpdateNotificationChannelResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	channel, err := s.svc.UpdateChannel(ctx, req.Msg.Id, &model.NotificationChannelUpdateRequest{
		Name:    req.Msg.Name,
		Config:  req.Msg.Config,
		Enabled: req.Msg.Enabled,
	})
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.UpdateNotificationChannelResponse{
		Channel: modelChannelToProto(channel),
	}), nil
}

// DeleteNotificationChannel deletes a notification channel.
func (s *AlertServiceServer) DeleteNotificationChannel(
	ctx context.Context,
	req *connect.Request[labv1.DeleteNotificationChannelRequest],
) (*connect.Response[labv1.DeleteNotificationChannelResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.svc.DeleteChannel(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.DeleteNotificationChannelResponse{}), nil
}

// --- Alert Rules ---

// CreateAlertRule creates a new alert rule.
func (s *AlertServiceServer) CreateAlertRule(
	ctx context.Context,
	req *connect.Request[labv1.CreateAlertRuleRequest],
) (*connect.Response[labv1.CreateAlertRuleResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if req.Msg.Type == labv1.AlertRuleType_ALERT_RULE_TYPE_UNSPECIFIED {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("type is required"))
	}
	if req.Msg.ChannelId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("channel_id is required"))
	}

	rule, err := s.svc.CreateRule(ctx, &model.AlertRuleCreateRequest{
		Name:            req.Msg.Name,
		Description:     req.Msg.Description,
		Type:            protoAlertRuleTypeToModel(req.Msg.Type),
		Threshold:       req.Msg.Threshold,
		DurationMinutes: int(req.Msg.DurationMinutes),
		EntityType:      req.Msg.EntityType,
		EntityID:        req.Msg.EntityId,
		ChannelID:       req.Msg.ChannelId,
		Enabled:         req.Msg.Enabled,
	})
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.CreateAlertRuleResponse{
		Rule: modelAlertRuleToProto(rule),
	}), nil
}

// GetAlertRule retrieves an alert rule by ID.
func (s *AlertServiceServer) GetAlertRule(
	ctx context.Context,
	req *connect.Request[labv1.GetAlertRuleRequest],
) (*connect.Response[labv1.GetAlertRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	rule, err := s.svc.GetRule(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.GetAlertRuleResponse{
		Rule: modelAlertRuleToProto(rule),
	}), nil
}

// ListAlertRules lists all alert rules.
func (s *AlertServiceServer) ListAlertRules(
	ctx context.Context,
	req *connect.Request[labv1.ListAlertRulesRequest],
) (*connect.Response[labv1.ListAlertRulesResponse], error) {
	rules, err := s.svc.ListRules(ctx, req.Msg.EnabledOnly)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	protoRules := make([]*labv1.AlertRule, len(rules))
	for i, r := range rules {
		protoRules[i] = modelAlertRuleToProto(r)
	}

	return connect.NewResponse(&labv1.ListAlertRulesResponse{
		Rules: protoRules,
	}), nil
}

// UpdateAlertRule updates an alert rule.
func (s *AlertServiceServer) UpdateAlertRule(
	ctx context.Context,
	req *connect.Request[labv1.UpdateAlertRuleRequest],
) (*connect.Response[labv1.UpdateAlertRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	updateReq := &model.AlertRuleUpdateRequest{
		Name:        req.Msg.Name,
		Description: req.Msg.Description,
		Threshold:   req.Msg.Threshold,
		EntityType:  req.Msg.EntityType,
		EntityID:    req.Msg.EntityId,
		ChannelID:   req.Msg.ChannelId,
		Enabled:     req.Msg.Enabled,
	}
	if req.Msg.DurationMinutes != nil {
		val := int(*req.Msg.DurationMinutes)
		updateReq.DurationMinutes = &val
	}

	rule, err := s.svc.UpdateRule(ctx, req.Msg.Id, updateReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.UpdateAlertRuleResponse{
		Rule: modelAlertRuleToProto(rule),
	}), nil
}

// DeleteAlertRule deletes an alert rule.
func (s *AlertServiceServer) DeleteAlertRule(
	ctx context.Context,
	req *connect.Request[labv1.DeleteAlertRuleRequest],
) (*connect.Response[labv1.DeleteAlertRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.svc.DeleteRule(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.DeleteAlertRuleResponse{}), nil
}

// --- Fired Alerts ---

// GetAlert retrieves a fired alert by ID.
func (s *AlertServiceServer) GetAlert(
	ctx context.Context,
	req *connect.Request[labv1.GetAlertRequest],
) (*connect.Response[labv1.GetAlertResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	alert, err := s.svc.GetAlert(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.GetAlertResponse{
		Alert: modelAlertToProto(alert),
	}), nil
}

// ListAlerts lists fired alerts with optional filters.
func (s *AlertServiceServer) ListAlerts(
	ctx context.Context,
	req *connect.Request[labv1.ListAlertsRequest],
) (*connect.Response[labv1.ListAlertsResponse], error) {
	alerts, err := s.svc.ListAlerts(ctx, model.AlertFilter{
		Status:     protoAlertStatusToModel(req.Msg.Status),
		Severity:   protoAlertSeverityToModel(req.Msg.Severity),
		RuleID:     req.Msg.RuleId,
		EntityType: req.Msg.EntityType,
		EntityID:   req.Msg.EntityId,
		OpenOnly:   req.Msg.OpenOnly,
	})
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	protoAlerts := make([]*labv1.Alert, len(alerts))
	for i, a := range alerts {
		protoAlerts[i] = modelAlertToProto(a)
	}

	return connect.NewResponse(&labv1.ListAlertsResponse{
		Alerts: protoAlerts,
		Total:  int32(len(alerts)),
	}), nil
}

// AcknowledgeAlert acknowledges a fired alert.
func (s *AlertServiceServer) AcknowledgeAlert(
	ctx context.Context,
	req *connect.Request[labv1.AcknowledgeAlertRequest],
) (*connect.Response[labv1.AcknowledgeAlertResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.svc.AcknowledgeAlert(ctx, req.Msg.Id, "user"); err != nil {
		return nil, serviceErrToConnect(err)
	}

	alert, err := s.svc.GetAlert(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.AcknowledgeAlertResponse{
		Alert: modelAlertToProto(alert),
	}), nil
}

// ResolveAlert resolves a fired alert.
func (s *AlertServiceServer) ResolveAlert(
	ctx context.Context,
	req *connect.Request[labv1.ResolveAlertRequest],
) (*connect.Response[labv1.ResolveAlertResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.svc.ResolveAlert(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}

	alert, err := s.svc.GetAlert(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.ResolveAlertResponse{
		Alert: modelAlertToProto(alert),
	}), nil
}

// --- conversion helpers ---

func protoChannelTypeToModel(t labv1.NotificationChannelType) model.NotificationChannelType {
	switch t {
	case labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL:
		return model.ChannelTypeEmail
	case labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK:
		return model.ChannelTypeWebhook
	default:
		return ""
	}
}

func modelChannelTypeToProto(t model.NotificationChannelType) labv1.NotificationChannelType {
	switch t {
	case model.ChannelTypeEmail:
		return labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_EMAIL
	case model.ChannelTypeWebhook:
		return labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_WEBHOOK
	default:
		return labv1.NotificationChannelType_NOTIFICATION_CHANNEL_TYPE_UNSPECIFIED
	}
}

func protoAlertRuleTypeToModel(t labv1.AlertRuleType) model.AlertRuleType {
	switch t {
	case labv1.AlertRuleType_ALERT_RULE_TYPE_STORAGE_POOL_USAGE:
		return model.AlertTypeStoragePoolUsage
	case labv1.AlertRuleType_ALERT_RULE_TYPE_VM_STOPPED:
		return model.AlertTypeVMStopped
	case labv1.AlertRuleType_ALERT_RULE_TYPE_BACKUP_FAILED:
		return model.AlertTypeBackupFailed
	case labv1.AlertRuleType_ALERT_RULE_TYPE_NODE_OFFLINE:
		return model.AlertTypeNodeOffline
	case labv1.AlertRuleType_ALERT_RULE_TYPE_CPU_USAGE:
		return model.AlertTypeCPUUsage
	case labv1.AlertRuleType_ALERT_RULE_TYPE_MEMORY_USAGE:
		return model.AlertTypeMemoryUsage
	default:
		return ""
	}
}

func modelAlertRuleTypeToProto(t model.AlertRuleType) labv1.AlertRuleType {
	switch t {
	case model.AlertTypeStoragePoolUsage:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_STORAGE_POOL_USAGE
	case model.AlertTypeVMStopped:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_VM_STOPPED
	case model.AlertTypeBackupFailed:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_BACKUP_FAILED
	case model.AlertTypeNodeOffline:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_NODE_OFFLINE
	case model.AlertTypeCPUUsage:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_CPU_USAGE
	case model.AlertTypeMemoryUsage:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_MEMORY_USAGE
	default:
		return labv1.AlertRuleType_ALERT_RULE_TYPE_UNSPECIFIED
	}
}

func protoAlertSeverityToModel(s labv1.AlertSeverity) model.AlertSeverity {
	switch s {
	case labv1.AlertSeverity_ALERT_SEVERITY_INFO:
		return model.AlertSeverityInfo
	case labv1.AlertSeverity_ALERT_SEVERITY_WARNING:
		return model.AlertSeverityWarning
	case labv1.AlertSeverity_ALERT_SEVERITY_CRITICAL:
		return model.AlertSeverityCritical
	default:
		return ""
	}
}

func modelAlertSeverityToProto(s model.AlertSeverity) labv1.AlertSeverity {
	switch s {
	case model.AlertSeverityInfo:
		return labv1.AlertSeverity_ALERT_SEVERITY_INFO
	case model.AlertSeverityWarning:
		return labv1.AlertSeverity_ALERT_SEVERITY_WARNING
	case model.AlertSeverityCritical:
		return labv1.AlertSeverity_ALERT_SEVERITY_CRITICAL
	default:
		return labv1.AlertSeverity_ALERT_SEVERITY_UNSPECIFIED
	}
}

func protoAlertStatusToModel(s labv1.AlertStatus) model.AlertStatus {
	switch s {
	case labv1.AlertStatus_ALERT_STATUS_OPEN:
		return model.AlertStatusOpen
	case labv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED:
		return model.AlertStatusAcknowledged
	case labv1.AlertStatus_ALERT_STATUS_RESOLVED:
		return model.AlertStatusResolved
	default:
		return ""
	}
}

func modelAlertStatusToProto(s model.AlertStatus) labv1.AlertStatus {
	switch s {
	case model.AlertStatusOpen:
		return labv1.AlertStatus_ALERT_STATUS_OPEN
	case model.AlertStatusAcknowledged:
		return labv1.AlertStatus_ALERT_STATUS_ACKNOWLEDGED
	case model.AlertStatusResolved:
		return labv1.AlertStatus_ALERT_STATUS_RESOLVED
	default:
		return labv1.AlertStatus_ALERT_STATUS_UNSPECIFIED
	}
}

func modelChannelToProto(c *model.NotificationChannel) *labv1.NotificationChannel {
	return &labv1.NotificationChannel{
		Id:        c.ID,
		Name:      c.Name,
		Type:      modelChannelTypeToProto(c.Type),
		Config:    c.Config,
		Enabled:   c.Enabled,
		CreatedAt: c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func modelAlertRuleToProto(r *model.AlertRule) *labv1.AlertRule {
	proto := &labv1.AlertRule{
		Id:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		Type:            modelAlertRuleTypeToProto(r.Type),
		DurationMinutes: int32(r.DurationMinutes),
		EntityType:      r.EntityType,
		EntityId:        r.EntityID,
		ChannelId:       r.ChannelID,
		Enabled:         r.Enabled,
		CreatedAt:       r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if r.Threshold != nil {
		proto.Threshold = r.Threshold
	}

	if r.LastTriggeredAt != nil {
		ts := r.LastTriggeredAt.Format("2006-01-02T15:04:05Z07:00")
		proto.LastTriggeredAt = &ts
	}

	return proto
}

func modelAlertToProto(a *model.Alert) *labv1.Alert {
	proto := &labv1.Alert{
		Id:             a.ID,
		RuleId:         a.RuleID,
		RuleName:       a.RuleName,
		EntityType:     a.EntityType,
		EntityId:       a.EntityID,
		EntityName:     a.EntityName,
		Message:        a.Message,
		Severity:       modelAlertSeverityToProto(a.Severity),
		Status:         modelAlertStatusToProto(a.Status),
		FiredAt:        a.FiredAt.Format("2006-01-02T15:04:05Z07:00"),
		AcknowledgedBy: a.AcknowledgedBy,
		Metadata:       a.Metadata,
	}

	if a.AcknowledgedAt != nil {
		ts := a.AcknowledgedAt.Format("2006-01-02T15:04:05Z07:00")
		proto.AcknowledgedAt = &ts
	}

	if a.ResolvedAt != nil {
		ts := a.ResolvedAt.Format("2006-01-02T15:04:05Z07:00")
		proto.ResolvedAt = &ts
	}

	return proto
}
