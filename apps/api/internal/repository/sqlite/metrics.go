package sqlite

import (
	"context"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// MetricRepository handles metric storage and retrieval
type MetricRepository struct {
	repo *sqlitePkg.MetricRepository
}

// NewMetricRepository creates a new metric repository
func NewMetricRepository(db *sqlitePkg.DB) *MetricRepository {
	return &MetricRepository{repo: sqlitePkg.NewMetricRepository(db)}
}

// Record saves a metric data point
func (r *MetricRepository) Record(ctx context.Context, nodeID, resourceType string, value float64, unit string, resourceID *string) error {
	metric := &sqlitePkg.Metric{
		Timestamp:    time.Now().Unix(),
		NodeID:       nodeID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Value:        value,
		Unit:         unit,
	}
	return r.repo.Insert(ctx, metric)
}

// RecordBatch saves multiple metrics
func (r *MetricRepository) RecordBatch(ctx context.Context, metrics []*sqlitePkg.Metric) error {
	return r.repo.InsertBatch(ctx, metrics)
}

// Query retrieves metrics based on query parameters
func (r *MetricRepository) Query(ctx context.Context, q model.MetricQuery) ([]model.MetricPoint, error) {
	sqliteQuery := sqlitePkg.MetricQuery{
		NodeID:       q.NodeID,
		ResourceType: q.ResourceType,
		ResourceID:   q.ResourceID,
		StartTime:    q.StartTime,
		EndTime:      q.EndTime,
		Aggregate:    q.Aggregate,
	}
	if q.HostOnly {
		empty := ""
		sqliteQuery.ResourceID = &empty
	}

	var points []model.MetricPoint

	if q.Aggregate != "" || q.GroupBy != "" {
		// Use aggregated query
		results, err := r.repo.QueryAggregated(ctx, sqliteQuery)
		if err != nil {
			return nil, err
		}
		for _, row := range results {
			points = append(points, model.MetricPoint{
				Time:  row["time"].(string),
				Value: row["value"].(float64),
			})
		}
	} else {
		// Use raw query
		metrics, err := r.repo.Query(ctx, sqliteQuery)
		if err != nil {
			return nil, err
		}
		for _, m := range metrics {
			points = append(points, model.MetricPoint{
				Time:  time.Unix(m.Timestamp, 0).Format(time.RFC3339),
				Value: m.Value,
			})
		}
	}

	return points, nil
}

// GetLatest retrieves the most recent metric for a resource
func (r *MetricRepository) GetLatest(ctx context.Context, nodeID, resourceType string, resourceID *string) (*model.Metric, error) {
	metric, err := r.repo.GetLatest(ctx, nodeID, resourceType, resourceID)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		return nil, nil
	}

	return &model.Metric{
		ID:           metric.ID,
		Timestamp:    metric.Timestamp,
		NodeID:       metric.NodeID,
		ResourceType: metric.ResourceType,
		ResourceID:   metric.ResourceID,
		Value:        metric.Value,
		Unit:         metric.Unit,
	}, nil
}

// GetLatestValues retrieves the most recent metrics for all resources on a node
func (r *MetricRepository) GetLatestValues(ctx context.Context, nodeID string) (map[string]map[string]float64, error) {
	return r.repo.GetLatestValues(ctx, nodeID)
}

// DeleteOld removes metrics older than the specified number of days
func (r *MetricRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	return r.repo.DeleteOld(ctx, days)
}

// GetTimeRange returns the time range of stored metrics
func (r *MetricRepository) GetTimeRange(ctx context.Context) (min, max time.Time, err error) {
	minTs, maxTs, err := r.repo.GetTimeRange(ctx)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return time.Unix(minTs, 0), time.Unix(maxTs, 0), nil
}

// GetMetricTypes returns all unique metric types
func (r *MetricRepository) GetMetricTypes(ctx context.Context) ([]string, error) {
	return r.repo.GetMetricTypes(ctx)
}

// EventRepository handles event storage and retrieval
type EventRepository struct {
	repo *sqlitePkg.EventRepository
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *sqlitePkg.DB) *EventRepository {
	return &EventRepository{repo: sqlitePkg.NewEventRepository(db)}
}

// Log records a new event
func (r *EventRepository) Log(ctx context.Context, event *model.EventCreate) (int64, error) {
	sqliteEvent := &sqlitePkg.Event{
		Timestamp:  time.Now().Unix(),
		NodeID:     event.NodeID,
		ResourceID: event.ResourceID,
		EventType:  string(event.EventType),
		Severity:   string(event.Severity),
		Message:    event.Message,
		Metadata:   event.Metadata,
	}
	return r.repo.Insert(ctx, sqliteEvent)
}

// Query retrieves events based on query parameters
func (r *EventRepository) Query(ctx context.Context, q model.EventQuery) ([]model.Event, error) {
	sqliteQuery := sqlitePkg.EventQuery{
		NodeID:     q.NodeID,
		ResourceID: q.ResourceID,
		EventType:  string(q.EventType),
		Severity:   string(q.Severity),
		StartTime:  q.StartTime,
		EndTime:    q.EndTime,
		Limit:      q.Limit,
		Offset:     q.Offset,
	}

	events, err := r.repo.Query(ctx, sqliteQuery)
	if err != nil {
		return nil, err
	}

	var result []model.Event
	for _, e := range events {
		result = append(result, model.Event{
			ID:         e.ID,
			Timestamp:  e.Timestamp,
			NodeID:     e.NodeID,
			ResourceID: e.ResourceID,
			EventType:  model.EventType(e.EventType),
			Severity:   model.EventSeverity(e.Severity),
			Message:    e.Message,
			Metadata:   e.Metadata,
		})
	}

	return result, nil
}

// DeleteOld removes events older than the specified number of days
func (r *EventRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	return r.repo.DeleteOld(ctx, days)
}

// LogHelper provides a convenient way to log events
func (r *EventRepository) LogHelper(ctx context.Context, nodeID string, eventType model.EventType, severity model.EventSeverity, message string, metadata interface{}) error {
	return r.repo.Log(ctx, nodeID, string(eventType), string(severity), message, metadata)
}

// GetEventTypes returns all unique event types
func (r *EventRepository) GetEventTypes(ctx context.Context) ([]string, error) {
	return r.repo.GetEventTypes(ctx)
}

// GetSeverityCounts returns counts of events by severity
func (r *EventRepository) GetSeverityCounts(ctx context.Context, startTime, endTime int64) (map[string]int, error) {
	return r.repo.GetSeverityCounts(ctx, startTime, endTime)
}
