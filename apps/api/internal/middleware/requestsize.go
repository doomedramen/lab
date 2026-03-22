package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// RequestSizeMiddleware limits the size of request bodies.
// Requests exceeding the limit will receive a 413 Request Entity Too Large response.
// If maxBytes is 0 or negative, no limit is applied (pass-through).
func RequestSizeMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If maxBytes is 0 or negative, no limit is applied
			if maxBytes <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Check Content-Length header for early rejection
			if r.ContentLength > maxBytes {
				writePayloadTooLarge(w, maxBytes)
				return
			}

			// Wrap the body with MaxBytesReader to enforce the limit during reading
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

// writePayloadTooLarge writes a JSON error response for oversized requests.
func writePayloadTooLarge(w http.ResponseWriter, maxBytes int64) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusRequestEntityTooLarge)

	errResp := map[string]interface{}{
		"error": fmt.Sprintf("request body too large (max %d bytes)", maxBytes),
		"code":  http.StatusRequestEntityTooLarge,
	}

	// Best effort JSON write - ignore errors as we can't do much about them
	_ = json.NewEncoder(w).Encode(errResp)
}
