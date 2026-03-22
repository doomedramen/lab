-- Migration 007: Virtual Networks and Firewall
-- Adds support for virtual network and firewall management

-- Virtual networks table
CREATE TABLE IF NOT EXISTS virtual_networks (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  type TEXT NOT NULL DEFAULT 'bridge',  -- 'bridge', 'vlan', 'vxlan', 'ovs', 'macvlan', 'ipvlan'
  status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'inactive', 'error'
  bridge_name TEXT,
  vlan_id INTEGER DEFAULT 0,
  subnet TEXT,  -- CIDR notation
  gateway TEXT,
  dhcp_enabled INTEGER DEFAULT 0,
  dhcp_range_start TEXT,
  dhcp_range_end TEXT,
  dns_servers TEXT,  -- Comma-separated
  isolated INTEGER DEFAULT 0,
  mtu INTEGER DEFAULT 1500,
  description TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  interface_count INTEGER DEFAULT 0
);

-- Network interfaces table
CREATE TABLE IF NOT EXISTS network_interfaces (
  id TEXT PRIMARY KEY,
  network_id TEXT NOT NULL,
  name TEXT NOT NULL,
  mac_address TEXT NOT NULL,
  ip_address TEXT,
  interface_type TEXT NOT NULL DEFAULT 'vm',  -- 'vm', 'container', 'host'
  entity_id INTEGER DEFAULT 0,
  entity_type TEXT NOT NULL DEFAULT 'vm',
  enabled INTEGER DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (network_id) REFERENCES virtual_networks(id) ON DELETE CASCADE
);

-- Firewall rules table
CREATE TABLE IF NOT EXISTS firewall_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 100,
  action TEXT NOT NULL DEFAULT 'accept',  -- 'accept', 'drop', 'reject', 'log'
  direction TEXT NOT NULL DEFAULT 'both',  -- 'inbound', 'outbound', 'both'
  source_cidr TEXT,
  dest_cidr TEXT,
  protocol TEXT DEFAULT 'any',  -- 'tcp', 'udp', 'icmp', 'any'
  source_port TEXT,
  dest_port TEXT,
  interface TEXT,
  enabled INTEGER DEFAULT 1,
  log INTEGER DEFAULT 0,
  description TEXT,
  scope_type TEXT DEFAULT 'global',  -- 'global', 'node', 'network', 'vm'
  scope_id TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Firewall groups table
CREATE TABLE IF NOT EXISTS firewall_groups (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  scope_type TEXT DEFAULT 'global',
  scope_id TEXT,
  description TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Firewall group rules mapping
CREATE TABLE IF NOT EXISTS firewall_group_rules (
  group_id TEXT NOT NULL,
  rule_id TEXT NOT NULL,
  PRIMARY KEY (group_id, rule_id),
  FOREIGN KEY (group_id) REFERENCES firewall_groups(id) ON DELETE CASCADE,
  FOREIGN KEY (rule_id) REFERENCES firewall_rules(id) ON DELETE CASCADE
);

-- DHCP static leases table
CREATE TABLE IF NOT EXISTS dhcp_static_leases (
  id TEXT PRIMARY KEY,
  network_id TEXT NOT NULL,
  mac_address TEXT NOT NULL,
  ip_address TEXT NOT NULL,
  hostname TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  FOREIGN KEY (network_id) REFERENCES virtual_networks(id) ON DELETE CASCADE,
  UNIQUE(network_id, mac_address)
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_virtual_networks_type ON virtual_networks(type);
CREATE INDEX IF NOT EXISTS idx_virtual_networks_status ON virtual_networks(status);
CREATE INDEX IF NOT EXISTS idx_network_interfaces_network ON network_interfaces(network_id);
CREATE INDEX IF NOT EXISTS idx_network_interfaces_entity ON network_interfaces(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_firewall_rules_scope ON firewall_rules(scope_type, scope_id);
CREATE INDEX IF NOT EXISTS idx_firewall_rules_priority ON firewall_rules(priority);
CREATE INDEX IF NOT EXISTS idx_firewall_rules_enabled ON firewall_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_dhcp_leases_network ON dhcp_static_leases(network_id);

-- View for active networks
CREATE VIEW IF NOT EXISTS virtual_networks_active AS
SELECT * FROM virtual_networks WHERE status = 'active';

-- View for enabled firewall rules
CREATE VIEW IF NOT EXISTS firewall_rules_enabled AS
SELECT * FROM firewall_rules WHERE enabled = 1 ORDER BY priority ASC;

-- View for firewall rules by scope
CREATE VIEW IF NOT EXISTS firewall_rules_by_scope AS
SELECT 
  scope_type,
  scope_id,
  COUNT(*) as rule_count,
  SUM(CASE WHEN enabled = 1 THEN 1 ELSE 0 END) as enabled_count
FROM firewall_rules
GROUP BY scope_type, scope_id;
