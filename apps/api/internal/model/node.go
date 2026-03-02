package model

import "time"

// NodeStatus represents the operational status of a node
type NodeStatus string

const (
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

// HostNode represents a physical or virtual host in the cluster
type HostNode struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Status      NodeStatus `json:"status"`
	IP          string     `json:"ip"`
	CPU         CPUInfo    `json:"cpu"`
	Memory      MemoryInfo `json:"memory"`
	Disk        DiskInfo   `json:"disk"`
	Uptime      string     `json:"uptime"`
	Kernel      string     `json:"kernel"`
	Version     string     `json:"version"`
	VMs         int        `json:"vms"`
	Containers  int        `json:"containers"`
	CPUModel    string     `json:"cpuModel"`
	LoadAvg     LoadAvg    `json:"loadAvg"`
	NetworkIn   float64    `json:"networkIn"`
	NetworkOut  float64    `json:"networkOut"`
	Arch        string     `json:"arch"` // e.g. "x86_64", "aarch64"
}

// HostShellToken is a one-time token granting WebSocket access to a host shell
type HostShellToken struct {
	NodeID    string
	CreatedAt time.Time
}
