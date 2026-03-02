package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/response"
)

// MetricsHandler handles metrics-related HTTP requests
type MetricsHandler struct {
	metricRepo *sqlite.MetricRepository
	eventRepo  *sqlite.EventRepository
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(metricRepo *sqlite.MetricRepository, eventRepo *sqlite.EventRepository) *MetricsHandler {
	return &MetricsHandler{
		metricRepo: metricRepo,
		eventRepo:  eventRepo,
	}
}

// QueryParams holds parsed query parameters
type QueryParams struct {
	NodeID       string
	ResourceType string
	ResourceID   string
	StartTime    int64
	EndTime      int64
	Aggregate    string
	GroupBy      string
	Limit        int
	HostOnly     bool
}

// parseQueryParams parses query parameters from the request
func parseQueryParams(r *http.Request) QueryParams {
	q := r.URL.Query()
	params := QueryParams{
		NodeID:       q.Get("node_id"),
		ResourceType: q.Get("resource_type"),
		ResourceID:   q.Get("resource_id"),
		Aggregate:    q.Get("aggregate"),
		GroupBy:      q.Get("group_by"),
		HostOnly:     q.Get("host_only") == "true",
	}

	// Parse time range
	if start := q.Get("start"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			params.StartTime = t.Unix()
		} else if ts, err := strconv.ParseInt(start, 10, 64); err == nil {
			params.StartTime = ts
		}
	}

	if end := q.Get("end"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			params.EndTime = t.Unix()
		} else if ts, err := strconv.ParseInt(end, 10, 64); err == nil {
			params.EndTime = ts
		}
	}

	// Default to last 24 hours if no time range specified
	if params.StartTime == 0 {
		params.StartTime = time.Now().Add(-24 * time.Hour).Unix()
	}
	if params.EndTime == 0 {
		params.EndTime = time.Now().Unix()
	}

	// Parse limit
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}

	return params
}

// GetMetrics handles GET /api/metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	params := parseQueryParams(r)

	q := model.MetricQuery{
		NodeID:       params.NodeID,
		ResourceType: params.ResourceType,
		StartTime:    params.StartTime,
		EndTime:      params.EndTime,
		Aggregate:    params.Aggregate,
		HostOnly:     params.HostOnly,
	}

	if params.ResourceID != "" {
		q.ResourceID = &params.ResourceID
	}

	ctx := r.Context()
	metrics, err := h.metricRepo.Query(ctx, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"metrics": metrics,
		"count":   len(metrics),
	})
}

// GetLatest handles GET /api/metrics/latest
func (h *MetricsHandler) GetLatest(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	resourceType := r.URL.Query().Get("resource_type")
	resourceID := r.URL.Query().Get("resource_id")

	if nodeID == "" {
		response.Error(w, http.StatusBadRequest, "node_id is required")
		return
	}

	var resID *string
	if resourceID != "" {
		resID = &resourceID
	}

	ctx := r.Context()
	metric, err := h.metricRepo.GetLatest(ctx, nodeID, resourceType, resID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if metric == nil {
		response.Error(w, http.StatusNotFound, "metric not found")
		return
	}

	response.JSON(w, http.StatusOK, metric)
}

// GetSummary handles GET /api/metrics/summary
func (h *MetricsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")

	if nodeID == "" {
		response.Error(w, http.StatusBadRequest, "node_id is required")
		return
	}

	ctx := r.Context()
	values, err := h.metricRepo.GetLatestValues(ctx, nodeID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"node_id": nodeID,
		"metrics": values,
	})
}

// GetTimeRange handles GET /api/metrics/time-range
func (h *MetricsHandler) GetTimeRange(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	min, max, err := h.metricRepo.GetTimeRange(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"min": min.Format(time.RFC3339),
		"max": max.Format(time.RFC3339),
	})
}

// GetMetricTypes handles GET /api/metrics/types
func (h *MetricsHandler) GetMetricTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	types, err := h.metricRepo.GetMetricTypes(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"types": types,
	})
}

// RegisterMetricsRoutes registers metrics routes on the router
func RegisterMetricsRoutes(r chi.Router, h *MetricsHandler) {
	r.Get("/metrics", h.GetMetrics)
	r.Get("/metrics/latest", h.GetLatest)
	r.Get("/metrics/summary", h.GetSummary)
	r.Get("/metrics/time-range", h.GetTimeRange)
	r.Get("/metrics/types", h.GetMetricTypes)
}
