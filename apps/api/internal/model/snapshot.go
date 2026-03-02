package model

// SnapshotStatus represents the operational status of a snapshot
type SnapshotStatus string

const (
	SnapshotStatusCreating SnapshotStatus = "creating"
	SnapshotStatusReady    SnapshotStatus = "ready"
	SnapshotStatusDeleting SnapshotStatus = "deleting"
	SnapshotStatusError    SnapshotStatus = "error"
)

// VMState represents the VM state when snapshot was taken
type VMState string

const (
	VMStateRunning VMState = "running"
	VMStateStopped VMState = "stopped"
)

// Snapshot represents a VM snapshot with its metadata
type Snapshot struct {
	ID           string         `json:"id"`
	VMID         int            `json:"vmid"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	CreatedAt    string         `json:"created_at"`
	ParentID     string         `json:"parent_id,omitempty"`
	SizeBytes    int64          `json:"size_bytes"`
	Status       SnapshotStatus `json:"status"`
	VMState      VMState        `json:"vm_state"`
	HasChildren  bool           `json:"has_children"`
	SnapshotPath string         `json:"snapshot_path,omitempty"`
}

// SnapshotTree represents a hierarchical view of snapshots
type SnapshotTree struct {
	Snapshot *Snapshot      `json:"snapshot"`
	Children []*SnapshotTree `json:"children,omitempty"`
}

// SnapshotCreateRequest represents the request to create a snapshot
type SnapshotCreateRequest struct {
	VMID           int    `json:"vmid"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Live           bool   `json:"live,omitempty"`
	IncludeMemory  bool   `json:"include_memory,omitempty"`
}

// SnapshotDeleteRequest represents the request to delete a snapshot
type SnapshotDeleteRequest struct {
	VMID           int    `json:"vmid"`
	SnapshotID     string `json:"snapshot_id"`
	IncludeChildren bool  `json:"include_children,omitempty"`
}

// SnapshotRestoreRequest represents the request to restore a snapshot
type SnapshotRestoreRequest struct {
	VMID       int    `json:"vmid"`
	SnapshotID string `json:"snapshot_id"`
	StartAfter bool   `json:"start_after,omitempty"`
}

// SnapshotInfoRequest represents the request to get snapshot info
type SnapshotInfoRequest struct {
	VMID       int    `json:"vmid"`
	SnapshotID string `json:"snapshot_id"`
}
