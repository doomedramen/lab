package model

// MetricPoint represents a single time-series metric data point
type MetricPoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// Metric represents a metric data point stored in SQLite
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
	NodeID       string  `json:"node_id,omitempty"`
	ResourceType string  `json:"resource_type,omitempty"`
	ResourceID   *string `json:"resource_id,omitempty"`
	StartTime    int64   `json:"start_time,omitempty"`
	EndTime      int64   `json:"end_time,omitempty"`
	Aggregate    string  `json:"aggregate,omitempty"` // 'avg', 'max', 'min', 'sum'
	GroupBy      string  `json:"group_by,omitempty"`  // 'hour', 'day'
	HostOnly     bool    `json:"host_only,omitempty"` // filter for host-level metrics (resource_id IS NULL)
}

// MetricSummary represents aggregated metric statistics
type MetricSummary struct {
	NodeID       string  `json:"node_id"`
	ResourceType string  `json:"resource_type"`
	ResourceID   *string `json:"resource_id,omitempty"`
	CurrentValue float64 `json:"current_value"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Avg          float64 `json:"avg"`
	Unit         string  `json:"unit"`
}
