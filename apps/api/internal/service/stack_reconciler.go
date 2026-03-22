package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// StackReconciler reconciles DockerStack resources from GitOps manifests
type StackReconciler struct {
	stackService *StackService
}

// NewStackReconciler creates a new stack reconciler
func NewStackReconciler(stackService *StackService) *StackReconciler {
	return &StackReconciler{
		stackService: stackService,
	}
}

// Kind returns the resource kind this reconciler handles
func (r *StackReconciler) Kind() string {
	return "DockerStack"
}

// Reconcile ensures the actual Docker stack state matches the desired state from GitOps manifest
func (r *StackReconciler) Reconcile(ctx context.Context, desired *model.GitOpsResource) (*ReconcileResult, error) {
	// Extract Docker stack spec from GitOps resource
	spec := extractStackSpec(desired.Spec)
	if spec == nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: "Invalid Docker stack spec in manifest",
		}, nil
	}

	// Check if stack already exists by name
	actual, err := r.GetActualState(ctx, desired.ConfigID, desired.Name)
	if err != nil && err != sql.ErrNoRows {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to get actual state: %v", err),
		}, nil
	}

	if actual == nil {
		// Stack doesn't exist - create it
		return r.createStack(ctx, desired, spec)
	}

	// Stack exists - check if update is needed
	return r.updateStack(ctx, desired, actual, spec)
}

// Delete removes the Docker stack from actual state
func (r *StackReconciler) Delete(ctx context.Context, resource *model.GitOpsResource) error {
	// Find stack by name and delete
	stacks, err := r.stackService.GetAll(ctx)
	if err != nil {
		return err
	}

	for _, stack := range stacks {
		if stack.Name == resource.Name {
			// First bring down the stack, then delete it
			if err := r.stackService.Down(ctx, stack.ID); err != nil {
				return fmt.Errorf("failed to bring down stack: %w", err)
			}
			return r.stackService.Delete(ctx, stack.ID)
		}
	}

	return nil
}

// GetActualState retrieves the current state of the Docker stack
func (r *StackReconciler) GetActualState(ctx context.Context, configID, name string) (*model.GitOpsResource, error) {
	// Get all stacks and find by name
	stacks, err := r.stackService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, stack := range stacks {
		if stack.Name == name {
			return stackToGitOpsResource(configID, stack), nil
		}
	}

	return nil, sql.ErrNoRows
}

// createStack creates a new Docker stack from the GitOps manifest
func (r *StackReconciler) createStack(ctx context.Context, desired *model.GitOpsResource, spec *StackSpec) (*ReconcileResult, error) {
	// Convert GitOps spec to stack create request
	createReq := &model.StackCreateRequest{
		Name:    spec.Name,
		Compose: spec.Compose,
		Env:     spec.Env,
	}

	stack, err := r.stackService.Create(ctx, createReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to create Docker stack: %v", err),
		}, nil
	}

	// Start the stack after creation
	if err := r.stackService.Start(ctx, stack.ID); err != nil {
		// Log warning but don't fail - stack is created but not started
	}

	changes := []FieldChange{
		{Field: "name", NewValue: stack.Name},
		{Field: "status", NewValue: string(stack.Status)},
	}

	return &ReconcileResult{
		Action:      ReconcileActionCreated,
		Message:     fmt.Sprintf("Created Docker stack %s", stack.Name),
		Changes:     changes,
		ActualState: stackToGitOpsResource(desired.ConfigID, stack),
	}, nil
}

// updateStack updates an existing Docker stack if needed
func (r *StackReconciler) updateStack(ctx context.Context, desired, actual *model.GitOpsResource, spec *StackSpec) (*ReconcileResult, error) {
	// Compare specs and calculate changes
	changes := calculateStackChanges(actual.Spec, spec)

	if len(changes) == 0 {
		// No changes needed
		return &ReconcileResult{
			Action:      ReconcileActionUnchanged,
			Message:     fmt.Sprintf("Docker stack %s exists and is up to date", desired.Name),
			ActualState: actual,
		}, nil
	}

	// Get the stack ID from actual state
	stackID := getStackIDFromResource(actual)
	if stackID == "" {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: "Failed to get stack ID from actual state",
		}, nil
	}

	// Update the stack
	updateReq := &model.StackUpdateRequest{
		Compose: spec.Compose,
		Env:     spec.Env,
	}

	stack, err := r.stackService.Update(ctx, stackID, updateReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to update Docker stack: %v", err),
		}, nil
	}

	// Restart the stack to apply changes
	if err := r.stackService.Restart(ctx, stack.ID); err != nil {
		// Log warning but don't fail
	}

	return &ReconcileResult{
		Action:      ReconcileActionUpdated,
		Message:     fmt.Sprintf("Updated Docker stack %s", stack.Name),
		Changes:     changes,
		ActualState: stackToGitOpsResource(desired.ConfigID, stack),
	}, nil
}

// StackSpec represents the spec section of a Docker stack manifest
type StackSpec struct {
	Name        string `yaml:"name"`
	Compose     string `yaml:"compose"`
	Env         string `yaml:"env"`
	Description string `yaml:"description"`
}

// extractStackSpec extracts Docker stack spec from untyped manifest spec
func extractStackSpec(spec map[string]any) *StackSpec {
	if spec == nil {
		return nil
	}

	stackSpec := &StackSpec{}

	if name, ok := spec["name"].(string); ok {
		stackSpec.Name = name
	}
	if compose, ok := spec["compose"].(string); ok {
		stackSpec.Compose = compose
	}
	if env, ok := spec["env"].(string); ok {
		stackSpec.Env = env
	}
	if desc, ok := spec["description"].(string); ok {
		stackSpec.Description = desc
	}

	return stackSpec
}

// calculateStackChanges calculates the differences between actual and desired stack specs
func calculateStackChanges(actualSpec map[string]any, desiredSpec *StackSpec) []FieldChange {
	var changes []FieldChange

	if actualSpec == nil {
		return changes
	}

	// Compare name
	if actualName, ok := actualSpec["name"].(string); ok && actualName != desiredSpec.Name {
		changes = append(changes, FieldChange{
			Field:    "name",
			OldValue: actualName,
			NewValue: desiredSpec.Name,
		})
	}

	// Compare compose content
	if actualCompose, ok := actualSpec["compose"].(string); ok && actualCompose != desiredSpec.Compose {
		changes = append(changes, FieldChange{
			Field:    "compose",
			OldValue: actualCompose,
			NewValue: desiredSpec.Compose,
		})
	}

	// Compare env content
	if actualEnv, ok := actualSpec["env"].(string); ok && actualEnv != desiredSpec.Env {
		changes = append(changes, FieldChange{
			Field:    "env",
			OldValue: actualEnv,
			NewValue: desiredSpec.Env,
		})
	}

	// Compare description
	if actualDesc, ok := actualSpec["description"].(string); ok && actualDesc != desiredSpec.Description {
		changes = append(changes, FieldChange{
			Field:    "description",
			OldValue: actualDesc,
			NewValue: desiredSpec.Description,
		})
	}

	// Compare status
	if actualStatus, ok := actualSpec["status"].(string); ok {
		desiredStatus := string(model.StackStatusRunning)
		if actualStatus != desiredStatus {
			changes = append(changes, FieldChange{
				Field:    "status",
				OldValue: actualStatus,
				NewValue: desiredStatus,
			})
		}
	}

	return changes
}

// stackToGitOpsResource converts a DockerStack model to GitOpsResource
func stackToGitOpsResource(configID string, stack *model.DockerStack) *model.GitOpsResource {
	// Build container info from stack containers
	var containers []map[string]any
	for _, c := range stack.Containers {
		containers = append(containers, map[string]any{
			"serviceName":   c.ServiceName,
			"containerName": c.ContainerName,
			"containerId":   c.ContainerID,
			"image":         c.Image,
			"state":         c.State,
			"ports":         c.Ports,
		})
	}

	return &model.GitOpsResource{
		ConfigID:      configID,
		Kind:          "DockerStack",
		Name:          stack.Name,
		Namespace:     "default",
		Status:        model.GitOpsStatusHealthy,
		StatusMessage: fmt.Sprintf("Docker stack %s is %s", stack.Name, stack.Status),
		Spec: map[string]any{
			"name":        stack.Name,
			"compose":     stack.Compose,
			"env":         stack.Env,
			"status":      string(stack.Status),
			"containers":  containers,
			"description": "",
		},
	}
}

// getStackIDFromResource extracts the stack ID from a GitOpsResource
func getStackIDFromResource(resource *model.GitOpsResource) string {
	if resource == nil || resource.Spec == nil {
		return ""
	}
	if id, ok := resource.Spec["id"].(string); ok {
		return id
	}
	return ""
}

// calculateStackDiff generates a human-readable diff between old and new stack specs
func calculateStackDiff(oldSpec, newSpec *StackSpec) string {
	var diff string

	if oldSpec == nil {
		return "New stack creation"
	}

	if oldSpec.Compose != newSpec.Compose {
		diff += "  compose: changed\n"
	}
	if oldSpec.Env != newSpec.Env {
		diff += "  env: changed\n"
	}
	if oldSpec.Name != newSpec.Name {
		diff += fmt.Sprintf("  name: %s -> %s\n", oldSpec.Name, newSpec.Name)
	}

	if diff == "" {
		return "No changes"
	}

	return diff
}

// validateStackSpec validates a stack spec before reconciliation
func validateStackSpec(spec *StackSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("stack name is required")
	}
	if spec.Compose == "" {
		return fmt.Errorf("compose content is required")
	}
	return nil
}
