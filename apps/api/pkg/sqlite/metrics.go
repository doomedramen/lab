package sqlite

import (
	"context"
	"database/sql"
)

// Metric represents a single metric data point
type Metric struct {
	ID           int64   `json:"id"`
	Timestamp    int64   `json:"ts"`
	NodeID       string  `json:"node_id"`
	ResourceType string  `json:"resource_type"`
	ResourceID   *string `json:"resource_id,omitempty"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
}

// MetricQuery holds query parameters for filtering metrics
type MetricQuery struct {
	NodeID       string
	ResourceType string
	ResourceID   *string
	StartTime    int64
	EndTime      int64
	Aggregate    string // 'avg', 'max', 'min', 'sum'
	GroupBy      string // 'hour', 'day'
}

// MetricRepository provides methods for storing and querying metrics
type MetricRepository struct {
	db *DB
}

// NewMetricRepository creates a new metric repository
func NewMetricRepository(db *DB) *MetricRepository {
	return &MetricRepository{db: db}
}

// Insert saves a single metric
func (r *MetricRepository) Insert(ctx context.Context, metric *Metric) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO metrics (ts, node_id, resource_type, resource_id, value, unit)
		VALUES (?, ?, ?, ?, ?, ?)
	`, metric.Timestamp, metric.NodeID, metric.ResourceType, metric.ResourceID, metric.Value, metric.Unit)
	return err
}

// InsertBatch saves multiple metrics in a single transaction
func (r *MetricRepository) InsertBatch(ctx context.Context, metrics []*Metric) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO metrics (ts, node_id, resource_type, resource_id, value, unit)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		_, err := stmt.ExecContext(ctx, m.Timestamp, m.NodeID, m.ResourceType, m.ResourceID, m.Value, m.Unit)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Query retrieves metrics based on query parameters
func (r *MetricRepository) Query(ctx context.Context, q MetricQuery) ([]*Metric, error) {
	query := `SELECT id, ts, node_id, resource_type, resource_id, value, unit FROM metrics WHERE 1=1`
	args := []interface{}{}

	if q.NodeID != "" {
		query += ` AND node_id = ?`
		args = append(args, q.NodeID)
	}

	if q.ResourceType != "" {
		query += ` AND resource_type = ?`
		args = append(args, q.ResourceType)
	}

	if q.ResourceID != nil {
		if *q.ResourceID == "" {
			query += ` AND resource_id IS NULL`
		} else {
			query += ` AND resource_id = ?`
			args = append(args, *q.ResourceID)
		}
	}

	if q.StartTime > 0 {
		query += ` AND ts >= ?`
		args = append(args, q.StartTime)
	}

	if q.EndTime > 0 {
		query += ` AND ts <= ?`
		args = append(args, q.EndTime)
	}

	query += ` ORDER BY ts ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []*Metric
	for rows.Next() {
		var m Metric
		var resourceID sql.NullString
		if err := rows.Scan(&m.ID, &m.Timestamp, &m.NodeID, &m.ResourceType, &resourceID, &m.Value, &m.Unit); err != nil {
			return nil, err
		}
		if resourceID.Valid {
			m.ResourceID = &resourceID.String
		}
		metrics = append(metrics, &m)
	}

	return metrics, rows.Err()
}

// QueryAggregated retrieves aggregated metrics (e.g., hourly averages)
func (r *MetricRepository) QueryAggregated(ctx context.Context, q MetricQuery) ([]map[string]interface{}, error) {
	// Determine the time grouping based on the time range
	timeRange := q.EndTime - q.StartTime
	var groupExpr string

	// Use strftime with ISO 8601 format ('T' separator) so JavaScript new Date() parses correctly.
	if timeRange > 7*86400 { // More than 7 days - group by day
		groupExpr = "strftime('%Y-%m-%dT%H:%M:%S', ts, 'unixepoch', 'localtime', 'start of day')"
	} else if timeRange > 24*3600 { // More than 1 day - group by hour
		groupExpr = "strftime('%Y-%m-%dT%H:%M:%S', ts, 'unixepoch', 'localtime', 'start of hour')"
	} else { // Less than 1 day - group by 5 minutes
		groupExpr = "strftime('%Y-%m-%dT%H:%M:%S', ts - (ts % 300), 'unixepoch', 'localtime')"
	}

	// Default aggregate function
	aggFunc := "AVG"
	switch q.Aggregate {
	case "max":
		aggFunc = "MAX"
	case "min":
		aggFunc = "MIN"
	case "sum":
		aggFunc = "SUM"
	}

	// When no node filter: aggregate across all nodes (one row per time bucket).
	// When node filter set: preserve per-node granularity.
	nodeSelect := "'' AS node_id"
	groupBy := "time_bucket, resource_type, resource_id"
	if q.NodeID != "" {
		nodeSelect = "node_id"
		groupBy = "time_bucket, node_id, resource_type, resource_id"
	}

	query := `
		SELECT
			` + groupExpr + ` as time_bucket,
			` + aggFunc + `(value) as value,
			` + nodeSelect + `,
			resource_type,
			resource_id,
			unit,
			COUNT(*) as count
		FROM metrics WHERE 1=1
	`
	args := []interface{}{}

	if q.NodeID != "" {
		query += ` AND node_id = ?`
		args = append(args, q.NodeID)
	}

	if q.ResourceType != "" {
		query += ` AND resource_type = ?`
		args = append(args, q.ResourceType)
	}

	if q.ResourceID != nil {
		if *q.ResourceID == "" {
			query += ` AND resource_id IS NULL`
		} else {
			query += ` AND resource_id = ?`
			args = append(args, *q.ResourceID)
		}
	}

	if q.StartTime > 0 {
		query += ` AND ts >= ?`
		args = append(args, q.StartTime)
	}

	if q.EndTime > 0 {
		query += ` AND ts <= ?`
		args = append(args, q.EndTime)
	}

	query += ` GROUP BY ` + groupBy + ` ORDER BY time_bucket ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var timeBucket string
		var value float64
		var nodeID, resourceType, unit string
		var resourceID sql.NullString
		var count int

		if err := rows.Scan(&timeBucket, &value, &nodeID, &resourceType, &resourceID, &unit, &count); err != nil {
			return nil, err
		}

		row := map[string]interface{}{
			"time":          timeBucket,
			"value":         value,
			"node_id":       nodeID,
			"resource_type": resourceType,
			"unit":          unit,
			"count":         count,
		}
		if resourceID.Valid {
			row["resource_id"] = resourceID.String
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// DeleteOld removes metrics older than the specified number of days
func (r *MetricRepository) DeleteOld(ctx context.Context, days int) (int64, error) {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM metrics WHERE ts < (strftime('%s', 'now') - (? * 86400))
	`, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetLatest retrieves the most recent metric for a specific resource
func (r *MetricRepository) GetLatest(ctx context.Context, nodeID, resourceType string, resourceID *string) (*Metric, error) {
	query := `SELECT id, ts, node_id, resource_type, resource_id, value, unit FROM metrics 
		WHERE node_id = ? AND resource_type = ?`
	args := []interface{}{nodeID, resourceType}

	if resourceID != nil {
		if *resourceID == "" {
			query += ` AND resource_id IS NULL`
		} else {
			query += ` AND resource_id = ?`
			args = append(args, *resourceID)
		}
	}

	query += ` ORDER BY ts DESC LIMIT 1`

	var m Metric
	var resourceIDVal sql.NullString
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&m.ID, &m.Timestamp, &m.NodeID, &m.ResourceType, &resourceIDVal, &m.Value, &m.Unit,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if resourceIDVal.Valid {
		m.ResourceID = &resourceIDVal.String
	}
	return &m, nil
}

// GetLatestValues retrieves the most recent metrics for all resources on a node
func (r *MetricRepository) GetLatestValues(ctx context.Context, nodeID string) (map[string]map[string]float64, error) {
	query := `
		SELECT resource_type, resource_id, value, ts
		FROM metrics m1
		WHERE node_id = ? AND ts = (
			SELECT MAX(ts) FROM metrics m2 
			WHERE m2.node_id = m1.node_id 
			AND m2.resource_type = m1.resource_type 
			AND (m2.resource_id = m1.resource_id OR (m2.resource_id IS NULL AND m1.resource_id IS NULL))
		)
	`

	rows, err := r.db.QueryContext(ctx, query, nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Structure: map[resourceType]map[resourceID]value
	result := make(map[string]map[string]float64)

	for rows.Next() {
		var resourceType string
		var resourceID sql.NullString
		var value float64

		if err := rows.Scan(&resourceType, &resourceID, &value); err != nil {
			return nil, err
		}

		if _, ok := result[resourceType]; !ok {
			result[resourceType] = make(map[string]float64)
		}

		id := ""
		if resourceID.Valid {
			id = resourceID.String
		}
		result[resourceType][id] = value
	}

	return result, rows.Err()
}

// GetTimeRange returns the time range of stored metrics
func (r *MetricRepository) GetTimeRange(ctx context.Context) (min, max int64, err error) {
	err = r.db.QueryRowContext(ctx, `SELECT MIN(ts), MAX(ts) FROM metrics`).Scan(&min, &max)
	return
}

// GetMetricTypes returns all unique metric types
func (r *MetricRepository) GetMetricTypes(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT resource_type FROM metrics ORDER BY resource_type`)
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
