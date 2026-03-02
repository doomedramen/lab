package handler

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
)

// TestProxyHandler_CreateProxyHost_MissingDomain verifies that an empty domain
// triggers CodeInvalidArgument before the service is called.
func TestProxyHandler_CreateProxyHost_MissingDomain(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.CreateProxyHost(context.Background(), connect.NewRequest(&labv1.CreateProxyHostRequest{
		TargetUrl: "http://192.168.1.100:3000",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_CreateProxyHost_MissingTargetURL verifies that an empty
// target_url triggers CodeInvalidArgument.
func TestProxyHandler_CreateProxyHost_MissingTargetURL(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.CreateProxyHost(context.Background(), connect.NewRequest(&labv1.CreateProxyHostRequest{
		Domain: "app.example.com",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_GetProxyHost_MissingID verifies that an empty id triggers
// CodeInvalidArgument.
func TestProxyHandler_GetProxyHost_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.GetProxyHost(context.Background(), connect.NewRequest(&labv1.GetProxyHostRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_UpdateProxyHost_MissingID verifies that an empty id triggers
// CodeInvalidArgument.
func TestProxyHandler_UpdateProxyHost_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.UpdateProxyHost(context.Background(), connect.NewRequest(&labv1.UpdateProxyHostRequest{
		Domain:    "app.example.com",
		TargetUrl: "http://192.168.1.100:3000",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_UpdateProxyHost_MissingDomain verifies that an empty domain
// triggers CodeInvalidArgument.
func TestProxyHandler_UpdateProxyHost_MissingDomain(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.UpdateProxyHost(context.Background(), connect.NewRequest(&labv1.UpdateProxyHostRequest{
		Id:        "some-id",
		TargetUrl: "http://192.168.1.100:3000",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_UpdateProxyHost_MissingTargetURL verifies that an empty
// target_url triggers CodeInvalidArgument.
func TestProxyHandler_UpdateProxyHost_MissingTargetURL(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.UpdateProxyHost(context.Background(), connect.NewRequest(&labv1.UpdateProxyHostRequest{
		Id:     "some-id",
		Domain: "app.example.com",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_DeleteProxyHost_MissingID verifies that an empty id triggers
// CodeInvalidArgument.
func TestProxyHandler_DeleteProxyHost_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.DeleteProxyHost(context.Background(), connect.NewRequest(&labv1.DeleteProxyHostRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_GetProxyStatus_MissingID verifies that an empty id triggers
// CodeInvalidArgument.
func TestProxyHandler_GetProxyStatus_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.GetProxyStatus(context.Background(), connect.NewRequest(&labv1.GetProxyStatusRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_UploadCert_MissingFields verifies that missing required fields
// trigger CodeInvalidArgument.
func TestProxyHandler_UploadCert_MissingFields(t *testing.T) {
	h := &ProxyServiceServer{}

	// Missing all fields
	_, err := h.UploadCert(context.Background(), connect.NewRequest(&labv1.UploadCertRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing cert_pem
	_, err = h.UploadCert(context.Background(), connect.NewRequest(&labv1.UploadCertRequest{
		ProxyHostId: "some-id",
		KeyPem:      "-----BEGIN RSA PRIVATE KEY-----",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing key_pem
	_, err = h.UploadCert(context.Background(), connect.NewRequest(&labv1.UploadCertRequest{
		ProxyHostId: "some-id",
		CertPem:     "-----BEGIN CERTIFICATE-----",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing proxy_host_id
	_, err = h.UploadCert(context.Background(), connect.NewRequest(&labv1.UploadCertRequest{
		CertPem: "-----BEGIN CERTIFICATE-----",
		KeyPem:  "-----BEGIN RSA PRIVATE KEY-----",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// ---- Uptime monitor validation tests ----

// TestProxyHandler_CreateMonitor_MissingName verifies name is required.
func TestProxyHandler_CreateMonitor_MissingName(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.CreateMonitor(context.Background(), connect.NewRequest(&labv1.CreateMonitorRequest{
		Url: "http://example.com",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_CreateMonitor_MissingURL verifies url is required.
func TestProxyHandler_CreateMonitor_MissingURL(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.CreateMonitor(context.Background(), connect.NewRequest(&labv1.CreateMonitorRequest{
		Name: "My Monitor",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_GetMonitor_MissingID verifies id is required.
func TestProxyHandler_GetMonitor_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.GetMonitor(context.Background(), connect.NewRequest(&labv1.GetMonitorRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_UpdateMonitor_MissingFields verifies required fields.
func TestProxyHandler_UpdateMonitor_MissingFields(t *testing.T) {
	h := &ProxyServiceServer{}

	// Missing id
	_, err := h.UpdateMonitor(context.Background(), connect.NewRequest(&labv1.UpdateMonitorRequest{
		Name: "My Monitor",
		Url:  "http://example.com",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing name
	_, err = h.UpdateMonitor(context.Background(), connect.NewRequest(&labv1.UpdateMonitorRequest{
		Id:  "some-id",
		Url: "http://example.com",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)

	// Missing url
	_, err = h.UpdateMonitor(context.Background(), connect.NewRequest(&labv1.UpdateMonitorRequest{
		Id:   "some-id",
		Name: "My Monitor",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_DeleteMonitor_MissingID verifies id is required.
func TestProxyHandler_DeleteMonitor_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.DeleteMonitor(context.Background(), connect.NewRequest(&labv1.DeleteMonitorRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_GetMonitorHistory_MissingID verifies id is required.
func TestProxyHandler_GetMonitorHistory_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.GetMonitorHistory(context.Background(), connect.NewRequest(&labv1.GetMonitorHistoryRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// TestProxyHandler_GetMonitorStats_MissingID verifies id is required.
func TestProxyHandler_GetMonitorStats_MissingID(t *testing.T) {
	h := &ProxyServiceServer{}
	_, err := h.GetMonitorStats(context.Background(), connect.NewRequest(&labv1.GetMonitorStatsRequest{}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}
