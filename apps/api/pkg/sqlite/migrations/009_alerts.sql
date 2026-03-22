-- Alerting system tables
-- Stores alert rules, notification channels, and fired alerts

-- Notification channels (email, webhook)
CREATE TABLE IF NOT EXISTS notification_channels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'email' or 'webhook'
    config TEXT NOT NULL, -- JSON: {smtp_host, smtp_port, smtp_user, smtp_pass, from_address} or {url, headers}
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL, -- 'storage_pool_usage', 'vm_stopped', 'backup_failed', 'node_offline', 'cpu_usage', 'memory_usage'
    threshold REAL, -- percentage for usage alerts, null for boolean alerts
    duration_minutes INTEGER DEFAULT 0, -- sustained duration before firing (0 = immediate)
    entity_type TEXT, -- 'vm', 'node', 'storage_pool', 'backup', null for all
    entity_id TEXT, -- specific entity ID, null for all entities of type
    channel_id TEXT NOT NULL, -- FK to notification_channels
    enabled INTEGER NOT NULL DEFAULT 1,
    last_triggered_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id)
);

-- Fired alerts (history)
CREATE TABLE IF NOT EXISTS fired_alerts (
    id TEXT PRIMARY KEY,
    rule_id TEXT NOT NULL,
    rule_name TEXT NOT NULL,
    entity_type TEXT,
    entity_id TEXT,
    entity_name TEXT,
    message TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'warning', -- 'info', 'warning', 'critical'
    status TEXT NOT NULL DEFAULT 'open', -- 'open', 'acknowledged', 'resolved'
    fired_at TEXT NOT NULL DEFAULT (datetime('now')),
    acknowledged_at TEXT,
    acknowledged_by TEXT,
    resolved_at TEXT,
    metadata TEXT, -- JSON with additional context
    FOREIGN KEY (rule_id) REFERENCES alert_rules(id)
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_alert_rules_type ON alert_rules(type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled);
CREATE INDEX IF NOT EXISTS idx_fired_alerts_rule_id ON fired_alerts(rule_id);
CREATE INDEX IF NOT EXISTS idx_fired_alerts_status ON fired_alerts(status);
CREATE INDEX IF NOT EXISTS idx_fired_alerts_fired_at ON fired_alerts(fired_at);
CREATE INDEX IF NOT EXISTS idx_fired_alerts_entity ON fired_alerts(entity_type, entity_id);
