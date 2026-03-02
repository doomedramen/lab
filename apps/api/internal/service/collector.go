package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/repository"
	"github.com/doomedramen/lab/apps/api/internal/repository/sqlite"
	"github.com/doomedramen/lab/apps/api/pkg/libvirtx"
	sqlitePkg "github.com/doomedramen/lab/apps/api/pkg/sqlite"
)

// CollectorConfig holds configuration for the metrics collector
type CollectorConfig struct {
	CollectionInterval time.Duration // How often to collect metrics (default: 10s)
	RetentionDays      int           // Days to retain metrics (default: 30)
	EventRetentionDays int           // Days to retain events (default: 90)
	Enabled            bool          // Whether collection is enabled
}

// DefaultCollectorConfig returns the default collector configuration
func DefaultCollectorConfig() CollectorConfig {
	// Allow override via environment variable (in seconds)
	intervalSec := 60 // default: 1 minute
	if envInterval := os.Getenv("METRICS_INTERVAL_SEC"); envInterval != "" {
		if parsed, err := strconv.Atoi(envInterval); err == nil && parsed > 0 {
			intervalSec = parsed
		}
	}

	// Allow override via environment variables (in days)
	metricsRetention := 30 // default: 30 days
	if envRetention := os.Getenv("METRICS_RETENTION_DAYS"); envRetention != "" {
		if parsed, err := strconv.Atoi(envRetention); err == nil && parsed > 0 {
			metricsRetention = parsed
		}
	}

	eventsRetention := 90 // default: 90 days
	if envRetention := os.Getenv("EVENTS_RETENTION_DAYS"); envRetention != "" {
		if parsed, err := strconv.Atoi(envRetention); err == nil && parsed > 0 {
			eventsRetention = parsed
		}
	}

	return CollectorConfig{
		CollectionInterval: time.Duration(intervalSec) * time.Second,
		RetentionDays:      metricsRetention,
		EventRetentionDays: eventsRetention,
		Enabled:            true,
	}
}

// networkSnapshot holds the previous cumulative network reading for a node,
// used to compute per-interval rates rather than storing cumulative totals.
type networkSnapshot struct {
	in float64
	out float64
	ts  int64
}

// Collector collects and aggregates VM logs and metrics
type Collector struct {
	config          CollectorConfig
	client          *libvirtx.Client
	metricRepo      *sqlite.MetricRepository
	eventRepo       *sqlite.EventRepository
	vmSvc           *VMService
	nodeRepo        repository.NodeRepository
	vmRepo          repository.VMRepository
	containerRepo   repository.ContainerRepository
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	lastCollectTime time.Time
	collectionCount int64
	networkPrev     map[string]networkSnapshot // keyed by node ID
	vmStatePrev     map[int]model.VMStatus     // keyed by VM ID - track state for change detection
}

// NewCollector creates a new metrics collector
func NewCollector(
	config CollectorConfig,
	client *libvirtx.Client,
	metricRepo *sqlite.MetricRepository,
	eventRepo *sqlite.EventRepository,
	vmSvc *VMService,
	nodeRepo repository.NodeRepository,
	vmRepo repository.VMRepository,
	containerRepo repository.ContainerRepository,
) *Collector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Collector{
		config:        config,
		client:        client,
		metricRepo:    metricRepo,
		eventRepo:     eventRepo,
		vmSvc:         vmSvc,
		nodeRepo:      nodeRepo,
		vmRepo:        vmRepo,
		containerRepo: containerRepo,
		ctx:           ctx,
		cancel:        cancel,
		networkPrev:   make(map[string]networkSnapshot),
		vmStatePrev:   make(map[int]model.VMStatus),
	}
}

// Start begins the background collection loop
func (c *Collector) Start() {
	if !c.config.Enabled {
		log.Println("[collector] disabled, skipping")
		return
	}

	// One-time cleanup of spam logs from before the fix
	if err := c.CleanupCollectorLogs(); err != nil {
		log.Printf("[collector] cleanup failed: %v", err)
	}

	// Initialize tracked VM states from current state
	// This prevents false "VM started" logs when the API restarts
	c.initializeVMStates()

	c.wg.Add(2)
	go c.collectLoop()
	go c.cleanupLoop()
	log.Printf("[collector] started with %v interval", c.config.CollectionInterval)
}

// initializeVMStates loads the current state of all VMs
// so we don't log false state transitions on startup
func (c *Collector) initializeVMStates() {
	vms, err := c.vmRepo.GetAll(context.Background())
	if err != nil {
		log.Printf("[collector] failed to initialize VM states: %v", err)
		return
	}

	for _, vm := range vms {
		c.vmStatePrev[vm.VMID] = vm.Status
	}
	log.Printf("[collector] initialized states for %d VMs", len(vms))
}

// Stop gracefully stops the collector
func (c *Collector) Stop() {
	log.Println("[collector] stopping...")
	c.cancel()
	c.wg.Wait()
	log.Println("[collector] stopped")
}

// collectLoop runs the collection logic at regular intervals
func (c *Collector) collectLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	c.collectOnce()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.collectOnce()
		}
	}
}

// cleanupLoop runs retention cleanup daily
func (c *Collector) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.runCleanup()
		}
	}
}

// collectOnce performs a single collection cycle
func (c *Collector) collectOnce() {
	startTime := time.Now()
	var metrics []*sqlitePkg.Metric
	ts := time.Now().Unix()

	// Collect node metrics
	nodes, _ := c.nodeRepo.GetAll(context.Background())
	for _, node := range nodes {
		// CPU
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       node.ID,
			ResourceType: "cpu",
			ResourceID:   nil, // Host-level metric
			Value:        node.CPU.Used,
			Unit:         "%",
		})

		// Memory
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       node.ID,
			ResourceType: "memory",
			ResourceID:   nil,
			Value:        node.Memory.Used,
			Unit:         "GB",
		})

		// Disk
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       node.ID,
			ResourceType: "disk",
			ResourceID:   nil,
			Value:        node.Disk.Used,
			Unit:         "GB",
		})

		// Network: compute rate (MiB/s) from delta between consecutive readings.
		// GetNetworkStats returns cumulative MiB since boot; subtract previous
		// reading and divide by elapsed seconds to get instantaneous rate.
		if prev, ok := c.networkPrev[node.ID]; ok {
			timeDelta := float64(ts - prev.ts)
			if timeDelta > 0 {
				inDelta := node.NetworkIn - prev.in
				outDelta := node.NetworkOut - prev.out
				// Guard against counter resets (e.g. node reboot)
				if inDelta < 0 {
					inDelta = 0
				}
				if outDelta < 0 {
					outDelta = 0
				}
				metrics = append(metrics, &sqlitePkg.Metric{
					Timestamp:    ts,
					NodeID:       node.ID,
					ResourceType: "network_in",
					ResourceID:   nil,
					Value:        inDelta / timeDelta,
					Unit:         "MiB/s",
				})
				metrics = append(metrics, &sqlitePkg.Metric{
					Timestamp:    ts,
					NodeID:       node.ID,
					ResourceType: "network_out",
					ResourceID:   nil,
					Value:        outDelta / timeDelta,
					Unit:         "MiB/s",
				})
			}
		}
		c.networkPrev[node.ID] = networkSnapshot{in: node.NetworkIn, out: node.NetworkOut, ts: ts}

	}

	// Collect VM metrics
	vms, _ := c.vmRepo.GetAll(context.Background())
	for _, vm := range vms {
		vmIDStr := fmt.Sprintf("%d", vm.VMID)  // Convert QEMU VMID to string

		// VM CPU
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       vm.Node,
			ResourceType: "cpu",
			ResourceID:   &vmIDStr,
			Value:        vm.CPU.Used,
			Unit:         "%",
		})

		// VM Memory
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       vm.Node,
			ResourceType: "memory",
			ResourceID:   &vmIDStr,
			Value:        float64(vm.Memory.Used),
			Unit:         "GB",
		})

		// VM Disk
		metrics = append(metrics, &sqlitePkg.Metric{
			Timestamp:    ts,
			NodeID:       vm.Node,
			ResourceType: "disk",
			ResourceID:   &vmIDStr,
			Value:        vm.Disk.Used,
			Unit:         "GB",
		})

		// Ingest VM logs
		if c.vmSvc != nil {
			c.ingestVMLogs(vm)
		}
	}

	// Collect container metrics (if container repo is available)
	if c.containerRepo != nil {
		containers, _ := c.containerRepo.GetAll(context.Background())
		for _, ct := range containers {
			ctIDStr := fmt.Sprintf("%d", ct.CTID)  // Convert LXC CTID to string

			// Container CPU
			metrics = append(metrics, &sqlitePkg.Metric{
				Timestamp:    ts,
				NodeID:       ct.Node,
				ResourceType: "cpu",
				ResourceID:   &ctIDStr,
				Value:        ct.CPU.Used,
				Unit:         "%",
			})

			// Container Memory
			metrics = append(metrics, &sqlitePkg.Metric{
				Timestamp:    ts,
				NodeID:       ct.Node,
				ResourceType: "memory",
				ResourceID:   &ctIDStr,
				Value:        float64(ct.Memory.Used),
				Unit:         "GB",
			})

			// Container Disk
			metrics = append(metrics, &sqlitePkg.Metric{
				Timestamp:    ts,
				NodeID:       ct.Node,
				ResourceType: "disk",
				ResourceID:   &ctIDStr,
				Value:        ct.Disk.Used,
				Unit:         "GB",
			})
		}
	}

	// Batch insert metrics
	if len(metrics) > 0 {
		ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		defer cancel()

		if err := c.metricRepo.RecordBatch(ctx, metrics); err != nil {
			log.Printf("[collector] failed to save metrics: %v", err)
		} else {
			c.lastCollectTime = startTime
			c.collectionCount++
			elapsed := time.Since(startTime)
			log.Printf("[collector] saved %d metrics in %v", len(metrics), elapsed)
		}
	}
}

// runCleanup removes old metrics and events
func (c *Collector) runCleanup() {
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Delete old metrics
	deletedMetrics, err := c.metricRepo.DeleteOld(ctx, c.config.RetentionDays)
	if err != nil {
		log.Printf("[collector] failed to cleanup old metrics: %v", err)
	} else if deletedMetrics > 0 {
		log.Printf("[collector] deleted %d old metrics", deletedMetrics)
	}

	// Delete old events
	deletedEvents, err := c.eventRepo.DeleteOld(ctx, c.config.EventRetentionDays)
	if err != nil {
		log.Printf("[collector] failed to cleanup old events: %v", err)
	} else if deletedEvents > 0 {
		log.Printf("[collector] deleted %d old events", deletedEvents)
	}
}

// LastCollectTime returns the time of the last successful collection
func (c *Collector) LastCollectTime() time.Time {
	return c.lastCollectTime
}

// CollectionCount returns the total number of collection cycles
func (c *Collector) CollectionCount() int64 {
	return c.collectionCount
}

// ingestVMLogs ingests logs from VM state into the database
// Only logs state CHANGES (transitions), not the current state
func (c *Collector) ingestVMLogs(vm *model.VM) {
	if c.vmSvc == nil {
		return
	}

	// Get previous state for this VM
	prevState := c.vmStatePrev[vm.VMID]

	// Only log if state changed
	if vm.Status != prevState {
		// Log the state transition
		var message string
		switch vm.Status {
		case model.VMStatusRunning:
			message = "VM started"
		case model.VMStatusStopped:
			message = "VM stopped"
		case model.VMStatusPaused:
			message = "VM paused"
		case model.VMStatusSuspended:
			message = "VM suspended"
		default:
			message = fmt.Sprintf("VM state changed to %s", vm.Status)
		}

		logs := []*VMLogEntry{
			{
				Level:     "INFO",
				Source:    "collector",
				Message:   message,
			},
		}

		if err := c.vmSvc.IngestVMLogs(vm.VMID, logs); err != nil {
			log.Printf("[collector] failed to ingest VM logs for vm %d: %v", vm.VMID, err)
		}

		// Update tracked state
		c.vmStatePrev[vm.VMID] = vm.Status
	}
}

// CleanupCollectorLogs removes spam logs from the collector
// This is a one-time cleanup for logs created before the fix
func (c *Collector) CleanupCollectorLogs() error {
	if c.vmSvc == nil || c.vmSvc.logRepo == nil {
		return nil
	}

	ctx := context.Background()

	// Delete logs with messages that were part of the spam
	deleted, err := c.vmSvc.logRepo.DeleteByMessage(ctx, "VM is running")
	if err != nil {
		return err
	}
	if deleted > 0 {
		log.Printf("[collector] cleaned up %d 'VM is running' log entries", deleted)
	}

	deleted2, err := c.vmSvc.logRepo.DeleteByMessage(ctx, "VM is stopped")
	if err != nil {
		return err
	}
	if deleted2 > 0 {
		log.Printf("[collector] cleaned up %d 'VM is stopped' log entries", deleted2)
	}

	deleted3, err := c.vmSvc.logRepo.DeleteByMessage(ctx, "Resource usage snapshot")
	if err != nil {
		return err
	}
	if deleted3 > 0 {
		log.Printf("[collector] cleaned up %d 'Resource usage snapshot' log entries", deleted3)
	}

	// Delete logs with epoch timestamp (Created before the timestamp fix)
	// These have created_at = 0 or very old timestamps
	deleted4, err := c.vmSvc.logRepo.DeleteOldEpoch(ctx)
	if err != nil {
		return err
	}
	if deleted4 > 0 {
		log.Printf("[collector] cleaned up %d epoch timestamp log entries", deleted4)
	}

	return nil
}

// LogEvent records an event to the database
func (c *Collector) LogEvent(nodeID, eventType, severity, message string, metadata interface{}) error {
	event := &model.EventCreate{
		NodeID:     nodeID,
		EventType:  model.EventType(eventType),
		Severity:   model.EventSeverity(severity),
		Message:    message,
	}
	if metadata != nil {
		data, _ := json.Marshal(metadata)
		event.Metadata = data
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.eventRepo.Log(ctx, event)
	return err
}
