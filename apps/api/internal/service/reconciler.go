package service

import (
	"context"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// Reconciler defines the interface for reconciling a specific resource kind
type Reconciler interface {
	// Kind returns the resource kind this reconciler handles
	Kind() string
	
	// Reconcile ensures the actual state matches the desired state
	Reconcile(ctx context.Context, desired *model.GitOpsResource) (*ReconcileResult, error)
	
	// Delete removes the resource from actual state
	Delete(ctx context.Context, resource *model.GitOpsResource) error
	
	// GetActualState retrieves the current state of the resource
	GetActualState(ctx context.Context, configID, name string) (*model.GitOpsResource, error)
}

// ReconcileResult represents the outcome of a reconciliation operation
type ReconcileResult struct {
	Action      ReconcileAction `json:"action"`      // created, updated, unchanged, failed
	Message     string          `json:"message"`     // Human-readable description
	Changes     []FieldChange   `json:"changes"`     // List of field changes
	ActualState *model.GitOpsResource `json:"actual_state"` // Current state after reconciliation
}

// ReconcileAction represents the type of reconciliation action
type ReconcileAction string

const (
	ReconcileActionCreated   ReconcileAction = "created"
	ReconcileActionUpdated   ReconcileAction = "updated"
	ReconcileActionUnchanged ReconcileAction = "unchanged"
	ReconcileActionFailed    ReconcileAction = "failed"
	ReconcileActionDeleted   ReconcileAction = "deleted"
)

// FieldChange represents a single field change during reconciliation
type FieldChange struct {
	Field    string `json:"field"`
	OldValue any    `json:"old_value,omitempty"`
	NewValue any    `json:"new_value,omitempty"`
}

// ReconcilerRegistry holds all available reconcilers
type ReconcilerRegistry struct {
	reconcilers map[string]Reconciler
}

// NewReconcilerRegistry creates a new reconciler registry
func NewReconcilerRegistry() *ReconcilerRegistry {
	return &ReconcilerRegistry{
		reconcilers: make(map[string]Reconciler),
	}
}

// Register adds a reconciler to the registry
func (r *ReconcilerRegistry) Register(reconciler Reconciler) {
	r.reconcilers[reconciler.Kind()] = reconciler
}

// GetReconciler returns the reconciler for a specific kind
func (r *ReconcilerRegistry) GetReconciler(kind string) (Reconciler, bool) {
	reconciler, ok := r.reconcilers[kind]
	return reconciler, ok
}

// ListKinds returns all registered resource kinds
func (r *ReconcilerRegistry) ListKinds() []string {
	kinds := make([]string, 0, len(r.reconcilers))
	for kind := range r.reconcilers {
		kinds = append(kinds, kind)
	}
	return kinds
}
