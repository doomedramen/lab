package libvirt

import (
	"fmt"
	"net"
	"time"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// LibvirtNetworkRepository manages virtual networks via the libvirt API.
// It is responsible for the actual network lifecycle (create/destroy/undefine),
// DHCP lease inspection, and static lease management.  SQLite metadata
// (IDs, descriptions, etc.) is handled separately by the SQLite repository.
type LibvirtNetworkRepository struct {
	client *libvirtx.Client
}

// NewLibvirtNetworkRepository creates a new LibvirtNetworkRepository.
func NewLibvirtNetworkRepository(client *libvirtx.Client) *LibvirtNetworkRepository {
	return &LibvirtNetworkRepository{client: client}
}

// CreateNetwork defines and starts a libvirt virtual network from a model.VirtualNetwork.
// The network is made persistent (auto-start) by default.
func (r *LibvirtNetworkRepository) CreateNetwork(network *model.VirtualNetwork) error {
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("libvirt connection error: %w", err)
	}

	xmlDesc, err := r.buildNetworkXML(network)
	if err != nil {
		return fmt.Errorf("failed to build network XML: %w", err)
	}

	net, err := conn.NetworkDefineXML(xmlDesc)
	if err != nil {
		return fmt.Errorf("failed to define network %q: %w", network.Name, err)
	}
	defer net.Free()

	// Activate the network
	if err := net.Create(); err != nil {
		// Best-effort undefine on activation failure
		_ = net.Undefine()
		return fmt.Errorf("failed to start network %q: %w", network.Name, err)
	}

	// Make it start automatically on host boot
	if err := net.SetAutostart(true); err != nil {
		// Non-fatal — network is running, autostart is a convenience feature
		_ = err
	}

	return nil
}

// DeleteNetwork stops and undefines a libvirt virtual network by its bridge name
// (which is the unique identifier visible to libvirt).
func (r *LibvirtNetworkRepository) DeleteNetwork(name string) error {
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("libvirt connection error: %w", err)
	}

	net, err := conn.LookupNetworkByName(name)
	if err != nil {
		// Network may not exist in libvirt even if it's in SQLite — that's fine
		return nil
	}
	defer net.Free()

	active, err := net.IsActive()
	if err == nil && active {
		if err := net.Destroy(); err != nil {
			return fmt.Errorf("failed to stop network %q: %w", name, err)
		}
	}

	if err := net.Undefine(); err != nil {
		return fmt.Errorf("failed to undefine network %q: %w", name, err)
	}

	return nil
}

// GetDHCPLeases returns active DHCP leases for the given network name.
func (r *LibvirtNetworkRepository) GetDHCPLeases(networkName string) ([]*model.DHCPLease, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("libvirt connection error: %w", err)
	}

	net, err := conn.LookupNetworkByName(networkName)
	if err != nil {
		return nil, fmt.Errorf("network %q not found in libvirt: %w", networkName, err)
	}
	defer net.Free()

	leases, err := net.GetDHCPLeases()
	if err != nil {
		return nil, fmt.Errorf("failed to get DHCP leases for %q: %w", networkName, err)
	}

	var result []*model.DHCPLease
	for _, l := range leases {
		result = append(result, &model.DHCPLease{
			ID:         fmt.Sprintf("%s-%s", networkName, l.Mac),
			NetworkID:  networkName,
			MACAddress: l.Mac,
			IPAddress:  l.IPaddr,
			Hostname:   l.Hostname,
			ExpiresAt:  l.ExpiryTime.Format(time.RFC3339),
			IsStatic:   false,
		})
	}

	return result, nil
}

// AddStaticDHCPLease adds a static DHCP host entry to a libvirt-managed network.
// The change is applied both to the live configuration and the persisted definition.
func (r *LibvirtNetworkRepository) AddStaticDHCPLease(networkName, mac, ip, hostname string) error {
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("libvirt connection error: %w", err)
	}

	net, err := conn.LookupNetworkByName(networkName)
	if err != nil {
		return fmt.Errorf("network %q not found in libvirt: %w", networkName, err)
	}
	defer net.Free()

	host := &libvirtxml.NetworkDHCPHost{
		MAC:  mac,
		IP:   ip,
		Name: hostname,
	}
	hostXML, err := host.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal DHCP host XML: %w", err)
	}

	flags := libvirt.NETWORK_UPDATE_AFFECT_LIVE | libvirt.NETWORK_UPDATE_AFFECT_CONFIG
	if err := net.Update(
		libvirt.NETWORK_UPDATE_COMMAND_ADD_LAST,
		libvirt.NETWORK_SECTION_IP_DHCP_HOST,
		-1,
		hostXML,
		flags,
	); err != nil {
		return fmt.Errorf("failed to add static DHCP lease to %q: %w", networkName, err)
	}

	return nil
}

// RemoveStaticDHCPLease removes a static DHCP host entry from a libvirt-managed network.
func (r *LibvirtNetworkRepository) RemoveStaticDHCPLease(networkName, mac string) error {
	conn, err := r.client.Connection()
	if err != nil {
		return fmt.Errorf("libvirt connection error: %w", err)
	}

	net, err := conn.LookupNetworkByName(networkName)
	if err != nil {
		return fmt.Errorf("network %q not found in libvirt: %w", networkName, err)
	}
	defer net.Free()

	host := &libvirtxml.NetworkDHCPHost{
		MAC: mac,
	}
	hostXML, err := host.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal DHCP host XML: %w", err)
	}

	flags := libvirt.NETWORK_UPDATE_AFFECT_LIVE | libvirt.NETWORK_UPDATE_AFFECT_CONFIG
	if err := net.Update(
		libvirt.NETWORK_UPDATE_COMMAND_DELETE,
		libvirt.NETWORK_SECTION_IP_DHCP_HOST,
		-1,
		hostXML,
		flags,
	); err != nil {
		return fmt.Errorf("failed to remove static DHCP lease from %q: %w", networkName, err)
	}

	return nil
}

// IsActive returns whether the named libvirt network is currently running.
func (r *LibvirtNetworkRepository) IsActive(networkName string) (bool, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return false, fmt.Errorf("libvirt connection error: %w", err)
	}

	net, err := conn.LookupNetworkByName(networkName)
	if err != nil {
		return false, nil // Not found → not active
	}
	defer net.Free()

	return net.IsActive()
}

// buildNetworkXML constructs a libvirt network XML definition from a model.VirtualNetwork.
func (r *LibvirtNetworkRepository) buildNetworkXML(network *model.VirtualNetwork) (string, error) {
	def := &libvirtxml.Network{
		Name: network.Name,
	}

	// Bridge configuration
	bridgeName := network.BridgeName
	if bridgeName == "" {
		bridgeName = network.Name
	}
	def.Bridge = &libvirtxml.NetworkBridge{
		Name:  bridgeName,
		STP:   "on",
		Delay: "0",
	}

	// MTU (libvirt silently ignores MTU if the element is omitted)
	if network.MTU > 0 && network.MTU != 1500 {
		def.MTU = &libvirtxml.NetworkMTU{Size: uint(network.MTU)}
	}

	// Forward mode: isolated networks have no <forward> element
	if !network.Isolated {
		def.Forward = &libvirtxml.NetworkForward{Mode: "nat"}
	}

	// IP configuration
	if network.Subnet != "" {
		ip, ipNet, err := net.ParseCIDR(network.Subnet)
		if err != nil {
			return "", fmt.Errorf("invalid subnet %q: %w", network.Subnet, err)
		}

		gateway := network.Gateway
		if gateway == "" {
			// Default gateway: first host address in the subnet
			gateway = nextIP(ipNet.IP)
		}

		netmask := fmt.Sprintf("%d.%d.%d.%d",
			ipNet.Mask[0], ipNet.Mask[1], ipNet.Mask[2], ipNet.Mask[3])

		_ = ip // parsed but unused after CIDR check

		netIP := libvirtxml.NetworkIP{
			Address: gateway,
			Netmask: netmask,
		}

		if network.DHCPEnabled && network.DHCPRangeStart != "" && network.DHCPRangeEnd != "" {
			netIP.DHCP = &libvirtxml.NetworkDHCP{
				Ranges: []libvirtxml.NetworkDHCPRange{
					{Start: network.DHCPRangeStart, End: network.DHCPRangeEnd},
				},
			}
		}

		def.IPs = []libvirtxml.NetworkIP{netIP}
	}

	xml, err := def.Marshal()
	if err != nil {
		return "", fmt.Errorf("failed to marshal network XML: %w", err)
	}
	return xml, nil
}

// nextIP returns the first host address of a network (network address + 1).
func nextIP(ip net.IP) string {
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	result := make(net.IP, 4)
	copy(result, ip)
	result[3]++
	return result.String()
}
