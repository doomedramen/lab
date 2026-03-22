package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// Event represents a system event or log entry
type Event struct {
	ID         int64           `json:"id"`
	Timestamp  int64           `json:"ts"`
	NodeID     string          `json:"node_id"`
	ResourceID *string         `json:"resource_id,omitempty"`
	EventType  string          `json:"event_type"`
	Severity   string          `json:"severity"`
	Message    string          `json:"message"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// EventQuery holds query parameters for filtering events
type EventQuery struct {
	NodeID      string
	ResourceID  *string
	EventType   string
	Severity    string
	StartTime   int64
	EndTime     int64
	Limit       int
	Offset      int
}

// EventRepository provides methods for storing and querying events
type EventRepository struct {
	db *DB
}

// NewEventRepository creates a new event repository
func NewEventRepository(db *DB) *EventRepository {
	return &EventRepository{db: db}
}

// Insert saves a single event
func (r *EventRepository) Insert(ctx context.Context, event *Event) (int64, error) {
	var metadataJSON *string
	if len(event.Metadata) > 0 {
		s := string(event.Metadata)
		metadataJSON = &s
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO events (ts, node_id, resource_id, event_type, severity, message, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, event.Timestamp, event.NodeID, event.ResourceID, event.EventType, event.Severity, event.Message, metadataJSON)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// InsertBatch saves multiple events in a single transaction
func (r *EventRepository) InsertBatch(ctx context.Context, events []*Event) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO events (ts, node_id, resource_id, event_type, severity, message, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range events {
		var metadataJSON *string
		if len(e.Metadata) > 0 {
			s := string(e.Metadata)
			metadataJSON = &s
		}
		_, err := stmt.ExecContext(ctx, e.Timestamp, e.NodeID, e.ResourceID, e.EventType, e.Severity, e.Message, metadataJSON)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query retrieves events based on query parameters
func (r *EventRepository) Query(ctx context.Context, q EventQuery) ([]*Event, error) {
	query := `SELECT id, ts, node_id, resource_id, event_type, severity, message, metadata FROM events WHERE 1=1`
	args := []interface{}{}

	if q.NodeID != "" {
		query += ` AND node_id = ?`
		args = append(args, q.NodeID)
	}

	if q.ResourceID != nil {
		if *q.ResourceID == "" {
			query += ` AND resource_id IS NULL`
		} else {
			query += ` AND resource_id = ?`
			args = append(args, *q.ResourceID)
		}
	}

	if q.EventType != "" {
		query += ` AND event_type = ?`
		args = append(args, q.EventType)
	}

	if q.Severity != "" {
		query += ` AND severity = ?`
		args = append(args, q.Severity)
	}

	if q.StartTime > 0 {
		query += ` AND ts >= ?`
		args = append(args, q.StartTime)
	}

	if q.EndTime > 0 {
		query += ` AND ts <= ?`
		args = append(args, q.EndTime)
	}

	query += ` ORDER BY ts DESC`

	// Apply limit and offset
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, limit)

	if q.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, q.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		var resourceID, metadataJSON sql.NullString
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.NodeID, &resourceID, &e.EventType, &e.Severity, &e.Message, &metadataJSON); err != nil {
			return nil, err
		}
		if resourceID.Valid {
			e.ResourceID = &resourceID.String
		}
		if metadataJSON.Valid {
			e.Metadata = json.RawMessage(metadataJSON.String)
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}

// DeleteOld removes events older than the specified number of days
func (r *EventRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM events WHERE ts < (strftime('%s', 'now') - (? * 86400))
	`, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetEventTypes returns all unique event types
func (r *EventRepository) GetEventTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT event_type FROM events ORDER BY event_type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		types = append(types, t)
	}
	return types, rows.Err()
}

// GetSeverityCounts returns counts of events by severity
func (r *EventRepository) GetSeverityCounts(ctx context.Context, startTime, endTime int64) (map[string]int, error) {
	query := `SELECT severity, COUNT(*) FROM events WHERE 1=1`
	args := []interface{}{}

	if startTime > 0 {
		query += ` AND ts >= ?`
		args = append(args, startTime)
	}
	if endTime > 0 {
		query += ` AND ts <= ?`
		args = append(args, endTime)
	}

	query += ` GROUP BY severity`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, err
		}
		counts[severity] = count
	}

	return counts, rows.Err()
}

// Log creates a helper for easily logging events
func (r *EventRepository) Log(ctx context.Context, nodeID, eventType, severity, message string, metadata interface{}) error {
	event := &Event{
		Timestamp:  time.Now().Unix(),
		NodeID:     nodeID,
		EventType:  eventType,
		Severity:   severity,
		Message:    message,
	}

	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return err
		}
		event.Metadata = data
	}

	_, err := r.Insert(ctx, event)
	return err
}
