package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTimeout_Default(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that context has deadline
		_, hasDeadline := r.Context().Deadline()
		if !hasDeadline {
			t.Error("Expected context to have deadline")
		}
		w.WriteHeader(http.StatusOK)
	})

	timeoutMiddleware := Timeout(DefaultTimeoutConfig())
	wrapped := timeoutMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, status)
	}
}

func TestTimeout_PathSpecific(t *testing.T) {
	config := TimeoutConfig{
		Default: 100 * time.Millisecond,
		PathTimeouts: map[string]time.Duration{
			"/long": 5 * time.Second,
		},
	}

	var gotTimeout time.Duration
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, _ := r.Context().Deadline()
		gotTimeout = time.Until(deadline)
		w.WriteHeader(http.StatusOK)
	})

	timeoutMiddleware := Timeout(config)
	wrapped := timeoutMiddleware(handler)

	// Test default timeout
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if gotTimeout > 200*time.Millisecond {
		t.Errorf("Expected default timeout (~100ms), got %v", gotTimeout)
	}

	// Test path-specific timeout
	gotTimeout = 0
	req = httptest.NewRequest("GET", "/long/operation", nil)
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if gotTimeout < 4*time.Second {
		t.Errorf("Expected long timeout (~5s), got %v", gotTimeout)
	}
}

func TestTimeout_ActualTimeout(t *testing.T) {
	config := TimeoutConfig{
		Default: 50 * time.Millisecond,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow handler
		select {
		case <-time.After(200 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
			// Context cancelled - this is expected
			return
		}
	})

	timeoutMiddleware := Timeout(config)
	wrapped := timeoutMiddleware(handler)

	req := httptest.NewRequest("GET", "/slow", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	wrapped.ServeHTTP(rr, req)
	elapsed := time.Since(start)

	// Should timeout quickly, not wait for handler
	if elapsed > 150*time.Millisecond {
		t.Errorf("Expected timeout around 50ms, took %v", elapsed)
	}
}

func TestTimeout_ContextCancellation(t *testing.T) {
	config := TimeoutConfig{
		Default: 10 * time.Millisecond,
	}

	contextCancelled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		contextCancelled = true
	})

	timeoutMiddleware := Timeout(config)
	wrapped := timeoutMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if !contextCancelled {
		t.Error("Expected context to be cancelled on timeout")
	}
}

func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config.Default != 60*time.Second {
		t.Errorf("Expected default timeout 60s, got %v", config.Default)
	}

	// Check backup timeout
	backupTimeout, ok := config.PathTimeouts["/lab.v1.BackupService/"]
	if !ok {
		t.Error("Expected backup service timeout")
	} else if backupTimeout != 5*time.Minute {
		t.Errorf("Expected backup timeout 5m, got %v", backupTimeout)
	}

	// Check health timeout
	healthTimeout, ok := config.PathTimeouts["/lab.v1.HealthService/"]
	if !ok {
		t.Error("Expected health service timeout")
	} else if healthTimeout != 10*time.Second {
		t.Errorf("Expected health timeout 10s, got %v", healthTimeout)
	}
}

func TestRequestID(t *testing.T) {
	var gotRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequestID = r.Header.Get("X-Request-ID")
		if gotRequestID == "" {
			// Try context
			if val := r.Context().Value("requestID"); val != nil {
				gotRequestID = val.(string)
			}
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestID(handler)

	// Test without existing request ID
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if gotRequestID == "" {
		t.Error("Expected request ID to be generated")
	}
	if rr.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID in response header")
	}

	// Test with existing request ID
	gotRequestID = ""
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-id-123")
	rr = httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if gotRequestID != "existing-id-123" {
		t.Errorf("Expected to preserve existing request ID, got %s", gotRequestID)
	}
}

func TestTimeoutResponseWriter(t *testing.T) {
	rw := &timeoutResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
		timedOut:       false,
		wroteHeader:    false,
	}

	// First write should succeed
	rw.WriteHeader(http.StatusOK)
	if !rw.wroteHeader {
		t.Error("Expected wroteHeader to be true")
	}

	// Second write should be ignored
	rw.WriteHeader(http.StatusInternalServerError)
	// Should still be 200
}

func TestTimeout_WithSlowHandler(t *testing.T) {
	config := TimeoutConfig{
		Default: 100 * time.Millisecond,
	}

	handlerCalled := false
	contextCancelled := false
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		
		// Wait for context cancellation or complete
		select {
		case <-time.After(200 * time.Millisecond):
			// Handler completed (shouldn't happen in this test)
			w.Write([]byte("late response"))
		case <-r.Context().Done():
			contextCancelled = true
			return
		}
	})

	timeoutMiddleware := Timeout(config)
	wrapped := timeoutMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	start := time.Now()
	wrapped.ServeHTTP(rr, req)
	elapsed := time.Since(start)

	if !handlerCalled {
		t.Error("Expected handler to be called")
	}

	if !contextCancelled {
		t.Error("Expected context to be cancelled")
	}

	// Handler should return quickly after context cancellation
	// Allow some slack for goroutine scheduling
	if elapsed > 180*time.Millisecond {
		t.Errorf("Expected quick return after timeout (~100ms), took %v", elapsed)
	}
}
