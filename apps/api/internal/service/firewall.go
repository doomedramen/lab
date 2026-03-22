package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

// FirewallService handles firewall rule management
type FirewallService struct {
	ruleRepo    repository.FirewallRuleRepository
	groupRepo   repository.FirewallGroupRepository
	networkRepo repository.NetworkRepository
}

// NewFirewallService creates a new firewall service
func NewFirewallService(
	ruleRepo repository.FirewallRuleRepository,
	groupRepo repository.FirewallGroupRepository,
	networkRepo repository.NetworkRepository,
) *FirewallService {
	return &FirewallService{
		ruleRepo:    ruleRepo,
		groupRepo:   groupRepo,
		networkRepo: networkRepo,
	}
}

// ListFirewallRules returns firewall rules with optional filters
func (s *FirewallService) ListFirewallRules(ctx context.Context, scopeType, scopeID string, enabledOnly bool) ([]*labv1.FirewallRule, int32, error) {
	rules, err := s.ruleRepo.List(ctx, scopeType, scopeID, enabledOnly)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list firewall rules: %w", err)
	}

	var protoRules []*labv1.FirewallRule
	for _, r := range rules {
		protoRules = append(protoRules, s.modelToProto(r))
	}

	return protoRules, int32(len(protoRules)), nil
}

// GetFirewallRule returns details of a specific rule
func (s *FirewallService) GetFirewallRule(ctx context.Context, id string) (*labv1.FirewallRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("firewall rule not found: %s", id)
	}
	return s.modelToProto(rule), nil
}

// CreateFirewallRule creates a new firewall rule
func (s *FirewallService) CreateFirewallRule(ctx context.Context, req *labv1.CreateFirewallRuleRequest) (*labv1.FirewallRule, error) {
	rule := &model.FirewallRule{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Priority:    int(req.Priority),
		Action:      protoToModelFirewallAction(req.Action),
		Direction:   protoToModelFirewallDirection(req.Direction),
		SourceCIDR:  req.SourceCidr,
		DestCIDR:    req.DestCidr,
		Protocol:    req.Protocol,
		SourcePort:  req.SourcePort,
		DestPort:    req.DestPort,
		Interface:   req.Interface,
		Enabled:     true,
		Log:         req.Log,
		Description: req.Description,
		
		ScopeID:     req.ScopeId,
		CreatedAt:   time.Now().Format(time.RFC3339),
		UpdatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := s.ruleRepo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create firewall rule: %w", err)
	}

	// Apply rule to iptables/nftables
	if err := s.applyRule(rule); err != nil {
		slog.Warn("Failed to apply firewall rule to system", "error", err, "rule", rule.ID)
	}

	return s.modelToProto(rule), nil
}

// UpdateFirewallRule updates an existing rule
func (s *FirewallService) UpdateFirewallRule(ctx context.Context, req *labv1.UpdateFirewallRuleRequest) (*labv1.FirewallRule, error) {
	rule, err := s.ruleRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall rule: %w", err)
	}
	if rule == nil {
		return nil, fmt.Errorf("firewall rule not found: %s", req.Id)
	}

	// Update fields
	if req.Name != "" {
		rule.Name = req.Name
	}
	rule.Priority = int(req.Priority)
	if req.Action != labv1.FirewallAction_FIREWALL_ACTION_UNSPECIFIED {
		rule.Action = protoToModelFirewallAction(req.Action)
	}
	if req.Direction != labv1.FirewallDirection_FIREWALL_DIRECTION_UNSPECIFIED {
		rule.Direction = protoToModelFirewallDirection(req.Direction)
	}
	rule.SourceCIDR = req.SourceCidr
	rule.DestCIDR = req.DestCidr
	rule.Protocol = req.Protocol
	rule.SourcePort = req.SourcePort
	rule.DestPort = req.DestPort
	rule.Log = req.Log
	if req.Description != "" {
		rule.Description = req.Description
	}
	rule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update firewall rule: %w", err)
	}

	// Apply updated rule
	if err := s.applyRule(rule); err != nil {
		slog.Warn("Failed to apply updated firewall rule", "error", err, "rule", rule.ID)
	}

	return s.modelToProto(rule), nil
}

// DeleteFirewallRule deletes a rule
func (s *FirewallService) DeleteFirewallRule(ctx context.Context, id string) error {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get firewall rule: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("firewall rule not found: %s", id)
	}

	// Remove rule from system
	if err := s.removeRule(rule); err != nil {
		slog.Warn("Failed to remove firewall rule from system", "error", err, "rule", rule.ID)
	}

	if err := s.ruleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete firewall rule: %w", err)
	}

	return nil
}

// EnableFirewallRule enables a rule
func (s *FirewallService) EnableFirewallRule(ctx context.Context, id string) error {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get firewall rule: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("firewall rule not found: %s", id)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return fmt.Errorf("failed to enable firewall rule: %w", err)
	}

	if err := s.applyRule(rule); err != nil {
		slog.Warn("Failed to apply enabled firewall rule", "error", err, "rule", rule.ID)
	}

	return nil
}

// DisableFirewallRule disables a rule
func (s *FirewallService) DisableFirewallRule(ctx context.Context, id string) error {
	rule, err := s.ruleRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get firewall rule: %w", err)
	}
	if rule == nil {
		return fmt.Errorf("firewall rule not found: %s", id)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := s.ruleRepo.Update(ctx, rule); err != nil {
		return fmt.Errorf("failed to disable firewall rule: %w", err)
	}

	if err := s.removeRule(rule); err != nil {
		slog.Warn("Failed to remove disabled firewall rule", "error", err, "rule", rule.ID)
	}

	return nil
}

// ListFirewallGroups returns firewall groups with optional scope filters
func (s *FirewallService) ListFirewallGroups(ctx context.Context, scopeType, scopeID string) ([]*labv1.FirewallGroup, int32, error) {
	if s.groupRepo == nil {
		return []*labv1.FirewallGroup{}, 0, nil
	}

	groups, err := s.groupRepo.List(ctx, scopeType, scopeID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list firewall groups: %w", err)
	}

	var protoGroups []*labv1.FirewallGroup
	for _, g := range groups {
		protoGroups = append(protoGroups, s.groupModelToProto(g))
	}

	return protoGroups, int32(len(protoGroups)), nil
}

// CreateFirewallGroup creates a firewall group
func (s *FirewallService) CreateFirewallGroup(ctx context.Context, req *labv1.CreateFirewallGroupRequest) (*labv1.FirewallGroup, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("firewall group storage not available")
	}

	group := &model.FirewallGroup{
		ID:          uuid.New().String(),
		Name:        req.Name,
		RuleIDs:     req.RuleIds,
		ScopeType:   req.ScopeType,
		ScopeID:     req.ScopeId,
		Description: req.Description,
		CreatedAt:   time.Now().Format(time.RFC3339),
	}

	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create firewall group: %w", err)
	}

	return s.groupModelToProto(group), nil
}

// UpdateFirewallGroup updates a firewall group's name, description, and rule set
func (s *FirewallService) UpdateFirewallGroup(ctx context.Context, req *labv1.UpdateFirewallGroupRequest) (*labv1.FirewallGroup, error) {
	if s.groupRepo == nil {
		return nil, fmt.Errorf("firewall group storage not available")
	}

	group, err := s.groupRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get firewall group: %w", err)
	}
	if group == nil {
		return nil, fmt.Errorf("firewall group not found: %s", req.Id)
	}

	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.RuleIds != nil {
		group.RuleIDs = req.RuleIds
	}

	if err := s.groupRepo.Update(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to update firewall group: %w", err)
	}

	return s.groupModelToProto(group), nil
}

// DeleteFirewallGroup deletes a firewall group
func (s *FirewallService) DeleteFirewallGroup(ctx context.Context, id string) error {
	if s.groupRepo == nil {
		return fmt.Errorf("firewall group storage not available")
	}

	group, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get firewall group: %w", err)
	}
	if group == nil {
		return fmt.Errorf("firewall group not found: %s", id)
	}

	if err := s.groupRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete firewall group: %w", err)
	}

	return nil
}

// GetFirewallStatus returns firewall status
func (s *FirewallService) GetFirewallStatus(ctx context.Context, scopeType, scopeID string) (*labv1.GetFirewallStatusResponse, error) {
	enabled, err := s.isFirewallEnabled()
	if err != nil {
		enabled = false
	}

	ruleCount, err := s.getRuleCount(scopeType, scopeID)
	if err != nil {
		ruleCount = 0
	}

	activeConns := s.getActiveConnections()
	packetsProcessed, packetsDropped := s.getIptablesCounters()

	return &labv1.GetFirewallStatusResponse{
		Enabled:           enabled,
		RuleCount:         int32(ruleCount),
		ActiveConnections: int32(activeConns),
		PacketsProcessed:  int64(packetsProcessed),
		PacketsDropped:    int64(packetsDropped),
	}, nil
}

// EnableFirewall enables the firewall
func (s *FirewallService) EnableFirewall(ctx context.Context, scopeType, scopeID string) error {
	// Enable iptables/nftables
	cmd := exec.Command("iptables", "-P", "INPUT", "DROP")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable firewall: %w", err)
	}
	return nil
}

// DisableFirewall disables the firewall
func (s *FirewallService) DisableFirewall(ctx context.Context, scopeType, scopeID string) error {
	// Disable iptables/nftables
	cmd := exec.Command("iptables", "-P", "INPUT", "ACCEPT")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to disable firewall: %w", err)
	}
	return nil
}

// groupModelToProto converts model.FirewallGroup to labv1.FirewallGroup
func (s *FirewallService) groupModelToProto(g *model.FirewallGroup) *labv1.FirewallGroup {
	if g == nil {
		return nil
	}
	return &labv1.FirewallGroup{
		Id:          g.ID,
		Name:        g.Name,
		RuleIds:     g.RuleIDs,
		ScopeType:   g.ScopeType,
		ScopeId:     g.ScopeID,
		Description: g.Description,
		CreatedAt:   g.CreatedAt,
	}
}

// modelToProto converts model.FirewallRule to labv1.FirewallRule
func (s *FirewallService) modelToProto(rule *model.FirewallRule) *labv1.FirewallRule {
	if rule == nil {
		return nil
	}

	return &labv1.FirewallRule{
		Id:          rule.ID,
		Name:        rule.Name,
		Priority:    int32(rule.Priority),
		Action:      modelFirewallActionToProto(rule.Action),
		Direction:   modelFirewallDirectionToProto(rule.Direction),
		SourceCidr:  rule.SourceCIDR,
		DestCidr:    rule.DestCIDR,
		Protocol:    rule.Protocol,
		SourcePort:  rule.SourcePort,
		DestPort:    rule.DestPort,
		Interface:   rule.Interface,
		Enabled:     rule.Enabled,
		Log:         rule.Log,
		Description: rule.Description,
		
		
		CreatedAt:   rule.CreatedAt,
		UpdatedAt:   rule.UpdatedAt,
	}
}

// applyRule applies a firewall rule to the system
func (s *FirewallService) applyRule(rule *model.FirewallRule) error {
	if !rule.Enabled {
		return nil
	}

	// Build iptables command
	args := []string{"-A", s.getChainForDirection(rule.Direction)}

	if rule.SourceCIDR != "" {
		args = append(args, "-s", rule.SourceCIDR)
	}
	if rule.DestCIDR != "" {
		args = append(args, "-d", rule.DestCIDR)
	}
	if rule.Protocol != "" && rule.Protocol != "any" {
		args = append(args, "-p", rule.Protocol)
	}
	if rule.SourcePort != "" {
		args = append(args, "--sport", rule.SourcePort)
	}
	if rule.DestPort != "" {
		args = append(args, "--dport", rule.DestPort)
	}
	if rule.Interface != "" {
		args = append(args, "-i", rule.Interface)
	}
	if rule.Log {
		args = append(args, "-j", "LOG", "--log-prefix", fmt.Sprintf("[%s] ", rule.Name))
	}

	switch rule.Action {
	case model.FirewallActionAccept:
		args = append(args, "-j", "ACCEPT")
	case model.FirewallActionDrop:
		args = append(args, "-j", "DROP")
	case model.FirewallActionReject:
		args = append(args, "-j", "REJECT")
	case model.FirewallActionLog:
		args = append(args, "-j", "LOG")
	}

	cmd := exec.Command("iptables", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables command failed: %w, output: %s", err, string(output))
	}

	return nil
}

// removeRule removes a firewall rule from the system
func (s *FirewallService) removeRule(rule *model.FirewallRule) error {
	// Delete by rule content (simplified - in production would track rule IDs)
	args := []string{"-D", s.getChainForDirection(rule.Direction)}

	if rule.SourceCIDR != "" {
		args = append(args, "-s", rule.SourceCIDR)
	}
	if rule.DestCIDR != "" {
		args = append(args, "-d", rule.DestCIDR)
	}
	if rule.Protocol != "" && rule.Protocol != "any" {
		args = append(args, "-p", rule.Protocol)
	}
	if rule.DestPort != "" {
		args = append(args, "--dport", rule.DestPort)
	}

	switch rule.Action {
	case model.FirewallActionAccept:
		args = append(args, "-j", "ACCEPT")
	case model.FirewallActionDrop:
		args = append(args, "-j", "DROP")
	case model.FirewallActionReject:
		args = append(args, "-j", "REJECT")
	}

	cmd := exec.Command("iptables", args...)
	_ = cmd.Run() // Ignore errors - rule may not exist

	return nil
}

// getChainForDirection returns the iptables chain for a direction
func (s *FirewallService) getChainForDirection(direction model.FirewallDirection) string {
	switch direction {
	case model.FirewallDirectionInbound:
		return "INPUT"
	case model.FirewallDirectionOutbound:
		return "OUTPUT"
	default:
		return "INPUT"
	}
}

// isFirewallEnabled checks if the firewall is enabled
func (s *FirewallService) isFirewallEnabled() (bool, error) {
	cmd := exec.Command("iptables", "-L", "INPUT", "-n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, err
	}

	// Check if default policy is DROP or REJECT
	return strings.Contains(string(output), "policy DROP") || strings.Contains(string(output), "policy REJECT"), nil
}

// getRuleCount returns the number of rules for a scope
func (s *FirewallService) getRuleCount(scopeType, scopeID string) (int, error) {
	ctx := context.Background()
	rules, err := s.ruleRepo.List(ctx, scopeType, scopeID, false)
	if err != nil {
		return 0, err
	}
	return len(rules), nil
}

// Helper functions for type conversion
func protoToModelFirewallAction(a labv1.FirewallAction) model.FirewallAction {
	switch a {
	case labv1.FirewallAction_FIREWALL_ACTION_ACCEPT:
		return model.FirewallActionAccept
	case labv1.FirewallAction_FIREWALL_ACTION_DROP:
		return model.FirewallActionDrop
	case labv1.FirewallAction_FIREWALL_ACTION_REJECT:
		return model.FirewallActionReject
	case labv1.FirewallAction_FIREWALL_ACTION_LOG:
		return model.FirewallActionLog
	default:
		return model.FirewallActionAccept
	}
}

func modelFirewallActionToProto(a model.FirewallAction) labv1.FirewallAction {
	switch a {
	case model.FirewallActionAccept:
		return labv1.FirewallAction_FIREWALL_ACTION_ACCEPT
	case model.FirewallActionDrop:
		return labv1.FirewallAction_FIREWALL_ACTION_DROP
	case model.FirewallActionReject:
		return labv1.FirewallAction_FIREWALL_ACTION_REJECT
	case model.FirewallActionLog:
		return labv1.FirewallAction_FIREWALL_ACTION_LOG
	default:
		return labv1.FirewallAction_FIREWALL_ACTION_ACCEPT
	}
}

func protoToModelFirewallDirection(d labv1.FirewallDirection) model.FirewallDirection {
	switch d {
	case labv1.FirewallDirection_FIREWALL_DIRECTION_INBOUND:
		return model.FirewallDirectionInbound
	case labv1.FirewallDirection_FIREWALL_DIRECTION_OUTBOUND:
		return model.FirewallDirectionOutbound
	case labv1.FirewallDirection_FIREWALL_DIRECTION_BOTH:
		return model.FirewallDirectionBoth
	default:
		return model.FirewallDirectionBoth
	}
}

func modelFirewallDirectionToProto(d model.FirewallDirection) labv1.FirewallDirection {
	switch d {
	case model.FirewallDirectionInbound:
		return labv1.FirewallDirection_FIREWALL_DIRECTION_INBOUND
	case model.FirewallDirectionOutbound:
		return labv1.FirewallDirection_FIREWALL_DIRECTION_OUTBOUND
	case model.FirewallDirectionBoth:
		return labv1.FirewallDirection_FIREWALL_DIRECTION_BOTH
	default:
		return labv1.FirewallDirection_FIREWALL_DIRECTION_BOTH
	}
}

// getActiveConnections returns the number of active connections from conntrack
func (s *FirewallService) getActiveConnections() int {
	// Try to read from /proc/net/nf_conntrack
	conntrackPath := "/proc/net/nf_conntrack"
	file, err := os.Open(conntrackPath)
	if err != nil {
		// conntrack not available or not readable
		slog.Debug("conntrack not available", "error", err)
		return 0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		slog.Debug("error reading conntrack", "error", err)
		return 0
	}

	return count
}

// getIptablesCounters returns packet counters from iptables
// Returns (packetsProcessed, packetsDropped)
func (s *FirewallService) getIptablesCounters() (uint64, uint64) {
	// Get iptables counters using iptables -L -v -n
	cmd := exec.Command("iptables", "-L", "-v", "-n")
	output, err := cmd.Output()
	if err != nil {
		slog.Debug("failed to get iptables counters", "error", err)
		return 0, 0
	}

	var packetsProcessed, packetsDropped uint64

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Skip header lines
		if strings.HasPrefix(line, "Chain") || strings.HasPrefix(line, "target") || line == "" {
			continue
		}

		// Parse iptables output: Chain INPUT (policy ACCEPT 1234 packets, 56789 bytes)
		if strings.Contains(line, "policy") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "packets" && i > 0 {
					if count, err := strconv.ParseUint(parts[i-1], 10, 64); err == nil {
						// Check if this is INPUT chain (processed packets)
						if strings.Contains(line, "INPUT") {
							packetsProcessed += count
						}
						// Check for DROP policy (dropped packets)
						if strings.Contains(line, "policy DROP") {
							packetsDropped += count
						}
					}
				}
			}
		}

		// Parse rule lines: num   pkts bytes target     prot opt in     out     source               destination
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			if pkts, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
				packetsProcessed += pkts
				// Check if target is DROP or REJECT
				if len(fields) >= 4 && (fields[2] == "DROP" || fields[2] == "REJECT") {
					packetsDropped += pkts
				}
			}
		}
	}

	return packetsProcessed, packetsDropped
}
