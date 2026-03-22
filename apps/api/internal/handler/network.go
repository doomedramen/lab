package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// NetworkServiceServer implements the NetworkService Connect RPC server
type NetworkServiceServer struct {
	networkService *service.NetworkService
}

// NewNetworkServiceServer creates a new network service server
func NewNetworkServiceServer(networkService *service.NetworkService) *NetworkServiceServer {
	return &NetworkServiceServer{networkService: networkService}
}

var _ labv1connect.NetworkServiceHandler = (*NetworkServiceServer)(nil)

// ListNetworks lists virtual networks
func (s *NetworkServiceServer) ListNetworks(
	ctx context.Context,
	req *connect.Request[labv1.ListNetworksRequest],
) (*connect.Response[labv1.ListNetworksResponse], error) {
	networks, total, err := s.networkService.ListNetworks(
		ctx,
		req.Msg.Type,
		req.Msg.Status,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListNetworksResponse{
		Networks: networks,
		Total:    total,
	}), nil
}

// GetNetwork gets a network
func (s *NetworkServiceServer) GetNetwork(
	ctx context.Context,
	req *connect.Request[labv1.GetNetworkRequest],
) (*connect.Response[labv1.GetNetworkResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID is required"))
	}

	network, err := s.networkService.GetNetwork(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetNetworkResponse{
		Network: network,
	}), nil
}

// CreateNetwork creates a network
func (s *NetworkServiceServer) CreateNetwork(
	ctx context.Context,
	req *connect.Request[labv1.CreateNetworkRequest],
) (*connect.Response[labv1.CreateNetworkResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network name is required"))
	}

	network, err := s.networkService.CreateNetwork(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateNetworkResponse{
		Network: network,
	}), nil
}

// UpdateNetwork updates a network
func (s *NetworkServiceServer) UpdateNetwork(
	ctx context.Context,
	req *connect.Request[labv1.UpdateNetworkRequest],
) (*connect.Response[labv1.UpdateNetworkResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID is required"))
	}

	network, err := s.networkService.UpdateNetwork(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateNetworkResponse{
		Network: network,
	}), nil
}

// DeleteNetwork deletes a network
func (s *NetworkServiceServer) DeleteNetwork(
	ctx context.Context,
	req *connect.Request[labv1.DeleteNetworkRequest],
) (*connect.Response[labv1.DeleteNetworkResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID is required"))
	}

	if err := s.networkService.DeleteNetwork(ctx, req.Msg.Id, req.Msg.Force); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteNetworkResponse{}), nil
}

// ListVmNetworkInterfaces lists network interfaces
func (s *NetworkServiceServer) ListVmNetworkInterfaces(
	ctx context.Context,
	req *connect.Request[labv1.ListVmNetworkInterfacesRequest],
) (*connect.Response[labv1.ListVmNetworkInterfacesResponse], error) {
	interfaces, total, err := s.networkService.ListNetworkInterfaces(
		ctx,
		req.Msg.NetworkId,
		req.Msg.EntityId,
		req.Msg.EntityType,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListVmNetworkInterfacesResponse{
		Interfaces: interfaces,
		Total:      total,
	}), nil
}

// CreateVmNetworkInterface creates a network interface
func (s *NetworkServiceServer) CreateVmNetworkInterface(
	ctx context.Context,
	req *connect.Request[labv1.CreateVmNetworkInterfaceRequest],
) (*connect.Response[labv1.CreateVmNetworkInterfaceResponse], error) {
	if req.Msg.NetworkId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID is required"))
	}

	iface, err := s.networkService.CreateNetworkInterface(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateVmNetworkInterfaceResponse{
		Interface: iface,
	}), nil
}

// UpdateVmNetworkInterface updates a network interface
func (s *NetworkServiceServer) UpdateVmNetworkInterface(
	ctx context.Context,
	req *connect.Request[labv1.UpdateVmNetworkInterfaceRequest],
) (*connect.Response[labv1.UpdateVmNetworkInterfaceResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("interface ID is required"))
	}

	iface, err := s.networkService.UpdateNetworkInterface(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.UpdateVmNetworkInterfaceResponse{
		Interface: iface,
	}), nil
}

// DeleteVmNetworkInterface deletes a network interface
func (s *NetworkServiceServer) DeleteVmNetworkInterface(
	ctx context.Context,
	req *connect.Request[labv1.DeleteVmNetworkInterfaceRequest],
) (*connect.Response[labv1.DeleteVmNetworkInterfaceResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("interface ID is required"))
	}

	if err := s.networkService.DeleteNetworkInterface(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteVmNetworkInterfaceResponse{}), nil
}

// ListBridges lists bridges
func (s *NetworkServiceServer) ListBridges(
	ctx context.Context,
	req *connect.Request[labv1.ListBridgesRequest],
) (*connect.Response[labv1.ListBridgesResponse], error) {
	bridges, total, err := s.networkService.ListBridges(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.ListBridgesResponse{
		Bridges: bridges,
		Total:   total,
	}), nil
}

// CreateBridge creates a bridge
func (s *NetworkServiceServer) CreateBridge(
	ctx context.Context,
	req *connect.Request[labv1.CreateBridgeRequest],
) (*connect.Response[labv1.CreateBridgeResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bridge name is required"))
	}

	bridge, err := s.networkService.CreateBridge(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.CreateBridgeResponse{
		Bridge: bridge,
	}), nil
}

// DeleteBridge deletes a bridge
func (s *NetworkServiceServer) DeleteBridge(
	ctx context.Context,
	req *connect.Request[labv1.DeleteBridgeRequest],
) (*connect.Response[labv1.DeleteBridgeResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bridge name is required"))
	}

	if err := s.networkService.DeleteBridge(ctx, req.Msg.Name, req.Msg.Force); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.DeleteBridgeResponse{}), nil
}

// GetDHCPLeases gets DHCP leases
func (s *NetworkServiceServer) GetDHCPLeases(
	ctx context.Context,
	req *connect.Request[labv1.GetDHCPLeasesRequest],
) (*connect.Response[labv1.GetDHCPLeasesResponse], error) {
	leases, total, err := s.networkService.GetDHCPLeases(ctx, req.Msg.NetworkId, req.Msg.StaticOnly)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.GetDHCPLeasesResponse{
		Leases: leases,
		Total:  total,
	}), nil
}

// AddDHCPStaticLease adds a static DHCP lease
func (s *NetworkServiceServer) AddDHCPStaticLease(
	ctx context.Context,
	req *connect.Request[labv1.AddDHCPStaticLeaseRequest],
) (*connect.Response[labv1.AddDHCPStaticLeaseResponse], error) {
	if req.Msg.NetworkId == "" || req.Msg.MacAddress == "" || req.Msg.IpAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID, MAC address, and IP address are required"))
	}

	lease, err := s.networkService.AddDHCPStaticLease(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.AddDHCPStaticLeaseResponse{
		Lease: lease,
	}), nil
}

// RemoveDHCPStaticLease removes a static DHCP lease
func (s *NetworkServiceServer) RemoveDHCPStaticLease(
	ctx context.Context,
	req *connect.Request[labv1.RemoveDHCPStaticLeaseRequest],
) (*connect.Response[labv1.RemoveDHCPStaticLeaseResponse], error) {
	if req.Msg.NetworkId == "" || req.Msg.MacAddress == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("network ID and MAC address are required"))
	}

	if err := s.networkService.RemoveDHCPStaticLease(ctx, req.Msg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&labv1.RemoveDHCPStaticLeaseResponse{}), nil
}
