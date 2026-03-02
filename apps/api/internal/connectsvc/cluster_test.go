package connectsvc

import (
	"testing"

	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelEntityCountsToProto(t *testing.T) {
	ec := model.EntityCounts{Total: 10, Running: 7}
	p := modelEntityCountsToProto(ec)

	if p.Total != 10 {
		t.Errorf("Total = %d, want 10", p.Total)
	}
	if p.Running != 7 {
		t.Errorf("Running = %d, want 7", p.Running)
	}
}

func TestModelEntityCountsToProto_Zero(t *testing.T) {
	ec := model.EntityCounts{}
	p := modelEntityCountsToProto(ec)

	if p.Total != 0 {
		t.Errorf("Total = %d, want 0", p.Total)
	}
	if p.Running != 0 {
		t.Errorf("Running = %d, want 0", p.Running)
	}
}

func TestModelTimeSeriesSliceToProto(t *testing.T) {
	pts := []model.TimeSeriesDataPoint{
		{Time: "12:00", Value: 50.0},
		{Time: "13:00", Value: 75.5},
		{Time: "14:00", Value: 42.3},
	}

	result := modelTimeSeriesSliceToProto(pts)

	if len(result) != 3 {
		t.Fatalf("expected 3 points, got %d", len(result))
	}
	if result[0].Time != "12:00" {
		t.Errorf("result[0].Time = %q, want 12:00", result[0].Time)
	}
	if result[1].Value != 75.5 {
		t.Errorf("result[1].Value = %v, want 75.5", result[1].Value)
	}
}

func TestModelTimeSeriesSliceToProto_Empty(t *testing.T) {
	result := modelTimeSeriesSliceToProto(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 points, got %d", len(result))
	}
}
