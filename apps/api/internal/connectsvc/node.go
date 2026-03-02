package connectsvc

import (
	"context"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// NodeServiceServer implements labv1connect.NodeServiceHandler.
type NodeServiceServer struct {
	labv1connect.UnimplementedNodeServiceHandler
	svc *service.NodeService
}

// NewNodeServiceServer creates a new NodeServiceServer.
func NewNodeServiceServer(svc *service.NodeService) *NodeServiceServer {
	return &NodeServiceServer{svc: svc}
}

// ListNodes returns all nodes.
func (s *NodeServiceServer) ListNodes(
	ctx context.Context,
	_ *connect.Request[labv1.ListNodesRequest],
) (*connect.Response[labv1.ListNodesResponse], error) {
	nodes, err := s.svc.GetAll(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	protoNodes := make([]*labv1.Node, len(nodes))
	for i, n := range nodes {
		protoNodes[i] = modelNodeToProto(n)
	}
	return connect.NewResponse(&labv1.ListNodesResponse{Nodes: protoNodes}), nil
}

// GetNode returns a single node by ID.
func (s *NodeServiceServer) GetNode(
	ctx context.Context,
	req *connect.Request[labv1.GetNodeRequest],
) (*connect.Response[labv1.GetNodeResponse], error) {
	node, err := s.svc.GetByID(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetNodeResponse{Node: modelNodeToProto(node)}), nil
}

// RebootNode reboots a node.
func (s *NodeServiceServer) RebootNode(
	ctx context.Context,
	req *connect.Request[labv1.NodeActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Reboot(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Node reboot initiated"}), nil
}

// ShutdownNode shuts down a node.
func (s *NodeServiceServer) ShutdownNode(
	ctx context.Context,
	req *connect.Request[labv1.NodeActionRequest],
) (*connect.Response[labv1.ActionResponse], error) {
	if err := s.svc.Shutdown(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.ActionResponse{Success: true, Message: "Node shutdown initiated"}), nil
}

// GetHostShellToken generates a one-time token for WebSocket shell access.
func (s *NodeServiceServer) GetHostShellToken(
	_ context.Context,
	req *connect.Request[labv1.GetHostShellTokenRequest],
) (*connect.Response[labv1.GetHostShellTokenResponse], error) {
	token, err := s.svc.GetHostShellToken(req.Msg.NodeId)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetHostShellTokenResponse{Token: token}), nil
}

// --- conversion helpers ---

func modelNodeStatusToProto(s model.NodeStatus) labv1.NodeStatus {
	switch s {
	case model.NodeStatusOnline:
		return labv1.NodeStatus_NODE_STATUS_ONLINE
	case model.NodeStatusOffline:
		return labv1.NodeStatus_NODE_STATUS_OFFLINE
	case model.NodeStatusMaintenance:
		return labv1.NodeStatus_NODE_STATUS_MAINTENANCE
	default:
		return labv1.NodeStatus_NODE_STATUS_UNSPECIFIED
	}
}

func modelNodeToProto(n *model.HostNode) *labv1.Node {
	return &labv1.Node{
		Id:         n.ID,
		Name:       n.Name,
		Status:     modelNodeStatusToProto(n.Status),
		Ip:         n.IP,
		Cpu:        modelCPUInfoToProto(n.CPU),
		Memory:     modelMemoryInfoToProto(n.Memory),
		Disk:       modelDiskInfoToProto(n.Disk),
		Uptime:     n.Uptime,
		Kernel:     n.Kernel,
		Version:    n.Version,
		Vms:        int32(n.VMs),
		Containers: int32(n.Containers),
		CpuModel:   n.CPUModel,
		LoadAvg:    modelLoadAvgToProto(n.LoadAvg),
		NetworkIn:  n.NetworkIn,
		NetworkOut: n.NetworkOut,
		Arch:       n.Arch,
	}
}
