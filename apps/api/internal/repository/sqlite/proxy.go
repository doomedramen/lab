package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// ProxyRepository handles reverse proxy host storage and retrieval.
type ProxyRepository struct {
	db *sqlitePkg.DB
}

// NewProxyRepository creates a new proxy repository.
func NewProxyRepository(db *sqlitePkg.DB) *ProxyRepository {
	return &ProxyRepository{db: db}
}

// Create saves a new proxy host to the database.
func (r *ProxyRepository) Create(ctx context.Context, host *model.ProxyHost) error {
	reqHeaders, err := marshalHeaders(host.CustomRequestHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal request headers: %w", err)
	}
	respHeaders, err := marshalHeaders(host.CustomResponseHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal response headers: %w", err)
	}

	query := `
		INSERT INTO proxy_hosts (
			id, domain, target_url, ssl_mode,
			basic_auth_enabled, basic_auth_user, basic_auth_password_hash,
			custom_request_headers, custom_response_headers,
			websocket_support, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = r.db.ExecContext(ctx, query,
		host.ID,
		host.Domain,
		host.TargetURL,
		string(host.SSLMode),
		boolToInt(host.BasicAuthEnabled),
		nullString(host.BasicAuthUser),
		nullString(host.BasicAuthPasswordHash),
		nullString(reqHeaders),
		nullString(respHeaders),
		boolToInt(host.WebsocketSupport),
		boolToInt(host.Enabled),
	)
	if err != nil {
		return fmt.Errorf("failed to create proxy host: %w", err)
	}
	return nil
}

// GetByID retrieves a proxy host by its ID.
func (r *ProxyRepository) GetByID(ctx context.Context, id string) (*model.ProxyHost, error) {
	query := `
		SELECT id, domain, target_url, ssl_mode,
		       basic_auth_enabled, basic_auth_user, basic_auth_password_hash,
		       custom_request_headers, custom_response_headers,
		       websocket_support, enabled, created_at, updated_at
		FROM proxy_hosts
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanProxyHost(row)
}

// GetByDomain retrieves a proxy host by its domain name.
func (r *ProxyRepository) GetByDomain(ctx context.Context, domain string) (*model.ProxyHost, error) {
	query := `
		SELECT id, domain, target_url, ssl_mode,
		       basic_auth_enabled, basic_auth_user, basic_auth_password_hash,
		       custom_request_headers, custom_response_headers,
		       websocket_support, enabled, created_at, updated_at
		FROM proxy_hosts
		WHERE domain = ?
	`
	row := r.db.QueryRowContext(ctx, query, domain)
	return scanProxyHost(row)
}

// List returns all proxy hosts ordered by domain.
func (r *ProxyRepository) List(ctx context.Context) ([]*model.ProxyHost, error) {
	query := `
		SELECT id, domain, target_url, ssl_mode,
		       basic_auth_enabled, basic_auth_user, basic_auth_password_hash,
		       custom_request_headers, custom_response_headers,
		       websocket_support, enabled, created_at, updated_at
		FROM proxy_hosts
		ORDER BY domain
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxy hosts: %w", err)
	}
	defer rows.Close()

	var hosts []*model.ProxyHost
	for rows.Next() {
		host, err := scanProxyHostRow(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, rows.Err()
}

// Update updates an existing proxy host.
func (r *ProxyRepository) Update(ctx context.Context, host *model.ProxyHost) error {
	reqHeaders, err := marshalHeaders(host.CustomRequestHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal request headers: %w", err)
	}
	respHeaders, err := marshalHeaders(host.CustomResponseHeaders)
	if err != nil {
		return fmt.Errorf("failed to marshal response headers: %w", err)
	}

	query := `
		UPDATE proxy_hosts
		SET domain = ?, target_url = ?, ssl_mode = ?,
		    basic_auth_enabled = ?, basic_auth_user = ?, basic_auth_password_hash = ?,
		    custom_request_headers = ?, custom_response_headers = ?,
		    websocket_support = ?, enabled = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`
	_, err = r.db.ExecContext(ctx, query,
		host.Domain,
		host.TargetURL,
		string(host.SSLMode),
		boolToInt(host.BasicAuthEnabled),
		nullString(host.BasicAuthUser),
		nullString(host.BasicAuthPasswordHash),
		nullString(reqHeaders),
		nullString(respHeaders),
		boolToInt(host.WebsocketSupport),
		boolToInt(host.Enabled),
		host.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update proxy host: %w", err)
	}
	return nil
}

// Delete removes a proxy host (and its cert via CASCADE) from the database.
func (r *ProxyRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM proxy_hosts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete proxy host: %w", err)
	}
	return nil
}

// SaveCert inserts or replaces the TLS certificate for a proxy host.
func (r *ProxyRepository) SaveCert(ctx context.Context, cert *model.ProxyCert) error {
	query := `
		INSERT INTO proxy_host_certs (id, proxy_host_id, cert_pem, key_pem, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(proxy_host_id) DO UPDATE SET
			cert_pem = excluded.cert_pem,
			key_pem  = excluded.key_pem,
			expires_at = excluded.expires_at
	`
	_, err := r.db.ExecContext(ctx, query,
		cert.ID,
		cert.ProxyHostID,
		cert.CertPEM,
		cert.KeyPEM,
		nullString(cert.ExpiresAt),
	)
	if err != nil {
		return fmt.Errorf("failed to save proxy cert: %w", err)
	}
	return nil
}

// GetCert retrieves the TLS certificate for a proxy host.
func (r *ProxyRepository) GetCert(ctx context.Context, proxyHostID string) (*model.ProxyCert, error) {
	query := `
		SELECT id, proxy_host_id, cert_pem, key_pem, expires_at, created_at
		FROM proxy_host_certs
		WHERE proxy_host_id = ?
	`
	row := r.db.QueryRowContext(ctx, query, proxyHostID)

	var c model.ProxyCert
	var expiresAt sql.NullString

	err := row.Scan(&c.ID, &c.ProxyHostID, &c.CertPEM, &c.KeyPEM, &expiresAt, &c.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get proxy cert: %w", err)
	}
	c.ExpiresAt = expiresAt.String
	return &c, nil
}

// scanProxyHost scans a single *sql.Row into a ProxyHost.
func scanProxyHost(row scanner) (*model.ProxyHost, error) {
	var h model.ProxyHost
	var basicAuthEnabled, websocketSupport, enabled int
	var basicAuthUser, basicAuthPwHash, reqHeaders, respHeaders sql.NullString

	err := row.Scan(
		&h.ID,
		&h.Domain,
		&h.TargetURL,
		&h.SSLMode,
		&basicAuthEnabled,
		&basicAuthUser,
		&basicAuthPwHash,
		&reqHeaders,
		&respHeaders,
		&websocketSupport,
		&enabled,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan proxy host: %w", err)
	}

	h.BasicAuthEnabled = basicAuthEnabled == 1
	h.WebsocketSupport = websocketSupport == 1
	h.Enabled = enabled == 1
	h.BasicAuthUser = basicAuthUser.String
	h.BasicAuthPasswordHash = basicAuthPwHash.String
	h.CustomRequestHeaders = unmarshalHeaders(reqHeaders.String)
	h.CustomResponseHeaders = unmarshalHeaders(respHeaders.String)
	return &h, nil
}

// scanProxyHostRow scans a row from *sql.Rows into a ProxyHost.
func scanProxyHostRow(rows rowsScanner) (*model.ProxyHost, error) {
	var h model.ProxyHost
	var basicAuthEnabled, websocketSupport, enabled int
	var basicAuthUser, basicAuthPwHash, reqHeaders, respHeaders sql.NullString

	err := rows.Scan(
		&h.ID,
		&h.Domain,
		&h.TargetURL,
		&h.SSLMode,
		&basicAuthEnabled,
		&basicAuthUser,
		&basicAuthPwHash,
		&reqHeaders,
		&respHeaders,
		&websocketSupport,
		&enabled,
		&h.CreatedAt,
		&h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan proxy host row: %w", err)
	}

	h.BasicAuthEnabled = basicAuthEnabled == 1
	h.WebsocketSupport = websocketSupport == 1
	h.Enabled = enabled == 1
	h.BasicAuthUser = basicAuthUser.String
	h.BasicAuthPasswordHash = basicAuthPwHash.String
	h.CustomRequestHeaders = unmarshalHeaders(reqHeaders.String)
	h.CustomResponseHeaders = unmarshalHeaders(respHeaders.String)
	return &h, nil
}

// ---- Uptime monitor methods ----

// CreateMonitor inserts a new uptime monitor.
func (r *ProxyRepository) CreateMonitor(ctx context.Context, m *model.UptimeMonitor) error {
	query := `
		INSERT INTO uptime_monitors (
			id, name, url, proxy_host_id,
			interval_seconds, timeout_seconds, expected_status_code, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		m.ID,
		m.Name,
		m.URL,
		nullString(m.ProxyHostID),
		m.IntervalSeconds,
		m.TimeoutSeconds,
		m.ExpectedStatusCode,
		boolToInt(m.Enabled),
	)
	if err != nil {
		return fmt.Errorf("failed to create uptime monitor: %w", err)
	}
	return nil
}

// GetMonitorByID retrieves an uptime monitor by ID.
func (r *ProxyRepository) GetMonitorByID(ctx context.Context, id string) (*model.UptimeMonitor, error) {
	query := `
		SELECT id, name, url, proxy_host_id,
		       interval_seconds, timeout_seconds, expected_status_code, enabled,
		       created_at, updated_at
		FROM uptime_monitors
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUptimeMonitor(row)
}

// ListMonitors returns all uptime monitors ordered by name.
func (r *ProxyRepository) ListMonitors(ctx context.Context) ([]*model.UptimeMonitor, error) {
	return r.queryMonitors(ctx, `
		SELECT id, name, url, proxy_host_id,
		       interval_seconds, timeout_seconds, expected_status_code, enabled,
		       created_at, updated_at
		FROM uptime_monitors
		ORDER BY name
	`)
}

// ListEnabledMonitors returns only enabled uptime monitors.
func (r *ProxyRepository) ListEnabledMonitors(ctx context.Context) ([]*model.UptimeMonitor, error) {
	return r.queryMonitors(ctx, `
		SELECT id, name, url, proxy_host_id,
		       interval_seconds, timeout_seconds, expected_status_code, enabled,
		       created_at, updated_at
		FROM uptime_monitors
		WHERE enabled = 1
		ORDER BY name
	`)
}

func (r *ProxyRepository) queryMonitors(ctx context.Context, query string, args ...interface{}) ([]*model.UptimeMonitor, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query uptime monitors: %w", err)
	}
	defer rows.Close()

	var monitors []*model.UptimeMonitor
	for rows.Next() {
		m, err := scanUptimeMonitorRow(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

// UpdateMonitor updates an existing uptime monitor.
func (r *ProxyRepository) UpdateMonitor(ctx context.Context, m *model.UptimeMonitor) error {
	query := `
		UPDATE uptime_monitors
		SET name = ?, url = ?,
		    interval_seconds = ?, timeout_seconds = ?,
		    expected_status_code = ?, enabled = ?,
		    updated_at = datetime('now')
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		m.Name,
		m.URL,
		m.IntervalSeconds,
		m.TimeoutSeconds,
		m.ExpectedStatusCode,
		boolToInt(m.Enabled),
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update uptime monitor: %w", err)
	}
	return nil
}

// DeleteMonitor removes an uptime monitor (and its results via CASCADE).
func (r *ProxyRepository) DeleteMonitor(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM uptime_monitors WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete uptime monitor: %w", err)
	}
	return nil
}

// ---- Uptime result methods ----

// LogUptimeResult inserts a single check result.
func (r *ProxyRepository) LogUptimeResult(ctx context.Context, res *model.UptimeResult) error {
	query := `
		INSERT INTO uptime_results (id, monitor_id, status_code, response_time_ms, success, error)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query,
		res.ID,
		res.MonitorID,
		res.StatusCode,
		res.ResponseTimeMs,
		boolToInt(res.Success),
		res.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to log uptime result: %w", err)
	}
	return nil
}

// GetUptimeHistory returns the most recent results for a monitor, newest first.
func (r *ProxyRepository) GetUptimeHistory(ctx context.Context, monitorID string, limit int) ([]*model.UptimeResult, error) {
	if limit <= 0 {
		limit = 100
	}
	query := `
		SELECT id, monitor_id, status_code, response_time_ms, success, error, checked_at
		FROM uptime_results
		WHERE monitor_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get uptime history: %w", err)
	}
	defer rows.Close()

	var results []*model.UptimeResult
	for rows.Next() {
		res, err := scanUptimeResult(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return results, rows.Err()
}

// GetUptimeStats returns aggregated statistics for a monitor.
func (r *ProxyRepository) GetUptimeStats(ctx context.Context, monitorID string) (*model.UptimeStats, error) {
	stats := &model.UptimeStats{MonitorID: monitorID}

	// Last result
	lastRow := r.db.QueryRowContext(ctx, `
		SELECT id, monitor_id, status_code, response_time_ms, success, error, checked_at
		FROM uptime_results
		WHERE monitor_id = ?
		ORDER BY checked_at DESC
		LIMIT 1
	`, monitorID)
	last, err := scanUptimeResult(lastRow)
	if err != nil && err.Error() != "no rows" {
		return nil, fmt.Errorf("failed to get last uptime result: %w", err)
	}
	stats.LastResult = last

	// 24h uptime %
	var total24h, success24h int
	r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(success), 0)
		FROM uptime_results
		WHERE monitor_id = ?
		  AND checked_at >= datetime('now', '-24 hours')
	`, monitorID).Scan(&total24h, &success24h)
	if total24h > 0 {
		stats.UptimePercent24h = float64(success24h) / float64(total24h) * 100
	}

	// 7d uptime %
	var total7d, success7d int
	r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(success), 0)
		FROM uptime_results
		WHERE monitor_id = ?
		  AND checked_at >= datetime('now', '-7 days')
	`, monitorID).Scan(&total7d, &success7d)
	if total7d > 0 {
		stats.UptimePercent7d = float64(success7d) / float64(total7d) * 100
	}

	// Average response time (24h, successful checks only)
	var avgMs sql.NullFloat64
	r.db.QueryRowContext(ctx, `
		SELECT AVG(response_time_ms)
		FROM uptime_results
		WHERE monitor_id = ?
		  AND success = 1
		  AND checked_at >= datetime('now', '-24 hours')
	`, monitorID).Scan(&avgMs)
	if avgMs.Valid {
		stats.AvgResponseMs24h = avgMs.Float64
	}

	// Recent results (last 50)
	recent, err := r.GetUptimeHistory(ctx, monitorID, 50)
	if err != nil {
		return nil, err
	}
	stats.RecentResults = recent

	return stats, nil
}

// PruneOldResults deletes results older than the given duration.
func (r *ProxyRepository) PruneOldResults(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM uptime_results
		WHERE checked_at < ?
	`, cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, fmt.Errorf("failed to prune old uptime results: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ---- scan helpers for uptime models ----

type uptimeMonitorScanner interface {
	Scan(dest ...interface{}) error
}

func scanUptimeMonitor(row uptimeMonitorScanner) (*model.UptimeMonitor, error) {
	var m model.UptimeMonitor
	var proxyHostID sql.NullString
	var enabled int
	err := row.Scan(
		&m.ID, &m.Name, &m.URL, &proxyHostID,
		&m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatusCode, &enabled,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan uptime monitor: %w", err)
	}
	m.Enabled = enabled == 1
	m.ProxyHostID = proxyHostID.String
	return &m, nil
}

func scanUptimeMonitorRow(rows *sql.Rows) (*model.UptimeMonitor, error) {
	var m model.UptimeMonitor
	var proxyHostID sql.NullString
	var enabled int
	err := rows.Scan(
		&m.ID, &m.Name, &m.URL, &proxyHostID,
		&m.IntervalSeconds, &m.TimeoutSeconds, &m.ExpectedStatusCode, &enabled,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan uptime monitor row: %w", err)
	}
	m.Enabled = enabled == 1
	m.ProxyHostID = proxyHostID.String
	return &m, nil
}

type uptimeResultScanner interface {
	Scan(dest ...interface{}) error
}

func scanUptimeResult(row uptimeResultScanner) (*model.UptimeResult, error) {
	var r model.UptimeResult
	var success int
	err := row.Scan(
		&r.ID, &r.MonitorID, &r.StatusCode, &r.ResponseTimeMs, &success, &r.Error, &r.CheckedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan uptime result: %w", err)
	}
	r.Success = success == 1
	return &r, nil
}

// marshalHeaders encodes a header map to JSON, returning "" for nil/empty maps.
func marshalHeaders(headers map[string]string) (string, error) {
	if len(headers) == 0 {
		return "", nil
	}
	b, err := json.Marshal(headers)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// unmarshalHeaders decodes a JSON string into a header map.
// Returns nil if the input is empty or invalid JSON.
func unmarshalHeaders(s string) map[string]string {
	if s == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
