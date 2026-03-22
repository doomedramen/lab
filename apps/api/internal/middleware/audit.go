// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/repository/auth"
	"log/slog"
)

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	// Paths to audit (empty = all paths)
	Paths []string
	
	// Paths to exclude from auditing
	ExcludePaths []string
	
	// Log only failed requests (default: log all)
	FailedOnly bool
}

// DefaultAuditConfig returns sensible default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Paths: []string{
			"/lab.v1.AuthService/",      // All auth operations
			"/lab.v1.UserService/",      // All user management
			"/lab.v1.VMService/",        // All VM operations
			"/lab.v1.BackupService/",    // All backup operations
			"/lab.v1.SnapshotService/",  // All snapshot operations
		},
		ExcludePaths: []string{
			"/health",              // Basic health check (too frequent)
			"/metrics",             // Metrics endpoint (too frequent)
		},
		FailedOnly: false,
	}
}

// Audit middleware logs important actions for security compliance
func Audit(repo *auth.AuditLogRepository, config AuditConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if this path should be audited
			if !shouldAudit(r.URL.Path, config) {
				next.ServeHTTP(w, r)
				return
			}
			
			// Capture start time
			start := time.Now()
			
			// Wrap response writer to capture status code
			rw := &auditResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			// Get user from context (if authenticated)
			user := GetUserFromContext(r.Context())
			
			// Get request ID for correlation
			requestID := GetRequestID(r.Context())
			
			// Call next handler
			next.ServeHTTP(rw, r)
			
			// Determine if action was successful
			success := rw.statusCode < 400
			
			// Skip if configured to log only failures
			if config.FailedOnly && success {
				return
			}
			
			// Create audit log entry
			auditLog := &auth.AuditLog{
				UserID:       "",
				Action:       string(mapPathToAction(r.URL.Path, r.Method)),
				ResourceType: extractResourceType(r.URL.Path),
				ResourceID:   extractResource(r.URL.Path),
				Details: map[string]any{
					"method":      r.Method,
					"path":        r.URL.Path,
					"status":      rw.statusCode,
					"duration_ms": time.Since(start).Milliseconds(),
					"request_id":  requestID,
				},
				IPAddress: GetClientIPFromContext(r.Context()),
				UserAgent: r.UserAgent(),
				Status:    auth.StatusSuccess,
				CreatedAt: start,
			}
			
			// Set failure status if applicable
			if !success {
				auditLog.Status = auth.StatusFailure
				auditLog.Details["error"] = http.StatusText(rw.statusCode)
			}
			
			// Add user info if authenticated
			if user != nil {
				auditLog.UserID = user.ID
			}
			
			// Save audit log asynchronously (don't block request)
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				
				if err := repo.Create(ctx, auditLog); err != nil {
					slog.Error("Failed to create audit log", "error", err)
				}
			}()
		})
	}
}

// auditResponseWriter wraps http.ResponseWriter to capture status code
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *auditResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// shouldAudit checks if a path should be audited based on config
func shouldAudit(path string, config AuditConfig) bool {
	// Check exclusions first
	for _, exclude := range config.ExcludePaths {
		if len(path) >= len(exclude) && path[:len(exclude)] == exclude {
			return false
		}
	}
	
	// If no include paths specified, audit everything not excluded
	if len(config.Paths) == 0 {
		return true
	}
	
	// Check if path matches any include path
	for _, include := range config.Paths {
		if len(path) >= len(include) && path[:len(include)] == include {
			return true
		}
	}
	
	return false
}

// mapPathToAction maps HTTP path to audit action
func mapPathToAction(path, method string) string {
	// Auth endpoints
	if contains(path, "/lab.v1.AuthService/") {
		if contains(path, "Login") {
			return "auth.login"
		}
		if contains(path, "Register") {
			return "user.create"
		}
		if contains(path, "MFA") {
			if contains(path, "Enable") {
				return "auth.mfa_enable"
			}
			return "auth.mfa_disable"
		}
	}
	
	// VM endpoints
	if contains(path, "/lab.v1.VMService/") {
		// Check specific actions first
		if contains(path, "Start") {
			return "vm.start"
		}
		if contains(path, "Stop") {
			return "vm.stop"
		}
		if contains(path, "Clone") {
			return "vm.clone"
		}
		if method == "POST" {
			return "vm.create"
		}
		if method == "DELETE" {
			return "vm.delete"
		}
		return "vm.update"
	}
	
	// Backup endpoints
	if contains(path, "/lab.v1.BackupService/") {
		if method == "POST" {
			return "backup.create"
		}
		if method == "DELETE" {
			return "backup.delete"
		}
	}
	
	// Snapshot endpoints
	if contains(path, "/lab.v1.SnapshotService/") {
		if method == "POST" {
			return "snapshot.create"
		}
		if method == "DELETE" {
			return "snapshot.delete"
		}
	}
	
	// Default: generic action
	return "http." + method
}

// extractResource extracts resource identifier from path
func extractResource(path string) string {
	parts := splitPath(path)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// extractResourceType extracts resource type from path
func extractResourceType(path string) string {
	if contains(path, "VM") {
		return "vm"
	}
	if contains(path, "User") {
		return "user"
	}
	if contains(path, "Backup") {
		return "backup"
	}
	if contains(path, "Snapshot") {
		return "snapshot"
	}
	if contains(path, "Auth") {
		return "auth"
	}
	return "api"
}

// Helper functions
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func splitPath(path string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				parts = append(parts, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		parts = append(parts, path[start:])
	}
	return parts
}
