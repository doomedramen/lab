-- Uptime monitoring: monitors (config) and results (history)

CREATE TABLE IF NOT EXISTS uptime_monitors (
    id                   TEXT    PRIMARY KEY,
    name                 TEXT    NOT NULL,
    url                  TEXT    NOT NULL,
    proxy_host_id        TEXT    REFERENCES proxy_hosts(id) ON DELETE CASCADE,
    interval_seconds     INTEGER NOT NULL DEFAULT 60,
    timeout_seconds      INTEGER NOT NULL DEFAULT 10,
    expected_status_code INTEGER NOT NULL DEFAULT 200,
    enabled              INTEGER NOT NULL DEFAULT 1,
    created_at           DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at           DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_uptime_monitors_proxy_host_id ON uptime_monitors(proxy_host_id);
CREATE INDEX IF NOT EXISTS idx_uptime_monitors_enabled       ON uptime_monitors(enabled);

CREATE TABLE IF NOT EXISTS uptime_results (
    id               TEXT    PRIMARY KEY,
    monitor_id       TEXT    NOT NULL REFERENCES uptime_monitors(id) ON DELETE CASCADE,
    status_code      INTEGER NOT NULL DEFAULT 0,
    response_time_ms INTEGER NOT NULL DEFAULT 0,
    success          INTEGER NOT NULL DEFAULT 0,
    error            TEXT    NOT NULL DEFAULT '',
    checked_at       DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_uptime_results_monitor_id  ON uptime_results(monitor_id);
CREATE INDEX IF NOT EXISTS idx_uptime_results_checked_at  ON uptime_results(checked_at);
