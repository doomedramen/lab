# Error Handling Policy

This document describes the error handling policy for the Lab API codebase.

## Principles

1. **Fail Fast** - Validate inputs early and return clear errors
2. **Preserve Context** - Wrap errors with operation context, don't swallow them
3. **Use Structured Errors** - Use `APIError` types for consistent error responses
4. **Log Appropriately** - Log errors at the right level (DEBUG, INFO, WARN, ERROR)
5. **Never Panic** - Return errors instead of panicking (except for unrecoverable bugs)
6. **Don't Leak Details** - Never expose internal details to clients

## Error Types

### APIError (Base Type)

All application errors should be `*APIError` instances:

```go
type APIError struct {
    Code       ErrorCode         // Application error code
    Message    string            // Human-readable message
    Details    map[string]string // Additional metadata
    Cause      error             // Underlying cause (for wrapping)
    StackTrace string            // Stack trace for debugging
    Operation  string            // Operation being performed
    Resource   string            // Resource type (e.g., "vm", "user")
    ResourceID string            // Resource identifier
}
```

### Error Codes

| Code                 | HTTP Status | Description                          |
| -------------------- | ----------- | ------------------------------------ |
| `OK`                 | 200         | No error                             |
| `INVALID_ARGUMENT`   | 400         | Client specified invalid argument    |
| `NOT_FOUND`          | 404         | Resource not found                   |
| `ALREADY_EXISTS`     | 409         | Resource already exists              |
| `PERMISSION_DENIED`  | 403         | Insufficient permissions             |
| `UNAUTHENTICATED`    | 401         | Authentication required              |
| `INTERNAL`           | 500         | Internal server error                |
| `UNAVAILABLE`        | 503         | Service unavailable                  |
| `CONFLICT`           | 409         | Request conflicts with current state |
| `RESOURCE_EXHAUSTED` | 429         | Resource quota exceeded              |
| `CANCELLED`          | 408         | Operation cancelled                  |
| `DEADLINE_EXCEEDED`  | 504         | Deadline expired                     |
| `UNIMPLEMENTED`      | 501         | Operation not implemented            |
| `DATA_LOSS`          | 500         | Unrecoverable data loss              |

## Creating Errors

### Simple Errors

```go
// Basic error
err := errors.New(errors.NotFound, "resource not found")

// With resource context
err := errors.NewNotFoundError("vm", "100")

// With field details
err := errors.NewInvalidArgumentError("memory", "must be greater than 0")
```

### Wrapping Errors

```go
// Wrap with context
err := errors.Wrap(cause, errors.Internal, "failed to save VM")

// Wrap with formatted message
err := errors.Wrapf(cause, errors.Internal, "failed to %s VM", operation)

// Specialized wrappers
err := errors.NewInternalError(cause, "database connection failed")
err := errors.NewInternalErrorf(cause, "timeout after %d seconds", 30)
```

### Adding Metadata

```go
err := errors.New(errors.InvalidArgument, "invalid configuration").
    WithDetails("field", "memory").
    WithDetails("value", "-1").
    WithOperation("create VM").
    WithResource("vm", "100")
```

## Checking Errors

### Type Checks

```go
// Check error type
if errors.IsNotFound(err) {
    // Handle not found
}

if errors.IsInvalidArgument(err) {
    // Handle invalid argument
}

if errors.IsPermissionDenied(err) {
    // Handle permission denied
}
```

### Extract Information

```go
// Get error code
code := errors.GetErrorCode(err)

// Get HTTP status
status := errors.GetHTTPStatus(err)

// Access error details
if apiErr, ok := err.(*errors.APIError); ok {
    field := apiErr.Details["field"]
    operation := apiErr.Operation
}
```

### Error Chaining

```go
// Check if error wraps a specific error
if errors.Is(err, specificError) {
    // Handle specific error
}

// Unwrap to get cause
var apiErr *errors.APIError
if errors.As(err, &apiErr) {
    cause := apiErr.Unwrap()
}
```

## Logging Errors

### Log Levels

```go
// DEBUG - Detailed technical information for debugging
slog.Debug("Database query executed",
    "query", query,
    "duration_ms", duration.Milliseconds(),
    "rows_affected", rows)

// INFO - Normal operational messages
slog.Info("VM created successfully",
    "vmid", vmid,
    "name", name)

// WARN - Unexpected but handled situations
slog.Warn("VM already running",
    "vmid", vmid,
    "requested_state", "running",
    "current_state", "running")

// ERROR - Errors that prevent operation completion
slog.Error("Failed to create VM",
    "error", err,
    "vmid", vmid,
    "operation", "create")
```

### Error Logging Pattern

```go
// In service layer
func (s *VMService) CreateVM(ctx context.Context, req *CreateVMRequest) (*VM, error) {
    vm, err := s.repo.Create(ctx, req)
    if err != nil {
        // Log with full context
        slog.Error("Failed to create VM",
            "error", err,
            "name", req.Name,
            "operation", "create_vm")

        // Return wrapped error
        return nil, errors.NewInternalError(err, "failed to create VM")
    }
    return vm, nil
}

// In handler layer
func (h *vmHandler) CreateVM(ctx context.Context, req *connect.Request[labv1.CreateVMRequest]) (*connect.Response[labv1.CreateVMResponse], error) {
    vm, err := h.vmSvc.CreateVM(ctx, req.Msg)
    if err != nil {
        // Don't log here - already logged in service layer
        // Just convert to Connect error
        return nil, connectErrorFromAPIError(err)
    }
    return connect.NewResponse(vm), nil
}
```

## Error Handling by Layer

### Repository Layer

```go
// Return specific errors for known conditions
func (r *VMRepository) GetByID(ctx context.Context, id string) (*model.VM, error) {
    row := r.db.QueryRowContext(ctx, "SELECT * FROM vms WHERE id = ?", id)

    vm, err := scanVM(row)
    if err == sql.ErrNoRows {
        return nil, errors.NewNotFoundError("vm", id)
    }
    if err != nil {
        return nil, errors.NewInternalError(err, "failed to query VM")
    }

    return vm, nil
}
```

### Service Layer

```go
// Wrap repository errors with business context
func (s *VMService) StartVM(ctx context.Context, vmid int) error {
    vm, err := s.vmRepo.GetByVMID(ctx, vmid)
    if err != nil {
        // Let NotFound errors propagate
        return err
    }

    if vm.Status == model.VMStatusRunning {
        return errors.NewConflictError("VM is already running")
    }

    if err := s.libvirt.StartDomain(vmid); err != nil {
        return errors.NewInternalError(err, "failed to start domain")
    }

    return nil
}
```

### Handler Layer

```go
// Convert API errors to Connect errors
func connectErrorFromAPIError(err *errors.APIError) *connect.Error {
    var code connect.Code
    switch err.Code {
    case errors.InvalidArgument:
        code = connect.CodeInvalidArgument
    case errors.NotFound:
        code = connect.CodeNotFound
    case errors.AlreadyExists:
        code = connect.CodeAlreadyExists
    case errors.PermissionDenied:
        code = connect.CodePermissionDenied
    case errors.Unauthenticated:
        code = connect.CodeUnauthenticated
    case errors.Internal:
        code = connect.CodeInternal
    case errors.Unavailable:
        code = connect.CodeUnavailable
    case errors.Conflict:
        code = connect.CodeConflict
    default:
        code = connect.CodeInternal
    }

    connectErr := connect.NewError(code, err)

    // Add error details
    for key, value := range err.Details {
        connectErr.Meta().Set(key, value)
    }

    return connectErr
}

func (h *vmHandler) GetVM(ctx context.Context, req *connect.Request[labv1.GetVMRequest]) (*connect.Response[labv1.GetVMResponse], error) {
    vm, err := h.vmSvc.GetVM(ctx, int(req.Msg.Vmid))
    if err != nil {
        return nil, connectErrorFromAPIError(err)
    }
    return connect.NewResponse(modelToProto(vm)), nil
}
```

## Panic Recovery

```go
// In HTTP middleware
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if r := recover(); r != nil {
                // Log panic with stack trace
                slog.Error("Panic recovered",
                    "error", r,
                    "stack", string(debug.Stack()),
                    "path", r.URL.Path)

                // Return 500 to client
                http.Error(w, "Internal server error", http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}
```

## Testing Errors

```go
func TestVMService_CreateVM_InvalidName(t *testing.T) {
    svc := setupVMService(t)

    _, err := svc.CreateVM(context.Background(), &CreateVMRequest{
        Name: "", // Invalid: empty name
    })

    if !errors.IsInvalidArgument(err) {
        t.Errorf("Expected InvalidArgument error, got %v", err)
    }

    apiErr, ok := err.(*errors.APIError)
    if !ok {
        t.Fatal("Expected APIError")
    }

    if apiErr.Details["field"] != "name" {
        t.Errorf("Expected field=name, got %s", apiErr.Details["field"])
    }
}

func TestVMService_GetVM_NotFound(t *testing.T) {
    svc := setupVMService(t)

    _, err := svc.GetVM(context.Background(), 99999)

    if !errors.IsNotFound(err) {
        t.Errorf("Expected NotFound error, got %v", err)
    }

    apiErr := err.(*errors.APIError)
    if apiErr.Resource != "vm" {
        t.Errorf("Expected resource=vm, got %s", apiErr.Resource)
    }
}
```

## Migration Guide

### From Standard Errors

**Before:**

```go
if err != nil {
    return fmt.Errorf("failed to create VM: %w", err)
}
```

**After:**

```go
if err != nil {
    return errors.NewInternalError(err, "failed to create VM")
}
```

### From String Comparison

**Before:**

```go
if err.Error() == "VM not found" {
    // Handle not found
}
```

**After:**

```go
if errors.IsNotFound(err) {
    // Handle not found
}
```

### From Error Code Constants

**Before:**

```go
if err == ErrVMNotFound {
    // Handle not found
}
```

**After:**

```go
if errors.IsNotFound(err) {
    // Handle not found
}
```

## Summary

| Do                                  | Don't                      |
| ----------------------------------- | -------------------------- |
| Use `APIError` types                | Return raw `fmt.Errorf()`  |
| Wrap errors with context            | Swallow errors with `_`    |
| Log at appropriate level            | Log everything as ERROR    |
| Check errors with `Is*` functions   | Compare error strings      |
| Preserve error chains with `Wrap()` | Lose context when wrapping |
| Return errors, don't panic          | Use panic for control flow |
| Sanitize error messages for clients | Expose internal details    |
