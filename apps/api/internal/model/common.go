package model

// CPUInfo represents CPU usage information with total percentage
type CPUInfo struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
	Cores int     `json:"cores"`
}

// CPUInfoPartial represents CPU info for VMs/Containers (no total percentage)
type CPUInfoPartial struct {
	Used    float64 `json:"used"`
	Sockets int     `json:"sockets"`
	Cores   int     `json:"cores"`
}

// MemoryInfo represents memory usage in GB
type MemoryInfo struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

// DiskInfo represents disk usage in GB (or TB for large values)
type DiskInfo struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

// SwapInfo represents swap usage in GB
type SwapInfo struct {
	Used  float64 `json:"used"`
	Total float64 `json:"total"`
}

// LoadAvg represents 1, 5, and 15 minute load averages
type LoadAvg [3]float64
