package model

// NetworkType represents the type of virtual network
type VirtualNetworkType string

const (
	VNetBridge  VirtualNetworkType = "bridge"
	VNetVLAN  VirtualNetworkType = "vlan"
	VNetVXLAN  VirtualNetworkType = "vxlan"
	VNetOVS  VirtualNetworkType = "ovs"
	VNetMACVLAN  VirtualNetworkType = "macvlan"
	VNetIPVLAN  VirtualNetworkType = "ipvlan"
)

// NetworkStatus represents the operational status of a network
type NetworkStatus string

const (
	NetworkStatusActive   NetworkStatus = "active"
	NetworkStatusInactive NetworkStatus = "inactive"
	NetworkStatusError    NetworkStatus = "error"
)

// FirewallAction represents the action for a firewall rule
type FirewallAction string

const (
	FirewallActionAccept FirewallAction = "accept"
	FirewallActionDrop   FirewallAction = "drop"
	FirewallActionReject FirewallAction = "reject"
	FirewallActionLog    FirewallAction = "log"
)

// FirewallDirection represents the direction of traffic for a rule
type FirewallDirection string

const (
	FirewallDirectionInbound  FirewallDirection = "inbound"
	FirewallDirectionOutbound FirewallDirection = "outbound"
	FirewallDirectionBoth     FirewallDirection = "both"
)

// VirtualNetwork represents a virtual network
type VirtualNetwork struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Type           VirtualNetworkType `json:"type"`
	Status         NetworkStatus      `json:"status"`
	BridgeName     string             `json:"bridge_name"`
	VLANID         int                `json:"vlan_id"`
	Subnet         string             `json:"subnet"`
	Gateway        string             `json:"gateway"`
	DHCPEnabled    bool               `json:"dhcp_enabled"`
	DHCPRangeStart string             `json:"dhcp_range_start"`
	DHCPRangeEnd   string             `json:"dhcp_range_end"`
	DNSServers     string             `json:"dns_servers"`
	Isolated       bool               `json:"isolated"`
	MTU            int                `json:"mtu"`
	Description    string             `json:"description"`
	CreatedAt      string             `json:"created_at"`
	InterfaceCount int                `json:"interface_count"`
}

// NetworkInterface represents a virtual network interface
type NetworkInterface struct {
	ID            string `json:"id"`
	NetworkID     string `json:"network_id"`
	Name          string `json:"name"`
	MACAddress    string `json:"mac_address"`
	IPAddress     string `json:"ip_address"`
	InterfaceType string `json:"interface_type"` // "vm", "container", "host"
	EntityID      int    `json:"entity_id"`
	EntityType    string `json:"entity_type"` // "vm" or "container"
	Enabled       bool   `json:"enabled"`
	CreatedAt     string `json:"created_at"`
}

// FirewallRule represents a firewall rule
type FirewallRule struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Priority    int              `json:"priority"`
	Action      FirewallAction   `json:"action"`
	Direction   FirewallDirection `json:"direction"`
	SourceCIDR  string           `json:"source_cidr"`
	DestCIDR    string           `json:"dest_cidr"`
	Protocol    string           `json:"protocol"`
	SourcePort  string           `json:"source_port"`
	DestPort    string           `json:"dest_port"`
	Interface   string           `json:"interface"`
	Enabled     bool             `json:"enabled"`
	Log         bool             `json:"log"`
	Description string           `json:"description"`
	ScopeType   string           `json:"scope_type"` // "global", "node", "network", "vm"
	ScopeID     string           `json:"scope_id"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

// FirewallGroup represents a group of firewall rules
type FirewallGroup struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	RuleIDs     []string `json:"rule_ids"`
	ScopeType   string   `json:"scope_type"`
	ScopeID     string   `json:"scope_id"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"created_at"`
}

// DHCPLease represents a DHCP lease
type DHCPLease struct {
	ID         string `json:"id"`
	NetworkID  string `json:"network_id"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
	ExpiresAt  string `json:"expires_at"`
	IsStatic   bool   `json:"is_static"`
}

// Bridge represents a Linux bridge
type Bridge struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	InterfaceName string `json:"interface_name"`
	STPEnabled  bool     `json:"stp_enabled"`
	Priority    int      `json:"priority"`
	Ports       []string `json:"ports"`
	CreatedAt   string   `json:"created_at"`
}

// NetworkCreateRequest represents a request to create a network
type NetworkCreateRequest struct {
	Name           string      `json:"name"`
	Type           VirtualNetworkType `json:"type"`
	BridgeName     string      `json:"bridge_name"`
	VLANID         int         `json:"vlan_id"`
	Subnet         string      `json:"subnet"`
	Gateway        string      `json:"gateway"`
	DHCPEnabled    bool        `json:"dhcp_enabled"`
	DHCPRangeStart string      `json:"dhcp_range_start"`
	DHCPRangeEnd   string      `json:"dhcp_range_end"`
	DNSServers     string      `json:"dns_servers"`
	Isolated       bool        `json:"isolated"`
	MTU            int         `json:"mtu"`
	Description    string      `json:"description"`
}

// NetworkUpdateRequest represents a request to update a network
type NetworkUpdateRequest struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	Status         NetworkStatus `json:"status"`
	DHCPEnabled    bool   `json:"dhcp_enabled"`
	DHCPRangeStart string `json:"dhcp_range_start"`
	DHCPRangeEnd   string `json:"dhcp_range_end"`
	DNSServers     string `json:"dns_servers"`
	Isolated       bool   `json:"isolated"`
	MTU            int    `json:"mtu"`
	Description    string `json:"description"`
}

// FirewallRuleCreateRequest represents a request to create a firewall rule
type FirewallRuleCreateRequest struct {
	Name        string            `json:"name"`
	Priority    int               `json:"priority"`
	Action      FirewallAction    `json:"action"`
	Direction   FirewallDirection `json:"direction"`
	SourceCIDR  string            `json:"source_cidr"`
	DestCIDR    string            `json:"dest_cidr"`
	Protocol    string            `json:"protocol"`
	SourcePort  string            `json:"source_port"`
	DestPort    string            `json:"dest_port"`
	Interface   string            `json:"interface"`
	Log         bool              `json:"log"`
	Description string            `json:"description"`
	ScopeType   string            `json:"scope_type"`
	ScopeID     string            `json:"scope_id"`
}

// FirewallRuleUpdateRequest represents a request to update a firewall rule
type FirewallRuleUpdateRequest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name,omitempty"`
	Priority    int               `json:"priority"`
	Action      FirewallAction    `json:"action"`
	Direction   FirewallDirection `json:"direction"`
	SourceCIDR  string            `json:"source_cidr"`
	DestCIDR    string            `json:"dest_cidr"`
	Protocol    string            `json:"protocol"`
	SourcePort  string            `json:"source_port"`
	DestPort    string            `json:"dest_port"`
	Log         bool              `json:"log"`
	Description string            `json:"description"`
}

// DHCPStaticLeaseRequest represents a request to add a static DHCP lease
type DHCPStaticLeaseRequest struct {
	NetworkID  string `json:"network_id"`
	MACAddress string `json:"mac_address"`
	IPAddress  string `json:"ip_address"`
	Hostname   string `json:"hostname"`
}
