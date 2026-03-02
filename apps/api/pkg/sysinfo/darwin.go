//go:build darwin
// +build darwin

package sysinfo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// darwinInfo implements SystemInfo on macOS by calling system utilities
// (sysctl, vm_stat, top, netstat, diskutil) and reading kernel structs.
type darwinInfo struct{}

// New returns the Darwin SystemInfo implementation.
func New() SystemInfo { return &darwinInfo{} }

func (d *darwinInfo) GetCPUUsage() float64 {
	// top -l 1 -n 0 samples the CPU once without process listing.
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "CPU usage:") {
			continue
		}
		// "CPU usage: 15.2% user, 8.3% sys, 76.5% idle"
		var user, sys float64
		for _, part := range strings.Split(line, ",") {
			part = strings.TrimSpace(part)
			fields := strings.Fields(part)
			if len(fields) < 2 {
				continue
			}
			pct, _ := strconv.ParseFloat(strings.TrimSuffix(fields[0], "%"), 64)
			switch {
			case strings.Contains(part, "user"):
				user = pct
			case strings.Contains(part, "sys"):
				sys = pct
			}
		}
		return user + sys
	}
	return 0
}

func (d *darwinInfo) GetMemoryInfo() (used, total float64) {
	// Total physical memory via sysctl.
	totalOut, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, 0
	}
	totalBytes, _ := strconv.ParseFloat(strings.TrimSpace(string(totalOut)), 64)
	totalGB := totalBytes / 1024 / 1024 / 1024

	// Used = Active + Wired + Compressed (matches Activity Monitor).
	vmOut, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, totalGB
	}

	var pageSize uint64
	stats := make(map[string]uint64)

	for _, line := range strings.Split(string(vmOut), "\n") {
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// First line: "Mach Virtual Memory Statistics: (page size of 16384 bytes)"
		if key == "Mach Virtual Memory Statistics" {
			if s := strings.Index(val, "("); s != -1 {
				if e := strings.Index(val, ")"); e != -1 {
					fields := strings.Fields(val[s+1 : e])
					for i, f := range fields {
						if f == "of" && i+1 < len(fields) {
							pageSize, _ = strconv.ParseUint(fields[i+1], 10, 64)
							break
						}
					}
				}
			}
			continue
		}

		n, err := strconv.ParseUint(strings.TrimSuffix(val, "."), 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "Pages active":
			stats["active"] = n
		case "Pages wired down":
			stats["wired"] = n
		case "Pages occupied by compressor":
			stats["compressor"] = n
		}
	}

	if pageSize == 0 {
		return 0, totalGB
	}
	usedPages := stats["active"] + stats["wired"] + stats["compressor"]
	usedGB := float64(usedPages) * float64(pageSize) / 1024 / 1024 / 1024
	return usedGB, totalGB
}

func (d *darwinInfo) GetDiskInfo() (used, total float64) {
	// Use statfs on root — no external binary needed.
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

func (d *darwinInfo) GetUptime() string {
	bootOut, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return "N/A"
	}
	// "{ sec = 1234567890, usec = 123456 } Fri Feb 20 ..."
	s := string(bootOut)
	start := strings.Index(s, "= ")
	comma := strings.Index(s, ",")
	if start == -1 || comma == -1 || comma <= start {
		return "N/A"
	}
	bootSec, err := strconv.ParseInt(strings.TrimSpace(s[start+2:comma]), 10, 64)
	if err != nil {
		return "N/A"
	}

	nowOut, err := exec.Command("date", "+%s").Output()
	if err != nil {
		return "N/A"
	}
	nowSec, _ := strconv.ParseInt(strings.TrimSpace(string(nowOut)), 10, 64)
	upSec := nowSec - bootSec
	return fmt.Sprintf("%dd %dh %dm", upSec/86400, (upSec%86400)/3600, (upSec%3600)/60)
}

func (d *darwinInfo) GetLoadAvg() [3]float64 {
	var avg [3]float64
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		return avg
	}
	// "{ 2.5 1.8 1.2 }"
	parts := strings.Fields(strings.Trim(string(out), "{} \n"))
	for i := 0; i < 3 && i < len(parts); i++ {
		avg[i], _ = strconv.ParseFloat(parts[i], 64)
	}
	return avg
}

func (d *darwinInfo) GetNetworkStats() (in, out float64) {
	netOut, err := exec.Command("netstat", "-ib").Output()
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(string(netOut), "\n")
	if len(lines) < 2 {
		return 0, 0
	}
	for _, line := range lines[1:] { // skip header
		parts := strings.Fields(line)
		if len(parts) < 10 || parts[0] == "lo0" {
			continue
		}
		rx, _ := strconv.ParseFloat(parts[6], 64)
		tx, _ := strconv.ParseFloat(parts[9], 64)
		in += rx / 1024 / 1024
		out += tx / 1024 / 1024
	}
	return in, out
}

func (d *darwinInfo) FirmwarePath(guestArch string) string {
	var candidates []string
	switch guestArch {
	case "aarch64":
		candidates = []string{
			"/opt/homebrew/share/qemu/edk2-aarch64-code.fd",
			"/usr/local/share/qemu/edk2-aarch64-code.fd",
		}
	default: // x86_64
		candidates = []string{
			"/opt/homebrew/share/qemu/edk2-x86_64-code.fd",
			"/usr/local/share/qemu/edk2-x86_64-code.fd",
		}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func (d *darwinInfo) EmulatorPath(guestArch string) string {
	candidates := []string{
		fmt.Sprintf("/opt/homebrew/bin/qemu-system-%s", guestArch),
		fmt.Sprintf("/usr/local/bin/qemu-system-%s", guestArch),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return candidates[0]
}

func (d *darwinInfo) GetKernelVersion() string {
	release, err := exec.Command("sysctl", "-n", "kern.osrelease").Output()
	if err != nil {
		return "Darwin"
	}
	return fmt.Sprintf("Darwin %s", strings.TrimSpace(string(release)))
}

func (d *darwinInfo) DefaultCPUModel(guestArch string) string {
	if guestArch == "aarch64" {
		return "maximum"
	}
	if runtime.GOARCH == "arm64" && guestArch == "x86_64" {
		// For x86_64 guests on Apple Silicon, use a basic compatible CPU model
		// "qemu64" may not be supported by all libvirt versions
		return "Penryn"
	}
	return "host-passthrough"
}

func (d *darwinInfo) HostArch() string {
	if runtime.GOARCH == "arm64" {
		return "aarch64"
	}
	return "x86_64"
}

func (d *darwinInfo) SupportsBridgeNetworking() bool {
	// macOS uses vmnet framework for bridging, which is built into QEMU
	// Check if vmnet-bridged is available by looking for the vmnet framework
	// vmnet is available on macOS 11.0+ (Big Sur and later)
	
	// Check if /Library/Frameworks/vmnet.framework exists
	if _, err := os.Stat("/Library/Frameworks/vmnet.framework"); err == nil {
		return true
	}
	
	// vmnet is built into modern macOS, so we can also check the OS version
	// macOS 11.0+ has vmnet support
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return false
	}
	
	version := strings.TrimSpace(string(out))
	parts := strings.Split(version, ".")
	if len(parts) > 0 {
		major, err := strconv.Atoi(parts[0])
		if err == nil && major >= 11 {
			return true
		}
	}
	
	return false
}

func (d *darwinInfo) IsDarwin() bool { return true }

func (d *darwinInfo) DefaultBridgeName() string {
	// macOS vmnet-shared doesn't need an interface name
	return ""
}

func (d *darwinInfo) DefaultNetworkType() string {
	if d.SupportsBridgeNetworking() {
		return "bridge"
	}
	return "user"
}

func (d *darwinInfo) DataDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Library", "Application Support", "lab")
}

func (d *darwinInfo) ConfigPaths() []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		"config.yaml",
		"config.yml",
		filepath.Join(homeDir, "Library", "Preferences", "lab", "config.yaml"),
		filepath.Join(homeDir, ".config", "lab", "config.yaml"),
	}
}

func (d *darwinInfo) ISODir(baseDir string) string {
	return filepath.Join(baseDir, "isos")
}

func (d *darwinInfo) VMDiskDir(baseDir string) string {
	return filepath.Join(baseDir, "disks")
}
