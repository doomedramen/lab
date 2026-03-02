package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

func newTestStackService() *service.StackService {
	return service.NewStackService(sqlite.NewStackRepository())
}

// --- ContainerBashHandler ---

func TestContainerBashHandler_MissingToken(t *testing.T) {
	h := ContainerBashHandler(newTestStackService())
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-bash", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestContainerBashHandler_InvalidToken(t *testing.T) {
	h := ContainerBashHandler(newTestStackService())
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-bash?token=badtoken", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestContainerBashHandler_ValidToken_TriesToUpgradeWebSocket(t *testing.T) {
	svc := newTestStackService()
	token, err := svc.GetContainerToken("mystack", "web-1")
	if err != nil {
		t.Fatalf("GetContainerToken: %v", err)
	}

	h := ContainerBashHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-bash?token="+token, nil)
	// No Upgrade header — WebSocket accept will fail, but we should NOT get 401 or 403
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("expected non-auth error with valid token, got %d", w.Code)
	}
}

// --- StackLogsHandler ---

func TestStackLogsHandler_MissingToken(t *testing.T) {
	h := StackLogsHandler(newTestStackService(), "/tmp/stacks")
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-logs", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestStackLogsHandler_InvalidToken(t *testing.T) {
	h := StackLogsHandler(newTestStackService(), "/tmp/stacks")
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-logs?token=badtoken", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestStackLogsHandler_ValidToken_MissingComposeFile(t *testing.T) {
	svc := newTestStackService()
	token, err := svc.GetStackLogsToken("nonexistent-stack")
	if err != nil {
		t.Fatalf("GetStackLogsToken: %v", err)
	}

	// Use a non-existent stacks dir — docker compose will fail to start
	h := StackLogsHandler(svc, "/tmp/nonexistent-stacks-dir")
	req := httptest.NewRequest(http.MethodGet, "/ws/stack-logs?token="+token, nil)
	w := httptest.NewRecorder()
	h(w, req)

	// Should not be 401 (auth passed), should be 500 because docker command will fail
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("expected non-auth error for valid token, got %d", w.Code)
	}
}
