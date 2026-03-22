package connectsvc

import (
	"context"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// StackServiceServer implements labv1connect.StackServiceHandler.
type StackServiceServer struct {
	labv1connect.UnimplementedStackServiceHandler
	svc *service.StackService
}

// NewStackServiceServer creates a new StackServiceServer.
func NewStackServiceServer(svc *service.StackService) *StackServiceServer {
	return &StackServiceServer{svc: svc}
}

// ListStacks returns all stacks.
func (s *StackServiceServer) ListStacks(
	ctx context.Context,
	_ *connect.Request[labv1.ListStacksRequest],
) (*connect.Response[labv1.ListStacksResponse], error) {
	stacks, err := s.svc.GetAll(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	proto := make([]*labv1.Stack, len(stacks))
	for i, st := range stacks {
		proto[i] = modelStackToProto(st)
	}
	return connect.NewResponse(&labv1.ListStacksResponse{Stacks: proto}), nil
}

// GetStack returns a single stack by ID.
func (s *StackServiceServer) GetStack(
	ctx context.Context,
	req *connect.Request[labv1.GetStackRequest],
) (*connect.Response[labv1.GetStackResponse], error) {
	st, err := s.svc.GetByID(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetStackResponse{Stack: modelStackToProto(st)}), nil
}

// CreateStack creates a new Docker Compose stack.
func (s *StackServiceServer) CreateStack(
	ctx context.Context,
	req *connect.Request[labv1.CreateStackRequest],
) (*connect.Response[labv1.CreateStackResponse], error) {
	modelReq := &model.StackCreateRequest{
		Name:    req.Msg.Name,
		Compose: req.Msg.Compose,
		Env:     req.Msg.Env,
	}
	st, err := s.svc.Create(ctx, modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.CreateStackResponse{Stack: modelStackToProto(st)}), nil
}

// UpdateStack updates the compose/env files for an existing stack.
func (s *StackServiceServer) UpdateStack(
	ctx context.Context,
	req *connect.Request[labv1.UpdateStackRequest],
) (*connect.Response[labv1.UpdateStackResponse], error) {
	modelReq := &model.StackUpdateRequest{
		Compose: req.Msg.Compose,
		Env:     req.Msg.Env,
	}
	st, err := s.svc.Update(ctx, req.Msg.Id, modelReq)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.UpdateStackResponse{Stack: modelStackToProto(st)}), nil
}

// DeleteStack deletes a stack (docker compose down + remove folder).
func (s *StackServiceServer) DeleteStack(
	ctx context.Context,
	req *connect.Request[labv1.DeleteStackRequest],
) (*connect.Response[labv1.DeleteStackResponse], error) {
	if err := s.svc.Delete(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.DeleteStackResponse{}), nil
}

// StartStack runs docker compose up -d.
func (s *StackServiceServer) StartStack(
	ctx context.Context,
	req *connect.Request[labv1.StackActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Start(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Stack started"}), nil
}

// StopStack runs docker compose stop.
func (s *StackServiceServer) StopStack(
	ctx context.Context,
	req *connect.Request[labv1.StackActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Stop(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Stack stopped"}), nil
}

// RestartStack runs docker compose restart.
func (s *StackServiceServer) RestartStack(
	ctx context.Context,
	req *connect.Request[labv1.StackActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Restart(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Stack restarted"}), nil
}

// UpdateStackImages pulls images and recreates containers.
func (s *StackServiceServer) UpdateStackImages(
	ctx context.Context,
	req *connect.Request[labv1.StackActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.UpdateImages(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Images updated and stack restarted"}), nil
}

// DownStack runs docker compose down.
func (s *StackServiceServer) DownStack(
	ctx context.Context,
	req *connect.Request[labv1.StackActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Down(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Stack down"}), nil
}

// GetContainerToken generates a one-time token for container bash access.
func (s *StackServiceServer) GetContainerToken(
	_ context.Context,
	req *connect.Request[labv1.GetContainerTokenRequest],
) (*connect.Response[labv1.GetContainerTokenResponse], error) {
	token, err := s.svc.GetContainerToken(req.Msg.StackId, req.Msg.ContainerName)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetContainerTokenResponse{Token: token}), nil
}

// GetStackLogsToken generates a one-time token for stack log streaming.
func (s *StackServiceServer) GetStackLogsToken(
	_ context.Context,
	req *connect.Request[labv1.GetStackLogsTokenRequest],
) (*connect.Response[labv1.GetStackLogsTokenResponse], error) {
	token, err := s.svc.GetStackLogsToken(req.Msg.StackId)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetStackLogsTokenResponse{Token: token}), nil
}

// --- conversion helpers ---

func modelStackStatusToProto(s model.StackStatus) labv1.StackStatus {
	switch s {
	case model.StackStatusRunning:
		return labv1.StackStatus_STACK_STATUS_RUNNING
	case model.StackStatusPartiallyRunning:
		return labv1.StackStatus_STACK_STATUS_PARTIALLY_RUNNING
	case model.StackStatusStopped:
		return labv1.StackStatus_STACK_STATUS_STOPPED
	default:
		return labv1.StackStatus_STACK_STATUS_UNSPECIFIED
	}
}

func modelDockerContainerToProto(c model.DockerContainer) *labv1.DockerContainer {
	return &labv1.DockerContainer{
		ServiceName:   c.ServiceName,
		ContainerName: c.ContainerName,
		ContainerId:   c.ContainerID,
		Image:         c.Image,
		Status:        c.Status,
		State:         c.State,
		Ports:         c.Ports,
	}
}

func modelStackToProto(st *model.DockerStack) *labv1.Stack {
	containers := make([]*labv1.DockerContainer, len(st.Containers))
	for i, c := range st.Containers {
		containers[i] = modelDockerContainerToProto(c)
	}
	return &labv1.Stack{
		Id:         st.ID,
		Name:       st.Name,
		Compose:    st.Compose,
		Env:        st.Env,
		Status:     modelStackStatusToProto(st.Status),
		Containers: containers,
		CreatedAt:  st.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
