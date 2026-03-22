package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

// NetworkService handles virtual network management
type NetworkService struct {
	networkRepo    repository.NetworkRepository
	interfaceRepo  repository.NetworkInterfaceRepository
	firewallRepo   repository.FirewallRuleRepository
	libvirtNetRepo repository.LibvirtNetworkRepository // optional: nil when libvirt is unavailable
	dhcpLeaseRepo  repository.DHCPLeaseRepository      // optional: nil when DB is unavailable
}

// NewNetworkService creates a new network service
func NewNetworkService(
	networkRepo repository.NetworkRepository,
	interfaceRepo repository.NetworkInterfaceRepository,
	firewallRepo repository.FirewallRuleRepository,
) *NetworkService {
	return &NetworkService{
		networkRepo:   networkRepo,
		interfaceRepo: interfaceRepo,
		firewallRepo:  firewallRepo,
	}
}

// WithLibvirtNetworkRepo attaches a libvirt network repository for actual host networking.
func (s *NetworkService) WithLibvirtNetworkRepo(repo repository.LibvirtNetworkRepository) {
	s.libvirtNetRepo = repo
}

// WithDHCPLeaseRepo attaches a DHCP lease repository for static lease persistence.
func (s *NetworkService) WithDHCPLeaseRepo(repo repository.DHCPLeaseRepository) {
	s.dhcpLeaseRepo = repo
}

// ListNetworks returns virtual networks with optional filters
func (s *NetworkService) ListNetworks(ctx context.Context, networkType labv1.NetworkType, status labv1.NetworkStatus) ([]*labv1.VirtualNetwork, int32, error) {
	// Note: networkType from proto is limited (USER/BRIDGE), we use string in model
	var modelType model.VirtualNetworkType
	if networkType == labv1.NetworkType_NETWORK_TYPE_BRIDGE {
		modelType = model.VNetBridge
	}
	
	modelStatus := protoToModelNetworkStatus(status)

	networks, err := s.networkRepo.List(ctx, modelType, modelStatus)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list networks: %w", err)
	}

	var protoNetworks []*labv1.VirtualNetwork
	for _, n := range networks {
		protoNetworks = append(protoNetworks, s.modelToProto(n))
	}

	return protoNetworks, int32(len(protoNetworks)), nil
}

// GetNetwork returns details of a specific network
func (s *NetworkService) GetNetwork(ctx context.Context, id string) (*labv1.VirtualNetwork, error) {
	network, err := s.networkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}
	if network == nil {
		return nil, fmt.Errorf("network not found: %s", id)
	}
	return s.modelToProto(network), nil
}

// CreateNetwork creates a new virtual network
func (s *NetworkService) CreateNetwork(ctx context.Context, req *labv1.CreateNetworkRequest) (*labv1.VirtualNetwork, error) {
	// Validate subnet if provided
	if req.Subnet != "" {
		if _, _, err := net.ParseCIDR(req.Subnet); err != nil {
			return nil, fmt.Errorf("invalid subnet CIDR: %w", err)
		}
	}

	// Map proto NetworkType to model VirtualNetworkType
	var modelType model.VirtualNetworkType
	switch req.Type {
	case labv1.NetworkType_NETWORK_TYPE_BRIDGE:
		modelType = model.VNetBridge
	default:
		modelType = model.VNetBridge
	}

	// Create network record
	network := &model.VirtualNetwork{
		ID:             uuid.New().String(),
		Name:           req.Name,
		Type:           modelType,
		Status:         model.NetworkStatusActive,
		BridgeName:     req.BridgeName,
		VLANID:         int(req.VlanId),
		Subnet:         req.Subnet,
		Gateway:        req.Gateway,
		DHCPEnabled:    req.DhcpEnabled,
		DHCPRangeStart: req.DhcpRangeStart,
		DHCPRangeEnd:   req.DhcpRangeEnd,
		DNSServers:     req.DnsServers,
		Isolated:       req.Isolated,
		MTU:            int(req.Mtu),
		Description:    req.Description,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	// Create the network via libvirt if available, otherwise fall back to brctl.
	if s.libvirtNetRepo != nil {
		if err := s.libvirtNetRepo.CreateNetwork(network); err != nil {
			return nil, fmt.Errorf("failed to create libvirt network: %w", err)
		}
	} else if network.Type == model.VNetBridge && network.BridgeName != "" {
		if err := s.createBridge(network.BridgeName); err != nil {
			slog.Warn("Failed to create bridge", "error", err, "bridge", network.BridgeName)
		}
	}

	if err := s.networkRepo.Create(ctx, network); err != nil {
		// Attempt rollback of libvirt network creation on metadata failure.
		if s.libvirtNetRepo != nil {
			if rbErr := s.libvirtNetRepo.DeleteNetwork(network.Name); rbErr != nil {
				slog.Warn("Failed to roll back libvirt network after metadata error",
					"network", network.Name, "error", rbErr)
			}
		}
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	return s.modelToProto(network), nil
}

// UpdateNetwork updates an existing network
func (s *NetworkService) UpdateNetwork(ctx context.Context, req *labv1.UpdateNetworkRequest) (*labv1.VirtualNetwork, error) {
	network, err := s.networkRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get network: %w", err)
	}
	if network == nil {
		return nil, fmt.Errorf("network not found: %s", req.Id)
	}

	// Update fields
	if req.Name != "" {
		network.Name = req.Name
	}
	if req.Status != labv1.NetworkStatus_NETWORK_STATUS_UNSPECIFIED {
		network.Status = protoToModelNetworkStatus(req.Status)
	}
	network.DHCPEnabled = req.DhcpEnabled
	network.DHCPRangeStart = req.DhcpRangeStart
	network.DHCPRangeEnd = req.DhcpRangeEnd
	network.DNSServers = req.DnsServers
	network.Isolated = req.Isolated
	network.MTU = int(req.Mtu)
	if req.Description != "" {
		network.Description = req.Description
	}

	if err := s.networkRepo.Update(ctx, network); err != nil {
		return nil, fmt.Errorf("failed to update network: %w", err)
	}

	return s.modelToProto(network), nil
}

// DeleteNetwork deletes a network
func (s *NetworkService) DeleteNetwork(ctx context.Context, id string, force bool) error {
	network, err := s.networkRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get network: %w", err)
	}
	if network == nil {
		return fmt.Errorf("network not found: %s", id)
	}

	// Check if network has interfaces
	interfaces, err := s.interfaceRepo.List(ctx, id, 0, "")
	if err != nil {
		return fmt.Errorf("failed to check for interfaces: %w", err)
	}

	if len(interfaces) > 0 && !force {
		return fmt.Errorf("network has %d interfaces, use force=true to delete", len(interfaces))
	}

	// Remove from libvirt or fall back to brctl.
	if s.libvirtNetRepo != nil {
		if err := s.libvirtNetRepo.DeleteNetwork(network.Name); err != nil {
			slog.Warn("Failed to delete libvirt network", "error", err, "network", network.Name)
		}
	} else if network.Type == model.VNetBridge && network.BridgeName != "" {
		if err := s.deleteBridge(network.BridgeName); err != nil {
			slog.Warn("Failed to delete bridge", "error", err, "bridge", network.BridgeName)
		}
	}

	if err := s.networkRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// ListNetworkInterfaces returns network interfaces with optional filters
func (s *NetworkService) ListNetworkInterfaces(ctx context.Context, networkID string, entityID int32, entityType string) ([]*labv1.VmNetworkInterface, int32, error) {
	interfaces, err := s.interfaceRepo.List(ctx, networkID, int(entityID), entityType)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list network interfaces: %w", err)
	}

	var protoInterfaces []*labv1.VmNetworkInterface
	for _, i := range interfaces {
		protoInterfaces = append(protoInterfaces, s.interfaceModelToProto(i))
	}

	return protoInterfaces, int32(len(protoInterfaces)), nil
}

// CreateNetworkInterface creates a new network interface
func (s *NetworkService) CreateNetworkInterface(ctx context.Context, req *labv1.CreateVmNetworkInterfaceRequest) (*labv1.VmNetworkInterface, error) {
	// Validate MAC address if provided
	if req.MacAddress != "" {
		if _, err := net.ParseMAC(req.MacAddress); err != nil {
			return nil, fmt.Errorf("invalid MAC address: %w", err)
		}
	}

	// Validate IP address if provided
	if req.IpAddress != "" {
		if ip := net.ParseIP(req.IpAddress); ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", req.IpAddress)
		}
	}

	iface := &model.NetworkInterface{
		ID:            uuid.New().String(),
		NetworkID:     req.NetworkId,
		Name:          req.Name,
		MACAddress:    req.MacAddress,
		IPAddress:     req.IpAddress,
		InterfaceType: req.InterfaceType,
		EntityID:      int(req.EntityId),
		EntityType:    req.EntityType,
		Enabled:       true,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	if err := s.interfaceRepo.Create(ctx, iface); err != nil {
		return nil, fmt.Errorf("failed to create network interface: %w", err)
	}

	// Update network interface count
	if err := s.updateNetworkInterfaceCount(ctx, req.NetworkId); err != nil {
		slog.Warn("Failed to update network interface count", "error", err)
	}

	return s.interfaceModelToProto(iface), nil
}

// UpdateNetworkInterface updates an existing interface
func (s *NetworkService) UpdateNetworkInterface(ctx context.Context, req *labv1.UpdateVmNetworkInterfaceRequest) (*labv1.VmNetworkInterface, error) {
	iface, err := s.interfaceRepo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface: %w", err)
	}
	if iface == nil {
		return nil, fmt.Errorf("interface not found: %s", req.Id)
	}

	// Validate MAC address if provided
	if req.MacAddress != "" {
		if _, err := net.ParseMAC(req.MacAddress); err != nil {
			return nil, fmt.Errorf("invalid MAC address: %w", err)
		}
		iface.MACAddress = req.MacAddress
	}

	// Validate IP address if provided
	if req.IpAddress != "" {
		if ip := net.ParseIP(req.IpAddress); ip == nil {
			return nil, fmt.Errorf("invalid IP address: %s", req.IpAddress)
		}
		iface.IPAddress = req.IpAddress
	}

	iface.Enabled = req.Enabled

	if err := s.interfaceRepo.Update(ctx, iface); err != nil {
		return nil, fmt.Errorf("failed to update network interface: %w", err)
	}

	return s.interfaceModelToProto(iface), nil
}

// DeleteNetworkInterface deletes an interface
func (s *NetworkService) DeleteNetworkInterface(ctx context.Context, id string) error {
	iface, err := s.interfaceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}
	if iface == nil {
		return fmt.Errorf("interface not found: %s", id)
	}

	networkID := iface.NetworkID

	if err := s.interfaceRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete network interface: %w", err)
	}

	// Update network interface count
	if err := s.updateNetworkInterfaceCount(ctx, networkID); err != nil {
		slog.Warn("Failed to update network interface count", "error", err)
	}

	return nil
}

// ListBridges returns Linux bridges
func (s *NetworkService) ListBridges(ctx context.Context) ([]*labv1.Bridge, int32, error) {
	cmd := exec.Command("brctl", "show")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// brctl not available, return empty list
		return []*labv1.Bridge{}, 0, nil
	}

	var bridges []*labv1.Bridge
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && fields[0] != "bridge" && fields[0] != "name" {
			bridges = append(bridges, &labv1.Bridge{
				Id:            fields[0],
				Name:          fields[0],
				InterfaceName: fields[0],
				StpEnabled:    len(fields) > 1 && fields[1] != "no",
				CreatedAt:     time.Now().Format(time.RFC3339),
			})
		}
	}

	return bridges, int32(len(bridges)), nil
}

// CreateBridge creates a Linux bridge
func (s *NetworkService) CreateBridge(ctx context.Context, req *labv1.CreateBridgeRequest) (*labv1.Bridge, error) {
	if err := s.createBridge(req.Name); err != nil {
		return nil, fmt.Errorf("failed to create bridge: %w", err)
	}

	bridge := &labv1.Bridge{
		Id:            req.Name,
		Name:          req.Name,
		InterfaceName: req.Name,
		StpEnabled:    req.StpEnabled,
		Priority:      int32(req.Priority),
		Ports:         req.Ports,
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	return bridge, nil
}

// DeleteBridge deletes a Linux bridge
func (s *NetworkService) DeleteBridge(ctx context.Context, name string, force bool) error {
	if err := s.deleteBridge(name); err != nil {
		return fmt.Errorf("failed to delete bridge: %w", err)
	}
	return nil
}

// GetDHCPLeases returns DHCP leases for a network.
// If libvirt is available, live leases are returned; otherwise only static leases from SQLite.
// When staticOnly is true, only the persisted static leases are returned.
func (s *NetworkService) GetDHCPLeases(ctx context.Context, networkID string, staticOnly bool) ([]*labv1.DHCPLease, int32, error) {
	// Look up the network to get its name (libvirt uses name, not UUID).
	network, err := s.networkRepo.GetByID(ctx, networkID)
	if err != nil || network == nil {
		return []*labv1.DHCPLease{}, 0, nil
	}

	var leases []*labv1.DHCPLease

	// Static leases from SQLite
	if s.dhcpLeaseRepo != nil {
		staticLeases, err := s.dhcpLeaseRepo.List(ctx, networkID)
		if err != nil {
			slog.Warn("Failed to list static DHCP leases", "error", err)
		}
		for _, l := range staticLeases {
			leases = append(leases, dhcpLeaseModelToProto(l))
		}
	}

	if !staticOnly && s.libvirtNetRepo != nil {
		liveLeases, err := s.libvirtNetRepo.GetDHCPLeases(network.Name)
		if err != nil {
			slog.Warn("Failed to get live DHCP leases from libvirt", "error", err)
		}
		for _, l := range liveLeases {
			leases = append(leases, dhcpLeaseModelToProto(l))
		}
	}

	return leases, int32(len(leases)), nil
}

// AddDHCPStaticLease adds a static DHCP lease, persisting it in SQLite and pushing it
// to libvirt if available.
func (s *NetworkService) AddDHCPStaticLease(ctx context.Context, req *labv1.AddDHCPStaticLeaseRequest) (*labv1.DHCPLease, error) {
	network, err := s.networkRepo.GetByID(ctx, req.NetworkId)
	if err != nil || network == nil {
		return nil, fmt.Errorf("network not found: %s", req.NetworkId)
	}

	lease := &model.DHCPLease{
		ID:         uuid.New().String(),
		NetworkID:  req.NetworkId,
		MACAddress: req.MacAddress,
		IPAddress:  req.IpAddress,
		Hostname:   req.Hostname,
		IsStatic:   true,
	}

	// Persist in SQLite
	if s.dhcpLeaseRepo != nil {
		if err := s.dhcpLeaseRepo.Create(ctx, lease); err != nil {
			return nil, fmt.Errorf("failed to persist static DHCP lease: %w", err)
		}
	}

	// Apply to libvirt
	if s.libvirtNetRepo != nil {
		if err := s.libvirtNetRepo.AddStaticDHCPLease(network.Name, req.MacAddress, req.IpAddress, req.Hostname); err != nil {
			slog.Warn("Failed to add static DHCP lease to libvirt", "error", err, "network", network.Name)
		}
	}

	return dhcpLeaseModelToProto(lease), nil
}

// RemoveDHCPStaticLease removes a static DHCP lease from SQLite and from libvirt.
func (s *NetworkService) RemoveDHCPStaticLease(ctx context.Context, req *labv1.RemoveDHCPStaticLeaseRequest) error {
	network, err := s.networkRepo.GetByID(ctx, req.NetworkId)
	if err != nil || network == nil {
		return fmt.Errorf("network not found: %s", req.NetworkId)
	}

	if s.dhcpLeaseRepo != nil {
		if err := s.dhcpLeaseRepo.Delete(ctx, req.NetworkId, req.MacAddress); err != nil {
			return fmt.Errorf("failed to remove static DHCP lease: %w", err)
		}
	}

	if s.libvirtNetRepo != nil {
		if err := s.libvirtNetRepo.RemoveStaticDHCPLease(network.Name, req.MacAddress); err != nil {
			slog.Warn("Failed to remove static DHCP lease from libvirt", "error", err)
		}
	}

	return nil
}

// dhcpLeaseModelToProto converts model.DHCPLease to labv1.DHCPLease
func dhcpLeaseModelToProto(l *model.DHCPLease) *labv1.DHCPLease {
	return &labv1.DHCPLease{
		Id:         l.ID,
		NetworkId:  l.NetworkID,
		MacAddress: l.MACAddress,
		IpAddress:  l.IPAddress,
		Hostname:   l.Hostname,
		ExpiresAt:  l.ExpiresAt,
		IsStatic:   l.IsStatic,
	}
}

// modelToProto converts model.VirtualNetwork to labv1.VirtualNetwork
func (s *NetworkService) modelToProto(network *model.VirtualNetwork) *labv1.VirtualNetwork {
	if network == nil {
		return nil
	}

	// Map model VirtualNetworkType to proto NetworkType
	var protoType labv1.NetworkType
	if network.Type == model.VNetBridge {
		protoType = labv1.NetworkType_NETWORK_TYPE_BRIDGE
	} else {
		protoType = labv1.NetworkType_NETWORK_TYPE_BRIDGE
	}

	return &labv1.VirtualNetwork{
		Id:             network.ID,
		Name:           network.Name,
		Type:           protoType,
		Status:         modelNetworkStatusToProto(network.Status),
		BridgeName:     network.BridgeName,
		VlanId:         int32(network.VLANID),
		Subnet:         network.Subnet,
		Gateway:        network.Gateway,
		DhcpEnabled:    network.DHCPEnabled,
		DhcpRangeStart: network.DHCPRangeStart,
		DhcpRangeEnd:   network.DHCPRangeEnd,
		DnsServers:     network.DNSServers,
		Isolated:       network.Isolated,
		Mtu:            int32(network.MTU),
		Description:    network.Description,
		CreatedAt:      network.CreatedAt,
		InterfaceCount: int32(network.InterfaceCount),
	}
}

// interfaceModelToProto converts model.NetworkInterface to labv1.VmNetworkInterface
func (s *NetworkService) interfaceModelToProto(iface *model.NetworkInterface) *labv1.VmNetworkInterface {
	if iface == nil {
		return nil
	}

	return &labv1.VmNetworkInterface{
		Id:            iface.ID,
		NetworkId:     iface.NetworkID,
		Name:          iface.Name,
		MacAddress:    iface.MACAddress,
		IpAddress:     iface.IPAddress,
		InterfaceType: iface.InterfaceType,
		EntityId:      int32(iface.EntityID),
		EntityType:    iface.EntityType,
		Enabled:       iface.Enabled,
		CreatedAt:     iface.CreatedAt,
	}
}

// updateNetworkInterfaceCount updates the interface count for a network
func (s *NetworkService) updateNetworkInterfaceCount(ctx context.Context, networkID string) error {
	interfaces, err := s.interfaceRepo.List(ctx, networkID, 0, "")
	if err != nil {
		return err
	}
	return s.networkRepo.UpdateInterfaceCount(ctx, networkID, len(interfaces))
}

// createBridge creates a Linux bridge using brctl
func (s *NetworkService) createBridge(name string) error {
	cmd := exec.Command("brctl", "addbr", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create bridge %s: %w", name, err)
	}

	// Bring up the bridge
	cmd = exec.Command("ip", "link", "set", name, "up")
	if err := cmd.Run(); err != nil {
		slog.Warn("Failed to bring up bridge", "bridge", name, "error", err)
	}

	return nil
}

// deleteBridge deletes a Linux bridge
func (s *NetworkService) deleteBridge(name string) error {
	cmd := exec.Command("ip", "link", "set", name, "down")
	_ = cmd.Run() // Ignore error if already down

	cmd = exec.Command("brctl", "delbr", name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete bridge %s: %w", name, err)
	}

	return nil
}

// Helper functions for type conversion
func protoToModelNetworkStatus(s labv1.NetworkStatus) model.NetworkStatus {
	switch s {
	case labv1.NetworkStatus_NETWORK_STATUS_ACTIVE:
		return model.NetworkStatusActive
	case labv1.NetworkStatus_NETWORK_STATUS_INACTIVE:
		return model.NetworkStatusInactive
	case labv1.NetworkStatus_NETWORK_STATUS_ERROR:
		return model.NetworkStatusError
	default:
		return ""
	}
}

func modelNetworkStatusToProto(s model.NetworkStatus) labv1.NetworkStatus {
	switch s {
	case model.NetworkStatusActive:
		return labv1.NetworkStatus_NETWORK_STATUS_ACTIVE
	case model.NetworkStatusInactive:
		return labv1.NetworkStatus_NETWORK_STATUS_INACTIVE
	case model.NetworkStatusError:
		return labv1.NetworkStatus_NETWORK_STATUS_ERROR
	default:
		return labv1.NetworkStatus_NETWORK_STATUS_UNSPECIFIED
	}
}
