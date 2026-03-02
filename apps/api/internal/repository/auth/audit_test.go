package auth

import (
	"context"
	"testing"
	"time"
)

func TestAuditLogRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	log := &AuditLog{
		UserID:       "user-1",
		Action:       "vm.start",
		ResourceType: "vm",
		ResourceID:   "vm-100",
		Details:      map[string]any{"node": "pve1"},
		IPAddress:    "10.0.0.1",
		UserAgent:    "test-agent",
		Status:       StatusSuccess,
	}

	if err := repo.Create(ctx, log); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Verify via List
	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Action != "vm.start" {
		t.Errorf("Action = %q, want vm.start", logs[0].Action)
	}
	if logs[0].Details["node"] != "pve1" {
		t.Errorf("Details[node] = %v, want pve1", logs[0].Details["node"])
	}
}

func TestAuditLogRepository_Create_NilDetails(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	log := &AuditLog{
		UserID: "user-1",
		Action: "user.logout",
		Status: StatusSuccess,
	}

	if err := repo.Create(ctx, log); err != nil {
		t.Fatalf("Create with nil details: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Details == nil {
		t.Error("expected Details to be initialized (empty map)")
	}
}

func TestAuditLogRepository_LogLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	if err := repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "Mozilla/5.0", true); err != nil {
		t.Fatalf("LogLogin: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Action != "user.login" {
		t.Errorf("Action = %q, want user.login", logs[0].Action)
	}
	if logs[0].Status != StatusSuccess {
		t.Errorf("Status = %q, want success", logs[0].Status)
	}
	if logs[0].Details["email"] != "alice@example.com" {
		t.Errorf("Details[email] = %v, want alice@example.com", logs[0].Details["email"])
	}
}

func TestAuditLogRepository_LogLogin_Failure(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	repo.LogLogin(ctx, "", "bad@example.com", "10.0.0.1", "curl", false)

	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Status != StatusFailure {
		t.Errorf("Status = %q, want failure", logs[0].Status)
	}
}

func TestAuditLogRepository_LogLogout(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	if err := repo.LogLogout(ctx, "user-1", "10.0.0.1", "Mozilla/5.0"); err != nil {
		t.Fatalf("LogLogout: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].Action != "user.logout" {
		t.Errorf("Action = %q, want user.logout", logs[0].Action)
	}
}

func TestAuditLogRepository_LogAPIKeyCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	err := repo.LogAPIKeyCreate(ctx, "user-1", "key-123", "my-key", "10.0.0.1", "cli")
	if err != nil {
		t.Fatalf("LogAPIKeyCreate: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].ResourceType != "api_key" {
		t.Errorf("ResourceType = %q, want api_key", logs[0].ResourceType)
	}
	if logs[0].ResourceID != "key-123" {
		t.Errorf("ResourceID = %q, want key-123", logs[0].ResourceID)
	}
}

func TestAuditLogRepository_LogAPIKeyUse(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	err := repo.LogAPIKeyUse(ctx, "user-1", "key-123", "10.0.0.1", "cli")
	if err != nil {
		t.Fatalf("LogAPIKeyUse: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if logs[0].Action != "api_key.use" {
		t.Errorf("Action = %q, want api_key.use", logs[0].Action)
	}
}

func TestAuditLogRepository_LogResourceAction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	details := map[string]any{"reason": "maintenance"}
	err := repo.LogResourceAction(ctx, "user-1", "vm.stop", "vm", "vm-100", "10.0.0.1", "api", details, true)
	if err != nil {
		t.Fatalf("LogResourceAction: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if logs[0].Action != "vm.stop" {
		t.Errorf("Action = %q, want vm.stop", logs[0].Action)
	}
	if logs[0].Status != StatusSuccess {
		t.Errorf("Status = %q, want success", logs[0].Status)
	}
}

func TestAuditLogRepository_LogResourceAction_Failure(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	err := repo.LogResourceAction(ctx, "user-1", "vm.delete", "vm", "vm-100", "10.0.0.1", "api", nil, false)
	if err != nil {
		t.Fatalf("LogResourceAction: %v", err)
	}

	logs, _ := repo.List(ctx, 10, 0)
	if logs[0].Status != StatusFailure {
		t.Errorf("Status = %q, want failure", logs[0].Status)
	}
}

func TestAuditLogRepository_List_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	for range 5 {
		repo.LogLogout(ctx, "user-1", "10.0.0.1", "agent")
	}

	// First page
	logs, _ := repo.List(ctx, 2, 0)
	if len(logs) != 2 {
		t.Errorf("page 1: expected 2, got %d", len(logs))
	}

	// Second page
	logs, _ = repo.List(ctx, 2, 2)
	if len(logs) != 2 {
		t.Errorf("page 2: expected 2, got %d", len(logs))
	}

	// Third page
	logs, _ = repo.List(ctx, 2, 4)
	if len(logs) != 1 {
		t.Errorf("page 3: expected 1, got %d", len(logs))
	}
}

func TestAuditLogRepository_ListByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "agent", true)
	repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "agent", true)
	repo.LogLogin(ctx, "user-2", "bob@example.com", "10.0.0.2", "agent", true)

	logs, err := repo.ListByUser(ctx, "user-1", 10, 0)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs for user-1, got %d", len(logs))
	}
}

func TestAuditLogRepository_ListByAction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "agent", true)
	repo.LogLogout(ctx, "user-1", "10.0.0.1", "agent")
	repo.LogLogin(ctx, "user-2", "bob@example.com", "10.0.0.2", "agent", true)

	logs, err := repo.ListByAction(ctx, "user.login", 10, 0)
	if err != nil {
		t.Fatalf("ListByAction: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 login logs, got %d", len(logs))
	}
}

func TestAuditLogRepository_DeleteOld(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	// Create some logs
	repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "agent", true)
	repo.LogLogin(ctx, "user-2", "bob@example.com", "10.0.0.2", "agent", true)

	// Delete logs older than 1 hour in the future (should delete all)
	deleted, err := repo.DeleteOld(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("DeleteOld: %v", err)
	}
	if deleted != 2 {
		t.Errorf("deleted = %d, want 2", deleted)
	}

	count, _ := repo.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0 after delete, got %d", count)
	}
}

func TestAuditLogRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAuditLogRepository(db)
	ctx := context.Background()

	count, _ := repo.Count(ctx)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	repo.LogLogin(ctx, "user-1", "alice@example.com", "10.0.0.1", "agent", true)
	repo.LogLogout(ctx, "user-1", "10.0.0.1", "agent")

	count, _ = repo.Count(ctx)
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}
