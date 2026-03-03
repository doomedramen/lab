# Go API Style Guide

This document establishes naming conventions and coding standards for the Lab API codebase.

## Package Naming

### Directory Structure

```
apps/api/
├── cmd/
│   └── server/          # Main application entry point
├── internal/            # Private packages (unexported types)
│   ├── config/          # Configuration loading
│   ├── connectsvc/      # Connect RPC service implementations
│   ├── handler/         # HTTP/Connect handlers
│   ├── middleware/      # HTTP/Connect middleware
│   ├── model/           # Domain models
│   ├── repository/      # Data access layer
│   │   ├── auth/        # Auth-specific repositories
│   │   ├── docker/      # Docker-specific repositories
│   │   ├── libvirt/     # Libvirt-specific repositories
│   │   └── sqlite/      # SQLite-specific repositories
│   ├── router/          # HTTP router setup
│   ├── service/         # Business logic layer
│   └── uiserver/        # Embedded UI server
├── pkg/                 # Public packages (exported types)
│   ├── auth/            # Authentication utilities
│   ├── libvirtx/        # Libvirt extensions
│   ├── osinfo/          # OS information registry
│   ├── response/        # HTTP response helpers
│   ├── sqlite/          # SQLite utilities
│   ├── sysinfo/         # System information
│   └── tus/             # Tus upload protocol
└── gen/                 # Generated code (proto)
    └── lab/v1/          # Generated protobuf code
```

### Naming Rules

- Use `snake_case` for package directories: `pkg/sysinfo`, `internal/connectsvc`
- Package names should be short and descriptive
- Avoid generic names like `utils`, `common`, `helpers`
- Group by functionality, not by type

## Constructor Naming

### Services

```go
// Pattern: New<Service>Service
func NewAuthService(...) *AuthService
func NewVMService(...) *VMService
func NewBackupService(...) *BackupService
func NewSnapshotService(...) *SnapshotService
func NewStorageService(...) *StorageService
func NewNetworkService(...) *NetworkService
func NewAlertService(...) *AlertService
func NewProxyService(...) *ProxyService
func NewClusterService(...) *ClusterService
func NewNodeService(...) *NodeService
func NewContainerService(...) *ContainerService
func NewStackService(...) *StackService
func NewISOService(...) *ISOService
func NewTaskService(...) *TaskService
func NewFirewallService(...) *FirewallService
```

### Repositories

```go
// Pattern: New<Repo>Repository
func NewUserRepository(...) *UserRepository
func NewVMRepository(...) *VMRepository
func NewMetricRepository(...) *MetricRepository
func NewEventRepository(...) *EventRepository
func NewBackupRepository(...) *BackupRepository
func NewSnapshotRepository(...) *SnapshotRepository
func NewTaskRepository(...) *TaskRepository
func NewNetworkRepository(...) *NetworkRepository
func NewStoragePoolRepository(...) *StoragePoolRepository
func NewAlertRepository(...) *AlertRepository
func NewProxyRepository(...) *ProxyRepository
```

### Handlers

```go
// Pattern: New<Handler>Handler
func NewHealthHandler(...) *HealthHandler
func NewMetricsHandler(...) *MetricsHandler
func NewEventsHandler(...) *EventsHandler
func NewAuthServiceServer(...) *AuthServiceServer
func NewVmServiceServer(...) *VmServiceServer
func NewBackupServiceServer(...) *BackupServiceServer
func NewSnapshotServiceServer(...) *SnapshotServiceServer
func NewStorageServiceServer(...) *StorageServiceServer
func NewNetworkServiceServer(...) *NetworkServiceServer
func NewFirewallServiceServer(...) *FirewallServiceServer
func NewTaskServiceServer(...) *TaskServiceServer
func NewProxyServiceServer(...) *ProxyServiceServer
```

### Middleware

```go
// Pattern: New<Middleware> or New<Middleware>Interceptor
func NewAuthInterceptor(...) *AuthInterceptor
func NewRateLimiter(...) *RateLimiter
func NewLoggingMiddleware(...) *LoggingMiddleware
func NewCORSMiddleware(...) *CORSMiddleware
```

### Utilities

```go
// Pattern: New<Type>
func NewJWT(...) *JWT
func NewPassword(...) *Password
func NewMFA(...) *MFA
func NewRegistry(...) *Registry
func NewClient(...) *Client
func NewCollector(...) *Collector
```

## Variable Naming

### Services

```go
// Use <type>Svc or full name <type>Service
authSvc := NewAuthService(...)
vmSvc := NewVMService(...)
backupSvc := NewBackupService(...)

// Or use full name for clarity
authService := NewAuthService(...)
vmService := NewVMService(...)
```

### Repositories

```go
// Use <type>Repo
userRepo := NewUserRepository(...)
vmRepo := NewVMRepository(...)
metricRepo := NewMetricRepository(...)
taskRepo := NewTaskRepository(...)
```

### Handlers

```go
// Use <type>Handler
healthHandler := NewHealthHandler(...)
metricsHandler := NewMetricsHandler(...)
authHandler := NewAuthServiceServer(...)
```

### Context and Errors

```go
// Always use ctx for context
func (s *VMService) CreateVM(ctx context.Context, req *CreateVMRequest) error

// Always use err for errors
if err != nil {
    return err
}

// Use specific names for multiple errors
if parseErr != nil {
    return parseErr
}
if validateErr != nil {
    return validateErr
}
```

### Request/Response

```go
// Use req for request
func (h *Handler) CreateVM(ctx context.Context, req *connect.Request[labv1.CreateVMRequest]) error

// Use resp for response
resp := connect.NewResponse(&labv1.CreateVMResponse{})

// Use msg for request message
vmID := req.Msg.Vmid
```

## Error Naming

### Error Variables

```go
// Pattern: Err<Resource><Condition>
var (
    ErrVMNotFound       = errors.New("VM not found")
    ErrVMAlreadyRunning = errors.New("VM is already running")
    ErrVMAlreadyStopped = errors.New("VM is already stopped")
    ErrAuthFailed       = errors.New("authentication failed")
    ErrInvalidToken     = errors.New("invalid token")
)
```

### Error Types

```go
// Pattern: <Condition>Error
type ValidationError struct {
    Field   string
    Message string
}

type NotFoundError struct {
    ResourceType string
    ResourceID   string
}

type ConflictError struct {
    Message string
}
```

### Error Methods

```go
func (e *ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *NotFoundError) Error() string {
    return fmt.Sprintf("%s not found: %s", e.ResourceType, e.ResourceID)
}
```

## Function and Method Naming

### CRUD Operations

```go
// Standard CRUD naming
func (r *Repository) Create(ctx context.Context, entity *Entity) error
func (r *Repository) GetByID(ctx context.Context, id string) (*Entity, error)
func (r *Repository) List(ctx context.Context, filter Filter) ([]*Entity, error)
func (r *Repository) Update(ctx context.Context, entity *Entity) error
func (r *Repository) Delete(ctx context.Context, id string) error
```

### Boolean Returns

```go
// Use Is/Has/Can/Should prefixes for boolean returns
func (vm *VM) IsRunning() bool
func (user *User) HasPermission(perm string) bool
func (svc *Service) CanAccess(ctx context.Context, resource string) bool
func (rule *AlertRule) ShouldFire(ctx AlertContext) bool
```

### Async Operations

```go
// Use Start/Begin for async operations
func (c *Collector) Start(ctx context.Context)
func (s *AlertService) Start()
func (b *BackupService) StartBackup(ctx context.Context, vmID int) (taskID string, err error)
```

## Testing

### Test File Naming

```go
// Pattern: <file>_test.go
// vm.go → vm_test.go
// auth.go → auth_test.go
```

### Test Function Naming

```go
// Pattern: Test<Service><Method><Scenario>
func TestVMServiceCreateVM_ValidRequest(t *testing.T)
func TestVMServiceCreateVM_VMAlreadyExists(t *testing.T)
func TestVMServiceDeleteVM_VMNotFound(t *testing.T)
func TestAuthServiceLogin_InvalidCredentials(t *testing.T)
func TestBackupServiceRunBackup_Success(t *testing.T)
```

### Table-Driven Tests

```go
func TestVMServiceValidateVMName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantError bool
    }{
        {"valid name", "my-vm", false},
        {"empty name", "", true},
        {"too long", strings.Repeat("a", 65), true},
        {"invalid chars", "my@vm", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateVMName(tt.input)
            if (err != nil) != tt.wantError {
                t.Errorf("ValidateVMName() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

### Test Helpers

```go
// Pattern: new<Type> or create<Type>
func newTestVM(t *testing.T) *model.VM
func createTestUser(t *testing.T, role auth.Role) *auth.User
func setupAuthService(t *testing.T) (*AuthService, *sql.DB)
```

## Comments and Documentation

### Package Comments

```go
// Package sysinfo provides system information utilities for platform-specific
// configuration and defaults.
package sysinfo
```

### Function Comments

```go
// CreateVM creates a new virtual machine with the specified configuration.
// The VM will be created in a stopped state. Use StartVM to start it.
//
// Returns ErrVMAlreadyExists if a VM with the same name already exists.
// Returns ErrInvalidConfig if the configuration is invalid.
func (s *VMService) CreateVM(ctx context.Context, req *model.VMCreateRequest) (*model.VM, error)
```

### TODO Comments

```go
// TODO: Add support for live migration
// TODO(martin): Refactor this to use the new repository pattern
// FIXME: This causes a race condition under high load
// NOTE: This is a temporary workaround until libvirt 10.0
```

## Code Organization

### File Structure

```go
// 1. Package declaration
package service

// 2. Imports (grouped and sorted)
import (
    // Standard library
    "context"
    "fmt"
    
    // Third-party
    "github.com/google/uuid"
    
    // Internal
    "github.com/doomedramen/lab/apps/api/internal/model"
    "github.com/doomedramen/lab/apps/api/internal/repository"
)

// 3. Constants
const (
    DefaultMemory = 4
    DefaultCPU    = 2
)

// 4. Variables
var (
    ErrVMNotFound = errors.New("VM not found")
)

// 5. Types
type VMService struct {
    // ...
}

// 6. Constructor
func NewVMService(...) *VMService {
    // ...
}

// 7. Methods (grouped by functionality)
// CRUD operations
func (s *VMService) Create(...) { }
func (s *VMService) Get(...) { }
func (s *VMService) Update(...) { }
func (s *VMService) Delete(...) { }

// Business logic
func (s *VMService) Start(...) { }
func (s *VMService) Stop(...) { }
func (s *VMService) Reboot(...) { }

// Helpers
func (s *VMService) validate(...) { }
func (s *VMService) buildXML(...) { }
```

## Import Ordering

Imports should be grouped and sorted:

```go
import (
    // 1. Standard library (alphabetical)
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"
    
    // 2. Third-party (alphabetical)
    "github.com/google/uuid"
    "github.com/go-chi/chi/v5"
    "golang.org/x/time/rate"
    
    // 3. Internal (alphabetical)
    "github.com/doomedramen/lab/apps/api/internal/config"
    "github.com/doomedramen/lab/apps/api/internal/model"
    "github.com/doomedramen/lab/apps/api/internal/repository"
)
```

## Logging

### Log Levels

```go
// DEBUG: Detailed technical information for debugging
slog.Debug("Processing VM request", "vmid", vmid, "action", action)

// INFO: Normal operational messages
slog.Info("VM created successfully", "vmid", vmid, "name", name)

// WARN: Unexpected but handled situations
slog.Warn("VM already running", "vmid", vmid)

// ERROR: Errors that prevent operation completion
slog.Error("Failed to create VM", "error", err, "vmid", vmid)
```

### Log Message Format

```go
// Use structured logging with key-value pairs
slog.Info("Backup completed",
    "backup_id", backup.ID,
    "vmid", backup.VMID,
    "size_bytes", backup.SizeBytes,
    "duration_ms", duration.Milliseconds(),
)

// Use snake_case for log keys
slog.Warn("Rate limit exceeded", "client_ip", ip, "endpoint", endpoint)
```

## Summary

| Category | Pattern | Example |
|----------|---------|---------|
| Services | `New<Service>Service` | `NewVMService` |
| Repositories | `New<Repo>Repository` | `NewUserRepository` |
| Handlers | `New<Handler>Handler` | `NewHealthHandler` |
| Variables | `<type>Svc`, `<type>Repo` | `vmSvc`, `userRepo` |
| Context | Always `ctx` | `func Foo(ctx context.Context)` |
| Errors | Always `err` | `if err != nil` |
| Error vars | `Err<Resource><Condition>` | `ErrVMNotFound` |
| Test funcs | `Test<Service><Method><Scenario>` | `TestVMServiceCreateVM_ValidRequest` |
