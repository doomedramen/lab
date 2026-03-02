package service

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/osinfo"
	"github.com/doomedramen/lab/apps/api/pkg/sysinfo"
	libvirt "libvirt.org/go/libvirt"
	libvirtxml "libvirt.org/go/libvirtxml"
)

var (
	ErrVMNotFound       = errors.New("VM not found")
	ErrVMAlreadyRunning = errors.New("VM is already running")
	ErrVMAlreadyStopped = errors.New("VM is already stopped")
	ErrVMInvalidState   = errors.New("VM is in an invalid state for this action")
)

// VMTemplate represents a pre-configured VM template
type VMTemplate struct {
	ID          string
	Name        string
	Description string
	Icon        string
	// ISO URLs per architecture (empty string = not supported for that arch)
	ISOURLx86_64  string // URL for x86_64/amd64
	ISOURLaarch64 string // URL for aarch64/arm64
	ISOName       string
	CPUCores      int
	MemoryGB      float64
	DiskGB        float64
	OS            model.OSConfig
}

// GetISOURLForArch returns the ISO URL for the specified architecture
func (t *VMTemplate) GetISOURLForArch(arch string) string {
	switch arch {
	case "x86_64":
		return t.ISOURLx86_64
	case "aarch64":
		return t.ISOURLaarch64
	default:
		return ""
	}
}

// SupportsArch returns true if the template supports the specified architecture
func (t *VMTemplate) SupportsArch(arch string) bool {
	return t.GetISOURLForArch(arch) != ""
}

// GetOSMetadata returns OS metadata from the osinfo registry
func (t *VMTemplate) GetOSMetadata() (osinfo.OSDefinition, bool) {
	registry := osinfo.New()
	
	var family osinfo.OSFamily
	switch t.OS.Type {
	case model.OSTypeLinux:
		family = osinfo.OSFamilyLinux
	case model.OSTypeWindows:
		family = osinfo.OSFamilyWindows
	default:
		family = osinfo.OSFamilyOther
	}
	
	libosinfoID := registry.FromOSConfig(family, t.OS.Version)
	return registry.Get(libosinfoID)
}

// VMTemplates returns available VM templates with explicit URLs per architecture
func VMTemplates() []VMTemplate {
	return []VMTemplate{
		{
			ID:            "alpine-virt",
			Name:          "Alpine Linux Virtual",
			Description:   "Alpine Linux 3.23 Virtual Edition - Small, fast, perfect for testing",
			Icon:          "🏔️",
			ISOURLx86_64:  "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/x86_64/alpine-virt-3.23.3-x86_64.iso",
			ISOURLaarch64: "https://dl-cdn.alpinelinux.org/alpine/v3.23/releases/aarch64/alpine-virt-3.23.3-aarch64.iso",
			ISOName:       "alpine-virt-3.23.3-${arch}.iso",
			CPUCores:      1,
			MemoryGB:      1,
			DiskGB:        10,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "alpine-3.23"},
		},
		{
			ID:            "ubuntu-24.04",
			Name:          "Ubuntu 24.04 LTS",
			Description:   "Ubuntu 24.04.4 LTS (Noble Numbat) - Latest LTS release",
			Icon:          "🐧",
			ISOURLx86_64:  "https://releases.ubuntu.com/noble/ubuntu-24.04.4-live-server-amd64.iso",
			ISOURLaarch64: "", // Not available on releases.ubuntu.com
			ISOName:       "ubuntu-24.04.4-live-server-amd64.iso",
			CPUCores:      2,
			MemoryGB:      4,
			DiskGB:        40,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-24.04"},
		},
		{
			ID:            "ubuntu-22.04",
			Name:          "Ubuntu 22.04 LTS",
			Description:   "Ubuntu 22.04.5 LTS (Jammy Jellyfish)",
			Icon:          "🐧",
			ISOURLx86_64:  "https://releases.ubuntu.com/jammy/ubuntu-22.04.5-live-server-amd64.iso",
			ISOURLaarch64: "", // Not available on releases.ubuntu.com
			ISOName:       "ubuntu-22.04.5-live-server-amd64.iso",
			CPUCores:      2,
			MemoryGB:      4,
			DiskGB:        40,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "ubuntu-22.04"},
		},
		{
			ID:            "debian-13",
			Name:          "Debian 13",
			Description:   "Debian 13 (Trixie) - Latest stable release",
			Icon:          "🐧",
			ISOURLx86_64:  "https://cdimage.debian.org/debian-cd/current/amd64/iso-cd/debian-13.3.0-amd64-netinst.iso",
			ISOURLaarch64: "https://cdimage.debian.org/debian-cd/current/arm64/iso-cd/debian-13.3.0-arm64-netinst.iso",
			ISOName:       "debian-13.3.0-${arch}-netinst.iso",
			CPUCores:      2,
			MemoryGB:      2,
			DiskGB:        20,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "debian-13"},
		},
		{
			ID:            "rocky-9",
			Name:          "Rocky Linux 9",
			Description:   "Rocky Linux 9.7 - RHEL compatible",
			Icon:          "🎩",
			ISOURLx86_64:  "https://download.rockylinux.org/pub/rocky/9/isos/x86_64/Rocky-9.7-x86_64-dvd.iso",
			ISOURLaarch64: "https://download.rockylinux.org/pub/rocky/9/isos/aarch64/Rocky-9.7-aarch64-dvd.iso",
			ISOName:       "Rocky-9.7-${arch}-dvd.iso",
			CPUCores:      2,
			MemoryGB:      4,
			DiskGB:        40,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "rocky-9"},
		},
		{
			ID:            "almalinux-9",
			Name:          "AlmaLinux 9",
			Description:   "AlmaLinux 9 - RHEL compatible (latest)",
			Icon:          "🎩",
			// AlmaLinux uses 'latest' symlink for most recent release
			ISOURLx86_64:  "https://repo.almalinux.org/almalinux/9/isos/x86_64/AlmaLinux-9-latest-x86_64-dvd.iso",
			ISOURLaarch64: "https://repo.almalinux.org/almalinux/9/isos/aarch64/AlmaLinux-9-latest-aarch64-dvd.iso",
			ISOName:       "AlmaLinux-9-latest-${arch}-dvd.iso",
			CPUCores:      2,
			MemoryGB:      4,
			DiskGB:        40,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "almalinux-9"},
		},
		{
			ID:            "fedora-41",
			Name:          "Fedora 41",
			Description:   "Fedora 41 - Latest stable release",
			Icon:          "🎩",
			ISOURLx86_64:  "https://download.fedoraproject.org/pub/fedora/linux/releases/41/Server/x86_64/iso/Fedora-Server-dvd-x86_64-41-1.4.iso",
			ISOURLaarch64: "https://download.fedoraproject.org/pub/fedora/linux/releases/41/Server/aarch64/iso/Fedora-Server-dvd-aarch64-41-1.4.iso",
			ISOName:       "Fedora-Server-dvd-${arch}-41-1.4.iso",
			CPUCores:      2,
			MemoryGB:      4,
			DiskGB:        40,
			OS:            model.OSConfig{Type: model.OSTypeLinux, Version: "fedora-41"},
		},
		{
			ID:            "windows-11",
			Name:          "Windows 11",
			Description:   "Windows 11 - Requires license",
			Icon:          "🪟",
			ISOURLx86_64:  "", // User must provide ISO
			ISOURLaarch64: "", // User must provide ISO
			ISOName:       "Windows11.iso",
			CPUCores:      4,
			MemoryGB:      8,
			DiskGB:        80,
			OS:            model.OSConfig{Type: model.OSTypeWindows, Version: "11"},
		},
		{
			ID:            "windows-server-2022",
			Name:          "Windows Server 2022",
			Description:   "Windows Server 2022 - Requires license",
			Icon:          "🪟",
			ISOURLx86_64:  "", // User must provide ISO
			ISOURLaarch64: "", // User must provide ISO
			ISOName:       "WindowsServer2022.iso",
			CPUCores:      4,
			MemoryGB:      8,
			DiskGB:        100,
			OS:            model.OSConfig{Type: model.OSTypeWindows, Version: "2022"},
		},
	}
}

// ConsoleToken holds the VNC port and metadata for a one-time console token.
type ConsoleToken struct {
	VMID      int
	Port      int
	CreatedAt time.Time
}

// VMService provides business logic for VM operations
type VMService struct {
	repo            repository.VMRepository
	isoRepo         repository.ISORepository
	logRepo         *sqlite.VMLogRepository
	taskSvc         *TaskService
	guestAgentRepo  repository.GuestAgentRepository
	pciRepo         repository.PCIRepository
	tokensMu        sync.Mutex
	consoleTokens   map[string]ConsoleToken
	logRetention    int // days
}

// NewVMService creates a new VM service
func NewVMService(repo repository.VMRepository, isoRepo repository.ISORepository, logRepo *sqlite.VMLogRepository, taskSvc *TaskService, logRetentionDays int) *VMService {
	svc := &VMService{
		repo:          repo,
		isoRepo:       isoRepo,
		logRepo:       logRepo,
		taskSvc:       taskSvc,
		logRetention:  logRetentionDays,
		consoleTokens: make(map[string]ConsoleToken),
	}
	go svc.cleanupExpiredTokens()
	go svc.cleanupOldLogs()
	return svc
}

// WithGuestAgentRepo sets the guest agent repository for the VM service
func (s *VMService) WithGuestAgentRepo(repo repository.GuestAgentRepository) *VMService {
	s.guestAgentRepo = repo
	return s
}

// WithPCIRepo sets the PCI device repository for the VM service
func (s *VMService) WithPCIRepo(repo repository.PCIRepository) *VMService {
	s.pciRepo = repo
	return s
}

// ListPCIDevices returns all PCI devices on the host
func (s *VMService) ListPCIDevices(ctx context.Context) ([]model.PCIDevice, bool, bool, error) {
	if s.pciRepo == nil {
		return nil, false, false, errors.New("PCI repository not configured")
	}

	devices, err := s.pciRepo.ListHostDevices(ctx)
	if err != nil {
		return nil, false, false, err
	}

	iommuAvailable := s.pciRepo.IsIOMMUAvailable()
	vfioAvailable := s.pciRepo.IsVFIOAvailable()

	return devices, iommuAvailable, vfioAvailable, nil
}

// cleanupExpiredTokens periodically removes expired console tokens.
func (s *VMService) cleanupExpiredTokens() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		s.tokensMu.Lock()
		for token, ct := range s.consoleTokens {
			if time.Since(ct.CreatedAt) > 30*time.Second {
				delete(s.consoleTokens, token)
			}
		}
		s.tokensMu.Unlock()
	}
}

// cleanupOldLogs periodically removes VM logs older than the retention period.
func (s *VMService) cleanupOldLogs() {
	if s.logRetention <= 0 {
		return // Retention disabled
	}

	ticker := time.NewTicker(1 * time.Hour) // Run every hour
	defer ticker.Stop()
	for range ticker.C {
		ctx := context.Background()
		deleted, err := s.logRepo.DeleteOld(ctx, s.logRetention)
		if err != nil {
			slog.Warn("Failed to cleanup old VM logs", "error", err)
		} else if deleted > 0 {
			slog.Info("Cleaned up old VM logs", "deleted", deleted)
		}
	}
}

// GetConsoleToken generates a one-time token for VNC console access.
// Returns the token and the VNC port.
func (s *VMService) GetConsoleToken(ctx context.Context, vmid int) (string, error) {
	port, err := s.repo.GetVNCPort(ctx, vmid)
	if err != nil {
		return "", err
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)

	s.tokensMu.Lock()
	s.consoleTokens[token] = ConsoleToken{
		VMID:      vmid,
		Port:      port,
		CreatedAt: time.Now(),
	}
	s.tokensMu.Unlock()

	return token, nil
}

// ValidateConsoleToken validates and consumes a one-time console token.
// Returns the token data and true if valid, or zero-value and false if invalid/expired.
func (s *VMService) ValidateConsoleToken(token string) (ConsoleToken, bool) {
	s.tokensMu.Lock()
	defer s.tokensMu.Unlock()

	ct, ok := s.consoleTokens[token]
	if !ok {
		return ConsoleToken{}, false
	}

	// Tokens expire after 30 seconds
	if time.Since(ct.CreatedAt) > 30*time.Second {
		delete(s.consoleTokens, token)
		return ConsoleToken{}, false
	}

	// One-time use — delete after validation
	delete(s.consoleTokens, token)
	return ct, true
}

// GetAll returns all VMs
func (s *VMService) GetAll(ctx context.Context) ([]*model.VM, error) {
	return s.repo.GetAll(ctx)
}

// GetByNode returns VMs filtered by node
func (s *VMService) GetByNode(ctx context.Context, node string) ([]*model.VM, error) {
	return s.repo.GetByNode(ctx, node)
}

// GetByID returns a VM by ID
func (s *VMService) GetByID(ctx context.Context, id string) (*model.VM, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByVMID returns a VM by numeric VMID
func (s *VMService) GetByVMID(ctx context.Context, vmid int) (*model.VM, error) {
	return s.repo.GetByVMID(ctx, vmid)
}

// GetGuestNetworkInterfaces retrieves network interfaces from the QEMU guest agent
func (s *VMService) GetGuestNetworkInterfaces(ctx context.Context, vmid int) ([]model.GuestNetworkInterface, bool, error) {
	if s.guestAgentRepo == nil {
		return nil, false, errors.New("guest agent repository not configured")
	}

	// Check if agent is responsive
	connected := s.guestAgentRepo.Ping(ctx, vmid)
	if !connected {
		return nil, false, nil
	}

	interfaces, err := s.guestAgentRepo.GetNetworkInterfaces(ctx, vmid)
	if err != nil {
		return nil, true, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	return interfaces, true, nil
}

// GetGuestAgentStatus checks if the guest agent is connected for a VM
func (s *VMService) GetGuestAgentStatus(ctx context.Context, vmid int) bool {
	if s.guestAgentRepo == nil {
		return false
	}
	return s.guestAgentRepo.Ping(ctx, vmid)
}

// GetTemplates returns available VM templates
func (s *VMService) GetTemplates() []VMTemplate {
	return VMTemplates()
}

// Clone creates a clone of an existing VM.
// It runs in a background goroutine with progress tracking.
func (s *VMService) Clone(ctx context.Context, req *model.VMCloneRequest) (*model.VM, string, error) {
	// Validate source VM exists
	_, err := s.repo.GetByVMID(ctx, req.SourceVMID)
	if err != nil {
		return nil, "", fmt.Errorf("source VM %d not found: %w", req.SourceVMID, err)
	}

	// Validate name is provided
	if strings.TrimSpace(req.Name) == "" {
		return nil, "", fmt.Errorf("clone name is required")
	}

	// If no task service, run synchronously
	if s.taskSvc == nil {
		vm, err := s.repo.Clone(ctx, req, nil)
		if err != nil {
			return nil, "", err
		}
		return vm, "", nil
	}

	// Start task for tracking
	task, err := s.taskSvc.Start(ctx, model.TaskTypeClone, model.ResourceTypeVM, fmt.Sprintf("vm/%d", req.SourceVMID),
		fmt.Sprintf("Cloning VM %d to '%s'", req.SourceVMID, req.Name))
	if err != nil {
		return nil, "", fmt.Errorf("failed to start clone task: %w", err)
	}

	// Run clone in background
	go func() {
		bgCtx := context.Background()

		progressFunc := func(progress int, message string) {
			s.taskSvc.Progress(bgCtx, task.ID, progress, message)
		}

		clonedVM, err := s.repo.Clone(bgCtx, req, progressFunc)
		if err != nil {
			s.taskSvc.Fail(bgCtx, task.ID, err.Error())
			return
		}

		s.taskSvc.Complete(bgCtx, task.ID)
		slog.Info("VM cloned successfully",
			"source_vmid", req.SourceVMID,
			"clone_vmid", clonedVM.VMID,
			"clone_name", clonedVM.Name,
			"task_id", task.ID)
	}()

	return nil, task.ID, nil
}

// GetTaskService returns the task service for external access
func (s *VMService) GetTaskService() *TaskService {
	return s.taskSvc
}

// Create creates a new VM
func (s *VMService) Create(ctx context.Context, req *model.VMCreateRequest) (*model.VM, error) {
	slog.Info("Creating VM",
		"name", req.Name,
		"node", req.Node,
		"arch", req.Arch,
		"iso", req.ISO,
		"cpu", req.CPUCores,
		"memory_gb", req.Memory,
		"disk_gb", req.Disk,
	)

	applyVMDefaults(req)
	vm, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}
	slog.Info("VM created", "name", vm.Name, "vmid", vm.VMID)
	return vm, nil
}

// applyVMDefaults fills in smart defaults for missing VMCreateRequest fields.
// It is called before the request is handed to the repository so that all
// layers below can assume the fields are fully populated.
func applyVMDefaults(req *model.VMCreateRequest) {
	sys := sysinfo.New()

	// Arch: default to host architecture
	if req.Arch == "" {
		req.Arch = sys.HostArch()
	}

	// CPU defaults
	if req.CPUSockets == 0 {
		req.CPUSockets = 1
	}
	if req.CPUCores == 0 {
		req.CPUCores = 1
	}
	// Machine type defaults based on arch then OS
	isWindows := req.OS.Type == model.OSTypeWindows
	isGuestAArch64 := req.Arch == "aarch64"

	if req.CPUModel == "" {
		req.CPUModel = sys.DefaultCPUModel(req.Arch)
	}
	if req.MachineType == "" {
		switch {
		case isGuestAArch64:
			req.MachineType = model.MachineTypeVirt
		case isWindows:
			req.MachineType = model.MachineTypeQ35
		default:
			req.MachineType = model.MachineTypePC
		}
	}

	// BIOS defaults: aarch64 and Windows both need OVMF/EFI
	if req.BIOS == "" {
		if isGuestAArch64 || isWindows {
			req.BIOS = model.BIOSTypeOVMF
		} else {
			req.BIOS = model.BIOSTypeSeaBIOS
		}
	}

	// Network: use platform-specific defaults
	// - Linux: bridge networking with vmbr0
	// - macOS: vmnet-shared networking (no interface name needed)
	// - Other: user-mode networking (NAT)
	if len(req.Network) == 0 {
		req.Network = []model.NetworkConfig{
			{
				Type:   model.NetworkType(sys.DefaultNetworkType()),
				Bridge: sys.DefaultBridgeName(),
				Model:  model.NetworkModelVirtio,
			},
		}
	}

	// OS type defaults
	if req.OS.Type == "" {
		req.OS.Type = model.OSTypeOther
	}
}

// Update updates an existing VM
func (s *VMService) Update(ctx context.Context, vmid int, req *model.VMUpdateRequest) (*model.VM, error) {
	return s.repo.Update(ctx, vmid, req)
}

// Delete removes a VM. The VM must be stopped first.
func (s *VMService) Delete(ctx context.Context, vmid int) error {
	vm, err := s.repo.GetByVMID(ctx, vmid)
	if err != nil {
		return ErrVMNotFound
	}
	if vm.Status == model.VMStatusRunning || vm.Status == model.VMStatusPaused {
		return ErrVMInvalidState
	}
	return s.repo.Delete(ctx, vmid)
}

// Start starts a VM
func (s *VMService) Start(ctx context.Context, vmid int) error {
	return s.repo.Start(ctx, vmid)
}

// Stop stops a VM (force power off)
func (s *VMService) Stop(ctx context.Context, vmid int) error {
	return s.repo.Stop(ctx, vmid)
}

// Shutdown gracefully shuts down a VM with timeout and fallback to force stop
func (s *VMService) Shutdown(ctx context.Context, vmid int) error {
	// First try graceful shutdown
	err := s.repo.Shutdown(ctx, vmid)
	if err != nil {
		return err
	}

	// Wait for VM to shut down (max 30 seconds)
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)

		vm, err := s.GetByVMID(ctx, vmid)
		if err != nil {
			// VM might have been deleted
			return nil
		}

		if vm.Status == model.VMStatusStopped {
			// Successfully shut down
			return nil
		}
	}

	// Timeout - force stop
	slog.Warn("Graceful shutdown timed out, forcing stop", "vmid", vmid)
	return s.repo.Stop(ctx, vmid)
}

// Pause pauses a VM
func (s *VMService) Pause(ctx context.Context, vmid int) error {
	return s.repo.Pause(ctx, vmid)
}

// Resume resumes a paused VM
func (s *VMService) Resume(ctx context.Context, vmid int) error {
	return s.repo.Resume(ctx, vmid)
}

// Reboot reboots a VM
func (s *VMService) Reboot(ctx context.Context, vmid int) error {
	return s.repo.Reboot(ctx, vmid)
}

// VMLogEntry represents a single log entry
type VMLogEntry struct {
	ID        string            `json:"id"`
	VMID      int               `json:"vmid"`
	Level     string            `json:"level"`
	Timestamp string            `json:"timestamp"`
	Source    string            `json:"source"`
	Message   string            `json:"message"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// GetVMLogs retrieves logs for a VM from the database
func (s *VMService) GetVMLogs(ctx context.Context, vmid int, limit int) ([]*VMLogEntry, error) {
	// First check if VM exists
	vm, err := s.GetByVMID(ctx, vmid)
	if err != nil {
		return nil, ErrVMNotFound
	}

	// Query logs from database
	entries, err := s.logRepo.Query(ctx, vmid, limit)
	if err != nil {
		return nil, err
	}

	// Convert to service log entries
	var result []*VMLogEntry
	for _, e := range entries {
		result = append(result, &VMLogEntry{
			ID:        fmt.Sprintf("log-%d-%d", vmid, e.ID),
			VMID:      e.VMID,
			Level:     e.Level,
			Timestamp: time.Unix(e.CreatedAt, 0).Format(time.RFC3339),
			Source:    e.Source,
			Message:   e.Message,
			Metadata:  e.Metadata,
		})
	}

	// If no logs in database, generate synthetic logs from current VM state
	if len(result) == 0 {
		result = s.generateSyntheticLogs(vm, func(level, source, message string) *VMLogEntry {
			return &VMLogEntry{
				ID:        fmt.Sprintf("synthetic-%d", len(result)),
				VMID:      vmid,
				Level:     level,
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    source,
				Message:   message,
				Metadata:  map[string]string{"vm_name": vm.Name, "synthetic": "true"},
			}
		})
	}

	return result, nil
}

// IngestVMLogs ingests log entries for a VM into the database
func (s *VMService) IngestVMLogs(vmid int, entries []*VMLogEntry) error {
	if s.logRepo == nil {
		return nil // No log repository configured
	}

	ctx := context.Background()
	
	// Convert to model entries
	modelEntries := make([]*model.VMLogEntry, len(entries))
	for i, e := range entries {
		modelEntries[i] = &model.VMLogEntry{
			VMID:     vmid,
			Level:    e.Level,
			Source:   e.Source,
			Message:  e.Message,
			Metadata: e.Metadata,
		}
	}

	return s.logRepo.RecordBatch(ctx, vmid, modelEntries)
}

// IngestVMLog ingests a single log entry for a VM
func (s *VMService) IngestVMLog(vmid int, level, source, message string, metadata map[string]string) error {
	if s.logRepo == nil {
		return nil
	}

	ctx := context.Background()
	return s.logRepo.Record(ctx, vmid, level, source, message, metadata)
}

// readLibvirtLog reads the libvirt VM log file
func (s *VMService) readLibvirtLog(path string, limit int, newEntry func(string, string, string) *VMLogEntry) ([]*VMLogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []*VMLogEntry
	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() && count < limit {
		line := scanner.Text()
		entries = append(entries, newEntry("INFO", "libvirt", line))
		count++
	}

	return entries, scanner.Err()
}

// readSerialLog reads the serial console log file
func (s *VMService) readSerialLog(path string, limit int, newEntry func(string, string, string) *VMLogEntry) ([]*VMLogEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []*VMLogEntry
	scanner := bufio.NewScanner(file)
	count := 0

	for scanner.Scan() && count < limit {
		line := scanner.Text()
		if line != "" {
			entries = append(entries, newEntry("INFO", "console", line))
			count++
		}
	}

	return entries, scanner.Err()
}

// readJournalEntries reads journal entries for the VM
func (s *VMService) readJournalEntries(vmID string, limit int, newEntry func(string, string, string) *VMLogEntry) ([]*VMLogEntry, error) {
	// Try to get journal entries for libvirt and qemu related to this VM
	cmd := exec.Command("journalctl", "-t", "libvirtd", "-t", "qemu", "--no-pager", "-n", fmt.Sprintf("%d", limit))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []*VMLogEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		entries = append(entries, newEntry("INFO", "journal", line))
	}

	return entries, nil
}

// generateSyntheticLogs generates synthetic logs based on VM state
// These are shown when no logs exist in the database yet
func (s *VMService) generateSyntheticLogs(vm *model.VM, newEntry func(string, string, string) *VMLogEntry) []*VMLogEntry {
	var entries []*VMLogEntry

	// Add VM state information (use consistent wording with collector)
	switch vm.Status {
	case model.VMStatusRunning:
		entries = append(entries, newEntry("INFO", "system", fmt.Sprintf("VM %s started", vm.Name)))
		entries = append(entries, newEntry("INFO", "system", fmt.Sprintf("VM %s uptime: %s", vm.Name, vm.Uptime)))
		entries = append(entries, newEntry("INFO", "system", fmt.Sprintf("VM %s IP address: %s", vm.Name, vm.IP)))
	case model.VMStatusStopped:
		entries = append(entries, newEntry("INFO", "system", fmt.Sprintf("VM %s stopped", vm.Name)))
	case model.VMStatusPaused:
		entries = append(entries, newEntry("INFO", "system", fmt.Sprintf("VM %s paused", vm.Name)))
	}

	// Add configuration information
	entries = append(entries, newEntry("INFO", "config", fmt.Sprintf("VM %s CPU: %d sockets x %d cores = %d vCPUs", vm.Name, vm.CPU.Sockets, vm.CPU.Cores, vm.CPU.Sockets*vm.CPU.Cores)))
	entries = append(entries, newEntry("INFO", "config", fmt.Sprintf("VM %s Memory: %.1f GB", vm.Name, vm.Memory.Total)))
	entries = append(entries, newEntry("INFO", "config", fmt.Sprintf("VM %s Disk: %.1f GB / %.1f GB used", vm.Name, vm.Disk.Total, vm.Disk.Used)))

	// Add network information
	if len(vm.Network) > 0 {
		for i, nic := range vm.Network {
			netInfo := fmt.Sprintf("VM %s net%d: model=%s, type=%s", vm.Name, i, nic.Model, nic.Type)
			if nic.Type == model.NetworkTypeBridge && nic.Bridge != "" {
				netInfo += fmt.Sprintf(", bridge=%s", nic.Bridge)
			}
			if nic.VLAN > 0 {
				netInfo += fmt.Sprintf(", vlan=%d", nic.VLAN)
			}
			entries = append(entries, newEntry("INFO", "network", netInfo))
		}
	}

	// Add recent activity based on status
	if vm.Status == model.VMStatusRunning {
		entries = append(entries, newEntry("INFO", "qemu", "QEMU process running"))
		entries = append(entries, newEntry("INFO", "libvirt", "Domain active"))
		entries = append(entries, newEntry("INFO", "system", "VNC console enabled"))
	}

	return entries
}

// GetDiagnostics returns comprehensive diagnostic information for a VM
func (s *VMService) GetDiagnostics(vm *model.VM) (*labv1.VMDiagnostics, error) {
	// Get libvirt connection
	conn, err := libvirt.NewConnect("qemu:///session")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to libvirt: %w", err)
	}
	defer conn.Close()

	// Lookup domain
	domain, err := conn.LookupDomainByName(vm.ID)
	if err != nil {
		return nil, fmt.Errorf("domain not found: %w", err)
	}
	defer domain.Free()

	// Get domain info
	info, err := domain.GetInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain info: %w", err)
	}

	// Get domain name and UUID
	name, err := domain.GetName()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain name: %w", err)
	}

	uuid, err := domain.GetUUIDString()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain UUID: %w", err)
	}

	// Get domain ID
	domID, err := domain.GetID()
	if err != nil {
		domID = 0
	}

	// Get XML configuration
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		xmlDesc = ""
	}

	// Get autostart
	autostart, err := domain.GetAutostart()
	if err != nil {
		autostart = false
	}

	// Get persistent status
	persistent, err := domain.IsPersistent()
	if err != nil {
		persistent = false
	}

	// Get OS type
	osType, err := domain.GetOSType()
	if err != nil {
		osType = "unknown"
	}

	// Build domain info
	domainInfo := &labv1.DomainInfo{
		Id:            int32(domID),
		Name:          name,
		Uuid:          uuid,
		OsType:        osType,
		State:         domainStateToString(info.State),
		MaxMemoryKb:   int32(info.MaxMem),
		UsedMemoryKb:  int32(info.Memory),
		CpuCount:      int32(info.NrVirtCpu),
		Autostart:     map[bool]string{true: "yes", false: "no"}[autostart],
		Persistent:    map[bool]string{true: "yes", false: "no"}[persistent],
	}

	// Get network interfaces
	networkInterfaces := getNetworkInterfaces(domain)

	// Get disk info
	disks := getDiskInfo(domain, xmlDesc)

	// Get QEMU monitor info
	qemuMonitor := getQemuMonitorInfo(conn, domain)

	// Get host info
	hostInfo := getHostInfo(conn)

	return &labv1.VMDiagnostics{
		Info:               domainInfo,
		XmlConfig:          xmlDesc,
		NetworkInterfaces:  networkInterfaces,
		Disks:              disks,
		QemuMonitor:        qemuMonitor,
		Host:               hostInfo,
	}, nil
}

// domainStateToString converts libvirt domain state to string
func domainStateToString(state libvirt.DomainState) string {
	switch state {
	case libvirt.DOMAIN_NOSTATE:
		return "nostate"
	case libvirt.DOMAIN_RUNNING:
		return "running"
	case libvirt.DOMAIN_BLOCKED:
		return "blocked"
	case libvirt.DOMAIN_PAUSED:
		return "paused"
	case libvirt.DOMAIN_SHUTDOWN:
		return "shutdown"
	case libvirt.DOMAIN_SHUTOFF:
		return "shutoff"
	case libvirt.DOMAIN_CRASHED:
		return "crashed"
	case libvirt.DOMAIN_PMSUSPENDED:
		return "pmsuspended"
	default:
		return "unknown"
	}
}

// getNetworkInterfaces gets network interface information
func getNetworkInterfaces(domain *libvirt.Domain) []*labv1.NetworkInterface {
	// GetInterfaceAddresses requires libvirt with guest agent support
	// For now, return empty - IP addresses shown in main VM view come from other sources
	return nil
}

// getDiskInfo gets disk information from domain XML
func getDiskInfo(domain *libvirt.Domain, xmlDesc string) []*labv1.DiskDeviceInfo {
	var disks []*labv1.DiskDeviceInfo

	if xmlDesc == "" {
		return disks
	}

	// Parse XML for disk information
	parsed := &libvirtxml.Domain{}
	if err := parsed.Unmarshal(xmlDesc); err != nil {
		return disks
	}

	if parsed.Devices == nil || parsed.Devices.Disks == nil {
		return disks
	}

	for _, disk := range parsed.Devices.Disks {
		if disk.Device != "disk" {
			continue
		}

		diskInfo := &labv1.DiskDeviceInfo{
			TargetDev:  disk.Target.Dev,
			DriverType: disk.Driver.Type,
			Bus:        string(disk.Target.Bus),
		}

		if disk.Source.File != nil {
			diskInfo.SourceFile = disk.Source.File.File
		}

		disks = append(disks, diskInfo)
	}

	return disks
}

// getQemuMonitorInfo gets QEMU monitor information from domain XML
func getQemuMonitorInfo(conn *libvirt.Connect, domain *libvirt.Domain) *labv1.QemuMonitorInfo {
	info := &labv1.QemuMonitorInfo{}

	// Get VNC info from XML
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return info
	}

	// Simple XML parsing for VNC port
	if idx := strings.Index(xmlDesc, "type='vnc'"); idx >= 0 {
		// Extract port from XML - simple regex-style parsing
		if portIdx := strings.Index(xmlDesc[idx:], "port='"); portIdx >= 0 {
			portStr := xmlDesc[idx+portIdx+6:]
			if endIdx := strings.Index(portStr, "'"); endIdx >= 0 {
				if port, err := strconv.Atoi(portStr[:endIdx]); err == nil {
					info.VncPort = int32(port)
				}
			}
		}
		// Extract listen address
		if listenIdx := strings.Index(xmlDesc[idx:], "listen='"); listenIdx >= 0 {
			listenStr := xmlDesc[idx+listenIdx+8:]
			if endIdx := strings.Index(listenStr, "'"); endIdx >= 0 {
				info.VncServer = listenStr[:endIdx]
			}
		}
	}

	// Extract serial/console device paths from XML
	info.CharDevices = extractCharDevices(xmlDesc)

	return info
}

// extractCharDevices extracts character device information from domain XML
func extractCharDevices(xmlDesc string) []*labv1.CharDevice {
	var devices []*labv1.CharDevice

	// Find serial devices
	serialIdx := 0
	for {
		idx := strings.Index(xmlDesc[serialIdx:], "<serial ")
		if idx < 0 {
			break
		}
		serialIdx += idx
		endIdx := strings.Index(xmlDesc[serialIdx:], "</serial>")
		if endIdx < 0 {
			break
		}
		serialBlock := xmlDesc[serialIdx : serialIdx+endIdx]

		cd := &labv1.CharDevice{Name: "serial"}
		// Extract path from source element
		if pathIdx := strings.Index(serialBlock, "path='"); pathIdx >= 0 {
			pathStr := serialBlock[pathIdx+6:]
			if endPathIdx := strings.Index(pathStr, "'"); endPathIdx >= 0 {
				cd.SourcePath = pathStr[:endPathIdx]
			}
		}
		devices = append(devices, cd)
		serialIdx += endIdx + 9
	}

	// Find console devices
	consoleIdx := 0
	for {
		idx := strings.Index(xmlDesc[consoleIdx:], "<console ")
		if idx < 0 {
			break
		}
		consoleIdx += idx
		endIdx := strings.Index(xmlDesc[consoleIdx:], "</console>")
		if endIdx < 0 {
			break
		}
		consoleBlock := xmlDesc[consoleIdx : consoleIdx+endIdx]

		cd := &labv1.CharDevice{Name: "console"}
		if pathIdx := strings.Index(consoleBlock, "path='"); pathIdx >= 0 {
			pathStr := consoleBlock[pathIdx+6:]
			if endPathIdx := strings.Index(pathStr, "'"); endPathIdx >= 0 {
				cd.SourcePath = pathStr[:endPathIdx]
			}
		}
		devices = append(devices, cd)
		consoleIdx += endIdx + 10
	}

	return devices
}

// getHostInfo gets host system information
func getHostInfo(conn *libvirt.Connect) *labv1.HostInfo {
	info := &labv1.HostInfo{}

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	info.Hostname = hostname

	// Get system info
	sysInfo := sysinfo.New()
	info.Arch = sysInfo.HostArch()

	// Get libvirt info
	info.LibvirtUri = "qemu:///session"

	libvirtVer, err := conn.GetLibVersion()
	if err == nil {
		info.LibvirtVersion = fmt.Sprintf("%d.%d.%d",
			libvirtVer/1000000,
			(libvirtVer/1000)%1000,
			libvirtVer%1000,
		)
	}

	return info
}
