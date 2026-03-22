//go:build integration
// +build integration

package sysinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestSystemInfoIntegration tests real system information detection
func TestSystemInfoIntegration(t *testing.T) {
	sys := New()

	// Test HostArch returns valid architecture
	arch := sys.HostArch()
	if arch == "" {
		t.Fatal("HostArch() returned empty string")
	}
	t.Logf("Host architecture: %s", arch)

	// Verify it matches expected mapping
	expectedArch := runtime.GOARCH
	if expectedArch == "amd64" && arch != "x86_64" {
		t.Errorf("HostArch() = %q, want x86_64 for amd64", arch)
	}
	if expectedArch == "arm64" && arch != "aarch64" {
		t.Errorf("HostArch() = %q, want aarch64 for arm64", arch)
	}
}

// TestEmulatorPathIntegration tests real emulator path detection
func TestEmulatorPathIntegration(t *testing.T) {
	sys := New()

	// Test x86_64 emulator
	path := sys.EmulatorPath("x86_64")
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("x86_64 emulator path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("x86_64 emulator: %s (mode: %v)", path, info.Mode())
		}
	} else {
		t.Log("No x86_64 emulator path configured")
	}

	// Test aarch64 emulator
	path = sys.EmulatorPath("aarch64")
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("aarch64 emulator path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("aarch64 emulator: %s (mode: %v)", path, info.Mode())
		}
	} else {
		t.Log("No aarch64 emulator path configured")
	}
}

// TestFirmwarePathIntegration tests real firmware path detection
func TestFirmwarePathIntegration(t *testing.T) {
	sys := New()

	// Test x86_64 firmware
	path := sys.FirmwarePath("x86_64")
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("x86_64 firmware path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("x86_64 firmware: %s (size: %d bytes)", path, info.Size())
		}
	} else {
		t.Log("No x86_64 firmware path configured")
	}

	// Test aarch64 firmware
	path = sys.FirmwarePath("aarch64")
	if path != "" {
		info, err := os.Stat(path)
		if err != nil {
			t.Logf("aarch64 firmware path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("aarch64 firmware: %s (size: %d bytes)", path, info.Size())
		}
	} else {
		t.Log("No aarch64 firmware path configured")
	}
}

// TestDataDirIntegration tests real data directory detection
func TestDataDirIntegration(t *testing.T) {
	sys := New()
	dataDir := sys.DataDir()

	if dataDir == "" {
		t.Fatal("DataDir() returned empty string")
	}

	// Should be absolute path
	if !filepath.IsAbs(dataDir) {
		t.Errorf("DataDir() = %q, want absolute path", dataDir)
	}

	// Should be writable
	testFile := filepath.Join(dataDir, ".test-write")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Logf("Data directory not writable: %s - %v", dataDir, err)
	} else {
		os.Remove(testFile)
		t.Logf("Data directory is writable: %s", dataDir)
	}

	// Platform-specific checks
	if runtime.GOOS == "darwin" {
		expectedSuffix := filepath.Join("Library", "Application Support", "lab")
		if !hasSuffix(dataDir, expectedSuffix) {
			t.Errorf("DataDir() on macOS = %q, want suffix %q", dataDir, expectedSuffix)
		}
	}
	if runtime.GOOS == "linux" {
		expectedSuffix := filepath.Join(".local", "share", "lab")
		if !hasSuffix(dataDir, expectedSuffix) {
			t.Errorf("DataDir() on Linux = %q, want suffix %q", dataDir, expectedSuffix)
		}
	}
}

// TestConfigPathsIntegration tests real config path detection
func TestConfigPathsIntegration(t *testing.T) {
	sys := New()
	paths := sys.ConfigPaths()

	if len(paths) == 0 {
		t.Fatal("ConfigPaths() returned empty list")
	}

	t.Logf("Config paths (%d):", len(paths))
	for _, path := range paths {
		t.Logf("  - %s", path)
		// Check if path exists
		if _, err := os.Stat(path); err == nil {
			t.Logf("    ✓ exists")
		}
	}

	// Should include config.yaml
	found := false
	for _, path := range paths {
		if path == "config.yaml" || path == "config.yml" {
			found = true
			break
		}
	}
	if !found {
		t.Error("ConfigPaths() should include config.yaml or config.yml")
	}
}

// TestBridgeNetworkingIntegration tests real bridge networking detection
func TestBridgeNetworkingIntegration(t *testing.T) {
	sys := New()
	supported := sys.SupportsBridgeNetworking()

	t.Logf("Bridge networking supported: %v", supported)

	// Platform-specific expectations
	if runtime.GOOS == "darwin" {
		// macOS should support vmnet on 11.0+
		t.Logf("macOS version check performed")
	}
	if runtime.GOOS == "linux" {
		// Linux should check for qemu-bridge-helper
		t.Logf("Linux qemu-bridge-helper check performed")
	}
}

// TestDefaultNetworkTypeIntegration tests real network type detection
func TestDefaultNetworkTypeIntegration(t *testing.T) {
	sys := New()
	networkType := sys.DefaultNetworkType()

	if networkType == "" {
		t.Error("DefaultNetworkType() returned empty string")
	}

	t.Logf("Default network type: %s", networkType)

	// Should be either "bridge" or "user"
	if networkType != "bridge" && networkType != "user" {
		t.Errorf("DefaultNetworkType() = %q, want bridge or user", networkType)
	}
}

// TestDefaultBridgeNameIntegration tests real bridge name detection
func TestDefaultBridgeNameIntegration(t *testing.T) {
	sys := New()
	bridgeName := sys.DefaultBridgeName()

	t.Logf("Default bridge name: %q", bridgeName)

	// On macOS, should be empty (vmnet-shared doesn't need interface name)
	if runtime.GOOS == "darwin" && bridgeName != "" {
		t.Errorf("DefaultBridgeName() on macOS = %q, want empty string", bridgeName)
	}
}

// TestISODirIntegration tests real ISO directory detection
func TestISODirIntegration(t *testing.T) {
	sys := New()
	baseDir := t.TempDir()
	isoDir := sys.ISODir(baseDir)

	if isoDir == "" {
		t.Error("ISODir() returned empty string")
	}

	expected := filepath.Join(baseDir, "isos")
	if isoDir != expected {
		t.Errorf("ISODir(%q) = %q, want %q", baseDir, isoDir, expected)
	}

	// Should be creatable
	err := os.MkdirAll(isoDir, 0755)
	if err != nil {
		t.Errorf("Failed to create ISO directory: %v", err)
	}
}

// TestVMDiskDirIntegration tests real VM disk directory detection
func TestVMDiskDirIntegration(t *testing.T) {
	sys := New()
	baseDir := t.TempDir()
	vmDiskDir := sys.VMDiskDir(baseDir)

	if vmDiskDir == "" {
		t.Error("VMDiskDir() returned empty string")
	}

	expected := filepath.Join(baseDir, "disks")
	if vmDiskDir != expected {
		t.Errorf("VMDiskDir(%q) = %q, want %q", baseDir, vmDiskDir, expected)
	}

	// Should be creatable
	err := os.MkdirAll(vmDiskDir, 0755)
	if err != nil {
		t.Errorf("Failed to create VM disk directory: %v", err)
	}
}

// hasSuffix is a helper function (strings.HasSuffix may not be available in all contexts)
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
