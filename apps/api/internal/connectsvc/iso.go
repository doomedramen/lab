package connectsvc

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// IsoServiceServer implements labv1connect.IsoServiceHandler.
type IsoServiceServer struct {
	labv1connect.UnimplementedIsoServiceHandler
	svc *service.ISOService
}

// NewIsoServiceServer creates a new IsoServiceServer.
func NewIsoServiceServer(svc *service.ISOService) *IsoServiceServer {
	return &IsoServiceServer{svc: svc}
}

// ListISOs returns all ISO images.
func (s *IsoServiceServer) ListISOs(
	ctx context.Context,
	_ *connect.Request[labv1.ListISOsRequest],
) (*connect.Response[labv1.ListISOsResponse], error) {
	isos, err := s.svc.GetAll(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	proto := make([]*labv1.ISOImage, len(isos))
	for i, iso := range isos {
		proto[i] = modelISOToProto(iso)
	}
	return connect.NewResponse(&labv1.ListISOsResponse{
		Isos:  proto,
		Total: int32(len(isos)),
	}), nil
}

// GetISO returns a single ISO by ID.
func (s *IsoServiceServer) GetISO(
	ctx context.Context,
	req *connect.Request[labv1.GetISORequest],
) (*connect.Response[labv1.GetISOResponse], error) {
	iso, err := s.svc.GetByID(ctx, req.Msg.Id)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.GetISOResponse{Iso: modelISOToProto(iso)}), nil
}

// DeleteISO deletes an ISO.
func (s *IsoServiceServer) DeleteISO(
	ctx context.Context,
	req *connect.Request[labv1.DeleteISORequest],
) (*connect.Response[labv1.DeleteISOResponse], error) {
	if err := s.svc.Delete(ctx, req.Msg.Id); err != nil {
		return nil, serviceErrToConnect(err)
	}
	return connect.NewResponse(&labv1.DeleteISOResponse{}), nil
}

// ListStoragePools returns available storage pools.
func (s *IsoServiceServer) ListStoragePools(
	ctx context.Context,
	_ *connect.Request[labv1.ListStoragePoolsRequest],
) (*connect.Response[labv1.ListStoragePoolsResponse], error) {
	pools, err := s.svc.GetStoragePools(ctx)
	if err != nil {
		return nil, serviceErrToConnect(err)
	}
	proto := make([]*labv1.StoragePool, len(pools))
	for i, p := range pools {
		proto[i] = modelStoragePoolToProto(p)
	}
	return connect.NewResponse(&labv1.ListStoragePoolsResponse{Pools: proto}), nil
}

// DownloadISO starts an ISO download from a URL in the background.
func (s *IsoServiceServer) DownloadISO(
	ctx context.Context,
	req *connect.Request[labv1.DownloadISORequest],
) (*connect.Response[labv1.DownloadISOResponse], error) {
	filename := req.Msg.Filename
	if filename == "" {
		// Extract filename from the URL path.
		parts := strings.Split(req.Msg.Url, "/")
		filename = parts[len(parts)-1]
		if filename == "" {
			filename = "downloaded.iso"
		}
	}

	// Sanitize: strip any directory components to prevent path traversal.
	filename = filepath.Base(filename)
	if filename == "" || filename == "." {
		filename = "downloaded.iso"
	}
	// Ensure .iso extension.
	if !strings.HasSuffix(strings.ToLower(filename), ".iso") {
		filename += ".iso"
	}

	// Run download in the background using server-lifetime context.
	// The download continues after client disconnect but cancels on server shutdown.
	go func() {
		_, err := s.svc.DownloadISO(s.svc.ShutdownContext(), req.Msg.Url, filename)
		if err != nil {
			slog.Error("ISO download failed", "filename", filename, "url", req.Msg.Url, "err", err)
		} else {
			slog.Info("ISO download completed", "filename", filename)
		}
	}()

	return connect.NewResponse(&labv1.DownloadISOResponse{
		Id:       filename,
		Filename: filename,
		Status:   "downloading",
	}), nil
}

// ListISODownloadProgress returns all active (and recently finished) ISO download progress entries.
func (s *IsoServiceServer) ListISODownloadProgress(
	_ context.Context,
	_ *connect.Request[labv1.ListISODownloadProgressRequest],
) (*connect.Response[labv1.ListISODownloadProgressResponse], error) {
	all := service.GetAllISODownloadProgresses()
	downloads := make([]*labv1.GetISODownloadProgressResponse, 0, len(all))
	for _, p := range all {
		downloads = append(downloads, &labv1.GetISODownloadProgressResponse{
			Url:           p.URL,
			Filename:      p.Filename,
			Downloaded:    p.Downloaded,
			Total:         p.Total,
			Percent:       p.Percent,
			Status:        p.Status,
			Error:         p.Error,
			StartTime:     p.StartTime.Format(time.RFC3339),
			EstimatedTime: p.EstimatedTime,
		})
	}
	return connect.NewResponse(&labv1.ListISODownloadProgressResponse{Downloads: downloads}), nil
}

// GetISODownloadProgress returns the progress of an ISO download.
func (s *IsoServiceServer) GetISODownloadProgress(
	_ context.Context,
	req *connect.Request[labv1.GetISODownloadProgressRequest],
) (*connect.Response[labv1.GetISODownloadProgressResponse], error) {
	progress, ok := service.GetISODownloadProgress(req.Msg.Filename)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("download not found"))
	}

	return connect.NewResponse(&labv1.GetISODownloadProgressResponse{
		Url:           progress.URL,
		Filename:      progress.Filename,
		Downloaded:    progress.Downloaded,
		Total:         progress.Total,
		Percent:       progress.Percent,
		Status:        progress.Status,
		Error:         progress.Error,
		StartTime:     progress.StartTime.Format(time.RFC3339),
		EstimatedTime: progress.EstimatedTime,
	}), nil
}

// --- conversion helpers ---

func modelISOToProto(iso *model.ISOImage) *labv1.ISOImage {
	return &labv1.ISOImage{
		Id:          iso.ID,
		Name:        iso.Name,
		Size:        iso.Size,
		Path:        iso.Path,
		Description: iso.Description,
		Os:          iso.OS,
		Status:      iso.Status,
		CreatedAt:   iso.CreatedAt,
	}
}

func modelStoragePoolToProto(p *model.StoragePool) *labv1.StoragePool {
	return &labv1.StoragePool{
		Id:             p.ID,
		Name:           p.Name,
		Type:           modelStorageTypeToProto(p.Type),
		Path:           p.Path,
		CapacityBytes:  p.CapacityBytes,
		AvailableBytes: p.AvailableBytes,
		UsedBytes:      p.UsedBytes,
		Status:         modelStorageStatusToProto(p.Status),
	}
}

func modelStorageTypeToProto(t model.StorageType) labv1.StorageType {
	switch t {
	case model.StorageTypeDir:
		return labv1.StorageType_STORAGE_TYPE_DIR
	default:
		return labv1.StorageType_STORAGE_TYPE_DIR
	}
}

func modelStorageStatusToProto(s model.StorageStatus) labv1.StorageStatus {
	switch s {
	case model.StorageStatusActive:
		return labv1.StorageStatus_STORAGE_STATUS_ACTIVE
	default:
		return labv1.StorageStatus_STORAGE_STATUS_ACTIVE
	}
}
