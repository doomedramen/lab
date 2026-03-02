//go:build integration
// +build integration

package osinfo

import (
	"testing"
)

// TestRegistryIntegration tests the OS registry with real data
func TestRegistryIntegration(t *testing.T) {
	registry := New()

	// Test that we have registered OS definitions
	all := registry.GetAll()
	if len(all) == 0 {
		t.Fatal("GetAll() returned empty list - registry not populated")
	}

	t.Logf("Registry contains %d OS definitions", len(all))

	// Test specific OS lookups
	tests := []struct {
		family  OSFamily
		version string
		wantID  string
	}{
		{OSFamilyLinux, "ubuntu-24.04", "http://ubuntu.com/ubuntu/24.04"},
		{OSFamilyLinux, "debian-12", "http://debian.org/debian/12"},
		{OSFamilyWindows, "11", "http://microsoft.com/windows/11"},
		{OSFamilyLinux, "fedora-41", "http://fedoraproject.org/fedora/41"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			id := registry.FromOSConfig(tt.family, tt.version)
			if id != tt.wantID {
				t.Errorf("FromOSConfig(%v, %q) = %q, want %q", tt.family, tt.version, id, tt.wantID)
			}

			// Verify we can get the OS definition
			def, found := registry.Get(id)
			if !found {
				t.Errorf("Get(%q) returned not found", id)
			}
			if def.ID != tt.wantID {
				t.Errorf("Get(%q) ID = %q, want %q", id, def.ID, tt.wantID)
			}

			t.Logf("OS: %s, Family: %s, RAM: %dMB, Disk: %dGB",
				def.Name, def.Family, def.RecommendedRAM, def.RecommendedDisk)
		})
	}
}

// TestOSMetadataCompleteness tests that all registered OS have complete metadata
func TestOSMetadataCompleteness(t *testing.T) {
	registry := New()
	all := registry.GetAll()

	for _, os := range all {
		t.Run(os.Name, func(t *testing.T) {
			if os.ID == "" {
				t.Error("OS definition missing ID")
			}
			if os.Name == "" {
				t.Error("OS definition missing Name")
			}
			if os.Family == "" {
				t.Error("OS definition missing Family")
			}
			if os.RecommendedRAM <= 0 {
				t.Errorf("OS %s has invalid RecommendedRAM: %d", os.Name, os.RecommendedRAM)
			}
			if os.RecommendedDisk <= 0 {
				t.Errorf("OS %s has invalid RecommendedDisk: %d", os.Name, os.RecommendedDisk)
			}
			if len(os.Architecture) == 0 {
				t.Errorf("OS %s has no architectures listed", os.Name)
			}
		})
	}
}

// TestOSFamilyString tests OS family string conversion
func TestOSFamilyString(t *testing.T) {
	tests := []struct {
		family OSFamily
		want   string
	}{
		{OSFamilyLinux, "linux"},
		{OSFamilyWindows, "windows"},
		{OSFamilyBSD, "bsd"},
		{OSFamilyOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.family.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
