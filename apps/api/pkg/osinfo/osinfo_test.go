package osinfo

import (
	"testing"
)

func TestRegistry_FromOSConfig(t *testing.T) {
	registry := New()

	tests := []struct {
		name    string
		family  OSFamily
		version string
		want    string
	}{
		// Ubuntu
		{
			name:    "Ubuntu 24.04",
			family:  OSFamilyLinux,
			version: "ubuntu-24.04",
			want:    "http://ubuntu.com/ubuntu/24.04",
		},
		{
			name:    "Ubuntu 22.04",
			family:  OSFamilyLinux,
			version: "ubuntu-22.04",
			want:    "http://ubuntu.com/ubuntu/22.04",
		},
		{
			name:    "Ubuntu unversioned",
			family:  OSFamilyLinux,
			version: "ubuntu",
			want:    "http://ubuntu.com/ubuntu/24.04",
		},
		// Debian
		{
			name:    "Debian 13",
			family:  OSFamilyLinux,
			version: "debian-13",
			want:    "http://debian.org/debian/13",
		},
		{
			name:    "Debian 12",
			family:  OSFamilyLinux,
			version: "debian-12",
			want:    "http://debian.org/debian/12",
		},
		{
			name:    "Debian 11",
			family:  OSFamilyLinux,
			version: "debian-11",
			want:    "http://debian.org/debian/11",
		},
		{
			name:    "Debian unversioned",
			family:  OSFamilyLinux,
			version: "debian",
			want:    "http://debian.org/debian/12",
		},
		// Rocky Linux
		{
			name:    "Rocky Linux 9",
			family:  OSFamilyLinux,
			version: "rocky-9",
			want:    "http://rockylinux.org/rocky/9",
		},
		{
			name:    "Rocky Linux 8",
			family:  OSFamilyLinux,
			version: "rocky-8",
			want:    "http://rockylinux.org/rocky/8",
		},
		// AlmaLinux
		{
			name:    "AlmaLinux 9",
			family:  OSFamilyLinux,
			version: "alma-9",
			want:    "http://almalinux.org/almalinux/9",
		},
		// Fedora
		{
			name:    "Fedora 41",
			family:  OSFamilyLinux,
			version: "fedora-41",
			want:    "http://fedoraproject.org/fedora/41",
		},
		{
			name:    "Fedora 40",
			family:  OSFamilyLinux,
			version: "fedora-40",
			want:    "http://fedoraproject.org/fedora/40",
		},
		{
			name:    "Fedora unversioned",
			family:  OSFamilyLinux,
			version: "fedora",
			want:    "http://fedoraproject.org/fedora/40",
		},
		// Alpine
		{
			name:    "Alpine 3.23",
			family:  OSFamilyLinux,
			version: "alpine-3.23",
			want:    "http://alpinelinux.org/alpine/v3.23",
		},
		// Windows
		{
			name:    "Windows 11",
			family:  OSFamilyWindows,
			version: "11",
			want:    "http://microsoft.com/windows/11",
		},
		{
			name:    "Windows 10",
			family:  OSFamilyWindows,
			version: "10",
			want:    "http://microsoft.com/windows/10",
		},
		{
			name:    "Windows Server 2022",
			family:  OSFamilyWindows,
			version: "2022",
			want:    "http://microsoft.com/windows/2022",
		},
		{
			name:    "Windows Server 2019",
			family:  OSFamilyWindows,
			version: "2019",
			want:    "http://microsoft.com/windows/2019",
		},
		{
			name:    "Windows unversioned",
			family:  OSFamilyWindows,
			version: "",
			want:    "http://microsoft.com/windows/11",
		},
		// Solaris
		{
			name:    "Solaris 11",
			family:  OSFamilyOther,
			version: "11",
			want:    "http://oracle.com/solaris/11",
		},
		// Generic fallbacks
		{
			name:    "Unknown Linux",
			family:  OSFamilyLinux,
			version: "arch-linux",
			want:    "http://generic.org/generic",
		},
		{
			name:    "Empty Linux",
			family:  OSFamilyLinux,
			version: "",
			want:    "http://generic.org/generic",
		},
		{
			name:    "Empty Other",
			family:  OSFamilyOther,
			version: "",
			want:    "http://generic.org/generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := registry.FromOSConfig(tt.family, tt.version)
			if got != tt.want {
				t.Errorf("FromOSConfig(%v, %q) = %q, want %q", tt.family, tt.version, got, tt.want)
			}
		})
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := New()

	tests := []struct {
		name      string
		id        string
		wantFound bool
		wantName  string
	}{
		{
			name:      "Ubuntu 24.04 exists",
			id:        "http://ubuntu.com/ubuntu/24.04",
			wantFound: true,
			wantName:  "Ubuntu 24.04 LTS",
		},
		{
			name:      "Windows 11 exists",
			id:        "http://microsoft.com/windows/11",
			wantFound: true,
			wantName:  "Windows 11",
		},
		{
			name:      "Non-existent ID",
			id:        "http://example.com/invalid",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := registry.Get(tt.id)
			if found != tt.wantFound {
				t.Errorf("Get(%q) found = %v, want %v", tt.id, found, tt.wantFound)
			}
			if tt.wantFound && got.Name != tt.wantName {
				t.Errorf("Get(%q) Name = %q, want %q", tt.id, got.Name, tt.wantName)
			}
		})
	}
}

func TestRegistry_GetAll(t *testing.T) {
	registry := New()
	all := registry.GetAll()

	if len(all) == 0 {
		t.Fatal("GetAll() returned empty list")
	}

	// Verify all entries have required fields
	for _, os := range all {
		if os.ID == "" {
			t.Errorf("OS definition missing ID: %+v", os)
		}
		if os.Name == "" {
			t.Errorf("OS definition missing Name: %+v", os)
		}
		if os.Family == "" {
			t.Errorf("OS definition missing Family: %+v", os)
		}
		if os.RecommendedRAM <= 0 {
			t.Errorf("OS definition has invalid RecommendedRAM: %+v", os)
		}
		if os.RecommendedDisk <= 0 {
			t.Errorf("OS definition has invalid RecommendedDisk: %+v", os)
		}
	}
}

func TestOSDefinition_Fields(t *testing.T) {
	registry := New()

	// Test Ubuntu definition
	ubuntu, found := registry.Get("http://ubuntu.com/ubuntu/24.04")
	if !found {
		t.Fatal("Ubuntu 24.04 not found")
	}

	if ubuntu.Family != OSFamilyLinux {
		t.Errorf("Ubuntu Family = %v, want %v", ubuntu.Family, OSFamilyLinux)
	}
	if ubuntu.Vendor != "canonical" {
		t.Errorf("Ubuntu Vendor = %q, want canonical", ubuntu.Vendor)
	}
	if ubuntu.RecommendedRAM < 1024 {
		t.Errorf("Ubuntu RecommendedRAM = %d, want >= 1024", ubuntu.RecommendedRAM)
	}
	if ubuntu.RecommendedDisk < 20 {
		t.Errorf("Ubuntu RecommendedDisk = %d, want >= 20", ubuntu.RecommendedDisk)
	}

	// Test Windows definition
	windows, found := registry.Get("http://microsoft.com/windows/11")
	if !found {
		t.Fatal("Windows 11 not found")
	}

	if windows.Family != OSFamilyWindows {
		t.Errorf("Windows Family = %v, want %v", windows.Family, OSFamilyWindows)
	}
	if windows.Vendor != "microsoft" {
		t.Errorf("Windows Vendor = %q, want microsoft", windows.Vendor)
	}
	if windows.RecommendedRAM < 4096 {
		t.Errorf("Windows RecommendedRAM = %d, want >= 4096", windows.RecommendedRAM)
	}
}

func TestOSFamily_String(t *testing.T) {
	tests := []struct {
		family OSFamily
		want   string
	}{
		{OSFamilyLinux, "linux"},
		{OSFamilyWindows, "windows"},
		{OSFamilyBSD, "bsd"},
		{OSFamilyOther, "other"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.family), func(t *testing.T) {
			if got := tt.family.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}
