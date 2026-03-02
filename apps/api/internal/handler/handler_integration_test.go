//go:build integration
// +build integration

package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthCheckIntegration tests the health check handler with real HTTP
func TestHealthCheckIntegration(t *testing.T) {
	// Create real HTTP test server
	server := httptest.NewServer(http.HandlerFunc(HealthCheck))
	defer server.Close()

	// Test GET request
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	expected := `{"status":"ok"}`
	if string(body) != expected && string(body) != expected+"\n" {
		t.Errorf("GET /health body = %q, want %q", string(body), expected)
	}

	// Test POST request (should also work)
	resp, err = http.Post(server.URL, "application/json", nil)
	if err != nil {
		t.Fatalf("POST /health failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("POST /health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestHealthCheckConcurrent tests concurrent health check requests
func TestHealthCheckConcurrent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(HealthCheck))
	defer server.Close()

	// Make 10 concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			resp, err := http.Get(server.URL)
			if err != nil {
				t.Errorf("Concurrent request failed: %v", err)
				done <- false
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Concurrent request status = %d, want %d", resp.StatusCode, http.StatusOK)
				done <- false
				return
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
