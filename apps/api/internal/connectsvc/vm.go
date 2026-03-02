package connectsvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
	"github.com/doomedramen/lab/apps/api/pkg/sysinfo"
)

// VmServiceServer implements labv1connect.VmServiceHandler.
type VmServiceServer struct {
	labv1connect.UnimplementedVmServiceHandler
	svc *service.VMService
}

// NewVmServiceServer creates a new VmServiceServer.
func NewVmServiceServer(svc *service.VMService) *VmServiceServer {
	return &VmServiceServer{svc: svc}
}

// ListVMs returns all VMs, optionally filtered by node.
func (s *VmServiceServer) ListVMs(
	ctx context.Context,
	req *connect.Request[labv1.ListVMsRequest],
) (*connect.Response[labv1.ListVMsResponse], error) {
	var vms []*model.VM
	var err error
	if req.Msg.Node != "" {
		vms, err = s.svc.GetByNode(ctx, req.Msg.Node)
	} else {
		vms, err = s.svc.GetAll(ctx)
	}
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	protoVMs := make([]*labv1.VM, len(vms))
	for i, v := range vms {
		protoVMs[i] = modelVMToProto(v)
	}
	return connect.NewResponse(&labv1.ListVMsResponse{Vms: protoVMs}), nil
}

// GetVM returns a single VM by VMID.
func (s *VmServiceServer) GetVM(
	ctx context.Context,
	req *connect.Request[labv1.GetVMRequest],
) (*connect.Response[labv1.GetVMResponse], error) {
	vm, err := s.svc.GetByVMID(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetVMResponse{Vm: modelVMToProto(vm)}), nil
}

// validVMName matches names that contain only letters, digits, hyphens, and underscores.
var validVMName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// CreateVM creates a new VM.
func (s *VmServiceServer) CreateVM(
	ctx context.Context,
	req *connect.Request[labv1.CreateVMRequest],
) (*connect.Response[labv1.CreateVMResponse], error) {
	// --- Input validation ---
	if strings.TrimSpace(req.Msg.Name) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if len(req.Msg.Name) > 63 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name must be 63 characters or fewer, got %d", len(req.Msg.Name)))
	}
	if !validVMName.MatchString(req.Msg.Name) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name may only contain letters, digits, hyphens, and underscores"))
	}
	if req.Msg.CpuCores <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cpu_cores must be greater than 0"))
	}
	if req.Msg.CpuCores > 512 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cpu_cores %d exceeds the maximum of 512", req.Msg.CpuCores))
	}
	if req.Msg.MemoryGb <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("memory_gb must be greater than 0"))
	}
	if req.Msg.MemoryGb > 4096 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("memory_gb %.1f exceeds the maximum of 4096 GB", req.Msg.MemoryGb))
	}
	if req.Msg.DiskGb <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disk_gb must be greater than 0"))
	}
	if req.Msg.DiskGb > 102400 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("disk_gb %.1f exceeds the maximum of 102400 GB (100 TB)", req.Msg.DiskGb))
	}
	// --- End validation ---

	modelReq := protoCreateVMRequestToModel(req.Msg)

	// Resolve architecture placeholders in ISO URL if provided
	if modelReq.ISOURL != "" {
		modelReq.ISOURL = resolveArchPlaceholder(modelReq.ISOURL, modelReq.Arch)
	}
	if modelReq.ISOName != "" {
		modelReq.ISOName = resolveArchPlaceholder(modelReq.ISOName, modelReq.Arch)
	}
	
	vm, err := s.svc.Create(ctx, modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	if vm == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to create VM"))
	}
	return connect.NewResponse(&labv1.CreateVMResponse{Vm: modelVMToProto(vm)}), nil
}

// resolveArchPlaceholder replaces ${arch} with the appropriate architecture string
func resolveArchPlaceholder(s string, arch string) string {
	// Map common arch names to URL-friendly variants
	archVariants := map[string]string{
		"x86_64":  "amd64",
		"aarch64": "arm64",
		"amd64":   "amd64",
		"arm64":   "arm64",
	}
	
	variant := archVariants[arch]
	if variant == "" {
		variant = arch // fallback to original
	}
	
	return strings.ReplaceAll(s, "${arch}", variant)
}

// UpdateVM updates an existing VM.
func (s *VmServiceServer) UpdateVM(
	ctx context.Context,
	req *connect.Request[labv1.UpdateVMRequest],
) (*connect.Response[labv1.UpdateVMResponse], error) {
	// Build model request - use pointers to distinguish unset from zero values
	modelReq := &model.VMUpdateRequest{}

	// Live updates (no restart required)
	if req.Msg.Name != "" {
		modelReq.Name = &req.Msg.Name
	}
	if req.Msg.Description != "" {
		modelReq.Description = &req.Msg.Description
	}
	if len(req.Msg.Tags) > 0 {
		modelReq.Tags = req.Msg.Tags
	}

	// Offline updates (VM must be stopped)
	// Note: We use a helper to convert proto "optional" fields to pointers
	if req.Msg.CpuSockets > 0 {
		v := int(req.Msg.CpuSockets)
		modelReq.CPUSockets = &v
	}
	if req.Msg.CpuCores > 0 {
		v := int(req.Msg.CpuCores)
		modelReq.CPUCores = &v
	}
	if req.Msg.MemoryGb > 0 {
		v := req.Msg.MemoryGb
		modelReq.Memory = &v
	}
	// For bool fields, we need to check if they were explicitly set
	// In proto3, bools default to false, so we need a different approach
	// We'll use a mask field or accept that false means "don't change"
	// For now, we'll only set if true (user can only enable, not disable via this API)
	if req.Msg.Agent {
		v := true
		modelReq.Agent = &v
	}
	if req.Msg.NestedVirt {
		v := true
		modelReq.NestedVirt = &v
	}
	if req.Msg.StartOnBoot {
		v := true
		modelReq.StartOnBoot = &v
	}
	if req.Msg.Tpm {
		v := true
		modelReq.TPM = &v
	}
	if req.Msg.SecureBoot {
		v := true
		modelReq.SecureBoot = &v
	}
	if len(req.Msg.BootOrder) > 0 {
		modelReq.BootOrder = req.Msg.BootOrder
	}

	vm, err := s.svc.Update(ctx, int(req.Msg.Vmid), modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.UpdateVMResponse{Vm: modelVMToProto(vm)}), nil
}

// DeleteVM deletes a VM.
func (s *VmServiceServer) DeleteVM(
	ctx context.Context,
	req *connect.Request[labv1.DeleteVMRequest],
) (*connect.Response[labv1.DeleteVMResponse], error) {
	if err := s.svc.Delete(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.DeleteVMResponse{}), nil
}

// CloneVM clones an existing VM.
func (s *VmServiceServer) CloneVM(
	ctx context.Context,
	req *connect.Request[labv1.CloneVMRequest],
) (*connect.Response[labv1.CloneVMResponse], error) {
	// --- Input validation ---
	if strings.TrimSpace(req.Msg.Name) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}
	if len(req.Msg.Name) > 63 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name must be 63 characters or fewer, got %d", len(req.Msg.Name)))
	}
	if !validVMName.MatchString(req.Msg.Name) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name may only contain letters, digits, hyphens, and underscores"))
	}
	if req.Msg.SourceVmid <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source_vmid must be greater than 0"))
	}
	// --- End validation ---

	modelReq := &model.VMCloneRequest{
		SourceVMID:      int(req.Msg.SourceVmid),
		Name:            req.Msg.Name,
		Full:            req.Msg.Full,
		TargetPool:      req.Msg.TargetPool,
		Description:     req.Msg.Description,
		StartAfterClone: req.Msg.StartAfterClone,
	}

	vm, taskID, err := s.svc.Clone(ctx, modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.CloneVMResponse{
		Vm:     modelVMToProto(vm),
		TaskId: taskID,
	}), nil
}

// StartVM starts a VM.
func (s *VmServiceServer) StartVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Start(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM start initiated"}), nil
}

// StopVM stops a VM.
func (s *VmServiceServer) StopVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Stop(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM stop initiated"}), nil
}

// PauseVM pauses a VM.
func (s *VmServiceServer) PauseVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Pause(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM pause initiated"}), nil
}

// ResumeVM resumes a paused VM.
func (s *VmServiceServer) ResumeVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Resume(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM resume initiated"}), nil
}

// ShutdownVM gracefully shuts down a VM.
func (s *VmServiceServer) ShutdownVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Shutdown(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM shutdown initiated"}), nil
}

// RebootVM reboots a VM.
func (s *VmServiceServer) RebootVM(
	ctx context.Context,
	req *connect.Request[labv1.VmActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Reboot(ctx, int(req.Msg.Vmid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "VM reboot initiated"}), nil
}

// GetVMConsole returns a WebSocket URL and token for console access.
func (s *VmServiceServer) GetVMConsole(
	ctx context.Context,
	req *connect.Request[labv1.GetVMConsoleRequest],
) (*connect.Response[labv1.GetVMConsoleResponse], error) {
	// Determine console type (default to VNC for backward compatibility)
	consoleType := req.Msg.ConsoleType
	if consoleType == labv1.ConsoleType_CONSOLE_TYPE_UNSPECIFIED {
		consoleType = labv1.ConsoleType_CONSOLE_TYPE_VNC
	}

	token, err := s.svc.GetConsoleToken(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	// Select WebSocket URL based on console type
	var wsURL string
	switch consoleType {
	case labv1.ConsoleType_CONSOLE_TYPE_SERIAL:
		wsURL = "/ws/serial"
	case labv1.ConsoleType_CONSOLE_TYPE_VNC:
		wsURL = "/ws/vnc"
	case labv1.ConsoleType_CONSOLE_TYPE_WEBSOCKIFY:
		// websockify would use a different port/URL
		// For now, return unimplemented
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("websockify console type is not yet implemented"))
	default:
		wsURL = "/ws/vnc"
	}

	return connect.NewResponse(&labv1.GetVMConsoleResponse{
		WebsocketUrl: wsURL,
		Token:        token,
		ConsoleType:  consoleType,
	}), nil
}

// GetVMDiagnostics returns comprehensive diagnostic information for a VM.
func (s *VmServiceServer) GetVMDiagnostics(
	ctx context.Context,
	req *connect.Request[labv1.GetVMDiagnosticsRequest],
) (*connect.Response[labv1.GetVMDiagnosticsResponse], error) {
	vm, err := s.svc.GetByVMID(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	diagnostics, err := s.svc.GetDiagnostics(vm)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	return connect.NewResponse(&labv1.GetVMDiagnosticsResponse{
		Diagnostics: diagnostics,
	}), nil
}

// ListVMTemplates returns available VM templates.
func (s *VmServiceServer) ListVMTemplates(
	_ context.Context,
	_ *connect.Request[labv1.ListVMTemplatesRequest],
) (*connect.Response[labv1.ListVMTemplatesResponse], error) {
	templates := s.svc.GetTemplates()

	// Detect host architecture
	sys := sysinfo.New()
	hostArch := sys.HostArch()

	protoTemplates := make([]*labv1.VMTemplate, len(templates))
	for i, t := range templates {
		// Get the URL for the host architecture
		isoURL := t.GetISOURLForArch(hostArch)
		if isoURL == "" {
			// Fallback to other architecture if not available
			if hostArch == "x86_64" {
				isoURL = t.GetISOURLForArch("aarch64")
			} else {
				isoURL = t.GetISOURLForArch("x86_64")
			}
		}

		protoTemplates[i] = &labv1.VMTemplate{
			Id:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Icon:        t.Icon,
			IsoUrl:      isoURL,
			IsoName:     t.ISOName,
			CpuCores:    int32(t.CPUCores),
			MemoryGb:    t.MemoryGB,
			DiskGb:      t.DiskGB,
			Os:          modelOSConfigToProto(t.OS),
			Arch:        hostArch,
		}
	}
	return connect.NewResponse(&labv1.ListVMTemplatesResponse{Templates: protoTemplates}), nil
}

// GetVMLogs returns log entries for a VM.
func (s *VmServiceServer) GetVMLogs(
	ctx context.Context,
	req *connect.Request[labv1.GetVMLogsRequest],
) (*connect.Response[labv1.GetVMLogsResponse], error) {
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 100
	}

	entries, err := s.svc.GetVMLogs(ctx, int(req.Msg.Vmid), limit)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	protoEntries := make([]*labv1.VMLogEntry, len(entries))
	for i, e := range entries {
		protoEntries[i] = &labv1.VMLogEntry{
			Id:        e.ID,
			Vmid:      int32(e.VMID),
			Level:     vmLogLevelStringToProto(e.Level),
			Timestamp: e.Timestamp,
			Source:    e.Source,
			Message:   e.Message,
			Metadata:  e.Metadata,
		}
	}

	return connect.NewResponse(&labv1.GetVMLogsResponse{
		Entries: protoEntries,
		Total:   int32(len(entries)),
	}), nil
}

// GetVMGuestNetworkInterfaces retrieves network interfaces from the QEMU guest agent
func (s *VmServiceServer) GetVMGuestNetworkInterfaces(
	ctx context.Context,
	req *connect.Request[labv1.GetVMGuestNetworkInterfacesRequest],
) (*connect.Response[labv1.GetVMGuestNetworkInterfacesResponse], error) {
	if req.Msg.Vmid <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("vmid is required"))
	}

	interfaces, connected, err := s.svc.GetGuestNetworkInterfaces(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	// Convert model types to proto
	protoInterfaces := make([]*labv1.GuestNetworkInterface, len(interfaces))
	for i, iface := range interfaces {
		protoIPs := make([]*labv1.GuestIPAddress, len(iface.IPAddresses))
		for j, ip := range iface.IPAddresses {
			protoIPs[j] = &labv1.GuestIPAddress{
				Address:     ip.Address,
				Prefix:      int32(ip.Prefix),
				AddressType: ip.AddressType,
			}
		}
		protoInterfaces[i] = &labv1.GuestNetworkInterface{
			Name:         iface.Name,
			MacAddress:   iface.MACAddress,
			IpAddresses:  protoIPs,
		}
	}

	return connect.NewResponse(&labv1.GetVMGuestNetworkInterfacesResponse{
		Interfaces:      protoInterfaces,
		AgentConnected:  connected,
	}), nil
}

// ListPCIDevices returns available PCI devices on the host
func (s *VmServiceServer) ListPCIDevices(
	ctx context.Context,
	req *connect.Request[labv1.ListPCIDevicesRequest],
) (*connect.Response[labv1.ListPCIDevicesResponse], error) {
	devices, iommuAvailable, vfioAvailable, err := s.svc.ListPCIDevices(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}

	protoDevices := make([]*labv1.PCIDevice, len(devices))
	for i, dev := range devices {
		protoDevices[i] = &labv1.PCIDevice{
			Address:     dev.Address,
			VendorId:    dev.VendorID,
			VendorName:  dev.VendorName,
			ProductId:   dev.ProductID,
			ProductName: dev.ProductName,
			Driver:      dev.Driver,
			IommuGroup:  int32(dev.IOMMUGroup),
			Class:       dev.Class,
			ClassName:   dev.ClassName,
		}
	}

	return connect.NewResponse(&labv1.ListPCIDevicesResponse{
		Devices:        protoDevices,
		IommuAvailable: iommuAvailable,
		VfioAvailable:  vfioAvailable,
	}), nil
}

// --- conversion helpers ---

func modelVMStatusToProto(s model.VMStatus) labv1.VmStatus {
	switch s {
	case model.VMStatusRunning:
		return labv1.VmStatus_VM_STATUS_RUNNING
	case model.VMStatusStopped:
		return labv1.VmStatus_VM_STATUS_STOPPED
	case model.VMStatusPaused:
		return labv1.VmStatus_VM_STATUS_PAUSED
	case model.VMStatusSuspended:
		return labv1.VmStatus_VM_STATUS_SUSPENDED
	default:
		return labv1.VmStatus_VM_STATUS_UNSPECIFIED
	}
}

func modelVMToProto(v *model.VM) *labv1.VM {
	return &labv1.VM{
		Id:          v.ID,
		Vmid:        int32(v.VMID),
		Name:        v.Name,
		Node:        v.Node,
		Status:      modelVMStatusToProto(v.Status),
		Cpu:         modelCPUInfoPartialToProto(v.CPU),
		Memory:      modelMemoryInfoToProto(v.Memory),
		Disk:        modelDiskInfoToProto(v.Disk),
		Uptime:      v.Uptime,
		Os:          modelOSConfigToProto(v.OS),
		Arch:        v.Arch,
		MachineType: modelMachineTypeToProto(v.MachineType),
		Bios:        modelBIOSTypeToProto(v.BIOS),
		CpuModel:    v.CPUModel,
		Network:     modelNetworkConfigsToProto(v.Network),
		Ip:          v.IP,
		Tags:        v.Tags,
		Ha:          v.HA,
		Description: v.Description,
		NestedVirt:  v.NestedVirt,
		StartOnBoot: v.StartOnBoot,
		Agent:       v.Agent,
		Tpm:         v.TPM,
		SecureBoot:  v.SecureBoot,
		BootOrder:   v.BootOrder,
	}
}

func protoCreateVMRequestToModel(r *labv1.CreateVMRequest) *model.VMCreateRequest {
	return &model.VMCreateRequest{
		Name:        r.Name,
		Node:        r.Node,
		Tags:        r.Tags,
		Description: r.Description,
		StartOnBoot: r.StartOnBoot,
		OS:          protoOSConfigToModel(r.Os),
		Arch:        r.Arch,
		MachineType: protoMachineTypeToModel(r.MachineType),
		BIOS:        protoBIOSTypeToModel(r.Bios),
		Agent:       r.Agent,
		ISO:         r.Iso,
		ISOURL:      r.IsoUrl,
		ISOName:     r.IsoName,
		Disk:        r.DiskGb,
		CPUSockets:  int(r.CpuSockets),
		CPUCores:    int(r.CpuCores),
		CPUModel:    r.CpuModel,
		NestedVirt:  r.NestedVirt,
		Memory:      r.MemoryGb,
		Network:     protoNetworkConfigsToModel(r.Network),
		TPM:         r.Tpm,
		SecureBoot:  r.SecureBoot,
		BootOrder:   r.BootOrder,
	}
}

func vmLogLevelStringToProto(level string) labv1.VMLogLevel {
	switch level {
	case "DEBUG":
		return labv1.VMLogLevel_VM_LOG_LEVEL_DEBUG
	case "INFO":
		return labv1.VMLogLevel_VM_LOG_LEVEL_INFO
	case "WARNING":
		return labv1.VMLogLevel_VM_LOG_LEVEL_WARNING
	case "ERROR":
		return labv1.VMLogLevel_VM_LOG_LEVEL_ERROR
	case "CRITICAL":
		return labv1.VMLogLevel_VM_LOG_LEVEL_CRITICAL
	default:
		return labv1.VMLogLevel_VM_LOG_LEVEL_UNSPECIFIED
	}
}
