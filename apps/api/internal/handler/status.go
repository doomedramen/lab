package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	"github.com/doomedramen/lab/apps/api/pkg/response"
)

// StatusHandler handles status/metrics API requests
// Provides generic endpoints for dashboards and monitoring tools
type StatusHandler struct {
	metricRepo    *sqlite.MetricRepository
	eventRepo     *sqlite.EventRepository
	libvirtClient libvirtx.LibvirtClient
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(metricRepo *sqlite.MetricRepository, eventRepo *sqlite.EventRepository, libvirtClient libvirtx.LibvirtClient) *StatusHandler {
	return &StatusHandler{
		metricRepo:    metricRepo,
		eventRepo:     eventRepo,
		libvirtClient: libvirtClient,
	}
}

// SystemStatusResponse represents system metrics response
type SystemStatusResponse struct {
	CPUUsage      float64 `json:"cpu_usage"`
	MemoryUsage   float64 `json:"memory_usage"`
	DiskUsage     float64 `json:"disk_usage"`
	UptimeSeconds int64   `json:"uptime_seconds"`
	UpdatedAt     string  `json:"updated_at"`
}

// GetSystemStatus handles GET /api/status/system
// Returns basic system metrics (CPU, memory, disk, uptime)
// Public endpoint - no authentication required
func (h *StatusHandler) GetSystemStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := h.getSystemStatus(ctx)

	response.JSON(w, http.StatusOK, resp)
}

// VMStatusResponse represents VM status response
type VMStatusResponse struct {
	Items   []VMItem `json:"items"`
	Total   int      `json:"total"`
	Running int      `json:"running"`
	Stopped int      `json:"stopped"`
}

// VMItem represents a single VM in the status response
type VMItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// GetVMStatus handles GET /api/status/vms
// Returns list of VMs with their current state
// Protected endpoint - requires API key authentication
func (h *StatusHandler) GetVMStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp, err := h.getVMStatus(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ContainerStatusResponse represents container status response
type ContainerStatusResponse struct {
	Items   []ContainerItem `json:"items"`
	Total   int             `json:"total"`
	Running int             `json:"running"`
	Stopped int             `json:"stopped"`
}

// ContainerItem represents a single container in the status response
type ContainerItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// GetContainerStatus handles GET /api/status/containers
// Returns list of containers with their current state
// Protected endpoint - requires API key authentication
func (h *StatusHandler) GetContainerStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp, err := h.getContainerStatus(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// StorageStatusResponse represents storage pool status response
type StorageStatusResponse struct {
	Items []StorageItem `json:"items"`
	Total int           `json:"total"`
}

// StorageItem represents a single storage pool in the status response
type StorageItem struct {
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Capacity  uint64  `json:"capacity"`
	Allocated uint64  `json:"allocated"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

// GetStorageStatus handles GET /api/status/storage
// Returns storage pool information
// Public endpoint - no authentication required
func (h *StatusHandler) GetStorageStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := h.getStorageStatus(ctx)

	response.JSON(w, http.StatusOK, resp)
}

// NetworkStatusResponse represents network status response
type NetworkStatusResponse struct {
	Items []NetworkItem `json:"items"`
	Total int           `json:"total"`
}

// NetworkItem represents a single network in the status response
type NetworkItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetNetworkStatus handles GET /api/status/networks
// Returns virtual network information
// Public endpoint - no authentication required
func (h *StatusHandler) GetNetworkStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := h.getNetworkStatus(ctx)

	response.JSON(w, http.StatusOK, resp)
}

// ServicesStatusResponse represents services health status response
type ServicesStatusResponse struct {
	Items []ServiceItem `json:"items"`
	Total int           `json:"total"`
	OK    int           `json:"ok"`
}

// ServiceItem represents a single service in the status response
type ServiceItem struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GetServicesStatus handles GET /api/status/services
// Returns core service health status
// Public endpoint - no authentication required
func (h *StatusHandler) GetServicesStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	resp := h.getServicesStatus(ctx)

	response.JSON(w, http.StatusOK, resp)
}

// AlertsResponse represents alerts response
type AlertsResponse struct {
	Items []AlertItem `json:"items"`
	Total int         `json:"total"`
}

// AlertItem represents a single alert in the status response
type AlertItem struct {
	ID        string `json:"id"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
}

// GetAlerts handles GET /api/status/alerts
// Returns recent alerts/events
// Protected endpoint - requires API key authentication
func (h *StatusHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse limit from query param (default 10)
	limit := 10
	if r.URL.Query().Get("limit") != "" {
		if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
			limit = l
		}
	}

	resp, err := h.getAlerts(ctx, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// getSystemStatus retrieves system metrics
func (h *StatusHandler) getSystemStatus(ctx context.Context) *SystemStatusResponse {
	resp := &SystemStatusResponse{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Get metrics from repository if available
	if h.metricRepo != nil {
		now := time.Now()
		startTime := now.Add(-1 * time.Hour).Unix()

		metrics, err := h.metricRepo.Query(ctx, model.MetricQuery{
			ResourceType: "host",
			StartTime:    startTime,
			EndTime:      now.Unix(),
			Aggregate:    "avg",
		})
		if err == nil {
			for _, m := range metrics {
				switch m.MetricType {
				case "cpu_usage":
					resp.CPUUsage = m.Value
				case "memory_usage":
					resp.MemoryUsage = m.Value
				case "disk_usage":
					resp.DiskUsage = m.Value
				}
			}
		}
	}

	// Get uptime from libvirt if available
	if h.libvirtClient != nil {
		if info, err := h.libvirtClient.GetNodeInfo(); err == nil {
			resp.UptimeSeconds = int64(info.Uptime)
		}
	}

	return resp
}

// getVMStatus retrieves VM status summary
func (h *StatusHandler) getVMStatus(ctx context.Context) (*VMStatusResponse, error) {
	resp := &VMStatusResponse{
		Items: []VMItem{},
	}

	if h.libvirtClient == nil {
		return resp, nil
	}

	doms, err := h.libvirtClient.ListDomains()
	if err != nil {
		return resp, err
	}

	for _, dom := range doms {
		state, err := h.libvirtClient.GetDomainState(dom)
		if err != nil {
			continue
		}

		item := VMItem{
			ID:    dom,
			Name:  dom,
			State: state.String(),
		}
		resp.Items = append(resp.Items, item)

		switch state {
		case libvirtx.DomainRunning:
			resp.Running++
		case libvirtx.DomainShutoff:
			resp.Stopped++
		}
	}

	resp.Total = len(resp.Items)
	return resp, nil
}

// getContainerStatus retrieves container status summary
func (h *StatusHandler) getContainerStatus(ctx context.Context) (*ContainerStatusResponse, error) {
	resp := &ContainerStatusResponse{
		Items: []ContainerItem{},
	}

	// TODO: Implement when container support is added
	// This is a placeholder for future LXC/container integration

	resp.Total = len(resp.Items)
	return resp, nil
}

// getStorageStatus retrieves storage pool status
func (h *StatusHandler) getStorageStatus(ctx context.Context) *StorageStatusResponse {
	resp := &StorageStatusResponse{
		Items: []StorageItem{},
	}

	if h.libvirtClient == nil {
		return resp
	}

	pools, err := h.libvirtClient.ListStoragePools()
	if err != nil {
		return resp
	}

	for _, pool := range pools {
		info, err := h.libvirtClient.GetStoragePoolInfo(pool)
		if err != nil {
			continue
		}

		item := StorageItem{
			Name:      pool,
			State:     info.State.String(),
			Capacity:  info.Capacity,
			Allocated: info.Allocated,
			Available: info.Available,
			Usage:     calculateUsagePercent(info.Allocated, info.Capacity),
		}
		resp.Items = append(resp.Items, item)
	}

	resp.Total = len(resp.Items)
	return resp
}

// getNetworkStatus retrieves network status
func (h *StatusHandler) getNetworkStatus(ctx context.Context) *NetworkStatusResponse {
	resp := &NetworkStatusResponse{
		Items: []NetworkItem{},
	}

	if h.libvirtClient == nil {
		return resp
	}

	networks, err := h.libvirtClient.ListNetworks()
	if err != nil {
		return resp
	}

	for _, net := range networks {
		active, _ := h.libvirtClient.IsNetworkActive(net)

		status := "inactive"
		if active {
			status = "active"
		}

		item := NetworkItem{
			Name:   net,
			Status: status,
		}
		resp.Items = append(resp.Items, item)
	}

	resp.Total = len(resp.Items)
	return resp
}

// getServicesStatus retrieves core service health status
func (h *StatusHandler) getServicesStatus(ctx context.Context) *ServicesStatusResponse {
	resp := &ServicesStatusResponse{
		Items: []ServiceItem{},
	}

	// Check libvirt status
	libvirtItem := ServiceItem{Name: "libvirt"}
	if h.libvirtClient == nil {
		libvirtItem.Status = "disabled"
		libvirtItem.Message = "not configured"
	} else if h.libvirtClient.IsConnected() {
		libvirtItem.Status = "ok"
		resp.OK++
	} else {
		libvirtItem.Status = "error"
		libvirtItem.Message = "not connected"
	}
	resp.Items = append(resp.Items, libvirtItem)

	// API is always ok if we're serving the request
	apiItem := ServiceItem{
		Name:   "api",
		Status: "ok",
	}
	resp.Items = append(resp.Items, apiItem)
	resp.OK++

	resp.Total = len(resp.Items)
	return resp
}

// getAlerts retrieves recent alerts
func (h *StatusHandler) getAlerts(ctx context.Context, limit int) (*AlertsResponse, error) {
	resp := &AlertsResponse{
		Items: []AlertItem{},
	}

	if h.eventRepo == nil {
		return resp, nil
	}

	events, err := h.eventRepo.List(ctx, model.EventQuery{
		Limit: limit,
	})
	if err != nil {
		return resp, err
	}

	for _, e := range events {
		item := AlertItem{
			ID:        e.ID,
			Level:     e.Severity,
			Message:   e.Message,
			Timestamp: e.CreatedAt,
			Source:    e.Source,
		}
		resp.Items = append(resp.Items, item)
	}

	resp.Total = len(resp.Items)
	return resp, nil
}

// calculateUsagePercent calculates usage percentage
func calculateUsagePercent(allocated, capacity uint64) float64 {
	if capacity == 0 {
		return 0
	}
	return float64(allocated) / float64(capacity) * 100
}

// RegisterStatusRoutes registers status API routes
func RegisterStatusRoutes(r chi.Router, h *StatusHandler, apiKeyAuth func(http.HandlerFunc) http.HandlerFunc) {
	// Public endpoints (no auth required)
	r.Get("/status/system", h.GetSystemStatus)
	r.Get("/status/storage", h.GetStorageStatus)
	r.Get("/status/networks", h.GetNetworkStatus)
	r.Get("/status/services", h.GetServicesStatus)

	// Protected endpoints (API key required)
	r.Get("/status/vms", apiKeyAuth(h.GetVMStatus))
	r.Get("/status/containers", apiKeyAuth(h.GetContainerStatus))
	r.Get("/status/alerts", apiKeyAuth(h.GetAlerts))
}
