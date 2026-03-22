package libvirt

import (
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestMapOSToLibosinfo(t *testing.T) {
	tests := []struct {
		name  string
		input model.OSConfig
		want  string
	}{
		// Linux variants
		{
			name:  "Ubuntu 24.04",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-24.04"},
			want:  "http://ubuntu.com/ubuntu/24.04",
		},
		{
			name:  "Ubuntu 22.04",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-22.04"},
			want:  "http://ubuntu.com/ubuntu/22.04",
		},
		{
			name:  "Ubuntu 20.04",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-20.04"},
			want:  "http://ubuntu.com/ubuntu/20.04",
		},
		{
			name:  "Ubuntu unversioned falls back to 24.04",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu"},
			want:  "http://ubuntu.com/ubuntu/24.04",
		},
		{
			name:  "Debian 12",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "debian-12"},
			want:  "http://debian.org/debian/12",
		},
		{
			name:  "Debian 11",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "debian-11"},
			want:  "http://debian.org/debian/11",
		},
		{
			name:  "Debian unversioned falls back to 12",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "debian"},
			want:  "http://debian.org/debian/12",
		},
		{
			name:  "Rocky Linux 9",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "rocky-9"},
			want:  "http://rockylinux.org/rocky/9",
		},
		{
			name:  "Rocky Linux 8",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "rocky-8"},
			want:  "http://rockylinux.org/rocky/8",
		},
		{
			name:  "AlmaLinux 9",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "alma-9"},
			want:  "http://almalinux.org/almalinux/9",
		},
		{
			name:  "Fedora 40",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "fedora-40"},
			want:  "http://fedoraproject.org/fedora/40",
		},
		{
			name:  "Fedora unversioned falls back to 40",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "fedora"},
			want:  "http://fedoraproject.org/fedora/40",
		},
		{
			name:  "Unknown Linux version falls back to generic",
			input: model.OSConfig{Type: model.OSTypeLinux, Version: "arch-linux"},
			want:  "http://generic.org/generic",
		},
		// Windows variants
		{
			name:  "Windows 11",
			input: model.OSConfig{Type: model.OSTypeWindows, Version: "11"},
			want:  "http://microsoft.com/windows/11",
		},
		{
			name:  "Windows 10",
			input: model.OSConfig{Type: model.OSTypeWindows, Version: "10"},
			want:  "http://microsoft.com/windows/10",
		},
		{
			name:  "Windows Server 2022",
			input: model.OSConfig{Type: model.OSTypeWindows, Version: "2022"},
			want:  "http://microsoft.com/windows/2022",
		},
		{
			name:  "Windows Server 2019",
			input: model.OSConfig{Type: model.OSTypeWindows, Version: "2019"},
			want:  "http://microsoft.com/windows/2019",
		},
		{
			name:  "Windows unversioned falls back to 11",
			input: model.OSConfig{Type: model.OSTypeWindows, Version: ""},
			want:  "http://microsoft.com/windows/11",
		},
		// Solaris
		{
			name:  "Solaris",
			input: model.OSConfig{Type: model.OSTypeSolaris, Version: "11"},
			want:  "http://oracle.com/solaris/11",
		},
		// Other / unknown
		{
			name:  "Other type falls back to generic",
			input: model.OSConfig{Type: model.OSTypeOther, Version: ""},
			want:  "http://generic.org/generic",
		},
		{
			name:  "Empty config falls back to generic",
			input: model.OSConfig{},
			want:  "http://generic.org/generic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapOSToLibosinfo(tt.input)
			if got != tt.want {
				t.Errorf("mapOSToLibosinfo(%+v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
