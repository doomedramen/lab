package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// NetworkRepository handles virtual network storage and retrieval
type NetworkRepository struct {
	db *sqlitePkg.DB
}

// NewNetworkRepository creates a new network repository
func NewNetworkRepository(db *sqlitePkg.DB) *NetworkRepository {
	return &NetworkRepository{db: db}
}

// Create saves a new virtual network to the database
func (r *NetworkRepository) Create(ctx context.Context, network *model.VirtualNetwork) error {
	query := `
		INSERT INTO virtual_networks (
			id, name, type, status, bridge_name, vlan_id, subnet, gateway,
			dhcp_enabled, dhcp_range_start, dhcp_range_end, dns_servers,
			isolated, mtu, description, interface_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		network.ID,
		network.Name,
		string(network.Type),
		string(network.Status),
		network.BridgeName,
		network.VLANID,
		network.Subnet,
		network.Gateway,
		boolToInt(network.DHCPEnabled),
		nullString(network.DHCPRangeStart),
		nullString(network.DHCPRangeEnd),
		network.DNSServers,
		boolToInt(network.Isolated),
		network.MTU,
		network.Description,
		network.InterfaceCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	return nil
}

// GetByID retrieves a network by its ID
func (r *NetworkRepository) GetByID(ctx context.Context, id string) (*model.VirtualNetwork, error) {
	query := `
		SELECT id, name, type, status, bridge_name, vlan_id, subnet, gateway,
		       dhcp_enabled, dhcp_range_start, dhcp_range_end, dns_servers,
		       isolated, mtu, description, created_at, interface_count
		FROM virtual_networks
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanNetwork(row)
}

// List retrieves networks with optional filters
func (r *NetworkRepository) List(ctx context.Context, networkType model.VirtualNetworkType, status model.NetworkStatus) ([]*model.VirtualNetwork, error) {
	query := `
		SELECT id, name, type, status, bridge_name, vlan_id, subnet, gateway,
		       dhcp_enabled, dhcp_range_start, dhcp_range_end, dns_servers,
		       isolated, mtu, description, created_at, interface_count
		FROM virtual_networks
		WHERE 1=1
	`

	args := []interface{}{}
	if networkType != "" {
		query += " AND type = ?"
		args = append(args, string(networkType))
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, string(status))
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}
	defer rows.Close()

	var networks []*model.VirtualNetwork
	for rows.Next() {
		network, err := scanNetworkRow(rows)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}

	return networks, rows.Err()
}

// Update updates an existing network
func (r *NetworkRepository) Update(ctx context.Context, network *model.VirtualNetwork) error {
	query := `
		UPDATE virtual_networks
		SET name = ?, status = ?, dhcp_enabled = ?, dhcp_range_start = ?,
		    dhcp_range_end = ?, dns_servers = ?, isolated = ?, mtu = ?,
		    description = ?, interface_count = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		network.Name,
		string(network.Status),
		boolToInt(network.DHCPEnabled),
		nullString(network.DHCPRangeStart),
		nullString(network.DHCPRangeEnd),
		network.DNSServers,
		boolToInt(network.Isolated),
		network.MTU,
		network.Description,
		network.InterfaceCount,
		network.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update network: %w", err)
	}

	return nil
}

// Delete removes a network from the database
func (r *NetworkRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM virtual_networks WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete network: %w", err)
	}

	return nil
}

// UpdateInterfaceCount updates the interface count for a network
func (r *NetworkRepository) UpdateInterfaceCount(ctx context.Context, id string, count int) error {
	query := `UPDATE virtual_networks SET interface_count = ? WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, count, id)
	if err != nil {
		return fmt.Errorf("failed to update interface count: %w", err)
	}

	return nil
}

// NetworkInterfaceRepository handles network interface storage and retrieval
type NetworkInterfaceRepository struct {
	db *sqlitePkg.DB
}

// NewNetworkInterfaceRepository creates a new network interface repository
func NewNetworkInterfaceRepository(db *sqlitePkg.DB) *NetworkInterfaceRepository {
	return &NetworkInterfaceRepository{db: db}
}

// Create saves a new network interface to the database
func (r *NetworkInterfaceRepository) Create(ctx context.Context, iface *model.NetworkInterface) error {
	query := `
		INSERT INTO network_interfaces (
			id, network_id, name, mac_address, ip_address, interface_type,
			entity_id, entity_type, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		iface.ID,
		iface.NetworkID,
		iface.Name,
		iface.MACAddress,
		iface.IPAddress,
		iface.InterfaceType,
		iface.EntityID,
		iface.EntityType,
		boolToInt(iface.Enabled),
	)

	if err != nil {
		return fmt.Errorf("failed to create network interface: %w", err)
	}

	return nil
}

// GetByID retrieves an interface by its ID
func (r *NetworkInterfaceRepository) GetByID(ctx context.Context, id string) (*model.NetworkInterface, error) {
	query := `
		SELECT id, network_id, name, mac_address, ip_address, interface_type,
		       entity_id, entity_type, enabled, created_at
		FROM network_interfaces
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanNetworkInterface(row)
}

// List retrieves interfaces with optional filters
func (r *NetworkInterfaceRepository) List(ctx context.Context, networkID string, entityID int, entityType string) ([]*model.NetworkInterface, error) {
	query := `
		SELECT id, network_id, name, mac_address, ip_address, interface_type,
		       entity_id, entity_type, enabled, created_at
		FROM network_interfaces
		WHERE 1=1
	`

	args := []interface{}{}
	if networkID != "" {
		query += " AND network_id = ?"
		args = append(args, networkID)
	}
	if entityID > 0 {
		query += " AND entity_id = ?"
		args = append(args, entityID)
	}
	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}

	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list network interfaces: %w", err)
	}
	defer rows.Close()

	var interfaces []*model.NetworkInterface
	for rows.Next() {
		iface, err := scanNetworkInterfaceRow(rows)
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, iface)
	}

	return interfaces, rows.Err()
}

// Update updates an existing interface
func (r *NetworkInterfaceRepository) Update(ctx context.Context, iface *model.NetworkInterface) error {
	query := `
		UPDATE network_interfaces
		SET mac_address = ?, ip_address = ?, enabled = ?
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		iface.MACAddress,
		iface.IPAddress,
		boolToInt(iface.Enabled),
		iface.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update network interface: %w", err)
	}

	return nil
}

// Delete removes an interface from the database
func (r *NetworkInterfaceRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM network_interfaces WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete network interface: %w", err)
	}

	return nil
}

// FirewallRuleRepository handles firewall rule storage and retrieval
type FirewallRuleRepository struct {
	db *sqlitePkg.DB
}

// NewFirewallRuleRepository creates a new firewall rule repository
func NewFirewallRuleRepository(db *sqlitePkg.DB) *FirewallRuleRepository {
	return &FirewallRuleRepository{db: db}
}

// Create saves a new firewall rule to the database
func (r *FirewallRuleRepository) Create(ctx context.Context, rule *model.FirewallRule) error {
	query := `
		INSERT INTO firewall_rules (
			id, name, priority, action, direction, source_cidr, dest_cidr,
			protocol, source_port, dest_port, interface, enabled, log,
			description, scope_type, scope_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Priority,
		string(rule.Action),
		string(rule.Direction),
		nullString(rule.SourceCIDR),
		nullString(rule.DestCIDR),
		rule.Protocol,
		nullString(rule.SourcePort),
		nullString(rule.DestPort),
		nullString(rule.Interface),
		boolToInt(rule.Enabled),
		boolToInt(rule.Log),
		rule.Description,
		rule.ScopeType,
		rule.ScopeID,
	)

	if err != nil {
		return fmt.Errorf("failed to create firewall rule: %w", err)
	}

	return nil
}

// GetByID retrieves a rule by its ID
func (r *FirewallRuleRepository) GetByID(ctx context.Context, id string) (*model.FirewallRule, error) {
	query := `
		SELECT id, name, priority, action, direction, source_cidr, dest_cidr,
		       protocol, source_port, dest_port, interface, enabled, log,
		       description, scope_type, scope_id, created_at, updated_at
		FROM firewall_rules
		WHERE id = ?
	`

	row := r.db.QueryRowContext(ctx, query, id)
	return scanFirewallRule(row)
}

// List retrieves rules with optional filters
func (r *FirewallRuleRepository) List(ctx context.Context, scopeType, scopeID string, enabledOnly bool) ([]*model.FirewallRule, error) {
	query := `
		SELECT id, name, priority, action, direction, source_cidr, dest_cidr,
		       protocol, source_port, dest_port, interface, enabled, log,
		       description, scope_type, scope_id, created_at, updated_at
		FROM firewall_rules
		WHERE 1=1
	`

	args := []interface{}{}
	if scopeType != "" {
		query += " AND scope_type = ?"
		args = append(args, scopeType)
	}
	if scopeID != "" {
		query += " AND scope_id = ?"
		args = append(args, scopeID)
	}
	if enabledOnly {
		query += " AND enabled = 1"
	}

	query += " ORDER BY priority ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list firewall rules: %w", err)
	}
	defer rows.Close()

	var rules []*model.FirewallRule
	for rows.Next() {
		rule, err := scanFirewallRuleRow(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// Update updates an existing rule
func (r *FirewallRuleRepository) Update(ctx context.Context, rule *model.FirewallRule) error {
	query := `
		UPDATE firewall_rules
		SET name = ?, priority = ?, action = ?, direction = ?,
		    source_cidr = ?, dest_cidr = ?, protocol = ?, source_port = ?,
		    dest_port = ?, interface = ?, enabled = ?, log = ?,
		    description = ?, updated_at = datetime('now')
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		rule.Name,
		rule.Priority,
		string(rule.Action),
		string(rule.Direction),
		nullString(rule.SourceCIDR),
		nullString(rule.DestCIDR),
		rule.Protocol,
		nullString(rule.SourcePort),
		nullString(rule.DestPort),
		nullString(rule.Interface),
		boolToInt(rule.Enabled),
		boolToInt(rule.Log),
		rule.Description,
		rule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update firewall rule: %w", err)
	}

	return nil
}

// Delete removes a rule from the database
func (r *FirewallRuleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM firewall_rules WHERE id = ?`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete firewall rule: %w", err)
	}

	return nil
}

// scanNetwork scans a single row into a VirtualNetwork struct
func scanNetwork(row scanner) (*model.VirtualNetwork, error) {
	var n model.VirtualNetwork
	var dhcpRangeStart, dhcpRangeEnd sql.NullString

	err := row.Scan(
		&n.ID,
		&n.Name,
		&n.Type,
		&n.Status,
		&n.BridgeName,
		&n.VLANID,
		&n.Subnet,
		&n.Gateway,
		&n.DHCPEnabled,
		&dhcpRangeStart,
		&dhcpRangeEnd,
		&n.DNSServers,
		&n.Isolated,
		&n.MTU,
		&n.Description,
		&n.CreatedAt,
		&n.InterfaceCount,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan network: %w", err)
	}

	n.DHCPRangeStart = dhcpRangeStart.String
	n.DHCPRangeEnd = dhcpRangeEnd.String
	return &n, nil
}

// scanNetworkRow scans a row from Rows into a VirtualNetwork struct
func scanNetworkRow(rows rowsScanner) (*model.VirtualNetwork, error) {
	var n model.VirtualNetwork
	var dhcpRangeStart, dhcpRangeEnd sql.NullString

	err := rows.Scan(
		&n.ID,
		&n.Name,
		&n.Type,
		&n.Status,
		&n.BridgeName,
		&n.VLANID,
		&n.Subnet,
		&n.Gateway,
		&n.DHCPEnabled,
		&dhcpRangeStart,
		&dhcpRangeEnd,
		&n.DNSServers,
		&n.Isolated,
		&n.MTU,
		&n.Description,
		&n.CreatedAt,
		&n.InterfaceCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan network row: %w", err)
	}

	n.DHCPRangeStart = dhcpRangeStart.String
	n.DHCPRangeEnd = dhcpRangeEnd.String
	return &n, nil
}

// scanNetworkInterface scans a single row into a NetworkInterface struct
func scanNetworkInterface(row scanner) (*model.NetworkInterface, error) {
	var n model.NetworkInterface
	var enabled int

	err := row.Scan(
		&n.ID,
		&n.NetworkID,
		&n.Name,
		&n.MACAddress,
		&n.IPAddress,
		&n.InterfaceType,
		&n.EntityID,
		&n.EntityType,
		&enabled,
		&n.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan network interface: %w", err)
	}

	n.Enabled = enabled == 1
	return &n, nil
}

// scanNetworkInterfaceRow scans a row from Rows into a NetworkInterface struct
func scanNetworkInterfaceRow(rows rowsScanner) (*model.NetworkInterface, error) {
	var n model.NetworkInterface
	var enabled int

	err := rows.Scan(
		&n.ID,
		&n.NetworkID,
		&n.Name,
		&n.MACAddress,
		&n.IPAddress,
		&n.InterfaceType,
		&n.EntityID,
		&n.EntityType,
		&enabled,
		&n.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan network interface row: %w", err)
	}

	n.Enabled = enabled == 1
	return &n, nil
}

// FirewallGroupRepository handles firewall group storage and retrieval
type FirewallGroupRepository struct {
	db *sqlitePkg.DB
}

// NewFirewallGroupRepository creates a new firewall group repository
func NewFirewallGroupRepository(db *sqlitePkg.DB) *FirewallGroupRepository {
	return &FirewallGroupRepository{db: db}
}

// Create saves a new firewall group to the database
func (r *FirewallGroupRepository) Create(ctx context.Context, group *model.FirewallGroup) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		INSERT INTO firewall_groups (id, name, scope_type, scope_id, description)
		VALUES (?, ?, ?, ?, ?)`,
		group.ID, group.Name, group.ScopeType, group.ScopeID, group.Description,
	)
	if err != nil {
		return fmt.Errorf("failed to create firewall group: %w", err)
	}

	for _, ruleID := range group.RuleIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO firewall_group_rules (group_id, rule_id) VALUES (?, ?)`,
			group.ID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("failed to add rule %s to group: %w", ruleID, err)
		}
	}

	return tx.Commit()
}

// GetByID retrieves a firewall group by its ID, including its rule IDs
func (r *FirewallGroupRepository) GetByID(ctx context.Context, id string) (*model.FirewallGroup, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, scope_type, scope_id, description, created_at
		FROM firewall_groups WHERE id = ?`, id)

	group, err := scanFirewallGroup(row)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, nil
	}

	ruleIDs, err := r.getRuleIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	group.RuleIDs = ruleIDs
	return group, nil
}

// List retrieves firewall groups with optional scope filters
func (r *FirewallGroupRepository) List(ctx context.Context, scopeType, scopeID string) ([]*model.FirewallGroup, error) {
	query := `SELECT id, name, scope_type, scope_id, description, created_at FROM firewall_groups WHERE 1=1`
	args := []interface{}{}

	if scopeType != "" {
		query += " AND scope_type = ?"
		args = append(args, scopeType)
	}
	if scopeID != "" {
		query += " AND scope_id = ?"
		args = append(args, scopeID)
	}
	query += " ORDER BY name"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list firewall groups: %w", err)
	}
	defer rows.Close()

	var groups []*model.FirewallGroup
	for rows.Next() {
		group, err := scanFirewallGroupRow(rows)
		if err != nil {
			return nil, err
		}
		ruleIDs, err := r.getRuleIDs(ctx, group.ID)
		if err != nil {
			return nil, err
		}
		group.RuleIDs = ruleIDs
		groups = append(groups, group)
	}

	return groups, rows.Err()
}

// Update updates an existing firewall group's name/description and replaces its rule set
func (r *FirewallGroupRepository) Update(ctx context.Context, group *model.FirewallGroup) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	_, err = tx.ExecContext(ctx, `
		UPDATE firewall_groups SET name = ?, description = ? WHERE id = ?`,
		group.Name, group.Description, group.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update firewall group: %w", err)
	}

	// Replace rule associations
	_, err = tx.ExecContext(ctx, `DELETE FROM firewall_group_rules WHERE group_id = ?`, group.ID)
	if err != nil {
		return fmt.Errorf("failed to clear group rules: %w", err)
	}

	for _, ruleID := range group.RuleIDs {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO firewall_group_rules (group_id, rule_id) VALUES (?, ?)`,
			group.ID, ruleID,
		)
		if err != nil {
			return fmt.Errorf("failed to add rule %s to group: %w", ruleID, err)
		}
	}

	return tx.Commit()
}

// Delete removes a firewall group and its rule associations
func (r *FirewallGroupRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM firewall_groups WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete firewall group: %w", err)
	}
	return nil
}

func (r *FirewallGroupRepository) getRuleIDs(ctx context.Context, groupID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT rule_id FROM firewall_group_rules WHERE group_id = ? ORDER BY rule_id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule IDs for group %s: %w", groupID, err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DHCPLeaseRepository handles static DHCP lease storage
type DHCPLeaseRepository struct {
	db *sqlitePkg.DB
}

// NewDHCPLeaseRepository creates a new DHCP lease repository
func NewDHCPLeaseRepository(db *sqlitePkg.DB) *DHCPLeaseRepository {
	return &DHCPLeaseRepository{db: db}
}

// Create stores a static DHCP lease
func (r *DHCPLeaseRepository) Create(ctx context.Context, lease *model.DHCPLease) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO dhcp_static_leases (id, network_id, mac_address, ip_address, hostname)
		VALUES (?, ?, ?, ?, ?)`,
		lease.ID, lease.NetworkID, lease.MACAddress, lease.IPAddress, lease.Hostname,
	)
	if err != nil {
		return fmt.Errorf("failed to create DHCP lease: %w", err)
	}
	return nil
}

// Delete removes a static DHCP lease by network ID and MAC address
func (r *DHCPLeaseRepository) Delete(ctx context.Context, networkID, mac string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM dhcp_static_leases WHERE network_id = ? AND mac_address = ?`,
		networkID, mac,
	)
	if err != nil {
		return fmt.Errorf("failed to delete DHCP lease: %w", err)
	}
	return nil
}

// List returns static DHCP leases for a given network
func (r *DHCPLeaseRepository) List(ctx context.Context, networkID string) ([]*model.DHCPLease, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, network_id, mac_address, ip_address, hostname, created_at
		FROM dhcp_static_leases WHERE network_id = ? ORDER BY ip_address`,
		networkID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list DHCP leases: %w", err)
	}
	defer rows.Close()

	var leases []*model.DHCPLease
	for rows.Next() {
		var l model.DHCPLease
		var createdAt string
		if err := rows.Scan(&l.ID, &l.NetworkID, &l.MACAddress, &l.IPAddress, &l.Hostname, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan DHCP lease: %w", err)
		}
		l.IsStatic = true
		leases = append(leases, &l)
	}
	return leases, rows.Err()
}

// scanFirewallGroup scans a single row into a FirewallGroup struct (without rule IDs)
func scanFirewallGroup(row scanner) (*model.FirewallGroup, error) {
	var g model.FirewallGroup
	err := row.Scan(&g.ID, &g.Name, &g.ScopeType, &g.ScopeID, &g.Description, &g.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan firewall group: %w", err)
	}
	return &g, nil
}

// scanFirewallGroupRow scans a Rows row into a FirewallGroup struct (without rule IDs)
func scanFirewallGroupRow(rows rowsScanner) (*model.FirewallGroup, error) {
	var g model.FirewallGroup
	err := rows.Scan(&g.ID, &g.Name, &g.ScopeType, &g.ScopeID, &g.Description, &g.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to scan firewall group row: %w", err)
	}
	return &g, nil
}

// scanFirewallRule scans a single row into a FirewallRule struct
func scanFirewallRule(row scanner) (*model.FirewallRule, error) {
	var r model.FirewallRule
	var sourceCIDR, destCIDR, sourcePort, destPort, interfaceName sql.NullString
	var enabled, log int

	err := row.Scan(
		&r.ID,
		&r.Name,
		&r.Priority,
		&r.Action,
		&r.Direction,
		&sourceCIDR,
		&destCIDR,
		&r.Protocol,
		&sourcePort,
		&destPort,
		&interfaceName,
		&enabled,
		&log,
		&r.Description,
		&r.ScopeType,
		&r.ScopeID,
		&r.CreatedAt,
		&r.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan firewall rule: %w", err)
	}

	r.SourceCIDR = sourceCIDR.String
	r.DestCIDR = destCIDR.String
	r.SourcePort = sourcePort.String
	r.DestPort = destPort.String
	r.Interface = interfaceName.String
	r.Enabled = enabled == 1
	r.Log = log == 1
	return &r, nil
}

// scanFirewallRuleRow scans a row from Rows into a FirewallRule struct
func scanFirewallRuleRow(rows rowsScanner) (*model.FirewallRule, error) {
	var r model.FirewallRule
	var sourceCIDR, destCIDR, sourcePort, destPort, interfaceName sql.NullString
	var enabled, log int

	err := rows.Scan(
		&r.ID,
		&r.Name,
		&r.Priority,
		&r.Action,
		&r.Direction,
		&sourceCIDR,
		&destCIDR,
		&r.Protocol,
		&sourcePort,
		&destPort,
		&interfaceName,
		&enabled,
		&log,
		&r.Description,
		&r.ScopeType,
		&r.ScopeID,
		&r.CreatedAt,
		&r.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan firewall rule row: %w", err)
	}

	r.SourceCIDR = sourceCIDR.String
	r.DestCIDR = destCIDR.String
	r.SourcePort = sourcePort.String
	r.DestPort = destPort.String
	r.Interface = interfaceName.String
	r.Enabled = enabled == 1
	r.Log = log == 1
	return &r, nil
}

