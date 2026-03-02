//go:build integration
// +build integration

package libvirt

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	lv "libvirt.org/go/libvirt"

	"github.com/doomedramen/lab/apps/api/internal/config"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

const (
	alpineVersion  = "3.23.3"
	alpineBase     = "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases"
	testVMName     = "lab-integration-test"
)

// alpineArch maps the host GOARCH to the Alpine release directory and ISO arch
// name. We always match the guest arch to the host so QEMU can use hardware
// acceleration (HVF on macOS, KVM on Linux) instead of slow TCG emulation.
func alpineArch() (string, bool) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64", true
	case "arm64":
		return "aarch64", true
	case "386":
		return "x86", true
	default:
		return "", false
	}
}

// ensureISO downloads the Alpine virt ISO for the host architecture into
// testdata/ if it is not already present. The file is never deleted
// automatically (it is gitignored via testdata/.gitignore).
func ensureISO(t *testing.T) (isoPath string, arch string) {
	t.Helper()

	var ok bool
	arch, ok = alpineArch()
	if !ok {
		t.Skipf("no Alpine release mapping for GOARCH=%s", runtime.GOARCH)
	}

	isoName := fmt.Sprintf("alpine-virt-%s-%s.iso", alpineVersion, arch)
	isoDir := "testdata"
	isoPath = filepath.Join(isoDir, isoName)

	if _, err := os.Stat(isoPath); err == nil {
		t.Logf("ISO found at %s", isoPath)
		return isoPath, arch
	}

	if err := os.MkdirAll(isoDir, 0755); err != nil {
		t.Fatalf("failed to create testdata dir: %v", err)
	}

	url := fmt.Sprintf("%s/%s/%s", alpineBase, arch, isoName)
	t.Logf("Downloading Alpine virt ISO (%s)…", url)

	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("ISO download failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ISO download returned HTTP %d for %s", resp.StatusCode, url)
	}

	// Write to a temp file first; rename on success to avoid partial files.
	tmp, err := os.CreateTemp(isoDir, ".alpine-*.iso.tmp")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		if _, err := os.Stat(tmpPath); err == nil {
			os.Remove(tmpPath)
		}
	}()

	n, err := io.Copy(tmp, resp.Body)
	tmp.Close()
	if err != nil {
		t.Fatalf("failed to write ISO: %v", err)
	}
	if err := os.Rename(tmpPath, isoPath); err != nil {
		t.Fatalf("failed to rename temp ISO: %v", err)
	}

	t.Logf("ISO downloaded: %.1f MB → %s", float64(n)/(1024*1024), isoPath)
	return isoPath, arch
}

// createTestDisk uses qemu-img to create a small qcow2 disk at path.
func createTestDisk(t *testing.T, path string) {
	t.Helper()
	cmd := exec.Command("qemu-img", "create", "-f", "qcow2", path, "1G")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("qemu-img create failed: %v\n%s", err, out)
	}
}

// cleanupDomain destroys and undefines the named domain if it exists.
// It uses DOMAIN_UNDEFINE_NVRAM so OVMF domains are fully removed.
func cleanupDomain(conn *lv.Connect, name string) {
	domain, err := conn.LookupDomainByName(name)
	if err != nil {
		return
	}
	defer domain.Free()

	if state, _, err := domain.GetState(); err == nil {
		if state == lv.DOMAIN_RUNNING {
			domain.DestroyFlags(lv.DOMAIN_DESTROY_DEFAULT)
		}
	}
	domain.UndefineFlags(lv.DOMAIN_UNDEFINE_NVRAM)
}

// TestCreateAndStartVM is an end-to-end integration test that:
//  1. Downloads the Alpine virt ISO for the host architecture (if not cached)
//  2. Creates a VM definition via the libvirt repository
//  3. Starts the VM and confirms it reaches the running state
//  4. Stops the VM and confirms it is stopped
//  5. Cleans up the domain and disk
//
// Run with:
//
//	go test -tags 'libvirt integration' ./internal/repository/libvirt/ -run TestCreateAndStartVM -v
func TestCreateAndStartVM(t *testing.T) {
	isoRelPath, arch := ensureISO(t)

	isoPath, err := filepath.Abs(isoRelPath)
	if err != nil {
		t.Fatalf("failed to resolve ISO path: %v", err)
	}

	// Connect to libvirt user session (no root required).
	client, err := libvirtx.NewClient(&libvirtx.Config{URI: "qemu:///session"})
	if err != nil {
		t.Skipf("libvirt not available: %v", err)
	}
	defer client.Disconnect()

	conn, err := client.Connection()
	if err != nil {
		t.Fatalf("libvirt connection lost: %v", err)
	}

	// Clean up any leftover domain from a previous failed run.
	cleanupDomain(conn, testVMName)

	// Build a test config using a temp directory for the VM disk.
	diskDir := t.TempDir()
	cfg := config.Load()
	cfg.Storage.VMDiskDir = diskDir

	// Pre-create the qcow2 disk so QEMU has a writable target.
	diskPath := filepath.Join(diskDir, testVMName+"."+cfg.VM.DiskFormat)
	createTestDisk(t, diskPath)

	repo := NewVMRepository(client, cfg)

	// Determine machine type and CPU model for this arch — match applyVMDefaults logic.
	machineType := model.MachineTypePC
	bios := model.BIOSTypeSeaBIOS
	cpuModel := "host-passthrough" // KVM on Linux
	if arch == "aarch64" {
		machineType = model.MachineTypeVirt
		bios = model.BIOSTypeOVMF
		cpuModel = "maximum" // works with HVF (macOS) and KVM/TCG
	}

	req := &model.VMCreateRequest{
		Name:        testVMName,
		Node:        "localhost",
		OS:          model.OSConfig{Type: model.OSTypeLinux, Version: "alpine"},
		Arch:        arch,
		MachineType: machineType,
		BIOS:        bios,
		CPUSockets:  1,
		CPUCores:    1,
		CPUModel:    cpuModel,
		Memory:      0.5, // 512 MiB
		Disk:        1,
		ISO:         isoPath,
		Network:     []model.NetworkConfig{{Type: model.NetworkTypeUser, Model: model.NetworkModelVirtio}},
		Agent:       false,
	}

	// Register cleanup before Create so the domain is always removed even on failure.
	// Opens a fresh connection because defer client.Disconnect() runs before t.Cleanup.
	t.Cleanup(func() {
		c, err := libvirtx.NewClient(&libvirtx.Config{URI: "qemu:///session"})
		if err != nil {
			return
		}
		defer c.Disconnect()
		if conn, err := c.Connection(); err == nil {
			cleanupDomain(conn, testVMName)
		}
	})

	created := repo.Create(req)
	if created == nil {
		t.Fatal("Create returned nil — check libvirt logs")
	}
	t.Logf("Domain defined: name=%s arch=%s machine=%s bios=%s",
		created.ID, created.Arch, created.MachineType, created.BIOS)

	// Fetch back via GetByID to get the consistently-resolved VMID
	// (Create uses a timestamp VMID; GetByVMID resolves via domain name).
	vm := repo.GetByID(created.ID)
	if vm == nil {
		t.Fatalf("GetByID(%q) returned nil after Create", created.ID)
	}

	// --- Start ---
	if err := repo.Start(vm.VMID); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Log("Start called — polling for running state…")

	const (
		pollInterval = 500 * time.Millisecond
		pollTimeout  = 30 * time.Second
	)
	deadline := time.Now().Add(pollTimeout)
	var running bool
	for time.Now().Before(deadline) {
		got := repo.GetByID(vm.ID)
		if got != nil && got.Status == model.VMStatusRunning {
			running = true
			break
		}
		time.Sleep(pollInterval)
	}
	if !running {
		t.Errorf("VM did not reach running state within %s", pollTimeout)
	} else {
		t.Log("VM is running ✓")
	}

	// --- Stop ---
	if err := repo.Stop(vm.VMID); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	got := repo.GetByID(vm.ID)
	if got == nil {
		t.Fatal("GetByID returned nil after Stop")
	}
	if got.Status != model.VMStatusStopped {
		t.Errorf("Status after Stop: got %q, want stopped", got.Status)
	} else {
		t.Log("VM is stopped ✓")
	}
}
