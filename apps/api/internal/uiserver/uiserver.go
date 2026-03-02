package uiserver

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Config holds UI server configuration
type Config struct {
	// Enabled determines if UI server should be started
	Enabled bool
	// Port is the port the UI server listens on
	Port int
	// Dir is the path to the UI server files (standalone Next.js build)
	Dir string
	// NodePath is the path to the Node.js binary
	NodePath string
}

// Server manages the Next.js UI server
type Server struct {
	config     Config
	cmd        *exec.Cmd
	proxy      *httputil.ReverseProxy
	cancelFunc context.CancelFunc
}

// New creates a new UI server
func New(cfg Config) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the UI server and waits for it to be ready
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}

	// Check if UI directory exists
	if s.config.Dir == "" {
		// Try default locations
		s.config.Dir = findUIDirectory()
	}

	if s.config.Dir == "" {
		return fmt.Errorf("UI directory not found")
	}

	// Check if server.js exists
	serverJS := filepath.Join(s.config.Dir, "server.js")
	if _, err := os.Stat(serverJS); os.IsNotExist(err) {
		return fmt.Errorf("UI server.js not found at %s", serverJS)
	}

	// Find Node.js binary
	if s.config.NodePath == "" {
		s.config.NodePath = findNodeBinary()
	}

	if s.config.NodePath == "" {
		return fmt.Errorf("Node.js binary not found")
	}

	// Create context for managing server lifecycle
	ctx, s.cancelFunc = context.WithCancel(ctx)

	// Start Next.js server
	s.cmd = exec.CommandContext(ctx, s.config.NodePath, serverJS)
	s.cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", s.config.Port),
		"NODE_ENV=production",
	)
	s.cmd.Dir = s.config.Dir

	// Start the command
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start UI server: %w", err)
	}

	// Wait for server to be ready
	if err := s.waitForReady(ctx, 30*time.Second); err != nil {
		return fmt.Errorf("UI server failed to start: %w", err)
	}

	// Setup reverse proxy
	targetURL := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("localhost:%d", s.config.Port),
	}
	s.proxy = httputil.NewSingleHostReverseProxy(targetURL)

	return nil
}

// waitForReady waits for the UI server to be ready
func (s *Server) waitForReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			resp, err := client.Get(fmt.Sprintf("http://localhost:%d/health", s.config.Port))
			if err == nil {
				resp.Body.Close()
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}

	return fmt.Errorf("timeout waiting for UI server")
}

// Handler returns an http.Handler that proxies requests to the UI server
func (s *Server) Handler() http.Handler {
	if s.proxy == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "UI server not available", http.StatusServiceUnavailable)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.proxy.ServeHTTP(w, r)
	})
}

// Stop stops the UI server
func (s *Server) Stop(ctx context.Context) error {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}

	if s.cmd != nil {
		// Give the process time to shutdown gracefully
		done := make(chan error, 1)
		go func() {
			done <- s.cmd.Wait()
		}()

		select {
		case <-ctx.Done():
			// Force kill if context is cancelled
			return s.cmd.Process.Kill()
		case err := <-done:
			return err
		}
	}

	return nil
}

// findUIDirectory looks for the UI server in common locations
func findUIDirectory() string {
	locations := []string{
		"./ui",
		"./apps/web/.next/standalone",
		"/app/ui",
		"/usr/share/lab/ui",
		"/opt/lab/ui",
	}

	for _, loc := range locations {
		serverJS := filepath.Join(loc, "server.js")
		if _, err := os.Stat(serverJS); err == nil {
			return loc
		}
	}

	return ""
}

// findNodeBinary looks for the Node.js binary
func findNodeBinary() string {
	locations := []string{
		"node",
		"/usr/bin/node",
		"/usr/local/bin/node",
		"./node",
	}

	for _, loc := range locations {
		if path, err := exec.LookPath(loc); err == nil {
			return path
		}
	}

	return ""
}

// EmbeddedFS serves UI files from an embedded filesystem
// This is used when the UI is compiled into the binary
type EmbeddedFS struct {
	fs fs.FS
}

// NewEmbeddedFS creates a new embedded filesystem handler
func NewEmbeddedFS(filesystem fs.FS) *EmbeddedFS {
	return &EmbeddedFS{
		fs: filesystem,
	}
}

// ServeHTTP serves files from the embedded filesystem
func (e *EmbeddedFS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Remove leading slash
	path := r.URL.Path
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Try to open the file
	f, err := e.fs.Open(path)
	if err != nil {
		// If file not found, try index.html for SPA routing
		f, err = e.fs.Open("index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
	}
	defer f.Close()

	// Get file info
	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	// If it's a directory, try index.html
	if stat.IsDir() {
		f, err = e.fs.Open(path + "/index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		defer f.Close()
		stat, _ = f.Stat()
	}

	// Serve the file
	http.ServeContent(w, r, path, stat.ModTime(), f.(io.ReadSeeker))
}
