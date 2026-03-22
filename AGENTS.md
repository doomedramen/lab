# AGENTS.md — Guide for AI Agents

This document provides essential context for AI agents working on the Lab project.

---

## Project Overview

**Lab** is a home server virtualization management platform (similar to Proxmox/XCP-NG) consisting of:

| Component | Technology | Location |
|-----------|------------|----------|
| **API** | Go + ConnectRPC + Libvirt | `apps/api/` |
| **Web** | Next.js + React + Tailwind | `apps/web/` |
| **Proto** | Protocol Buffers (ConnectRPC) | `packages/proto/lab/v1/` |

---

## Monorepo Structure

```
lab/
├── apps/
│   ├── api/           # Go backend
│   │   ├── cmd/server/          # Entry points (main.go, init_*.go)
│   │   ├── internal/
│   │   │   ├── config/          # Configuration loading
│   │   │   ├── connectsvc/      # ConnectRPC services (VMs, containers, etc.)
│   │   │   ├── handler/         # ConnectRPC handlers (auth, proxy, etc.)
│   │   │   ├── middleware/      # HTTP middleware (auth, CORS, rate limiting)
│   │   │   ├── model/           # Domain models
│   │   │   ├── repository/      # Data access interfaces
│   │   │   │   ├── libvirt/     # Libvirt implementations
│   │   │   │   ├── sqlite/      # SQLite implementations
│   │   │   │   ├── docker/      # Docker implementations
│   │   │   │   └── mock/        # Test mocks
│   │   │   ├── service/         # Business logic layer
│   │   │   └── router/          # HTTP routing
│   │   ├── pkg/sqlite/migrations/  # Database migrations
│   │   └── config.example.yaml     # Example configuration
│   │
│   └── web/           # Next.js frontend
│       ├── app/                 # App router pages
│       │   └── (dashboard)/     # Authenticated pages
│       ├── components/          # React components
│       └── lib/api/             # API client, queries, mutations
│
├── packages/
│   └── proto/lab/v1/            # Protocol buffer definitions
│
├── Makefile                     # Build and test commands
├── PLAN.md                      # Project roadmap and feature status
└── lefthook.yml                 # Git hooks (pre-commit, pre-push)
```

---

## Essential Commands

### Development

```bash
# Start both API and web dev servers
make dev
# or
pnpm dev
```

### Testing

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run E2E tests
make test-e2e

# Run Go tests directly (in apps/api)
go test ./...

# Run specific Go test
go test ./internal/service/... -v
```

### Docker CI (Recommended for Testing)

Most CI operations should use Docker to ensure consistent environments:

```bash
# Build CI Docker image
make docker-ci

# Run tests in Docker
make docker-test

# Run linters in Docker
make docker-lint

# Run Go vet in Docker
make docker-vet

# Interactive Docker shell
make docker-shell
```

### Code Quality

```bash
# Go vet
go vet ./...

# Type check (web)
pnpm check-types
```

---

## Go API Conventions

### Module Path

```go
import "github.com/doomedramen/lab/apps/api/..."
```

### Architecture Pattern

The API follows a layered architecture:

```
Handler (ConnectRPC) → Service (Business Logic) → Repository (Data Access)
```

- **Handlers** (`internal/handler/`, `internal/connectsvc/`) — Validate input, call service, return ConnectRPC responses
- **Services** (`internal/service/`) — Business logic, orchestration, transaction management
- **Repositories** (`internal/repository/`) — Data access interfaces with implementations in subdirectories

### Adding a New Feature

1. **Model** — Add domain types to `internal/model/`
2. **Repository Interface** — Add interface to `internal/repository/repository.go`
3. **Repository Implementation** — Implement in `internal/repository/sqlite/` (and `libvirt/` if needed)
4. **Migration** — Add schema changes to `pkg/sqlite/migrations/NNN_feature.sql`
5. **Service** — Add business logic to `internal/service/`
6. **Proto** — Define RPCs in `packages/proto/lab/v1/`
7. **Handler** — Implement ConnectRPC handler in `internal/handler/` or `internal/connectsvc/`
8. **Wire** — Register in `cmd/server/main.go` and `internal/router/router.go`

### Handler Input Validation

Validate inputs **before** calling the service layer. Return `connect.CodeInvalidArgument`:

```go
func (h *Handler) CreateFoo(ctx context.Context, req *connect.Request[foov1.CreateFooRequest]) (*connect.Response[foov1.CreateFooResponse], error) {
    if req.Msg.Name == "" {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
    }
    // ... call service
}
```

### Error Handling

Use typed errors from `internal/errors/` package:

```go
import "github.com/doomedramen/lab/apps/api/internal/errors"

// Return typed errors
return nil, errors.NotFound("VM not found")
return nil, errors.InvalidInput("invalid name")
return nil, errors.Internal("database error", err)
```

### Configuration

Configuration is loaded from YAML with environment variable overrides. See `apps/api/config.example.yaml` for all options.

Key config paths:
- `cfg.Server.Port`, `cfg.Server.Env`
- `cfg.Libvirt.URI`
- `cfg.Storage.ISODir`, `cfg.Storage.VMDiskDir`
- `cfg.Backend` (BackendMock | BackendLibvirt)

---

## Frontend Conventions (Next.js)

### API Client

The frontend uses ConnectRPC client generated from proto files:

```typescript
// lib/api/client.ts
import { createPromiseClient } from "@connectrpc/connect";

// Query hooks in lib/api/queries/
// Mutation hooks in lib/api/mutations/
```

### Adding a New Page

1. Create page in `app/(dashboard)/feature/page.tsx`
2. Add query hooks in `lib/api/queries/feature.ts`
3. Add mutation hooks in `lib/api/mutations/feature.ts`
4. Add navigation item in `components/app-sidebar.tsx`

### Shadcn/ui Components

- **DO NOT** create new files in `components/ui/`
- **CAN** modify existing files for bug fixes
- Add new components via: `pnpm dlx shadcn@latest add [component]`

---

## Git Hooks (Lefthook)

The project uses Lefthook for local CI:

- **pre-commit**: `go vet ./...` + `pnpm check-types` (parallel)
- **pre-push**: `go test ./...` + `pnpm check-types` (sequential)

Install hooks: `lefthook install` (or just `pnpm install`)

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `apps/api/cmd/server/main.go` | Application entry, wiring dependencies |
| `apps/api/internal/config/config.go` | Configuration struct |
| `apps/api/internal/repository/repository.go` | Repository interfaces |
| `apps/api/pkg/sqlite/migrations/` | Database schema migrations |
| `packages/proto/lab/v1/` | API protocol definitions |
| `apps/api/config.example.yaml` | Example configuration |
| `PLAN.md` | Project roadmap and feature status |

---

## Common Patterns

### Creating a Repository

```go
// internal/repository/repository.go
type FooRepository interface {
    Create(ctx context.Context, foo *model.Foo) error
    Get(ctx context.Context, id string) (*model.Foo, error)
    List(ctx context.Context) ([]*model.Foo, error)
    Update(ctx context.Context, foo *model.Foo) error
    Delete(ctx context.Context, id string) error
}

// internal/repository/sqlite/foo.go
type fooRepository struct {
    db *sql.DB
}

func NewFooRepository(db *sql.DB) repository.FooRepository {
    return &fooRepository{db: db}
}
```

### Creating a Service

```go
// internal/service/foo.go
type FooService struct {
    repo   repository.FooRepository
    logger *slog.Logger
}

func NewFooService(repo repository.FooRepository, logger *slog.Logger) *FooService {
    return &FooService{repo: repo, logger: logger}
}

func (s *FooService) Create(ctx context.Context, foo *model.Foo) error {
    // Business logic here
    return s.repo.Create(ctx, foo)
}
```

### Creating a Handler

```go
// internal/handler/foo.go
type FooHandler struct {
    service *service.FooService
}

func NewFooHandler(service *service.FooService) *FooHandler {
    return &FooHandler{service: service}
}

func (h *FooHandler) Create(
    ctx context.Context,
    req *connect.Request[foov1.CreateFooRequest],
) (*connect.Response[foov1.CreateFooResponse], error) {
    // Validate input
    if req.Msg.Name == "" {
        return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
    }

    // Call service
    foo := &model.Foo{Name: req.Msg.Name}
    if err := h.service.Create(ctx, foo); err != nil {
        return nil, err
    }

    return connect.NewResponse(&foov1.CreateFooResponse{Foo: foo.ToProto()}), nil
}
```

---

## Testing Guidelines

### Go Tests

- Unit tests go alongside source files: `foo.go` → `foo_test.go`
- Integration tests use `_integration_test.go` suffix
- Use test helpers from `internal/handler/connect_test_helpers_test.go`

### Running Specific Tests

```bash
# Run tests matching a pattern
go test ./internal/service/... -run TestVMService -v

# Run tests with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Important Notes

1. **No time pressure** — Take time to understand the codebase and write quality code
2. **Check code before marking tasks complete** — Verify implementations work
3. **Update/run tests** — Maintain test coverage
4. **Commit big changes** — Create meaningful commits for significant work
5. **No shortcuts** — Implement properly with error handling

---

## Related Documentation

- `PLAN.md` — Project roadmap and feature status
