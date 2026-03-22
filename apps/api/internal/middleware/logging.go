package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// Logging provides request logging middleware
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response wrapper to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Get request ID from context for correlation
		requestID := middleware.GetReqID(r.Context())

		// Include request ID in log if present
		if requestID != "" {
			log.Printf("[%s] %s %s %d %v",
				requestID,
				r.Method,
				r.URL.Path,
				rw.statusCode,
				time.Since(start),
			)
		} else {
			log.Printf("%s %s %d %v",
				r.Method,
				r.URL.Path,
				rw.statusCode,
				time.Since(start),
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
