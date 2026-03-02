package response

import (
	"encoding/json"
	"errors"
	"net/http"
)

// JSON writes a JSON response
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

// Error writes an error response
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{
		Message: message,
		Code:    status,
	})
}

// ErrorFromError writes an error response from a Go error
func ErrorFromError(w http.ResponseWriter, err error) {
	var status int
	message := err.Error()

	// Map common errors to HTTP status codes
	switch {
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrBadRequest):
		status = http.StatusBadRequest
	case errors.Is(err, ErrConflict):
		status = http.StatusConflict
	default:
		status = http.StatusInternalServerError
	}

	Error(w, status, message)
}

// Common errors
var (
	ErrNotFound   = errors.New("resource not found")
	ErrBadRequest = errors.New("bad request")
	ErrConflict   = errors.New("conflict")
)

// ActionResponse represents a response for action endpoints
type ActionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
