package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
)

var (
	ErrISONotFound = errors.New("ISO not found")
	ErrISOInUse    = errors.New("ISO is in use by a VM")
	ErrISOTooLarge = errors.New("ISO exceeds maximum allowed size")
)

// ISODownloadProgress tracks the progress of an ISO download.
type ISODownloadProgress struct {
	URL           string    `json:"url"`
	Filename      string    `json:"filename"`
	Downloaded    int64     `json:"downloaded"`
	Total         int64     `json:"total"`
	Percent       float64   `json:"percent"`
	Status        string    `json:"status"` // "downloading", "complete", "error"
	Error         string    `json:"error,omitempty"`
	StartTime     time.Time `json:"start_time"`
	EstimatedTime string    `json:"estimated_time,omitempty"`
}

// isoDownloadProgresses stores active ISO download progress.
// isoProgressMu guards both the map and every ISODownloadProgress struct stored in it.
var (
	isoDownloadProgresses = make(map[string]*ISODownloadProgress)
	isoProgressMu         sync.RWMutex
)

// GetISODownloadProgress returns a value copy of the progress for an ISO download.
// Callers receive an independent snapshot and may read all fields without holding any lock.
func GetISODownloadProgress(filename string) (ISODownloadProgress, bool) {
	isoProgressMu.RLock()
	defer isoProgressMu.RUnlock()
	prog, ok := isoDownloadProgresses[filename]
	if !ok {
		return ISODownloadProgress{}, false
	}
	return *prog, true
}

// GetAllISODownloadProgresses returns value copies of all active ISO download progresses.
func GetAllISODownloadProgresses() map[string]ISODownloadProgress {
	isoProgressMu.RLock()
	defer isoProgressMu.RUnlock()
	result := make(map[string]ISODownloadProgress, len(isoDownloadProgresses))
	for k, v := range isoDownloadProgresses {
		result[k] = *v
	}
	return result
}

// cleanupOldDownloads removes completed/errored downloads older than 5 minutes.
func cleanupOldDownloads() {
	isoProgressMu.Lock()
	defer isoProgressMu.Unlock()
	now := time.Now()
	for filename, prog := range isoDownloadProgresses {
		if prog.Status == "complete" || prog.Status == "error" {
			if now.Sub(prog.StartTime) > 5*time.Minute {
				delete(isoDownloadProgresses, filename)
			}
		}
	}
}

// progressWriter wraps an os.File to track write progress.
// All mutations to the progress struct are performed under isoProgressMu.Lock() to
// eliminate the data race with concurrent readers in GetISODownloadProgress.
type progressWriter struct {
	file     *os.File
	progress *ISODownloadProgress
	written  int64
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.file.Write(p)
	pw.written += int64(n)

	isoProgressMu.Lock()
	pw.progress.Downloaded = pw.written
	if pw.progress.Total > 0 {
		pw.progress.Percent = float64(pw.written) / float64(pw.progress.Total) * 100
	}
	if pw.written > 0 && pw.progress.Total > 0 {
		elapsed := time.Since(pw.progress.StartTime)
		bytesPerSec := float64(pw.written) / elapsed.Seconds()
		if bytesPerSec > 0 {
			remaining := pw.progress.Total - pw.written
			remainingSec := float64(remaining) / bytesPerSec
			pw.progress.EstimatedTime = formatDuration(time.Duration(remainingSec) * time.Second)
		}
	}
	isoProgressMu.Unlock()

	return n, err
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
}

// ISOService provides business logic for ISO operations.
type ISOService struct {
	repo               repository.ISORepository
	isoDir             string
	isoDownloadTempDir string
	maxISOSize         int64
	shutdownCtx        context.Context
}

// NewISOService creates a new ISO service.
func NewISOService(repo repository.ISORepository, isoDir string, isoDownloadTempDir string, maxISOSize int64, shutdownCtx context.Context) *ISOService {
	return &ISOService{
		repo:               repo,
		isoDir:             isoDir,
		isoDownloadTempDir: isoDownloadTempDir,
		maxISOSize:         maxISOSize,
		shutdownCtx:        shutdownCtx,
	}
}

// ShutdownContext returns the context that is cancelled when the server shuts down.
func (s *ISOService) ShutdownContext() context.Context {
	return s.shutdownCtx
}

// GetAll returns all ISO images.
func (s *ISOService) GetAll(ctx context.Context) ([]*model.ISOImage, error) {
	return s.repo.GetAll(ctx)
}

// GetByID returns an ISO image by ID.
func (s *ISOService) GetByID(ctx context.Context, id string) (*model.ISOImage, error) {
	iso, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, ErrISONotFound
		}
		return nil, fmt.Errorf("failed to get ISO: %w", err)
	}
	return iso, nil
}

// Upload handles ISO upload (delegated to Tus handler for actual upload).
func (s *ISOService) Upload(ctx context.Context, name string, reader io.Reader, size int64) (*model.ISOImage, error) {
	return s.repo.Upload(ctx, name, reader, size)
}

// Delete removes an ISO.
func (s *ISOService) Delete(ctx context.Context, id string) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ErrISONotFound
		}
		return fmt.Errorf("failed to delete ISO: %w", err)
	}
	return nil
}

// GetStoragePools returns available storage pools.
func (s *ISOService) GetStoragePools(ctx context.Context) ([]*model.StoragePool, error) {
	return s.repo.GetStoragePools(ctx)
}

// setProgressError records an error state on a progress struct under the global lock.
func setProgressError(p *ISODownloadProgress, msg string) {
	isoProgressMu.Lock()
	p.Status = "error"
	p.Error = msg
	isoProgressMu.Unlock()
}

// DownloadISO downloads an ISO from a URL to the ISO storage directory with progress tracking.
func (s *ISOService) DownloadISO(ctx context.Context, url string, filename string) (string, error) {
	// Register progress entry.
	progress := &ISODownloadProgress{
		URL:       url,
		Filename:  filename,
		Status:    "downloading",
		StartTime: time.Now(),
	}
	isoProgressMu.Lock()
	isoDownloadProgresses[filename] = progress
	isoProgressMu.Unlock()

	defer cleanupOldDownloads()

	// Determine final ISO directory.
	isoDir := s.isoDir
	if isoDir == "" {
		pools, err := s.repo.GetStoragePools(ctx)
		if err != nil || len(pools) == 0 {
			setProgressError(progress, "no storage pool available")
			return "", errors.New("no storage pool available for ISO")
		}
		isoDir = pools[0].Path
	}

	if err := os.MkdirAll(isoDir, 0755); err != nil {
		setProgressError(progress, fmt.Sprintf("failed to create directory: %v", err))
		return "", fmt.Errorf("failed to create ISO directory: %w", err)
	}

	isoPath := filepath.Join(isoDir, filename)

	// Short-circuit if already present.
	if _, err := os.Stat(isoPath); err == nil {
		isoProgressMu.Lock()
		progress.Status = "complete"
		progress.Percent = 100
		progress.Downloaded = progress.Total
		isoProgressMu.Unlock()
		return isoPath, nil
	}

	// Prepare temp download location.
	tempDir := s.isoDownloadTempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		setProgressError(progress, fmt.Sprintf("failed to create temp directory: %v", err))
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	tmpPath := filepath.Join(tempDir, filename+".tmp")

	// Issue HTTP request.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		setProgressError(progress, fmt.Sprintf("failed to create request: %v", err))
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		setProgressError(progress, fmt.Sprintf("failed to download: %v", err))
		return "", fmt.Errorf("failed to download ISO: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		setProgressError(progress, fmt.Sprintf("HTTP %d", resp.StatusCode))
		return "", fmt.Errorf("failed to download ISO: HTTP %d", resp.StatusCode)
	}

	// Enforce maximum ISO size from Content-Length header.
	if s.maxISOSize > 0 && resp.ContentLength > s.maxISOSize {
		setProgressError(progress, fmt.Sprintf("ISO size %d bytes exceeds maximum %d bytes", resp.ContentLength, s.maxISOSize))
		return "", fmt.Errorf("ISO size %d bytes exceeds maximum allowed size of %d bytes", resp.ContentLength, s.maxISOSize)
	}

	isoProgressMu.Lock()
	progress.Total = resp.ContentLength
	isoProgressMu.Unlock()

	// Create temp file.
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		setProgressError(progress, fmt.Sprintf("failed to create temp file: %v", err))
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	writer := &progressWriter{
		file:     tmpFile,
		progress: progress,
	}

	// Wrap body with a LimitedReader as defence-in-depth for chunked transfers
	// that have no Content-Length header.
	var body io.Reader = resp.Body
	if s.maxISOSize > 0 {
		body = io.LimitReader(resp.Body, s.maxISOSize+1)
	}

	written, err := io.Copy(writer, body)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		setProgressError(progress, fmt.Sprintf("failed to write: %v", err))
		return "", fmt.Errorf("failed to write ISO: %w", err)
	}

	// Detect size overflow via the LimitedReader sentinel (+1 byte read means limit exceeded).
	if s.maxISOSize > 0 && written > s.maxISOSize {
		os.Remove(tmpPath)
		setProgressError(progress, fmt.Sprintf("ISO exceeds maximum allowed size of %d bytes", s.maxISOSize))
		return "", ErrISOTooLarge
	}

	// Move temp file to final destination.
	if err := os.Rename(tmpPath, isoPath); err != nil {
		// Rename fails across filesystems — fall back to copy.
		if err := copyFile(tmpPath, isoPath); err != nil {
			os.Remove(tmpPath)
			setProgressError(progress, fmt.Sprintf("failed to finalize: %v", err))
			return "", fmt.Errorf("failed to finalize ISO: %w", err)
		}
		os.Remove(tmpPath)
	}

	isoProgressMu.Lock()
	progress.Status = "complete"
	progress.Percent = 100
	progress.Downloaded = written
	isoProgressMu.Unlock()

	return isoPath, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
