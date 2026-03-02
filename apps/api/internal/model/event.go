package model

import "encoding/json"

// EventSeverity represents the severity level of an event
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "info"
	EventSeverityWarning  EventSeverity = "warning"
	EventSeverityError    EventSeverity = "error"
	EventSeverityCritical EventSeverity = "critical"
)

// EventType represents the type of system event
type EventType string

const (
	// VM events
	EventVMStart    EventType = "vm_start"
	EventVMStop     EventType = "vm_stop"
	EventVMPause    EventType = "vm_pause"
	EventVMResume   EventType = "vm_resume"
	EventVMReboot   EventType = "vm_reboot"
	EventVMShutdown EventType = "vm_shutdown"
	EventVMCreate   EventType = "vm_create"
	EventVMDelete   EventType = "vm_delete"

	// Container events
	EventContainerStart   EventType = "container_start"
	EventContainerStop    EventType = "container_stop"
	EventContainerCreate  EventType = "container_create"
	EventContainerDelete  EventType = "container_delete"

	// Node events
	EventNodeOnline      EventType = "node_online"
	EventNodeOffline     EventType = "node_offline"
	EventNodeMaintenance EventType = "node_maintenance"

	// System events
	EventAlert     EventType = "alert"
	EventError     EventType = "error"
	EventInfo      EventType = "info"
	EventUserLogin EventType = "user_login"
)

// Event represents a system event or log entry
type Event struct {
	ID         int64           `json:"id"`
	Timestamp  int64           `json:"ts"`
	NodeID     string          `json:"node_id"`
	ResourceID *string         `json:"resource_id,omitempty"`
	EventType  EventType       `json:"event_type"`
	Severity   EventSeverity   `json:"severity"`
	Message    string          `json:"message"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// EventQuery holds query parameters for filtering events
type EventQuery struct {
	NodeID     string        `json:"node_id,omitempty"`
	ResourceID *string       `json:"resource_id,omitempty"`
	EventType  EventType     `json:"event_type,omitempty"`
	Severity   EventSeverity `json:"severity,omitempty"`
	StartTime  int64         `json:"start_time,omitempty"`
	EndTime    int64         `json:"end_time,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
}

// EventCreate holds data for creating a new event
type EventCreate struct {
	NodeID     string          `json:"node_id"`
	ResourceID *string         `json:"resource_id,omitempty"`
	EventType  EventType       `json:"event_type"`
	Severity   EventSeverity   `json:"severity"`
	Message    string          `json:"message"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}
