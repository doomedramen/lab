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

**Bug:** VM performance graphs showed no data because the metrics collector was
storing VM metrics with the internal database ID (`vm.ID`) instead of the QEMU
VM ID (`vm.VMID` like 100, 101). The frontend queries used `vm.vmid.toString()`,
so no metrics were found.

**Fix:** Changed collector to use `vm.VMID` converted to string for `resource_id`.

**Files modified:**

| File                                        | Change                                                    |
| ------------------------------------------- | --------------------------------------------------------- |
| `apps/api/internal/service/collector.go`    | Use `vm.VMID` instead of `vm.ID` for metrics resource_id  |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx` | Added metrics fetching and data transformation for charts |

---

### False "VM Started" Logs on API Restart

**Status:** ✅ **FIXED**

**Bug:** The collector logged "VM started" every time the API server restarted,
even for VMs that were already running. This happened because the in-memory
state tracker (`vmStatePrev`) was empty on startup.

**Fix:** Added `initializeVMStates()` to load current VM states before starting
collection loop.

**Files modified:**

| File                                     | Change                                                  |
| ---------------------------------------- | ------------------------------------------------------- |
| `apps/api/internal/service/collector.go` | Initialize `vmStatePrev` from actual VM states on start |

---

### Inconsistent VM Log Messages

**Status:** ✅ **FIXED**

**Bug:** Two different wordings for the same VM state:
- "VM stopped" (from collector state transitions)
- "VM is stopped" (from synthetic logs)

**Fix:** Standardized synthetic logs to match collector format: "VM {name} started/stopped/paused".

**Files modified:**

| File                                  | Change                                           |
| ------------------------------------- | ------------------------------------------------ |
| `apps/api/internal/service/vm.go`     | Changed synthetic log messages to match format   |
| `apps/api/internal/service/collector.go` | Added cleanup for old "VM is stopped" logs    |

---

### VM Log Viewer — Incorrect Timestamp Order

**Status:** ✅ **FIXED**

**Bug:** VM logs appeared out of chronological order, making it look like VMs
were boot looping. Old spam logs with incorrect timestamps were interleaved
with newer logs.

**Fix:** Added client-side sorting by timestamp (newest first) in the log viewer.

**Files modified:**

| File                                   | Change                                    |
| -------------------------------------- | ----------------------------------------- |
| `apps/web/components/log-viewer.tsx`   | Sort logs by timestamp before filtering   |

---

### Login Redirect Ignores `from` Query Parameter

**Status:** ✅ **FIXED**

**Bug:** Login page always redirected to `/vms` after login, ignoring the
`from` query parameter (e.g., `/login?from=%2Fvms%2F100`).

**Fix:** Read `from` query parameter and redirect to original page. Default
changed from `/vms` to `/` (dashboard).

**Files modified:**

| File                            | Change                                     |
| ------------------------------- | ------------------------------------------ |
| `apps/web/app/(auth)/login/page.tsx` | Use `useSearchParams` to read `from` param |
| `apps/web/e2e/auth.spec.ts`     | Updated tests to expect `/` as default     |

---

### QEMU Guest Agent Segfault

**Status:** ✅ **FIXED**

**Bug:** Calling `GetVMGuestNetworkInterfaces` on VMs without a working QEMU
guest agent caused a segmentation fault in libvirt's C library.

**Fix:** Added `hasGuestAgent()` check using libvirt's safer `GetGuestInfo` API
before attempting any `QemuAgentCommand` calls.

**Files modified:**

| File                                              | Change                                               |
| ------------------------------------------------- | ---------------------------------------------------- |
| `apps/api/internal/repository/libvirt/guest_agent.go` | Add `hasGuestAgent()` check before agent commands |

---

### All `fetch()` Usage Removed — Migrated to ConnectRPC

**Status:** ✅ **FIXED**

**Bug:** The metrics API used REST/fetch instead of ConnectRPC like the rest
of the application, leading to inconsistent auth error handling.

**Fix:** 
- Added `QueryMetrics` RPC to ClusterService proto
- Migrated `useMetrics` hook to use ConnectRPC `clusterClient.queryMetrics()`
- Removed all REST/fetch code from `lib/api/client.ts`

**Files modified:**

| File                                      | Change                                      |
| ----------------------------------------- | ------------------------------------------- |
| `packages/proto/lab/v1/cluster.proto`     | Added `QueryMetrics` RPC                    |
| `apps/api/internal/connectsvc/cluster.go` | Implemented `QueryMetrics` handler          |
| `apps/api/internal/service/cluster.go`    | Added `QueryMetrics` method                 |
| `apps/web/lib/api/queries/metrics.ts`     | Migrated to ConnectRPC                      |
| `apps/web/lib/api/client.ts`              | Removed all REST/fetch code                 |
| `apps/web/lib/utils/format-time.ts`       | Added `formatChartTime` utility             |

---

## Phase 1 — Operational Foundation

These gaps make the platform unreliable or unusable in day-to-day operation.
They should be addressed before adding any new features.

---

### 1.1 Task / Job Tracking

**Status:** ✅ **COMPLETED**

**Why first.** Every async operation (backup, restore, snapshot, future clone)
already returns a `task_id` UUID, but nothing ever persists it. Users have no
way to know whether the backup that ran at 2 AM succeeded or failed. This is
dangerous for a home server you're relying on.

**Deliverable.** A persistent task table, a service that async workers write
progress into, a `GetTask` / `ListTasks` / `CancelTask` API, and a `/tasks`
UI page.

**Files created / modified:**

| File                                           | Change                                                                                                                                                  | Status |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/pkg/sqlite/migrations/008_tasks.sql` | New — tasks table: id, type, status, progress, message, resource_type, resource_id, created_at, updated_at, completed_at                                | ✅     |
| `apps/api/internal/model/task.go`              | New — `Task`, `TaskStatus` (pending/running/completed/failed/cancelled), `TaskType` enums                                                               | ✅     |
| `apps/api/internal/repository/sqlite/task.go`  | New — `TaskRepository`: Create, Update, GetByID, List, ListByResource, DeleteOld                                                                        | ✅     |
| `apps/api/internal/service/task.go`            | New — `TaskService`: wraps repo, provides `Start(ctx, type, resourceID) *Task`, `Progress(id, pct, msg)`, `Complete(id)`, `Fail(id, err)`, `Cancel(id)` | ✅     |
| `packages/proto/lab/v1/task.proto`             | New — `ListTasksRequest/Response`, `GetTaskRequest/Response`, `CancelTaskRequest/Response`                                                              | ✅     |
| `apps/api/internal/handler/task.go`            | New — Connect RPC handler implementation                                                                                                                | ✅     |
| `apps/api/internal/router/router.go`           | Wire task handler                                                                                                                                       | ✅     |
| `apps/api/cmd/server/main.go`                  | Wire TaskService into dependency injection                                                                                                              | ✅     |
| `apps/api/internal/service/backup.go`          | Use TaskService instead of bare UUID; update progress during backup run                                                                                 | ✅     |
| `apps/api/internal/service/snapshot.go`        | Same                                                                                                                                                    | ✅     |
| `apps/web/app/(dashboard)/tasks/page.tsx`      | New — task list with status, type, resource link, progress bar, cancel button                                                                           | ✅     |
| `apps/web/lib/api/queries/tasks.ts`            | New — React Query hooks for tasks                                                                                                                       | ✅     |
| `apps/web/lib/api/client.ts`                   | Add TaskService client                                                                                                                                  | ✅     |

**Integration pattern for async services:**

```go
task := taskSvc.Start(ctx, model.TaskTypeBackup, fmt.Sprintf("vm/%d", vmid))
defer func() {
    if err != nil { taskSvc.Fail(task.ID, err) } else { taskSvc.Complete(task.ID) }
}()
taskSvc.Progress(task.ID, 10, "creating disk snapshot")
// ... do work ...
taskSvc.Progress(task.ID, 90, "writing backup file")
```

**Complexity:** Medium. SQLite schema + service wrapper is straightforward.
The integration into existing services is mechanical but requires touching
backup.go and snapshot.go carefully.

**Implementation notes:**

- TaskService is initialized before BackupService and SnapshotService in main.go (dependency order)
- Fallback to bare UUID task tracking if TaskService is unavailable
- CompletedAt field is optional (only set for terminal states)
- Frontend uses 2-second polling for active tasks, 1-second for individual task detail
- UI includes cancel button for pending/running tasks with confirmation dialog

---

### 1.2 VM Update (Fix the Stub)

**Status:** ✅ **COMPLETED**

**Why second.** `VMRepository.Update` returns `"VM update is not yet implemented"`.
You cannot change a VM's CPU, memory, name, description, or tags after creation.
The API and frontend form already exist — the only missing piece is the libvirt
implementation.

**Deliverable.** Working VM update for offline changes (requires stopped VM)
and a subset of live changes (description, tags, name stored in metadata).

**What each field requires:**

- `Name`, `Description`, `Tags` — stored in libvirt metadata (XML `<title>` and `<description>` tags) + re-define domain; no VM restart required
- `CPUCores`, `CPUSockets` — update XML, re-define domain; requires stopped VM
- `Memory` — update XML, re-define domain; requires stopped VM
- `StartOnBoot` — libvirt autostart flag via `virDomainSetAutostart`; can be changed while running
- `Agent` — add/remove virtio channel from XML, re-define domain; requires stopped VM
- `NestedVirt` — add/remove CPU feature from XML, re-define domain; requires stopped VM

**Files created / modified:**

| File                                         | Change                                                                                                                            | Status |
| -------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/repository/libvirt/vm.go` | Implement `Update`: read current XML via `GetXMLDesc`, patch changed fields, call `DomainDefineXML`; handle live vs offline cases | ✅     |
| `apps/api/internal/model/vm.go`              | Extend `VMUpdateRequest` with pointer fields for `StartOnBoot`, `Agent`, `NestedVirt`, `CPUSockets`, `CPUCores`, `Memory`         | ✅     |
| `apps/api/internal/connectsvc/vm.go`         | Update `UpdateVM` handler to pass all fields                                                                                      | ✅     |
| `packages/proto/lab/v1/vm.proto`             | Extend `UpdateVMRequest` with new fields                                                                                          | ✅     |
| `apps/web/lib/api/mutations/vms.ts`          | Add `useUpdateVM` mutation hook                                                                                                   | ✅     |

**Implementation details:**

- Live updates (no restart): `Name`, `Description`, `StartOnBoot` (autostart)
- Offline updates (VM must be stopped): `CPUSockets`, `CPUCores`, `Memory`, `Agent`, `NestedVirt`
- Error message clearly indicates which fields require VM to be stopped
- Uses libvirtxml structs for XML parsing and marshaling
- Guest agent channel uses `org.qemu.guest_agent.0` target name
- Nested virtualization uses `vmx` (Intel) or `svm` (AMD) CPU features

**Complexity:** Medium. libvirt XML patching with `libvirtxml` structs is
well-trodden. The tricky part is deciding gracefully what's live-changeable
vs what requires a stopped VM and surfacing that to the user.

---

### 1.3 Disk Management (Add / Remove / Resize)

**Status:** ✅ **COMPLETED**

**Why third.** `StorageService` already has `CreateStorageDisk`, `ResizeStorageDisk`,
`DeleteStorageDisk`, and `MoveStorageDisk`, but none are wired to VM attachment
via libvirt, and there is no disk management tab in the VM detail UI.

**Deliverable.** A "Disks" tab on the VM detail page where you can add a new
disk, resize an existing disk, detach and delete a disk.

**Files created / modified:**

| File                                           | Change                                                                                                                         | Status |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ | ------ |
| `apps/api/internal/repository/libvirt/disk.go` | New — `AttachDisk`, `DetachDisk`, `ListVMDisks`, `ResizeDiskImage`, `CreateDiskImage`, `GetDiskInfo`, `IsRootDisk` via libvirt | ✅     |
| `apps/api/internal/model/vm.go`                | Add `VMDisk` model: target, path, size_bytes, bus, format, readonly, boot_order                                                | ✅     |
| `apps/api/internal/repository/repository.go`   | Add `LibvirtDiskRepository` interface                                                                                          | ✅     |
| `apps/api/internal/service/storage.go`         | Add `AttachVMdisk`, `DetachVMdisk`, `ListVMDisks`, `ResizeVMdisk` delegating to libvirt disk repo                              | ✅     |
| `packages/proto/lab/v1/storage.proto`          | Add `ListVMDisks`, `AttachVMdisk`, `DetachVMdisk`, `ResizeVMdisk` RPCs and messages                                            | ✅     |
| `apps/api/internal/handler/storage.go`         | Wire new disk RPCs                                                                                                             | ✅     |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx` | Add Disks tab with table, attach/detach/resize dialogs                                                                         | ✅     |
| `apps/web/lib/api/queries/storage.ts`          | Add `useVMDisks` query hook                                                                                                    | ✅     |
| `apps/web/lib/api/mutations/storage.ts`        | Add `useVMDiskMutations` for attach/detach/resize                                                                              | ✅     |
| `apps/web/lib/api/enum-helpers.ts`             | Add `diskBusToString`, `diskFormatToString`, and fromString helpers                                                            | ✅     |

**Implementation notes:**

- Disk table shows target device, path, size, bus type, format, boot order
- Root disk (first disk, vda/sda) cannot be detached — button disabled
- Attach dialog supports two modes: select existing unassigned disk or create new
- Resize dialog validates new size >= current size
- Detach dialog shows warning for running VMs (hot-detach) and option to delete disk file
- Uses `virDomainAttachDeviceFlags` and `virDomainDetachDeviceFlags` for live attachment

**Multiple disks at VM creation** (bonus, low effort once disk.go exists):

Add `AdditionalDisks []DiskConfig` to `VMCreateRequest` and iterate in
`buildDomainXML`. The disk creation already works — it just needs to loop.

**Complexity:** Medium. `virDomainAttachDeviceFlags` with the correct XML
fragment is the core; listing and parsing current disks from XML is mechanical.
Hot-detach of the root disk must be blocked in the service layer.

---

### 1.4 VM Clone

**Status:** ✅ **COMPLETED**

**Why fourth.** Without clone, every new VM requires a full OS install from ISO.
Templates exist in the model but the "spin up from template" flow is: clone a
base VM. This is the single most-used operation in day-to-day Proxmox usage.

**Deliverable.** Full clone of a stopped VM (disk copy + new domain definition).
Linked clones are phase 2.

**Implementation approach:**

1. Get source VM's XML via `GetXMLDesc`
2. Copy each disk image: `qemu-img create -f qcow2 -b <src> -F qcow2 <dst>` for linked,
   or `qemu-img convert -O qcow2 <src> <dst>` for full clone
3. Patch XML: new domain name, new UUID, new MAC addresses, new disk paths
4. `DomainDefineXML` the patched XML
5. Write a task record so the user can watch progress

**Files created / modified:**

| File                                                           | Change                                                                                                                                                      | Status |
| -------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/repository/libvirt/vm.go`                   | Add `Clone(ctx, req *model.VMCloneRequest, progressFunc) (*model.VM, error)` with qemu-img disk copying, XML patching, MAC regeneration, cleanup on failure | ✅     |
| `apps/api/internal/model/vm.go`                                | Add `VMCloneRequest`: SourceVMID, Name, Full bool (linked vs full), TargetPool, Description, StartAfterClone                                                | ✅     |
| `apps/api/internal/repository/repository.go`                   | Add `Clone` method to `VMRepository` interface                                                                                                              | ✅     |
| `apps/api/internal/service/vm.go`                              | Add `Clone` method with task tracking, pass TaskService to VMService                                                                                        | ✅     |
| `packages/proto/lab/v1/vm.proto`                               | Add `CloneVMRequest/Response` with vm and task_id fields                                                                                                    | ✅     |
| `apps/api/internal/connectsvc/vm.go`                           | `CloneVM` handler with validation                                                                                                                           | ✅     |
| `apps/api/cmd/server/main.go`                                  | Pass taskSvc to NewVMService                                                                                                                                | ✅     |
| `packages/components/src/components/entity-action-buttons.tsx` | Add `onClone` prop and Clone button                                                                                                                         | ✅     |
| `apps/web/lib/api/mutations/vms.ts`                            | Add `cloneVM` mutation with task tracking                                                                                                                   | ✅     |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx`                 | Add Clone dialog with name, description, full/linked toggle, start-after-clone option                                                                       | ✅     |

**Implementation notes:**

- Clone operation runs asynchronously with task progress tracking
- Full clones use `qemu-img convert` for independent disk copies
- Linked clones use `qemu-img create -b` for backing file references
- New VM gets unique UUID and regenerated MAC addresses
- NVRAM files are copied for UEFI VMs
- Cleanup on failure removes any created disk files
- Clone button in EntityActionButtons component
- Clone dialog supports name, description, full/linked toggle, start-after-clone

**Complexity:** Medium. The disk copy is a shell-out to `qemu-img` (same
pattern as backup). XML patching is straightforward with `libvirtxml`. The
main risk is ensuring disk paths don't collide and MAC addresses are unique.

---

## Phase 2 — Home Server Essentials

Features that meaningfully differentiate a home server from a bare hypervisor.

---

### 2.1 Alerting System

**Status:** ✅ **COMPLETED**

**Why.** You need to know when things go wrong when you're not watching:
backup failure, disk at 90%, VM that should be running is stopped, node went
offline. Without alerts, you only discover problems when services are down.

**Deliverable.** Alert rule engine with threshold-based rules and two
notification channels: email (SMTP) and webhook (HTTP POST).

**Alert types implemented:**

- Storage pool usage > threshold % (80 / 90 / 95)
- VM stopped unexpectedly (was running, now stopped, not user-initiated)
- Backup job failed
- Node offline
- CPU usage > threshold % sustained for N minutes
- Memory usage > threshold % sustained for N minutes

**Files created / modified:**

| File                                            | Change                                                                                                                               | Status |
| ----------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ | ------ |
| `apps/api/pkg/sqlite/migrations/009_alerts.sql` | New — notification_channels, alert_rules, fired_alerts tables                                                                        | ✅     |
| `apps/api/internal/model/alert.go`              | New — `AlertRule`, `AlertRuleType`, `NotificationChannel`, `NotificationChannelType`, `AlertSeverity`, `AlertStatus`, `Alert` models | ✅     |
| `apps/api/internal/repository/sqlite/alert.go`  | New — CRUD for channels, rules, alerts; `HasOpenAlert` for deduplication                                                             | ✅     |
| `apps/api/internal/service/alert.go`            | New — `AlertService`: background evaluation loop, metric provider interfaces, rule evaluation for all alert types                    | ✅     |
| `apps/api/internal/service/notifier.go`         | New — `Notifier` interface; `EmailNotifier` (SMTP with TLS); `WebhookNotifier` (HTTP POST)                                           | ✅     |
| `apps/api/internal/service/storage.go`          | Add `ListStoragePoolsForAlerts` method for alert integration                                                                         | ✅     |
| `apps/api/internal/service/backup.go`           | Add `ListBackupsForAlerts` method for alert integration                                                                              | ✅     |
| `packages/proto/lab/v1/alert.proto`             | New — AlertService with CRUD RPCs for channels, rules, alerts                                                                        | ✅     |
| `apps/api/internal/connectsvc/alert.go`         | New — ConnectRPC handler implementation                                                                                              | ✅     |
| `apps/api/cmd/server/main.go`                   | Wire alert repository, service with metric providers; start/stop service                                                             | ✅     |
| `apps/api/internal/router/router.go`            | Wire alert handler                                                                                                                   | ✅     |
| `apps/web/lib/api/queries/alerts.ts`            | New — React Query hooks: useNotificationChannels, useAlertRules, useAlerts                                                           | ✅     |
| `apps/web/lib/api/queries/index.ts`             | Export alerts module                                                                                                                 | ✅     |
| `apps/web/app/(dashboard)/alerts/page.tsx`      | New — three-tab UI: Alerts, Rules, Channels with create/edit dialogs                                                                 | ✅     |

**Implementation notes:**

- Background goroutine evaluates rules every 60 seconds
- Metric providers wired: NodeRepository, VMRepository, StorageService, BackupService
- Alert deduplication prevents spam (checks for open alerts before firing)
- Email notifier supports SMTP with TLS (port 465) and STARTTLS (port 587)
- Webhook notifier sends JSON POST with alert details
- UI has three tabs: Alerts (fired alerts list), Rules (CRUD), Channels (CRUD)
- Alerts can be acknowledged or resolved from the UI

**Complexity:** Medium. Rule evaluation is simple threshold comparison against
existing metric data. The SMTP and webhook notifiers are straightforward Go.
The deduplication logic (open/closed alert state) needs care to avoid spam.

---

### 2.2 QEMU Guest Agent Integration

**Status:** ✅ **COMPLETED**

**Why.** The guest agent channel is already created in the domain XML when
`Agent=true`, but nothing queries it. The immediate benefit is IP address
discovery — without it, you don't know a DHCP VM's IP without checking your
router. A secondary benefit is filesystem freeze/thaw for consistent backups.

**Deliverable.** `GetVMNetworkInterfaces` API that returns IPs via agent;
VM detail page shows real IPs; backup service calls freeze/thaw when agent
is available.

**Files created / modified:**

| File                                                  | Change                                                                                                               | Status |
| ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/repository/libvirt/guest_agent.go` | New — `Ping`, `GetNetworkInterfaces`, `FreezeFilesystems`, `ThawFilesystems`, `GetPrimaryIP` via QEMU agent commands | ✅     |
| `apps/api/internal/model/vm.go`                       | Add `GuestNetworkInterface`, `GuestIPAddress`, `GuestAgentStatus` models                                             | ✅     |
| `apps/api/internal/repository/repository.go`          | Add `GuestAgentRepository` interface                                                                                 | ✅     |
| `apps/api/internal/service/vm.go`                     | Add `WithGuestAgentRepo`, `GetGuestNetworkInterfaces`, `GetGuestAgentStatus` methods                                 | ✅     |
| `apps/api/internal/service/backup.go`                 | Add `WithGuestAgentRepo`, `freezeFilesystems`, `thawFilesystems` for consistent backups                              | ✅     |
| `packages/proto/lab/v1/vm.proto`                      | Add `GetVMGuestNetworkInterfaces` RPC and messages                                                                   | ✅     |
| `apps/api/internal/connectsvc/vm.go`                  | Add `GetVMGuestNetworkInterfaces` handler                                                                            | ✅     |
| `apps/api/cmd/server/main.go`                         | Initialize and wire guest agent repository to VM and backup services                                                 | ✅     |
| `apps/web/lib/api/queries/vms.ts`                     | Add `useGuestNetworkInterfaces` hook                                                                                 | ✅     |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx`        | Display guest IPs in Network tab; show agent connection status badge                                                 | ✅     |

**Implementation notes:**

- Uses `virDomainQemuAgentCommand` to execute QMP commands on guest agent
- `guest-network-get-interfaces` returns all network interfaces with IP addresses
- `guest-fsfreeze-freeze` and `guest-fsfreeze-thaw` for consistent backups
- Backup service automatically attempts freeze before backup and thaw after
- Network tab shows agent connection status badge when agent is enabled
- Discovered IPs displayed in real-time from guest agent responses

**Complexity:** Low-Medium. The libvirt agent command interface is
well-documented. JSON parsing of the guest-network-get-interfaces response
is the main work.

---

### 2.3 TPM 2.0 + Secure Boot

**Status:** ✅ **COMPLETED**

**Why.** Windows 11 requires TPM 2.0 — it won't install without it. This is
a home server staple feature. The OVMF firmware is already implemented;
adding TPM is a small XML addition.

**Host prerequisite:** `swtpm` package installed on the hypervisor host.

**Deliverable.** `tpm: true` option on VM create/update that adds a software
TPM 2.0 device. `secureBoot: true` that selects the OVMF secure boot variant.

**Files created / modified:**

| File                                           | Change                                                                                                                                                                     | Status |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/model/vm.go`                | Add `TPM bool` and `SecureBoot bool` to `VMCreateRequest`, `VMUpdateRequest`, and `VM` model                                                                               | ✅     |
| `apps/api/internal/config/config.go`           | Add `OVMFSecureBootPath` config field and `GetOVMFSecureBootPathForArch` method                                                                                            | ✅     |
| `apps/api/internal/repository/libvirt/vm.go`   | In `buildDomainXML`: add TPM 2.0 XML (`tpm-crb` emulator) when `req.TPM=true`; select OVMF_CODE.secboot.fd when `req.SecureBoot=true`                                      | ✅     |
| `apps/api/internal/repository/libvirt/vm.go`   | Add `boolToStr` helper; add TPM/SecureBoot to offline-only change validation in `Update`                                                                                   | ✅     |
| `packages/proto/lab/v1/vm.proto`               | Add `tpm` and `secure_boot` fields to `VM`, `CreateVMRequest`, and `UpdateVMRequest`                                                                                       | ✅     |
| `apps/api/internal/connectsvc/vm.go`           | Map TPM and SecureBoot fields in proto/model conversion functions                                                                                                          | ✅     |
| `apps/web/components/create-vm-modal.tsx`      | Add TPM 2.0 and Secure Boot toggles in Advanced tab; disable for ARM architecture                                                                                          | ✅     |
| `apps/web/app/(dashboard)/vms/[vmid]/page.tsx` | Display TPM and Secure Boot status in VM details section                                                                                                                   | ✅     |

**Implementation notes:**

- TPM 2.0 uses `tpm-crb` model with software emulator backend
- Secure Boot selects OVMF_CODE.secboot.fd firmware with secure='yes' attribute
- NVRAM template uses OVMF_VARS.secboot.fd for Secure Boot VMs
- Both features disabled for ARM/aarch64 architecture (not supported)
- TPM requires OVMF BIOS; Secure Boot implicitly enables OVMF

**Complexity:** Low. Mostly XML generation. The main consideration is
detecting whether `swtpm` is available on the host at startup and surfacing
a warning if not.

---

### 2.4 PCI / GPU Passthrough

**Status:** ✅ **COMPLETED**

**Why.** Passing a GPU to a VM is one of the primary reasons people run a
home server with KVM. Gaming VMs, Plex transcoding, AI inference — all need
direct GPU access. This requires IOMMU groups to be configured on the host
(kernel parameters), but the software side is well-defined.

**Deliverable.** List available host PCI devices grouped by IOMMU group;
attach one or more PCI devices to a VM at creation or via the hardware tab.

**Files created / modified:**

| File                                           | Change                                                                                                                                                                     | Status |
| ---------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/model/vm.go`                | Add `PCIDevice` struct with address, vendor, product, IOMMU group; add `PCIDevices []PCIDevice` to `VMCreateRequest` and `VM`                                              | ✅     |
| `apps/api/internal/repository/libvirt/pci.go`  | New — `ListHostDevices`, `GetDevicesByIOMMUGroup`, `IsIOMMUAvailable`, `IsVFIOAvailable`, `AttachPCIDeviceToVM`, `DetachPCIDeviceFromVM`                                   | ✅     |
| `apps/api/internal/repository/repository.go`   | Add `PCIRepository` interface                                                                                                                                              | ✅     |
| `apps/api/internal/repository/libvirt/vm.go`   | Add `parsePCIAddress` helper; generate `<hostdev>` XML for each PCI device in `buildDomainXML`                                                                             | ✅     |
| `apps/api/internal/service/vm.go`              | Add `pciRepo` field, `WithPCIRepo`, `ListPCIDevices` method                                                                                                                | ✅     |
| `packages/proto/lab/v1/vm.proto`               | Add `PCIDevice`, `ListPCIDevicesRequest/Response`, `IOMMUGroup` messages; `pci_devices` field on `VM` and `pci_device_addresses` on `CreateVMRequest`                      | ✅     |
| `apps/api/internal/connectsvc/vm.go`           | Add `ListPCIDevices` handler                                                                                                                                               | ✅     |
| `apps/api/cmd/server/main.go`                  | Wire PCI repository to VM service                                                                                                                                          | ✅     |
| `apps/web/lib/api/queries/vms.ts`              | Add `usePCIDevices` hook for fetching PCI devices                                                                                                                         | ✅     |
| `apps/web/components/create-vm-modal.tsx`      | Add PCI Devices section in Advanced tab with IOMMU group display, status badges, and device selection                                                                      | ✅     |

**Implementation notes:**

- Uses `virConnectListAllNodeDevices` to enumerate PCI devices
- Reads `/sys/bus/pci/devices/<addr>/iommu_group` for IOMMU grouping
- Reads `/sys/bus/pci/devices/<addr>/driver` for current driver info
- Generates `<hostdev mode='subsystem' type='pci' managed='yes'>` XML
- `IsIOMMUAvailable()` checks for IOMMU groups or kernel cmdline params
- `IsVFIOAvailable()` checks for vfio-pci module

**Complexity:** Medium-High. The libvirt API side is clean. The complexity
is in the UX: users must pass the entire IOMMU group (GPU + audio function),
and the host must have `intel_iommu=on` or `amd_iommu=on` in kernel args.
The UI should detect and warn about misconfigured IOMMU groups.

---

## Phase 3 — Management & Hardening

---

### 3.1 Auth Rate Limiting

**Status:** ✅ **COMPLETED**

**Why this comes before host shell / other features.** The login endpoint
has no rate limiting. This is a security issue that takes 30 minutes to fix
and should not wait.

**Files created / modified:**

| File                                              | Change                                                                                                                                  | Status |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/middleware/ratelimit.go`       | New — token bucket rate limiter per IP using `golang.org/x/time/rate`; configurable burst and rate; returns 429 with Retry-After header | ✅     |
| `apps/api/internal/middleware/ratelimit_test.go`  | Tests for HTTP middleware, Connect interceptor, IP extraction, per-IP buckets                                                           | ✅     |
| `apps/api/internal/router/router.go`              | Apply rate limiter interceptor to AuthService handler (Login, Register, MFA endpoints)                                                  | ✅     |

**Implementation notes:**

- Uses `golang.org/x/time/rate` token bucket algorithm
- Per-IP rate limiting with separate buckets for each client
- Supports both Connect RPC (Interceptor) and plain HTTP (HTTPMiddleware)
- IP extraction from X-Real-IP and X-Forwarded-For headers
- Automatic cleanup of idle entries (10-minute TTL) to prevent memory leaks
- Returns `Retry-After` header on rate limit exceeded

**Complexity:** Very Low. One middleware file, two lines in the router.

---

### 3.2 Host Shell Access

**Status:** ✅ **COMPLETED**

**Why.** When something goes wrong on the host, you should be able to open
a terminal without a separate SSH client. The Docker container bash WebSocket
handler is an exact template for this.

**Deliverable.** A "Shell" tab on the host detail page that opens an
xterm.js terminal connected to a WebSocket that runs a PTY on the local host.

**Files created / modified:**

| File                                              | Change                                                                                                                                                    | Status |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/model/node.go`                 | Add `HostShellToken` struct for one-time token authentication                                                                                             | ✅     |
| `apps/api/internal/service/node.go`               | Add `GetHostShellToken` and `ValidateHostShellToken` methods with token cleanup                                                                           | ✅     |
| `apps/api/internal/handler/host_shell.go`         | New — WebSocket handler; spawns shell via `pty.Start`; relays stdin/stdout; auth via one-time token                                                       | ✅     |
| `apps/api/internal/router/router.go`              | Register `/ws/host-shell` route                                                                                                                           | ✅     |
| `packages/proto/lab/v1/node.proto`                | Add `GetHostShellToken` RPC and request/response messages                                                                                                 | ✅     |
| `apps/api/internal/connectsvc/node.go`            | Add `GetHostShellToken` handler                                                                                                                           | ✅     |
| `apps/web/lib/api/queries/nodes.ts`               | Add `useHostShellToken` mutation hook                                                                                                                     | ✅     |
| `apps/web/components/host-shell.tsx`              | New — xterm.js terminal component with WebSocket connection, resize handling, error states                                                               | ✅     |
| `apps/web/app/(dashboard)/hosts/[id]/page.tsx`    | Add Shell tab with HostShell component                                                                                                                    | ✅     |

**Implementation notes:**

- Uses one-time token authentication (30-second expiry) for WebSocket access
- Supports multiple shells: SHELL env, /bin/bash, /bin/zsh, /bin/sh fallback
- PTY resize messages sent as JSON `{Width, Height}` over WebSocket
- Token cleanup goroutine removes expired entries every 30 seconds
- Terminal theme matches Windows Terminal color scheme

**Complexity:** Low. The container bash WebSocket handler is the pattern;
this is the same thing without SSH — just a local PTY.

---

### 3.3 Boot Order Configuration

**Status:** ✅ **COMPLETED**

**Why.** Without boot order control, changing what a VM boots from requires
editing libvirt XML manually. Needed for: booting from ISO to reinstall,
network booting, changing disk boot priority.

**Files created / modified:**

| File                                              | Change                                                                                                          | Status |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------- | ------ |
| `apps/api/internal/model/vm.go`                   | Add `BootOrder []string` to `VM`, `VMCreateRequest` and `VMUpdateRequest`                                       | ✅     |
| `apps/api/internal/repository/libvirt/vm.go`      | Add `buildBootOrderXML` helper; generate boot devices in `buildDomainXML`; handle boot order in `Update`        | ✅     |
| `packages/proto/lab/v1/vm.proto`                  | Add `boot_order` to `VM`, `CreateVMRequest` and `UpdateVMRequest`                                               | ✅     |
| `apps/api/internal/connectsvc/vm.go`              | Add `BootOrder` to proto-to-model and model-to-proto conversions                                                | ✅     |
| `apps/web/components/create-vm-modal.tsx`         | Add boot order selection UI with add/remove/reorder functionality                                               | ✅     |

**Implementation notes:**

- Valid boot devices: `hd` (hard disk), `cdrom` (CD-ROM), `network` (PXE boot)
- Default boot order: `["hd", "cdrom"]`
- Boot order is an offline-only update (VM must be stopped)
- Boot devices are parsed from libvirt XML on VM read
- Frontend uses badge-style display with add/remove via dropdown

**Complexity:** Low.

---

### 3.4 Backup Improvements

**Status:** ✅ **COMPLETED**

**Three targeted additions that materially improve backup reliability:**

**a) Backup verification**

| File                                              | Change                                                                         | Status |
| ------------------------------------------------- | ------------------------------------------------------------------------------ | ------ |
| `apps/api/internal/model/backup.go`              | Add `VerifiedAt`, `VerificationStatus`, `VerificationError` fields            | ✅     |
| `packages/proto/lab/v1/backup.proto`             | Add `VerifyBackup` RPC, `VerificationStatus` enum, verification fields        | ✅     |
| `apps/api/internal/service/backup.go`            | Add `VerifyBackup` method calling libvirt repository                          | ✅     |
| `apps/api/internal/repository/libvirt/backup.go` | Add `Verify` method using `tar -tzf` or `qemu-img check`                      | ✅     |
| `apps/api/internal/handler/backup.go`            | Add `VerifyBackup` handler                                                     | ✅     |

**b) Backup encryption**

| File                                              | Change                                                                         | Status |
| ------------------------------------------------- | ------------------------------------------------------------------------------ | ------ |
| `apps/api/internal/model/backup.go`              | Add `Encrypted` to Backup, `Encrypt`/`EncryptionPassphrase` to requests        | ✅     |
| `packages/proto/lab/v1/backup.proto`             | Add encryption fields to CreateBackupRequest, RestoreBackupRequest             | ✅     |
| `apps/api/internal/repository/libvirt/backup.go` | Add `encryptFile`/`decryptFile` using OpenSSL AES-256-CBC                      | ✅     |
| `apps/api/internal/service/backup.go`            | Pass encryption params through backup creation/restore                         | ✅     |

**c) Retention count policy**

| File                                              | Change                                                                         | Status |
| ------------------------------------------------- | ------------------------------------------------------------------------------ | ------ |
| `apps/api/internal/model/backup.go`              | Add `RetainCount` to BackupSchedule and requests                              | ✅     |
| `packages/proto/lab/v1/backup.proto`             | Add `retain_count` field to schedule messages                                  | ✅     |
| `apps/api/internal/service/backup.go`            | Add `applyRetainCountPolicy` to delete old backups after scheduled backup     | ✅     |

**Implementation notes:**

- Verification uses `tar -tzf` for tar archives, `qemu-img check` for disk images
- Encrypted backups verified by checking OpenSSL "Salted__" magic header
- Encryption uses OpenSSL AES-256-CBC with PBKDF2 key derivation
- Retention count policy runs after each scheduled backup completes
- Encrypted backups require passphrase for restore

**Complexity per item:** Low. Each is a self-contained addition.

---

### 3.5 Session Management

**Status:** ✅ **COMPLETED**

**Why.** Currently there is no way to see what sessions are active or
revoke one (e.g. a device you lost).

**Files created / modified:**

| File                                                | Change                                                                                         | Status |
| --------------------------------------------------- | ---------------------------------------------------------------------------------------------- | ------ |
| `apps/api/pkg/sqlite/migrations/010_sessions.sql`   | New — sessions table: id, user_id, jti, ip_address, user_agent, device_name, issued_at, last_seen_at, expires_at, revoked | ✅ |
| `apps/api/internal/repository/auth/session.go`      | New — SessionRepository with Create, GetByJTI, GetByID, UpdateLastSeen, Revoke, RevokeByJTI, RevokeAllUserSessions, ListByUser, IsRevoked, DeleteExpired, CountByUser | ✅ |
| `apps/api/internal/middleware/auth.go`              | Updated — Added sessionRepo parameter; check revoked flag on each request; update last_seen_at; added SessionJTIKey to context; added GetSessionJTIFromContext | ✅ |
| `apps/api/internal/service/auth.go`                 | Updated — Added sessionRepo parameter; sessions created on login/register/refresh; added ListSessions, RevokeSession, RevokeOtherSessions, UpdateSessionLastSeen, IsSessionRevoked | ✅ |
| `apps/api/pkg/auth/jwt.go`                          | Updated — GenerateAccessToken now returns (token, jti, error) for session tracking             | ✅ |
| `packages/proto/lab/v1/auth.proto`                  | Added Session message, ListSessions/RevokeSession/RevokeOtherSessions RPCs                    | ✅ |
| `apps/api/internal/handler/auth.go`                 | Added ListSessions, RevokeSession, RevokeOtherSessions handlers                               | ✅ |
| `apps/api/cmd/server/main.go`                       | Added sessionRepo initialization and passed to auth service and middleware                     | ✅ |
| `apps/web/lib/api/queries/auth.ts`                  | Added useSessions hook for fetching user sessions                                             | ✅ |
| `apps/web/lib/api/mutations/auth.ts`                | Added useRevokeSession and useRevokeOtherSessions mutation hooks                              | ✅ |
| `apps/web/app/(dashboard)/settings/page.tsx`        | Added Sessions tab with current session display, other sessions list, revoke functionality    | ✅ |

**Implementation notes:**

- JTI (JWT ID) is now embedded in every access token and tracked in sessions table
- On each authenticated request, middleware checks if session is revoked and updates last_seen_at
- Sessions are created on login, register, and token refresh
- Users can list their active sessions and revoke individual ones or all others
- Frontend UI shows current session with badge, other sessions with device info (browser, OS, IP)
- Relative time formatting for last activity (e.g. "2 hours ago")
- Confirmation dialogs for session revocation

**Complexity:** Low-Medium. JTI (JWT ID) claim needs adding to token generation; middleware check is one DB lookup.

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

**Why.** The codebase contained hardcoded insecure JWT secrets that could be
accidentally used in production, allowing attackers to forge authentication tokens.

**Changes made:**

| Location | Change |
|----------|--------|
| `apps/api/pkg/auth/jwt.go` | Changed `DefaultConfig()` to return `SecretKey: nil` instead of insecure default |
| `apps/api/internal/service/auth.go` | Changed `DefaultAuthServiceConfig()` to return `JWTSecret: nil` instead of insecure default |
| `apps/api/cmd/server/main.go` | Added strict validation: fails fast if JWT secret is empty or < 16 characters |
| `apps/api/config.example.yaml` | Updated to show `jwt_secret: ""` with prominent warning |
| `apps/api/pkg/auth/jwt_test.go` | Updated test to expect `nil` secret instead of insecure default |
| `apps/api/internal/service/auth_test.go` | Updated test to expect `nil` secret instead of insecure default |

**Validation:**

- Server now fails to start if `jwt_secret` is not configured
- Minimum 16 character length enforced
- Clear error message with instruction: `openssl rand -base64 32`

**Complexity:** Low. Simple validation logic, but critical for security.

---

### 7.2 Refactor Monolithic main.go

**Status:** ⏳ **PENDING**

**Why.** `cmd/server/main.go` is 360+ lines with all dependency injection, service
initialization, and server setup in a single function. This makes it hard to:
- Understand initialization order
- Test initialization logic
- Add new services without increasing complexity
- Identify circular dependencies

**Deliverable.** Modular initialization functions with clear separation of concerns.

**Proposed structure:**

```go
func main() {
    cfg := config.Load()
    setupLogging(cfg)
    
    db := initDatabase(cfg)
    defer db.Close()
    
    repos := initRepositories(db, cfg)
    services := initServices(repos, cfg)
    handlers := initHandlers(services, cfg)
    
    r := router.Router(handlers)
    srv := initServer(r, cfg)
    
    if err := runServer(srv); err != nil {
        log.Fatalf("Server error: %v", err)
    }
}
```

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/cmd/server/main.go` | Refactor into modular init functions |
| `apps/api/cmd/server/init_database.go` | New — Database initialization and migrations |
| `apps/api/cmd/server/init_repositories.go` | New — Repository initialization |
| `apps/api/cmd/server/init_services.go` | New — Service initialization with dependency injection |
| `apps/api/cmd/server/init_handlers.go` | New — Handler initialization |
| `apps/api/cmd/server/init_server.go` | New — HTTP server setup and graceful shutdown |

**Implementation notes:**

- Keep dependency injection explicit (no magic containers)
- Maintain initialization order: DB → Repos → Services → Handlers → Router → Server
- Add logging for each initialization step
- Consider using `google/wire` for compile-time dependency injection (future enhancement)
- Preserve graceful shutdown logic

**Complexity:** Medium. Mechanical refactoring but requires careful attention to
dependency order and error handling.

**Testing:**

- Add unit tests for individual init functions
- Add integration test for full initialization

---

### 7.3 Add Input Validation Layer

**Status:** ✅ **COMPLETED**

**Why.** Handlers and services were accepting requests without comprehensive validation, leading to:
- Invalid data entering the system
- Unclear error messages for users
- Potential security vulnerabilities (e.g., path traversal, injection)
- Inconsistent validation across endpoints

**Deliverable.** Comprehensive input validation framework with common validators and resource-specific validation.

**Files created:**

| File | Change |
|------|--------|
| `apps/api/internal/validator/validator.go` | New — Core validation types and common validators (email, password, names, tags, paths, etc.) |
| `apps/api/internal/validator/vm.go` | New — VM-specific validation (VM create/update, network config, disk config) |
| `apps/api/internal/validator/auth.go` | New — Auth validation (register, login, MFA, API keys, notification channels, alert rules) |
| `apps/api/internal/validator/storage.go` | New — Storage validation (pools, disks, ISOs, backups, snapshots, networks, firewall rules) |
| `apps/api/internal/validator/validator_test.go` | New — Comprehensive test suite with 100+ test cases |

**Validation rules implemented:**

### Common Validators
- `ValidateVMName`: 1-64 chars, starts with letter, alphanumeric + `_ -`
- `ValidateEmail`: RFC 5322 format, max 254 chars
- `ValidatePassword`: 8-128 chars, uppercase, lowercase, digit, special char
- `ValidateMemoryGB`: Min/max bounds checking
- `ValidateCPUCores`: Min/max bounds checking
- `ValidateDiskSizeGB`: Min/max bounds checking (1 GB - 10 TB)
- `ValidateTags`: Max 20 tags, lowercase alphanumeric + `-`, no duplicates
- `ValidateISOName`: Must end with `.iso`, no path traversal, max 255 chars
- `ValidateWebhookURL`: Must start with `http://` or `https://`, max 2048 chars
- `ValidateMFACode`: Exactly 6 digits
- `ValidatePath`: No path traversal (`..`), must be absolute

### VM Validation
- VM create: name, memory, CPU, disk, OS config, network config, tags
- VM update: optional fields with same validation as create
- Network config: type (user/bridge), bridge name, model, VLAN, port forwards
- Disk config: size, format (qcow2/raw/vmdk/vdi), bus (virtio/sata/scsi/ide)

### Auth Validation
- Register: email, password strength, role
- Login: email, password, MFA code (6 digits)
- API keys: name, permissions (validated against allowed list), expiresAt
- Notification channels: name, type (email/webhook), config validation
- Alert rules: name, type, severity, threshold, channel IDs

### Storage Validation
- Storage pools: name, type (dir/lvm/nfs/iscsi/zfs/glusterfs)
- Disks: name, size, format
- ISO: name, size limits, download URL validation
- Backups: VM ID, name, storage pool, retention days (1-3650)
- Snapshots: VM ID, name, description
- Networks: name, type, CIDR notation validation
- Firewall rules: direction, action, protocol, port ranges

**Validation error handling:**

```go
// ValidationError implements error interface
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

// Multiple errors
type ValidationErrors []*ValidationError

func (es ValidationErrors) Error() string {
    // Returns: "2 validation errors: name is required; email is invalid"
}
```

**Integration pattern (to be completed in handlers):**

```go
// In handler
func (h *vmHandler) CreateVM(ctx context.Context, req *connect.Request[labv1.CreateVMRequest]) (*connect.Response[labv1.CreateVMResponse], error) {
    // Validate input
    v := validator.DefaultVMCreateRequestValidator()
    errs := v.Validate(req.Msg.Name, req.Msg.Memory, req.Msg.CpuCores, req.Msg.DiskSize, ...)
    if len(errs) > 0 {
        return nil, connect.NewError(connect.CodeInvalidArgument, errs)
    }
    
    // ... proceed with business logic
}
```

**Testing:**
- 100+ test cases covering valid and invalid inputs
- Table-driven tests for all validators
- Edge cases: empty strings, too long, invalid characters, out of range values

**Complexity:** Medium-High. Created comprehensive validation framework with 4 files and extensive tests.

---

### 7.4 Consistent Error Handling Policy

**Status:** ✅ **COMPLETED**

**Why.** The codebase had inconsistent error handling:
- Mix of `log.Fatalf` (exits process) and graceful error returns
- No standard error types for API responses
- Error context lost when wrapping
- Inconsistent logging levels

**Deliverable.** Comprehensive error handling framework with typed errors, wrapping, and documentation.

**Files created:**

| File | Change |
|------|--------|
| `apps/api/internal/errors/errors.go` | New — Error types, codes, helpers (400+ lines) |
| `apps/api/ERROR_HANDLING.md` | New — Error handling policy documentation |
| `apps/api/internal/errors/errors_test.go` | New — Comprehensive test suite (40+ tests) |

**Error Types:**

### Error Codes
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_ARGUMENT` | 400 | Client specified invalid argument |
| `NOT_FOUND` | 404 | Resource not found |
| `ALREADY_EXISTS` | 409 | Resource already exists |
| `PERMISSION_DENIED` | 403 | Insufficient permissions |
| `UNAUTHENTICATED` | 401 | Authentication required |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service unavailable |
| `CONFLICT` | 409 | Request conflicts with current state |
| `RESOURCE_EXHAUSTED` | 429 | Resource quota exceeded |

### APIError Structure
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

**Error Creation Functions:**
- `New(code, message)` — Create basic error
- `Wrap(err, code, message)` — Wrap with context
- `Wrapf(err, code, format, args...)` — Wrap with formatted message
- `NewNotFoundError(resourceType, resourceID)` — Resource not found
- `NewInvalidArgumentError(field, message)` — Invalid input
- `NewAlreadyExistsError(resourceType, resourceID)` — Resource exists
- `NewPermissionDeniedError(message)` — Access denied
- `NewUnauthenticatedError(message)` — Not authenticated
- `NewInternalError(cause, message)` — Internal error
- `NewUnavailableError(message)` — Service unavailable
- `NewConflictError(message)` — Conflict with current state

**Error Checking Functions:**
- `IsNotFound(err)` — Check if not found error
- `IsInvalidArgument(err)` — Check if invalid argument
- `IsAlreadyExists(err)` — Check if already exists
- `IsPermissionDenied(err)` — Check if permission denied
- `IsUnauthenticated(err)` — Check if unauthenticated
- `IsInternal(err)` — Check if internal error
- `GetErrorCode(err)` — Get error code
- `GetHTTPStatus(err)` — Get HTTP status code

**Usage Pattern:**

```go
// Repository layer
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

// Service layer
func (s *VMService) CreateVM(ctx context.Context, req *CreateVMRequest) (*VM, error) {
    // Validate
    if req.Name == "" {
        return nil, errors.NewInvalidArgumentError("name", "is required")
    }
    
    // Check for duplicates
    _, err := s.repo.GetByName(ctx, req.Name)
    if err == nil {
        return nil, errors.NewAlreadyExistsError("vm", req.Name)
    }
    
    // Create
    vm, err := s.repo.Create(ctx, req)
    if err != nil {
        return nil, errors.NewInternalError(err, "failed to create VM")
    }
    return vm, nil
}

// Handler layer
func (h *vmHandler) CreateVM(ctx context.Context, req *connect.Request[labv1.CreateVMRequest]) (*connect.Response[labv1.CreateVMResponse], error) {
    vm, err := h.vmSvc.CreateVM(ctx, req.Msg)
    if err != nil {
        return nil, connectErrorFromAPIError(err) // Convert to Connect error
    }
    return connect.NewResponse(modelToProto(vm)), nil
}
```

**Complexity:** Medium-High. Created comprehensive error handling framework with proper error chaining, stack traces, and documentation.

---

### 7.5 Decouple from libvirt (Interface Extraction)

**Status:** ⏳ **PENDING**

**Why.** Services are tightly coupled to libvirt implementations:
- Direct imports of `libvirt.org/go/libvirt` in service layer
- Hard to test without libvirt
- Hard to support alternative backends (e.g., Docker-only mode)
- Violates dependency inversion principle

**Current tight coupling:**

```go
// In internal/service/vm.go
import libvirt "libvirt.org/go/libvirt"

// Direct libvirt calls
if state == libvirt.DOMAIN_RUNNING {
    // ...
}
```

**Deliverable.** Extract libvirt operations behind interfaces for testability
and future backend flexibility.

**Files to create/modify:**

| File | Change |
|------|--------|
| `apps/api/internal/repository/libvirt/interfaces.go` | New — Define interfaces for libvirt operations |
| `apps/api/internal/repository/libvirt/vm.go` | Implement `VMProvider` interface |
| `apps/api/internal/repository/libvirt/network.go` | Implement `NetworkProvider` interface |
| `apps/api/internal/repository/libvirt/storage.go` | Implement `StorageProvider` interface |
| `apps/api/internal/service/vm.go` | Depend on interfaces, not concrete libvirt types |
| `apps/api/internal/service/*.go` | Update to use interfaces |
| `apps/api/pkg/libvirtx/mock.go` | New — Mock implementation for testing |

**Proposed interfaces:**

```go
package libvirt

// VMProvider abstracts libvirt VM operations
type VMProvider interface {
    GetDomainState(vmID int) (DomainState, error)
    GetDomainXML(vmID int) (string, error)
    DefineDomain(xml string) (int, error)
    StartDomain(vmID int) error
    StopDomain(vmID int) error
    // ... other VM operations
}

// NetworkProvider abstracts libvirt network operations
type NetworkProvider interface {
    ListNetworks() ([]Network, error)
    CreateNetwork(cfg NetworkConfig) error
    // ... other network operations
}

// StorageProvider abstracts libvirt storage operations
type StorageProvider interface {
    ListStoragePools() ([]StoragePool, error)
    CreateStorageDisk(pool string, cfg DiskConfig) error
    // ... other storage operations
}
```

**Service layer usage:**

```go
type VMService struct {
    vmProvider libvirt.VMProvider
    // ... other dependencies
}

func NewVMService(vmProvider libvirt.VMProvider, ...) *VMService {
    return &VMService{
        vmProvider: vmProvider,
        // ...
    }
}
```

**Complexity:** High. Requires careful extraction of libvirt dependencies
while maintaining functionality. Should be done incrementally.

**Testing:**

- Add unit tests using mock providers
- Verify interface coverage for all libvirt operations used

---

### 7.6 Configure SQLite Connection Pool

**Status:** ✅ **COMPLETED**

**Why.** SQLite connection was used without proper pool configuration, which can lead to:
- "database is locked" errors under concurrent load
- Connection leaks
- Poor performance with multiple concurrent requests

**Changes made:**

| File | Change |
|------|--------|
| `apps/api/pkg/sqlite/db.go` | Added `SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`, `SetConnMaxLifetime(time.Hour)` |

**Implementation:**

```go
// SQLite allows only one writer at a time, so we serialize writes
db.SetMaxOpenConns(1)           // Serialize writes to prevent "database is locked"
db.SetMaxIdleConns(1)           // Keep one connection warm
db.SetConnMaxLifetime(time.Hour) // Recycle connections periodically
```

**Note:** WAL mode and busy_timeout were already configured in the original code.

**Complexity:** Low. Simple configuration change.

---

### 7.7 Add Global Rate Limiting

**Status:** ✅ **COMPLETED**

**Why.** Rate limiting was only applied to auth endpoints (5 req/s). Other endpoints had no protection against:
- Accidental runaway scripts
- Denial of service attacks
- Resource exhaustion (e.g., flooding VM creation)

**Changes made:**

| File | Change |
|------|--------|
| `apps/api/internal/router/router.go` | Added global rate limiter middleware: 100 req/s, burst 200 |

**Implementation:**

```go
// Global rate limiter: 100 requests/second with burst of 200
// This protects against accidental runaway scripts and DoS attacks.
// Per-endpoint rate limiters (e.g., auth endpoints) use stricter limits.
globalRateLimiter := appmiddleware.NewRateLimiter(rate.Limit(100), 200)
r.Use(globalRateLimiter.HTTPMiddleware)
```

**Rate limit tiers:**

| Tier | Limit | Endpoints |
|------|-------|-----------|
| Global | 100 req/s, burst 200 | All endpoints (default) |
| Auth | 5 req/s, burst 10 | Login, Register, MFA (already existed) |
| Expensive | 10 req/s, burst 20 | VM create/delete, backup, snapshot (future enhancement) |
| Read-only | 200 req/s, burst 500 | List operations, metrics (future enhancement) |

**Complexity:** Low. Single line to create limiter, single line to add middleware.

---

### 7.8 Add Context Propagation

**Status:** ✅ **COMPLETED**

**Why.** Some goroutines and long-running operations didn't propagate context, leading to:
- Operations that can't be cancelled
- Resource leaks on shutdown
- No timeout enforcement

**Changes made:**

| File | Change |
|------|--------|
| `apps/api/internal/service/collector.go` | Added context to `Start(ctx)`, `collectLoop(ctx)`, `cleanupLoop(ctx)`, `collectOnce(ctx)`, `runCleanup(ctx)` |
| `apps/api/internal/service/backup.go` | Added context to `NewBackupService(ctx, ...)`, `startScheduler(ctx)`, `checkDueSchedules(ctx)`, `runScheduledBackup(ctx, ...)` |
| `apps/api/cmd/server/main.go` | Pass `context.Background()` to `Collector.Start()` and `NewBackupService()` |
| `apps/api/internal/service/backup_test.go` | Updated all test calls to `NewBackupService()` to include context |

**Implementation pattern:**

```go
// Service layer - accept context for long-running operations
func (c *Collector) Start(ctx context.Context) {
    // ...
    go c.collectLoop(ctx)
    go c.cleanupLoop(ctx)
}

// Goroutines respect context cancellation
func (c *Collector) collectLoop(ctx context.Context) {
    ticker := time.NewTicker(c.config.CollectionInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Println("[collector] context cancelled, stopping")
            return
        case <-ticker.C:
            c.collectOnce(ctx)
        }
    }
}

// Database operations use context with timeout
func (c *Collector) collectOnce(ctx context.Context) {
    // ... collect metrics ...
    
    metricCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    if err := c.metricRepo.RecordBatch(metricCtx, metrics); err != nil {
        // handle error
    }
}
```

**Benefits:**

- Graceful shutdown when server receives SIGTERM/SIGINT
- Operations can be cancelled by clients (via Connect RPC deadline propagation)
- Prevents resource leaks from orphaned goroutines
- Enables timeout enforcement for database operations

**Complexity:** Medium. Required threading context through many call sites.

---

### 7.9 API Versioning Strategy

**Status:** ✅ **COMPLETED**

**Why.** The API is hardcoded as `v1` with no documented migration strategy. As the API evolves, breaking changes will require careful version management.

**Deliverable.** Documented API versioning and deprecation policy.

**Files created:**

| File | Change |
|------|--------|
| `apps/api/API_VERSIONING.md` | New — Comprehensive versioning strategy and deprecation policy document |

**Key policies documented:**

- **Breaking changes** require new major version (v1 → v2)
- **Deprecation period**: minimum 6 months
- **Non-breaking changes** (adding fields, services) are safe in existing version
- **Migration guides** required for each major version
- **Backward compatibility** supported for one version

**Deprecation phases:**

1. **Phase 1**: Add `deprecated = true` to proto, log warnings
2. **Phase 2**: 6+ month warning period with usage monitoring
3. **Phase 3**: Remove in next major version

**HTTP headers for REST endpoints:**
```http
Deprecation: true
Sunset: Sat, 01 Mar 2027 00:00:00 GMT
Link: </api/v2>; rel="successor-version"
```

**Complexity:** Low. Documentation only, but requires discipline for future changes.

---

### 7.10 Establish Naming Conventions

**Status:** ✅ **COMPLETED**

**Why.** Inconsistent naming makes the codebase harder to navigate:
- Package naming: `pkg/tus` vs `internal/handler`
- Constructor naming: `NewAuthService` vs `NewCollector`
- Variable naming: `repo` vs `repository`, `svc` vs `service`

**Deliverable.** Documented naming conventions.

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

### Completed Items (8/10)

| # | Item | Status | Security | Effort | Impact | Commit |
|---|------|--------|----------|--------|--------|--------|
| 7.1 | Remove Insecure JWT Defaults | ✅ | **HIGH** | Low | **HIGH** | `4cb27e2` |
| 7.3 | Add Input Validation Layer | ✅ | MEDIUM | Medium-High | **HIGH** | `457eb5e` |
| 7.4 | Consistent Error Handling Policy | ✅ | LOW | Medium-High | MEDIUM | _pending_ |
| 7.6 | Configure SQLite Connection Pool | ✅ | LOW | Low | LOW | `f16fc87` |
| 7.7 | Add Global Rate Limiting | ✅ | MEDIUM | Low | MEDIUM | `6a3c10e` |
| 7.8 | Add Context Propagation | ✅ | LOW | Medium | MEDIUM | `999ce8a` |
| 7.9 | API Versioning Strategy | ✅ | LOW | Low | LOW | `04841e3` |
| 7.10 | Establish Naming Conventions | ✅ | LOW | Low | LOW | `e9e1b55` |

### Remaining Items (2/10)

| Priority | Item | Security | Effort | Impact |
|----------|------|----------|--------|--------|
| ⚠️ **MEDIUM** | 7.2 Refactor Monolithic main.go | LOW | Medium | MEDIUM |
| 📝 **LOW** | 7.5 Decouple from libvirt | LOW | High | MEDIUM |

**Recommended next steps:**

1. **7.2 Refactor Monolithic main.go** — Improves maintainability (last substantive item)
2. **7.5 Decouple from libvirt** — Large refactoring, do incrementally as part of feature work

