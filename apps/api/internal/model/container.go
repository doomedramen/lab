package model

// ContainerStatus represents the operational status of a container
type ContainerStatus string

const (
	ContainerStatusRunning ContainerStatus = "running"
	ContainerStatusStopped ContainerStatus = "stopped"
	ContainerStatusFrozen  ContainerStatus = "frozen"
)

// Container represents an LXC container
type Container struct {
	ID           string          `json:"id"`
	CTID         int             `json:"ctid"`
	Name         string          `json:"name"`
	Node         string          `json:"node"`
	Status       ContainerStatus `json:"status"`
	CPU          CPUInfoPartial  `json:"cpu"`
	Memory       MemoryInfo      `json:"memory"`
	Disk         DiskInfo        `json:"disk"`
	Uptime       string          `json:"uptime"`
	OS           string          `json:"os"`
	IP           string          `json:"ip"`
	Tags         []string        `json:"tags"`
	Unprivileged bool            `json:"unprivileged"`
	Swap         SwapInfo        `json:"swap"`
	Description  string          `json:"description"`
	StartOnBoot  bool            `json:"startOnBoot"`
}

// ContainerCreateRequest represents the request body for creating a container
type ContainerCreateRequest struct {
	Name         string   `json:"name"`
	Node         string   `json:"node"`
	CPUCores     int      `json:"cpuCores"`
	Memory       float64  `json:"memory"`
	Disk         float64  `json:"disk"`
	OS           string   `json:"os"`
	Tags         []string `json:"tags"`
	Unprivileged bool     `json:"unprivileged"`
	Description  string   `json:"description"`
	StartOnBoot  bool     `json:"startOnBoot"`
}

// ContainerUpdateRequest represents the request body for updating a container
type ContainerUpdateRequest struct {
	Name        string   `json:"name,omitempty"`
	CPUCores    int      `json:"cpuCores,omitempty"`
	Memory      float64  `json:"memory,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Description string   `json:"description,omitempty"`
}
