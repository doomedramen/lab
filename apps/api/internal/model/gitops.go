package model

import "time"

// GitOpsKind represents the type of GitOps resource
type GitOpsKind string

const (
	GitOpsKindVirtualMachine  GitOpsKind = "VirtualMachine"
	GitOpsKindContainer       GitOpsKind = "Container"
	GitOpsKindNetwork         GitOpsKind = "Network"
	GitOpsKindStoragePool     GitOpsKind = "StoragePool"
	GitOpsKindDockerStack     GitOpsKind = "DockerStack"
	GitOpsKindGitOpsConfig    GitOpsKind = "GitOpsConfig"
)

// GitOpsStatus represents the reconciliation status
type GitOpsStatus string

const (
	GitOpsStatusHealthy   GitOpsStatus = "Healthy"
	GitOpsStatusOutOfSync GitOpsStatus = "OutOfSync"
	GitOpsStatusFailed    GitOpsStatus = "Failed"
	GitOpsStatusPending   GitOpsStatus = "Pending"
)

// GitOpsConfig represents a GitOps configuration (Git repository + settings)
type GitOpsConfig struct {
	ID                string        `json:"id"`
	Name              string        `json:"name"`
	Description       string        `json:"description"`
	GitURL            string        `json:"git_url"`
	GitBranch         string        `json:"git_branch"`
	GitPath           string        `json:"git_path"` // Path within repo to scan
	SyncInterval      time.Duration `json:"sync_interval"`
	LastSync          time.Time     `json:"last_sync"`
	LastSyncHash      string        `json:"last_sync_hash"` // Git commit hash
	Status            GitOpsStatus  `json:"status"`
	StatusMessage     string        `json:"status_message"`
	Enabled           bool          `json:"enabled"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	NextSync          time.Time     `json:"next_sync"`
	SyncRetries       int           `json:"sync_retries"`
	MaxSyncRetries    int           `json:"max_sync_retries"`
}

// GitOpsResource represents a parsed resource from Git manifest
type GitOpsResource struct {
	ID              string            `json:"id"`
	ConfigID        string            `json:"config_id"`
	Kind            GitOpsKind        `json:"kind"`
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	ManifestPath    string            `json:"manifest_path"` // Path in Git repo
	ManifestHash    string            `json:"manifest_hash"` // Hash of manifest content
	Spec            map[string]any    `json:"spec"`          // Parsed spec
	Status          GitOpsStatus      `json:"status"`
	StatusMessage   string            `json:"status_message"`
	LastApplied     time.Time         `json:"last_applied"`
	LastDiff        string            `json:"last_diff"` // Human-readable diff
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// GitOpsSyncLog represents a sync operation log entry
type GitOpsSyncLog struct {
	ID          string       `json:"id"`
	ConfigID    string       `json:"config_id"`
	StartTime   time.Time    `json:"start_time"`
	EndTime     time.Time    `json:"end_time"`
	Duration    time.Duration `json:"duration"`
	Status      GitOpsStatus `json:"status"`
	Message     string       `json:"message"`
	CommitHash  string       `json:"commit_hash"`
	ResourcesScanned int     `json:"resources_scanned"`
	ResourcesCreated int     `json:"resources_created"`
	ResourcesUpdated int     `json:"resources_updated"`
	ResourcesDeleted int     `json:"resources_deleted"`
	ResourcesFailed  int     `json:"resources_failed"`
}

// GitOpsManifest represents a parsed YAML manifest
type GitOpsManifest struct {
	APIVersion string         `yaml:"apiVersion"`
	Kind       string         `yaml:"kind"`
	Metadata   ManifestMetadata `yaml:"metadata"`
	Spec       map[string]any `yaml:"spec"`
}

// ManifestMetadata represents metadata section of a manifest
type ManifestMetadata struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// GitOpsDiff represents the difference between desired and actual state
type GitOpsDiff struct {
	ResourceID string       `json:"resource_id"`
	Kind       GitOpsKind   `json:"kind"`
	Name       string       `json:"name"`
	Action     string       `json:"action"` // create, update, delete, unchanged
	Changes    []FieldChange `json:"changes,omitempty"`
}

// FieldChange represents a single field change
type FieldChange struct {
	Field      string `json:"field"`
	OldValue   any    `json:"old_value,omitempty"`
	NewValue   any    `json:"new_value,omitempty"`
}
