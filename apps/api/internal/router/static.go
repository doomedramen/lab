package router

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// staticFileHandler creates a handler for static files with SPA fallback.
// It serves files from the embedded filesystem, and falls back to index.html
// for unknown paths (enables client-side routing in SPAs).
func staticFileHandler(fsys fs.FS) http.HandlerFunc {
	// Cache of known files (computed once at startup)
	knownFiles := make(map[string]bool)

	// Pre-scan for existing files
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			knownFiles["/"+path] = true
		}
		return nil
	})

	// Create file server
	fileServer := http.FileServer(http.FS(fsys))

	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Check if it's a WebSocket or API path - let it pass through
		if strings.HasPrefix(path, "/ws/") ||
			strings.HasPrefix(path, "/tus/") ||
			strings.HasPrefix(path, "/lab.v1") ||
			strings.HasPrefix(path, "/api/") {
			// Not a static file request
			http.NotFound(w, r)
			return
		}

		// Check if file exists in embedded FS
		if knownFiles[path] {
			// File exists, serve it
			fileServer.ServeHTTP(w, r)
			return
		}

		// File doesn't exist - check for index.html in that directory
		indexpath := filepath.Join(path, "index.html")
		if knownFiles[indexpath] {
			// Serve index.html for SPA routing
			r.URL = &url.URL{Path: indexpath}
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback to root index.html
		r.URL = &url.URL{Path: "/index.html"}
		fileServer.ServeHTTP(w, r)
	}
}

//go:embed web/*
var webFS embed.FS

// WebFS returns the embedded web filesystem for external use
func WebFS() fs.FS {
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		panic("failed to create web sub-filesystem: " + err.Error())
	}
	return subFS
}
