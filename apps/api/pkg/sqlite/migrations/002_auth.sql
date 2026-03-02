-- Authentication schema
-- Supports users, sessions, API keys, and audit logging

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',  -- 'admin', 'operator', 'viewer'
    mfa_secret TEXT,                       -- TOTP secret (encrypted at rest in production)
    mfa_enabled BOOLEAN DEFAULT FALSE,
    mfa_backup_codes TEXT,                 -- JSON array of backup codes (hashed)
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),
    last_login_at INTEGER,
    is_active BOOLEAN DEFAULT TRUE
);

-- Refresh tokens table
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- API keys table (for CLI/automation)
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    prefix TEXT NOT NULL,                 -- First 8 chars for identification (e.g., labkey_abc123)
    permissions TEXT,                     -- JSON array: ["vm:start", "vm:stop", "vm:read"]
    last_used_at INTEGER,
    expires_at INTEGER,                   -- NULL = never expires
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT,
    action TEXT NOT NULL,                 -- e.g., 'user.login', 'vm.start', 'api_key.create'
    resource_type TEXT,                   -- e.g., 'user', 'vm', 'api_key'
    resource_id TEXT,                     -- ID of the affected resource
    details TEXT,                         -- JSON blob with additional context
    ip_address TEXT,
    user_agent TEXT,
    status TEXT NOT NULL DEFAULT 'success',  -- 'success', 'failure'
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

-- Indexes for auth queries
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);

-- View for active sessions (non-revoked, non-expired refresh tokens)
CREATE VIEW IF NOT EXISTS active_sessions AS
SELECT 
    rt.id AS session_id,
    rt.user_id,
    u.email,
    rt.created_at,
    rt.expires_at
FROM refresh_tokens rt
JOIN users u ON rt.user_id = u.id
WHERE rt.revoked = FALSE AND rt.expires_at > strftime('%s', 'now');

-- View for active API keys
CREATE VIEW IF NOT EXISTS active_api_keys AS
SELECT 
    id,
    user_id,
    name,
    prefix,
    permissions,
    last_used_at,
    expires_at,
    created_at
FROM api_keys
WHERE (expires_at IS NULL OR expires_at > strftime('%s', 'now'));
