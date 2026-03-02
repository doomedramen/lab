package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if cfg.Server.Port != "8080" {
		t.Errorf("Server.Port = %q, want 8080", cfg.Server.Port)
	}
	if cfg.Server.Env != "development" {
		t.Errorf("Server.Env = %q, want development", cfg.Server.Env)
	}
	if cfg.Backend != BackendLibvirt {
		t.Errorf("Backend = %q, want libvirt", cfg.Backend)
	}
	if cfg.Libvirt.URI != "qemu:///session" {
		t.Errorf("Libvirt.URI = %q, want qemu:///session", cfg.Libvirt.URI)
	}
	if cfg.VM.DiskFormat != "qcow2" {
		t.Errorf("VM.DiskFormat = %q, want qcow2", cfg.VM.DiskFormat)
	}
	if cfg.VM.DiskBus != "virtio" {
		t.Errorf("VM.DiskBus = %q, want virtio", cfg.VM.DiskBus)
	}
	if cfg.Upload.ChunkSize != 50*1024*1024 {
		t.Errorf("Upload.ChunkSize = %d, want %d", cfg.Upload.ChunkSize, 50*1024*1024)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("Logging.Level = %q, want info", cfg.Logging.Level)
	}
	if cfg.Logging.VMLogRetentionDays != 7 {
		t.Errorf("Logging.VMLogRetentionDays = %d, want 7", cfg.Logging.VMLogRetentionDays)
	}
	if cfg.Auth.AccessTokenExpiry != "15m" {
		t.Errorf("Auth.AccessTokenExpiry = %q, want 15m", cfg.Auth.AccessTokenExpiry)
	}
	if cfg.Auth.RefreshTokenExpiry != "168h" {
		t.Errorf("Auth.RefreshTokenExpiry = %q, want 168h", cfg.Auth.RefreshTokenExpiry)
	}
	if cfg.Auth.Issuer != "lab-api" {
		t.Errorf("Auth.Issuer = %q, want lab-api", cfg.Auth.Issuer)
	}
	if cfg.Auth.MFA.IssuerName != "Lab" {
		t.Errorf("Auth.MFA.IssuerName = %q, want Lab", cfg.Auth.MFA.IssuerName)
	}
}

func TestValidate_ProductionRequiresJWTSecret(t *testing.T) {
	cfg := defaults()
	cfg.Server.Env = "production"
	cfg.Auth.JWTSecret = ""

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for production without JWT secret")
	}
}

func TestValidate_ProductionWithJWTSecret(t *testing.T) {
	cfg := defaults()
	cfg.Server.Env = "production"
	cfg.Auth.JWTSecret = "super-secret-key"

	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_DevelopmentAllowsEmptyJWTSecret(t *testing.T) {
	cfg := defaults()
	cfg.Server.Env = "development"
	cfg.Auth.JWTSecret = ""

	if err := cfg.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name    string
		envKey  string
		envVal  string
		check   func(*Config) bool
		desc    string
	}{
		{"PORT", "PORT", "9090", func(c *Config) bool { return c.Server.Port == "9090" }, "Server.Port"},
		{"ENV", "ENV", "production", func(c *Config) bool { return c.Server.Env == "production" }, "Server.Env"},
		{"BACKEND", "BACKEND", "libvirt", func(c *Config) bool { return c.Backend == BackendLibvirt }, "Backend"},
		{"LIBVIRT_URI", "LIBVIRT_URI", "qemu:///system", func(c *Config) bool { return c.Libvirt.URI == "qemu:///system" }, "Libvirt.URI"},
		{"ISO_DIR", "ISO_DIR", "/tmp/isos", func(c *Config) bool { return c.Storage.ISODir == "/tmp/isos" }, "Storage.ISODir"},
		{"VM_DISK_DIR", "VM_DISK_DIR", "/tmp/disks", func(c *Config) bool { return c.Storage.VMDiskDir == "/tmp/disks" }, "Storage.VMDiskDir"},
		{"STACKS_DIR", "STACKS_DIR", "/tmp/stacks", func(c *Config) bool { return c.Storage.StacksDir == "/tmp/stacks" }, "Storage.StacksDir"},
		{"JWT_SECRET", "JWT_SECRET", "my-secret", func(c *Config) bool { return c.Auth.JWTSecret == "my-secret" }, "Auth.JWTSecret"},
		{"ACCESS_TOKEN_EXPIRY", "ACCESS_TOKEN_EXPIRY", "30m", func(c *Config) bool { return c.Auth.AccessTokenExpiry == "30m" }, "Auth.AccessTokenExpiry"},
		{"REFRESH_TOKEN_EXPIRY", "REFRESH_TOKEN_EXPIRY", "24h", func(c *Config) bool { return c.Auth.RefreshTokenExpiry == "24h" }, "Auth.RefreshTokenExpiry"},
		{"JWT_ISSUER", "JWT_ISSUER", "my-app", func(c *Config) bool { return c.Auth.Issuer == "my-app" }, "Auth.Issuer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaults()
			t.Setenv(tt.envKey, tt.envVal)
			applyEnvOverrides(cfg)

			if !tt.check(cfg) {
				t.Errorf("%s was not set correctly from env %s=%s", tt.desc, tt.envKey, tt.envVal)
			}
		})
	}
}

func TestApplyEnvOverrides_DoesNotOverrideWhenEmpty(t *testing.T) {
	cfg := defaults()
	original := cfg.Server.Port

	// Ensure PORT is not set
	t.Setenv("PORT", "")

	applyEnvOverrides(cfg)

	// Empty string should NOT override (the function checks for empty)
	// Looking at the code: `if port := os.Getenv("PORT"); port != "" {`
	// So empty string won't override. But t.Setenv sets it to empty string,
	// and os.Getenv returns "" which fails the `!= ""` check.
	if cfg.Server.Port != original {
		t.Errorf("Port changed from %q to %q with empty env", original, cfg.Server.Port)
	}
}

func TestGetConfigPath_UsesEnvVar(t *testing.T) {
	t.Setenv("CONFIG_PATH", "/tmp/test-config.yaml")
	path := getConfigPath()
	if path != "/tmp/test-config.yaml" {
		t.Errorf("got %q, want /tmp/test-config.yaml", path)
	}
}

func TestGetConfigPath_FallsBackToDefault(t *testing.T) {
	t.Setenv("CONFIG_PATH", "")
	path := getConfigPath()
	// Should return either a found config path or "config.yaml" default
	if path == "" {
		t.Error("expected non-empty config path")
	}
}

func TestGetEmulatorPath_ExplicitPath(t *testing.T) {
	cfg := defaults()
	cfg.VM.EmulatorPath = "/custom/qemu"

	path := cfg.GetEmulatorPath("x86_64")
	if path != "/custom/qemu" {
		t.Errorf("got %q, want /custom/qemu", path)
	}
}

func TestGetEmulatorPath_ArchSpecific(t *testing.T) {
	cfg := defaults()
	cfg.VM.EmulatorPath = "" // Clear global override

	// The arch-specific paths should be populated from defaults
	for _, arch := range []string{"x86_64", "aarch64"} {
		path := cfg.GetEmulatorPath(arch)
		if path == "" {
			t.Errorf("expected non-empty emulator path for %s", arch)
		}
	}
}

func TestGetEmulatorPath_UnknownArch(t *testing.T) {
	cfg := defaults()
	cfg.VM.EmulatorPath = ""

	// Unknown arch falls through to sysinfo default
	path := cfg.GetEmulatorPath("riscv64")
	// Should not panic, may return empty or a default
	_ = path
}

func TestGetOVMFPathForArch(t *testing.T) {
	cfg := defaults()

	// With explicit paths set
	cfg.VM.OVMFPath = "/usr/share/OVMF/OVMF_CODE.fd"
	cfg.VM.OVMFPathARM64 = "/usr/share/AAVMF/AAVMF_CODE.fd"

	if got := cfg.GetOVMFPathForArch("x86_64"); got != "/usr/share/OVMF/OVMF_CODE.fd" {
		t.Errorf("x86_64: got %q, want OVMF path", got)
	}
	if got := cfg.GetOVMFPathForArch("aarch64"); got != "/usr/share/AAVMF/AAVMF_CODE.fd" {
		t.Errorf("aarch64: got %q, want AAVMF path", got)
	}
}

func TestGetOVMFPathForArch_FallbackToSysinfo(t *testing.T) {
	cfg := defaults()
	cfg.VM.OVMFPath = ""
	cfg.VM.OVMFPathARM64 = ""

	// Should not panic — falls back to sysinfo detection
	_ = cfg.GetOVMFPathForArch("x86_64")
	_ = cfg.GetOVMFPathForArch("aarch64")
}

func TestEnsureDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaults()
	cfg.Storage.DataDir = filepath.Join(tmpDir, "data")
	cfg.Storage.ISODir = filepath.Join(tmpDir, "isos")
	cfg.Storage.VMDiskDir = filepath.Join(tmpDir, "disks")

	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("EnsureDirectories: %v", err)
	}

	for _, dir := range []string{cfg.Storage.DataDir, cfg.Storage.ISODir, cfg.Storage.VMDiskDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("dir %s not created: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}
}

func TestEnsureDirectories_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := defaults()
	cfg.Storage.DataDir = filepath.Join(tmpDir, "data")
	cfg.Storage.ISODir = filepath.Join(tmpDir, "isos")
	cfg.Storage.VMDiskDir = filepath.Join(tmpDir, "disks")

	// Call twice — should not error
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if err := cfg.EnsureDirectories(); err != nil {
		t.Fatalf("second call: %v", err)
	}
}

func TestConfigString(t *testing.T) {
	cfg := defaults()
	s := cfg.String()

	if s == "" {
		t.Error("expected non-empty string")
	}
	// Should contain key fields
	if !containsStr(s, "8080") {
		t.Error("expected port in string representation")
	}
	if !containsStr(s, "libvirt") {
		t.Error("expected backend in string representation")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findStr(s, substr)
}

func findStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestExpandPaths_MakesAbsolute(t *testing.T) {
	cfg := defaults()
	cfg.Storage.ISODir = "relative/path"
	cfg.Storage.VMDiskDir = "another/relative"

	cfg.expandPaths()

	if !filepath.IsAbs(cfg.Storage.ISODir) {
		t.Errorf("ISODir not absolute: %q", cfg.Storage.ISODir)
	}
	if !filepath.IsAbs(cfg.Storage.VMDiskDir) {
		t.Errorf("VMDiskDir not absolute: %q", cfg.Storage.VMDiskDir)
	}
}

func TestLoad_ReturnsNonNil(t *testing.T) {
	// Load should always return a valid config even with no config file
	t.Setenv("CONFIG_PATH", "/nonexistent/config.yaml")
	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}
	if cfg.Server.Port == "" {
		t.Error("expected default port to be set")
	}
}
