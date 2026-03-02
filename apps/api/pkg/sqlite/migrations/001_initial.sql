-- Enable WAL mode for better concurrent access
PRAGMA journal_mode = WAL;

-- Metrics table for time-series data (CPU, memory, disk, network)
CREATE TABLE IF NOT EXISTS metrics (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER NOT NULL,              -- unix timestamp (seconds)
  node_id TEXT NOT NULL,            -- host node identifier
  resource_type TEXT NOT NULL,      -- 'cpu', 'memory', 'disk', 'network_in', 'network_out'
  resource_id TEXT,                 -- VM ID, CT ID, or NULL for host-level metrics
  value REAL NOT NULL,              -- metric value
  unit TEXT NOT NULL DEFAULT '',    -- unit of measurement (%, GB, MB/s, etc)
  created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Events table for logs and significant occurrences
CREATE TABLE IF NOT EXISTS events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts INTEGER NOT NULL,              -- unix timestamp (seconds)
  node_id TEXT NOT NULL,            -- host node identifier
  resource_id TEXT,                 -- VM ID, CT ID, or NULL for host-level events
  event_type TEXT NOT NULL,         -- 'vm_start', 'vm_stop', 'error', 'alert', etc
  severity TEXT NOT NULL DEFAULT 'info',  -- 'info', 'warning', 'error', 'critical'
  message TEXT NOT NULL,            -- human-readable message
  metadata TEXT,                    -- JSON blob for extra context
  created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_metrics_ts ON metrics(ts DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_node ON metrics(node_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_resource ON metrics(resource_type, resource_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_node_type ON metrics(node_id, resource_type, ts DESC);

CREATE INDEX IF NOT EXISTS idx_events_ts ON events(ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_node ON events(node_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type, ts DESC);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity, ts DESC);

-- Retention: view for metrics older than 30 days (for cleanup)
CREATE VIEW IF NOT EXISTS metrics_old AS
SELECT * FROM metrics WHERE ts < (strftime('%s', 'now') - (30 * 86400));

-- Retention: view for events older than 90 days (for cleanup)
CREATE VIEW IF NOT EXISTS events_old AS
SELECT * FROM events WHERE ts < (strftime('%s', 'now') - (90 * 86400));
