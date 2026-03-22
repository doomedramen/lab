package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/go-git/go-billy/v5"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"gopkg.in/yaml.v3"
)

// GitOpsService manages GitOps reconciliation
type GitOpsService struct {
	gitopsRepo  repository.GitOpsRepository
	registry    *ReconcilerRegistry
}

// NewGitOpsService creates a new GitOps service
func NewGitOpsService(gitopsRepo repository.GitOpsRepository) *GitOpsService {
	service := &GitOpsService{
		gitopsRepo: gitopsRepo,
		registry:   NewReconcilerRegistry(),
	}

	// Register default reconcilers (will be wired with actual repos in main.go)
	// service.registry.Register(NewVMReconciler(vmRepo, vmService))
	// service.registry.Register(NewContainerReconciler(containerRepo, containerService))
	// service.registry.Register(NewNetworkReconciler(networkRepo, networkService))
	// service.registry.Register(NewStoragePoolReconciler(storageService))
	// service.registry.Register(NewStackReconciler(stackService))

	return service
}

// WithReconciler adds a reconciler to the service
func (s *GitOpsService) WithReconciler(reconciler Reconciler) {
	s.registry.Register(reconciler)
}

// GitRepository represents a fetched Git repository
type GitRepository struct {
	Repo       *git.Repository
	Head       *plumbing.Hash
	HeadString string
}

// FetchResult represents the result of fetching and parsing a Git repository
type FetchResult struct {
	CommitHash string
	CommitTime time.Time
	Message    string
	Manifests  []ParsedManifest
}

// ParsedManifest represents a parsed YAML manifest
type ParsedManifest struct {
	Path     string
	Content  string
	Hash     string
	Manifest model.GitOpsManifest
}

// FetchRepository fetches a Git repository and returns its contents
func (s *GitOpsService) FetchRepository(ctx context.Context, config *model.GitOpsConfig) (*FetchResult, error) {
	// Clone repository to memory
	cloneOpts := &git.CloneOptions{
		URL:           config.GitURL,
		ReferenceName: plumbing.NewBranchReferenceName(config.GitBranch),
		SingleBranch:  true,
		Depth:         1, // Shallow clone for efficiency
	}

	repo, err := git.CloneContext(ctx, memory.NewStorage(), nil, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get head commit
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository head: %w", err)
	}

	// Get commit details
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get commit object: %w", err)
	}

	// Get worktree filesystem
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Parse manifests from repository using go-git's filesystem
	manifests, err := s.parseManifestsFromGit(ctx, worktree.Filesystem, config.GitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse manifests: %w", err)
	}

	return &FetchResult{
		CommitHash: head.Hash().String(),
		CommitTime: commit.Author.When,
		Message:    commit.Message,
		Manifests:  manifests,
	}, nil
}

// parseManifestsFromGit recursively parses YAML manifests from a go-git filesystem
func (s *GitOpsService) parseManifestsFromGit(ctx context.Context, fsys billy.Filesystem, basePath string) ([]ParsedManifest, error) {
	var manifests []ParsedManifest

	if basePath == "" {
		basePath = "/"
	}

	// Read directory contents
	entries, err := fsys.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	for _, entry := range entries {
		path := basePath + "/" + entry.Name()
		
		if entry.IsDir() {
			// Recurse into subdirectories
			subManifests, err := s.parseManifestsFromGit(ctx, fsys, path)
			if err != nil {
				continue // Skip directories that can't be read
			}
			manifests = append(manifests, subManifests...)
			continue
		}

		// Only process YAML files
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".yaml") &&
			!strings.HasSuffix(strings.ToLower(entry.Name()), ".yml") {
			continue
		}

		// Read file content
		file, err := fsys.Open(path)
		if err != nil {
			continue // Skip files that can't be opened
		}

		content := make([]byte, entry.Size())
		_, err = file.Read(content)
		file.Close()
		if err != nil {
			continue
		}

		// Parse YAML content
		var manifest model.GitOpsManifest
		if err := yaml.Unmarshal(content, &manifest); err != nil {
			// Skip invalid YAML files
			continue
		}

		// Skip if no kind specified
		if manifest.Kind == "" {
			continue
		}

		// Calculate content hash
		hash := fmt.Sprintf("%x", sha256.Sum256(content))

		manifests = append(manifests, ParsedManifest{
			Path:     path,
			Content:  string(content),
			Hash:     hash,
			Manifest: manifest,
		})
	}

	return manifests, nil
}

// ReconcileConfig runs a reconciliation loop for a GitOps configuration
func (s *GitOpsService) ReconcileConfig(ctx context.Context, configID string) (*model.GitOpsSyncLog, error) {
	// Get configuration
	config, err := s.gitopsRepo.GetConfig(ctx, configID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitOps config: %w", err)
	}
	if config == nil {
		return nil, fmt.Errorf("GitOps config not found: %s", configID)
	}

	// Create sync log
	syncLog := &model.GitOpsSyncLog{
		ID:         generateID(),
		ConfigID:   config.ID,
		StartTime:  time.Now(),
		Status:     model.GitOpsStatusPending,
		Message:    "Starting reconciliation",
		CommitHash: config.LastSyncHash,
	}

	defer func() {
		syncLog.EndTime = time.Now()
		syncLog.Duration = time.Since(syncLog.StartTime)
		// Save sync log
		if err := s.gitopsRepo.CreateSyncLog(ctx, syncLog); err != nil {
			// Log error but don't fail
		}
		// Update config sync status
		if err := s.gitopsRepo.UpdateConfigSync(ctx, config.ID, syncLog.Status, syncLog.CommitHash, syncLog.Message); err != nil {
			// Log error but don't fail
		}
	}()

	// Fetch repository
	result, err := s.FetchRepository(ctx, config)
	if err != nil {
		syncLog.Status = model.GitOpsStatusFailed
		syncLog.Message = fmt.Sprintf("Failed to fetch repository: %v", err)
		return syncLog, err
	}

	syncLog.CommitHash = result.CommitHash
	syncLog.ResourcesScanned = len(result.Manifests)

	// Process each manifest
	for _, manifest := range result.Manifests {
		// Get existing resource or create new one
		resource, err := s.gitopsRepo.GetResourceByManifest(ctx, config.ID, manifest.Path)
		if err != nil {
			syncLog.ResourcesFailed++
			continue
		}

		// Get the appropriate reconciler for this resource kind
		kind := manifest.Manifest.Kind
		reconciler, hasReconciler := s.registry.GetReconciler(kind)
		
		if resource == nil {
			// Create new resource
			resource = &model.GitOpsResource{
				ID:            generateID(),
				ConfigID:      config.ID,
				Kind:          model.GitOpsKind(kind),
				Name:          manifest.Manifest.Metadata.Name,
				Namespace:     "default", // Default namespace
				ManifestPath:  manifest.Path,
				ManifestHash:  manifest.Hash,
				Spec:          manifest.Manifest.Spec,
				Status:        model.GitOpsStatusPending,
				StatusMessage: "New resource detected",
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			// Reconcile the resource if we have a reconciler
			if hasReconciler {
				reconcileResult, err := reconciler.Reconcile(ctx, resource)
				if err != nil || reconcileResult.Action == ReconcileActionFailed {
					resource.Status = model.GitOpsStatusFailed
					if err != nil {
						resource.StatusMessage = fmt.Sprintf("Reconciliation failed: %v", err)
					} else {
						resource.StatusMessage = reconcileResult.Message
					}
					syncLog.ResourcesFailed++
				} else {
					resource.Status = model.GitOpsStatusHealthy
					resource.StatusMessage = reconcileResult.Message
					switch reconcileResult.Action {
					case ReconcileActionCreated:
						syncLog.ResourcesCreated++
					case ReconcileActionUpdated:
						syncLog.ResourcesUpdated++
					default:
						syncLog.ResourcesCreated++ // Count as processed
					}
				}
			} else {
				// No reconciler - just track the resource
				resource.Status = model.GitOpsStatusOutOfSync
				resource.StatusMessage = "No reconciler available for this resource kind"
				syncLog.ResourcesCreated++
			}

			if err := s.gitopsRepo.CreateResource(ctx, resource); err != nil {
				syncLog.ResourcesFailed++
				continue
			}
		} else if resource.ManifestHash != manifest.Hash {
			// Update existing resource
			resource.ManifestHash = manifest.Hash
			resource.Spec = manifest.Manifest.Spec
			resource.UpdatedAt = time.Now()

			// Reconcile the resource if we have a reconciler
			if hasReconciler {
				reconcileResult, err := reconciler.Reconcile(ctx, resource)
				if err != nil || reconcileResult.Action == ReconcileActionFailed {
					resource.Status = model.GitOpsStatusFailed
					if err != nil {
						resource.StatusMessage = fmt.Sprintf("Reconciliation failed: %v", err)
					} else {
						resource.StatusMessage = reconcileResult.Message
					}
					syncLog.ResourcesFailed++
				} else {
					resource.Status = model.GitOpsStatusHealthy
					resource.StatusMessage = reconcileResult.Message
					if reconcileResult.Action == ReconcileActionUpdated {
						syncLog.ResourcesUpdated++
					} else {
						syncLog.ResourcesCreated++ // Count as processed
					}
				}
			} else {
				resource.Status = model.GitOpsStatusOutOfSync
				resource.StatusMessage = "Manifest changed, no reconciler available"
				syncLog.ResourcesCreated++
			}

			if err := s.gitopsRepo.UpdateResource(ctx, resource); err != nil {
				syncLog.ResourcesFailed++
				continue
			}
		} else {
			// Resource unchanged
			syncLog.ResourcesCreated++ // Count as processed
		}
	}

	// Update sync log status
	if syncLog.ResourcesFailed > 0 {
		syncLog.Status = model.GitOpsStatusFailed
		syncLog.Message = fmt.Sprintf("Reconciliation completed with %d failures", syncLog.ResourcesFailed)
	} else {
		syncLog.Status = model.GitOpsStatusHealthy
		syncLog.Message = fmt.Sprintf("Reconciliation successful: %d resources scanned", syncLog.ResourcesScanned)
	}

	return syncLog, nil
}

// StartReconciliationLoop starts the background reconciliation loop
func (s *GitOpsService) StartReconciliationLoop(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Get configs due for sync
			configs, err := s.gitopsRepo.GetConfigsDueForSync(ctx, time.Now())
			if err != nil {
				continue
			}

			// Reconcile each config
			for _, config := range configs {
				go func(cfg *model.GitOpsConfig) {
					_, err := s.ReconcileConfig(ctx, cfg.ID)
					if err != nil {
						// Error already logged in sync log
					}
				}(config)
			}
		}
	}
}

// generateID generates a unique ID for resources
func generateID() string {
	return fmt.Sprintf("gitops-%d", time.Now().UnixNano())
}
