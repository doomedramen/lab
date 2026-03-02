package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

func TestFirewallHandler_GetFirewallRule_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.GetFirewallRule(context.Background(), connect.NewRequest(&labv1.GetFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_CreateFirewallRule_MissingName(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.CreateFirewallRule(context.Background(), connect.NewRequest(&labv1.CreateFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_UpdateFirewallRule_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.UpdateFirewallRule(context.Background(), connect.NewRequest(&labv1.UpdateFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_DeleteFirewallRule_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.DeleteFirewallRule(context.Background(), connect.NewRequest(&labv1.DeleteFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_EnableFirewallRule_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.EnableFirewallRule(context.Background(), connect.NewRequest(&labv1.EnableFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_DisableFirewallRule_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.DisableFirewallRule(context.Background(), connect.NewRequest(&labv1.DisableFirewallRuleRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_CreateFirewallGroup_MissingName(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.CreateFirewallGroup(context.Background(), connect.NewRequest(&labv1.CreateFirewallGroupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_UpdateFirewallGroup_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.UpdateFirewallGroup(context.Background(), connect.NewRequest(&labv1.UpdateFirewallGroupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestFirewallHandler_DeleteFirewallGroup_MissingId(t *testing.T) {
	h := &FirewallServiceServer{}
	_, err := h.DeleteFirewallGroup(context.Background(), connect.NewRequest(&labv1.DeleteFirewallGroupRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}
