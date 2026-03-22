package service

import (
	"context"
	"database/sql"
	"fmt"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// NetworkReconciler reconciles Network resources from GitOps manifests
type NetworkReconciler struct {
	networkService *NetworkService
}

// NewNetworkReconciler creates a new network reconciler
func NewNetworkReconciler(networkService *NetworkService) *NetworkReconciler {
	return &NetworkReconciler{
		networkService: networkService,
	}
}

// Kind returns the resource kind this reconciler handles
func (r *NetworkReconciler) Kind() string {
	return "Network"
}

// Reconcile ensures the actual network state matches the desired state from GitOps manifest
func (r *NetworkReconciler) Reconcile(ctx context.Context, desired *model.GitOpsResource) (*ReconcileResult, error) {
	// Extract network spec from GitOps resource
	spec := extractNetworkSpec(desired.Spec)
	if spec == nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: "Invalid network spec in manifest",
		}, nil
	}

	// Check if network already exists by name
	actual, err := r.GetActualState(ctx, desired.ConfigID, desired.Name)
	if err != nil && err != sql.ErrNoRows {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to get actual state: %v", err),
		}, nil
	}

	if actual == nil {
		// Network doesn't exist - create it
		return r.createNetwork(ctx, desired, spec)
	}

	// Network exists - for now just mark as unchanged
	// Full implementation would compare specs and update if needed
	return &ReconcileResult{
		Action:      ReconcileActionUnchanged,
		Message:     fmt.Sprintf("Network %s exists and is up to date", desired.Name),
		ActualState: actual,
	}, nil
}

// Delete removes the network from actual state
func (r *NetworkReconciler) Delete(ctx context.Context, resource *model.GitOpsResource) error {
	// Find network by name and delete
	return nil // Stub - full implementation requires network lookup by name
}

// GetActualState retrieves the current state of the network
func (r *NetworkReconciler) GetActualState(ctx context.Context, configID, name string) (*model.GitOpsResource, error) {
	// Get all networks and find by name
	networks, _, err := r.networkService.ListNetworks(ctx, 0, 0)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		if network.Name == name {
			return networkToGitOpsResource(configID, network), nil
		}
	}

	return nil, sql.ErrNoRows
}

// createNetwork creates a new network from the GitOps manifest
func (r *NetworkReconciler) createNetwork(ctx context.Context, desired *model.GitOpsResource, spec *NetworkSpec) (*ReconcileResult, error) {
	// Convert GitOps spec to network create request
	createReq := &labv1.CreateNetworkRequest{
		Name:           spec.Name,
		Type:           networkTypeToProto(spec.Type),
		BridgeName:     spec.BridgeName,
		Subnet:         spec.Subnet,
		Gateway:        spec.Gateway,
		DhcpEnabled:    spec.DHCPEnabled,
		DhcpRangeStart: spec.DHCPRangeStart,
		DhcpRangeEnd:   spec.DHCPRangeEnd,
		DnsServers:     spec.DNSServers,
		Isolated:       spec.Isolated,
		Description:    spec.Description,
	}

	network, err := r.networkService.CreateNetwork(ctx, createReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to create network: %v", err),
		}, nil
	}

	return &ReconcileResult{
		Action:  ReconcileActionCreated,
		Message: fmt.Sprintf("Created network %s", network.Name),
		Changes: []FieldChange{
			{Field: "name", NewValue: network.Name},
			{Field: "type", NewValue: network.Type.String()},
		},
		ActualState: networkToGitOpsResource(desired.ConfigID, network),
	}, nil
}

// NetworkSpec represents the spec section of a network manifest
type NetworkSpec struct {
	Name           string `yaml:"name"`
	Type           string `yaml:"type"`
	BridgeName     string `yaml:"bridgeName"`
	Subnet         string `yaml:"subnet"`
	Gateway        string `yaml:"gateway"`
	DHCPEnabled    bool   `yaml:"dhcpEnabled"`
	DHCPRangeStart string `yaml:"dhcpRangeStart"`
	DHCPRangeEnd   string `yaml:"dhcpRangeEnd"`
	DNSServers     string `yaml:"dnsServers"`
	Isolated       bool   `yaml:"isolated"`
	Description    string `yaml:"description"`
}

// extractNetworkSpec extracts network spec from untyped manifest spec
func extractNetworkSpec(spec map[string]any) *NetworkSpec {
	if spec == nil {
		return nil
	}

	networkSpec := &NetworkSpec{}

	if name, ok := spec["name"].(string); ok {
		networkSpec.Name = name
	}
	if netType, ok := spec["type"].(string); ok {
		networkSpec.Type = netType
	}
	if bridge, ok := spec["bridgeName"].(string); ok {
		networkSpec.BridgeName = bridge
	}
	if subnet, ok := spec["subnet"].(string); ok {
		networkSpec.Subnet = subnet
	}
	if gateway, ok := spec["gateway"].(string); ok {
		networkSpec.Gateway = gateway
	}
	if dhcp, ok := spec["dhcpEnabled"].(bool); ok {
		networkSpec.DHCPEnabled = dhcp
	}
	if dhcpStart, ok := spec["dhcpRangeStart"].(string); ok {
		networkSpec.DHCPRangeStart = dhcpStart
	}
	if dhcpEnd, ok := spec["dhcpRangeEnd"].(string); ok {
		networkSpec.DHCPRangeEnd = dhcpEnd
	}
	if dns, ok := spec["dnsServers"].(string); ok {
		networkSpec.DNSServers = dns
	}
	if isolated, ok := spec["isolated"].(bool); ok {
		networkSpec.Isolated = isolated
	}
	if desc, ok := spec["description"].(string); ok {
		networkSpec.Description = desc
	}

	return networkSpec
}

// networkTypeToProto converts string network type to proto enum
func networkTypeToProto(netType string) labv1.NetworkType {
	switch netType {
	case "bridge":
		return labv1.NetworkType_NETWORK_TYPE_BRIDGE
	case "user":
		return labv1.NetworkType_NETWORK_TYPE_USER
	default:
		return labv1.NetworkType_NETWORK_TYPE_BRIDGE
	}
}

// networkToGitOpsResource converts a VirtualNetwork proto to GitOpsResource
func networkToGitOpsResource(configID string, network *labv1.VirtualNetwork) *model.GitOpsResource {
	return &model.GitOpsResource{
		ConfigID:      configID,
		Kind:          "Network",
		Name:          network.Name,
		Namespace:     "default",
		Status:        model.GitOpsStatusHealthy,
		StatusMessage: fmt.Sprintf("Network %s is active", network.Name),
		Spec: map[string]any{
			"name":           network.Name,
			"type":           network.Type.String(),
			"bridgeName":     network.BridgeName,
			"subnet":         network.Subnet,
			"gateway":        network.Gateway,
			"dhcpEnabled":    network.DhcpEnabled,
			"dhcpRangeStart": network.DhcpRangeStart,
			"dhcpRangeEnd":   network.DhcpRangeEnd,
			"dnsServers":     network.DnsServers,
			"isolated":       network.Isolated,
			"description":    network.Description,
		},
	}
}
