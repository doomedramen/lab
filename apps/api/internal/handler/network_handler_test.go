package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

func TestNetworkHandler_GetNetwork_MissingId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.GetNetwork(context.Background(), connect.NewRequest(&labv1.GetNetworkRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_CreateNetwork_MissingName(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.CreateNetwork(context.Background(), connect.NewRequest(&labv1.CreateNetworkRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_UpdateNetwork_MissingId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.UpdateNetwork(context.Background(), connect.NewRequest(&labv1.UpdateNetworkRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_DeleteNetwork_MissingId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.DeleteNetwork(context.Background(), connect.NewRequest(&labv1.DeleteNetworkRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_CreateVmNetworkInterface_MissingNetworkId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.CreateVmNetworkInterface(context.Background(), connect.NewRequest(&labv1.CreateVmNetworkInterfaceRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_UpdateVmNetworkInterface_MissingId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.UpdateVmNetworkInterface(context.Background(), connect.NewRequest(&labv1.UpdateVmNetworkInterfaceRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_DeleteVmNetworkInterface_MissingId(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.DeleteVmNetworkInterface(context.Background(), connect.NewRequest(&labv1.DeleteVmNetworkInterfaceRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_CreateBridge_MissingName(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.CreateBridge(context.Background(), connect.NewRequest(&labv1.CreateBridgeRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_DeleteBridge_MissingName(t *testing.T) {
	h := &NetworkServiceServer{}
	_, err := h.DeleteBridge(context.Background(), connect.NewRequest(&labv1.DeleteBridgeRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_AddDHCPStaticLease_MissingFields(t *testing.T) {
	h := &NetworkServiceServer{}

	// Missing all fields
	_, err := h.AddDHCPStaticLease(context.Background(), connect.NewRequest(&labv1.AddDHCPStaticLeaseRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing MAC
	_, err = h.AddDHCPStaticLease(context.Background(), connect.NewRequest(&labv1.AddDHCPStaticLeaseRequest{
		NetworkId: "net-1",
		IpAddress: "192.168.1.100",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing IP
	_, err = h.AddDHCPStaticLease(context.Background(), connect.NewRequest(&labv1.AddDHCPStaticLeaseRequest{
		NetworkId:  "net-1",
		MacAddress: "aa:bb:cc:dd:ee:ff",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNetworkHandler_RemoveDHCPStaticLease_MissingFields(t *testing.T) {
	h := &NetworkServiceServer{}

	// Missing both
	_, err := h.RemoveDHCPStaticLease(context.Background(), connect.NewRequest(&labv1.RemoveDHCPStaticLeaseRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing MAC only
	_, err = h.RemoveDHCPStaticLease(context.Background(), connect.NewRequest(&labv1.RemoveDHCPStaticLeaseRequest{
		NetworkId: "net-1",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}
