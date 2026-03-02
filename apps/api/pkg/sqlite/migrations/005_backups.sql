-- Migration 005: VM Backups and Schedules
-- Adds support for VM backup management and scheduled backups

-- Backups table
CREATE TABLE IF NOT EXISTS backups (
  id TEXT PRIMARY KEY,
  vmid INTEGER NOT NULL,
  vm_name TEXT NOT NULL,
  name TEXT DEFAULT '',
  type TEXT NOT NULL DEFAULT 'full',  -- 'full', 'incremental', 'snapshot'
  status TEXT NOT NULL DEFAULT 'pending',  -- 'pending', 'running', 'completed', 'failed', 'deleting'
  size_bytes INTEGER DEFAULT 0,
  storage_pool TEXT NOT NULL,
  backup_path TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  completed_at TEXT,
  expires_at TEXT,
  error_message TEXT,
  retention_days INTEGER DEFAULT 0,  -- 0 = keep forever
  FOREIGN KEY (vmid) REFERENCES vms(vmid) ON DELETE CASCADE
);

-- Backup schedules table
CREATE TABLE IF NOT EXISTS backup_schedules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  entity_type TEXT NOT NULL DEFAULT 'vm',  -- 'vm' or 'container'
  entity_id INTEGER NOT NULL,
  storage_pool TEXT NOT NULL,
  schedule TEXT NOT NULL,  -- cron expression
  backup_type TEXT NOT NULL DEFAULT 'full',
  retention_days INTEGER DEFAULT 30,
  enabled INTEGER DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_run_at TEXT,
  next_run_at TEXT,
  total_backups INTEGER DEFAULT 0
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_backups_vmid ON backups(vmid);
CREATE INDEX IF NOT EXISTS idx_backups_status ON backups(status);
CREATE INDEX IF NOT EXISTS idx_backups_storage ON backups(storage_pool);
CREATE INDEX IF NOT EXISTS idx_backups_created ON backups(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backups_expires ON backups(expires_at);

CREATE INDEX IF NOT EXISTS idx_backup_schedules_entity ON backup_schedules(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_enabled ON backup_schedules(enabled);
CREATE INDEX IF NOT EXISTS idx_backup_schedules_next_run ON backup_schedules(next_run_at);

-- View for expired backups (for cleanup)
CREATE VIEW IF NOT EXISTS backups_expired AS
SELECT * FROM backups
WHERE expires_at IS NOT NULL 
  AND expires_at != '' 
  AND datetime(expires_at) < datetime('now');

-- View for active schedules
CREATE VIEW IF NOT EXISTS backup_schedules_active AS
SELECT * FROM backup_schedules
WHERE enabled = 1 AND next_run_at IS NOT NULL;
