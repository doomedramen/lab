package config

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/doomedramen/lab/apps/api/pkg/sysinfo"
)

// BackendType defines the backend to use for data
type BackendType string

const (
	BackendLibvirt BackendType = "libvirt"
)

// Config holds application configuration
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Backend    BackendType      `yaml:"backend"`
	Libvirt    LibvirtConfig    `yaml:"libvirt"`
	Storage    StorageConfig    `yaml:"storage"`
	VM         VMConfig         `yaml:"vm_defaults"`
	Upload     UploadConfig     `yaml:"upload"`
	Logging    LoggingConfig    `yaml:"logging"`
	Auth       AuthConfig       `yaml:"auth"`
	Proxy      ProxyConfig      `yaml:"proxy"`
	Containers ContainersConfig `yaml:"containers"`
	Security   SecurityConfig   `yaml:"security"`
}

// ProxyConfig holds reverse proxy server configuration.
type ProxyConfig struct {
	Enabled        bool   `yaml:"enabled"`
	HTTPPort       int    `yaml:"http_port"`
	HTTPSPort      int    `yaml:"https_port"`
	ACMEEmail      string `yaml:"acme_email"`
	ACMEStorageDir string `yaml:"acme_storage_dir"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port               string `yaml:"port"`
	Env                string `yaml:"env"`
	TLSEnabled         bool   `yaml:"tls_enabled"`
	TLSCertFile        string `yaml:"tls_cert_file"`
	TLSKeyFile         string `yaml:"tls_key_file"`
	MaxRequestBodySize int64  `yaml:"max_request_body_size"` // Max request body size in bytes (default: 10MB)
}

// LibvirtConfig holds libvirt connection settings
type LibvirtConfig struct {
	URI               string   `yaml:"uri"`
	BridgeHelperPaths []string `yaml:"bridge_helper_paths"` // Paths to check for qemu-bridge-helper
}

// ContainersConfig holds LXC container configuration
type ContainersConfig struct {
	RootDir      string `yaml:"root_dir"`      // Base directory for container rootfs
	EmulatorPath string `yaml:"emulator_path"` // Path to libvirt_lxc emulator
}

// StorageConfig holds storage path configuration
type StorageConfig struct {
	DataDir           string   `yaml:"data_dir"`
	ISODir            string   `yaml:"iso_dir"`
	ISODownloadTempDir string  `yaml:"iso_download_temp_dir"`  // Temp directory for ISO downloads
	VMDiskDir         string   `yaml:"vm_disk_dir"`
	MaxISOSize        int64    `yaml:"max_iso_size"`
	AllowedExtensions []string `yaml:"allowed_extensions"`
	StacksDir         string   `yaml:"stacks_dir"` // Directory for Docker Compose stacks (must be set explicitly)
}

// VMConfig holds VM default configuration
type VMConfig struct {
	DiskFormat         string            `yaml:"disk_format"`
	DiskBus            string            `yaml:"disk_bus"`
	EmulatorPath       string            `yaml:"emulator_path"`
	EmulatorPaths      map[string]string `yaml:"emulator_paths"`
	OVMFPath           string            `yaml:"ovmf_path"`             // x86_64 OVMF
	OVMFPathARM64      string            `yaml:"ovmf_path_arm64"`       // aarch64 OVMF/AAVMF
	OVMFSecureBootPath string            `yaml:"ovmf_secure_boot_path"` // x86_64 OVMF Secure Boot variant
	OVMFVarsPath       string            `yaml:"ovmf_vars_path"`        // NVRAM template for secure boot
}

// UploadConfig holds upload (Tus) configuration
type UploadConfig struct {
	ChunkSize   int64  `yaml:"chunk_size"`
	RetryDelays []int  `yaml:"retry_delays"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level            string `yaml:"level"`
	Format           string `yaml:"format"`
	VMLogRetentionDays int    `yaml:"vm_log_retention_days"` // Days to retain VM logs (default: 7)
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret         string        `yaml:"jwt_secret"`
	AccessTokenExpiry string        `yaml:"access_token_expiry"`
	RefreshTokenExpiry string       `yaml:"refresh_token_expiry"`
	Issuer            string        `yaml:"issuer"`
	MFA               MFAConfig     `yaml:"mfa"`
}

// MFAConfig holds MFA configuration
type MFAConfig struct {
	IssuerName       string `yaml:"issuer_name"`
	RequiredForAdmin bool   `yaml:"required_for_admin"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	// AllowedCIDRs is a list of CIDR blocks that are allowed to access the API.
	// If empty, all IPs are allowed. Supports IPv4 and IPv6 CIDRs.
	// Example: ["192.168.1.0/24", "10.0.0.0/8", "::1/128"]
	AllowedCIDRs []string `yaml:"allowed_cidrs"`
}

// Load reads configuration from file and environment variables
func Load() *Config {
	cfg := defaults()

	// Try to load config file
	configPath := getConfigPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			log.Printf("Warning: Failed to parse config file: %v", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Expand paths
	cfg.expandPaths()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Printf("Warning: Configuration validation failed: %v", err)
	}

	return cfg
}

// defaults returns the default configuration
func defaults() *Config {
	sys := sysinfo.New()
	dataBaseDir := sys.DataDir()

	return &Config{
		Server: ServerConfig{
			Port:               "8080",
			Env:                "development",
			MaxRequestBodySize: 10 * 1024 * 1024, // 10 MB default
		},
		Backend: BackendLibvirt,
		Libvirt: LibvirtConfig{
			URI: "qemu:///session",
		},
		Storage: StorageConfig{
			DataDir:            dataBaseDir,
			ISODir:             sys.ISODir(dataBaseDir),
			ISODownloadTempDir: "", // Empty means use system temp (/tmp)
			VMDiskDir:          sys.VMDiskDir(dataBaseDir),
			MaxISOSize:         50 * 1024 * 1024 * 1024, // 50 GB
			AllowedExtensions:  []string{".iso"},
		},
		VM: VMConfig{
			DiskFormat:   "qcow2",
			DiskBus:      "virtio",
			EmulatorPath: "",
			EmulatorPaths: map[string]string{
				"x86_64":  sys.EmulatorPath("x86_64"),
				"aarch64": sys.EmulatorPath("aarch64"),
			},
			OVMFPath:      sys.FirmwarePath("x86_64"),
			OVMFPathARM64: sys.FirmwarePath("aarch64"),
		},
		Upload: UploadConfig{
			ChunkSize:   50 * 1024 * 1024, // 50 MB
			RetryDelays: []int{0, 1000, 3000, 5000},
		},
		Logging: LoggingConfig{
			Level:              "info",
			Format:             "text",
			VMLogRetentionDays: 7, // Default 7 days retention
		},
		Auth: AuthConfig{
			JWTSecret:          "", // MUST be set via config file or JWT_SECRET env var
			AccessTokenExpiry:  "15m",
			RefreshTokenExpiry: "168h", // 7 days
			Issuer:             "lab-api",
			MFA: MFAConfig{
				IssuerName:       "Lab",
				RequiredForAdmin: false,
			},
		},
		Proxy: ProxyConfig{
			Enabled:   true,
			HTTPPort:  80,
			HTTPSPort: 443,
		},
		Containers: ContainersConfig{
			RootDir:      "/var/lib/lxc",
			EmulatorPath: "/usr/lib/libvirt/libvirt_lxc",
		},
		Security: SecurityConfig{
			AllowedCIDRs: []string{}, // Empty means all IPs allowed
		},
	}
}

// Validate checks that required configuration is set
func (c *Config) Validate() error {
	// In production, JWT secret must be set
	if c.Server.Env == "production" && c.Auth.JWTSecret == "" {
		return errors.New("JWT_SECRET must be set in production")
	}

	// TLS validation
	if c.Server.TLSEnabled {
		if c.Server.TLSCertFile == "" {
			return errors.New("tls_cert_file is required when tls_enabled is true")
		}
		if c.Server.TLSKeyFile == "" {
			return errors.New("tls_key_file is required when tls_enabled is true")
		}
		// Check that cert file exists and is readable
		if _, err := os.Stat(c.Server.TLSCertFile); err != nil {
			return fmt.Errorf("cannot access tls_cert_file %q: %w", c.Server.TLSCertFile, err)
		}
		// Check that key file exists and is readable
		if _, err := os.Stat(c.Server.TLSKeyFile); err != nil {
			return fmt.Errorf("cannot access tls_key_file %q: %w", c.Server.TLSKeyFile, err)
		}
	}

	// Validate CIDR format for IP whitelist
	for _, cidr := range c.Security.AllowedCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
	}

	return nil
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	// Check for explicit config path
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// Use sysinfo to get platform-specific config paths
	sys := sysinfo.New()
	for _, path := range sys.ConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return default path (may not exist)
	return "config.yaml"
}

// applyEnvOverrides applies environment variable overrides to the config
func applyEnvOverrides(cfg *Config) {
	// Server
	if port := os.Getenv("PORT"); port != "" {
		cfg.Server.Port = port
	}
	if env := os.Getenv("ENV"); env != "" {
		cfg.Server.Env = env
	}

	// Backend
	if backend := os.Getenv("BACKEND"); backend != "" {
		cfg.Backend = BackendType(backend)
	}

	// Libvirt
	if uri := os.Getenv("LIBVIRT_URI"); uri != "" {
		cfg.Libvirt.URI = uri
	}

	// Storage
	if dir := os.Getenv("ISO_DIR"); dir != "" {
		cfg.Storage.ISODir = dir
	}
	if dir := os.Getenv("VM_DISK_DIR"); dir != "" {
		cfg.Storage.VMDiskDir = dir
	}
	if dir := os.Getenv("STACKS_DIR"); dir != "" {
		cfg.Storage.StacksDir = dir
	}

	// Auth
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.Auth.JWTSecret = secret
	}
	if expiry := os.Getenv("ACCESS_TOKEN_EXPIRY"); expiry != "" {
		cfg.Auth.AccessTokenExpiry = expiry
	}
	if expiry := os.Getenv("REFRESH_TOKEN_EXPIRY"); expiry != "" {
		cfg.Auth.RefreshTokenExpiry = expiry
	}
	if issuer := os.Getenv("JWT_ISSUER"); issuer != "" {
		cfg.Auth.Issuer = issuer
	}
}

// expandPaths expands default paths based on OS and user home directory
func (c *Config) expandPaths() {
	sys := sysinfo.New()
	baseDir := c.Storage.DataDir
	if baseDir == "" {
		baseDir = sys.DataDir()
	}

	// Expand ISO directory
	if c.Storage.ISODir == "" {
		c.Storage.ISODir = sys.ISODir(baseDir)
	}

	// Expand VM disk directory
	if c.Storage.VMDiskDir == "" {
		c.Storage.VMDiskDir = sys.VMDiskDir(baseDir)
	}

	// Ensure paths are absolute
	if !filepath.IsAbs(c.Storage.ISODir) {
		if abs, err := filepath.Abs(c.Storage.ISODir); err == nil {
			c.Storage.ISODir = abs
		}
	}
	if !filepath.IsAbs(c.Storage.VMDiskDir) {
		if abs, err := filepath.Abs(c.Storage.VMDiskDir); err == nil {
			c.Storage.VMDiskDir = abs
		}
	}
}

// GetOVMFPathForArch returns the OVMF/AAVMF firmware path for the given guest
// architecture, using config overrides where set and auto-detected paths otherwise.
func (c *Config) GetOVMFPathForArch(arch string) string {
	switch arch {
	case "aarch64":
		if c.VM.OVMFPathARM64 != "" {
			return c.VM.OVMFPathARM64
		}
	default: // x86_64
		if c.VM.OVMFPath != "" {
			return c.VM.OVMFPath
		}
	}
	return sysinfo.New().FirmwarePath(arch)
}

// GetOVMFSecureBootPathForArch returns the OVMF Secure Boot firmware path for the given guest architecture.
// For x86_64, this typically points to OVMF_CODE.secboot.fd. For aarch64, Secure Boot is not commonly supported.
func (c *Config) GetOVMFSecureBootPathForArch(arch string) string {
	// Secure Boot is primarily for x86_64
	if arch != "aarch64" {
		if c.VM.OVMFSecureBootPath != "" {
			return c.VM.OVMFSecureBootPath
		}
		// Common paths for secure boot firmware on Linux
		commonPaths := []string{
			"/usr/share/OVMF/OVMF_CODE.secboot.fd",
			"/usr/share/qemu/OVMF_CODE.secboot.fd",
			"/usr/share/edk2/ovmf/OVMF_CODE.secboot.fd",
		}
		for _, p := range commonPaths {
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	// Fall back to regular OVMF path
	return c.GetOVMFPathForArch(arch)
}

// GetEmulatorPath returns the emulator path for the given architecture.
func (c *Config) GetEmulatorPath(arch string) string {
	if c.VM.EmulatorPath != "" {
		return c.VM.EmulatorPath
	}
	if path, ok := c.VM.EmulatorPaths[arch]; ok {
		return path
	}
	return sysinfo.New().EmulatorPath(arch)
}

// GetBridgeHelperPaths returns the paths to check for qemu-bridge-helper.
// Uses config-provided paths if set, otherwise falls back to auto-detected defaults.
func (c *Config) GetBridgeHelperPaths() []string {
	if len(c.Libvirt.BridgeHelperPaths) > 0 {
		return c.Libvirt.BridgeHelperPaths
	}
	// Default paths to check
	return []string{
		"/usr/lib/qemu/qemu-bridge-helper",
		"/usr/libexec/qemu-bridge-helper",
		"/usr/lib/qemu-bridge-helper",
	}
}

// GetOVMFVarsPath returns the NVRAM template path for secure boot.
// Uses config-provided path if set, otherwise falls back to auto-detected defaults.
func (c *Config) GetOVMFVarsPath() string {
	if c.VM.OVMFVarsPath != "" {
		return c.VM.OVMFVarsPath
	}
	// Common paths for OVMF VARS template on Linux
	commonPaths := []string{
		"/usr/share/OVMF/OVMF_VARS.secboot.fd",
		"/usr/share/qemu/OVMF_VARS.secboot.fd",
		"/usr/share/edk2/ovmf/OVMF_VARS.secboot.fd",
	}
	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// GetContainersRootDir returns the LXC containers root directory.
func (c *Config) GetContainersRootDir() string {
	if c.Containers.RootDir != "" {
		return c.Containers.RootDir
	}
	return "/var/lib/lxc"
}

// GetContainersEmulatorPath returns the path to libvirt_lxc emulator.
func (c *Config) GetContainersEmulatorPath() string {
	if c.Containers.EmulatorPath != "" {
		return c.Containers.EmulatorPath
	}
	return "/usr/lib/libvirt/libvirt_lxc"
}

// EnsureDirectories creates the necessary directories if they don't exist
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Storage.DataDir,
		c.Storage.ISODir,
		c.Storage.VMDiskDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// IsIPWhitelistEnabled returns true if IP whitelisting is configured.
func (c *Config) IsIPWhitelistEnabled() bool {
	return len(c.Security.AllowedCIDRs) > 0
}

// ParseAllowedNetworks parses the allowed CIDRs into net.IPNet slices.
// Returns an error if any CIDR is invalid.
func (c *Config) ParseAllowedNetworks() ([]*net.IPNet, error) {
	if len(c.Security.AllowedCIDRs) == 0 {
		return nil, nil
	}

	networks := make([]*net.IPNet, 0, len(c.Security.AllowedCIDRs))
	for _, cidr := range c.Security.AllowedCIDRs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
		}
		networks = append(networks, network)
	}
	return networks, nil
}

// IsIPAllowed checks if the given IP address is in the allowed CIDRs.
// If no CIDRs are configured, all IPs are allowed.
func (c *Config) IsIPAllowed(ipStr string) bool {
	if !c.IsIPWhitelistEnabled() {
		return true
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	networks, err := c.ParseAllowedNetworks()
	if err != nil {
		// If parsing fails, deny access (fail closed)
		return false
	}

	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// String returns a string representation of the config (for logging)
func (c *Config) String() string {
	return fmt.Sprintf("Config{Port: %s, Backend: %s, ISODir: %s, VMDiskDir: %s}",
		c.Server.Port, c.Backend, c.Storage.ISODir, c.Storage.VMDiskDir)
}

// GetMaxRequestBodySize returns the maximum request body size in bytes.
// If not set or invalid, returns a safe default of 10MB.
func (c *Config) GetMaxRequestBodySize() int64 {
	if c.Server.MaxRequestBodySize <= 0 {
		return 10 * 1024 * 1024 // 10 MB default
	}
	// Cap at 100MB to prevent unreasonable values
	const maxSize = 100 * 1024 * 1024
	if c.Server.MaxRequestBodySize > maxSize {
		return maxSize
	}
	return c.Server.MaxRequestBodySize
}
