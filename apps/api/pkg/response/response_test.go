package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name   string
		status int
		data   interface{}
	}{
		{"ok with map", http.StatusOK, map[string]string{"status": "ok"}},
		{"created with struct", http.StatusCreated, ActionResponse{Success: true, Message: "done"}},
		{"empty body", http.StatusNoContent, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.status, tt.data)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}
			if ct := w.Header().Get("Content-Type"); ct != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", ct)
			}
		})
	}
}

func TestJSON_DecodesCorrectly(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, map[string]int{"count": 42})

	var got map[string]int
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["count"] != 42 {
		t.Errorf("count = %d, want 42", got["count"])
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		status  int
		message string
	}{
		{http.StatusBadRequest, "invalid input"},
		{http.StatusNotFound, "not found"},
		{http.StatusInternalServerError, "something went wrong"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.status), func(t *testing.T) {
			w := httptest.NewRecorder()
			Error(w, tt.status, tt.message)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}

			var got ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Message != tt.message {
				t.Errorf("message = %q, want %q", got.Message, tt.message)
			}
			if got.Code != tt.status {
				t.Errorf("code = %d, want %d", got.Code, tt.status)
			}
		})
	}
}

func TestErrorFromError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"not found", fmt.Errorf("item: %w", ErrNotFound), http.StatusNotFound},
		{"bad request", fmt.Errorf("parse: %w", ErrBadRequest), http.StatusBadRequest},
		{"conflict", fmt.Errorf("dup: %w", ErrConflict), http.StatusConflict},
		{"unknown error", errors.New("something failed"), http.StatusInternalServerError},
		{"bare not found", ErrNotFound, http.StatusNotFound},
		{"bare bad request", ErrBadRequest, http.StatusBadRequest},
		{"bare conflict", ErrConflict, http.StatusConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			ErrorFromError(w, tt.err)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			var got ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got.Message != tt.err.Error() {
				t.Errorf("message = %q, want %q", got.Message, tt.err.Error())
			}
		})
	}
}

func TestActionResponse_JSONRoundTrip(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, ActionResponse{Success: true, Message: "created"})

	var got ActionResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Success {
		t.Error("expected Success=true")
	}
	if got.Message != "created" {
		t.Errorf("Message = %q, want %q", got.Message, "created")
	}
}

func TestActionResponse_OmitsEmptyMessage(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, http.StatusOK, ActionResponse{Success: true})

	var raw map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := raw["message"]; ok {
		t.Error("expected message to be omitted when empty")
	}
}
