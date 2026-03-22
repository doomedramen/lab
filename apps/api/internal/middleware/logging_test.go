package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestLogging_CallsNext(t *testing.T) {
	nextCalled := false
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(w, req)

	if !nextCalled {
		t.Error("expected next handler to be called")
	}
}

func TestLogging_PreservesStatusCode(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestLogging_DefaultStatusOK(t *testing.T) {
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't explicitly write header — should default to 200
		w.Write([]byte("ok"))
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("statusCode = %d, want 201", rw.statusCode)
	}
	if w.Code != http.StatusCreated {
		t.Errorf("underlying writer code = %d, want 201", w.Code)
	}
}

func TestLogging_WithRequestID(t *testing.T) {
	testID := "test-request-id-12345"
	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request ID is accessible in the handler
		gotID := middleware.GetReqID(r.Context())
		if gotID != testID {
			t.Errorf("handler got request ID = %q, want %q", gotID, testID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// Add request ID to context
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, testID)
	req = req.WithContext(ctx)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
