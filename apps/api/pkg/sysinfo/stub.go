//go:build !linux
// +build !linux

package sysinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// stubInfo is a no-op implementation for unsupported platforms.
type stubInfo struct{}

// New returns a stub SystemInfo implementation that returns zero values.
func New() SystemInfo { return &stubInfo{} }

func (s *stubInfo) GetCPUUsage() float64             { return 0 }
func (s *stubInfo) GetMemoryInfo() (float64, float64) { return 0, 0 }
func (s *stubInfo) GetDiskInfo() (float64, float64)   { return 0, 0 }
func (s *stubInfo) GetUptime() string                 { return "N/A" }
func (s *stubInfo) GetLoadAvg() [3]float64            { return [3]float64{} }
func (s *stubInfo) GetNetworkStats() (float64, float64) { return 0, 0 }
func (s *stubInfo) GetKernelVersion() string            { return "unknown" }
func (s *stubInfo) DefaultCPUModel(guestArch string) string {
	if guestArch == "aarch64" {
		return "maximum"
	}
	if runtime.GOARCH == "arm64" && guestArch == "x86_64" {
		return "qemu64"
	}
	return "host-passthrough"
}

func (s *stubInfo) HostArch() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}
func (s *stubInfo) FirmwarePath(guestArch string) string { return "" }
func (s *stubInfo) EmulatorPath(guestArch string) string {
	return fmt.Sprintf("qemu-system-%s", guestArch)
}
func (s *stubInfo) SupportsBridgeNetworking() bool { return false }
func (s *stubInfo) DefaultBridgeName() string      { return "" }
func (s *stubInfo) DefaultNetworkType() string     { return "user" }
func (s *stubInfo) DataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".lab")
}
func (s *stubInfo) ConfigPaths() []string {
	return []string{"config.yaml", "config.yml"}
}
func (s *stubInfo) ISODir(baseDir string) string {
	return filepath.Join(baseDir, "isos")
}
func (s *stubInfo) VMDiskDir(baseDir string) string {
	return filepath.Join(baseDir, "disks")
}
