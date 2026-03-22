package service

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

// StoragePoolReconciler reconciles StoragePool resources from GitOps manifests
type StoragePoolReconciler struct {
	storageService *StorageService
}

// NewStoragePoolReconciler creates a new storage pool reconciler
func NewStoragePoolReconciler(storageService *StorageService) *StoragePoolReconciler {
	return &StoragePoolReconciler{
		storageService: storageService,
	}
}

// Kind returns the resource kind this reconciler handles
func (r *StoragePoolReconciler) Kind() string {
	return "StoragePool"
}

// Reconcile ensures the actual storage pool state matches the desired state from GitOps manifest
func (r *StoragePoolReconciler) Reconcile(ctx context.Context, desired *model.GitOpsResource) (*ReconcileResult, error) {
	// Extract storage pool spec from GitOps resource
	spec := extractStoragePoolSpec(desired.Spec)
	if spec == nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: "Invalid storage pool spec in manifest",
		}, nil
	}

	// Check if storage pool already exists by name
	actual, err := r.GetActualState(ctx, desired.ConfigID, desired.Name)
	if err != nil && err != sql.ErrNoRows {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to get actual state: %v", err),
		}, nil
	}

	if actual == nil {
		// Storage pool doesn't exist - create it
		return r.createStoragePool(ctx, desired, spec)
	}

	// Storage pool exists - check if update is needed
	return r.updateStoragePool(ctx, desired, actual, spec)
}

// Delete removes the storage pool from actual state
func (r *StoragePoolReconciler) Delete(ctx context.Context, resource *model.GitOpsResource) error {
	// Find storage pool by name and delete
	pools, _, err := r.storageService.ListStoragePools(ctx, labv1.StorageType_STORAGE_TYPE_UNSPECIFIED, labv1.StorageStatus_STORAGE_STATUS_UNSPECIFIED, false)
	if err != nil {
		return err
	}

	for _, pool := range pools {
		if pool.Name == resource.Name {
			// Delete with force=true to handle pools with disks
			return r.storageService.DeleteStoragePool(ctx, pool.Id, true)
		}
	}

	return nil
}

// GetActualState retrieves the current state of the storage pool
func (r *StoragePoolReconciler) GetActualState(ctx context.Context, configID, name string) (*model.GitOpsResource, error) {
	// Get all storage pools and find by name
	pools, _, err := r.storageService.ListStoragePools(ctx, labv1.StorageType_STORAGE_TYPE_UNSPECIFIED, labv1.StorageStatus_STORAGE_STATUS_UNSPECIFIED, false)
	if err != nil {
		return nil, err
	}

	for _, pool := range pools {
		if pool.Name == name {
			return storagePoolToGitOpsResource(configID, pool), nil
		}
	}

	return nil, sql.ErrNoRows
}

// createStoragePool creates a new storage pool from the GitOps manifest
func (r *StoragePoolReconciler) createStoragePool(ctx context.Context, desired *model.GitOpsResource, spec *StoragePoolSpec) (*ReconcileResult, error) {
	// Convert GitOps spec to storage pool create request
	createReq := &labv1.CreateStoragePoolRequest{
		Name:        spec.Name,
		Type:        storageTypeToProto(spec.Type),
		Path:        spec.Path,
		Options:     spec.Options,
		Enabled:     spec.Enabled,
		Description: spec.Description,
	}

	pool, err := r.storageService.CreateStoragePool(ctx, createReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to create storage pool: %v", err),
		}, nil
	}

	changes := []FieldChange{
		{Field: "name", NewValue: pool.Name},
		{Field: "type", NewValue: pool.Type.String()},
		{Field: "path", NewValue: pool.Path},
	}

	return &ReconcileResult{
		Action:      ReconcileActionCreated,
		Message:     fmt.Sprintf("Created storage pool %s", pool.Name),
		Changes:     changes,
		ActualState: storagePoolToGitOpsResource(desired.ConfigID, pool),
	}, nil
}

// updateStoragePool updates an existing storage pool if needed
func (r *StoragePoolReconciler) updateStoragePool(ctx context.Context, desired, actual *model.GitOpsResource, spec *StoragePoolSpec) (*ReconcileResult, error) {
	// Compare specs and calculate changes
	changes := calculateStoragePoolChanges(actual.Spec, spec)

	if len(changes) == 0 {
		// No changes needed
		return &ReconcileResult{
			Action:      ReconcileActionUnchanged,
			Message:     fmt.Sprintf("Storage pool %s exists and is up to date", desired.Name),
			ActualState: actual,
		}, nil
	}

	// Update the storage pool
	updateReq := &labv1.UpdateStoragePoolRequest{
		Id:          getPoolIDFromResource(actual),
		Name:        spec.Name,
		Status:      storageStatusToProto(spec.Status),
		Options:     spec.Options,
		Description: spec.Description,
		Enabled:     spec.Enabled,
	}

	pool, err := r.storageService.UpdateStoragePool(ctx, updateReq)
	if err != nil {
		return &ReconcileResult{
			Action:  ReconcileActionFailed,
			Message: fmt.Sprintf("Failed to update storage pool: %v", err),
		}, nil
	}

	return &ReconcileResult{
		Action:      ReconcileActionUpdated,
		Message:     fmt.Sprintf("Updated storage pool %s", pool.Name),
		Changes:     changes,
		ActualState: storagePoolToGitOpsResource(desired.ConfigID, pool),
	}, nil
}

// StoragePoolSpec represents the spec section of a storage pool manifest
type StoragePoolSpec struct {
	Name        string            `yaml:"name"`
	Type        string            `yaml:"type"`
	Path        string            `yaml:"path"`
	Status      string            `yaml:"status"`
	Options     map[string]string `yaml:"options"`
	Enabled     bool              `yaml:"enabled"`
	Description string            `yaml:"description"`
}

// extractStoragePoolSpec extracts storage pool spec from untyped manifest spec
func extractStoragePoolSpec(spec map[string]any) *StoragePoolSpec {
	if spec == nil {
		return nil
	}

	poolSpec := &StoragePoolSpec{}

	if name, ok := spec["name"].(string); ok {
		poolSpec.Name = name
	}
	if poolType, ok := spec["type"].(string); ok {
		poolSpec.Type = poolType
	}
	if path, ok := spec["path"].(string); ok {
		poolSpec.Path = path
	}
	if status, ok := spec["status"].(string); ok {
		poolSpec.Status = status
	}
	if enabled, ok := spec["enabled"].(bool); ok {
		poolSpec.Enabled = enabled
	}
	if desc, ok := spec["description"].(string); ok {
		poolSpec.Description = desc
	}
	if options, ok := spec["options"].(map[string]any); ok {
		poolSpec.Options = make(map[string]string)
		for k, v := range options {
			if vStr, ok := v.(string); ok {
				poolSpec.Options[k] = vStr
			}
		}
	}

	return poolSpec
}

// calculateStoragePoolChanges calculates the differences between actual and desired storage pool specs
func calculateStoragePoolChanges(actualSpec map[string]any, desiredSpec *StoragePoolSpec) []FieldChange {
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

	// Compare type
	if actualType, ok := actualSpec["type"].(string); ok && actualType != desiredSpec.Type {
		changes = append(changes, FieldChange{
			Field:    "type",
			OldValue: actualType,
			NewValue: desiredSpec.Type,
		})
	}

	// Compare path
	if actualPath, ok := actualSpec["path"].(string); ok && actualPath != desiredSpec.Path {
		changes = append(changes, FieldChange{
			Field:    "path",
			OldValue: actualPath,
			NewValue: desiredSpec.Path,
		})
	}

	// Compare status
	if actualStatus, ok := actualSpec["status"].(string); ok && actualStatus != desiredSpec.Status {
		changes = append(changes, FieldChange{
			Field:    "status",
			OldValue: actualStatus,
			NewValue: desiredSpec.Status,
		})
	}

	// Compare enabled
	if actualEnabled, ok := actualSpec["enabled"].(bool); ok && actualEnabled != desiredSpec.Enabled {
		changes = append(changes, FieldChange{
			Field:    "enabled",
			OldValue: actualEnabled,
			NewValue: desiredSpec.Enabled,
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

	// Compare options
	if actualOptions, ok := actualSpec["options"].(map[string]any); ok {
		if !reflect.DeepEqual(actualOptions, desiredSpec.Options) {
			changes = append(changes, FieldChange{
				Field:    "options",
				OldValue: actualOptions,
				NewValue: desiredSpec.Options,
			})
		}
	}

	return changes
}

// storageTypeToProto converts string storage type to proto enum
func storageTypeToProto(storageType string) labv1.StorageType {
	switch storageType {
	case "dir":
		return labv1.StorageType_STORAGE_TYPE_DIR
	case "lvm":
		return labv1.StorageType_STORAGE_TYPE_LVM
	case "zfs":
		return labv1.StorageType_STORAGE_TYPE_ZFS
	case "nfs":
		return labv1.StorageType_STORAGE_TYPE_NFS
	case "iscsi":
		return labv1.StorageType_STORAGE_TYPE_ISCSI
	case "ceph":
		return labv1.StorageType_STORAGE_TYPE_CEPH
	case "gluster":
		return labv1.StorageType_STORAGE_TYPE_GLUSTER
	default:
		return labv1.StorageType_STORAGE_TYPE_DIR
	}
}

// storageStatusToProto converts string storage status to proto enum
func storageStatusToProto(status string) labv1.StorageStatus {
	switch status {
	case "active":
		return labv1.StorageStatus_STORAGE_STATUS_ACTIVE
	case "inactive":
		return labv1.StorageStatus_STORAGE_STATUS_INACTIVE
	case "maintenance":
		return labv1.StorageStatus_STORAGE_STATUS_MAINTENANCE
	case "error":
		return labv1.StorageStatus_STORAGE_STATUS_ERROR
	default:
		return labv1.StorageStatus_STORAGE_STATUS_ACTIVE
	}
}

// storagePoolToGitOpsResource converts a StoragePool proto to GitOpsResource
func storagePoolToGitOpsResource(configID string, pool *labv1.StoragePool) *model.GitOpsResource {
	return &model.GitOpsResource{
		ConfigID:      configID,
		Kind:          "StoragePool",
		Name:          pool.Name,
		Namespace:     "default",
		Status:        model.GitOpsStatusHealthy,
		StatusMessage: fmt.Sprintf("Storage pool %s is %s", pool.Name, pool.Status.String()),
		Spec: map[string]any{
			"name":        pool.Name,
			"type":        pool.Type.String(),
			"path":        pool.Path,
			"status":      pool.Status.String(),
			"options":     pool.Options,
			"enabled":     pool.Enabled,
			"description": pool.Description,
		},
	}
}

// getPoolIDFromResource extracts the pool ID from a GitOpsResource
func getPoolIDFromResource(resource *model.GitOpsResource) string {
	if resource == nil || resource.Spec == nil {
		return ""
	}
	if id, ok := resource.Spec["id"].(string); ok {
		return id
	}
	return ""
}
