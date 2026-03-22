package model

import "time"

// StackStatus represents the overall status of a Docker Compose stack
type StackStatus string

const (
	StackStatusRunning          StackStatus = "running"
	StackStatusPartiallyRunning StackStatus = "partially_running"
	StackStatusStopped          StackStatus = "stopped"
)

// DockerContainer represents a Docker container within a compose stack
type DockerContainer struct {
	ServiceName   string
	ContainerName string
	ContainerID   string
	Image         string
	Status        string   // human-readable e.g. "Up 2 hours"
	State         string   // "running", "exited", "created", etc.
	Ports         []string // ["0.0.0.0:8080->80/tcp"]
}

// DockerStack represents a Docker Compose stack managed on disk
type DockerStack struct {
	ID         string
	Name       string
	Compose    string
	Env        string
	Status     StackStatus
	Containers []DockerContainer
	CreatedAt  time.Time
}

// StackCreateRequest represents the request body for creating a new stack
type StackCreateRequest struct {
	Name    string
	Compose string
	Env     string
}

// StackUpdateRequest represents the request body for updating compose/env files
type StackUpdateRequest struct {
	Compose string
	Env     string
}

// ContainerToken is a one-time token granting WebSocket PTY access to a container
type ContainerToken struct {
	StackID       string
	ContainerName string
	CreatedAt     time.Time
}

// LogsToken is a one-time token granting WebSocket access to stack logs
type LogsToken struct {
	StackID   string
	CreatedAt time.Time
}
