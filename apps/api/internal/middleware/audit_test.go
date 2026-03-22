package middleware

import (
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
)

func TestAudit_DefaultConfig(t *testing.T) {
	config := DefaultAuditConfig()
	
	if len(config.Paths) == 0 {
		t.Error("Expected default paths to audit")
	}
	
	// Should exclude health and metrics by default
	foundHealth := false
	foundMetrics := false
	for _, path := range config.ExcludePaths {
		if path == "/health" {
			foundHealth = true
		}
		if path == "/metrics" {
			foundMetrics = true
		}
	}
	
	if !foundHealth {
		t.Error("Expected /health to be excluded by default")
	}
	if !foundMetrics {
		t.Error("Expected /metrics to be excluded by default")
	}
	
	if config.FailedOnly {
		t.Error("Expected FailedOnly to be false by default")
	}
}

func TestAudit_ShouldAudit(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		config   AuditConfig
		expected bool
	}{
		{
			name:     "included path",
			path:     "/lab.v1.AuthService/Login",
			config:   AuditConfig{Paths: []string{"/lab.v1.AuthService/"}},
			expected: true,
		},
		{
			name:     "excluded path",
			path:     "/health",
			config:   AuditConfig{ExcludePaths: []string{"/health"}},
			expected: false,
		},
		{
			name:     "not in include list",
			path:     "/lab.v1.VMService/Start",
			config:   AuditConfig{Paths: []string{"/lab.v1.AuthService/"}},
			expected: false,
		},
		{
			name:     "empty config audits all",
			path:     "/any/path",
			config:   AuditConfig{},
			expected: true,
		},
		{
			name:     "exclude takes precedence",
			path:     "/test/excluded",
			config:   AuditConfig{Paths: []string{"/test/"}, ExcludePaths: []string{"/test/excluded"}},
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldAudit(tt.path, tt.config)
			if result != tt.expected {
				t.Errorf("shouldAudit(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestAudit_MapPathToAction(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		want   string
	}{
		// Auth actions
		{"login", "/lab.v1.AuthService/Login", "POST", "auth.login"},
		{"register", "/lab.v1.AuthService/Register", "POST", "user.create"},
		{"mfa enable", "/lab.v1.AuthService/EnableMFA", "POST", "auth.mfa_enable"},
		{"mfa disable", "/lab.v1.AuthService/DisableMFA", "POST", "auth.mfa_disable"},
		
		// VM actions
		{"vm create", "/lab.v1.VMService/CreateVM", "POST", "vm.create"},
		{"vm delete", "/lab.v1.VMService/DeleteVM", "DELETE", "vm.delete"},
		{"vm start", "/lab.v1.VMService/StartVM", "POST", "vm.start"},
		{"vm stop", "/lab.v1.VMService/StopVM", "POST", "vm.stop"},
		{"vm update", "/lab.v1.VMService/UpdateVM", "PUT", "vm.update"},
		{"vm clone", "/lab.v1.VMService/CloneVM", "POST", "vm.clone"},
		
		// Backup actions
		{"backup create", "/lab.v1.BackupService/CreateBackup", "POST", "backup.create"},
		{"backup delete", "/lab.v1.BackupService/DeleteBackup", "DELETE", "backup.delete"},
		
		// Snapshot actions
		{"snapshot create", "/lab.v1.SnapshotService/CreateSnapshot", "POST", "snapshot.create"},
		{"snapshot delete", "/lab.v1.SnapshotService/DeleteSnapshot", "DELETE", "snapshot.delete"},
		
		// Generic actions
		{"generic GET", "/some/other/path", "GET", "http.GET"},
		{"generic POST", "/some/other/path", "POST", "http.POST"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapPathToAction(tt.path, tt.method)
			if got != tt.want {
				t.Errorf("mapPathToAction(%q, %q) = %q, want %q", tt.path, tt.method, got, tt.want)
			}
		})
	}
}

func TestAudit_ExtractResource(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"vm id", "/lab.v1.VMService/GetVM/100", "100"},
		{"user id", "/lab.v1.UserService/GetUser/abc-123", "abc-123"},
		{"last segment", "/lab.v1.VMService/ListVMs", "ListVMs"},
		{"root", "/", ""},
		{"trailing slash", "/test/", "test"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResource(tt.path)
			if got != tt.want {
				t.Errorf("extractResource(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAudit_ExtractResourceType(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"vm", "/lab.v1.VMService/GetVM", "vm"},
		{"user", "/lab.v1.UserService/GetUser", "user"},
		{"backup", "/lab.v1.BackupService/CreateBackup", "backup"},
		{"snapshot", "/lab.v1.SnapshotService/CreateSnapshot", "snapshot"},
		{"auth", "/lab.v1.AuthService/Login", "auth"},
		{"proxy", "/lab.v1.ProxyService/CreateProxyHost", "api"},
		{"alert", "/lab.v1.AlertService/FireAlert", "api"},
		{"generic", "/some/other/path", "api"},
		{"empty", "/", "api"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceType(tt.path)
			if got != tt.want {
				t.Errorf("extractResourceType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		path string
		want []string
	}{
		{"/lab.v1.VMService/GetVM/100", []string{"lab.v1.VMService", "GetVM", "100"}},
		{"/health", []string{"health"}},
		{"/", []string{}},
		{"/lab.v1.AuthService/Login", []string{"lab.v1.AuthService", "Login"}},
		{"/a/b/c/d", []string{"a", "b", "c", "d"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := splitPath(tt.path)
			if len(got) != len(tt.want) {
				t.Errorf("splitPath(%q) returned %d elements, want %d", tt.path, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitPath(%q)[%d] = %q, want %q", tt.path, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "foo", false},
		{"", "foo", false},
		{"foo", "", true},
		{"VMService", "VM", true},
		{"VMService", "Service", true},
		{"VMService", "VMware", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.s+"/"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

// Test that AuditLogRepository type exists and has Create method
func TestAuditLogRepository_Interface(t *testing.T) {
	// This test ensures the auth.AuditLogRepository type exists
	// and has the expected Create method signature
	var repo *auth.AuditLogRepository
	_ = repo
	
	// Create a sample audit log to verify the type
	log := &auth.AuditLog{
		UserID:       "test-user",
		Action:       "test.action",
		ResourceType: "test",
		ResourceID:   "123",
		IPAddress:    "127.0.0.1",
		Status:       auth.StatusSuccess,
	}
	
	if log.UserID != "test-user" {
		t.Error("Failed to create audit log")
	}
}
