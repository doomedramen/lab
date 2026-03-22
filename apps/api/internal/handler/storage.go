package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// StorageServiceServer implements the StorageService Connect RPC server
type StorageServiceServer struct {
	storageService *service.StorageService
}

// NewStorageServiceServer creates a new storage service server
func NewStorageServiceServer(storageService *service.StorageService) *StorageServiceServer {
	return &StorageServiceServer{storageService: storageService}
}

var _ labv1connect.StorageServiceHandler = (*StorageServiceServer)(nil)

// ListStoragePools lists storage pools
func (s *StorageServiceServer) ListStoragePools(
	ctx context.Context,
	req *connect.Request[labv1.ListStoragePoolsRequest],
) (*connect.Response[labv1.ListStoragePoolsResponse], error) {
	pools, total, err := s.storageService.ListStoragePools(
		ctx,
		req.Msg.Type,
		req.Msg.Status,
		req.Msg.EnabledOnly,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListStoragePoolsResponse{
		Pools: pools,
		Total: total,
	}), nil
}

// GetStoragePool gets a storage pool
func (s *StorageServiceServer) GetStoragePool(
	ctx context.Context,
	req *connect.Request[labv1.GetStoragePoolRequest],
) (*connect.Response[labv1.GetStoragePoolResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool ID is required"))
	}

	pool, err := s.storageService.GetStoragePool(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetStoragePoolResponse{
		Pool: pool,
	}), nil
}

// CreateStoragePool creates a storage pool
func (s *StorageServiceServer) CreateStoragePool(
	ctx context.Context,
	req *connect.Request[labv1.CreateStoragePoolRequest],
) (*connect.Response[labv1.CreateStoragePoolResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool name is required"))
	}

	pool, err := s.storageService.CreateStoragePool(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateStoragePoolResponse{
		Pool: pool,
	}), nil
}

// UpdateStoragePool updates a storage pool
func (s *StorageServiceServer) UpdateStoragePool(
	ctx context.Context,
	req *connect.Request[labv1.UpdateStoragePoolRequest],
) (*connect.Response[labv1.UpdateStoragePoolResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool ID is required"))
	}

	pool, err := s.storageService.UpdateStoragePool(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateStoragePoolResponse{
		Pool: pool,
	}), nil
}

// DeleteStoragePool deletes a storage pool
func (s *StorageServiceServer) DeleteStoragePool(
	ctx context.Context,
	req *connect.Request[labv1.DeleteStoragePoolRequest],
) (*connect.Response[labv1.DeleteStoragePoolResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool ID is required"))
	}

	if err := s.storageService.DeleteStoragePool(ctx, req.Msg.Id, req.Msg.Force); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteStoragePoolResponse{}), nil
}

// RefreshStoragePool refreshes pool statistics
func (s *StorageServiceServer) RefreshStoragePool(
	ctx context.Context,
	req *connect.Request[labv1.RefreshStoragePoolRequest],
) (*connect.Response[labv1.RefreshStoragePoolResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool ID is required"))
	}

	pool, err := s.storageService.RefreshStoragePool(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RefreshStoragePoolResponse{
		Pool: pool,
	}), nil
}

// ListStorageDisks lists disks in a pool
func (s *StorageServiceServer) ListStorageDisks(
	ctx context.Context,
	req *connect.Request[labv1.ListStorageDisksRequest],
) (*connect.Response[labv1.ListStorageDisksResponse], error) {
	disks, total, err := s.storageService.ListStorageDisks(
		ctx,
		req.Msg.PoolId,
		int(req.Msg.Vmid),
		req.Msg.UnassignedOnly,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListStorageDisksResponse{
		Disks: disks,
		Total: total,
	}), nil
}

// CreateStorageDisk creates a disk
func (s *StorageServiceServer) CreateStorageDisk(
	ctx context.Context,
	req *connect.Request[labv1.CreateStorageDiskRequest],
) (*connect.Response[labv1.CreateStorageDiskResponse], error) {
	if req.Msg.PoolId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool ID is required"))
	}

	if req.Msg.SizeBytes <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("size must be positive"))
	}

	disk, err := s.storageService.CreateStorageDisk(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateStorageDiskResponse{
		Disk: disk,
	}), nil
}

// ResizeStorageDisk resizes a disk
func (s *StorageServiceServer) ResizeStorageDisk(
	ctx context.Context,
	req *connect.Request[labv1.ResizeStorageDiskRequest],
) (*connect.Response[labv1.ResizeStorageDiskResponse], error) {
	if req.Msg.DiskId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disk ID is required"))
	}

	if req.Msg.NewSizeBytes <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("new size must be positive"))
	}

	disk, err := s.storageService.ResizeStorageDisk(ctx, req.Msg.DiskId, req.Msg.NewSizeBytes, req.Msg.Shrink)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ResizeStorageDiskResponse{
		Disk: disk,
	}), nil
}

// DeleteStorageDisk deletes a disk
func (s *StorageServiceServer) DeleteStorageDisk(
	ctx context.Context,
	req *connect.Request[labv1.DeleteStorageDiskRequest],
) (*connect.Response[labv1.DeleteStorageDiskResponse], error) {
	if req.Msg.DiskId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disk ID is required"))
	}

	if err := s.storageService.DeleteStorageDisk(ctx, req.Msg.DiskId, req.Msg.Purge); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteStorageDiskResponse{}), nil
}

// MoveStorageDisk moves a disk to another pool
func (s *StorageServiceServer) MoveStorageDisk(
	ctx context.Context,
	req *connect.Request[labv1.MoveStorageDiskRequest],
) (*connect.Response[labv1.MoveStorageDiskResponse], error) {
	if req.Msg.DiskId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disk ID is required"))
	}

	if req.Msg.TargetPoolId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target pool ID is required"))
	}

	disk, taskID, err := s.storageService.MoveStorageDisk(
		ctx,
		req.Msg.DiskId,
		req.Msg.TargetPoolId,
		req.Msg.Format,
		req.Msg.DeleteOriginal,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.MoveStorageDiskResponse{
		Disk:   disk,
		TaskId: taskID,
	}), nil
}

// ListStorageContent lists content in a pool
func (s *StorageServiceServer) ListStorageContent(
	ctx context.Context,
	req *connect.Request[labv1.ListStorageContentRequest],
) (*connect.Response[labv1.ListStorageContentResponse], error) {
	if req.Msg.PoolId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("pool_id is required"))
	}

	contents, err := s.storageService.ListStorageContent(ctx, req.Msg.PoolId, req.Msg.ContentType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoContents := make([]*labv1.StorageContent, len(contents))
	for i, c := range contents {
		protoContents[i] = &labv1.StorageContent{
			Id:         c.ID,
			PoolId:     c.PoolID,
			Name:       c.Name,
			Type:       c.Type,
			SizeBytes:  c.SizeBytes,
			Format:     c.Format,
			CreatedAt:  c.CreatedAt,
			Description: c.Description,
		}
	}

	return connect.NewResponse(&labv1.ListStorageContentResponse{
		Content: protoContents,
		Total:   int32(len(contents)),
	}), nil
}

// ListVMDisks lists disks attached to a VM
func (s *StorageServiceServer) ListVMDisks(
	ctx context.Context,
	req *connect.Request[labv1.ListVMDisksRequest],
) (*connect.Response[labv1.ListVMDisksResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VM ID is required"))
	}

	disks, err := s.storageService.ListVMDisks(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListVMDisksResponse{
		Disks: disks,
	}), nil
}

// AttachVMdisk attaches a disk to a VM
func (s *StorageServiceServer) AttachVMdisk(
	ctx context.Context,
	req *connect.Request[labv1.AttachVMdiskRequest],
) (*connect.Response[labv1.AttachVMdiskResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VM ID is required"))
	}
	if req.Msg.DiskPath == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("disk path is required"))
	}

	bus := protoToModelDiskBus(req.Msg.Bus)
	target, err := s.storageService.AttachDiskToVM(ctx, int(req.Msg.Vmid), req.Msg.DiskPath, bus, req.Msg.Readonly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Get the attached disk info
	disks, err := s.storageService.ListVMDisks(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var attachedDisk *labv1.VMDisk
	for _, d := range disks {
		if d.Target == target {
			attachedDisk = d
			break
		}
	}

	return connect.NewResponse(&labv1.AttachVMdiskResponse{
		Disk:   attachedDisk,
		Target: target,
	}), nil
}

// DetachVMdisk detaches a disk from a VM
func (s *StorageServiceServer) DetachVMdisk(
	ctx context.Context,
	req *connect.Request[labv1.DetachVMdiskRequest],
) (*connect.Response[labv1.DetachVMdiskResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VM ID is required"))
	}
	if req.Msg.Target == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}

	if err := s.storageService.DetachDiskFromVM(ctx, int(req.Msg.Vmid), req.Msg.Target, req.Msg.DeleteDisk); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DetachVMdiskResponse{}), nil
}

// ResizeVMdisk resizes a VM disk
func (s *StorageServiceServer) ResizeVMdisk(
	ctx context.Context,
	req *connect.Request[labv1.ResizeVMdiskRequest],
) (*connect.Response[labv1.ResizeVMdiskResponse], error) {
	if req.Msg.Vmid == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("VM ID is required"))
	}
	if req.Msg.Target == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target is required"))
	}

	if err := s.storageService.ResizeVMDisk(ctx, int(req.Msg.Vmid), req.Msg.Target, float64(req.Msg.NewSizeBytes)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Get the resized disk info
	disks, err := s.storageService.ListVMDisks(ctx, int(req.Msg.Vmid))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	var resizedDisk *labv1.VMDisk
	for _, d := range disks {
		if d.Target == req.Msg.Target {
			resizedDisk = d
			break
		}
	}

	return connect.NewResponse(&labv1.ResizeVMdiskResponse{
		Disk: resizedDisk,
	}), nil
}

// protoToModelDiskBus converts labv1.DiskBus to model.DiskBus
func protoToModelDiskBus(b labv1.DiskBus) model.DiskBus {
	switch b {
	case labv1.DiskBus_DISK_BUS_VIRTIO:
		return model.DiskBusVirtIO
	case labv1.DiskBus_DISK_BUS_SATA:
		return model.DiskBusSATA
	case labv1.DiskBus_DISK_BUS_SCSI:
		return model.DiskBusSCSI
	case labv1.DiskBus_DISK_BUS_IDE:
		return model.DiskBusIDE
	case labv1.DiskBus_DISK_BUS_USB:
		return model.DiskBusUSB
	case labv1.DiskBus_DISK_BUS_NVME:
		return model.DiskBusNVMe
	default:
		return model.DiskBusVirtIO
	}
}
