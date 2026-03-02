-- Migration 004: VM Snapshots
-- Adds support for VM snapshot management

-- VM Snapshots table
CREATE TABLE IF NOT EXISTS vm_snapshots (
  id TEXT PRIMARY KEY,
  vmid INTEGER NOT NULL,
  name TEXT NOT NULL,
  description TEXT DEFAULT '',
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  parent_id TEXT,
  size_bytes INTEGER DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'ready',  -- 'creating', 'ready', 'deleting', 'error'
  vm_state TEXT NOT NULL DEFAULT 'stopped',  -- 'running', 'stopped'
  has_children INTEGER DEFAULT 0,
  snapshot_path TEXT,  -- Path to snapshot metadata in libvirt
  FOREIGN KEY (parent_id) REFERENCES vm_snapshots(id) ON DELETE SET NULL
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_vm_snapshots_vmid ON vm_snapshots(vmid);
CREATE INDEX IF NOT EXISTS idx_vm_snapshots_parent ON vm_snapshots(parent_id);
CREATE INDEX IF NOT EXISTS idx_vm_snapshots_status ON vm_snapshots(status);
CREATE INDEX IF NOT EXISTS idx_vm_snapshots_created ON vm_snapshots(created_at DESC);

-- Update parent's has_children when inserting a snapshot with a parent
CREATE TRIGGER IF NOT EXISTS update_snapshot_has_children_insert
AFTER INSERT ON vm_snapshots
WHEN NEW.parent_id IS NOT NULL
BEGIN
  UPDATE vm_snapshots SET has_children = 1 WHERE id = NEW.parent_id;
END;

-- Update parent's has_children when deleting a snapshot
CREATE TRIGGER IF NOT EXISTS update_snapshot_has_children_delete
AFTER DELETE ON vm_snapshots
WHEN OLD.parent_id IS NOT NULL
BEGIN
  -- Check if parent has any other children
  UPDATE vm_snapshots 
  SET has_children = (SELECT COUNT(*) FROM vm_snapshots WHERE parent_id = OLD.id)
  WHERE id = OLD.parent_id;
END;
