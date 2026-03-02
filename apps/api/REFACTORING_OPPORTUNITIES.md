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
    if sys.IsDarwin() {
        bridgeName = ""
    }
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

**Location:** `pkg/sysinfo/sysinfo.go`, `pkg/sysinfo/darwin.go`, `pkg/sysinfo/linux.go`

---

## 2. HIGH PRIORITY: Path Configuration by Platform

### Current Issue
`internal/config/config.go` has hardcoded platform checks:

```go
// Lines 126-128
dataBaseDir := filepath.Join(homeDir, ".local", "share", "lab")
if runtime.GOOS == "darwin" {
    dataBaseDir = filepath.Join(homeDir, "Library", "Application Support", "lab")
}

// Line 212
if runtime.GOOS != "windows" {
    paths = append(paths, "/etc/lab/config.yaml")
}
```

### Problem
- Scattered platform logic throughout config package
- Hard to add new platforms
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

// Implementation in darwin.go
func (d *darwinInfo) DataDir() string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, "Library", "Application Support", "lab")
}

func (d *darwinInfo) ConfigPaths() []string {
    homeDir, _ := os.UserHomeDir()
    return []string{
        "config.yaml",
        filepath.Join(homeDir, "Library", "Preferences", "lab", "config.yaml"),
        "/Library/Application Support/lab/config.yaml",
    }
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
// Before
func defaults() *Config {
    homeDir, _ := os.UserHomeDir()
    dataBaseDir := filepath.Join(homeDir, ".local", "share", "lab")
    if runtime.GOOS == "darwin" {
        dataBaseDir = filepath.Join(homeDir, "Library", "Application Support", "lab")
    }
    // ...
}

// After
func defaults() *Config {
    sys := sysinfo.New()
    dataBaseDir := sys.DataDir()
    // ...
}

// Before
func getConfigPath() string {
    paths := []string{"config.yaml", ...}
    if runtime.GOOS != "windows" {
        paths = append(paths, "/etc/lab/config.yaml")
    }
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
- ✅ All platform-specific paths in one place
- ✅ Easy to add new platforms (FreeBSD, etc.)
- ✅ Config package focuses on configuration logic, not paths
- ✅ Testable via mock SystemInfo

---

## 3. MEDIUM PRIORITY: OS Type Mapping

### Current Issue
`internal/connectsvc/common.go` and `internal/repository/libvirt/vm.go` have duplicated OS type mapping:

```go
// In connectsvc/common.go
func modelOSTypeToProto(t model.OSType) labv1.OsType {
    switch t {
    case model.OSTypeLinux:
        return labv1.OsType_OS_TYPE_LINUX
    case model.OSTypeWindows:
        return labv1.OsType_OS_TYPE_WINDOWS
    // ...
}

// In repository/libvirt/vm.go
func mapOSToLibosinfo(osCfg model.OSConfig) string {
    switch osCfg.Type {
    case model.OSTypeWindows:
        switch {
        case strings.Contains(version, "11"):
            return "http://microsoft.com/windows/11"
        // ... many cases ...
    case model.OSTypeLinux:
        switch {
        case strings.Contains(version, "ubuntu"):
            return "http://ubuntu.com/ubuntu/24.04"
        // ... many cases ...
    }
}
```

### Problem
- Large switch statements (100+ lines)
- Hard to add new OS types
- Logic mixed between proto conversion and libosinfo mapping

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

// Registry provides OS metadata
type Registry struct {
    definitions map[string]OSDefinition
}

// New creates a new OS registry
func New() *Registry {
    r := &Registry{definitions: make(map[string]OSDefinition)}
    r.registerDefaults()
    return r
}

// Get returns OS definition by libosinfo ID
func (r *Registry) Get(id string) (OSDefinition, bool) {
    def, ok := r.definitions[id]
    return def, ok
}

// FromVersion returns libosinfo ID from OS type and version
func (r *Registry) FromOSConfig(osType model.OSType, version string) string {
    // Centralized mapping logic
}

func (r *Registry) registerDefaults() {
    // Linux
    r.definitions["http://ubuntu.com/ubuntu/24.04"] = OSDefinition{
        ID: "http://ubuntu.com/ubuntu/24.04",
        Name: "Ubuntu 24.04",
        Family: "linux",
        Vendor: "canonical",
        // ...
    }
    
    // Windows
    r.definitions["http://microsoft.com/windows/11"] = OSDefinition{
        ID: "http://microsoft.com/windows/11",
        Name: "Windows 11",
        Family: "windows",
        Vendor: "microsoft",
        RecommendedRAM: 4096,
        RecommendedDisk: 64,
        // ...
    }
}
```

**Update:** `internal/repository/libvirt/vm.go`

```go
// Before
func mapOSToLibosinfo(osCfg model.OSConfig) string {
    // 100+ line switch statement
}

// After
var osRegistry = osinfo.New()

func mapOSToLibosinfo(osCfg model.OSConfig) string {
    return osRegistry.FromOSConfig(osCfg.Type, osCfg.Version)
}

// New function for getting OS metadata
func getOSMetadata(libosinfoID string) (osinfo.OSDefinition, error) {
    def, ok := osRegistry.Get(libosinfoID)
    if !ok {
        return osinfo.OSDefinition{}, fmt.Errorf("unknown OS: %s", libosinfoID)
    }
    return def, nil
}
```

**Benefits:**
- ✅ Centralized OS metadata
- ✅ Easy to add new OS types (just add to registry)
- ✅ Can expose via API for frontend dropdowns
- ✅ Can add OS-specific recommendations (RAM, disk, etc.)
- ✅ Testable independently

---

## 4. MEDIUM PRIORITY: Emulator and Firmware Paths

### Current Issue
`pkg/sysinfo/darwin.go` and `pkg/sysinfo/linux.go` have hardcoded paths:

```go
// darwin.go
func (d *darwinInfo) EmulatorPath(guestArch string) string {
    candidates := []string{
        fmt.Sprintf("/opt/homebrew/bin/qemu-system-%s", guestArch),
        fmt.Sprintf("/usr/local/bin/qemu-system-%s", guestArch),
    }
    // ...
}

func (d *darwinInfo) FirmwarePath(guestArch string) string {
    candidates := []string{
        "/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
        "/usr/local/share/qemu/edk2-aarch64-code.fd",
    }
    // ...
}
```

### Problem
- Hardcoded paths don't work for custom installations
- No way to configure alternative paths without code changes
- Nix, MacPorts, custom builds not supported

### Recommended Solution

**Enhance:** `pkg/sysinfo/sysinfo.go`

```go
type SystemInfo interface {
    // ... existing methods ...
    
    // SetEmulatorPath allows overriding the default emulator path
    SetEmulatorPath(arch, path string)
    
    // SetFirmwarePath allows overriding the default firmware path
    SetFirmwarePath(arch, path string)
    
    // SearchEmulatorPath searches for emulator in common locations
    SearchEmulatorPath(guestArch string) string
    
    // SearchFirmwarePath searches for firmware in common locations
    SearchFirmwarePath(guestArch string) string
}
```

**Add:** `pkg/sysinfo/paths_search.go` (new file)

```go
package sysinfo

import (
    "os"
    "os/exec"
    "path/filepath"
)

// CommonEmulatorLocations returns platform-specific search paths
func CommonEmulatorLocations(arch string) []string {
    switch runtime.GOOS {
    case "darwin":
        return []string{
            "/opt/homebrew/bin",
            "/usr/local/bin",
            "/opt/qemu/bin",
            filepath.Join(os.Getenv("HOME"), ".nix-profile/bin"),
        }
    case "linux":
        return []string{
            "/usr/bin",
            "/usr/local/bin",
            "/opt/qemu/bin",
        }
    default:
        return []string{}
    }
}

// CommonFirmwareLocations returns platform-specific firmware search paths
func CommonFirmwareLocations(arch string) []string {
    switch runtime.GOOS {
    case "darwin":
        return []string{
            "/opt/homebrew/share/qemu",
            "/usr/local/share/qemu",
        }
    case "linux":
        return []string{
            "/usr/share/qemu",
            "/usr/share/edk2",
            "/usr/share/OVMF",
        }
    default:
        return []string{}
    }
}

// SearchPath searches for a file in the given directories
func SearchPath(filename string, dirs []string) string {
    for _, dir := range dirs {
        path := filepath.Join(dir, filename)
        if _, err := os.Stat(path); err == nil {
            return path
        }
    }
    return ""
}

// SearchCommand searches for a command in PATH
func SearchCommand(name string) string {
    path, err := exec.LookPath(name)
    if err != nil {
        return ""
    }
    return path
}
```

**Benefits:**
- ✅ Supports multiple installation methods (Homebrew, Nix, source)
- ✅ Allows programmatic path overrides
- ✅ Better error messages when binaries not found
- ✅ Easy to add new search locations

---

## 5. LOW PRIORITY: CPU Model Defaults

### Current Issue
`pkg/sysinfo/darwin.go` and `pkg/sysinfo/linux.go` have CPU model logic:

```go
func (d *darwinInfo) DefaultCPUModel(guestArch string) string {
    if guestArch == "aarch64" {
        return "maximum"
    }
    if runtime.GOARCH == "arm64" && guestArch == "x86_64" {
        return "Penryn"
    }
    return "host-passthrough"
}
```

### Problem
- CPU model selection is complex and architecture-specific
- May need updates for new CPU types (Intel vs AMD, Apple Silicon generations)

### Recommended Solution

**Enhance:** `pkg/sysinfo/cpu.go` (new file)

```go
package sysinfo

// CPUInfo contains CPU information
type CPUInfo struct {
    Model      string
    Features   []string
    Vendor     string // "apple", "intel", "amd"
    Generation string // "m1", "m2", "m3", etc.
}

// GetCPUInfo returns information about the host CPU
func GetCPUInfo() CPUInfo {
    // Platform-specific implementation
}

// RecommendedCPUModel returns the recommended CPU model for a guest
func RecommendedCPUModel(hostArch, guestArch string, osType string) string {
    cpu := GetCPUInfo()
    
    // Same architecture → use host-passthrough or maximum
    if hostArch == guestArch {
        if cpu.Vendor == "apple" {
            return "maximum"
        }
        return "host-passthrough"
    }
    
    // Cross-architecture emulation
    if hostArch == "aarch64" && guestArch == "x86_64" {
        return "qemu64"
    }
    
    return "qemu64" // Safe default
}
```

**Benefits:**
- ✅ Can detect specific CPU generation (M1 vs M2 vs M3)
- ✅ Can enable/disable specific features based on CPU
- ✅ Better performance tuning opportunities

---

## 6. LOW PRIORITY: Network Type Constants

### Current Issue
Network type is a string enum in proto and model:

```go
// model/vm.go
type NetworkType string
const (
    NetworkTypeUser   NetworkType = "user"
    NetworkTypeBridge NetworkType = "bridge"
)

// proto/lab/v1/common.proto
enum NetworkType {
  NETWORK_TYPE_UNSPECIFIED = 0;
  NETWORK_TYPE_USER = 1;
  NETWORK_TYPE_BRIDGE = 2;
}
```

### Problem
- String comparison is error-prone
- No validation of network type values
- Hard to add platform-specific network types (e.g., `vmnet` on macOS)

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
    TypeVMNet // macOS-specific
    TypeVDE   // Advanced, Linux/BSD
)

// String returns the string representation
func (t Type) String() string {
    switch t {
    case TypeUser: return "user"
    case TypeBridge: return "bridge"
    case TypeVMNet: return "vmnet"
    case TypeVDE: return "vde"
    default: return "unspecified"
    }
}

// FromString parses a string to Type
func FromString(s string) Type {
    switch s {
    case "user": return TypeUser
    case "bridge": return TypeBridge
    case "vmnet": return TypeVMNet
    case "vde": return TypeVDE
    default: return TypeUnspecified
    }
}

// IsSupported returns true if this network type is supported on the current platform
func (t Type) IsSupported() bool {
    sys := sysinfo.New()
    switch t {
    case TypeUser:
        return true // Always supported
    case TypeBridge:
        return sys.SupportsBridgeNetworking()
    case TypeVMNet:
        return sys.IsDarwin() && sys.SupportsBridgeNetworking()
    case TypeVDE:
        return false // Not implemented
    default:
        return false
    }
}
```

**Benefits:**
- ✅ Type-safe network type handling
- ✅ Platform-specific network types
- ✅ Validation at compile time
- ✅ Easy to add new network types

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

## Next Steps

1. **Immediate:** Implement Path Configuration refactoring (Section 2)
2. **Short-term:** Create OS Type Registry (Section 3)
3. **Medium-term:** Enhance Emulator/Firmware path detection (Section 4)
4. **Long-term:** CPU detection and Network type constants (Sections 5-6)

---

## Testing Strategy

For each refactoring:

1. **Unit Tests:** Test each platform implementation independently
2. **Integration Tests:** Test end-to-end VM creation on each platform
3. **Mock Tests:** Use mock SystemInfo for testing without platform dependencies

Example:
```go
// Mock SystemInfo for testing
type mockSystemInfo struct {
    isDarwin bool
    bridgeName string
    networkType string
}

func (m *mockSystemInfo) IsDarwin() bool { return m.isDarwin }
func (m *mockSystemInfo) DefaultBridgeName() string { return m.bridgeName }
func (m *mockSystemInfo) DefaultNetworkType() string { return m.networkType }
// ... other methods ...

// Test
func TestVMDefaults(t *testing.T) {
    sys := &mockSystemInfo{isDarwin: true, bridgeName: "", networkType: "bridge"}
    req := &model.VMCreateRequest{}
    applyDefaults(sys, req)
    
    if req.Network[0].Bridge != "" {
        t.Errorf("expected empty bridge name on macOS, got %q", req.Network[0].Bridge)
    }
}
```

---

## Conclusion

The refactoring pattern established for platform-specific networking defaults should be applied consistently across the codebase. This will:

1. **Improve maintainability** - Platform logic in one place
2. **Increase extensibility** - Easy to add new platforms
3. **Enhance testability** - Mock platform-specific behavior
4. **Reduce duplication** - No scattered `runtime.GOOS` checks
5. **Better error messages** - Platform-aware validation

Start with high-priority items (Path Configuration, OS Registry) for maximum impact.
