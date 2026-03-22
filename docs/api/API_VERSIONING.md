# API Versioning Strategy

This document describes the versioning strategy for the Lab API.

## Versioning Scheme

The Lab API uses **protobuf-based versioning** via Connect RPC:

- **Proto package**: `lab.v1`, `lab.v2`, etc.
- **Connect RPC paths**: `/lab.v1.VMService/CreateVM`
- **Go package**: `github.com/doomedramen/lab/apps/api/gen/lab/v1`

### Breaking Changes Require New Version

A **breaking change** is any modification that would cause existing clients to fail:

- Removing or renaming a field
- Changing a field's type
- Removing a service or RPC method
- Changing the semantics of an existing field in a incompatible way

**Non-breaking changes** (safe to make in existing version):

- Adding new optional fields
- Adding new services or RPC methods
- Adding new enum values (if clients handle `UNSPECIFIED` default)

## Deprecation Policy

When a field, service, or method needs to be removed:

### Phase 1: Deprecation Notice (6 months)

1. Add `deprecated = true` to the proto field:

   ```protobuf
   // Deprecated: Use new_field instead. This field will be removed in v2.
   string old_field = 1 [deprecated = true];

   string new_field = 2;
   ```

2. Update handler to log deprecation warnings:

   ```go
   if req.OldField != "" {
       slog.Warn("Deprecated field used",
           "field", "old_field",
           "replacement", "new_field",
           "removal_version", "v2.0")
   }
   ```

3. Add deprecation headers to responses:
   ```go
   // In Connect RPC handler
   response.Deprecated = true
   response.DeprecationNotice = "Use new_field instead. Will be removed in v2.0"
   ```

### Phase 2: Warning Period (6+ months)

- Log all usage of deprecated fields
- Include deprecation warnings in API responses
- Update documentation to highlight deprecated fields
- Notify users via release notes

### Phase 3: Sunset (next major version)

- Remove deprecated fields in `v2`
- Update all proto definitions
- Regenerate code
- Update migration guide

## HTTP Response Headers

For REST endpoints (health checks, metrics, WebSocket), include deprecation headers:

```http
Deprecation: true
Sunset: Sat, 01 Mar 2027 00:00:00 GMT
Link: </api/v2>; rel="successor-version"
X-API-Version: v1.2.3
```

## Migration Guide

For each major version, provide:

### 1. Migration Guide Document

Create `MIGRATION_v1_TO_v2.md` with:

- List of breaking changes
- Step-by-step migration instructions
- Code examples for common migrations
- Timeline and deprecation schedule

### 2. Codemods (where possible)

Automate migration with scripts:

```bash
# Example: migrate from v1 to v2
./scripts/migrate-v2.sh --dry-run
./scripts/migrate-v2.sh --apply
```

### 3. Compatibility Layer (optional)

For critical fields, provide temporary compatibility:

```go
// In v2 handler
func (h *VMHandler) CreateVM(ctx context.Context, req *v2.CreateVMRequest) (*v2.CreateVMResponse, error) {
    // Support old field name for one version
    if req.Memory == 0 && req.OldMemory != 0 {
        slog.Warn("Using deprecated field", "field", "old_memory")
        req.Memory = req.OldMemory
    }
    // ... rest of handler
}
```

## Version Support Matrix

| API Version | Status      | Supported Until | Notes           |
| ----------- | ----------- | --------------- | --------------- |
| v1          | Current     | TBD             | Initial release |
| v2          | Development | -               | In planning     |

## Frontend Integration

The Next.js frontend uses Connect RPC clients:

```typescript
// apps/web/lib/api/client.ts
import { VMServiceClient } from "@/gen/lab/v1/labv1connect";

const vmClient = new VMServiceClient(httpClient, baseUrl, {
  interceptors: [authInterceptor],
});

// Usage
const response = await vmClient.createVM({
  name: "my-vm",
  memory: 4,
  // ...
});
```

When migrating to v2:

1. Generate new proto clients: `make proto`
2. Update client imports: `lab/v1` → `lab/v2`
3. Update field names in mutations
4. Test all affected components

## Backend Implementation

### Proto Definitions

```protobuf
// packages/proto/lab/v1/vm.proto
syntax = "proto3";
package lab.v1;

service VMService {
  rpc CreateVM(CreateVMRequest) returns (CreateVMResponse);
  rpc UpdateVM(UpdateVMRequest) returns (UpdateVMResponse);
  // ... more RPCs
}

message CreateVMRequest {
  string name = 1;
  int32 memory = 2;
  // Deprecated: Use memory instead
  int32 old_memory = 3 [deprecated = true];
}
```

### Handler Implementation

```go
// apps/api/internal/connectsvc/vm.go
func (s *vmServiceServer) CreateVM(
    ctx context.Context,
    req *connect.Request[labv1.CreateVMRequest],
) (*connect.Response[labv1.CreateVMResponse], error) {

    // Check for deprecated field usage
    if req.Msg.OldMemory != 0 {
        slog.Warn("Deprecated field used",
            "field", "old_memory",
            "replacement", "memory",
            "removal_version", "v2.0")
    }

    // ... rest of handler
}
```

## Testing Strategy

### Proto Linting

Run `buf lint` to catch proto violations:

```bash
cd packages/proto && buf lint
```

### Backward Compatibility Tests

Ensure old clients still work:

```go
func TestCreateVM_BackwardCompatibility(t *testing.T) {
    // Test with deprecated field
    req := &labv1.CreateVMRequest{
        OldMemory: 4096, // Deprecated but should still work
    }

    resp, err := handler.CreateVM(ctx, req)

    // Should succeed but log warning
    assert.NoError(t, err)
    assert.Equal(t, int32(4096), resp.Memory)
}
```

## Release Process

1. **Announce deprecation** in release notes
2. **Update documentation** with migration guide
3. **Monitor usage** via logs and metrics
4. **Set sunset date** (minimum 6 months from announcement)
5. **Remove in next major version**

## Communication

Deprecation announcements via:

- GitHub release notes
- In-dashboard notifications
- Email notifications (for enterprise users)
- Documentation updates

## Examples

### Example 1: Renaming a Field

**Before (v1):**

```protobuf
message VM {
  string vm_name = 1;
}
```

**After (v2):**

```protobuf
message VM {
  string name = 1;  // Renamed from vm_name
  // Deprecated: Use name instead
  string vm_name = 2 [deprecated = true];
}
```

**Migration:**

```typescript
// v1 client
const vm = await vmClient.getVM({ vmid: 100 });
console.log(vm.vm_name);

// v2 client
const vm = await vmClient.getVM({ vmid: 100 });
console.log(vm.name);
```

### Example 2: Removing a Service Method

**Before (v1):**

```protobuf
service VMService {
  rpc GetVMStats(GetVMStatsRequest) returns (GetVMStatsResponse);
}
```

**After (v2):**

```protobuf
service VMService {
  // GetVMStats removed - use GetVMMetrics instead
  rpc GetVMMetrics(GetVMMetricsRequest) returns (GetVMMetricsResponse);
}
```

## Summary

- **Breaking changes** → new major version (v1 → v2)
- **Deprecation period** → minimum 6 months
- **Migration guides** → required for each major version
- **Backward compatibility** → supported for one version
- **Communication** → release notes, docs, in-app notifications
