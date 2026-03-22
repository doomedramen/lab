package tus

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path/filepath"

	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

// Handler wraps the tusd handler with custom configuration
type Handler struct {
	*tusd.Handler
	uploadDir string
}

// Config holds configuration for the Tus handler
type Config struct {
	// UploadDir is the directory where files are stored
	UploadDir string
	// BasePath is the URL path prefix for Tus endpoints
	BasePath string
	// MaxSize is the maximum upload size in bytes (0 = no limit)
	MaxSize int64
}

// NewHandler creates a new Tus handler
func NewHandler(cfg Config) (*Handler, error) {
	// Set default upload directory
	uploadDir := cfg.UploadDir
	if uploadDir == "" {
		uploadDir = getUploadDir()
	}

	// Ensure upload directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Create file store
	store := filestore.New(uploadDir)

	// Create composer for the store
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	// Create handler config
	handlerConfig := tusd.Config{
		BasePath:      cfg.BasePath,
		StoreComposer: composer,
	}

	// Set max size if specified
	if cfg.MaxSize > 0 {
		handlerConfig.MaxSize = cfg.MaxSize
	}

	// Create the handler
	handler, err := tusd.NewHandler(handlerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create tus handler: %w", err)
	}

	// Set up hooks for post-processing
	setupHooks(handler, uploadDir)

	return &Handler{
		Handler:   handler,
		uploadDir: uploadDir,
	}, nil
}

// getUploadDir returns the default upload directory
func getUploadDir() string {
	// Check for custom directory in environment
	if dir := os.Getenv("ISO_DIR"); dir != "" {
		return dir
	}

	// Use user-specific directory
	currentUser, err := user.Current()
	if err != nil {
		return "./isos"
	}

	return filepath.Join(currentUser.HomeDir, "libvirt-images", "isos")
}

// setupHooks configures hooks for post-upload processing
func setupHooks(handler *tusd.Handler, uploadDir string) {
	// Log completed uploads
	go func() {
		for info := range handler.CompleteUploads {
			log.Printf("Upload completed: %s (%s)", info.Upload.ID, info.Upload.MetaData["filename"])

			// Rename file to original filename
			if filename, ok := info.Upload.MetaData["filename"]; ok {
				oldPath := filepath.Join(uploadDir, info.Upload.ID)
				newPath := filepath.Join(uploadDir, filename)

				// Check if file with same name exists
				if _, err := os.Stat(newPath); err == nil {
					// Delete the old file
					os.Remove(newPath)
				}

				// Rename the file
				if err := os.Rename(oldPath, newPath); err != nil {
					log.Printf("Failed to rename upload: %v", err)
				}
			}
		}
	}()
}

// Middleware returns a middleware function that adds CORS headers for Tus
func (h *Handler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add CORS headers for Tus protocol
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, HEAD, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata")
		w.Header().Set("Access-Control-Expose-Headers", "Upload-Offset, Location, Tus-Resumable, Upload-Metadata, Upload-Expires")

		// Handle preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ServeHTTP implements http.Handler
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(w, r)
}

// RegisterRoutes registers Tus routes on the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux, basePath string) {
	// Wrap with CORS middleware
	handler := h.Middleware(h.Handler)

	// Register the Tus handler
	mux.Handle(basePath, handler)
	mux.Handle(basePath+"/", handler)
}

// GetUploadDir returns the upload directory path
func (h *Handler) GetUploadDir() string {
	return h.uploadDir
}

// WaitForUploads waits for all pending uploads to complete
func (h *Handler) WaitForUploads(ctx context.Context) error {
	// This is a simplified implementation
	// In production, you'd track active uploads and wait for them
	<-ctx.Done()
	return ctx.Err()
}
