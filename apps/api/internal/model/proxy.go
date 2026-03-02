package model

// ProxySSLMode defines the TLS mode for a proxy host.
type ProxySSLMode string

const (
	// ProxySSLModeNone serves traffic over plain HTTP.
	ProxySSLModeNone ProxySSLMode = "none"
	// ProxySSLModeSelfSigned uses an auto-generated self-signed certificate (stdlib, no deps).
	ProxySSLModeSelfSigned ProxySSLMode = "self_signed"
	// ProxySSLModeACME uses Let's Encrypt / ZeroSSL via ACME (certmagic, future).
	ProxySSLModeACME ProxySSLMode = "acme"
	// ProxySSLModeCustom uses a user-uploaded certificate and private key.
	ProxySSLModeCustom ProxySSLMode = "custom"
)

// ProxyHost represents a reverse proxy host entry.
type ProxyHost struct {
	ID                    string            `json:"id"`
	Domain                string            `json:"domain"`
	TargetURL             string            `json:"target_url"`
	SSLMode               ProxySSLMode      `json:"ssl_mode"`
	BasicAuthEnabled      bool              `json:"basic_auth_enabled"`
	BasicAuthUser         string            `json:"basic_auth_user"`
	BasicAuthPasswordHash string            `json:"basic_auth_password_hash"`
	CustomRequestHeaders  map[string]string `json:"custom_request_headers"`
	CustomResponseHeaders map[string]string `json:"custom_response_headers"`
	WebsocketSupport      bool              `json:"websocket_support"`
	Enabled               bool              `json:"enabled"`
	CreatedAt             string            `json:"created_at"`
	UpdatedAt             string            `json:"updated_at"`
}

// ProxyHostCreateRequest is the input for creating a new proxy host.
type ProxyHostCreateRequest struct {
	Domain                string            `json:"domain"`
	TargetURL             string            `json:"target_url"`
	SSLMode               ProxySSLMode      `json:"ssl_mode"`
	BasicAuthEnabled      bool              `json:"basic_auth_enabled"`
	BasicAuthUser         string            `json:"basic_auth_user"`
	BasicAuthPassword     string            `json:"basic_auth_password"`
	CustomRequestHeaders  map[string]string `json:"custom_request_headers"`
	CustomResponseHeaders map[string]string `json:"custom_response_headers"`
	WebsocketSupport      bool              `json:"websocket_support"`
	Enabled               bool              `json:"enabled"`
}

// ProxyHostUpdateRequest is the input for updating an existing proxy host.
type ProxyHostUpdateRequest struct {
	ID                    string            `json:"id"`
	Domain                string            `json:"domain"`
	TargetURL             string            `json:"target_url"`
	SSLMode               ProxySSLMode      `json:"ssl_mode"`
	BasicAuthEnabled      bool              `json:"basic_auth_enabled"`
	BasicAuthUser         string            `json:"basic_auth_user"`
	BasicAuthPassword     string            `json:"basic_auth_password"`
	CustomRequestHeaders  map[string]string `json:"custom_request_headers"`
	CustomResponseHeaders map[string]string `json:"custom_response_headers"`
	WebsocketSupport      bool              `json:"websocket_support"`
	Enabled               bool              `json:"enabled"`
}

// ProxyCert stores TLS certificate material for a proxy host.
// Used for ssl_mode=custom (user-uploaded) and ssl_mode=self_signed (auto-generated).
type ProxyCert struct {
	ID          string `json:"id"`
	ProxyHostID string `json:"proxy_host_id"`
	CertPEM     string `json:"cert_pem"`
	KeyPEM      string `json:"key_pem"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

// ProxyStatus is a live status snapshot for a proxy host.
type ProxyStatus struct {
	ProxyHostID      string `json:"proxy_host_id"`
	IsRunning        bool   `json:"is_running"`
	BackendReachable bool   `json:"backend_reachable"`
	CertExpiry       string `json:"cert_expiry"`
	LastCheckAt      string `json:"last_check_at"`
}

// UptimeMonitorStatus is the current health status of a monitor.
type UptimeMonitorStatus string

const (
	UptimeStatusPending UptimeMonitorStatus = "pending" // no checks yet
	UptimeStatusUp      UptimeMonitorStatus = "up"
	UptimeStatusDown    UptimeMonitorStatus = "down"
	UptimeStatusPaused  UptimeMonitorStatus = "paused" // monitoring disabled
)

// UptimeMonitor represents a configured health check for a URL.
type UptimeMonitor struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	URL                string `json:"url"`
	ProxyHostID        string `json:"proxy_host_id,omitempty"` // optional link to a proxy host
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Enabled            bool   `json:"enabled"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// UptimeMonitorCreateRequest is the input for creating a new monitor.
type UptimeMonitorCreateRequest struct {
	Name               string `json:"name"`
	URL                string `json:"url"`
	ProxyHostID        string `json:"proxy_host_id,omitempty"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Enabled            bool   `json:"enabled"`
}

// UptimeMonitorUpdateRequest is the input for updating a monitor.
type UptimeMonitorUpdateRequest struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	URL                string `json:"url"`
	IntervalSeconds    int    `json:"interval_seconds"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ExpectedStatusCode int    `json:"expected_status_code"`
	Enabled            bool   `json:"enabled"`
}

// UptimeResult is a single health check outcome.
type UptimeResult struct {
	ID             string `json:"id"`
	MonitorID      string `json:"monitor_id"`
	StatusCode     int    `json:"status_code"`
	ResponseTimeMs int64  `json:"response_time_ms"`
	Success        bool   `json:"success"`
	Error          string `json:"error"`
	CheckedAt      string `json:"checked_at"`
}

// UptimeStats provides aggregated statistics for a monitor.
type UptimeStats struct {
	MonitorID         string          `json:"monitor_id"`
	Status            UptimeMonitorStatus `json:"status"`
	UptimePercent24h  float64         `json:"uptime_percent_24h"`
	UptimePercent7d   float64         `json:"uptime_percent_7d"`
	AvgResponseMs24h  float64         `json:"avg_response_ms_24h"`
	LastResult        *UptimeResult   `json:"last_result,omitempty"`
	RecentResults     []*UptimeResult `json:"recent_results"`
}
