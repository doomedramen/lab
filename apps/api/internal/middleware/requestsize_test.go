package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestSizeMiddleware_Allowed(t *testing.T) {
	maxBytes := int64(100)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// Create a request with body smaller than limit
	body := strings.NewReader("hello world")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(body.Len())
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", rec.Body.String())
	}
}

func TestRequestSizeMiddleware_ContentLengthExceeded(t *testing.T) {
	maxBytes := int64(10)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// Create a request with Content-Length exceeding limit
	body := strings.NewReader("this body is too long")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(body.Len()) // 21 bytes
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rec.Code)
	}
}

func TestRequestSizeMiddleware_ZeroMaxBytes(t *testing.T) {
	maxBytes := int64(0)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// With zero maxBytes, the middleware should still pass through
	body := strings.NewReader("hello")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	// With MaxBytesReader and limit 0, all reads should fail
	// but the handler should still be called
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequestSizeMiddleware_LargeBodyStream(t *testing.T) {
	maxBytes := int64(10)
	var readErr error
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// Create a request with a body larger than limit (streaming case)
	largeBody := bytes.NewReader(make([]byte, 100))
	req := httptest.NewRequest(http.MethodPost, "/test", largeBody)
	req.ContentLength = -1 // Unknown content length (streaming)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	// The handler should be called, but reading the body should fail
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if readErr == nil {
		t.Error("expected error reading body beyond limit, got nil")
	}
}

func TestRequestSizeMiddleware_NoBody(t *testing.T) {
	maxBytes := int64(100)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// GET request with no body
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRequestSizeMiddleware_ExactLimit(t *testing.T) {
	maxBytes := int64(11) // Exactly "hello world" length
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// Create a request with body exactly at limit
	body := strings.NewReader("hello world")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(body.Len())
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "hello world" {
		t.Errorf("expected body 'hello world', got '%s'", rec.Body.String())
	}
}

func TestRequestSizeMiddleware_JSONErrorResponse(t *testing.T) {
	maxBytes := int64(10)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestSizeMiddleware(maxBytes)(handler)

	// Create a request with Content-Length exceeding limit
	body := strings.NewReader("this body is too long for the limit")
	req := httptest.NewRequest(http.MethodPost, "/test", body)
	req.ContentLength = int64(body.Len())
	rec := httptest.NewRecorder()

	middleware.ServeHTTP(rec, req)

	// Verify Content-Type is JSON
	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Verify body contains JSON error
	if !strings.Contains(rec.Body.String(), "request body too large") {
		t.Errorf("expected error message in body, got '%s'", rec.Body.String())
	}
}
