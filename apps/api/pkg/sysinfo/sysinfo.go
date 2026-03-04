// Package sysinfo provides a platform-independent interface for gathering
// host system metrics such as CPU usage, memory, disk, uptime and network I/O.
//
// Obtain an instance via New() — the correct implementation is selected at
// compile time via build constraints (linux.go, stub.go).
package sysinfo

// SystemInfo is the interface through which callers retrieve host metrics.
// All implementations must be safe to call from multiple goroutines.
type SystemInfo interface {
	// GetCPUUsage returns the current CPU utilisation as a percentage (0–100).
	GetCPUUsage() float64

	// GetMemoryInfo returns memory used and total in GiB.
	GetMemoryInfo() (used, total float64)

	// GetDiskInfo returns disk used and total in GB (or TB when ≥ 1 000 GB).
	GetDiskInfo() (used, total float64)

	// GetUptime returns a human-readable uptime string, e.g. "3d 2h 15m".
	GetUptime() string

	// GetLoadAvg returns the 1-, 5- and 15-minute load averages.
	GetLoadAvg() [3]float64

	// GetNetworkStats returns cumulative bytes received and sent, in MiB.
	GetNetworkStats() (in, out float64)

	// GetKernelVersion returns a short identifier for the running OS/kernel,
	// e.g. "Linux 6.8.0".
	GetKernelVersion() string

	// DefaultCPUModel returns the recommended default CPU model for the given
	// guest architecture on the current host.
	DefaultCPUModel(guestArch string) string

	// HostArch returns the canonical libvirt name for the host architecture.
	HostArch() string

	// FirmwarePath returns the first available OVMF/AAVMF firmware image path
	// for the given guest architecture (e.g. "x86_64", "aarch64"), falling
	// back to the canonical default location if none are installed.
	FirmwarePath(guestArch string) string

	// EmulatorPath returns the path to the qemu-system-<arch> binary for the
	// given guest architecture.
	EmulatorPath(guestArch string) string

	// SupportsBridgeNetworking returns true if the host supports bridge
	// networking for VMs. On Linux this checks for qemu-bridge-helper.
	SupportsBridgeNetworking() bool

	// DefaultBridgeName returns the default bridge interface name for the platform.
	// Returns empty string if bridge name is not needed.
	DefaultBridgeName() string

	// DefaultNetworkType returns the default network type for the platform.
	// Returns "bridge" if bridge networking is supported, otherwise "user".
	DefaultNetworkType() string

	// DataDir returns the default data directory for the platform
	DataDir() string

	// ConfigPaths returns ordered list of config file locations to search
	ConfigPaths() []string

	// ISODir returns the default ISO storage directory
	ISODir(baseDir string) string

	// VMDiskDir returns the default VM disk directory
	VMDiskDir(baseDir string) string
}
