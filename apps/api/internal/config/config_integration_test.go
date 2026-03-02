//go:build integration
// +build integration

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// TestConfigIntegration tests that configuration loads correctly with real libvirt
func TestConfigIntegration(t *testing.T) {
	// Skip if libvirt is not available
	if _, err := os.Stat("/var/run/libvirt/libvirt-sock"); err != nil {
		t.Skip("libvirt socket not found - skipping integration test")
	}

	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	// Connect to real libvirt
	client, err := libvirtx.NewClient(&libvirtx.Config{URI: cfg.Libvirt.URI})
	if err != nil {
		t.Skipf("libvirt not available: %v", err)
	}
	defer client.Disconnect()

	conn, err := client.Connection()
	if err != nil {
		t.Fatalf("Failed to connect to libvirt: %v", err)
	}

	// Verify connection works
	version, err := conn.GetLibVersion()
	if err != nil {
		t.Fatalf("Failed to get libvirt version: %v", err)
	}

	if version == 0 {
		t.Error("Libvirt version should not be 0")
	}

	t.Logf("Connected to libvirt version: %d", version)
}

// TestConfigPathsIntegration tests that config paths are accessible
func TestConfigPathsIntegration(t *testing.T) {
	cfg := Load()

	// Ensure directories can be created
	err := cfg.EnsureDirectories()
	if err != nil {
		t.Fatalf("EnsureDirectories() error = %v", err)
	}

	// Verify directories exist
	for _, dir := range []string{cfg.Storage.DataDir, cfg.Storage.ISODir, cfg.Storage.VMDiskDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("Directory not accessible: %s - %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("Path is not a directory: %s", dir)
		}
	}

	// Clean up test directories if they're in temp location
	if filepath.HasPrefix(cfg.Storage.DataDir, os.TempDir()) {
		os.RemoveAll(cfg.Storage.DataDir)
	}
}

// TestEmulatorPathsIntegration tests that emulator paths are accessible
func TestEmulatorPathsIntegration(t *testing.T) {
	cfg := Load()

	// Test x86_64 emulator
	path := cfg.GetEmulatorPath("x86_64")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Logf("Emulator path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("x86_64 emulator found: %s", path)
		}
	} else {
		t.Log("No x86_64 emulator path configured")
	}

	// Test aarch64 emulator
	path = cfg.GetEmulatorPath("aarch64")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Logf("Emulator path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("aarch64 emulator found: %s", path)
		}
	} else {
		t.Log("No aarch64 emulator path configured")
	}
}

// TestFirmwarePathsIntegration tests that firmware paths are accessible
func TestFirmwarePathsIntegration(t *testing.T) {
	cfg := Load()

	// Test x86_64 firmware
	path := cfg.GetOVMFPathForArch("x86_64")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Logf("Firmware path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("x86_64 firmware found: %s", path)
		}
	} else {
		t.Log("No x86_64 firmware path configured")
	}

	// Test aarch64 firmware
	path = cfg.GetOVMFPathForArch("aarch64")
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			t.Logf("Firmware path exists but not accessible: %s - %v", path, err)
		} else {
			t.Logf("aarch64 firmware found: %s", path)
		}
	} else {
		t.Log("No aarch64 firmware path configured")
	}
}
