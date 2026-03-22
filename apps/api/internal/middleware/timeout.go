// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"context"
	"net/http"
	"time"
)

// TimeoutConfig holds timeout configuration
type TimeoutConfig struct {
	// Default timeout for all requests
	Default time.Duration
	
	// Timeout for specific paths (overrides default)
	PathTimeouts map[string]time.Duration
}

// DefaultTimeoutConfig returns sensible default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Default: 60 * time.Second,
		PathTimeouts: map[string]time.Duration{
			// Longer timeouts for operations that may take time
			"/lab.v1.BackupService/":   5 * time.Minute,
			"/lab.v1.SnapshotService/": 3 * time.Minute,
			"/lab.v1.ISOService/DownloadISO": 30 * time.Minute,
			// Shorter timeouts for simple operations
			"/lab.v1.HealthService/": 10 * time.Second,
		},
	}
}

// Timeout middleware sets a deadline on the request context.
// If the handler doesn't complete before the deadline, the context is cancelled.
//
// Usage:
//   r.Use(Timeout(middleware.DefaultTimeoutConfig()))
func Timeout(config TimeoutConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine timeout for this path
			timeout := config.Default
			for pathPrefix, pathTimeout := range config.PathTimeouts {
				if len(r.URL.Path) >= len(pathPrefix) && r.URL.Path[:len(pathPrefix)] == pathPrefix {
					timeout = pathTimeout
					break
				}
			}
			
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			// Wrap response writer to catch writes after timeout
			rw := &timeoutResponseWriter{ResponseWriter: w, timedOut: false}
			
			// Handle panic from chi middleware
			defer func() {
				if err := recover(); err != nil {
					panic(err)
				}
			}()
			
			// Call next handler with new context
			next.ServeHTTP(rw, r.WithContext(ctx))
			
			// Check if we timed out
			if rw.timedOut {
				// Handler took too long, log it
				rw.ResponseWriter.WriteHeader(http.StatusGatewayTimeout)
			}
		})
	}
}

// timeoutResponseWriter wraps http.ResponseWriter to track timeout state
type timeoutResponseWriter struct {
	http.ResponseWriter
	timedOut bool
	wroteHeader bool
}

func (rw *timeoutResponseWriter) WriteHeader(code int) {
	if rw.timedOut || rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *timeoutResponseWriter) Write(b []byte) (int, error) {
	if rw.timedOut || rw.wroteHeader {
		return 0, context.DeadlineExceeded
	}
	rw.wroteHeader = true
	return rw.ResponseWriter.Write(b)
}

// RequestID middleware adds a unique ID to each request for tracing.
// This complements the timeout middleware by providing correlation IDs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request already has an ID (from upstream proxy)
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID
			requestID = time.Now().Format("20060102150405") + "-" + generateID(8)
		}
		
		// Add to response header
		w.Header().Set("X-Request-ID", requestID)
		
		// Add to context for logging
		ctx := context.WithValue(r.Context(), "requestID", requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateID generates a random alphanumeric string of length n
func generateID(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond) // Ensure different values
	}
	return string(b)
}
