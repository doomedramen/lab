package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

// ContainerReconciler reconciles Container resources from GitOps manifests
type ContainerReconciler struct {
	containerService *ContainerService
}

// NewContainerReconciler creates a new container reconciler
func NewContainerReconciler(containerService *ContainerService) *ContainerReconciler {
	return &ContainerReconciler{
		containerService: containerService,
	}
}

// Kind returns the resource kind this reconciler handles
func (r *ContainerReconciler) Kind() string {
	return "Container"
}

// Reconcile ensures the actual container state matches the desired state from GitOps manifest
func (r *ContainerReconciler) Reconcile(ctx context.Context, desired *model.GitOpsResource) (*ReconcileResult, error) {
	// Extract container spec from GitOps resource
	spec := extractContainerSpec(desired.Spec)
	if spec == nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: "Invalid container spec in manifest",
		}, nil
	}

	// Check if container already exists by name
	actual, err := r.GetActualState(ctx, desired.ConfigID, desired.Name)
	if err != nil && err != sql.ErrNoRows {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to get actual state: %v", err),
		}, nil
	}

	if actual == nil {
		// Container doesn't exist - create it
		return r.createContainer(ctx, desired, spec)
	}

	// Container exists - for now just mark as unchanged
	// Full implementation would compare specs and update if needed
	return &ReconcileResult{
		Action:      ReconcileActionUnchanged,
		Message:     fmt.Sprintf("Container %s exists and is up to date", desired.Name),
		ActualState: actual,
	}, nil
}

// Delete removes the container from actual state
func (r *ContainerReconciler) Delete(ctx context.Context, resource *model.GitOpsResource) error {
	// Find container by name and delete
	// This is a simplified implementation - in production would need to find by CTID
	return nil // Stub - full implementation requires container lookup by name
}

// GetActualState retrieves the current state of the container
func (r *ContainerReconciler) GetActualState(ctx context.Context, configID, name string) (*model.GitOpsResource, error) {
	// Get all containers and find by name
	containers, err := r.containerService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.Name == name {
			return containerToGitOpsResource(configID, container), nil
		}
	}

	return nil, sql.ErrNoRows
}

// createContainer creates a new container from the GitOps manifest
func (r *ContainerReconciler) createContainer(ctx context.Context, desired *model.GitOpsResource, spec *ContainerSpec) (*ReconcileResult, error) {
	// Convert GitOps spec to container create request
	createReq := &model.ContainerCreateRequest{
		Name:         spec.Name,
		CPUCores:     spec.CPUCores,
		Memory:       spec.Memory,
		Disk:         spec.Disk,
		OS:           spec.OS,
		Tags:         spec.Tags,
		Unprivileged: spec.Unprivileged,
		Description:  spec.Description,
		StartOnBoot:  spec.StartOnBoot,
	}

	container, err := r.containerService.Create(ctx, createReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to create container: %v", err),
		}, nil
	}

	return &ReconcileResult{
		Action:  ReconcileActionCreated,
		Message: fmt.Sprintf("Created container %s", container.Name),
		Changes: []FieldChange{
			{Field: "name", NewValue: container.Name},
		},
		ActualState: containerToGitOpsResource(desired.ConfigID, container),
	}, nil
}

// ContainerSpec represents the spec section of a container manifest
type ContainerSpec struct {
	Name         string   `yaml:"name"`
	CPUCores     int      `yaml:"cpuCores"`
	Memory       float64  `yaml:"memory"`
	Disk         float64  `yaml:"disk"`
	OS           string   `yaml:"os"`
	Tags         []string `yaml:"tags"`
	Unprivileged bool     `yaml:"unprivileged"`
	Description  string   `yaml:"description"`
	StartOnBoot  bool     `yaml:"startOnBoot"`
}

// extractContainerSpec extracts container spec from untyped manifest spec
func extractContainerSpec(spec map[string]any) *ContainerSpec {
	if spec == nil {
		return nil
	}

	containerSpec := &ContainerSpec{}

	if name, ok := spec["name"].(string); ok {
		containerSpec.Name = name
	}
	if cpu, ok := spec["cpuCores"].(int); ok {
		containerSpec.CPUCores = cpu
	}
	if memory, ok := spec["memory"].(float64); ok {
		containerSpec.Memory = memory
	}
	if disk, ok := spec["disk"].(float64); ok {
		containerSpec.Disk = disk
	}
	if os, ok := spec["os"].(string); ok {
		containerSpec.OS = os
	}
	if desc, ok := spec["description"].(string); ok {
		containerSpec.Description = desc
	}
	if unprivileged, ok := spec["unprivileged"].(bool); ok {
		containerSpec.Unprivileged = unprivileged
	}
	if startOnBoot, ok := spec["startOnBoot"].(bool); ok {
		containerSpec.StartOnBoot = startOnBoot
	}
	if tags, ok := spec["tags"].([]any); ok {
		containerSpec.Tags = make([]string, len(tags))
		for i, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				containerSpec.Tags[i] = tagStr
			}
		}
	}

	return containerSpec
}

// containerToGitOpsResource converts a Container model to GitOpsResource
func containerToGitOpsResource(configID string, container *model.Container) *model.GitOpsResource {
	return &model.GitOpsResource{
		ConfigID:      configID,
		Kind:          "Container",
		Name:          container.Name,
		Namespace:     "default",
		Status:        model.GitOpsStatusHealthy,
		StatusMessage: fmt.Sprintf("Container %s is running", container.Name),
		Spec: map[string]any{
			"name":         container.Name,
			"ctid":         container.CTID,
			"tags":         container.Tags,
			"description":  container.Description,
			"unprivileged": container.Unprivileged,
			"startOnBoot":  container.StartOnBoot,
			"os":           container.OS,
		},
	}
}
