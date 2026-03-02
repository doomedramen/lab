package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	"github.com/doomedramen/lab/apps/api/pkg/response"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// HealthCheck returns a minimal liveness probe (no dependency checks).
// Used by load balancers that expect a very fast response.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// CheckResult represents the result of a single dependency check.
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HealthStatus holds the results of a deep health check.
type HealthStatus struct {
	Status    string                    `json:"status"`
	Version   string                    `json:"version"`
	Timestamp string                    `json:"timestamp"`
	Checks    map[string]CheckResult    `json:"checks"`
}

// HealthHandler performs a deep health check of critical dependencies.
// It checks libvirt connectivity and SQLite accessibility in parallel.
type HealthHandler struct {
	libvirtClient *libvirtx.Client
	db            *sqlitePkg.DB
	version       string
}

// NewHealthHandler creates a health handler with the given optional dependencies.
func NewHealthHandler(libvirtClient *libvirtx.Client, db *sqlitePkg.DB, version string) *HealthHandler {
	return &HealthHandler{libvirtClient: libvirtClient, db: db, version: version}
}

// ServeHTTP performs all health checks and returns a JSON summary.
// Responds 200 if all checks pass, 503 if any fail.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]CheckResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Libvirt check
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := h.checkLibvirt()
		mu.Lock()
		checks["libvirt"] = result
		mu.Unlock()
	}()

	// SQLite check
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := h.checkSQLite()
		mu.Lock()
		checks["sqlite"] = result
		mu.Unlock()
	}()

	wg.Wait()

	// Determine overall status
	overall := "ok"
	for _, check := range checks {
		if check.Status != "ok" {
			overall = "degraded"
			break
		}
	}

	status := HealthStatus{
		Status:    overall,
		Version:   h.version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}

	httpStatus := http.StatusOK
	if overall != "ok" {
		httpStatus = http.StatusServiceUnavailable
	}

	response.JSON(w, httpStatus, status)
}

func (h *HealthHandler) checkLibvirt() CheckResult {
	if h.libvirtClient == nil {
		return CheckResult{Status: "disabled", Message: "libvirt not configured"}
	}
	if !h.libvirtClient.IsConnected() {
		return CheckResult{Status: "error", Message: "not connected to libvirt"}
	}
	// Attempt a lightweight call to verify the connection is alive.
	_, err := h.libvirtClient.GetVersion()
	if err != nil {
		return CheckResult{Status: "error", Message: "libvirt ping failed: " + err.Error()}
	}
	return CheckResult{Status: "ok"}
}

func (h *HealthHandler) checkSQLite() CheckResult {
	if h.db == nil {
		return CheckResult{Status: "disabled", Message: "SQLite not configured"}
	}
	if err := h.db.DB.Ping(); err != nil {
		return CheckResult{Status: "error", Message: "SQLite ping failed: " + err.Error()}
	}
	return CheckResult{Status: "ok"}
}
