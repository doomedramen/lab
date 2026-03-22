-- Migration 008: Task Tracking System
-- Adds support for tracking async operations (backup, restore, snapshot, clone, etc.)

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,              -- 'backup', 'restore', 'snapshot_create', 'snapshot_delete', 'snapshot_restore', 'clone', 'migration'
  status TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'running', 'completed', 'failed', 'cancelled'
  progress INTEGER DEFAULT 0,      -- 0-100 percentage
  message TEXT DEFAULT '',         -- Human-readable status message
  resource_type TEXT NOT NULL,     -- 'vm', 'container', 'stack', 'backup', 'snapshot'
  resource_id TEXT NOT NULL,       -- Resource identifier
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  completed_at TEXT,
  error TEXT                       -- Error message if failed
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_type ON tasks(type);
CREATE INDEX IF NOT EXISTS idx_tasks_resource ON tasks(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at DESC);

-- View for active (non-terminal) tasks
CREATE VIEW IF NOT EXISTS tasks_active AS
SELECT * FROM tasks
WHERE status IN ('pending', 'running');
