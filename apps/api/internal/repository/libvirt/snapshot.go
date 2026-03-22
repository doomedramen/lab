package libvirt

import (
	"encoding/xml"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"libvirt.org/go/libvirtxml"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
)

// SnapshotRepository implements repository.SnapshotRepository using libvirt via virsh
type SnapshotRepository struct {
	client libvirtx.LibvirtClient
}

// NewSnapshotRepository creates a new libvirt snapshot repository
func NewSnapshotRepository(client libvirtx.LibvirtClient) *SnapshotRepository {
	return &SnapshotRepository{
		client: client,
	}
}

// Create creates a new snapshot of a VM using virsh
func (r *SnapshotRepository) Create(vmid int, name, description string, live, includeMemory bool) (*model.Snapshot, error) {
	domainName := fmt.Sprintf("vm-%d", vmid)

	// Check if VM exists first
	if _, err := r.client.GetDomainByName(domainName); err != nil {
		return nil, fmt.Errorf("VM %d not found: %w", vmid, err)
	}

	// Generate snapshot name if not provided
	if name == "" {
		name = fmt.Sprintf("snap-%d", time.Now().Unix())
	}

	// Build virsh command
	// virsh snapshot-create-as <domain> <name> <description> [--live] [--disk-only]
	args := []string{"snapshot-create-as", domainName, name}
	if description != "" {
		args = append(args, "--description", description)
	}
	if live {
		if includeMemory {
			args = append(args, "--live")
		} else {
			args = append(args, "--disk-only")
		}
	}

	cmd := exec.Command("virsh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w, output: %s", err, string(output))
	}

	// Get snapshot info
	snapshotXML, err := r.getSnapshotXML(domainName, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot info: %w", err)
	}

	// Parse snapshot XML
	var snapshotDef libvirtxml.DomainSnapshot
	if err := xml.Unmarshal([]byte(snapshotXML), &snapshotDef); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot XML: %w", err)
	}

	// Parse creation time
	createdAt := time.Now().Format(time.RFC3339)
	if snapshotDef.CreationTime != "" {
		if ts, err := strconv.ParseInt(snapshotDef.CreationTime, 10, 64); err == nil {
			createdAt = time.Unix(ts, 0).Format(time.RFC3339)
		}
	}

	// Calculate size
	sizeBytes, _ := r.parseDiskSizesFromXML(snapshotXML)

	// Determine VM state
	vmState := model.VMStateStopped
	if live {
		vmState = model.VMStateRunning
	}

	return &model.Snapshot{
		ID:          name,
		VMID:        vmid,
		Name:        name,
		Description: description,
		CreatedAt:   createdAt,
		SizeBytes:   sizeBytes,
		Status:      model.SnapshotStatusReady,
		VMState:     vmState,
		HasChildren: false,
	}, nil
}

// Delete deletes a snapshot using virsh
func (r *SnapshotRepository) Delete(vmid int, snapshotID string) error {
	domainName := fmt.Sprintf("vm-%d", vmid)

	cmd := exec.Command("virsh", "snapshot-delete", domainName, snapshotID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w, output: %s", err, string(output))
	}

	return nil
}

// Restore restores a VM to a snapshot state using virsh
func (r *SnapshotRepository) Restore(vmid int, snapshotID string) error {
	domainName := fmt.Sprintf("vm-%d", vmid)

	cmd := exec.Command("virsh", "snapshot-revert", domainName, snapshotID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restore snapshot: %w, output: %s", err, string(output))
	}

	return nil
}

// List returns all snapshots for a VM using virsh
func (r *SnapshotRepository) List(vmid int) ([]model.Snapshot, error) {
	domainName := fmt.Sprintf("vm-%d", vmid)

	// Check if VM exists first
	if _, err := r.client.GetDomainByName(domainName); err != nil {
		return nil, fmt.Errorf("VM %d not found: %w", vmid, err)
	}

	// List snapshots using virsh
	cmd := exec.Command("virsh", "snapshot-list", domainName, "--name")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w, output: %s", err, string(output))
	}

	var snapshots []model.Snapshot
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		snapshot, err := r.getSnapshotInfo(domainName, line)
		if err != nil {
			continue
		}
		snapshots = append(snapshots, *snapshot)
	}

	return snapshots, nil
}

// GetInfo returns detailed information about a snapshot
func (r *SnapshotRepository) GetInfo(vmid int, snapshotID string) (*model.Snapshot, error) {
	domainName := fmt.Sprintf("vm-%d", vmid)
	return r.getSnapshotInfo(domainName, snapshotID)
}

// getSnapshotInfo returns detailed information about a specific snapshot
func (r *SnapshotRepository) getSnapshotInfo(domainName, snapshotID string) (*model.Snapshot, error) {
	snapshotXML, err := r.getSnapshotXML(domainName, snapshotID)
	if err != nil {
		return nil, err
	}

	// Parse snapshot XML
	var snapshotDef libvirtxml.DomainSnapshot
	if err := xml.Unmarshal([]byte(snapshotXML), &snapshotDef); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot XML: %w", err)
	}

	// Parse VMID from domain name
	var vmid int
	fmt.Sscanf(domainName, "vm-%d", &vmid)

	// Parse creation time
	createdAt := time.Now().Format(time.RFC3339)
	if snapshotDef.CreationTime != "" {
		if ts, err := strconv.ParseInt(snapshotDef.CreationTime, 10, 64); err == nil {
			createdAt = time.Unix(ts, 0).Format(time.RFC3339)
		}
	}

	// Parse parent
	parentID := ""
	if snapshotDef.Parent != nil && snapshotDef.Parent.Name != "" {
		parentID = snapshotDef.Parent.Name
	}

	// Check if has children
	hasChildren, _ := r.hasChildrenFromXML(snapshotXML)

	// Calculate size
	sizeBytes, _ := r.parseDiskSizesFromXML(snapshotXML)

	return &model.Snapshot{
		ID:           snapshotID,
		VMID:         vmid,
		Name:         snapshotDef.Name,
		Description:  snapshotDef.Description,
		CreatedAt:    createdAt,
		ParentID:     parentID,
		SizeBytes:    sizeBytes,
		Status:       model.SnapshotStatusReady,
		VMState:      model.VMStateStopped,
		HasChildren:  hasChildren,
		SnapshotPath: "",
	}, nil
}

// getSnapshotXML gets the XML description of a snapshot
func (r *SnapshotRepository) getSnapshotXML(domainName, snapshotID string) (string, error) {
	cmd := exec.Command("virsh", "snapshot-dumpxml", domainName, snapshotID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get snapshot XML: %w, output: %s", err, string(output))
	}
	return string(output), nil
}

// hasChildrenFromXML checks if a snapshot has child snapshots from XML
func (r *SnapshotRepository) hasChildrenFromXML(xmlStr string) (bool, error) {
	return strings.Contains(xmlStr, "<children>"), nil
}

// parseDiskSizesFromXML extracts total disk size from snapshot XML
func (r *SnapshotRepository) parseDiskSizesFromXML(xmlStr string) (int64, error) {
	var totalSize int64

	// Simple parsing - look for <size> elements
	sizeStart := 0
	for {
		sizeTag := strings.Index(xmlStr[sizeStart:], "<size")
		if sizeTag == -1 {
			break
		}

		sizeTag += sizeStart
		valueStart := strings.Index(xmlStr[sizeTag:], ">")
		if valueStart == -1 {
			break
		}

		valueStart++
		valueEnd := strings.Index(xmlStr[valueStart:], "</size>")
		if valueEnd == -1 {
			break
		}

		sizeStr := xmlStr[valueStart : valueStart+valueEnd]
		size, err := strconv.ParseInt(sizeStr, 10, 64)
		if err == nil {
			totalSize += size
		}

		sizeStart = valueStart + valueEnd + len("</size>")
	}

	return totalSize, nil
}

// GetSnapshotTree builds a tree of snapshots
func (r *SnapshotRepository) GetSnapshotTree(vmid int) (*model.SnapshotTree, error) {
	snapshots, err := r.List(vmid)
	if err != nil {
		return nil, err
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	// Build snapshot map
	snapshotMap := make(map[string]*model.SnapshotTree)
	for i := range snapshots {
		s := snapshots[i]
		snapshotMap[s.ID] = &model.SnapshotTree{
			Snapshot: &s,
			Children: []*model.SnapshotTree{},
		}
	}

	var root *model.SnapshotTree

	// Build tree structure
	for _, s := range snapshots {
		if s.ParentID == "" {
			if root == nil {
				root = snapshotMap[s.ID]
			}
		} else {
			if parent, ok := snapshotMap[s.ParentID]; ok {
				parent.Children = append(parent.Children, snapshotMap[s.ID])
			}
		}
	}

	return root, nil
}
