-- Migration 014: GitOps Core
-- Creates tables for GitOps configuration and state tracking

-- GitOps configurations (Git repositories to watch)
CREATE TABLE IF NOT EXISTS gitops_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    git_url TEXT NOT NULL,
    git_branch TEXT DEFAULT 'main',
    git_path TEXT DEFAULT '/',
    sync_interval INTEGER DEFAULT 300,  -- seconds (default 5 min)
    last_sync DATETIME,
    last_sync_hash TEXT,
    status TEXT DEFAULT 'Pending',  -- Healthy, OutOfSync, Failed, Pending
    status_message TEXT,
    enabled INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    next_sync DATETIME,
    sync_retries INTEGER DEFAULT 0,
    max_sync_retries INTEGER DEFAULT 3
);

-- GitOps resources (parsed manifests from Git)
CREATE TABLE IF NOT EXISTS gitops_resources (
    id TEXT PRIMARY KEY,
    config_id TEXT NOT NULL,
    kind TEXT NOT NULL,  -- VirtualMachine, Container, Network, etc.
    name TEXT NOT NULL,
    namespace TEXT DEFAULT 'default',
    manifest_path TEXT NOT NULL,  -- Path in Git repo
    manifest_hash TEXT NOT NULL,  -- Hash of manifest content
    spec TEXT,  -- JSON blob
    status TEXT DEFAULT 'Pending',
    status_message TEXT,
    last_applied DATETIME,
    last_diff TEXT,  -- Human-readable diff
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (config_id) REFERENCES gitops_configs(id) ON DELETE CASCADE,
    UNIQUE(config_id, manifest_path)
);

-- GitOps sync logs (audit trail)
CREATE TABLE IF NOT EXISTS gitops_sync_logs (
    id TEXT PRIMARY KEY,
    config_id TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration INTEGER,  -- seconds
    status TEXT,
    message TEXT,
    commit_hash TEXT,
    resources_scanned INTEGER DEFAULT 0,
    resources_created INTEGER DEFAULT 0,
    resources_updated INTEGER DEFAULT 0,
    resources_deleted INTEGER DEFAULT 0,
    resources_failed INTEGER DEFAULT 0,
    FOREIGN KEY (config_id) REFERENCES gitops_configs(id) ON DELETE CASCADE
);

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_gitops_configs_enabled ON gitops_configs(enabled);
CREATE INDEX IF NOT EXISTS idx_gitops_configs_next_sync ON gitops_configs(next_sync);
CREATE INDEX IF NOT EXISTS idx_gitops_configs_status ON gitops_configs(status);
CREATE INDEX IF NOT EXISTS idx_gitops_resources_config ON gitops_resources(config_id);
CREATE INDEX IF NOT EXISTS idx_gitops_resources_kind ON gitops_resources(kind);
CREATE INDEX IF NOT EXISTS idx_gitops_resources_status ON gitops_resources(status);
CREATE INDEX IF NOT EXISTS idx_gitops_sync_logs_config ON gitops_sync_logs(config_id);
CREATE INDEX IF NOT EXISTS idx_gitops_sync_logs_time ON gitops_sync_logs(start_time DESC);
