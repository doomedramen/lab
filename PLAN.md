# Lab — Project Plan

This plan covers the highest-priority feature gaps relative to Proxmox/XCP-NG
for a home server use case. Clustering and HA are explicitly out of scope.
Items are ordered by operational impact: things that break the existing
experience come before nice-to-haves.

---

## Bug Fixes (Post-Implementation)

These are bugs discovered and fixed after the initial implementation of features.

### VM Metrics Collection — Wrong ID Field

**Status:** ✅ **FIXED**

Completed.

---

### False "VM Started" Logs on API Restart

**Status:** ✅ **FIXED**

Completed.

---

### Inconsistent VM Log Messages

**Status:** ✅ **FIXED**

Completed.

---

### VM Log Viewer — Incorrect Timestamp Order

**Status:** ✅ **FIXED**

Completed.

---

### Login Redirect Ignores `from` Query Parameter

**Status:** ✅ **FIXED**

Completed.

---

### QEMU Guest Agent Segfault

**Status:** ✅ **FIXED**

Completed.

---

### All `fetch()` Usage Removed — Migrated to ConnectRPC

**Status:** ✅ **FIXED**

Completed.

---

## Phase 1 — Operational Foundation

These gaps make the platform unreliable or unusable in day-to-day operation.
They should be addressed before adding any new features.

---

### 1.1 Task / Job Tracking

**Status:** ✅ **COMPLETED**

Completed.

---

### 1.2 VM Update (Fix the Stub)

**Status:** ✅ **COMPLETED**

Completed.

---

### 1.3 Disk Management (Add / Remove / Resize)

**Status:** ✅ **COMPLETED**

Completed.

---

### 1.4 VM Clone

**Status:** ✅ **COMPLETED**

Completed.

---

## Phase 2 — Home Server Essentials

Features that meaningfully differentiate a home server from a bare hypervisor.

---

### 2.1 Alerting System

**Status:** ✅ **COMPLETED**

Completed.

---

### 2.2 QEMU Guest Agent Integration

**Status:** ✅ **COMPLETED**

Completed.

---

### 2.3 TPM 2.0 + Secure Boot

**Status:** ✅ **COMPLETED**

Completed.

---

### 2.4 PCI / GPU Passthrough

**Status:** ✅ **COMPLETED**

Completed.

---

## Phase 3 — Management & Hardening

---

### 3.1 Auth Rate Limiting

**Status:** ✅ **COMPLETED**

Completed.

---

### 3.2 Host Shell Access

**Status:** ✅ **COMPLETED**

Completed.

---

### 3.3 Boot Order Configuration

**Status:** ✅ **COMPLETED**

Completed.

---

### 3.4 Backup Improvements

**Status:** ✅ **COMPLETED**

Completed.

---

### 3.5 Session Management

**Status:** ✅ **COMPLETED**

Completed.

---

## Phase 4 — RBAC & Security Polish

---

### 4.1 User Groups

**Why.** Currently RBAC is per-user with three roles. You cannot give a
group of users the same role, and you cannot delegate access to a subset
of VMs.

**Deliverable.** User groups with role assignment; basic resource pools
(a named set of VMs a group can see/manage).

**Files to create / modify:**

| File                                            | Change                                                                         |
| ----------------------------------------------- | ------------------------------------------------------------------------------ |
| `apps/api/pkg/sqlite/migrations/011_groups.sql` | New — groups, user_groups, resource_pools, pool_members tables                 |
| `apps/api/internal/model/auth.go`               | Add `Group`, `ResourcePool` models                                             |
| `apps/api/internal/repository/sqlite/group.go`  | New — CRUD                                                                     |
| `apps/api/internal/service/auth.go`             | Extend with group management; update permission checks to consider group roles |
| `packages/proto/lab/v1/auth.proto`              | Add group and pool CRUD RPCs                                                   |
| `apps/web/app/(dashboard)/settings/page.tsx`    | Admin section: Groups tab                                                      |

**Complexity:** Medium.

---

### 4.2 IP Whitelisting

**Why.** For a home server accessible from outside the LAN (via VPN or port
forward), IP whitelisting on the API is a simple and effective security layer.

**Files to modify:**

| File                                          | Change                                                                                  |
| --------------------------------------------- | --------------------------------------------------------------------------------------- |
| `apps/api/internal/middleware/ipwhitelist.go` | New — configurable CIDR allow list; applied before auth middleware; 403 if not matching |
| `apps/api/internal/config/config.go`          | Add `Security.AllowedCIDRs []string`                                                    |
| `apps/api/config.example.yaml`                | Document the config key                                                                 |

**Complexity:** Very Low.

---

### 4.3 TLS / Certificate Management

**Why.** The server currently runs plain HTTP. For any setup beyond
`localhost`, this is unacceptable.

**Option A — Built-in TLS (low complexity):**
Accept a cert and key path in config; serve HTTPS directly.

| File                                 | Change                                                |
| ------------------------------------ | ----------------------------------------------------- |
| `apps/api/internal/config/config.go` | Add `Server.TLSCertFile`, `Server.TLSKeyFile`         |
| `apps/api/cmd/server/main.go`        | If cert+key configured, call `http.ListenAndServeTLS` |

**Option B — ACME / Let's Encrypt (medium complexity):**
Use `golang.org/x/crypto/acme/autocert` for automatic cert provisioning.

| File                                 | Change                                             |
| ------------------------------------ | -------------------------------------------------- |
| `apps/api/internal/config/config.go` | Add `Server.ACMEDomain`, `Server.ACMEEmail`        |
| `apps/api/cmd/server/main.go`        | Set up autocert manager and wire to HTTPS listener |

**Recommendation:** Implement Option A first (20 minutes of work). Document
Caddy as the recommended reverse proxy for users who want automatic HTTPS.
Option B can follow if there's demand.

**Complexity:** Very Low (A) / Low (B).

---

## Phase 5 — Reverse Proxy & Uptime Monitoring

A built-in reverse proxy manager similar to Nginx Proxy Manager, providing
domain-based routing for VMs, LXC containers, and Docker stacks with automatic
HTTPS/SSL management and integrated uptime monitoring.

Some references for reverse proxy implementations:

- https://github.com/yusing/godoxy
- https://pkg.go.dev/net/http/httputil#example-ReverseProxy
- https://github.com/caddyserver/caddy

---

### 5.1 Reverse Proxy Core

**Status:** ✅ **COMPLETED**

**Decision:** Pure Go `net/http/httputil.ReverseProxy` + stdlib `crypto/tls` for self-signed certs (no external deps). ACME mode stubbed for future integration.

**SSL modes implemented:**

| Mode | Description |
|------|-------------|
| `none` | HTTP only |
| `self_signed` | Auto-generated self-signed cert (stdlib) |
| `acme` | Stub — returns "not yet implemented" |
| `custom` | User-uploaded cert + key |

**Files created / modified:**

| File | Change |
|------|--------|
| `apps/api/pkg/sqlite/migrations/011_proxy.sql` | New — proxy_hosts and proxy_host_certs tables |
| `apps/api/internal/model/proxy.go` | New — ProxyHost, ProxyCert, ProxyStatus models |
| `apps/api/internal/repository/repository.go` | Added ProxyRepository interface |
| `apps/api/internal/repository/sqlite/proxy.go` | New — SQLite implementation |
| `apps/api/internal/config/config.go` | Added ProxyConfig (http_port, https_port, acme_email, acme_storage_dir, enabled) |
| `apps/api/config.example.yaml` | Added proxy section |
| `apps/api/internal/service/proxy.go` | New — dynamic routing with RWMutex-protected map, SNI cert selection, basic auth, self-signed cert generation |
| `packages/proto/lab/v1/proxy.proto` | New — full CRUD + GetProxyStatus + UploadCert RPCs |
| `apps/api/internal/handler/proxy.go` | New — ConnectRPC handler with input validation |
| `apps/api/internal/handler/proxy_handler_test.go` | New — validation tests for all RPCs |
| `apps/api/cmd/server/main.go` | Wire proxyRepo, proxySvc (Start/Stop) |
| `apps/api/internal/router/router.go` | Register ProxyServiceHandler |
| `apps/web/lib/api/queries/proxy.ts` | New — useProxyHosts, useProxyStatus hooks |
| `apps/web/lib/api/mutations/proxy.ts` | New — CRUD mutations + useUploadCert |
| `apps/web/app/(dashboard)/proxy/page.tsx` | New — proxy management page with table, dialogs, live status badges |
| `apps/web/components/app-sidebar.tsx` | Added Proxy nav item |

---

### 5.2 VM/Container Proxy Integration

**Why.** Proxy hosts should be configurable as part of VM/Container settings,
not as a separate management task.

**Deliverable.** "Proxy Hosts" tab on VM and Container detail pages showing
active proxy mappings and allowing quick configuration.

**Files to modify:**

| File                                                | Change                                                                                      |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `apps/api/internal/model/vm.go`                     | Add `ProxyHosts []ProxyHostConfig` to `VMCreateRequest` and `VMUpdateRequest`               |
| `apps/api/internal/model/container.go`              | Add `ProxyHosts []ProxyHostConfig` to `ContainerCreateRequest` and `ContainerUpdateRequest` |
| `apps/api/internal/service/vm.go`                   | On VM update, sync proxy hosts (create/update/delete as needed)                             |
| `apps/api/internal/service/container.go`            | Same for containers                                                                         |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx`      | Add "Proxy Hosts" tab                                                                       |
| `apps/web/app/(dashboard)/containers/[id]/page.tsx` | Add "Proxy Hosts" tab                                                                       |

**Auto-discovery mode:**

- Optional `autoProxy: true` flag on VM/Container
- Automatically creates `vmname.defaultdomain.lab` → IP:80
- Useful for quick internal access without manual config

**Complexity:** Low. Extending existing models and wiring sync logic.

---

### 5.3 Uptime Monitoring

**Status:** ✅ **COMPLETED**

**Design.** Monitors are decoupled from proxy hosts — any URL can be monitored manually, and proxy hosts auto-create monitors on creation. Background polling loop (15s tick, per-monitor intervals) with per-monitor HTTP checks. Alert integration fires `uptime_check_failed` alerts after 3 consecutive failures.

**Files created / modified:**

| File | Change |
|------|--------|
| `apps/api/pkg/sqlite/migrations/012_uptime.sql` | New — uptime_monitors (nullable proxy_host_id FK), uptime_results tables |
| `apps/api/internal/model/proxy.go` | Add UptimeMonitor, UptimeResult, UptimeStats, UptimeMonitorStatus models |
| `apps/api/internal/model/alert.go` | Add AlertTypeUptimeCheckFailed |
| `apps/api/internal/repository/repository.go` | Extended ProxyRepository with uptime methods |
| `apps/api/internal/repository/sqlite/proxy.go` | Add CreateMonitor, ListMonitors, ListEnabledMonitors, UpdateMonitor, DeleteMonitor, LogUptimeResult, GetUptimeHistory, GetUptimeStats, PruneOldResults |
| `apps/api/internal/service/proxy.go` | Add monitoring loop, monitor CRUD, auto-create monitor on proxy host create, UptimeAlertSender interface, UptimeProvider implementation |
| `apps/api/internal/service/alert.go` | Add UptimeProvider interface, WithUptimeProvider, gatherUptimeCheckFailed, FireUptimeAlert, uptime_check_failed message format |
| `packages/proto/lab/v1/proxy.proto` | Add UptimeMonitor, UptimeResult, UptimeStats messages + 7 new RPCs |
| `apps/api/internal/handler/proxy.go` | Add CreateMonitor, GetMonitor, ListMonitors, UpdateMonitor, DeleteMonitor, GetMonitorHistory, GetMonitorStats handlers |
| `apps/api/internal/handler/proxy_handler_test.go` | Add validation tests for all 7 new RPCs |
| `apps/api/cmd/server/main.go` | Wire alert sender + uptime provider |
| `apps/web/lib/api/queries/proxy.ts` | Add useMonitors, useMonitor, useMonitorStats, useMonitorHistory hooks |
| `apps/web/lib/api/mutations/proxy.ts` | Add useMonitorMutations |
| `apps/web/app/(dashboard)/proxy/page.tsx` | Add Monitors tab with status badges, uptime %, avg response, inline response time chart |

**Implemented Features:**

- Configurable intervals: 30s, 1m, 5m, 15m
- Expected HTTP status code per monitor (default 200)
- Response time tracking (stored per-check, displayed as area chart)
- Background polling with TLS verification disabled (intentional for internal monitoring)
- Auto-create monitor when proxy host is created (linked via proxy_host_id FK)
- Alert integration: `uptime_check_failed` rule type fires after 3 consecutive failures
- Uptime % computed for last 24h and 7d via SQL aggregation
- Old results pruned daily (30-day retention)

---

### Implementation Order within Phase 5

| #   | Item                     | Complexity  | Dependencies      |
| --- | ------------------------ | ----------- | ----------------- |
| 5.1 | Reverse Proxy Core       | Medium-High | None              | ✅ DONE |
| 5.2 | VM/Container Integration | Low         | 5.1               |         |
| 5.3 | Uptime Monitoring        | Medium      | 5.1, 2.1 (Alerts) |         |

---

### Out of Scope (Future Phases)

- Load balancing across multiple backends
- Rate limiting per domain (could use existing middleware)
- WAF / ModSecurity integration
- Custom nginx config snippets (advanced users)
- Multi-node proxy clustering (requires clustering feature)
- Analytics / traffic logs beyond uptime metrics

---

## Implementation Order Summary

| #   | Item                                | Phase | Complexity  | Impact                                          | Status      |
| --- | ----------------------------------- | ----- | ----------- | ----------------------------------------------- | ----------- |
| 1   | Task / job tracking                 | 1     | Medium      | Critical — async operations have no feedback    | ✅ **DONE** |
| 2   | VM Update (fix stub)                | 1     | Medium      | Critical — VMs cannot be modified post-creation | ✅ **DONE** |
| 3   | Disk management (add/remove/resize) | 1     | Medium      | High — fundamental VM management                | ✅ **DONE** |
| 4   | VM Clone                            | 1     | Medium      | High — required for template workflows          | ✅ **DONE** |
| 5   | Alerting system                     | 2     | Medium      | High — operational safety net                   | ✅ **DONE** |
| 6   | Guest agent integration             | 2     | Low-Medium  | High — IP discovery, consistent backups         | ✅ **DONE** |
| 7   | TPM 2.0 + Secure Boot               | 2     | Low         | High — Windows 11 support                       | 🔄 **NEXT** |
| 8   | PCI / GPU Passthrough               | 2     | Medium-High | High — home server differentiator               |             |
| 9   | Auth rate limiting                  | 3     | Very Low    | Critical security fix                           |             |
| 10  | Host shell access                   | 3     | Low         | Medium — useful for troubleshooting             |             |
| 11  | Boot order config                   | 3     | Low         | Medium                                          |             |
| 12  | Backup verification                 | 3     | Low         | Medium — reliability                            |             |
| 13  | Backup encryption                   | 3     | Low         | Medium                                          |             |
| 14  | Backup retention count              | 3     | Low         | Medium                                          |             |
| 15  | Session management                  | 3     | Low-Medium  | Medium — security                               |             |
| 16  | User groups                         | 4     | Medium      | Low — single-user home server                   |             |
| 17  | IP whitelisting                     | 4     | Very Low    | Low-Medium                                      |             |
| 18  | TLS support                         | 4     | Very Low    | Medium — needed for remote access               |             |
| 19  | Reverse Proxy Core                  | 5     | Medium-High | Medium — convenience, unified management        |             |
| 20  | VM/Container Proxy Integration      | 5     | Low         | Low-Medium — tight integration                  |             |
| 21  | Uptime Monitoring                   | 5     | Medium      | Medium — operational visibility                 |             |

---

## What Is Intentionally Out of Scope

- Live migration — requires shared storage or storage migration infrastructure
- Clustering / multi-node — architecture change
- High Availability — depends on clustering
- LDAP / AD / SAML — not needed for a personal home server
- Terraform provider — ecosystem tooling, not core functionality
- SPICE console — VNC already works; SPICE adds complexity for marginal gain
- OVA/OVF import — nice to have but not blocking
- Load balancing across multiple backends (future Phase 5 extension)
- WAF / ModSecurity integration (future Phase 5 extension)
- Multi-node proxy clustering (requires clustering feature)

---

## Next Immediate Action

**Phase 1 is complete.** All four operational foundation tasks are done:

**1.1 Task / Job Tracking** — The task tracking infrastructure is now
operational with full API, UI, and integration into backup/snapshot/clone operations.

**1.2 VM Update (Fix the Stub)** — VMs can now be modified after creation:

- Live updates (no restart): Name, Description, StartOnBoot
- Offline updates (VM must be stopped): CPU sockets/cores, Memory, Guest Agent, Nested Virtualization

**1.3 Disk Management (Add / Remove / Resize)** — The VM detail page now has
a "Disks" tab with full disk management:

- View attached disks with target device, path, size, bus, format, boot order
- Attach existing unassigned disks or create new disks
- Resize disks with validation
- Detach disks with option to delete disk file
- Root disk (first disk) is protected from detachment

**1.4 VM Clone** — Full VM cloning is now available:

- Clone button in VM detail page action bar
- Full clones (independent copies) and linked clones (backing file references)
- Task progress tracking during disk copy
- Automatic UUID and MAC address regeneration
- NVRAM file copying for UEFI VMs
- Optional start after clone

**Phase 2.1 Alerting System is complete:**

- Alert rules with threshold-based evaluation (storage, VM, backup, node, CPU, memory)
- Notification channels: Email (SMTP with TLS) and Webhook (HTTP POST)
- Background evaluation loop running every 60 seconds
- Alert deduplication to prevent notification spam
- Full UI with three tabs: Alerts, Rules, Channels
- Acknowledge and resolve alerts from UI

**Next up: 2.3 TPM 2.0 + Secure Boot** — Add TPM 2.0 and Secure Boot support
for Windows 11 compatibility. This involves adding XML generation for TPM
devices and OVMF secure boot firmware selection.

---

### Phase 5 Available for Future Work

**Phase 5 — Reverse Proxy & Uptime Monitoring** has been added as a low-priority
phase for users who want an integrated alternative to Nginx Proxy Manager. This
phase provides:

- **Reverse Proxy Core (5.1)** — Domain-based routing for VMs/containers with
  automatic Let's Encrypt SSL (HTTP-01 and DNS-01 challenges)
- **VM/Container Integration (5.2)** — Proxy configuration as part of resource
  settings, with auto-discovery mode
- **Uptime Monitoring (5.3)** — Health checks with historical data and alert
  integration

This phase is intended for users who want a unified management experience rather
than running a separate reverse proxy tool.

---

## Phase 6 — Configuration Cleanup

---

### 6.1 Config Audit — Move Hardcoded Paths to Config

**Status:** ⏳ **PENDING**

**Why.** The codebase has numerous hardcoded paths scattered throughout that should
be user-configurable. Different distributions and setups place files in different
locations, and users should be able to override defaults without code changes.

**Current hardcoded paths that should be configurable:**

| Path | Location | Current Behavior | Should Be Configurable |
|------|----------|------------------|----------------------|
| `/usr/share/OVMF/OVMF_VARS.secboot.fd` | `vm.go:221` | NVRAM template for secure boot | ✅ Yes — `vm_defaults.ovmf_vars_path` |
| `/var/lib/lxc` | `container.go:33` | Default LXC root directory | ✅ Yes — `containers.root_dir` |
| `/usr/lib/libvirt/libvirt_lxc` | `container.go:627` | LXC emulator path | ✅ Yes — `containers.emulator_path` |
| `/usr/lib/qemu/qemu-bridge-helper` | `linux.go:224-226` | Bridge helper paths | ✅ Yes — `libvirt.bridge_helper_paths` |
| `/usr/bin/qemu-system-*` | `linux.go:187` | QEMU emulator paths | ⚠️ Partially — already in `vm_defaults.emulator_paths` |
| `/opt/homebrew/bin/qemu-system-*` | `darwin.go:231-232` | macOS QEMU paths | ⚠️ Partially — already in `vm_defaults.emulator_paths` |
| `/usr/share/OVMF/OVMF_CODE.fd` | `linux.go:170-172` | OVMF firmware paths | ⚠️ Partially — already in `vm_defaults.ovmf_path` |
| `/opt/homebrew/share/qemu/edk2-*` | `darwin.go:212-218` | macOS OVMF paths | ⚠️ Partially — already in `vm_defaults.ovmf_path` |

**Deliverable.** All hardcoded paths moved to config with sensible auto-detected defaults.

**Files to modify:**

| File | Change |
|------|--------|
| `apps/api/internal/config/config.go` | Add new config fields for container paths, bridge helper paths, OVMF vars path |
| `apps/api/pkg/sysinfo/linux.go` | Remove hardcoded paths, use config-provided paths or fallback to auto-detect |
| `apps/api/pkg/sysinfo/darwin.go` | Same as linux.go |
| `apps/api/internal/repository/libvirt/vm.go` | Use config for NVRAM template path |
| `apps/api/internal/repository/libvirt/container.go` | Use config for root_dir and emulator path |
| `apps/api/config.example.yaml` | Document all new configurable paths |

**Implementation notes:**

- Keep auto-detection as fallback when config value is empty
- Add `ovmf_vars_path` for NVRAM template (secure boot)
- Add `containers.root_dir` for LXC container storage
- Add `containers.emulator_path` for libvirt_lxc path
- Add `libvirt.bridge_helper_paths` as array of paths to check
- Consider adding `libvirt.uri` for non-standard libvirt setups

**Complexity:** Low-Medium. Mostly mechanical refactoring to thread config through
to repository constructors and replace string literals with config lookups.

---

## Phase 7 — API Code Quality & Security

Critical technical debt and security improvements identified during code review.
These items address security vulnerabilities, architectural issues, and maintainability
concerns in the Go API backend.

---

### 7.1 Remove Insecure JWT Defaults

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.2 Refactor Monolithic main.go

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.3 Add Input Validation Layer

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.4 Consistent Error Handling Policy

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.5 Decouple from libvirt (Interface Extraction)

**Status:** ⏳ **PENDING**

Pending.

---

### 7.6 Configure SQLite Connection Pool

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.7 Add Global Rate Limiting

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.8 Add Context Propagation

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.9 API Versioning Strategy

**Status:** ✅ **COMPLETED**

Completed.

---

### 7.10 Establish Naming Conventions

**Status:** ✅ **COMPLETED**

Completed.

---

## Summary — Phase 7 Priority

### Completed Items (9/10) — PHASE 7 COMPLETE! 🎉

| # | Item | Status | Security | Effort | Impact |
|---|------|--------|----------|--------|--------|
| 7.1 | Remove Insecure JWT Defaults | ✅ | **HIGH** | Low | **HIGH** |
| 7.2 | Refactor Monolithic main.go | ✅ | LOW | Medium-High | MEDIUM |
| 7.3 | Add Input Validation Layer | ✅ | MEDIUM | Medium-High | **HIGH** |
| 7.4 | Consistent Error Handling Policy | ✅ | LOW | Medium-High | MEDIUM |
| 7.6 | Configure SQLite Connection Pool | ✅ | LOW | Low | LOW |
| 7.7 | Add Global Rate Limiting | ✅ | MEDIUM | Low | MEDIUM |
| 7.8 | Add Context Propagation | ✅ | LOW | Medium | MEDIUM |
| 7.9 | API Versioning Strategy | ✅ | LOW | Low | LOW |
| 7.10 | Establish Naming Conventions | ✅ | LOW | Low | LOW |

### Remaining Items (1/10)

| Priority | Item | Security | Effort | Impact | Notes |
|----------|------|----------|--------|--------|-------|
| 📝 **LOW** | 7.5 Decouple from libvirt | LOW | High | MEDIUM | Should be done incrementally as part of feature work |

**Phase 7 is effectively complete.** Item 7.5 (decouple from libvirt) is a large architectural refactoring that should be done incrementally as new features are added, not as a standalone big-bang refactoring.

---

## Phase 7 Summary

**Total commits:** 10
**Total lines added:** ~4,800
**Total lines removed:** ~500
**Net change:** +4,300 lines

### Key Achievements

| Category | Improvement |
|----------|-------------|
| **Security** | JWT secret validation, input validation, rate limiting, path traversal prevention |
| **Reliability** | Context propagation, error handling framework, SQLite connection pooling |
| **Maintainability** | Modular main.go, style guide, API versioning docs, error handling policy |
| **Performance** | SQLite WAL mode, connection pooling, serialized writes |
| **Documentation** | 5 comprehensive guides (versioning, style, error handling, auth, validation) |

### Files Created

| File | Purpose |
|------|---------|
| `internal/validator/*.go` | Input validation framework (4 files, 1000+ lines) |
| `internal/errors/*.go` | Error handling framework (2 files, 500+ lines) |
| `cmd/server/init_*.go` | Modular initialization (3 files, 550+ lines) |
| `API_VERSIONING.md` | API versioning policy |
| `STYLE_GUIDE.md` | Go coding standards |
| `ERROR_HANDLING.md` | Error handling guidelines |

### Security Improvements

1. **JWT Secret Validation** — Fails fast if not configured, minimum 16 chars
2. **Input Validation** — Comprehensive validation for all API inputs
3. **Rate Limiting** — Global 100 req/s limit, auth endpoints 5 req/s
4. **Path Traversal Prevention** — Blocks `..` in file paths
5. **SQL Injection Prevention** — Validated inputs before database operations

### Code Quality Improvements

1. **Modular Architecture** — main.go split into focused initialization modules
2. **Error Handling** — Typed errors with stack traces and context
3. **Context Propagation** — Graceful shutdown, cancellation support
4. **Documentation** — Comprehensive guides for future development

---

## Code Review Fixes (March 2026)

These items were identified during a comprehensive code review of `apps/api`.
They address security, reliability, and operational gaps.

**Files created:**

| File | Change |
|------|--------|
| `apps/api/STYLE_GUIDE.md` | New — Comprehensive naming and style conventions document |

**Key conventions documented:**

### Constructor Naming

| Type | Pattern | Example |
|------|---------|---------|
| Services | `New<Service>Service` | `NewVMService`, `NewAuthService` |
| Repositories | `New<Repo>Repository` | `NewUserRepository`, `NewVMRepository` |
| Handlers | `New<Handler>Handler` | `NewHealthHandler`, `NewMetricsHandler` |
| Middleware | `New<Middleware>` | `NewAuthInterceptor`, `NewRateLimiter` |
| Utilities | `New<Type>` | `NewJWT`, `NewPassword`, `NewCollector` |

### Variable Naming

| Type | Pattern | Example |
|------|---------|---------|
| Services | `<type>Svc` or `<type>Service` | `vmSvc`, `authService` |
| Repositories | `<type>Repo` | `userRepo`, `vmRepo` |
| Handlers | `<type>Handler` | `healthHandler`, `metricsHandler` |
| Context | Always `ctx` | `func Foo(ctx context.Context)` |
| Errors | Always `err` | `if err != nil` |

### Error Naming

| Type | Pattern | Example |
|------|---------|---------|
| Error variables | `Err<Resource><Condition>` | `ErrVMNotFound`, `ErrAuthFailed` |
| Error types | `<Condition>Error` | `ValidationError`, `NotFoundError` |

### Testing

| Type | Pattern | Example |
|------|---------|---------|
| Test files | `<file>_test.go` | `vm_test.go`, `auth_test.go` |
| Test functions | `Test<Service><Method><Scenario>` | `TestVMServiceCreateVM_ValidRequest` |

**Complexity:** Low. Documentation only, to be enforced gradually through code review.

---

## Summary — Phase 7 Priority

### Completed Items (9/10) — PHASE 7 COMPLETE! 🎉

| # | Item | Status | Security | Effort | Impact | Commit |
|---|------|--------|----------|--------|--------|--------|
| 7.1 | Remove Insecure JWT Defaults | ✅ | **HIGH** | Low | **HIGH** | `4cb27e2` |
| 7.2 | Refactor Monolithic main.go | ✅ | LOW | Medium-High | MEDIUM | `2046b26` |
| 7.3 | Add Input Validation Layer | ✅ | MEDIUM | Medium-High | **HIGH** | `457eb5e` |
| 7.4 | Consistent Error Handling Policy | ✅ | LOW | Medium-High | MEDIUM | `9a22c40` |
| 7.6 | Configure SQLite Connection Pool | ✅ | LOW | Low | LOW | `f16fc87` |
| 7.7 | Add Global Rate Limiting | ✅ | MEDIUM | Low | MEDIUM | `6a3c10e` |
| 7.8 | Add Context Propagation | ✅ | LOW | Medium | MEDIUM | `999ce8a` |
| 7.9 | API Versioning Strategy | ✅ | LOW | Low | LOW | `04841e3` |
| 7.10 | Establish Naming Conventions | ✅ | LOW | Low | LOW | `e9e1b55` |

### Remaining Items (1/10)

| Priority | Item | Security | Effort | Impact | Notes |
|----------|------|----------|--------|--------|-------|
| 📝 **LOW** | 7.5 Decouple from libvirt | LOW | High | MEDIUM | Should be done incrementally as part of feature work |

**Phase 7 is effectively complete.** Item 7.5 (decouple from libvirt) is a large architectural refactoring that should be done incrementally as new features are added, not as a standalone big-bang refactoring.

---

## Phase 7 Summary

**Total commits:** 10
**Total lines added:** ~4,800
**Total lines removed:** ~500
**Net change:** +4,300 lines

### Key Achievements

| Category | Improvement |
|----------|-------------|
| **Security** | JWT secret validation, input validation, rate limiting, path traversal prevention |
| **Reliability** | Context propagation, error handling framework, SQLite connection pooling |
| **Maintainability** | Modular main.go, style guide, API versioning docs, error handling policy |
| **Performance** | SQLite WAL mode, connection pooling, serialized writes |
| **Documentation** | 5 comprehensive guides (versioning, style, error handling, auth, validation) |

### Files Created

| File | Purpose |
|------|---------|
| `internal/validator/*.go` | Input validation framework (4 files, 1000+ lines) |
| `internal/errors/*.go` | Error handling framework (2 files, 500+ lines) |
| `cmd/server/init_*.go` | Modular initialization (3 files, 550+ lines) |
| `API_VERSIONING.md` | API versioning policy |
| `STYLE_GUIDE.md` | Go coding standards |
| `ERROR_HANDLING.md` | Error handling guidelines |

### Security Improvements

1. **JWT Secret Validation** — Fails fast if not configured, minimum 16 chars
2. **Input Validation** — Comprehensive validation for all API inputs
3. **Rate Limiting** — Global 100 req/s limit, auth endpoints 5 req/s
4. **Path Traversal Prevention** — Blocks `..` in file paths
5. **SQL Injection Prevention** — Validated inputs before database operations

### Code Quality Improvements

1. **Modular Architecture** — main.go split into focused initialization modules
2. **Error Handling** — Typed errors with stack traces and context
3. **Context Propagation** — Graceful shutdown, cancellation support
4. **Documentation** — Comprehensive guides for future development

---

## Code Review Fixes (March 2026)

These items were identified during a comprehensive code review of `apps/api`.
They address security, reliability, and operational gaps.

### Critical Priority

#### 1. Add TLS Configuration

**Status:** ⬜ **PENDING**

**Issue:** Server listens on plain HTTP. JWT secrets and all authentication tokens transmitted over wire without encryption.

**Fix:**
- Add TLS support in production mode
- Generate self-signed certificates for development
- Support ACME/Let's Encrypt for production
- Redirect HTTP to HTTPS

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/config/config.go` | Add `TLS` config: cert path, key path, ACME settings |
| `apps/api/cmd/server/main.go` | Add `server.StartTLS()` for production mode |
| `apps/api/pkg/tls/cert_manager.go` | New — certificate management with ACME support |
| `apps/api/pkg/tls/self_signed.go` | New — self-signed cert generation for dev |

**Complexity:** Medium. ACME integration requires DNS challenge support.

---

#### 2. JWT Secret Runtime Validation

**Status:** ⬜ **PENDING**

**Issue:** JWT secret validated only at startup via `validateJWTSecret()`. No runtime validation if config changes or secret becomes invalid.

**Fix:**
- Add health check endpoint that validates JWT configuration
- Add runtime secret rotation support
- Log warning if secret is weak

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/handler/health.go` | Add JWT validation check to `/health/ready` |
| `apps/api/internal/service/auth.go` | Add `ValidateJWTSecret()` method |
| `apps/api/pkg/auth/jwt.go` | Add secret strength validation |

**Complexity:** Low.

---

#### 3. Request Size Limits on RPC

**Issue:** Rate limiting exists but no request size limits. Large payloads could exhaust memory.

**Fix:**
- Add `connect.WithReadMaxBytes()` to all Connect RPC handlers
- Set reasonable defaults (e.g., 10MB for most endpoints, 1GB for upload)
- Return `ResourceExhausted` error for oversized requests

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/router/router.go` | Add `connect.WithReadMaxBytes()` to all handlers |
| `apps/api/internal/config/config.go` | Add `MaxRequestBytes` config |

**Complexity:** Low.

---

### High Priority

#### 4. Implement Role-Based Access Control (RBAC)

**Status:** ⬜ **PENDING**

**Issue:** `RequireRole()` middleware exists but is not used. All authenticated users can access all endpoints.

**Fix:**
- Add role requirements to sensitive endpoints:
  - Admin-only: user management, API key management, audit logs, system settings
  - Operator: VM start/stop, disk operations, network changes
  - Viewer: read-only access to metrics, logs, VM status
- Update `router.go` to apply role middleware per service

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/router/router.go` | Apply `RequireRole()` to each service handler |
| `apps/api/internal/middleware/auth.go` | Extend `RequireRole()` to support multiple roles |
| `packages/proto/**/*.proto` | Add role requirements to RPC documentation |

**Role matrix:**

| Endpoint | Viewer | Operator | Admin |
|----------|--------|----------|-------|
| List VMs | ✅ | ✅ | ✅ |
| Start/Stop VM | ❌ | ✅ | ✅ |
| Delete VM | ❌ | ❌ | ✅ |
| User Management | ❌ | ❌ | ✅ |
| Audit Logs | ❌ | ❌ | ✅ |

**Complexity:** Medium. Requires careful audit of all endpoints.

---

#### 5. SQL Injection Prevention — Input Validation

**Status:** ⬜ **PENDING**

**Issue:** SQLite queries use parameterized queries (good) but no validation on input types (e.g., `id` parameters).

**Fix:**
- Add input validation layer before repository calls
- Validate UUIDs, integers, and strings
- Return `InvalidArgument` error for malformed inputs

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/validator/validator.go` | Add `ValidateID()`, `ValidateUUID()`, `ValidateIntID()` |
| `apps/api/internal/handler/*.go` | Add validation before service calls |

**Complexity:** Low.

---

#### 6. Circuit Breaker for Libvirt Calls

**Status:** ⬜ **PENDING**

**Issue:** Libvirt calls can hang indefinitely. No timeout or circuit breaker pattern.

**Fix:**
- Add context timeouts to all libvirt calls (default 30s)
- Implement circuit breaker: open after 3 consecutive failures
- Return `Unavailable` error when circuit is open

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/pkg/libvirtx/client.go` | Add timeout wrapper to all libvirt calls |
| `apps/api/internal/repository/libvirt/vm.go` | Add context with timeout |
| `apps/api/pkg/circuit/circuit.go` | New — circuit breaker implementation |

**Complexity:** Medium.

---

### Medium Priority

#### 7. Rate Limiter Memory Optimization

**Status:** ⬜ **PENDING**

**Issue:** Rate limiter cleanup runs every 10 minutes. High-traffic scenarios could accumulate entries and cause memory growth.

**Fix:**
- Reduce TTL from 10 min to 5 min
- Add max entries limit (e.g., 10,000 IPs)
- Evict oldest entries when limit reached

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/middleware/ratelimit.go` | Add `maxEntries` config, LRU eviction |

**Complexity:** Low.

---

#### 8. Request ID Tracing

**Status:** ⬜ **PENDING**

**Issue:** `middleware.RequestID` generates IDs but they're not propagated to logs.

**Fix:**
- Add request ID to all slog calls via context
- Include request ID in error responses
- Add request ID to audit log entries

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/middleware/logging.go` | Extract request ID and add to context |
| `apps/api/internal/slog/slog.go` | New — context-aware logger with request ID |
| `apps/api/internal/handler/*.go` | Use context logger instead of global |

**Complexity:** Low.

---

#### 9. Platform-Agnostic Path Handling

**Status:** ⬜ **PENDING**

**Issue:** Hardcoded paths like `/usr/share/OVMF/OVMF_CODE.secboot.fd` won't work on non-Linux systems.

**Fix:**
- Use `sysinfo` package consistently for all paths
- Add Windows/macOS firmware paths where applicable
- Validate paths exist before use

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/pkg/sysinfo/sysinfo.go` | Add `FirmwarePaths()` for all platforms |
| `apps/api/internal/config/config.go` | Remove hardcoded paths, use sysinfo |

**Complexity:** Low.

---

#### 10. Prometheus Metrics Export

**Status:** ⬜ **PENDING**

**Issue:** SQLite stores metrics but no export endpoint for external monitoring.

**Fix:**
- Add `/metrics` endpoint with Prometheus format
- Export: VM status, CPU/memory usage, disk usage, backup status
- Add alerting rules for Prometheus

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/pkg/metrics/prometheus.go` | New — Prometheus metrics exporter |
| `apps/api/internal/handler/metrics.go` | Add `/metrics` HTTP handler |
| `apps/api/internal/router/router.go` | Wire metrics endpoint |

**Complexity:** Medium.

---

### Low Priority

#### 11. Standardize Error Messages

**Status:** ⬜ **PENDING**

**Issue:** Inconsistent error message format: "VM not found" vs "vm not found".

**Fix:**
- Standardize on sentence case: "VM not found"
- Use resource type + ID format: "VM 100: not found"
- Update all error creation sites

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/errors/errors.go` | Add error message format guidelines |
| `apps/api/internal/service/*.go` | Update error messages |
| `apps/api/internal/handler/*.go` | Update error messages |

**Complexity:** Low (mechanical search/replace).

---

#### 12. Generate OpenAPI Documentation

**Status:** ⬜ **PENDING**

**Issue:** Proto files exist but no OpenAPI/Swagger documentation for REST clients.

**Fix:**
- Add `buf generate` target for OpenAPI
- Serve OpenAPI spec at `/api/openapi.json`
- Add Swagger UI at `/api/docs`

**Files to create/modify:**

| File | Change |
|------|--------|
| `packages/proto/buf.gen.openapi.yaml` | New — OpenAPI generation config |
| `apps/api/internal/router/router.go` | Serve OpenAPI spec and Swagger UI |
| `Makefile` | Add `make openapi` target |

**Complexity:** Low.

---

#### 13. Enhanced Health Checks

**Status:** ⬜ **PENDING**

**Issue:** `/health/ready` checks SQLite but not Libvirt connectivity.

**Fix:**
- Add Libvirt connectivity check to health endpoint
- Add dependency-specific health checks:
  - `/health/db` — SQLite only
  - `/health/libvirt` — Libvirt only
  - `/health/docker` — Docker only (for containers)
- Return detailed status for each dependency

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/handler/health.go` | Split into granular health checks |
| `apps/api/internal/router/router.go` | Add `/health/*` routes |

**Complexity:** Low.

---

## Summary

| Priority | Count | Items |
|----------|-------|-------|
| Critical | 3 | TLS, JWT runtime validation, request size limits |
| High | 3 | RBAC, SQL injection prevention, circuit breaker |
| Medium | 4 | Rate limiter optimization, request ID tracing, platform paths, Prometheus |
| Low | 3 | Error message standardization, OpenAPI docs, health checks |

**Total:** 13 items

