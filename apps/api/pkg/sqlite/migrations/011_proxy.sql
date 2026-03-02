-- Reverse proxy host management
-- Each row represents a domain-based proxy rule with optional TLS and basic auth

CREATE TABLE IF NOT EXISTS proxy_hosts (
    id TEXT PRIMARY KEY,
    domain TEXT NOT NULL UNIQUE,                     -- e.g. "app.example.com"
    target_url TEXT NOT NULL,                        -- e.g. "http://192.168.1.100:3000"
    ssl_mode TEXT NOT NULL DEFAULT 'none',           -- 'none' | 'self_signed' | 'acme' | 'custom'
    basic_auth_enabled INTEGER NOT NULL DEFAULT 0,
    basic_auth_user TEXT,
    basic_auth_password_hash TEXT,                   -- bcrypt hash
    custom_request_headers TEXT,                     -- JSON {"Header-Name": "value"}
    custom_response_headers TEXT,                    -- JSON {"Header-Name": "value"}
    websocket_support INTEGER NOT NULL DEFAULT 1,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_proxy_hosts_domain ON proxy_hosts(domain);
CREATE INDEX IF NOT EXISTS idx_proxy_hosts_enabled ON proxy_hosts(enabled);

-- Stores TLS certificate data for 'custom' and 'self_signed' SSL modes.
-- ACME certificates are managed by certmagic on disk and do not appear here.
CREATE TABLE IF NOT EXISTS proxy_host_certs (
    id TEXT PRIMARY KEY,
    proxy_host_id TEXT NOT NULL UNIQUE,
    cert_pem TEXT NOT NULL,
    key_pem TEXT NOT NULL,
    expires_at TEXT,                                 -- ISO-8601 expiry timestamp
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (proxy_host_id) REFERENCES proxy_hosts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_proxy_host_certs_host ON proxy_host_certs(proxy_host_id);
