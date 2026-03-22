package libvirt

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"libvirt.org/go/libvirt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// GuestAgentRepository provides access to QEMU guest agent commands
type GuestAgentRepository struct {
	client libvirtx.LibvirtClient
}

// NewGuestAgentRepository creates a new guest agent repository
func NewGuestAgentRepository(client libvirtx.LibvirtClient) *GuestAgentRepository {
	return &GuestAgentRepository{client: client}
}

// GuestAgentResponse represents a generic response from the guest agent
type GuestAgentResponse struct {
	Return json.RawMessage `json:"return"`
	Error  *AgentError     `json:"error,omitempty"`
}

// AgentError represents an error from the guest agent
type AgentError struct {
	Class string `json:"class"`
	Desc  string `json:"desc"`
}

// NetworkInterfaceResponse represents the response from guest-network-get-interfaces
type NetworkInterfaceResponse struct {
	Name       string   `json:"name"`
	HardwareAddress string `json:"hardware-address"`
	IPAddresses []IPAddress `json:"ip-addresses"`
}

// IPAddress represents an IP address from the guest agent
type IPAddress struct {
	Address     string `json:"ip-address"`
	Prefix      int    `json:"prefix"`
	AddressType string `json:"ip-address-type"` // "ipv4" or "ipv6"
}

// FreezeResponse represents the response from guest-fsfreeze-freeze
type FreezeResponse struct {
	Frozen int `json:"-"` // Number of frozen filesystems
}

// ThawResponse represents the response from guest-fsfreeze-thaw
type ThawResponse struct {
	Thawed int `json:"-"` // Number of thawed filesystems
}

// execAgentCommand executes a QEMU guest agent command on the specified domain
func (r *GuestAgentRepository) execAgentCommand(ctx context.Context, vmid int, command string, timeout time.Duration) ([]byte, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	// Find the domain by VMID
	domainName := fmt.Sprintf("vm-%d", vmid)
	domain, err := conn.LookupDomainByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("domain %s not found: %w", domainName, err)
	}
	defer domain.Free()

	// Check if domain is running
	state, _, err := domain.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain state: %w", err)
	}
	if state != libvirt.DOMAIN_RUNNING {
		return nil, fmt.Errorf("domain is not running (state: %d)", state)
	}

	// Check if guest agent is available before attempting to call it
	// This prevents segfaults when the agent channel exists but isn't functional
	hasAgent, err := r.hasGuestAgent(domain)
	if err != nil || !hasAgent {
		return nil, fmt.Errorf("guest agent not available")
	}

	// Build the QMP command
	cmd := fmt.Sprintf(`{"execute": "%s"}`, command)

	// Execute the guest agent command with timeout
	// libvirt.DomainQemuAgentCommand takes seconds as timeout
	timeoutSeconds := uint32(30)
	if timeout > 0 {
		timeoutSeconds = uint32(timeout.Seconds())
	}

	result, err := domain.QemuAgentCommand(cmd, libvirt.DOMAIN_QEMU_AGENT_COMMAND_BLOCK, timeoutSeconds)
	if err != nil {
		return nil, fmt.Errorf("guest agent command failed: %w", err)
	}

	return []byte(result), nil
}

// hasGuestAgent checks if the guest agent is available and responsive
// This uses libvirt's built-in guest info API which is safer than QemuAgentCommand
func (r *GuestAgentRepository) hasGuestAgent(domain *libvirt.Domain) (bool, error) {
	// Get guest info from libvirt - this doesn't require the agent to respond
	// and won't crash if the agent is broken
	info, err := domain.GetGuestInfo(libvirt.DOMAIN_GUEST_INFO_OS, 0)
	if err != nil {
		// If we can't get guest info, agent is likely not available
		return false, nil
	}
	
	// Check if we got any guest info - if not, agent isn't working
	if info == nil {
		return false, nil
	}
	
	// Check if OS info is populated (indicates agent is working)
	if info.OS != nil && info.OS.Name != "" {
		return true, nil
	}
	
	// Also check for network interfaces as a sign of working agent
	if len(info.Interfaces) > 0 {
		return true, nil
	}
	
	return false, nil
}

// Ping checks if the guest agent is responsive
func (r *GuestAgentRepository) Ping(ctx context.Context, vmid int) bool {
	// Use guest-ping to check if agent is alive
	_, err := r.execAgentCommand(ctx, vmid, "guest-ping", 5*time.Second)
	return err == nil
}

// GetNetworkInterfaces retrieves network interfaces and their IP addresses from the guest agent
func (r *GuestAgentRepository) GetNetworkInterfaces(ctx context.Context, vmid int) ([]model.GuestNetworkInterface, error) {
	result, err := r.execAgentCommand(ctx, vmid, "guest-network-get-interfaces", 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// Parse the response
	var response struct {
		Return []NetworkInterfaceResponse `json:"return"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse network interfaces response: %w", err)
	}

	// Convert to model types
	var interfaces []model.GuestNetworkInterface
	for _, iface := range response.Return {
		// Skip loopback and interfaces without IP addresses
		if iface.Name == "lo" || len(iface.IPAddresses) == 0 {
			continue
		}

		guestIface := model.GuestNetworkInterface{
			Name:           iface.Name,
			MACAddress:     iface.HardwareAddress,
			IPAddresses:    make([]model.GuestIPAddress, 0, len(iface.IPAddresses)),
		}

		for _, ip := range iface.IPAddresses {
			// Prefer IPv4 addresses for the primary IP
			guestIface.IPAddresses = append(guestIface.IPAddresses, model.GuestIPAddress{
				Address:     ip.Address,
				Prefix:      ip.Prefix,
				AddressType: ip.AddressType,
			})
		}

		interfaces = append(interfaces, guestIface)
	}

	return interfaces, nil
}

// FreezeFilesystems freezes guest filesystems for consistent backups
func (r *GuestAgentRepository) FreezeFilesystems(ctx context.Context, vmid int) (int, error) {
	result, err := r.execAgentCommand(ctx, vmid, "guest-fsfreeze-freeze", 30*time.Second)
	if err != nil {
		return 0, fmt.Errorf("failed to freeze filesystems: %w", err)
	}

	// Parse the response
	var response struct {
		Return int `json:"return"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to parse freeze response: %w", err)
	}

	return response.Return, nil
}

// ThawFilesystems thaws (unfreezes) guest filesystems after backup
func (r *GuestAgentRepository) ThawFilesystems(ctx context.Context, vmid int) (int, error) {
	result, err := r.execAgentCommand(ctx, vmid, "guest-fsfreeze-thaw", 30*time.Second)
	if err != nil {
		return 0, fmt.Errorf("failed to thaw filesystems: %w", err)
	}

	// Parse the response
	var response struct {
		Return int `json:"return"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return 0, fmt.Errorf("failed to parse thaw response: %w", err)
	}

	return response.Return, nil
}

// GetPrimaryIP returns the primary IPv4 address from the guest agent
// Returns empty string if no IPv4 address is found or agent is not available
func (r *GuestAgentRepository) GetPrimaryIP(ctx context.Context, vmid int) string {
	interfaces, err := r.GetNetworkInterfaces(ctx, vmid)
	if err != nil {
		return ""
	}

	// Find the first non-loopback interface with an IPv4 address
	for _, iface := range interfaces {
		for _, ip := range iface.IPAddresses {
			if ip.AddressType == "ipv4" && ip.Address != "" {
				return ip.Address
			}
		}
	}

	return ""
}
