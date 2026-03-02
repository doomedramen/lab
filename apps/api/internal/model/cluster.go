package model

// EntityCounts represents total and running counts for an entity type
type EntityCounts struct {
	Total   int `json:"total"`
	Running int `json:"running,omitempty"`
}

// CPUAggregate represents aggregated CPU information
type CPUAggregate struct {
	Cores    int     `json:"cores"`
	AvgUsage float64 `json:"avgUsage"`
}

// ClusterSummary provides an overview of the entire cluster
type ClusterSummary struct {
	Nodes      EntityCounts  `json:"nodes"`
	VMs        EntityCounts  `json:"vms"`
	Containers EntityCounts  `json:"containers"`
	Stacks     EntityCounts  `json:"stacks"`
	CPU        CPUAggregate  `json:"cpu"`
	Memory     MemoryInfo    `json:"memory"`
	Disk       DiskInfo      `json:"disk"`
}

// TimeSeriesDataPoint represents a single data point in a time series
type TimeSeriesDataPoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// ClusterMetrics contains time series data for cluster metrics
type ClusterMetrics struct {
	CPUUsage    []TimeSeriesDataPoint `json:"cpuUsage"`
	MemoryUsage []TimeSeriesDataPoint `json:"memoryUsage"`
	NetworkIn   []TimeSeriesDataPoint `json:"networkIn"`
	NetworkOut  []TimeSeriesDataPoint `json:"networkOut"`
}
