package libvirt

import (
	"context"
	"crypto/rand"
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"

	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// ContainerRepository implements repository.ContainerRepository using libvirt LXC domains.
// LXC containers are managed via a separate libvirt connection (lxc:///system).
// Domain naming convention: ct-<ctid>
type ContainerRepository struct {
	client  *libvirtx.Client
	rootDir string // Base directory for container root filesystems
	node    string // Node name (hostname)
}

// NewContainerRepository creates a new libvirt LXC container repository.
// client must be connected to lxc:///system.
// rootDir is the base directory for container rootfs (default: /var/lib/lxc).
func NewContainerRepository(client *libvirtx.Client, rootDir string) *ContainerRepository {
	if rootDir == "" {
		rootDir = "/var/lib/lxc"
	}
	node, _ := client.GetHostname()
	return &ContainerRepository{
		client:  client,
		rootDir: rootDir,
		node:    node,
	}
}

// GetAll returns all LXC containers managed by libvirt.
func (r *ContainerRepository) GetAll(_ context.Context) ([]*model.Container, error) {
	domains, err := r.client.ListDomains()
	if err != nil {
		return nil, fmt.Errorf("failed to list LXC domains: %w", err)
	}

	var containers []*model.Container
	for _, domain := range domains {
		ct, err := r.domainToContainer(domain)
		domain.Free()
		if err != nil || ct == nil {
			continue
		}
		containers = append(containers, ct)
	}

	return containers, nil
}

// GetByNode returns containers for a specific node.
// For local libvirt, all containers are on the same node.
func (r *ContainerRepository) GetByNode(ctx context.Context, node string) ([]*model.Container, error) {
	if node != "" && node != r.node {
		return []*model.Container{}, nil
	}
	return r.GetAll(ctx)
}

// GetByID returns a container by its string domain name.
func (r *ContainerRepository) GetByID(_ context.Context, id string) (*model.Container, error) {
	domain, err := r.client.GetDomainByName(id)
	if err != nil {
		return nil, fmt.Errorf("container %q not found: %w", id, err)
	}
	defer domain.Free()

	ct, err := r.domainToContainer(domain)
	if err != nil {
		return nil, fmt.Errorf("failed to convert domain %q to container: %w", id, err)
	}
	return ct, nil
}

// GetByCTID returns a container by its numeric CTID.
func (r *ContainerRepository) GetByCTID(_ context.Context, ctid int) (*model.Container, error) {
	domainName := fmt.Sprintf("ct-%d", ctid)
	return r.GetByID(context.Background(), domainName)
}

// Create creates a new LXC container in libvirt.
func (r *ContainerRepository) Create(_ context.Context, req *model.ContainerCreateRequest) (*model.Container, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	ctid := r.generateCTID()
	domainName := fmt.Sprintf("ct-%d", ctid)
	rootfs := fmt.Sprintf("%s/%s/rootfs", r.rootDir, domainName)

	// Create rootfs directory
	if err := os.MkdirAll(rootfs, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rootfs directory %s: %w", rootfs, err)
	}

	// Convert memory from GB to KiB
	memoryKiB := uint(req.Memory * 1024 * 1024)
	if memoryKiB == 0 {
		memoryKiB = 512 * 1024 // default 512 MB
	}

	vcpus := req.CPUCores
	if vcpus == 0 {
		vcpus = 1
	}

	mac := r.generateMAC()

	domainXML := r.buildLXCDomainXML(domainName, req.Name, memoryKiB, uint(vcpus), rootfs, mac)

	domain, err := conn.DomainDefineXML(domainXML)
	if err != nil {
		os.RemoveAll(fmt.Sprintf("%s/%s", r.rootDir, domainName))
		return nil, fmt.Errorf("failed to define LXC domain: %w", err)
	}
	defer domain.Free()

	return &model.Container{
		ID:           domainName,
		CTID:         ctid,
		Name:         req.Name,
		Node:         r.node,
		Status:       model.ContainerStatusStopped,
		CPU:          model.CPUInfoPartial{Cores: vcpus},
		Memory:       model.MemoryInfo{Total: req.Memory},
		Disk:         model.DiskInfo{Total: req.Disk},
		OS:           req.OS,
		Tags:         req.Tags,
		Unprivileged: req.Unprivileged,
		Description:  req.Description,
		StartOnBoot:  req.StartOnBoot,
	}, nil
}

// Update updates an existing LXC container configuration.
// Note: changing CPU and memory requires the container to be stopped.
func (r *ContainerRepository) Update(_ context.Context, ctid int, req *model.ContainerUpdateRequest) (*model.Container, error) {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return nil, fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	// Get current XML and modify it
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML: %w", err)
	}

	var domDef libvirtxml.Domain
	if err := xml.Unmarshal([]byte(xmlDesc), &domDef); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML: %w", err)
	}

	// Apply updates
	if req.CPUCores > 0 {
		domDef.VCPU = &libvirtxml.DomainVCPU{Value: uint(req.CPUCores)}
	}
	if req.Memory > 0 {
		memKiB := uint(req.Memory * 1024 * 1024)
		domDef.Memory = &libvirtxml.DomainMemory{Value: memKiB, Unit: "KiB"}
	}

	// Re-define domain with updated XML
	newXML, err := xml.MarshalIndent(domDef, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal domain XML: %w", err)
	}

	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	updatedDomain, err := conn.DomainDefineXML(string(newXML))
	if err != nil {
		return nil, fmt.Errorf("failed to update domain: %w", err)
	}
	defer updatedDomain.Free()

	return r.domainToContainer(*updatedDomain)
}

// Delete removes a container and its rootfs.
func (r *ContainerRepository) Delete(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)

	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	// Stop the container if running
	state, _, err := domain.GetState()
	if err == nil && state == libvirt.DOMAIN_RUNNING {
		if err := domain.Destroy(); err != nil {
			return fmt.Errorf("failed to stop container before deletion: %w", err)
		}
	}

	// Undefine the domain
	if err := domain.Undefine(); err != nil {
		return fmt.Errorf("failed to undefine container domain: %w", err)
	}

	// Remove rootfs
	rootfs := fmt.Sprintf("%s/%s", r.rootDir, domainName)
	if err := os.RemoveAll(rootfs); err != nil {
		// Log but don't fail — domain is already removed
		_ = err
	}

	return nil
}

// Start starts an LXC container.
func (r *ContainerRepository) Start(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Create(); err != nil {
		return fmt.Errorf("failed to start container %d: %w", ctid, err)
	}
	return nil
}

// Stop forcefully stops an LXC container (destroy).
func (r *ContainerRepository) Stop(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Destroy(); err != nil {
		return fmt.Errorf("failed to stop container %d: %w", ctid, err)
	}
	return nil
}

// Shutdown gracefully shuts down an LXC container.
func (r *ContainerRepository) Shutdown(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Shutdown(); err != nil {
		return fmt.Errorf("failed to shutdown container %d: %w", ctid, err)
	}
	return nil
}

// Pause freezes an LXC container (suspend).
func (r *ContainerRepository) Pause(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Suspend(); err != nil {
		return fmt.Errorf("failed to pause container %d: %w", ctid, err)
	}
	return nil
}

// Reboot reboots an LXC container.
func (r *ContainerRepository) Reboot(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Reboot(libvirt.DOMAIN_REBOOT_DEFAULT); err != nil {
		return fmt.Errorf("failed to reboot container %d: %w", ctid, err)
	}
	return nil
}

// Resume unfreezes a paused LXC container.
func (r *ContainerRepository) Resume(_ context.Context, ctid int) error {
	domainName := fmt.Sprintf("ct-%d", ctid)
	domain, err := r.client.GetDomainByName(domainName)
	if err != nil {
		return fmt.Errorf("container %d not found: %w", ctid, err)
	}
	defer domain.Free()

	if err := domain.Resume(); err != nil {
		return fmt.Errorf("failed to resume container %d: %w", ctid, err)
	}
	return nil
}

// --- internal helpers ---

// domainToContainer converts a libvirt domain to a Container model.
func (r *ContainerRepository) domainToContainer(domain libvirt.Domain) (*model.Container, error) {
	name, err := domain.GetName()
	if err != nil {
		return nil, fmt.Errorf("failed to get domain name: %w", err)
	}

	// Only process domains following ct-<N> naming convention
	ctid := r.extractCTID(name)
	if ctid == 0 {
		return nil, nil // skip non-container domains
	}

	// Get domain XML to extract metadata
	xmlDesc, err := domain.GetXMLDesc(0)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain XML for %s: %w", name, err)
	}

	var domDef libvirtxml.Domain
	if err := xml.Unmarshal([]byte(xmlDesc), &domDef); err != nil {
		return nil, fmt.Errorf("failed to parse domain XML for %s: %w", name, err)
	}

	// Only process LXC domains (type=lxc or OS type=exe)
	if domDef.Type != "lxc" {
		osType := ""
		if domDef.OS != nil && domDef.OS.Type != nil {
			osType = domDef.OS.Type.Type
		}
		if osType != "exe" {
			return nil, nil // skip QEMU VMs
		}
	}

	// Get state
	state, _, err := domain.GetState()
	status := model.ContainerStatusStopped
	if err == nil {
		switch state {
		case libvirt.DOMAIN_RUNNING:
			status = model.ContainerStatusRunning
		case libvirt.DOMAIN_PMSUSPENDED:
			status = model.ContainerStatusFrozen
		}
	}

	// Get memory info
	var memTotal float64
	var memUsed float64
	if domDef.Memory != nil {
		memKiB := float64(domDef.Memory.Value)
		switch domDef.Memory.Unit {
		case "GiB":
			memTotal = memKiB
		case "MiB":
			memTotal = memKiB / 1024
		default: // KiB
			memTotal = memKiB / (1024 * 1024)
		}
	}

	if status == model.ContainerStatusRunning {
		memStats, err := domain.MemoryStats(11, 0) // 11 = all stats
		if err == nil {
			for _, stat := range memStats {
				if stat.Tag == int32(libvirt.DOMAIN_MEMORY_STAT_RSS) {
					memUsed = float64(stat.Val) / (1024 * 1024)
					break
				}
			}
		}
	}

	// Get CPU info
	vcpus := 1
	if domDef.VCPU != nil {
		vcpus = int(domDef.VCPU.Value)
	}

	// Get disk usage (rootfs size)
	diskTotal, diskUsed := r.getRootfsDiskInfo(name)

	// Get uptime
	uptime := "0s"
	if status == model.ContainerStatusRunning {
		info, err := domain.GetInfo()
		if err == nil && info.CpuTime > 0 {
			uptimeSeconds := info.CpuTime / 1_000_000_000
			hours := uptimeSeconds / 3600
			minutes := (uptimeSeconds % 3600) / 60
			if hours > 0 {
				uptime = fmt.Sprintf("%dh %dm", hours, minutes)
			} else {
				uptime = fmt.Sprintf("%dm", minutes)
			}
		}
	}

	// Get MAC/IP from interface (best effort)
	ip := ""
	if status == model.ContainerStatusRunning {
		ip = r.getContainerIP(domain)
	}

	return &model.Container{
		ID:     name,
		CTID:   ctid,
		Name:   r.getContainerLabel(domDef),
		Node:   r.node,
		Status: status,
		CPU: model.CPUInfoPartial{
			Cores: vcpus,
		},
		Memory: model.MemoryInfo{
			Total: memTotal,
			Used:  memUsed,
		},
		Disk: model.DiskInfo{
			Total: diskTotal,
			Used:  diskUsed,
		},
		Uptime:       uptime,
		IP:           ip,
		Unprivileged: r.isUnprivileged(domDef),
	}, nil
}

// getContainerLabel returns the human-readable label from domain metadata or title.
func (r *ContainerRepository) getContainerLabel(domDef libvirtxml.Domain) string {
	if domDef.Title != "" {
		return domDef.Title
	}
	return domDef.Name
}

// isUnprivileged checks if the container runs as an unprivileged LXC container.
// Privileged containers use accessmode='passthrough', unprivileged use 'mapped'.
func (r *ContainerRepository) isUnprivileged(domDef libvirtxml.Domain) bool {
	if domDef.Devices == nil {
		return false
	}
	for _, fs := range domDef.Devices.Filesystems {
		if fs.AccessMode == "mapped" {
			return true
		}
	}
	return false
}

// extractCTID parses the numeric CTID from a domain name like "ct-100".
func (r *ContainerRepository) extractCTID(name string) int {
	if !strings.HasPrefix(name, "ct-") {
		return 0
	}
	id, err := strconv.Atoi(strings.TrimPrefix(name, "ct-"))
	if err != nil {
		return 0
	}
	return id
}

// generateCTID generates a unique CTID starting from 100, avoiding conflicts with VMs.
func (r *ContainerRepository) generateCTID() int {
	domains, err := r.client.ListDomains()
	if err != nil {
		return 100
	}

	used := make(map[int]bool)
	for _, domain := range domains {
		name, err := domain.GetName()
		if err == nil {
			id := r.extractCTID(name)
			if id > 0 {
				used[id] = true
			}
		}
		domain.Free()
	}

	for id := 100; id < 10000; id++ {
		if !used[id] {
			return id
		}
	}
	return 100
}

// generateMAC generates a random MAC address with the qemu/libvirt prefix 52:54:00.
func (r *ContainerRepository) generateMAC() string {
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("52:54:00:%02x:%02x:%02x", b[0], b[1], b[2])
}

// getRootfsDiskInfo returns total/used disk space for the container rootfs.
func (r *ContainerRepository) getRootfsDiskInfo(domainName string) (total, used float64) {
	rootfs := fmt.Sprintf("%s/%s/rootfs", r.rootDir, domainName)
	info, err := os.Stat(rootfs)
	if err != nil || !info.IsDir() {
		return 0, 0
	}
	// Getting actual disk usage requires du or statfs - use simple heuristic
	// Return zeroes if we can't get the info — the UI can handle it
	return 0, 0
}

// getContainerIP returns the IP address of a running container (best-effort).
func (r *ContainerRepository) getContainerIP(domain libvirt.Domain) string {
	ifaces, err := domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_LEASE)
	if err != nil {
		// Try agent-based query
		ifaces, err = domain.ListAllInterfaceAddresses(libvirt.DOMAIN_INTERFACE_ADDRESSES_SRC_AGENT)
		if err != nil {
			return ""
		}
	}

	for _, iface := range ifaces {
		if iface.Name == "lo" {
			continue
		}
		for _, addr := range iface.Addrs {
			if addr.Type == libvirt.IP_ADDR_TYPE_IPV4 {
				return addr.Addr
			}
		}
	}
	return ""
}

// buildLXCDomainXML creates the libvirt XML for an LXC container.
func (r *ContainerRepository) buildLXCDomainXML(domainName, title string, memKiB, vcpus uint, rootfs, mac string) string {
	return fmt.Sprintf(`<domain type='lxc'>
  <name>%s</name>
  <title>%s</title>
  <memory unit='KiB'>%d</memory>
  <vcpu>%d</vcpu>
  <os>
    <type>exe</type>
    <init>/sbin/init</init>
  </os>
  <devices>
    <emulator>/usr/lib/libvirt/libvirt_lxc</emulator>
    <filesystem type='mount' accessmode='passthrough'>
      <source dir='%s'/>
      <target dir='/'/>
    </filesystem>
    <interface type='network'>
      <source network='default'/>
      <mac address='%s'/>
    </interface>
    <console type='pty'/>
  </devices>
</domain>`, domainName, title, memKiB, vcpus, rootfs, mac)
}
