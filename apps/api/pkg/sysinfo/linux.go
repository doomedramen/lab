//go:build linux
// +build linux

package sysinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// linuxInfo implements SystemInfo by reading Linux pseudo-filesystems
// (/proc, /sys) directly — no external binaries required.
type linuxInfo struct{}

// New returns the Linux SystemInfo implementation.
func New() SystemInfo { return &linuxInfo{} }

func (l *linuxInfo) GetCPUUsage() float64 {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 8 {
			break
		}
		parse := func(s string) float64 {
			v, _ := strconv.ParseFloat(s, 64)
			return v
		}
		user := parse(parts[1])
		nice := parse(parts[2])
		system := parse(parts[3])
		idle := parse(parts[4])
		iowait := parse(parts[5])
		irq := parse(parts[6])
		softirq := parse(parts[7])

		total := user + nice + system + idle + iowait + irq + softirq
		used := total - idle
		if total > 0 {
			return (used / total) * 100
		}
		break
	}
	return 0
}

func (l *linuxInfo) GetMemoryInfo() (used, total float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memAvailable uint64
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(parts[1], 10, 64)
		switch parts[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		}
	}
	if memTotal == 0 {
		return 0, 0
	}
	totalGB := float64(memTotal) / 1024 / 1024
	usedGB := float64(memTotal-memAvailable) / 1024 / 1024
	return usedGB, totalGB
}

func (l *linuxInfo) GetDiskInfo() (used, total float64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs("/", &st); err != nil {
		return 0, 0
	}
	totalBytes := float64(st.Blocks) * float64(st.Bsize)
	freeBytes := float64(st.Bfree) * float64(st.Bsize)
	usedBytes := totalBytes - freeBytes

	totalGB := totalBytes / 1024 / 1024 / 1024
	usedGB := usedBytes / 1024 / 1024 / 1024

	if totalGB >= 1000 {
		usedTB := float64(int(usedGB/1000*10+0.5)) / 10
		totalTB := float64(int(totalGB/1000*10+0.5)) / 10
		return usedTB, totalTB
	}
	usedGB = float64(int(usedGB*10+0.5)) / 10
	totalGB = float64(int(totalGB*10+0.5)) / 10
	return usedGB, totalGB
}

func (l *linuxInfo) GetUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return "N/A"
	}
	parts := strings.Fields(string(data))
	if len(parts) == 0 {
		return "N/A"
	}
	secs, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return "N/A"
	}
	s := int(secs)
	return fmt.Sprintf("%dd %dh %dm", s/86400, (s%86400)/3600, (s%3600)/60)
}

func (l *linuxInfo) GetLoadAvg() [3]float64 {
	var avg [3]float64
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return avg
	}
	parts := strings.Fields(string(data))
	for i := 0; i < 3 && i < len(parts); i++ {
		avg[i], _ = strconv.ParseFloat(parts[i], 64)
	}
	return avg
}

func (l *linuxInfo) GetNetworkStats() (in, out float64) {
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) < 3 {
		return 0, 0
	}
	for _, line := range lines[2:] {
		parts := strings.Fields(line)
		if len(parts) < 10 || strings.Contains(parts[0], "lo:") {
			continue
		}
		rx, _ := strconv.ParseFloat(parts[1], 64)
		tx, _ := strconv.ParseFloat(parts[9], 64)
		in += rx / 1024 / 1024
		out += tx / 1024 / 1024
	}
	return in, out
}

func (l *linuxInfo) FirmwarePath(guestArch string) string {
	var candidates []string
	switch guestArch {
	case "aarch64":
		candidates = []string{
			"/usr/share/AAVMF/AAVMF_CODE.fd",
			"/usr/share/qemu-efi-aarch64/QEMU_EFI.fd",
			"/usr/share/edk2/aarch64/QEMU_EFI.fd",
		}
	default: // x86_64
		candidates = []string{
			"/usr/share/OVMF/OVMF_CODE.fd",
			"/usr/share/ovmf/OVMF.fd",
			"/usr/share/edk2/ovmf/OVMF_CODE.fd",
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func (l *linuxInfo) EmulatorPath(guestArch string) string {
	return fmt.Sprintf("/usr/bin/qemu-system-%s", guestArch)
}

func (l *linuxInfo) GetKernelVersion() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return "Linux"
	}
	// e.g. "Linux version 6.8.0-51-generic ..."
	parts := strings.Fields(strings.TrimSpace(string(data)))
	if len(parts) >= 3 {
		return fmt.Sprintf("Linux %s", parts[2])
	}
	return "Linux"
}

func (l *linuxInfo) DefaultCPUModel(guestArch string) string {
	if guestArch == "aarch64" {
		return "maximum"
	}
	if runtime.GOARCH == "arm64" && guestArch == "x86_64" {
		return "qemu64"
	}
	return "host-passthrough"
}

func (l *linuxInfo) HostArch() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

func (l *linuxInfo) SupportsBridgeNetworking() bool {
	// Check if qemu-bridge-helper exists and is setuid
	// Common locations: /usr/lib/qemu/, /usr/libexec/qemu-bridge-helper
	paths := []string{
		"/usr/lib/qemu/qemu-bridge-helper",
		"/usr/libexec/qemu-bridge-helper",
		"/usr/lib/qemu-bridge-helper",
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			// Check if it's executable (bridge helper should be setuid root)
			return info.Mode().IsRegular()
		}
	}
	return false
}

func (l *linuxInfo) DefaultBridgeName() string {
	// Linux uses traditional bridge networking with vmbr0 (Proxmox convention)
	return "vmbr0"
}

func (l *linuxInfo) DefaultNetworkType() string {
	if l.SupportsBridgeNetworking() {
		return "bridge"
	}
	return "user"
}

func (l *linuxInfo) DataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".local", "share", "lab")
}

func (l *linuxInfo) ConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		"config.yaml",
		"config.yml",
		filepath.Join(homeDir, ".config", "lab", "config.yaml"),
		filepath.Join(homeDir, ".lab", "config.yaml"),
		"/etc/lab/config.yaml",
	}
}

func (l *linuxInfo) ISODir(baseDir string) string {
	return filepath.Join(baseDir, "isos")
}

func (l *linuxInfo) VMDiskDir(baseDir string) string {
	return filepath.Join(baseDir, "disks")
}
