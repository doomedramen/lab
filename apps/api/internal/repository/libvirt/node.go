package libvirt

import (
	"context"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	"github.com/doomedramen/lab/apps/api/pkg/sysinfo"
)

// NodeRepository implements repository.NodeRepository using libvirt
type NodeRepository struct {
	client libvirtx.LibvirtClient
	sys    sysinfo.SystemInfo
}

// NewNodeRepository creates a new libvirt node repository
func NewNodeRepository(client libvirtx.LibvirtClient) *NodeRepository {
	return &NodeRepository{
		client: client,
		sys:    sysinfo.New(),
	}
}

// GetAll returns all nodes (for single-host libvirt, returns just the local host)
func (r *NodeRepository) GetAll(_ context.Context) ([]*model.HostNode, error) {
	hostname, err := r.client.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	node, err := r.getNodeInfo(hostname)
	if err != nil {
		return nil, err
	}
	return []*model.HostNode{node}, nil
}

// GetByID returns a node by ID
func (r *NodeRepository) GetByID(_ context.Context, id string) (*model.HostNode, error) {
	hostname, err := r.client.GetHostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}

	// For single-host, ID is the hostname
	if id == hostname || id == "local" || id == "localhost" {
		return r.getNodeInfo(hostname)
	}
	return nil, fmt.Errorf("node %q not found", id)
}

// GetByName returns a node by name
func (r *NodeRepository) GetByName(ctx context.Context, name string) (*model.HostNode, error) {
	return r.GetByID(ctx, name)
}

// Reboot initiates a node reboot
func (r *NodeRepository) Reboot(_ context.Context, id string) error {
	// In a real implementation, this would trigger an actual reboot
	_ = id
	return nil
}

// Shutdown initiates a node shutdown
func (r *NodeRepository) Shutdown(_ context.Context, id string) error {
	// In a real implementation, this would trigger an actual shutdown
	_ = id
	return nil
}

// getNodeInfo gathers node information from libvirt and the host OS.
func (r *NodeRepository) getNodeInfo(hostname string) (*model.HostNode, error) {
	conn, err := r.client.Connection()
	if err != nil {
		return nil, fmt.Errorf("failed to get libvirt connection: %w", err)
	}

	nodeInfo, err := conn.GetNodeInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	version, _ := r.client.GetVersion()

	domains, _ := r.client.ListDomains()
	vmCount := len(domains)
	for i := range domains {
		domains[i].Free()
	}

	diskUsed, diskTotal := r.sys.GetDiskInfo()
	uptimeStr := r.sys.GetUptime()
	cpuUsed := r.sys.GetCPUUsage()
	memoryUsed, memoryTotal := r.sys.GetMemoryInfo()
	loadAvg := r.sys.GetLoadAvg()
	netIn, netOut := r.sys.GetNetworkStats()
	kernel := r.sys.GetKernelVersion()

	return &model.HostNode{
		ID:         hostname,
		Name:       hostname,
		Status:     model.NodeStatusOnline,
		IP:         "127.0.0.1",
		CPU:        model.CPUInfo{Used: cpuUsed, Total: 100, Cores: int(nodeInfo.Cpus)},
		Memory:     model.MemoryInfo{Used: memoryUsed, Total: memoryTotal},
		Disk:       model.DiskInfo{Used: diskUsed, Total: diskTotal},
		Uptime:     uptimeStr,
		Kernel:     kernel,
		Version:    version,
		VMs:        vmCount,
		Containers: 0,
		CPUModel:   nodeInfo.Model,
		LoadAvg:    loadAvg,
		NetworkIn:  netIn,
		NetworkOut: netOut,
		Arch:       r.sys.HostArch(),
	}, nil
}
