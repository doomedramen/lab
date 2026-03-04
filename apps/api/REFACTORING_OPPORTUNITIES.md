# API Codebase Refactoring Opportunities

## Executive Summary

This document identifies opportunities to improve modularity, extensibility, and maintainability of the API codebase by applying the same pattern used for platform-specific networking defaults (moving logic to `pkg/sysinfo`).

---

## 1. ✅ COMPLETED: Platform-Specific Defaults

### Before
```go
// In internal/service/vm.go
if sys.SupportsBridgeNetworking() {
    bridgeName := "vmbr0"
    req.Network = []model.NetworkConfig{...}
}
```

### After
```go
// In internal/service/vm.go
req.Network = []model.NetworkConfig{
    {
        Type:   model.NetworkType(sys.DefaultNetworkType()),
        Bridge: sys.DefaultBridgeName(),
        Model:  model.NetworkModelVirtio,
    },
}
```

**Location:** `pkg/sysinfo/sysinfo.go`, `pkg/sysinfo/linux.go`

---

## 2. HIGH PRIORITY: Path Configuration by Platform

### Current Issue
`internal/config/config.go` has hardcoded paths that should be managed by `sysinfo`:

```go
// Lines 126-128
dataBaseDir := filepath.Join(homeDir, ".local", "share", "lab")

// Line 212
if runtime.GOOS != "windows" {
    paths = append(paths, "/etc/lab/config.yaml")
}
```

### Problem
- Scattered platform logic throughout config package
- Violates single responsibility principle

### Recommended Solution

**Create:** `pkg/sysinfo/paths.go` (new file)

```go
// Add to SystemInfo interface
type SystemInfo interface {
    // ... existing methods ...
    
    // DataDir returns the default data directory for the platform
    DataDir() string
    
    // ConfigPaths returns ordered list of config file locations
    ConfigPaths() []string
    
    // ISODir returns the default ISO storage directory
    ISODir() string
}

// Implementation in linux.go
func (l *linuxInfo) DataDir() string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".local", "share", "lab")
}

func (l *linuxInfo) ConfigPaths() []string {
    homeDir, _ := os.UserHomeDir()
    return []string{
        "config.yaml",
        filepath.Join(homeDir, ".config", "lab", "config.yaml"),
        filepath.Join(homeDir, ".lab", "config.yaml"),
        "/etc/lab/config.yaml",
    }
}
```

**Update:** `internal/config/config.go`

```go
// After
func defaults() *Config {
    sys := sysinfo.New()
    dataBaseDir := sys.DataDir()
    // ...
}

// After
func getConfigPath() string {
    sys := sysinfo.New()
    for _, path := range sys.ConfigPaths() {
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    return "config.yaml"
}
```

**Benefits:**
- ✅ All paths in one place
- ✅ Easy to add new platforms if needed
- ✅ Config package focuses on configuration logic, not paths
- ✅ Testable via mock SystemInfo

---

## 3. MEDIUM PRIORITY: OS Type Mapping

### Current Issue
`internal/connectsvc/common.go` and `internal/repository/libvirt/vm.go` have duplicated OS type mapping.

### Recommended Solution

**Create:** `pkg/osinfo/osinfo.go` (new package)

```go
package osinfo

// OSDefinition contains metadata about an operating system
type OSDefinition struct {
    ID              string   // libosinfo ID
    Name            string   // Human-readable name
    Family          string   // "linux", "windows", etc.
    Vendor          string   // "microsoft", "canonical", etc.
    Version         string   // Version string
    Architecture    []string // Supported architectures
    RecommendedRAM  int64    // Minimum recommended RAM in MB
    RecommendedDisk int64    // Minimum recommended disk in GB
}
```

**Benefits:**
- ✅ Centralized OS metadata
- ✅ Easy to add new OS types
- ✅ Can expose via API for frontend dropdowns
- ✅ Testable independently

---

## 4. MEDIUM PRIORITY: Emulator and Firmware Paths

### Current Issue
`pkg/sysinfo/linux.go` has hardcoded paths:

```go
func (l *linuxInfo) EmulatorPath(guestArch string) string {
    return fmt.Sprintf("/usr/bin/qemu-system-%s", guestArch)
}
```

### Recommended Solution

**Enhance:** `pkg/sysinfo/sysinfo.go`

```go
type SystemInfo interface {
    // ... existing methods ...
    
    // SearchEmulatorPath searches for emulator in common locations
    SearchEmulatorPath(guestArch string) string
    
    // SearchFirmwarePath searches for firmware in common locations
    SearchFirmwarePath(guestArch string) string
}
```

**Benefits:**
- ✅ Supports multiple installation methods
- ✅ Better error messages when binaries not found

---

## 5. LOW PRIORITY: CPU Model Defaults

### Current Issue
`pkg/sysinfo/linux.go` has CPU model logic:

```go
func (l *linuxInfo) DefaultCPUModel(guestArch string) string {
    if guestArch == "aarch64" {
        return "maximum"
    }
    return "host-passthrough"
}
```

### Recommended Solution

**Enhance:** `pkg/sysinfo/cpu.go` (new file)

```go
package sysinfo

// CPUInfo contains CPU information
type CPUInfo struct {
    Model      string
    Features   []string
    Vendor     string // "intel", "amd"
}
```

**Benefits:**
- ✅ Can detect specific CPU generation
- ✅ Can enable/disable specific features based on CPU

---

## 6. LOW PRIORITY: Network Type Constants

### Current Issue
Network type is a string enum in proto and model.

### Recommended Solution

**Create:** `pkg/network/network.go` (new package)

```go
package network

// Type represents a network backend type
type Type int

const (
    TypeUnspecified Type = iota
    TypeUser
    TypeBridge
)

// IsSupported returns true if this network type is supported on the current platform
func (t Type) IsSupported() bool {
    sys := sysinfo.New()
    switch t {
    case TypeUser:
        return true // Always supported
    case TypeBridge:
        return sys.SupportsBridgeNetworking()
    default:
        return false
    }
}
```

**Benefits:**
- ✅ Type-safe network type handling
- ✅ Validation at compile time

---

## Implementation Priority

| Priority | Refactoring | Effort | Impact |
|----------|-------------|--------|--------|
| ✅ DONE | Platform Defaults | Low | High |
| 🔥 HIGH | Path Configuration | Medium | High |
| 🔥 HIGH | OS Type Registry | Medium | Medium |
| ⚠️ MEDIUM | Emulator/Firmware Paths | Low | Medium |
| ⚠️ MEDIUM | CPU Model Detection | Medium | Low |
| 📝 LOW | Network Type Constants | Medium | Low |

---

## Testing Strategy

For each refactoring:

1. **Unit Tests:** Test each platform implementation independently
2. **Integration Tests:** Test end-to-end VM creation
3. **Mock Tests:** Use mock SystemInfo for testing without platform dependencies

Example:
```go
// Mock SystemInfo for testing
type mockSystemInfo struct {
    bridgeName string
    networkType string
}

func (m *mockSystemInfo) DefaultBridgeName() string { return m.bridgeName }
func (m *mockSystemInfo) DefaultNetworkType() string { return m.networkType }
// ... other methods ...
```
