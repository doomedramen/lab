package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

// UptimeAlertSender is an optional dependency that ProxyService uses to fire
// uptime failure alerts. Implemented by AlertService.
type UptimeAlertSender interface {
	// FireUptimeAlert is called when a monitor has had consecutive failures.
	FireUptimeAlert(ctx context.Context, monitorID, monitorName, url string, consecutiveFailures int, lastError string)
}

// UptimeMonitorFailure describes a monitor that is currently failing.
type UptimeMonitorFailure struct {
	MonitorID           string
	MonitorName         string
	URL                 string
	ConsecutiveFailures int
	LastError           string
}

// ProxyService manages the in-process reverse proxy: HTTP/HTTPS servers,
// dynamic route table, TLS certificate lifecycle, and uptime monitoring.
type ProxyService struct {
	repo      repository.ProxyRepository
	httpPort  int
	httpsPort int

	mu     sync.RWMutex
	routes map[string]*proxyEntry // domain -> entry

	httpSrv  *http.Server
	httpsSrv *http.Server

	// Uptime monitoring
	alertSender      UptimeAlertSender
	monitorMu        sync.Mutex
	consecutiveFails map[string]int // monitorID -> consecutive failure count
	lastRun          map[string]time.Time
	monitorCancel    context.CancelFunc
	monitorWg        sync.WaitGroup
}

// proxyEntry bundles the model record with its live *httputil.ReverseProxy.
type proxyEntry struct {
	host  *model.ProxyHost
	proxy *httputil.ReverseProxy
}

// NewProxyService creates a new proxy service.
func NewProxyService(repo repository.ProxyRepository, httpPort, httpsPort int) *ProxyService {
	return &ProxyService{
		repo:             repo,
		httpPort:         httpPort,
		httpsPort:        httpsPort,
		routes:           make(map[string]*proxyEntry),
		consecutiveFails: make(map[string]int),
		lastRun:          make(map[string]time.Time),
	}
}

// WithAlertSender sets the optional alert sender used for uptime failure notifications.
func (s *ProxyService) WithAlertSender(sender UptimeAlertSender) *ProxyService {
	s.alertSender = sender
	return s
}

// Start loads routes from the database, starts the HTTP/HTTPS server(s),
// and starts the uptime monitoring loop. It is non-blocking.
func (s *ProxyService) Start(ctx context.Context) error {
	if err := s.reloadRoutes(ctx); err != nil {
		return fmt.Errorf("proxy: failed to load routes: %w", err)
	}

	// HTTP server — always started (handles plain-HTTP and ACME HTTP-01)
	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.httpPort),
		Handler:      s.buildHTTPHandler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // websocket-friendly
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("proxy: HTTP server listening", "port", s.httpPort)
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("proxy: HTTP server error", "error", err)
		}
	}()

	// HTTPS server — started only if there are any TLS-enabled hosts
	if s.hasTLSHosts() {
		if err := s.startHTTPS(ctx); err != nil {
			slog.Warn("proxy: HTTPS server failed to start", "error", err)
		}
	}

	// Start uptime monitoring loop
	monCtx, cancel := context.WithCancel(context.Background())
	s.monitorCancel = cancel
	s.monitorWg.Add(1)
	go s.monitoringLoop(monCtx)

	return nil
}

// Stop gracefully shuts down the proxy servers and monitoring loop.
func (s *ProxyService) Stop() {
	// Stop monitoring loop first
	if s.monitorCancel != nil {
		s.monitorCancel()
		s.monitorWg.Wait()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			slog.Warn("proxy: HTTP server shutdown error", "error", err)
		}
	}
	if s.httpsSrv != nil {
		if err := s.httpsSrv.Shutdown(ctx); err != nil {
			slog.Warn("proxy: HTTPS server shutdown error", "error", err)
		}
	}
}

// ---- Uptime monitor CRUD ----

// CreateMonitor creates a new uptime monitor.
func (s *ProxyService) CreateMonitor(ctx context.Context, req *model.UptimeMonitorCreateRequest) (*model.UptimeMonitor, error) {
	m := &model.UptimeMonitor{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		URL:                req.URL,
		ProxyHostID:        req.ProxyHostID,
		IntervalSeconds:    req.IntervalSeconds,
		TimeoutSeconds:     req.TimeoutSeconds,
		ExpectedStatusCode: req.ExpectedStatusCode,
		Enabled:            req.Enabled,
	}
	if m.IntervalSeconds <= 0 {
		m.IntervalSeconds = 60
	}
	if m.TimeoutSeconds <= 0 {
		m.TimeoutSeconds = 10
	}
	if m.ExpectedStatusCode <= 0 {
		m.ExpectedStatusCode = 200
	}
	if err := s.repo.CreateMonitor(ctx, m); err != nil {
		return nil, fmt.Errorf("proxy: failed to create monitor: %w", err)
	}
	return m, nil
}

// GetMonitor retrieves an uptime monitor by ID.
func (s *ProxyService) GetMonitor(ctx context.Context, id string) (*model.UptimeMonitor, error) {
	m, err := s.repo.GetMonitorByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get monitor: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("proxy: monitor not found: %s", id)
	}
	return m, nil
}

// ListMonitors returns all uptime monitors.
func (s *ProxyService) ListMonitors(ctx context.Context) ([]*model.UptimeMonitor, error) {
	monitors, err := s.repo.ListMonitors(ctx)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to list monitors: %w", err)
	}
	return monitors, nil
}

// UpdateMonitor updates an existing uptime monitor.
func (s *ProxyService) UpdateMonitor(ctx context.Context, req *model.UptimeMonitorUpdateRequest) (*model.UptimeMonitor, error) {
	m, err := s.repo.GetMonitorByID(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get monitor: %w", err)
	}
	if m == nil {
		return nil, fmt.Errorf("proxy: monitor not found: %s", req.ID)
	}
	m.Name = req.Name
	m.URL = req.URL
	m.IntervalSeconds = req.IntervalSeconds
	m.TimeoutSeconds = req.TimeoutSeconds
	m.ExpectedStatusCode = req.ExpectedStatusCode
	m.Enabled = req.Enabled
	if m.IntervalSeconds <= 0 {
		m.IntervalSeconds = 60
	}
	if m.TimeoutSeconds <= 0 {
		m.TimeoutSeconds = 10
	}
	if m.ExpectedStatusCode <= 0 {
		m.ExpectedStatusCode = 200
	}
	if err := s.repo.UpdateMonitor(ctx, m); err != nil {
		return nil, fmt.Errorf("proxy: failed to update monitor: %w", err)
	}
	// Reset consecutive failure count on update (URL or interval may have changed).
	s.monitorMu.Lock()
	delete(s.consecutiveFails, req.ID)
	delete(s.lastRun, req.ID)
	s.monitorMu.Unlock()
	return m, nil
}

// DeleteMonitor removes an uptime monitor and its results.
func (s *ProxyService) DeleteMonitor(ctx context.Context, id string) error {
	if err := s.repo.DeleteMonitor(ctx, id); err != nil {
		return fmt.Errorf("proxy: failed to delete monitor: %w", err)
	}
	s.monitorMu.Lock()
	delete(s.consecutiveFails, id)
	delete(s.lastRun, id)
	s.monitorMu.Unlock()
	return nil
}

// GetMonitorHistory returns recent check results for a monitor.
func (s *ProxyService) GetMonitorHistory(ctx context.Context, monitorID string, limit int) ([]*model.UptimeResult, error) {
	results, err := s.repo.GetUptimeHistory(ctx, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get monitor history: %w", err)
	}
	return results, nil
}

// GetMonitorStats returns aggregated statistics for a monitor.
func (s *ProxyService) GetMonitorStats(ctx context.Context, monitorID string) (*model.UptimeStats, error) {
	m, err := s.repo.GetMonitorByID(ctx, monitorID)
	if err != nil || m == nil {
		return nil, fmt.Errorf("proxy: monitor not found: %s", monitorID)
	}
	stats, err := s.repo.GetUptimeStats(ctx, monitorID)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get monitor stats: %w", err)
	}
	// Compute derived status.
	switch {
	case !m.Enabled:
		stats.Status = model.UptimeStatusPaused
	case stats.LastResult == nil:
		stats.Status = model.UptimeStatusPending
	case stats.LastResult.Success:
		stats.Status = model.UptimeStatusUp
	default:
		stats.Status = model.UptimeStatusDown
	}
	return stats, nil
}

// GetUptimeMonitorFailures implements UptimeProvider for AlertService.
// Returns monitors that have had consecutive failures.
func (s *ProxyService) GetUptimeMonitorFailures(ctx context.Context) ([]*UptimeMonitorFailure, error) {
	monitors, err := s.repo.ListEnabledMonitors(ctx)
	if err != nil {
		return nil, err
	}

	var failures []*UptimeMonitorFailure
	s.monitorMu.Lock()
	for _, m := range monitors {
		fails := s.consecutiveFails[m.ID]
		if fails > 0 {
			failures = append(failures, &UptimeMonitorFailure{
				MonitorID:           m.ID,
				MonitorName:         m.Name,
				URL:                 m.URL,
				ConsecutiveFailures: fails,
			})
		}
	}
	s.monitorMu.Unlock()
	return failures, nil
}

// ---- Uptime monitoring background loop ----

// monitoringLoop runs a single goroutine that ticks every 15 seconds and
// dispatches individual checks for any monitor that is due.
func (s *ProxyService) monitoringLoop(ctx context.Context) {
	defer s.monitorWg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Prune old results daily.
	pruneTimer := time.NewTicker(24 * time.Hour)
	defer pruneTimer.Stop()

	slog.Info("proxy: uptime monitoring loop started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("proxy: uptime monitoring loop stopped")
			return
		case <-ticker.C:
			s.dispatchDueChecks(ctx)
		case <-pruneTimer.C:
			if n, err := s.repo.PruneOldResults(ctx, 30*24*time.Hour); err != nil {
				slog.Error("proxy: failed to prune uptime results", "error", err)
			} else if n > 0 {
				slog.Info("proxy: pruned old uptime results", "count", n)
			}
		}
	}
}

// dispatchDueChecks finds monitors that are due and runs their checks concurrently.
func (s *ProxyService) dispatchDueChecks(ctx context.Context) {
	monitors, err := s.repo.ListEnabledMonitors(ctx)
	if err != nil {
		slog.Error("proxy: failed to list monitors for dispatch", "error", err)
		return
	}

	now := time.Now()
	for _, m := range monitors {
		s.monitorMu.Lock()
		last := s.lastRun[m.ID]
		interval := time.Duration(m.IntervalSeconds) * time.Second
		due := now.Sub(last) >= interval
		s.monitorMu.Unlock()

		if !due {
			continue
		}

		// Capture loop variable.
		mon := m
		go func() {
			result := s.performCheck(ctx, mon)
			if err := s.repo.LogUptimeResult(ctx, result); err != nil {
				slog.Error("proxy: failed to log uptime result",
					"monitor_id", mon.ID, "error", err)
			}

			s.monitorMu.Lock()
			s.lastRun[mon.ID] = now
			if result.Success {
				s.consecutiveFails[mon.ID] = 0
			} else {
				s.consecutiveFails[mon.ID]++
				fails := s.consecutiveFails[mon.ID]
				s.monitorMu.Unlock()
				slog.Warn("proxy: uptime check failed",
					"monitor", mon.Name, "url", mon.URL,
					"consecutive_failures", fails,
					"error", result.Error)
				// Fire alert after 3 consecutive failures.
				if fails == 3 && s.alertSender != nil {
					s.alertSender.FireUptimeAlert(ctx, mon.ID, mon.Name, mon.URL, fails, result.Error)
				}
				return
			}
			s.monitorMu.Unlock()
		}()
	}
}

// performCheck executes a single HTTP health check against the monitor's URL.
func (s *ProxyService) performCheck(ctx context.Context, m *model.UptimeMonitor) *model.UptimeResult {
	result := &model.UptimeResult{
		ID:        uuid.New().String(),
		MonitorID: m.ID,
	}

	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	client := &http.Client{
		Timeout: timeout,
		// Follow redirects but limit hops.
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // intentional for internal monitoring
		},
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("invalid URL: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "lab-uptime/1.0")

	resp, err := client.Do(req)
	elapsed := time.Since(start)
	result.ResponseTimeMs = elapsed.Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Success = resp.StatusCode == m.ExpectedStatusCode
	if !result.Success {
		result.Error = fmt.Sprintf("expected status %d, got %d", m.ExpectedStatusCode, resp.StatusCode)
	}
	return result
}

// ---- CRUD operations ----

// CreateHost creates a new proxy host, persists it, and reloads routes.
func (s *ProxyService) CreateHost(ctx context.Context, req *labv1.CreateProxyHostRequest) (*labv1.ProxyHost, error) {
	host := &model.ProxyHost{
		ID:                   uuid.New().String(),
		Domain:               req.Domain,
		TargetURL:            req.TargetUrl,
		SSLMode:              protoSSLModeToModel(req.SslMode),
		BasicAuthEnabled:     req.BasicAuthEnabled,
		BasicAuthUser:        req.BasicAuthUser,
		CustomRequestHeaders: protoHeadersToMap(req.CustomRequestHeaders),
		CustomResponseHeaders: protoHeadersToMap(req.CustomResponseHeaders),
		WebsocketSupport:     req.WebsocketSupport,
		Enabled:              req.Enabled,
	}

	if req.BasicAuthPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.BasicAuthPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("proxy: failed to hash password: %w", err)
		}
		host.BasicAuthPasswordHash = string(hash)
	}

	if err := s.repo.Create(ctx, host); err != nil {
		return nil, fmt.Errorf("proxy: failed to create host: %w", err)
	}

	// For self-signed SSL, generate and persist a cert immediately.
	if host.SSLMode == model.ProxySSLModeSelfSigned {
		if err := s.ensureSelfSignedCert(ctx, host); err != nil {
			slog.Warn("proxy: failed to generate self-signed cert", "domain", host.Domain, "error", err)
		}
	}

	if err := s.reloadRoutes(ctx); err != nil {
		slog.Warn("proxy: route reload after create failed", "error", err)
	}

	// Auto-create an uptime monitor for the new proxy host.
	monitorURL := "http://" + host.Domain
	if host.SSLMode != model.ProxySSLModeNone {
		monitorURL = "https://" + host.Domain
	}
	monitorReq := &model.UptimeMonitorCreateRequest{
		Name:               host.Domain,
		URL:                monitorURL,
		ProxyHostID:        host.ID,
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
		Enabled:            host.Enabled,
	}
	if _, err := s.CreateMonitor(ctx, monitorReq); err != nil {
		slog.Warn("proxy: failed to auto-create uptime monitor", "domain", host.Domain, "error", err)
	}

	return s.modelToProto(host), nil
}

// GetHost returns a single proxy host by ID.
func (s *ProxyService) GetHost(ctx context.Context, id string) (*labv1.ProxyHost, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get host: %w", err)
	}
	if host == nil {
		return nil, fmt.Errorf("proxy: host not found: %s", id)
	}
	return s.modelToProto(host), nil
}

// ListHosts returns all proxy hosts.
func (s *ProxyService) ListHosts(ctx context.Context) ([]*labv1.ProxyHost, int32, error) {
	hosts, err := s.repo.List(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy: failed to list hosts: %w", err)
	}

	var out []*labv1.ProxyHost
	for _, h := range hosts {
		out = append(out, s.modelToProto(h))
	}
	return out, int32(len(out)), nil
}

// ListHostsByTargetIP returns proxy hosts that target a specific IP address.
// This is useful for showing associated proxy hosts on VM/Container detail pages.
func (s *ProxyService) ListHostsByTargetIP(ctx context.Context, ip string) ([]*labv1.ProxyHost, int32, error) {
	if ip == "" {
		return nil, 0, nil
	}
	hosts, err := s.repo.ListByTargetIP(ctx, ip)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy: failed to list hosts by target IP: %w", err)
	}

	var out []*labv1.ProxyHost
	for _, h := range hosts {
		out = append(out, s.modelToProto(h))
	}
	return out, int32(len(out)), nil
}

// UpdateHost updates an existing proxy host and reloads routes.
func (s *ProxyService) UpdateHost(ctx context.Context, req *labv1.UpdateProxyHostRequest) (*labv1.ProxyHost, error) {
	existing, err := s.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, fmt.Errorf("proxy: failed to get host: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("proxy: host not found: %s", req.Id)
	}

	existing.Domain = req.Domain
	existing.TargetURL = req.TargetUrl
	existing.SSLMode = protoSSLModeToModel(req.SslMode)
	existing.BasicAuthEnabled = req.BasicAuthEnabled
	existing.BasicAuthUser = req.BasicAuthUser
	existing.CustomRequestHeaders = protoHeadersToMap(req.CustomRequestHeaders)
	existing.CustomResponseHeaders = protoHeadersToMap(req.CustomResponseHeaders)
	existing.WebsocketSupport = req.WebsocketSupport
	existing.Enabled = req.Enabled

	if req.BasicAuthPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.BasicAuthPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("proxy: failed to hash password: %w", err)
		}
		existing.BasicAuthPasswordHash = string(hash)
	}

	if err := s.repo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("proxy: failed to update host: %w", err)
	}

	// Regenerate self-signed cert if mode changed to self_signed.
	if existing.SSLMode == model.ProxySSLModeSelfSigned {
		if err := s.ensureSelfSignedCert(ctx, existing); err != nil {
			slog.Warn("proxy: failed to regenerate self-signed cert", "domain", existing.Domain, "error", err)
		}
	}

	if err := s.reloadRoutes(ctx); err != nil {
		slog.Warn("proxy: route reload after update failed", "error", err)
	}

	return s.modelToProto(existing), nil
}

// DeleteHost removes a proxy host and reloads routes.
func (s *ProxyService) DeleteHost(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("proxy: failed to delete host: %w", err)
	}
	if err := s.reloadRoutes(ctx); err != nil {
		slog.Warn("proxy: route reload after delete failed", "error", err)
	}
	// Restart HTTPS server if no more TLS hosts exist.
	if !s.hasTLSHosts() && s.httpsSrv != nil {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpsSrv.Shutdown(shutCtx)
		s.httpsSrv = nil
	}
	return nil
}

// UploadCert stores a user-supplied TLS certificate for a custom-mode host.
func (s *ProxyService) UploadCert(ctx context.Context, req *labv1.UploadCertRequest) error {
	host, err := s.repo.GetByID(ctx, req.ProxyHostId)
	if err != nil || host == nil {
		return fmt.Errorf("proxy: host not found: %s", req.ProxyHostId)
	}

	// Validate the PEM pair.
	if _, err := tls.X509KeyPair([]byte(req.CertPem), []byte(req.KeyPem)); err != nil {
		return fmt.Errorf("proxy: invalid certificate/key pair: %w", err)
	}

	// Parse expiry from the cert.
	var expiresAt string
	block, _ := pem.Decode([]byte(req.CertPem))
	if block != nil {
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			expiresAt = cert.NotAfter.UTC().Format(time.RFC3339)
		}
	}

	cert := &model.ProxyCert{
		ID:          uuid.New().String(),
		ProxyHostID: req.ProxyHostId,
		CertPEM:     req.CertPem,
		KeyPEM:      req.KeyPem,
		ExpiresAt:   expiresAt,
	}
	if err := s.repo.SaveCert(ctx, cert); err != nil {
		return fmt.Errorf("proxy: failed to save cert: %w", err)
	}

	if err := s.reloadRoutes(ctx); err != nil {
		slog.Warn("proxy: route reload after cert upload failed", "error", err)
	}
	return nil
}

// GetStatus returns a live status snapshot for a proxy host.
func (s *ProxyService) GetStatus(ctx context.Context, id string) (*labv1.ProxyStatus, error) {
	host, err := s.repo.GetByID(ctx, id)
	if err != nil || host == nil {
		return nil, fmt.Errorf("proxy: host not found: %s", id)
	}

	s.mu.RLock()
	_, running := s.routes[host.Domain]
	s.mu.RUnlock()

	reachable := s.checkBackend(host.TargetURL)

	var certExpiry string
	if host.SSLMode != model.ProxySSLModeNone {
		if cert, err := s.repo.GetCert(ctx, id); err == nil && cert != nil {
			certExpiry = cert.ExpiresAt
		}
	}

	return &labv1.ProxyStatus{
		ProxyHostId:      id,
		IsRunning:        running && host.Enabled,
		BackendReachable: reachable,
		CertExpiry:       certExpiry,
		LastCheckAt:      time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ---- Internal helpers ----

// reloadRoutes rebuilds the in-memory route table from the database.
func (s *ProxyService) reloadRoutes(ctx context.Context) error {
	hosts, err := s.repo.List(ctx)
	if err != nil {
		return err
	}

	newRoutes := make(map[string]*proxyEntry, len(hosts))
	for _, h := range hosts {
		target, err := url.Parse(h.TargetURL)
		if err != nil {
			slog.Warn("proxy: skipping host with invalid target URL", "domain", h.Domain, "target", h.TargetURL, "error", err)
			continue
		}

		rp := httputil.NewSingleHostReverseProxy(target)
		host := h // capture

		// Inject custom request headers via Director.
		origDirector := rp.Director
		rp.Director = func(req *http.Request) {
			origDirector(req)
			for k, v := range host.CustomRequestHeaders {
				req.Header.Set(k, v)
			}
			// Preserve original Host header for upstream.
			req.Host = target.Host
		}

		// Inject custom response headers via ModifyResponse.
		if len(h.CustomResponseHeaders) > 0 {
			rp.ModifyResponse = func(resp *http.Response) error {
				for k, v := range host.CustomResponseHeaders {
					resp.Header.Set(k, v)
				}
				return nil
			}
		}

		newRoutes[h.Domain] = &proxyEntry{host: h, proxy: rp}
	}

	s.mu.Lock()
	s.routes = newRoutes
	s.mu.Unlock()

	slog.Debug("proxy: routes reloaded", "count", len(newRoutes))
	return nil
}

// buildHTTPHandler returns an http.Handler that routes by Host header.
func (s *ProxyService) buildHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := stripPort(r.Host)

		s.mu.RLock()
		entry, ok := s.routes[domain]
		s.mu.RUnlock()

		if !ok {
			writeJSONError(w, http.StatusNotFound, "no proxy route found for host")
			return
		}
		if !entry.host.Enabled {
			writeJSONError(w, http.StatusServiceUnavailable, "proxy host is disabled")
			return
		}
		if entry.host.SSLMode != model.ProxySSLModeNone {
			// Redirect plain-HTTP requests to HTTPS.
			target := "https://" + r.Host + r.RequestURI
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		if entry.host.BasicAuthEnabled {
			if !checkBasicAuth(w, r, entry.host) {
				return
			}
		}
		entry.proxy.ServeHTTP(w, r)
	})
}

// startHTTPS creates and starts the HTTPS server with SNI-based cert selection.
func (s *ProxyService) startHTTPS(ctx context.Context) error {
	tlsCfg := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return s.getCertificate(ctx, hello.ServerName)
		},
	}

	s.httpsSrv = &http.Server{
		Addr:      fmt.Sprintf(":%d", s.httpsPort),
		Handler:   s.buildHTTPSHandler(),
		TLSConfig: tlsCfg,
		ReadTimeout: 30 * time.Second,
		WriteTimeout: 0,
		IdleTimeout: 60 * time.Second,
	}

	go func() {
		slog.Info("proxy: HTTPS server listening", "port", s.httpsPort)
		// ListenAndServeTLS with empty cert/key files — TLS config handles everything.
		if err := s.httpsSrv.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("proxy: HTTPS server error", "error", err)
		}
	}()
	return nil
}

// buildHTTPSHandler returns an http.Handler for the HTTPS server.
func (s *ProxyService) buildHTTPSHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := stripPort(r.Host)

		s.mu.RLock()
		entry, ok := s.routes[domain]
		s.mu.RUnlock()

		if !ok {
			writeJSONError(w, http.StatusNotFound, "no proxy route found for host")
			return
		}
		if !entry.host.Enabled {
			writeJSONError(w, http.StatusServiceUnavailable, "proxy host is disabled")
			return
		}
		if entry.host.BasicAuthEnabled {
			if !checkBasicAuth(w, r, entry.host) {
				return
			}
		}
		entry.proxy.ServeHTTP(w, r)
	})
}

// getCertificate returns the TLS certificate for the given server name.
func (s *ProxyService) getCertificate(ctx context.Context, serverName string) (*tls.Certificate, error) {
	s.mu.RLock()
	entry, ok := s.routes[serverName]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("proxy: no route for %q", serverName)
	}

	switch entry.host.SSLMode {
	case model.ProxySSLModeSelfSigned, model.ProxySSLModeCustom:
		cert, err := s.repo.GetCert(ctx, entry.host.ID)
		if err != nil || cert == nil {
			if entry.host.SSLMode == model.ProxySSLModeSelfSigned {
				// Auto-generate on first use.
				if genErr := s.ensureSelfSignedCert(ctx, entry.host); genErr != nil {
					return nil, fmt.Errorf("proxy: cert generation failed for %q: %w", serverName, genErr)
				}
				cert, err = s.repo.GetCert(ctx, entry.host.ID)
				if err != nil || cert == nil {
					return nil, fmt.Errorf("proxy: cert unavailable for %q", serverName)
				}
			} else {
				return nil, fmt.Errorf("proxy: no cert uploaded for %q", serverName)
			}
		}
		tlsCert, err := tls.X509KeyPair([]byte(cert.CertPEM), []byte(cert.KeyPEM))
		if err != nil {
			return nil, fmt.Errorf("proxy: invalid cert for %q: %w", serverName, err)
		}
		return &tlsCert, nil

	case model.ProxySSLModeACME:
		return nil, fmt.Errorf("proxy: ACME mode is not yet implemented for %q", serverName)

	default:
		return nil, fmt.Errorf("proxy: host %q does not use TLS", serverName)
	}
}

// ensureSelfSignedCert generates and persists a self-signed TLS cert for the host.
func (s *ProxyService) ensureSelfSignedCert(ctx context.Context, host *model.ProxyHost) error {
	certPEM, keyPEM, expiresAt, err := generateSelfSignedCert(host.Domain)
	if err != nil {
		return err
	}
	cert := &model.ProxyCert{
		ID:          uuid.New().String(),
		ProxyHostID: host.ID,
		CertPEM:     certPEM,
		KeyPEM:      keyPEM,
		ExpiresAt:   expiresAt.UTC().Format(time.RFC3339),
	}
	return s.repo.SaveCert(ctx, cert)
}

// generateSelfSignedCert creates a self-signed RSA-2048 certificate for the given domain.
func generateSelfSignedCert(domain string) (certPEM, keyPEM string, expiresAt time.Time, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("proxy: failed to generate key: %w", err)
	}

	expiresAt = time.Now().Add(365 * 24 * time.Hour)

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("proxy: failed to generate serial: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: domain,
		},
		DNSNames:  []string{domain},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  expiresAt,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("proxy: failed to create certificate: %w", err)
	}

	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes}))
	keyPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	return certPEM, keyPEM, expiresAt, nil
}

// hasTLSHosts reports whether any enabled route requires TLS.
func (s *ProxyService) hasTLSHosts() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, e := range s.routes {
		if e.host.Enabled && e.host.SSLMode != model.ProxySSLModeNone {
			return true
		}
	}
	return false
}

// checkBackend does a quick TCP dial to see if the backend is reachable.
func (s *ProxyService) checkBackend(targetURL string) bool {
	u, err := url.Parse(targetURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		if u.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// checkBasicAuth validates the HTTP Basic auth credentials for a request.
// Returns true if authenticated; writes a 401 and returns false otherwise.
func checkBasicAuth(w http.ResponseWriter, r *http.Request, host *model.ProxyHost) bool {
	user, pass, ok := r.BasicAuth()
	if !ok || user != host.BasicAuthUser {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, host.Domain))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(host.BasicAuthPasswordHash), []byte(pass)); err != nil {
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm=%q`, host.Domain))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// stripPort removes the port from a host:port string.
func stripPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return host
}

// writeJSONError writes a simple JSON error response.
func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

// ---- Proto conversion helpers ----

func (s *ProxyService) modelToProto(h *model.ProxyHost) *labv1.ProxyHost {
	return &labv1.ProxyHost{
		Id:                    h.ID,
		Domain:                h.Domain,
		TargetUrl:             h.TargetURL,
		SslMode:               modelSSLModeToProto(h.SSLMode),
		BasicAuthEnabled:      h.BasicAuthEnabled,
		BasicAuthUser:         h.BasicAuthUser,
		CustomRequestHeaders:  h.CustomRequestHeaders,
		CustomResponseHeaders: h.CustomResponseHeaders,
		WebsocketSupport:      h.WebsocketSupport,
		Enabled:               h.Enabled,
		CreatedAt:             h.CreatedAt,
		UpdatedAt:             h.UpdatedAt,
	}
}

func protoSSLModeToModel(m labv1.ProxySSLMode) model.ProxySSLMode {
	switch m {
	case labv1.ProxySSLMode_PROXY_SSL_MODE_SELF_SIGNED:
		return model.ProxySSLModeSelfSigned
	case labv1.ProxySSLMode_PROXY_SSL_MODE_ACME:
		return model.ProxySSLModeACME
	case labv1.ProxySSLMode_PROXY_SSL_MODE_CUSTOM:
		return model.ProxySSLModeCustom
	default:
		return model.ProxySSLModeNone
	}
}

func modelSSLModeToProto(m model.ProxySSLMode) labv1.ProxySSLMode {
	switch m {
	case model.ProxySSLModeSelfSigned:
		return labv1.ProxySSLMode_PROXY_SSL_MODE_SELF_SIGNED
	case model.ProxySSLModeACME:
		return labv1.ProxySSLMode_PROXY_SSL_MODE_ACME
	case model.ProxySSLModeCustom:
		return labv1.ProxySSLMode_PROXY_SSL_MODE_CUSTOM
	default:
		return labv1.ProxySSLMode_PROXY_SSL_MODE_NONE
	}
}

func protoHeadersToMap(h map[string]string) map[string]string {
	if len(h) == 0 {
		return nil
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}
