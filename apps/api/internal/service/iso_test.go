package service

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// --- mock ISO repository ---

type mockISORepo struct {
	isos   []*model.ISOImage
	pools  []*model.StoragePool
	getErr error
	delErr error
}

func (m *mockISORepo) GetAll(_ context.Context) ([]*model.ISOImage, error) {
	return m.isos, nil
}

func (m *mockISORepo) GetByID(_ context.Context, id string) (*model.ISOImage, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, iso := range m.isos {
		if iso.ID == id {
			return iso, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockISORepo) Upload(_ context.Context, name string, _ io.Reader, size int64) (*model.ISOImage, error) {
	iso := &model.ISOImage{ID: "iso-new", Name: name, Size: size}
	return iso, nil
}

func (m *mockISORepo) Delete(_ context.Context, id string) error {
	if m.delErr != nil {
		return m.delErr
	}
	for i, iso := range m.isos {
		if iso.ID == id {
			m.isos = append(m.isos[:i], m.isos[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func (m *mockISORepo) GetStoragePools(_ context.Context) ([]*model.StoragePool, error) {
	return m.pools, nil
}

// --- tests ---

func TestISOService_GetAll(t *testing.T) {
	repo := &mockISORepo{
		isos: []*model.ISOImage{
			{ID: "iso-1", Name: "ubuntu.iso"},
			{ID: "iso-2", Name: "debian.iso"},
		},
	}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 10*1024*1024*1024, context.Background())

	isos, err := svc.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(isos) != 2 {
		t.Errorf("expected 2 ISOs, got %d", len(isos))
	}
}

func TestISOService_GetByID_Found(t *testing.T) {
	repo := &mockISORepo{
		isos: []*model.ISOImage{
			{ID: "iso-1", Name: "ubuntu.iso"},
		},
	}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 0, context.Background())

	iso, err := svc.GetByID(context.Background(), "iso-1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if iso.Name != "ubuntu.iso" {
		t.Errorf("Name = %q, want ubuntu.iso", iso.Name)
	}
}

func TestISOService_GetByID_NotFound(t *testing.T) {
	repo := &mockISORepo{}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 0, context.Background())

	_, err := svc.GetByID(context.Background(), "nonexistent")
	if !errors.Is(err, ErrISONotFound) {
		t.Errorf("expected ErrISONotFound, got %v", err)
	}
}

func TestISOService_Delete_Found(t *testing.T) {
	repo := &mockISORepo{
		isos: []*model.ISOImage{
			{ID: "iso-1", Name: "ubuntu.iso"},
		},
	}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 0, context.Background())

	if err := svc.Delete(context.Background(), "iso-1"); err != nil {
		t.Errorf("Delete: %v", err)
	}
}

func TestISOService_Delete_NotFound(t *testing.T) {
	repo := &mockISORepo{}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 0, context.Background())

	err := svc.Delete(context.Background(), "nonexistent")
	if !errors.Is(err, ErrISONotFound) {
		t.Errorf("expected ErrISONotFound, got %v", err)
	}
}

func TestISOService_GetStoragePools(t *testing.T) {
	repo := &mockISORepo{
		pools: []*model.StoragePool{
			{Name: "local", Path: "/var/lib/isos"},
		},
	}
	svc := NewISOService(repo, "/tmp/isos", "/tmp/iso-dl", 0, context.Background())

	pools, err := svc.GetStoragePools(context.Background())
	if err != nil {
		t.Fatalf("GetStoragePools: %v", err)
	}
	if len(pools) != 1 {
		t.Errorf("expected 1 pool, got %d", len(pools))
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m 0s"},
		{90 * time.Second, "1m 30s"},
		{5*time.Minute + 15*time.Second, "5m 15s"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.d)
		if got != tt.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestGetISODownloadProgress_NotFound(t *testing.T) {
	_, ok := GetISODownloadProgress("nonexistent.iso")
	if ok {
		t.Error("expected false for nonexistent download")
	}
}

func TestGetAllISODownloadProgresses_Empty(t *testing.T) {
	result := GetAllISODownloadProgresses()
	// Should not panic and should return a map
	if result == nil {
		t.Error("expected non-nil map")
	}
}
