package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/response"
)

// EventsHandler handles events-related HTTP requests
type EventsHandler struct {
	eventRepo *sqlite.EventRepository
}

// NewEventsHandler creates a new events handler
func NewEventsHandler(eventRepo *sqlite.EventRepository) *EventsHandler {
	return &EventsHandler{eventRepo: eventRepo}
}

// EventQueryParams holds parsed query parameters
type EventQueryParams struct {
	NodeID     string
	ResourceID string
	EventType  string
	Severity   string
	StartTime  int64
	EndTime    int64
	Limit      int
	Offset     int
}

// parseEventQueryParams parses query parameters from the request
func parseEventQueryParams(r *http.Request) EventQueryParams {
	q := r.URL.Query()
	params := EventQueryParams{
		NodeID:    q.Get("node_id"),
		ResourceID: q.Get("resource_id"),
		EventType: q.Get("event_type"),
		Severity:  q.Get("severity"),
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

	// Default to last 7 days if no time range specified
	if params.StartTime == 0 {
		params.StartTime = time.Now().Add(-7 * 24 * time.Hour).Unix()
	}
	if params.EndTime == 0 {
		params.EndTime = time.Now().Unix()
	}

	// Parse limit and offset
	if limit := q.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}
	if params.Limit == 0 {
		params.Limit = 100
	}

	if offset := q.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			params.Offset = o
		}
	}

	return params
}

// GetEvents handles GET /api/events
func (h *EventsHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	params := parseEventQueryParams(r)

	q := model.EventQuery{
		NodeID:    params.NodeID,
		EventType: model.EventType(params.EventType),
		Severity:  model.EventSeverity(params.Severity),
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
		Limit:     params.Limit,
		Offset:    params.Offset,
	}

	if params.ResourceID != "" {
		q.ResourceID = &params.ResourceID
	}

	ctx := r.Context()
	events, err := h.eventRepo.Query(ctx, q)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

// GetEventTypes handles GET /api/events/types
func (h *EventsHandler) GetEventTypes(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	types, err := h.eventRepo.GetEventTypes(ctx)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"types": types,
	})
}

// GetSeverityCounts handles GET /api/events/severity-counts
func (h *EventsHandler) GetSeverityCounts(w http.ResponseWriter, r *http.Request) {
	params := parseEventQueryParams(r)

	ctx := r.Context()
	counts, err := h.eventRepo.GetSeverityCounts(ctx, params.StartTime, params.EndTime)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"counts": counts,
	})
}

// CreateEvent handles POST /api/events
func (h *EventsHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeID     string          `json:"node_id"`
		ResourceID *string         `json:"resource_id,omitempty"`
		EventType  string          `json:"event_type"`
		Severity   string          `json:"severity"`
		Message    string          `json:"message"`
		Metadata   json.RawMessage `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.NodeID == "" {
		response.Error(w, http.StatusBadRequest, "node_id is required")
		return
	}
	if req.EventType == "" {
		response.Error(w, http.StatusBadRequest, "event_type is required")
		return
	}
	if req.Message == "" {
		response.Error(w, http.StatusBadRequest, "message is required")
		return
	}

	// Default severity
	if req.Severity == "" {
		req.Severity = "info"
	}

	event := &model.EventCreate{
		NodeID:     req.NodeID,
		ResourceID: req.ResourceID,
		EventType:  model.EventType(req.EventType),
		Severity:   model.EventSeverity(req.Severity),
		Message:    req.Message,
		Metadata:   req.Metadata,
	}

	ctx := r.Context()
	id, err := h.eventRepo.Log(ctx, event)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, map[string]interface{}{
		"id":      id,
		"success": true,
	})
}

// RegisterEventsRoutes registers events routes on the router
func RegisterEventsRoutes(r chi.Router, h *EventsHandler) {
	r.Get("/events", h.GetEvents)
	r.Get("/events/types", h.GetEventTypes)
	r.Get("/events/severity-counts", h.GetSeverityCounts)
	r.Post("/events", h.CreateEvent)
}
