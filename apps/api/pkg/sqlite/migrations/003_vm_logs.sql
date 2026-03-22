-- VM Logs schema
-- Stores persistent logs for virtual machines with configurable retention

-- VM logs table
CREATE TABLE IF NOT EXISTS vm_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    vmid INTEGER NOT NULL,                -- VM ID (references libvirt domain)
    level TEXT NOT NULL,                  -- 'DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL'
    source TEXT NOT NULL,                 -- 'libvirt', 'qemu', 'console', 'journal', 'system', 'config', 'network'
    message TEXT NOT NULL,                -- Log message
    metadata TEXT,                        -- JSON blob with additional context
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Indexes for VM log queries
CREATE INDEX IF NOT EXISTS idx_vm_logs_vmid ON vm_logs(vmid);
CREATE INDEX IF NOT EXISTS idx_vm_logs_level ON vm_logs(level);
CREATE INDEX IF NOT EXISTS idx_vm_logs_source ON vm_logs(source);
CREATE INDEX IF NOT EXISTS idx_vm_logs_created ON vm_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_vm_logs_vmid_created ON vm_logs(vmid, created_at DESC);

-- View for recent logs (last 24 hours)
CREATE VIEW IF NOT EXISTS vm_logs_recent AS
SELECT * FROM vm_logs
WHERE created_at > strftime('%s', 'now', '-24 hours');
