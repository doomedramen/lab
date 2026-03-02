package connectsvc

import (
	"context"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// ContainerServiceServer implements labv1connect.ContainerServiceHandler.
type ContainerServiceServer struct {
	labv1connect.UnimplementedContainerServiceHandler
	svc *service.ContainerService
}

// NewContainerServiceServer creates a new ContainerServiceServer.
func NewContainerServiceServer(svc *service.ContainerService) *ContainerServiceServer {
	return &ContainerServiceServer{svc: svc}
}

// ListContainers returns all containers, optionally filtered by node.
func (s *ContainerServiceServer) ListContainers(
	ctx context.Context,
	req *connect.Request[labv1.ListContainersRequest],
) (*connect.Response[labv1.ListContainersResponse], error) {
	var containers []*model.Container
	var err error
	if req.Msg.Node != "" {
		containers, err = s.svc.GetByNode(ctx, req.Msg.Node)
	} else {
		containers, err = s.svc.GetAll(ctx)
	}
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	proto := make([]*labv1.Container, len(containers))
	for i, c := range containers {
		proto[i] = modelContainerToProto(c)
	}
	return connect.NewResponse(&labv1.ListContainersResponse{Containers: proto}), nil
}

// GetContainer returns a single container by CTID.
func (s *ContainerServiceServer) GetContainer(
	ctx context.Context,
	req *connect.Request[labv1.GetContainerRequest],
) (*connect.Response[labv1.GetContainerResponse], error) {
	ct, err := s.svc.GetByCTID(ctx, int(req.Msg.Ctid))
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetContainerResponse{Container: modelContainerToProto(ct)}), nil
}

// CreateContainer creates a new container.
func (s *ContainerServiceServer) CreateContainer(
	ctx context.Context,
	req *connect.Request[labv1.CreateContainerRequest],
) (*connect.Response[labv1.CreateContainerResponse], error) {
	modelReq := &model.ContainerCreateRequest{
		Name:         req.Msg.Name,
		Node:         req.Msg.Node,
		CPUCores:     int(req.Msg.CpuCores),
		Memory:       req.Msg.MemoryGb,
		Disk:         req.Msg.DiskGb,
		OS:           req.Msg.Os,
		Tags:         req.Msg.Tags,
		Unprivileged: req.Msg.Unprivileged,
		Description:  req.Msg.Description,
		StartOnBoot:  req.Msg.StartOnBoot,
	}
	ct, err := s.svc.Create(ctx, modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.CreateContainerResponse{Container: modelContainerToProto(ct)}), nil
}

// UpdateContainer updates an existing container.
func (s *ContainerServiceServer) UpdateContainer(
	ctx context.Context,
	req *connect.Request[labv1.UpdateContainerRequest],
) (*connect.Response[labv1.UpdateContainerResponse], error) {
	modelReq := &model.ContainerUpdateRequest{
		Name:        req.Msg.Name,
		CPUCores:    int(req.Msg.CpuCores),
		Memory:      req.Msg.MemoryGb,
		Tags:        req.Msg.Tags,
		Description: req.Msg.Description,
	}
	ct, err := s.svc.Update(ctx, int(req.Msg.Ctid), modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.UpdateContainerResponse{Container: modelContainerToProto(ct)}), nil
}

// DeleteContainer deletes a container.
func (s *ContainerServiceServer) DeleteContainer(
	ctx context.Context,
	req *connect.Request[labv1.DeleteContainerRequest],
) (*connect.Response[labv1.DeleteContainerResponse], error) {
	if err := s.svc.Delete(ctx, int(req.Msg.Ctid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.DeleteContainerResponse{}), nil
}

// StartContainer starts a container.
func (s *ContainerServiceServer) StartContainer(
	ctx context.Context,
	req *connect.Request[labv1.ContainerActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Start(ctx, int(req.Msg.Ctid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Container start initiated"}), nil
}

// StopContainer stops a container.
func (s *ContainerServiceServer) StopContainer(
	ctx context.Context,
	req *connect.Request[labv1.ContainerActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Stop(ctx, int(req.Msg.Ctid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Container stop initiated"}), nil
}

// RebootContainer reboots a container.
func (s *ContainerServiceServer) RebootContainer(
	ctx context.Context,
	req *connect.Request[labv1.ContainerActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Reboot(ctx, int(req.Msg.Ctid)); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Container reboot initiated"}), nil
}

// --- conversion helpers ---

func modelContainerStatusToProto(s model.ContainerStatus) labv1.ContainerStatus {
	switch s {
	case model.ContainerStatusRunning:
		return labv1.ContainerStatus_CONTAINER_STATUS_RUNNING
	case model.ContainerStatusStopped:
		return labv1.ContainerStatus_CONTAINER_STATUS_STOPPED
	case model.ContainerStatusFrozen:
		return labv1.ContainerStatus_CONTAINER_STATUS_FROZEN
	default:
		return labv1.ContainerStatus_CONTAINER_STATUS_UNSPECIFIED
	}
}

func modelContainerToProto(c *model.Container) *labv1.Container {
	return &labv1.Container{
		Id:           c.ID,
		Ctid:         int32(c.CTID),
		Name:         c.Name,
		Node:         c.Node,
		Status:       modelContainerStatusToProto(c.Status),
		Cpu:          modelCPUInfoPartialToProto(c.CPU),
		Memory:       modelMemoryInfoToProto(c.Memory),
		Disk:         modelDiskInfoToProto(c.Disk),
		Uptime:       c.Uptime,
		Os:           c.OS,
		Ip:           c.IP,
		Tags:         c.Tags,
		Unprivileged: c.Unprivileged,
		Swap:         modelSwapInfoToProto(c.Swap),
		Description:  c.Description,
		StartOnBoot:  c.StartOnBoot,
	}
}
