package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

var (
	ErrStackNotFound   = errors.New("stack not found")
	ErrStacksDisabled  = errors.New("stacks feature is disabled: stacks_dir not configured")
	ErrInvalidName     = errors.New("stack name is invalid: use only letters, numbers, hyphens, and underscores")
)

var validNameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// StackRepository implements repository.StackRepository using the filesystem
// and the docker compose CLI.
type StackRepository struct {
	stacksDir string
}

// NewStackRepository creates a new Docker-backed stack repository.
// If stacksDir is empty all operations return ErrStacksDisabled.
func NewStackRepository(stacksDir string) *StackRepository {
	return &StackRepository{stacksDir: stacksDir}
}

func (r *StackRepository) enabled() error {
	if r.stacksDir == "" {
		return ErrStacksDisabled
	}
	return nil
}

func (r *StackRepository) stackDir(id string) string {
	return filepath.Join(r.stacksDir, id)
}

func (r *StackRepository) composePath(id string) string {
	return filepath.Join(r.stackDir(id), "docker-compose.yml")
}

func (r *StackRepository) envPath(id string) string {
	return filepath.Join(r.stackDir(id), ".env")
}

// composePsEntry mirrors one JSON line from `docker compose ps --format json`
type composePsEntry struct {
	ID      string `json:"ID"`
	Name    string `json:"Name"`
	Image   string `json:"Image"`
	Status  string `json:"Status"`
	State   string `json:"State"`
	Ports   string `json:"Ports"`
	Service string `json:"Service"`
}

// runPS executes `docker compose ps --format json` and returns parsed entries.
func (r *StackRepository) runPS(id string) ([]composePsEntry, error) {
	cmd := exec.Command("docker", "compose", "-f", r.composePath(id), "ps", "--format", "json")
	out, err := cmd.Output()
	if err != nil {
		// If compose file doesn't exist or no containers, return empty slice
		return nil, nil
	}

	var entries []composePsEntry
	// Output is one JSON object per line (not a JSON array)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e composePsEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			slog.Warn("docker compose ps: failed to parse line", "line", line, "err", err)
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func deriveStatus(containers []model.DockerContainer) model.StackStatus {
	if len(containers) == 0 {
		return model.StackStatusStopped
	}
	running := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}
	switch {
	case running == len(containers):
		return model.StackStatusRunning
	case running == 0:
		return model.StackStatusStopped
	default:
		return model.StackStatusPartiallyRunning
	}
}

func entriesToContainers(entries []composePsEntry) []model.DockerContainer {
	containers := make([]model.DockerContainer, 0, len(entries))
	for _, e := range entries {
		var ports []string
		if e.Ports != "" {
			ports = strings.Split(e.Ports, ",")
			for i := range ports {
				ports[i] = strings.TrimSpace(ports[i])
			}
		}
		containers = append(containers, model.DockerContainer{
			ServiceName:   e.Service,
			ContainerName: e.Name,
			ContainerID:   e.ID,
			Image:         e.Image,
			Status:        e.Status,
			State:         e.State,
			Ports:         ports,
		})
	}
	return containers
}

func (r *StackRepository) readStack(id string) (*model.DockerStack, error) {
	dir := r.stackDir(id)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, ErrStackNotFound
	}

	composeBytes, err := os.ReadFile(r.composePath(id))
	if err != nil {
		composeBytes = []byte{}
	}

	envBytes, _ := os.ReadFile(r.envPath(id))

	entries, _ := r.runPS(id)
	containers := entriesToContainers(entries)

	return &model.DockerStack{
		ID:         id,
		Name:       id,
		Compose:    string(composeBytes),
		Env:        string(envBytes),
		Status:     deriveStatus(containers),
		Containers: containers,
		CreatedAt:  info.ModTime(),
	}, nil
}

// GetAll lists all stack directories and returns a DockerStack for each.
func (r *StackRepository) GetAll(_ context.Context) ([]*model.DockerStack, error) {
	if err := r.enabled(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(r.stacksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading stacks dir: %w", err)
	}

	var stacks []*model.DockerStack
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		st, err := r.readStack(e.Name())
		if err != nil {
			slog.Warn("docker stack: skipping directory", "name", e.Name(), "err", err)
			continue
		}
		stacks = append(stacks, st)
	}
	return stacks, nil
}

// GetByID returns the stack with the given id (folder name).
func (r *StackRepository) GetByID(_ context.Context, id string) (*model.DockerStack, error) {
	if err := r.enabled(); err != nil {
		return nil, err
	}
	return r.readStack(id)
}

// Create writes the compose/env files and returns the new stack.
func (r *StackRepository) Create(_ context.Context, req *model.StackCreateRequest) (*model.DockerStack, error) {
	if err := r.enabled(); err != nil {
		return nil, err
	}
	if !validNameRe.MatchString(req.Name) {
		return nil, ErrInvalidName
	}

	dir := r.stackDir(req.Name)
	if _, err := os.Stat(dir); err == nil {
		return nil, fmt.Errorf("stack %q already exists", req.Name)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating stack directory: %w", err)
	}

	if err := os.WriteFile(r.composePath(req.Name), []byte(req.Compose), 0644); err != nil {
		return nil, fmt.Errorf("writing compose file: %w", err)
	}

	if req.Env != "" {
		if err := os.WriteFile(r.envPath(req.Name), []byte(req.Env), 0644); err != nil {
			return nil, fmt.Errorf("writing env file: %w", err)
		}
	}

	return &model.DockerStack{
		ID:        req.Name,
		Name:      req.Name,
		Compose:   req.Compose,
		Env:       req.Env,
		Status:    model.StackStatusStopped,
		CreatedAt: time.Now(),
	}, nil
}

// Update overwrites the compose and/or env files for an existing stack.
func (r *StackRepository) Update(_ context.Context, id string, req *model.StackUpdateRequest) (*model.DockerStack, error) {
	if err := r.enabled(); err != nil {
		return nil, err
	}
	if _, err := os.Stat(r.stackDir(id)); err != nil {
		return nil, ErrStackNotFound
	}

	if err := os.WriteFile(r.composePath(id), []byte(req.Compose), 0644); err != nil {
		return nil, fmt.Errorf("writing compose file: %w", err)
	}

	if err := os.WriteFile(r.envPath(id), []byte(req.Env), 0644); err != nil {
		return nil, fmt.Errorf("writing env file: %w", err)
	}

	return r.readStack(id)
}

// Delete runs docker compose down, then removes the stack directory.
func (r *StackRepository) Delete(_ context.Context, id string) error {
	if err := r.enabled(); err != nil {
		return err
	}
	if _, err := os.Stat(r.stackDir(id)); err != nil {
		return ErrStackNotFound
	}

	// Best-effort down before removal
	_ = r.Down(context.Background(), id)

	if err := os.RemoveAll(r.stackDir(id)); err != nil {
		return fmt.Errorf("removing stack directory: %w", err)
	}
	return nil
}

func (r *StackRepository) runCompose(id string, args ...string) error {
	if err := r.enabled(); err != nil {
		return err
	}
	fullArgs := append([]string{"compose", "-f", r.composePath(id)}, args...)
	cmd := exec.Command("docker", fullArgs...)
	cmd.Dir = r.stackDir(id)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return nil
}

// Start runs `docker compose up -d`.
func (r *StackRepository) Start(_ context.Context, id string) error {
	return r.runCompose(id, "up", "-d")
}

// Stop runs `docker compose stop`.
func (r *StackRepository) Stop(_ context.Context, id string) error {
	return r.runCompose(id, "stop")
}

// Restart runs `docker compose restart`.
func (r *StackRepository) Restart(_ context.Context, id string) error {
	return r.runCompose(id, "restart")
}

// UpdateImages pulls images then recreates containers.
func (r *StackRepository) UpdateImages(_ context.Context, id string) error {
	if err := r.runCompose(id, "pull"); err != nil {
		return err
	}
	return r.runCompose(id, "up", "-d")
}

// Down runs `docker compose down`.
func (r *StackRepository) Down(_ context.Context, id string) error {
	return r.runCompose(id, "down")
}
