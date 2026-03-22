package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestRequestIDWithResponseHeader_GeneratesNewID(t *testing.T) {
	handler := RequestIDWithResponseHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, req)

	// Check that a request ID was generated and added to response
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	// Chi's request ID format is: hostname/counter-random (e.g., "host/ABC123-000001")
	// Just verify it's not empty and has some content
	if len(requestID) < 5 {
		t.Errorf("request ID %q is too short", requestID)
	}
}

func TestRequestIDWithResponseHeader_PreservesExistingID(t *testing.T) {
	existingID := "my-custom-request-id"
	handler := RequestIDWithResponseHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	handler.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID != existingID {
		t.Errorf("request ID = %q, want %q", requestID, existingID)
	}
}

func TestRequestIDWithResponseHeader_StoresInContext(t *testing.T) {
	var contextID string
	handler := RequestIDWithResponseHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, req)

	responseID := w.Header().Get("X-Request-ID")
	if contextID != responseID {
		t.Errorf("context ID %q != response ID %q", contextID, responseID)
	}
	if contextID == "" {
		t.Error("expected request ID to be stored in context")
	}
}

func TestGetRequestID(t *testing.T) {
	t.Run("returns ID from context", func(t *testing.T) {
		expectedID := "test-id-123"
		ctx := context.WithValue(context.Background(), middleware.RequestIDKey, expectedID)
		got := GetRequestID(ctx)
		if got != expectedID {
			t.Errorf("GetRequestID() = %q, want %q", got, expectedID)
		}
	})

	t.Run("returns empty string for empty context", func(t *testing.T) {
		got := GetRequestID(context.Background())
		if got != "" {
			t.Errorf("GetRequestID() = %q, want empty string", got)
		}
	})

	t.Run("returns empty string for wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middleware.RequestIDKey, 12345)
		got := GetRequestID(ctx)
		if got != "" {
			t.Errorf("GetRequestID() = %q, want empty string", got)
		}
	})
}

func TestResponseHeaderWriter_WriteHeader(t *testing.T) {
	t.Run("sets request ID header on first WriteHeader call", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseHeaderWriter{
			ResponseWriter: rec,
			wroteHeader:    false,
			requestID:      "test-request-id",
		}

		rw.WriteHeader(http.StatusCreated)

		if rw.wroteHeader != true {
			t.Error("expected wroteHeader to be true")
		}
		if rec.Code != http.StatusCreated {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusCreated)
		}
		if rec.Header().Get("X-Request-ID") != "test-request-id" {
			t.Errorf("X-Request-ID header = %q, want %q", rec.Header().Get("X-Request-ID"), "test-request-id")
		}
	})

	t.Run("only sets header once on multiple WriteHeader calls", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseHeaderWriter{
			ResponseWriter: rec,
			wroteHeader:    false,
			requestID:      "test-request-id",
		}

		rw.WriteHeader(http.StatusCreated)
		rw.WriteHeader(http.StatusBadRequest)

		if rec.Code != http.StatusCreated {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusCreated)
		}
	})
}

func TestResponseHeaderWriter_Write(t *testing.T) {
	t.Run("calls WriteHeader if not already called", func(t *testing.T) {
		rec := httptest.NewRecorder()
		rw := &responseHeaderWriter{
			ResponseWriter: rec,
			wroteHeader:    false,
			requestID:      "test-request-id",
		}

		n, err := rw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("Write() error = %v", err)
		}
		if n != 5 {
			t.Errorf("Write() returned %d, want 5", n)
		}

		if rec.Code != http.StatusOK {
			t.Errorf("status code = %d, want %d", rec.Code, http.StatusOK)
		}
		if rec.Header().Get("X-Request-ID") != "test-request-id" {
			t.Errorf("X-Request-ID header = %q, want %q", rec.Header().Get("X-Request-ID"), "test-request-id")
		}
	})
}

func TestRequestIDWithResponseHeader_VariousStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := RequestIDWithResponseHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			handler.ServeHTTP(w, req)

			if w.Code != tc.statusCode {
				t.Errorf("status code = %d, want %d", w.Code, tc.statusCode)
			}
			if w.Header().Get("X-Request-ID") == "" {
				t.Error("expected X-Request-ID header to be set")
			}
		})
	}
}

func isValidUUID(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			return false
		}
		for _, c := range part {
			if !isHexDigit(c) {
				return false
			}
		}
	}
	return true
}

func isHexDigit(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
