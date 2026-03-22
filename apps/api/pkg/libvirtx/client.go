package libvirtx

import (
	"fmt"
	"log"
	"sync"

	"libvirt.org/go/libvirt"
)

// Domain represents a VM domain
type Domain = libvirt.Domain

// LibvirtClient defines the interface for libvirt operations.
// This interface enables mocking libvirt for testing and supports
// alternative implementations (e.g., remote libvirt, container backends).
type LibvirtClient interface {
	// Connection returns the underlying libvirt connection for direct operations.
	Connection() (*libvirt.Connect, error)
	
	// Connection management
	Connect() error
	Disconnect() error
	IsConnected() bool
	
	// Host information
	GetHostname() (string, error)
	GetVersion() (string, error)
	
	// Domain operations
	ListDomains() ([]Domain, error)
	GetDomainByName(name string) (Domain, error)
	GetDomainByID(id uint32) (Domain, error)
	GetDomainByVMID(vmid int) (Domain, error)
}

// Ensure Client implements LibvirtClient interface
var _ LibvirtClient = (*Client)(nil)

// Client wraps libvirt connection with connection pooling
type Client struct {
	mu        sync.RWMutex
	conn      *libvirt.Connect
	uri       string
	connected bool
}

// Config holds libvirt connection configuration
type Config struct {
	URI string // e.g., "qemu:///system" or "qemu+tcp://host/system"
}

// DefaultConfig returns default libvirt configuration
func DefaultConfig() *Config {
	return &Config{
		URI: "qemu:///system",
	}
}

// NewClient creates a new libvirt client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	client := &Client{
		uri: cfg.URI,
	}

	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %w", err)
	}

	return client, nil
}

// Connect establishes connection to libvirt
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	conn, err := libvirt.NewConnect(c.uri)
	if err != nil {
		return fmt.Errorf("failed to connect to libvirt at %s: %w", c.uri, err)
	}

	c.conn = conn
	c.connected = true
	log.Printf("Connected to libvirt at %s", c.uri)

	return nil
}

// Disconnect closes the libvirt connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	// Close returns (remaining_refs, error)
	_, err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to disconnect from libvirt: %w", err)
	}

	c.connected = false
	c.conn = nil
	log.Println("Disconnected from libvirt")

	return nil
}

// Connection returns the underlying libvirt connection
// This is safe to use concurrently as libvirt handles are thread-safe
func (c *Client) Connection() (*libvirt.Connect, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return nil, fmt.Errorf("not connected to libvirt")
	}

	return c.conn, nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// GetHostname returns the hostname of the libvirt host
func (c *Client) GetHostname() (string, error) {
	conn, err := c.Connection()
	if err != nil {
		return "", err
	}

	hostname, err := conn.GetHostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	return hostname, nil
}

// GetVersion returns the libvirt version
func (c *Client) GetVersion() (string, error) {
	conn, err := c.Connection()
	if err != nil {
		return "", err
	}

	version, err := conn.GetLibVersion()
	if err != nil {
		return "", fmt.Errorf("failed to get libvirt version: %w", err)
	}

	// Format version (e.g., 1009000 -> "1.9.0")
	major := version / 1000000
	minor := (version % 1000000) / 1000
	patch := version % 1000

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}

// ListDomains returns all domains (VMs)
func (c *Client) ListDomains() ([]Domain, error) {
	conn, err := c.Connection()
	if err != nil {
		return nil, err
	}

	// Get all domains (both active and inactive)
	active, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_ACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to list active domains: %w", err)
	}

	inactive, err := conn.ListAllDomains(libvirt.CONNECT_LIST_DOMAINS_INACTIVE)
	if err != nil {
		return nil, fmt.Errorf("failed to list inactive domains: %w", err)
	}

	return append(active, inactive...), nil
}

// GetDomainByName returns a domain by name
func (c *Client) GetDomainByName(name string) (Domain, error) {
	conn, err := c.Connection()
	if err != nil {
		return Domain{}, err
	}

	domain, err := conn.LookupDomainByName(name)
	if err != nil {
		return Domain{}, fmt.Errorf("domain %s not found: %w", name, err)
	}

	return *domain, nil
}

// GetDomainByID returns a domain by ID
func (c *Client) GetDomainByID(id uint32) (Domain, error) {
	conn, err := c.Connection()
	if err != nil {
		return Domain{}, err
	}

	domain, err := conn.LookupDomainById(id)
	if err != nil {
		return Domain{}, fmt.Errorf("domain with ID %d not found: %w", id, err)
	}

	return *domain, nil
}

// GetDomainByVMID returns a domain by our internal VMID (extracted from domain name)
// Domain names follow the pattern "vm-<VMID>" (e.g., "vm-100")
func (c *Client) GetDomainByVMID(vmid int) (Domain, error) {
	name := fmt.Sprintf("vm-%d", vmid)
	return c.GetDomainByName(name)
}
