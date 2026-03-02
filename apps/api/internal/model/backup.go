package model

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeSnapshot    BackupType = "snapshot"
)

// BackupStatus represents the status of a backup operation
type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
	BackupStatusDeleting  BackupStatus = "deleting"
)

// VerificationStatus represents the verification status of a backup
type VerificationStatus string

const (
	VerificationStatusPending   VerificationStatus = "pending"
	VerificationStatusVerified  VerificationStatus = "verified"
	VerificationStatusFailed    VerificationStatus = "failed"
	VerificationStatusNotRun    VerificationStatus = "not_run"
)

// Backup represents a VM backup
type Backup struct {
	ID                string            `json:"id"`
	VMID              int               `json:"vmid"`
	VMName            string            `json:"vm_name"`
	Name              string            `json:"name"`
	Type              BackupType        `json:"type"`
	Status            BackupStatus      `json:"status"`
	SizeBytes         int64             `json:"size_bytes"`
	StoragePool       string            `json:"storage_pool"`
	BackupPath        string            `json:"backup_path"`
	CreatedAt         string            `json:"created_at"`
	CompletedAt       string            `json:"completed_at,omitempty"`
	ExpiresAt         string            `json:"expires_at,omitempty"`
	ErrorMessage      string            `json:"error_message,omitempty"`
	RetentionDays     int               `json:"retention_days"`
	Encrypted         bool              `json:"encrypted"`
	VerifiedAt        string            `json:"verified_at,omitempty"`
	VerificationStatus VerificationStatus `json:"verification_status"`
	VerificationError string            `json:"verification_error,omitempty"`
}

// BackupSchedule represents a scheduled backup job
type BackupSchedule struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	EntityType    string     `json:"entity_type"`  // "vm" or "container"
	EntityID      int        `json:"entity_id"`
	StoragePool   string     `json:"storage_pool"`
	Schedule      string     `json:"schedule"`  // cron expression
	BackupType    BackupType `json:"backup_type"`
	RetentionDays int        `json:"retention_days"`
	RetainCount   int        `json:"retain_count"` // Keep last N backups (0 = use days-based)
	Encrypt       bool       `json:"encrypt"`      // Encrypt backups by default
	Enabled       bool       `json:"enabled"`
	CreatedAt     string     `json:"created_at"`
	UpdatedAt     string     `json:"updated_at"`
	LastRunAt     string     `json:"last_run_at,omitempty"`
	NextRunAt     string     `json:"next_run_at,omitempty"`
	TotalBackups  int        `json:"total_backups"`
}

// BackupCreateRequest represents a request to create a backup
type BackupCreateRequest struct {
	VMID                int        `json:"vmid"`
	Name                string     `json:"name,omitempty"`
	Type                BackupType `json:"type"`
	StoragePool         string     `json:"storage_pool"`
	Compress            bool       `json:"compress"`
	RetentionDays       int        `json:"retention_days"`
	Encrypt             bool       `json:"encrypt"`
	EncryptionPassphrase string    `json:"encryption_passphrase,omitempty"`
}

// BackupRestoreRequest represents a request to restore from a backup
type BackupRestoreRequest struct {
	BackupID             string `json:"backup_id"`
	TargetVMID           int    `json:"target_vmid"`  // 0 = restore to original
	TargetNode           string `json:"target_node"`
	StartAfter           bool   `json:"start_after"`
	DecryptionPassphrase string `json:"decryption_passphrase,omitempty"` // Required if backup is encrypted
}

// BackupScheduleCreateRequest represents a request to create a backup schedule
type BackupScheduleCreateRequest struct {
	Name          string     `json:"name"`
	EntityType    string     `json:"entity_type"`
	EntityID      int        `json:"entity_id"`
	StoragePool   string     `json:"storage_pool"`
	Schedule      string     `json:"schedule"`  // cron expression
	BackupType    BackupType `json:"backup_type"`
	RetentionDays int        `json:"retention_days"`
	RetainCount   int        `json:"retain_count"` // Keep last N backups (0 = use days-based)
	Encrypt       bool       `json:"encrypt"`
	Enabled       bool       `json:"enabled"`
}

// BackupScheduleUpdateRequest represents a request to update a backup schedule
type BackupScheduleUpdateRequest struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	Schedule      string     `json:"schedule,omitempty"`
	BackupType    BackupType `json:"backup_type"`
	RetentionDays int        `json:"retention_days"`
	RetainCount   int        `json:"retain_count"`
	Encrypt       bool       `json:"encrypt"`
	Enabled       *bool      `json:"enabled"`  // pointer to allow false
}
