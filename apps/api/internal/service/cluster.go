package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
)

// ClusterService provides business logic for cluster operations
type ClusterService struct {
	nodeRepo      repository.NodeRepository
	vmRepo        repository.VMRepository
	containerRepo repository.ContainerRepository
	stackRepo     repository.StackRepository
	metricRepo    *sqlite.MetricRepository
}

// NewClusterService creates a new cluster service
func NewClusterService(
	nodeRepo repository.NodeRepository,
	vmRepo repository.VMRepository,
	containerRepo repository.ContainerRepository,
	stackRepo repository.StackRepository,
	metricRepo *sqlite.MetricRepository,
) *ClusterService {
	return &ClusterService{
		nodeRepo:      nodeRepo,
		vmRepo:        vmRepo,
		containerRepo: containerRepo,
		stackRepo:     stackRepo,
		metricRepo:    metricRepo,
	}
}

// GetSummary returns aggregated cluster metrics
func (s *ClusterService) GetSummary(ctx context.Context) *model.ClusterSummary {
	nodes, _ := s.nodeRepo.GetAll(ctx)
	vms, _ := s.vmRepo.GetAll(ctx)
	containers, _ := s.containerRepo.GetAll(ctx)
	stacks, _ := s.stackRepo.GetAll(ctx)

	// Count online nodes
	onlineNodes := 0
	totalCores := 0
	totalMemory := 0.0
	usedMemory := 0.0
	totalDisk := 0.0
	usedDisk := 0.0
	cpuUsageSum := 0.0

	for _, node := range nodes {
		totalCores += node.CPU.Cores
		totalMemory += node.Memory.Total
		totalDisk += node.Disk.Total

		if node.Status == model.NodeStatusOnline {
			onlineNodes++
			usedMemory += node.Memory.Used
			usedDisk += node.Disk.Used
			cpuUsageSum += node.CPU.Used
		}
	}

	// Count running VMs
	runningVMs := 0
	for _, vm := range vms {
		if vm.Status == model.VMStatusRunning {
			runningVMs++
		}
	}

	// Count running containers
	runningContainers := 0
	for _, ctr := range containers {
		if ctr.Status == model.ContainerStatusRunning {
			runningContainers++
		}
	}

	// Count running stacks
	runningStacks := 0
	for _, stack := range stacks {
		if stack.Status == model.StackStatusRunning {
			runningStacks++
		}
	}

	// Calculate average CPU usage
	avgCpu := 0.0
	if onlineNodes > 0 {
		avgCpu = cpuUsageSum / float64(onlineNodes)
	}

	return &model.ClusterSummary{
		Nodes: model.EntityCounts{
			Total:   len(nodes),
			Running: onlineNodes,
		},
		VMs: model.EntityCounts{
			Total:   len(vms),
			Running: runningVMs,
		},
		Containers: model.EntityCounts{
			Total:   len(containers),
			Running: runningContainers,
		},
		Stacks: model.EntityCounts{
			Total:   len(stacks),
			Running: runningStacks,
		},
		CPU: model.CPUAggregate{
			Cores:    totalCores,
			AvgUsage: math.Round(avgCpu),
		},
		Memory: model.MemoryInfo{
			Used:  roundTo(usedMemory, 1),
			Total: roundTo(totalMemory, 0),
		},
		Disk: model.DiskInfo{
			Used:  roundTo(usedDisk, 1),
			Total: roundTo(totalDisk, 0),
		},
	}
}

// GetMetrics returns time series data for cluster metrics
func (s *ClusterService) GetMetrics(points int) *model.ClusterMetrics {
	if points <= 0 {
		points = 24
	}

	return &model.ClusterMetrics{
		CPUUsage:    generateTimeSeries(points, 20, 80, "stable"),
		MemoryUsage: generateTimeSeries(points, 40, 70, "up"),
		NetworkIn:   generateTimeSeries(points, 100, 500, "stable"),
		NetworkOut:  generateTimeSeries(points, 50, 400, "stable"),
	}
}

// generateTimeSeries creates time series data for cluster metrics
func generateTimeSeries(points int, min, max float64, trend string) []model.TimeSeriesDataPoint {
	data := make([]model.TimeSeriesDataPoint, points)
	current := min + (max-min)*0.5

	for i := 0; i < points; i++ {
		hour := formatHour(i)
		delta := (rand.Float64() - 0.5) * (max - min) * 0.15

		switch trend {
		case "up":
			current += math.Abs(delta) * 0.3
		case "down":
			current -= math.Abs(delta) * 0.3
		default:
			current += delta
		}

		current = math.Max(min, math.Min(max, current))
		data[i] = model.TimeSeriesDataPoint{
			Time:  hour,
			Value: roundTo(current, 1),
		}
	}

	return data
}

// roundTo rounds a float to the specified number of decimal places
func roundTo(val float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(val*multiplier) / multiplier
}

// formatHour formats an hour index to HH:00 string
func formatHour(hour int) string {
	h := hour % 24
	return fmt.Sprintf("%02d:00", h)
}

// QueryMetrics queries metrics with filters
func (s *ClusterService) QueryMetrics(ctx context.Context, req model.MetricQuery) ([]model.MetricPoint, error) {
	return s.metricRepo.Query(ctx, req)
}
