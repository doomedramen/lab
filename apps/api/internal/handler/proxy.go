package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// ProxyServiceServer implements the ProxyService Connect RPC server.
type ProxyServiceServer struct {
	proxySvc *service.ProxyService
}

// NewProxyServiceServer creates a new proxy service server.
func NewProxyServiceServer(svc *service.ProxyService) *ProxyServiceServer {
	return &ProxyServiceServer{proxySvc: svc}
}

var _ labv1connect.ProxyServiceHandler = (*ProxyServiceServer)(nil)

// CreateProxyHost creates a new reverse proxy host.
func (s *ProxyServiceServer) CreateProxyHost(
	ctx context.Context,
	req *connect.Request[labv1.CreateProxyHostRequest],
) (*connect.Response[labv1.CreateProxyHostResponse], error) {
	if req.Msg.Domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("domain is required"))
	}
	if req.Msg.TargetUrl == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target_url is required"))
	}

	host, err := s.proxySvc.CreateHost(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateProxyHostResponse{
		ProxyHost: host,
	}), nil
}

// GetProxyHost retrieves a proxy host by ID.
func (s *ProxyServiceServer) GetProxyHost(
	ctx context.Context,
	req *connect.Request[labv1.GetProxyHostRequest],
) (*connect.Response[labv1.GetProxyHostResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	host, err := s.proxySvc.GetHost(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetProxyHostResponse{
		ProxyHost: host,
	}), nil
}

// ListProxyHosts returns all proxy hosts.
func (s *ProxyServiceServer) ListProxyHosts(
	ctx context.Context,
	_ *connect.Request[labv1.ListProxyHostsRequest],
) (*connect.Response[labv1.ListProxyHostsResponse], error) {
	hosts, total, err := s.proxySvc.ListHosts(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListProxyHostsResponse{
		ProxyHosts: hosts,
		Total:      total,
	}), nil
}

// UpdateProxyHost updates an existing proxy host.
func (s *ProxyServiceServer) UpdateProxyHost(
	ctx context.Context,
	req *connect.Request[labv1.UpdateProxyHostRequest],
) (*connect.Response[labv1.UpdateProxyHostResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if req.Msg.Domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("domain is required"))
	}
	if req.Msg.TargetUrl == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target_url is required"))
	}

	host, err := s.proxySvc.UpdateHost(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateProxyHostResponse{
		ProxyHost: host,
	}), nil
}

// DeleteProxyHost removes a proxy host.
func (s *ProxyServiceServer) DeleteProxyHost(
	ctx context.Context,
	req *connect.Request[labv1.DeleteProxyHostRequest],
) (*connect.Response[labv1.DeleteProxyHostResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.proxySvc.DeleteHost(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteProxyHostResponse{}), nil
}

// GetProxyStatus returns a live status snapshot for a proxy host.
func (s *ProxyServiceServer) GetProxyStatus(
	ctx context.Context,
	req *connect.Request[labv1.GetProxyStatusRequest],
) (*connect.Response[labv1.GetProxyStatusResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	status, err := s.proxySvc.GetStatus(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetProxyStatusResponse{
		Status: status,
	}), nil
}

// UploadCert stores a user-supplied TLS certificate for a custom-mode host.
func (s *ProxyServiceServer) UploadCert(
	ctx context.Context,
	req *connect.Request[labv1.UploadCertRequest],
) (*connect.Response[labv1.UploadCertResponse], error) {
	if req.Msg.ProxyHostId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("proxy_host_id is required"))
	}
	if req.Msg.CertPem == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cert_pem is required"))
	}
	if req.Msg.KeyPem == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("key_pem is required"))
	}

	if err := s.proxySvc.UploadCert(ctx, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UploadCertResponse{}), nil
}

// ---- Uptime monitor handlers ----

// CreateMonitor creates a new uptime monitor.
func (s *ProxyServiceServer) CreateMonitor(
	ctx context.Context,
	req *connect.Request[labv1.CreateMonitorRequest],
) (*connect.Response[labv1.CreateMonitorResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if req.Msg.Url == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("url is required"))
	}

	m, err := s.proxySvc.CreateMonitor(ctx, &model.UptimeMonitorCreateRequest{
		Name:               req.Msg.Name,
		URL:                req.Msg.Url,
		ProxyHostID:        req.Msg.ProxyHostId,
		IntervalSeconds:    int(req.Msg.IntervalSeconds),
		TimeoutSeconds:     int(req.Msg.TimeoutSeconds),
		ExpectedStatusCode: int(req.Msg.ExpectedStatusCode),
		Enabled:            req.Msg.Enabled,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateMonitorResponse{
		Monitor: monitorToProto(m),
	}), nil
}

// GetMonitor retrieves an uptime monitor by ID.
func (s *ProxyServiceServer) GetMonitor(
	ctx context.Context,
	req *connect.Request[labv1.GetMonitorRequest],
) (*connect.Response[labv1.GetMonitorResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	m, err := s.proxySvc.GetMonitor(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetMonitorResponse{
		Monitor: monitorToProto(m),
	}), nil
}

// ListMonitors returns all uptime monitors.
func (s *ProxyServiceServer) ListMonitors(
	ctx context.Context,
	_ *connect.Request[labv1.ListMonitorsRequest],
) (*connect.Response[labv1.ListMonitorsResponse], error) {
	monitors, err := s.proxySvc.ListMonitors(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var out []*labv1.UptimeMonitor
	for _, m := range monitors {
		out = append(out, monitorToProto(m))
	}

	return connect.NewResponse(&labv1.ListMonitorsResponse{
		Monitors: out,
		Total:    int32(len(out)),
	}), nil
}

// UpdateMonitor updates an existing uptime monitor.
func (s *ProxyServiceServer) UpdateMonitor(
	ctx context.Context,
	req *connect.Request[labv1.UpdateMonitorRequest],
) (*connect.Response[labv1.UpdateMonitorResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if req.Msg.Url == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("url is required"))
	}

	m, err := s.proxySvc.UpdateMonitor(ctx, &model.UptimeMonitorUpdateRequest{
		ID:                 req.Msg.Id,
		Name:               req.Msg.Name,
		URL:                req.Msg.Url,
		IntervalSeconds:    int(req.Msg.IntervalSeconds),
		TimeoutSeconds:     int(req.Msg.TimeoutSeconds),
		ExpectedStatusCode: int(req.Msg.ExpectedStatusCode),
		Enabled:            req.Msg.Enabled,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateMonitorResponse{
		Monitor: monitorToProto(m),
	}), nil
}

// DeleteMonitor removes an uptime monitor.
func (s *ProxyServiceServer) DeleteMonitor(
	ctx context.Context,
	req *connect.Request[labv1.DeleteMonitorRequest],
) (*connect.Response[labv1.DeleteMonitorResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	if err := s.proxySvc.DeleteMonitor(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteMonitorResponse{}), nil
}

// GetMonitorHistory returns historical check results for a monitor.
func (s *ProxyServiceServer) GetMonitorHistory(
	ctx context.Context,
	req *connect.Request[labv1.GetMonitorHistoryRequest],
) (*connect.Response[labv1.GetMonitorHistoryResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	results, err := s.proxySvc.GetMonitorHistory(ctx, req.Msg.Id, int(req.Msg.Limit))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var out []*labv1.UptimeResult
	for _, r := range results {
		out = append(out, uptimeResultToProto(r))
	}

	return connect.NewResponse(&labv1.GetMonitorHistoryResponse{
		Results: out,
	}), nil
}

// GetMonitorStats returns aggregated statistics for a monitor.
func (s *ProxyServiceServer) GetMonitorStats(
	ctx context.Context,
	req *connect.Request[labv1.GetMonitorStatsRequest],
) (*connect.Response[labv1.GetMonitorStatsResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	stats, err := s.proxySvc.GetMonitorStats(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetMonitorStatsResponse{
		Stats: uptimeStatsToProto(stats),
	}), nil
}

// ---- proto conversion helpers ----

func monitorToProto(m *model.UptimeMonitor) *labv1.UptimeMonitor {
	if m == nil {
		return nil
	}
	return &labv1.UptimeMonitor{
		Id:                 m.ID,
		Name:               m.Name,
		Url:                m.URL,
		ProxyHostId:        m.ProxyHostID,
		IntervalSeconds:    int32(m.IntervalSeconds),
		TimeoutSeconds:     int32(m.TimeoutSeconds),
		ExpectedStatusCode: int32(m.ExpectedStatusCode),
		Enabled:            m.Enabled,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func uptimeResultToProto(r *model.UptimeResult) *labv1.UptimeResult {
	if r == nil {
		return nil
	}
	return &labv1.UptimeResult{
		Id:             r.ID,
		MonitorId:      r.MonitorID,
		StatusCode:     int32(r.StatusCode),
		ResponseTimeMs: r.ResponseTimeMs,
		Success:        r.Success,
		Error:          r.Error,
		CheckedAt:      r.CheckedAt,
	}
}

func uptimeStatsToProto(s *model.UptimeStats) *labv1.UptimeStats {
	if s == nil {
		return nil
	}
	out := &labv1.UptimeStats{
		MonitorId:         s.MonitorID,
		Status:            string(s.Status),
		UptimePercent_24H: s.UptimePercent24h,
		UptimePercent_7D:  s.UptimePercent7d,
		AvgResponseMs_24H: s.AvgResponseMs24h,
		LastResult:        uptimeResultToProto(s.LastResult),
	}
	for _, r := range s.RecentResults {
		out.RecentResults = append(out.RecentResults, uptimeResultToProto(r))
	}
	return out
}
