package libvirtx

import (
	"fmt"
	"sync"

	"libvirt.org/go/libvirt"
)

// MockLibvirtClient is a test double for LibvirtClient interface.
// It provides configurable behavior for unit testing without requiring
// a real libvirt connection.
type MockLibvirtClient struct {
	mu sync.RWMutex
	
	// Connection state
	Connected bool
	
	// Host information
	Hostname string
	Version  string
	
	// Domains database (name -> domain)
	Domains map[string]Domain
	
	// Error injection for testing error scenarios
	ListDomainsError       error
	GetDomainByNameError   error
	GetDomainByIDError     error
	GetDomainByVMIDError   error
	GetHostnameError       error
	GetVersionError        error
	ConnectionError        error
	
	// Call tracking for verification
	ListDomainsCalls       int
	GetDomainByNameCalls   map[string]int
	GetDomainByIDCalls     map[uint32]int
	GetDomainByVMIDCalls   map[int]int
}

// Ensure MockLibvirtClient implements LibvirtClient interface
var _ LibvirtClient = (*MockLibvirtClient)(nil)

// NewMockLibvirtClient creates a new mock client with default values.
func NewMockLibvirtClient() *MockLibvirtClient {
	return &MockLibvirtClient{
		Connected:            true,
		Hostname:             "test-host",
		Version:              "10.0.0",
		Domains:              make(map[string]Domain),
		GetDomainByNameCalls: make(map[string]int),
		GetDomainByIDCalls:   make(map[uint32]int),
		GetDomainByVMIDCalls: make(map[int]int),
	}
}

// Connection returns nil for mock (no real connection available)
func (m *MockLibvirtClient) Connection() (*libvirt.Connect, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if !m.Connected {
		return nil, fmt.Errorf("not connected")
	}
	
	// Return nil for mock - tests should not rely on direct connection access
	return nil, nil
}

// Connect sets the connected state
func (m *MockLibvirtClient) Connect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.ConnectionError != nil {
		return m.ConnectionError
	}
	
	m.Connected = true
	return nil
}

// Disconnect clears the connected state
func (m *MockLibvirtClient) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.Connected = false
	return nil
}

// IsConnected returns the connected state
func (m *MockLibvirtClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.Connected
}

// GetHostname returns the configured hostname
func (m *MockLibvirtClient) GetHostname() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.GetHostnameError != nil {
		return "", m.GetHostnameError
	}
	
	return m.Hostname, nil
}

// GetVersion returns the configured version
func (m *MockLibvirtClient) GetVersion() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.GetVersionError != nil {
		return "", m.GetVersionError
	}
	
	return m.Version, nil
}

// ListDomains returns all configured domains
func (m *MockLibvirtClient) ListDomains() ([]Domain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.ListDomainsCalls++
	
	if m.ListDomainsError != nil {
		return nil, m.ListDomainsError
	}
	
	domains := make([]Domain, 0, len(m.Domains))
	for _, domain := range m.Domains {
		domains = append(domains, domain)
	}
	
	return domains, nil
}

// GetDomainByName returns a domain by name
func (m *MockLibvirtClient) GetDomainByName(name string) (Domain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.GetDomainByNameCalls[name]++
	
	if m.GetDomainByNameError != nil {
		return Domain{}, m.GetDomainByNameError
	}
	
	domain, ok := m.Domains[name]
	if !ok {
		return Domain{}, fmt.Errorf("domain %s not found", name)
	}
	
	return domain, nil
}

// GetDomainByID returns a domain by ID
func (m *MockLibvirtClient) GetDomainByID(id uint32) (Domain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.GetDomainByIDCalls[id]++
	
	if m.GetDomainByIDError != nil {
		return Domain{}, m.GetDomainByIDError
	}
	
	// Search by ID
	for _, domain := range m.Domains {
		domainID, err := domain.GetID()
		if err == nil && uint(domainID) == uint(id) {
			return domain, nil
		}
	}
	
	return Domain{}, fmt.Errorf("domain with ID %d not found", id)
}

// GetDomainByVMID returns a domain by VMID (extracted from name pattern "vm-<VMID>")
func (m *MockLibvirtClient) GetDomainByVMID(vmid int) (Domain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := fmt.Sprintf("vm-%d", vmid)
	m.GetDomainByVMIDCalls[vmid]++
	
	if m.GetDomainByVMIDError != nil {
		return Domain{}, m.GetDomainByVMIDError
	}
	
	domain, ok := m.Domains[name]
	if !ok {
		return Domain{}, fmt.Errorf("domain vm-%d not found", vmid)
	}
	
	return domain, nil
}

// AddDomain adds a domain to the mock database
func (m *MockLibvirtClient) AddDomain(name string, domain Domain) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Domains[name] = domain
}

// RemoveDomain removes a domain from the mock database
func (m *MockLibvirtClient) RemoveDomain(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Domains, name)
}

// ClearDomains removes all domains from the mock database
func (m *MockLibvirtClient) ClearDomains() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Domains = make(map[string]Domain)
}

// ResetCallCounts resets all call counters
func (m *MockLibvirtClient) ResetCallCounts() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ListDomainsCalls = 0
	m.GetDomainByNameCalls = make(map[string]int)
	m.GetDomainByIDCalls = make(map[uint32]int)
	m.GetDomainByVMIDCalls = make(map[int]int)
}
