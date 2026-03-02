package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

func TestRateLimiter_AllowsWithinLimit(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(100), 10)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request should be allowed
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
}

func TestRateLimiter_BlocksWhenExceeded(t *testing.T) {
	// Very strict limiter: 1 per second, burst of 1 — second request must be denied
	rl := NewRateLimiter(rate.Limit(1), 1)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request: should pass (consumes the burst token)
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, req)
	if rr1.Code != http.StatusOK {
		t.Errorf("First request: expected 200, got %d", rr1.Code)
	}

	// Second request immediately: should be rate-limited
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req)
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request: expected 429, got %d", rr2.Code)
	}

	// Verify Retry-After header is set
	if rr2.Header().Get("Retry-After") == "" {
		t.Error("Expected Retry-After header to be set")
	}
}

func TestRateLimiter_DifferentIPsHaveSeparateBuckets(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 1)

	handler := rl.HTTPMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeReq := func(ip string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Real-IP", ip)
		return req
	}

	// First request from IP A should pass
	rr1 := httptest.NewRecorder()
	handler.ServeHTTP(rr1, makeReq("1.2.3.4"))
	if rr1.Code != http.StatusOK {
		t.Errorf("IP A first request: expected 200, got %d", rr1.Code)
	}

	// Second request from IP A should be blocked
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, makeReq("1.2.3.4"))
	if rr2.Code != http.StatusTooManyRequests {
		t.Errorf("IP A second request: expected 429, got %d", rr2.Code)
	}

	// First request from IP B should pass (separate bucket)
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, makeReq("5.6.7.8"))
	if rr3.Code != http.StatusOK {
		t.Errorf("IP B first request: expected 200, got %d", rr3.Code)
	}
}

func TestRateLimiter_Interceptor(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 1)
	interceptor := rl.Interceptor()

	callCount := 0
	handler := func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		callCount++
		return nil, nil
	}

	wrapped := interceptor(handler)

	// Create mock request with IP header
	req := func() connect.AnyRequest {
		return &mockRequest{header: http.Header{}}
	}

	// First request should pass
	_, err := wrapped(context.Background(), req())
	if err != nil {
		t.Errorf("First request: unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}

	// Second request should be rate limited
	_, err = wrapped(context.Background(), req())
	if err == nil {
		t.Error("Second request: expected error, got nil")
	} else {
		connectErr, ok := err.(*connect.Error)
		if !ok {
			t.Errorf("Expected connect.Error, got %T", err)
		} else if connectErr.Code() != connect.CodeResourceExhausted {
			t.Errorf("Expected CodeResourceExhausted, got %v", connectErr.Code())
		}
	}
}

func TestExtractIP_XRealIP(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Real-IP", "203.0.113.1")
	if got := extractIP(headers); got != "203.0.113.1" {
		t.Errorf("extractIP = %q, want 203.0.113.1", got)
	}
}

func TestExtractIP_XForwardedFor(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "203.0.113.2, 10.0.0.1")
	if got := extractIP(headers); got != "203.0.113.2" {
		t.Errorf("extractIP = %q, want 203.0.113.2", got)
	}
}

func TestExtractIP_NoHeaders(t *testing.T) {
	headers := http.Header{}
	if got := extractIP(headers); got != "unknown" {
		t.Errorf("extractIP = %q, want unknown", got)
	}
}

// mockRequest implements connect.AnyRequest for testing
type mockRequest struct {
	connect.AnyRequest
	header http.Header
}

func (m *mockRequest) Header() http.Header {
	return m.header
}
