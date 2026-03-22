package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// GetRequestID retrieves request ID from context.
// Returns empty string if not present.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(middleware.RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// responseHeaderWriter wraps http.ResponseWriter to capture when headers are written
// so we can add the request ID to the response.
type responseHeaderWriter struct {
	http.ResponseWriter
	wroteHeader bool
	requestID   string
}

func (rw *responseHeaderWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		// Add request ID to response headers before writing
		if rw.requestID != "" {
			rw.ResponseWriter.Header().Set("X-Request-ID", rw.requestID)
		}
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseHeaderWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// RequestIDWithResponseHeader creates middleware that:
// 1. Generates or preserves request ID using Chi's RequestID
// 2. Adds the request ID to response headers for client correlation
func RequestIDWithResponseHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer to inject the request ID header
		rw := &responseHeaderWriter{
			ResponseWriter: w,
			wroteHeader:    false,
			requestID:      "",
		}

		// Create a handler that will be wrapped by Chi's RequestID
		innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the request ID from context (set by Chi's RequestID)
			rw.requestID = middleware.GetReqID(r.Context())
			// Call the actual next handler with our wrapped response writer
			next.ServeHTTP(rw, r)
		})

		// Apply Chi's RequestID middleware
		middleware.RequestID(innerHandler).ServeHTTP(w, r)
	})
}
