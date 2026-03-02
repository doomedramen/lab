package service

import (
	"context"
	"testing"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// MockNetworkRepository mocks the network repository
type MockNetworkRepository struct {
	networks map[string]*model.VirtualNetwork
}

func NewMockNetworkRepository() *MockNetworkRepository {
	return &MockNetworkRepository{
		networks: make(map[string]*model.VirtualNetwork),
	}
}

func (m *MockNetworkRepository) Create(ctx context.Context, network *model.VirtualNetwork) error {
	m.networks[network.ID] = network
	return nil
}

func (m *MockNetworkRepository) GetByID(ctx context.Context, id string) (*model.VirtualNetwork, error) {
	if n, ok := m.networks[id]; ok {
		return n, nil
	}
	return nil, nil
}

func (m *MockNetworkRepository) List(ctx context.Context, networkType model.VirtualNetworkType, status model.NetworkStatus) ([]*model.VirtualNetwork, error) {
	var result []*model.VirtualNetwork
	for _, n := range m.networks {
		if networkType != "" && n.Type != networkType {
			continue
		}
		if status != "" && n.Status != status {
			continue
		}
		result = append(result, n)
	}
	return result, nil
}

func (m *MockNetworkRepository) Update(ctx context.Context, network *model.VirtualNetwork) error {
	m.networks[network.ID] = network
	return nil
}

func (m *MockNetworkRepository) Delete(ctx context.Context, id string) error {
	delete(m.networks, id)
	return nil
}

func (m *MockNetworkRepository) UpdateInterfaceCount(ctx context.Context, id string, count int) error {
	if n, ok := m.networks[id]; ok {
		n.InterfaceCount = count
	}
	return nil
}

// MockNetworkInterfaceRepository mocks the network interface repository
type MockNetworkInterfaceRepository struct {
	interfaces map[string]*model.NetworkInterface
}

func NewMockNetworkInterfaceRepository() *MockNetworkInterfaceRepository {
	return &MockNetworkInterfaceRepository{
		interfaces: make(map[string]*model.NetworkInterface),
	}
}

func (m *MockNetworkInterfaceRepository) Create(ctx context.Context, iface *model.NetworkInterface) error {
	m.interfaces[iface.ID] = iface
	return nil
}

func (m *MockNetworkInterfaceRepository) GetByID(ctx context.Context, id string) (*model.NetworkInterface, error) {
	if i, ok := m.interfaces[id]; ok {
		return i, nil
	}
	return nil, nil
}

func (m *MockNetworkInterfaceRepository) List(ctx context.Context, networkID string, entityID int, entityType string) ([]*model.NetworkInterface, error) {
	var result []*model.NetworkInterface
	for _, i := range m.interfaces {
		if networkID != "" && i.NetworkID != networkID {
			continue
		}
		if entityID > 0 && i.EntityID != entityID {
			continue
		}
		if entityType != "" && i.EntityType != entityType {
			continue
		}
		result = append(result, i)
	}
	return result, nil
}

func (m *MockNetworkInterfaceRepository) Update(ctx context.Context, iface *model.NetworkInterface) error {
	m.interfaces[iface.ID] = iface
	return nil
}

func (m *MockNetworkInterfaceRepository) Delete(ctx context.Context, id string) error {
	delete(m.interfaces, id)
	return nil
}

// MockFirewallRuleRepository mocks the firewall rule repository
type MockFirewallRuleRepository struct {
	rules map[string]*model.FirewallRule
}

func NewMockFirewallRuleRepository() *MockFirewallRuleRepository {
	return &MockFirewallRuleRepository{
		rules: make(map[string]*model.FirewallRule),
	}
}

func (m *MockFirewallRuleRepository) Create(ctx context.Context, rule *model.FirewallRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockFirewallRuleRepository) GetByID(ctx context.Context, id string) (*model.FirewallRule, error) {
	if r, ok := m.rules[id]; ok {
		return r, nil
	}
	return nil, nil
}

func (m *MockFirewallRuleRepository) List(ctx context.Context, scopeType, scopeID string, enabledOnly bool) ([]*model.FirewallRule, error) {
	var result []*model.FirewallRule
	for _, r := range m.rules {
		if scopeType != "" && r.ScopeType != scopeType {
			continue
		}
		if scopeID != "" && r.ScopeID != scopeID {
			continue
		}
		if enabledOnly && !r.Enabled {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (m *MockFirewallRuleRepository) Update(ctx context.Context, rule *model.FirewallRule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *MockFirewallRuleRepository) Delete(ctx context.Context, id string) error {
	delete(m.rules, id)
	return nil
}

// MockFirewallGroupRepository mocks the firewall group repository
type MockFirewallGroupRepository struct {
	groups map[string]*model.FirewallGroup
}

func NewMockFirewallGroupRepository() *MockFirewallGroupRepository {
	return &MockFirewallGroupRepository{
		groups: make(map[string]*model.FirewallGroup),
	}
}

func (m *MockFirewallGroupRepository) Create(ctx context.Context, g *model.FirewallGroup) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockFirewallGroupRepository) GetByID(ctx context.Context, id string) (*model.FirewallGroup, error) {
	if g, ok := m.groups[id]; ok {
		return g, nil
	}
	return nil, nil
}

func (m *MockFirewallGroupRepository) List(ctx context.Context, scopeType, scopeID string) ([]*model.FirewallGroup, error) {
	var result []*model.FirewallGroup
	for _, g := range m.groups {
		if scopeType != "" && g.ScopeType != scopeType {
			continue
		}
		if scopeID != "" && g.ScopeID != scopeID {
			continue
		}
		result = append(result, g)
	}
	return result, nil
}

func (m *MockFirewallGroupRepository) Update(ctx context.Context, g *model.FirewallGroup) error {
	m.groups[g.ID] = g
	return nil
}

func (m *MockFirewallGroupRepository) Delete(ctx context.Context, id string) error {
	delete(m.groups, id)
	return nil
}

func TestNetworkService_ListNetworks(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Add test data
	testNetwork := &model.VirtualNetwork{
		ID:     "net-1",
		Name:   "test-network",
		Type:   model.VNetBridge,
		Status: model.NetworkStatusActive,
	}
	networkRepo.networks[testNetwork.ID] = testNetwork

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	ctx := context.Background()
	networks, total, err := service.ListNetworks(ctx, labv1.NetworkType_NETWORK_TYPE_UNSPECIFIED, labv1.NetworkStatus_NETWORK_STATUS_UNSPECIFIED)

	if err != nil {
		t.Fatalf("ListNetworks returned error: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 network, got %d", total)
	}

	if len(networks) != 1 {
		t.Errorf("Expected 1 network in result, got %d", len(networks))
	}
}

func TestNetworkService_CreateNetwork(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	ctx := context.Background()
	network, err := service.CreateNetwork(ctx, &labv1.CreateNetworkRequest{
		Name:        "test-network",
		Type:        labv1.NetworkType_NETWORK_TYPE_BRIDGE,
		BridgeName:  "vmbr0",
		Subnet:      "192.168.1.0/24",
		Gateway:     "192.168.1.1",
		DhcpEnabled: true,
		Mtu:         1500,
	})

	if err != nil {
		t.Fatalf("CreateNetwork returned error: %v", err)
	}

	if network == nil {
		t.Fatal("Expected non-nil network")
	}

	if network.Name != "test-network" {
		t.Errorf("Expected name 'test-network', got '%s'", network.Name)
	}

	if !network.DhcpEnabled {
		t.Error("Expected DHCP to be enabled")
	}
}

func TestNetworkService_CreateNetworkInterface(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Create a test network
	testNetwork := &model.VirtualNetwork{
		ID:   "net-1",
		Name: "test-network",
		Type: model.VNetBridge,
	}
	networkRepo.networks[testNetwork.ID] = testNetwork

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	ctx := context.Background()
	iface, err := service.CreateNetworkInterface(ctx, &labv1.CreateVmNetworkInterfaceRequest{
		NetworkId:     "net-1",
		Name:          "eth0",
		MacAddress:    "00:11:22:33:44:55",
		IpAddress:     "192.168.1.100",
		InterfaceType: "vm",
		EntityId:      100,
		EntityType:    "vm",
	})

	if err != nil {
		t.Fatalf("CreateNetworkInterface returned error: %v", err)
	}

	if iface == nil {
		t.Fatal("Expected non-nil interface")
	}

	if iface.Name != "eth0" {
		t.Errorf("Expected name 'eth0', got '%s'", iface.Name)
	}

	if iface.MacAddress != "00:11:22:33:44:55" {
		t.Errorf("Expected MAC '00:11:22:33:44:55', got '%s'", iface.MacAddress)
	}
}

func TestNetworkService_DeleteNetwork(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Create a test network
	testNetwork := &model.VirtualNetwork{
		ID:   "net-1",
		Name: "test-network",
		Type: model.VNetBridge,
	}
	networkRepo.networks[testNetwork.ID] = testNetwork

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	ctx := context.Background()
	err := service.DeleteNetwork(ctx, "net-1", false)

	if err != nil {
		t.Fatalf("DeleteNetwork returned error: %v", err)
	}

	// Verify network was deleted
	_, err = networkRepo.GetByID(ctx, "net-1")
	if err != nil {
		t.Error("Expected network to be deleted")
	}
}

func TestNetworkService_ModelToProto(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	modelNetwork := &model.VirtualNetwork{
		ID:             "net-1",
		Name:           "test-network",
		Type:           model.VNetBridge,
		Status:         model.NetworkStatusActive,
		BridgeName:     "vmbr0",
		VLANID:         100,
		Subnet:         "192.168.1.0/24",
		Gateway:        "192.168.1.1",
		DHCPEnabled:    true,
		Isolated:       false,
		MTU:            1500,
		Description:    "Test network",
		InterfaceCount: 5,
	}

	protoNetwork := service.modelToProto(modelNetwork)

	if protoNetwork == nil {
		t.Fatal("Expected non-nil proto network")
	}

	if protoNetwork.Id != "net-1" {
		t.Errorf("Expected ID 'net-1', got '%s'", protoNetwork.Id)
	}

	if protoNetwork.InterfaceCount != 5 {
		t.Errorf("Expected interface count 5, got %d", protoNetwork.InterfaceCount)
	}

	if !protoNetwork.DhcpEnabled {
		t.Error("Expected DHCP to be enabled")
	}

	if protoNetwork.VlanId != 100 {
		t.Errorf("Expected VLAN ID 100, got %d", protoNetwork.VlanId)
	}
}

func TestNetworkService_InterfaceModelToProto(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	service := NewNetworkService(networkRepo, NewMockNetworkInterfaceRepository(), firewallRepo)

	modelIface := &model.NetworkInterface{
		ID:            "iface-1",
		NetworkID:     "net-1",
		Name:          "eth0",
		MACAddress:    "00:11:22:33:44:55",
		IPAddress:     "192.168.1.100",
		InterfaceType: "vm",
		EntityID:      100,
		EntityType:    "vm",
		Enabled:       true,
	}

	protoIface := service.interfaceModelToProto(modelIface)

	if protoIface == nil {
		t.Fatal("Expected non-nil proto interface")
	}

	if protoIface.Id != "iface-1" {
		t.Errorf("Expected ID 'iface-1', got '%s'", protoIface.Id)
	}

	if protoIface.EntityId != 100 {
		t.Errorf("Expected entity ID 100, got %d", protoIface.EntityId)
	}

	if !protoIface.Enabled {
		t.Error("Expected interface to be enabled")
	}
}

func TestNetworkService_NetworkStatusConversion(t *testing.T) {
	tests := []struct {
		proto    labv1.NetworkStatus
		expected model.NetworkStatus
	}{
		{labv1.NetworkStatus_NETWORK_STATUS_ACTIVE, model.NetworkStatusActive},
		{labv1.NetworkStatus_NETWORK_STATUS_INACTIVE, model.NetworkStatusInactive},
		{labv1.NetworkStatus_NETWORK_STATUS_ERROR, model.NetworkStatusError},
		{labv1.NetworkStatus_NETWORK_STATUS_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		result := protoToModelNetworkStatus(tt.proto)
		if result != tt.expected {
			t.Errorf("protoToModelNetworkStatus(%v) = %v, want %v", tt.proto, result, tt.expected)
		}
	}
}

func TestFirewallService_ListFirewallRules(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Add test data
	testRule := &model.FirewallRule{
		ID:        "rule-1",
		Name:      "Allow SSH",
		Priority:  100,
		Action:    model.FirewallActionAccept,
		Direction: model.FirewallDirectionInbound,
		Enabled:   true,
	}
	firewallRepo.rules[testRule.ID] = testRule

	service := NewFirewallService(firewallRepo, NewMockFirewallGroupRepository(), networkRepo)

	ctx := context.Background()
	rules, total, err := service.ListFirewallRules(ctx, "", "", false)

	if err != nil {
		t.Fatalf("ListFirewallRules returned error: %v", err)
	}

	if total != 1 {
		t.Errorf("Expected 1 rule, got %d", total)
	}

	if len(rules) != 1 {
		t.Errorf("Expected 1 rule in result, got %d", len(rules))
	}
}

func TestFirewallService_CreateFirewallRule(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	service := NewFirewallService(firewallRepo, NewMockFirewallGroupRepository(), networkRepo)

	ctx := context.Background()
	rule, err := service.CreateFirewallRule(ctx, &labv1.CreateFirewallRuleRequest{
		Name:        "Allow SSH",
		Priority:    100,
		Action:      labv1.FirewallAction_FIREWALL_ACTION_ACCEPT,
		Direction:   labv1.FirewallDirection_FIREWALL_DIRECTION_INBOUND,
		DestPort:    "22",
		Protocol:    "tcp",
		Description: "Allow SSH access",
	})

	if err != nil {
		t.Fatalf("CreateFirewallRule returned error: %v", err)
	}

	if rule == nil {
		t.Fatal("Expected non-nil rule")
	}

	if rule.Name != "Allow SSH" {
		t.Errorf("Expected name 'Allow SSH', got '%s'", rule.Name)
	}

	if rule.Priority != 100 {
		t.Errorf("Expected priority 100, got %d", rule.Priority)
	}
}

func TestFirewallService_EnableDisableFirewallRule(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Create a test rule
	testRule := &model.FirewallRule{
		ID:        "rule-1",
		Name:      "test-rule",
		Priority:  100,
		Action:    model.FirewallActionAccept,
		Direction: model.FirewallDirectionInbound,
		Enabled:   true,
	}
	firewallRepo.rules[testRule.ID] = testRule

	service := NewFirewallService(firewallRepo, NewMockFirewallGroupRepository(), networkRepo)

	ctx := context.Background()

	// Disable the rule
	err := service.DisableFirewallRule(ctx, "rule-1")
	if err != nil {
		t.Fatalf("DisableFirewallRule returned error: %v", err)
	}

	// Verify rule is disabled
	rule, _ := firewallRepo.GetByID(ctx, "rule-1")
	if rule.Enabled {
		t.Error("Expected rule to be disabled")
	}

	// Enable the rule
	err = service.EnableFirewallRule(ctx, "rule-1")
	if err != nil {
		t.Fatalf("EnableFirewallRule returned error: %v", err)
	}

	// Verify rule is enabled
	rule, _ = firewallRepo.GetByID(ctx, "rule-1")
	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}
}

func TestFirewallService_DeleteFirewallRule(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	// Create a test rule
	testRule := &model.FirewallRule{
		ID:        "rule-1",
		Name:      "test-rule",
		Priority:  100,
		Action:    model.FirewallActionAccept,
		Direction: model.FirewallDirectionInbound,
		Enabled:   true,
	}
	firewallRepo.rules[testRule.ID] = testRule

	service := NewFirewallService(firewallRepo, NewMockFirewallGroupRepository(), networkRepo)

	ctx := context.Background()
	err := service.DeleteFirewallRule(ctx, "rule-1")

	if err != nil {
		t.Fatalf("DeleteFirewallRule returned error: %v", err)
	}

	// Verify rule was deleted
	_, err = firewallRepo.GetByID(ctx, "rule-1")
	if err != nil {
		t.Error("Expected rule to be deleted")
	}
}

func TestFirewallService_ModelToProto(t *testing.T) {
	networkRepo := NewMockNetworkRepository()
	_ = NewMockNetworkInterfaceRepository()
	firewallRepo := NewMockFirewallRuleRepository()

	service := NewFirewallService(firewallRepo, NewMockFirewallGroupRepository(), networkRepo)

	modelRule := &model.FirewallRule{
		ID:          "rule-1",
		Name:        "Allow SSH",
		Priority:    100,
		Action:      model.FirewallActionAccept,
		Direction:   model.FirewallDirectionInbound,
		SourceCIDR:  "0.0.0.0/0",
		DestCIDR:    "192.168.1.0/24",
		Protocol:    "tcp",
		DestPort:    "22",
		Enabled:     true,
		Log:         false,
		Description: "Allow SSH access",
		ScopeType:   "global",
	}

	protoRule := service.modelToProto(modelRule)

	if protoRule == nil {
		t.Fatal("Expected non-nil proto rule")
	}

	if protoRule.Id != "rule-1" {
		t.Errorf("Expected ID 'rule-1', got '%s'", protoRule.Id)
	}

	if protoRule.Priority != 100 {
		t.Errorf("Expected priority 100, got %d", protoRule.Priority)
	}

	if !protoRule.Enabled {
		t.Error("Expected rule to be enabled")
	}
}

func TestFirewallService_FirewallActionConversion(t *testing.T) {
	tests := []struct {
		proto    labv1.FirewallAction
		expected model.FirewallAction
	}{
		{labv1.FirewallAction_FIREWALL_ACTION_ACCEPT, model.FirewallActionAccept},
		{labv1.FirewallAction_FIREWALL_ACTION_DROP, model.FirewallActionDrop},
		{labv1.FirewallAction_FIREWALL_ACTION_REJECT, model.FirewallActionReject},
		{labv1.FirewallAction_FIREWALL_ACTION_LOG, model.FirewallActionLog},
		{labv1.FirewallAction_FIREWALL_ACTION_UNSPECIFIED, model.FirewallActionAccept},
	}

	for _, tt := range tests {
		result := protoToModelFirewallAction(tt.proto)
		if result != tt.expected {
			t.Errorf("protoToModelFirewallAction(%v) = %v, want %v", tt.proto, result, tt.expected)
		}
	}
}

func TestFirewallService_FirewallDirectionConversion(t *testing.T) {
	tests := []struct {
		proto    labv1.FirewallDirection
		expected model.FirewallDirection
	}{
		{labv1.FirewallDirection_FIREWALL_DIRECTION_INBOUND, model.FirewallDirectionInbound},
		{labv1.FirewallDirection_FIREWALL_DIRECTION_OUTBOUND, model.FirewallDirectionOutbound},
		{labv1.FirewallDirection_FIREWALL_DIRECTION_BOTH, model.FirewallDirectionBoth},
		{labv1.FirewallDirection_FIREWALL_DIRECTION_UNSPECIFIED, model.FirewallDirectionBoth},
	}

	for _, tt := range tests {
		result := protoToModelFirewallDirection(tt.proto)
		if result != tt.expected {
			t.Errorf("protoToModelFirewallDirection(%v) = %v, want %v", tt.proto, result, tt.expected)
		}
	}
}

// --- Firewall Group tests ---

func TestFirewallService_CreateFirewallGroup(t *testing.T) {
	firewallRepo := NewMockFirewallRuleRepository()
	groupRepo := NewMockFirewallGroupRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, groupRepo, networkRepo)

	ctx := context.Background()
	group, err := svc.CreateFirewallGroup(ctx, &labv1.CreateFirewallGroupRequest{
		Name:        "web-rules",
		RuleIds:     []string{"rule-1", "rule-2"},
		ScopeType:   "global",
		Description: "Rules for web traffic",
	})

	if err != nil {
		t.Fatalf("CreateFirewallGroup returned error: %v", err)
	}
	if group == nil {
		t.Fatal("Expected non-nil group")
	}
	if group.Name != "web-rules" {
		t.Errorf("Name = %q, want web-rules", group.Name)
	}
	if len(group.RuleIds) != 2 {
		t.Errorf("Expected 2 rule IDs, got %d", len(group.RuleIds))
	}
}

func TestFirewallService_ListFirewallGroups(t *testing.T) {
	firewallRepo := NewMockFirewallRuleRepository()
	groupRepo := NewMockFirewallGroupRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, groupRepo, networkRepo)

	ctx := context.Background()

	// Seed two groups
	for _, name := range []string{"group-a", "group-b"} {
		if _, err := svc.CreateFirewallGroup(ctx, &labv1.CreateFirewallGroupRequest{Name: name}); err != nil {
			t.Fatalf("CreateFirewallGroup(%s): %v", name, err)
		}
	}

	groups, total, err := svc.ListFirewallGroups(ctx, "", "")
	if err != nil {
		t.Fatalf("ListFirewallGroups returned error: %v", err)
	}
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if len(groups) != 2 {
		t.Errorf("Expected 2 groups, got %d", len(groups))
	}
}

func TestFirewallService_UpdateFirewallGroup(t *testing.T) {
	firewallRepo := NewMockFirewallRuleRepository()
	groupRepo := NewMockFirewallGroupRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, groupRepo, networkRepo)

	ctx := context.Background()
	created, err := svc.CreateFirewallGroup(ctx, &labv1.CreateFirewallGroupRequest{
		Name:    "original",
		RuleIds: []string{"r1"},
	})
	if err != nil {
		t.Fatalf("CreateFirewallGroup: %v", err)
	}

	updated, err := svc.UpdateFirewallGroup(ctx, &labv1.UpdateFirewallGroupRequest{
		Id:      created.Id,
		Name:    "updated",
		RuleIds: []string{"r1", "r2", "r3"},
	})
	if err != nil {
		t.Fatalf("UpdateFirewallGroup: %v", err)
	}
	if updated.Name != "updated" {
		t.Errorf("Name = %q, want updated", updated.Name)
	}
	if len(updated.RuleIds) != 3 {
		t.Errorf("Expected 3 rule IDs, got %d", len(updated.RuleIds))
	}
}

func TestFirewallService_DeleteFirewallGroup(t *testing.T) {
	firewallRepo := NewMockFirewallRuleRepository()
	groupRepo := NewMockFirewallGroupRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, groupRepo, networkRepo)

	ctx := context.Background()
	created, err := svc.CreateFirewallGroup(ctx, &labv1.CreateFirewallGroupRequest{Name: "to-delete"})
	if err != nil {
		t.Fatalf("CreateFirewallGroup: %v", err)
	}

	if err := svc.DeleteFirewallGroup(ctx, created.Id); err != nil {
		t.Fatalf("DeleteFirewallGroup: %v", err)
	}

	// List should now be empty
	groups, total, err := svc.ListFirewallGroups(ctx, "", "")
	if err != nil {
		t.Fatalf("ListFirewallGroups: %v", err)
	}
	if total != 0 || len(groups) != 0 {
		t.Errorf("Expected 0 groups after delete, got %d", len(groups))
	}
}

func TestFirewallService_DeleteFirewallGroup_NotFound(t *testing.T) {
	firewallRepo := NewMockFirewallRuleRepository()
	groupRepo := NewMockFirewallGroupRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, groupRepo, networkRepo)

	ctx := context.Background()
	err := svc.DeleteFirewallGroup(ctx, "nonexistent-id")
	if err == nil {
		t.Error("Expected error for nonexistent group, got nil")
	}
}

func TestFirewallService_ListFirewallGroups_NoRepo(t *testing.T) {
	// When groupRepo is nil, ListFirewallGroups should return empty list, not error
	firewallRepo := NewMockFirewallRuleRepository()
	networkRepo := NewMockNetworkRepository()
	svc := NewFirewallService(firewallRepo, nil, networkRepo)

	ctx := context.Background()
	groups, total, err := svc.ListFirewallGroups(ctx, "", "")
	if err != nil {
		t.Fatalf("ListFirewallGroups with nil repo returned error: %v", err)
	}
	if total != 0 || len(groups) != 0 {
		t.Errorf("Expected 0 groups with nil repo, got %d", len(groups))
	}
}
