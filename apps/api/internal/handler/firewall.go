package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// FirewallServiceServer implements the FirewallService Connect RPC server
type FirewallServiceServer struct {
	firewallService *service.FirewallService
}

// NewFirewallServiceServer creates a new firewall service server
func NewFirewallServiceServer(firewallService *service.FirewallService) *FirewallServiceServer {
	return &FirewallServiceServer{firewallService: firewallService}
}

var _ labv1connect.FirewallServiceHandler = (*FirewallServiceServer)(nil)

// ListFirewallRules lists firewall rules
func (s *FirewallServiceServer) ListFirewallRules(
	ctx context.Context,
	req *connect.Request[labv1.ListFirewallRulesRequest],
) (*connect.Response[labv1.ListFirewallRulesResponse], error) {
	rules, total, err := s.firewallService.ListFirewallRules(
		ctx,
		req.Msg.ScopeType,
		req.Msg.ScopeId,
		req.Msg.EnabledOnly,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListFirewallRulesResponse{
		Rules: rules,
		Total: total,
	}), nil
}

// GetFirewallRule gets a firewall rule
func (s *FirewallServiceServer) GetFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.GetFirewallRuleRequest],
) (*connect.Response[labv1.GetFirewallRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule ID is required"))
	}

	rule, err := s.firewallService.GetFirewallRule(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetFirewallRuleResponse{
		Rule: rule,
	}), nil
}

// CreateFirewallRule creates a firewall rule
func (s *FirewallServiceServer) CreateFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.CreateFirewallRuleRequest],
) (*connect.Response[labv1.CreateFirewallRuleResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule name is required"))
	}

	rule, err := s.firewallService.CreateFirewallRule(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateFirewallRuleResponse{
		Rule: rule,
	}), nil
}

// UpdateFirewallRule updates a firewall rule
func (s *FirewallServiceServer) UpdateFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.UpdateFirewallRuleRequest],
) (*connect.Response[labv1.UpdateFirewallRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule ID is required"))
	}

	rule, err := s.firewallService.UpdateFirewallRule(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateFirewallRuleResponse{
		Rule: rule,
	}), nil
}

// DeleteFirewallRule deletes a firewall rule
func (s *FirewallServiceServer) DeleteFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.DeleteFirewallRuleRequest],
) (*connect.Response[labv1.DeleteFirewallRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule ID is required"))
	}

	if err := s.firewallService.DeleteFirewallRule(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteFirewallRuleResponse{}), nil
}

// EnableFirewallRule enables a firewall rule
func (s *FirewallServiceServer) EnableFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.EnableFirewallRuleRequest],
) (*connect.Response[labv1.EnableFirewallRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule ID is required"))
	}

	if err := s.firewallService.EnableFirewallRule(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.EnableFirewallRuleResponse{}), nil
}

// DisableFirewallRule disables a firewall rule
func (s *FirewallServiceServer) DisableFirewallRule(
	ctx context.Context,
	req *connect.Request[labv1.DisableFirewallRuleRequest],
) (*connect.Response[labv1.DisableFirewallRuleResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("rule ID is required"))
	}

	if err := s.firewallService.DisableFirewallRule(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DisableFirewallRuleResponse{}), nil
}

// ListFirewallGroups lists firewall groups
func (s *FirewallServiceServer) ListFirewallGroups(
	ctx context.Context,
	req *connect.Request[labv1.ListFirewallGroupsRequest],
) (*connect.Response[labv1.ListFirewallGroupsResponse], error) {
	groups, total, err := s.firewallService.ListFirewallGroups(
		ctx,
		req.Msg.ScopeType,
		req.Msg.ScopeId,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListFirewallGroupsResponse{
		Groups: groups,
		Total:  total,
	}), nil
}

// CreateFirewallGroup creates a firewall group
func (s *FirewallServiceServer) CreateFirewallGroup(
	ctx context.Context,
	req *connect.Request[labv1.CreateFirewallGroupRequest],
) (*connect.Response[labv1.CreateFirewallGroupResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("group name is required"))
	}

	group, err := s.firewallService.CreateFirewallGroup(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateFirewallGroupResponse{
		Group: group,
	}), nil
}

// UpdateFirewallGroup updates a firewall group
func (s *FirewallServiceServer) UpdateFirewallGroup(
	ctx context.Context,
	req *connect.Request[labv1.UpdateFirewallGroupRequest],
) (*connect.Response[labv1.UpdateFirewallGroupResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("group ID is required"))
	}

	group, err := s.firewallService.UpdateFirewallGroup(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateFirewallGroupResponse{
		Group: group,
	}), nil
}

// DeleteFirewallGroup deletes a firewall group
func (s *FirewallServiceServer) DeleteFirewallGroup(
	ctx context.Context,
	req *connect.Request[labv1.DeleteFirewallGroupRequest],
) (*connect.Response[labv1.DeleteFirewallGroupResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("group ID is required"))
	}

	if err := s.firewallService.DeleteFirewallGroup(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteFirewallGroupResponse{}), nil
}

// GetFirewallStatus gets firewall status
func (s *FirewallServiceServer) GetFirewallStatus(
	ctx context.Context,
	req *connect.Request[labv1.GetFirewallStatusRequest],
) (*connect.Response[labv1.GetFirewallStatusResponse], error) {
	status, err := s.firewallService.GetFirewallStatus(ctx, req.Msg.ScopeType, req.Msg.ScopeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(status), nil
}

// EnableFirewall enables the firewall
func (s *FirewallServiceServer) EnableFirewall(
	ctx context.Context,
	req *connect.Request[labv1.EnableFirewallRequest],
) (*connect.Response[labv1.EnableFirewallResponse], error) {
	if err := s.firewallService.EnableFirewall(ctx, req.Msg.ScopeType, req.Msg.ScopeId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.EnableFirewallResponse{}), nil
}

// DisableFirewall disables the firewall
func (s *FirewallServiceServer) DisableFirewall(
	ctx context.Context,
	req *connect.Request[labv1.DisableFirewallRequest],
) (*connect.Response[labv1.DisableFirewallResponse], error) {
	if err := s.firewallService.DisableFirewall(ctx, req.Msg.ScopeType, req.Msg.ScopeId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DisableFirewallResponse{}), nil
}
