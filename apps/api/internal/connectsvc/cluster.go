package connectsvc

import (
	"context"

	"connectrpc.com/connect"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	labv1connect "github.com/doomedramen/lab/apps/api/gen/lab/v1/labv1connect"
	"github.com/doomedramen/lab/apps/api/internal/model"
	"github.com/doomedramen/lab/apps/api/internal/service"
)

// ClusterServiceServer implements labv1connect.ClusterServiceHandler.
type ClusterServiceServer struct {
	labv1connect.UnimplementedClusterServiceHandler
	svc *service.ClusterService
}

// NewClusterServiceServer creates a new ClusterServiceServer.
func NewClusterServiceServer(svc *service.ClusterService) *ClusterServiceServer {
	return &ClusterServiceServer{svc: svc}
}

// GetClusterSummary returns the cluster summary.
func (s *ClusterServiceServer) GetClusterSummary(
	ctx context.Context,
	_ *connect.Request[labv1.GetClusterSummaryRequest],
) (*connect.Response[labv1.GetClusterSummaryResponse], error) {
	summary := s.svc.GetSummary(ctx)
	return connect.NewResponse(&labv1.GetClusterSummaryResponse{
		Nodes:      modelEntityCountsToProto(summary.Nodes),
		Vms:        modelEntityCountsToProto(summary.VMs),
		Containers: modelEntityCountsToProto(summary.Containers),
		Stacks:     modelEntityCountsToProto(summary.Stacks),
		Cpu: &labv1.CpuAggregate{
			Cores:    int32(summary.CPU.Cores),
			AvgUsage: summary.CPU.AvgUsage,
		},
		Memory: modelMemoryInfoToProto(summary.Memory),
		Disk:   modelDiskInfoToProto(summary.Disk),
	}), nil
}

// GetClusterMetrics returns cluster time-series metrics.
func (s *ClusterServiceServer) GetClusterMetrics(
	_ context.Context,
	req *connect.Request[labv1.GetClusterMetricsRequest],
) (*connect.Response[labv1.GetClusterMetricsResponse], error) {
	points := int(req.Msg.Points)
	if points <= 0 {
		points = 24
	}
	metrics := s.svc.GetMetrics(points)
	return connect.NewResponse(&labv1.GetClusterMetricsResponse{
		CpuUsage:    modelTimeSeriesSliceToProto(metrics.CPUUsage),
		MemoryUsage: modelTimeSeriesSliceToProto(metrics.MemoryUsage),
		NetworkIn:   modelTimeSeriesSliceToProto(metrics.NetworkIn),
		NetworkOut:  modelTimeSeriesSliceToProto(metrics.NetworkOut),
	}), nil
}

// QueryMetrics returns metrics with filters.
func (s *ClusterServiceServer) QueryMetrics(
	ctx context.Context,
	req *connect.Request[labv1.QueryMetricsRequest],
) (*connect.Response[labv1.QueryMetricsResponse], error) {
	q := model.MetricQuery{
		NodeID:       req.Msg.NodeId,
		ResourceType: req.Msg.ResourceType,
		StartTime:    req.Msg.StartTime,
		EndTime:      req.Msg.EndTime,
		Aggregate:    req.Msg.Aggregate,
		GroupBy:      req.Msg.GroupBy,
		HostOnly:     req.Msg.HostOnly,
	}

	if req.Msg.ResourceId != nil {
		q.ResourceID = req.Msg.ResourceId
	}

	metrics, err := s.svc.QueryMetrics(ctx, q)
	if err != nil {
		return nil, err
	}

	protoMetrics := make([]*labv1.MetricPoint, len(metrics))
	for i, m := range metrics {
		protoMetrics[i] = &labv1.MetricPoint{
			Time:  m.Time,
			Value: m.Value,
		}
	}

	return connect.NewResponse(&labv1.QueryMetricsResponse{
		Metrics: protoMetrics,
		Count:   int32(len(metrics)),
	}), nil
}

// --- conversion helpers ---

func modelEntityCountsToProto(ec model.EntityCounts) *labv1.EntityCounts {
	return &labv1.EntityCounts{
		Total:   int32(ec.Total),
		Running: int32(ec.Running),
	}
}

func modelTimeSeriesSliceToProto(pts []model.TimeSeriesDataPoint) []*labv1.TimeSeriesPoint {
	out := make([]*labv1.TimeSeriesPoint, len(pts))
	for i, p := range pts {
		out[i] = &labv1.TimeSeriesPoint{Time: p.Time, Value: p.Value}
	}
	return out
}
