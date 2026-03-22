-- Session management for tracking active user sessions
-- Allows users to view and revoke sessions (e.g., lost device)

-- Sessions table - tracks each login session
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,                 -- UUID matching refresh token ID
    user_id TEXT NOT NULL,
    jti TEXT UNIQUE NOT NULL,            -- JWT ID claim for access token revocation
    ip_address TEXT,
    user_agent TEXT,
    device_name TEXT,                    -- Parsed from user agent (e.g., "Chrome on macOS")
    issued_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for session queries
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_jti ON sessions(jti);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- Clean up expired sessions periodically (can be called by cleanup job)
-- Note: SQLite doesn't have scheduled jobs, so this is done via DELETE query
