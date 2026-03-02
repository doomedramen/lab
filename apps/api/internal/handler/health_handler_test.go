package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_ServeHTTP_NilDeps(t *testing.T) {
	// Both deps nil → both checks report "disabled", overall "degraded" → 503
	h := NewHealthHandler(nil, nil, "test-version")
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rr.Code)
	}

	var body HealthStatus
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Status != "degraded" {
		t.Errorf("overall status = %q, want degraded", body.Status)
	}

	if body.Version != "test-version" {
		t.Errorf("version = %q, want test-version", body.Version)
	}

	libvirtCheck, ok := body.Checks["libvirt"]
	if !ok {
		t.Fatal("expected libvirt check in response")
	}
	if libvirtCheck.Status != "disabled" {
		t.Errorf("libvirt check status = %q, want disabled", libvirtCheck.Status)
	}

	sqliteCheck, ok := body.Checks["sqlite"]
	if !ok {
		t.Fatal("expected sqlite check in response")
	}
	if sqliteCheck.Status != "disabled" {
		t.Errorf("sqlite check status = %q, want disabled", sqliteCheck.Status)
	}
}

func TestHealthHandler_checkLibvirt_NilClient(t *testing.T) {
	h := &HealthHandler{libvirtClient: nil, version: "test"}
	result := h.checkLibvirt()
	if result.Status != "disabled" {
		t.Errorf("status = %q, want disabled", result.Status)
	}
}

func TestHealthHandler_checkSQLite_NilDB(t *testing.T) {
	h := &HealthHandler{db: nil, version: "test"}
	result := h.checkSQLite()
	if result.Status != "disabled" {
		t.Errorf("status = %q, want disabled", result.Status)
	}
}
