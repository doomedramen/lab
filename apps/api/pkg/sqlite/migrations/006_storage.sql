-- Migration 006: Storage Pools and Disks
-- Adds support for storage pool and disk image management

-- Storage pools table
CREATE TABLE IF NOT EXISTS storage_pools (
  id TEXT PRIMARY KEY,
  name TEXT UNIQUE NOT NULL,
  type TEXT NOT NULL,  -- 'dir', 'lvm', 'zfs', 'nfs', 'iscsi', 'ceph', 'gluster'
  status TEXT NOT NULL DEFAULT 'active',  -- 'active', 'inactive', 'maintenance', 'error'
  path TEXT,  -- Local path or remote export
  capacity_bytes INTEGER DEFAULT 0,
  used_bytes INTEGER DEFAULT 0,
  available_bytes INTEGER DEFAULT 0,
  options TEXT,  -- JSON for type-specific options
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  enabled INTEGER DEFAULT 1,
  disk_count INTEGER DEFAULT 0,
  description TEXT
);

-- Storage disks table
CREATE TABLE IF NOT EXISTS storage_disks (
  id TEXT PRIMARY KEY,
  pool_id TEXT NOT NULL,
  name TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  format TEXT NOT NULL DEFAULT 'qcow2',  -- 'qcow2', 'raw', 'vmdk', 'vdi', 'vhdx'
  bus TEXT NOT NULL DEFAULT 'virtio',  -- 'virtio', 'sata', 'scsi', 'ide', 'usb', 'nvme'
  vmid INTEGER DEFAULT 0,  -- 0 = unassigned
  path TEXT,  -- Full path to disk image
  used_bytes INTEGER DEFAULT 0,  -- Actual usage for thin provisioned
  sparse INTEGER DEFAULT 1,  -- Thin provisioned
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  description TEXT,
  FOREIGN KEY (pool_id) REFERENCES storage_pools(id) ON DELETE CASCADE
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_storage_pools_type ON storage_pools(type);
CREATE INDEX IF NOT EXISTS idx_storage_pools_status ON storage_pools(status);
CREATE INDEX IF NOT EXISTS idx_storage_pools_enabled ON storage_pools(enabled);

CREATE INDEX IF NOT EXISTS idx_storage_disks_pool ON storage_disks(pool_id);
CREATE INDEX IF NOT EXISTS idx_storage_disks_vmid ON storage_disks(vmid);
CREATE INDEX IF NOT EXISTS idx_storage_disks_format ON storage_disks(format);

-- View for active storage pools
CREATE VIEW IF NOT EXISTS storage_pools_active AS
SELECT * FROM storage_pools WHERE enabled = 1 AND status = 'active';

-- View for unassigned disks
CREATE VIEW IF NOT EXISTS storage_disks_unassigned AS
SELECT * FROM storage_disks WHERE vmid = 0 OR vmid IS NULL;

-- View for storage pool usage summary
CREATE VIEW IF NOT EXISTS storage_pool_usage AS
SELECT 
  sp.id,
  sp.name,
  sp.type,
  sp.capacity_bytes,
  sp.used_bytes,
  sp.available_bytes,
  ROUND(100.0 * sp.used_bytes / NULLIF(sp.capacity_bytes, 0), 2) as usage_percent,
  COUNT(sd.id) as disk_count
FROM storage_pools sp
LEFT JOIN storage_disks sd ON sp.id = sd.pool_id
GROUP BY sp.id, sp.name, sp.type, sp.capacity_bytes, sp.used_bytes, sp.available_bytes;
