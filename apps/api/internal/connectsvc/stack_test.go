package connectsvc

import (
	"testing"
	"time"

	labv1 "github.com/doomedramen/lab/apps/api/gen/lab/v1"
	"github.com/doomedramen/lab/apps/api/internal/model"
)

func TestModelStackStatusToProto(t *testing.T) {
	cases := []struct {
		input    model.StackStatus
		expected labv1.StackStatus
	}{
		{model.StackStatusRunning, labv1.StackStatus_STACK_STATUS_RUNNING},
		{model.StackStatusPartiallyRunning, labv1.StackStatus_STACK_STATUS_PARTIALLY_RUNNING},
		{model.StackStatusStopped, labv1.StackStatus_STACK_STATUS_STOPPED},
		{"unknown", labv1.StackStatus_STACK_STATUS_UNSPECIFIED},
	}
	for _, tc := range cases {
		got := modelStackStatusToProto(tc.input)
		if got != tc.expected {
			t.Errorf("modelStackStatusToProto(%q) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

func TestModelDockerContainerToProto(t *testing.T) {
	c := model.DockerContainer{
		ServiceName:   "web",
		ContainerName: "myapp-web-1",
		ContainerID:   "abc123",
		Image:         "nginx:latest",
		Status:        "Up 2 hours",
		State:         "running",
		Ports:         []string{"0.0.0.0:8080->80/tcp"},
	}

	p := modelDockerContainerToProto(c)

	if p.ServiceName != c.ServiceName {
		t.Errorf("ServiceName: got %q, want %q", p.ServiceName, c.ServiceName)
	}
	if p.ContainerName != c.ContainerName {
		t.Errorf("ContainerName: got %q, want %q", p.ContainerName, c.ContainerName)
	}
	if p.ContainerId != c.ContainerID {
		t.Errorf("ContainerId: got %q, want %q", p.ContainerId, c.ContainerID)
	}
	if p.Image != c.Image {
		t.Errorf("Image: got %q, want %q", p.Image, c.Image)
	}
	if p.Status != c.Status {
		t.Errorf("Status: got %q, want %q", p.Status, c.Status)
	}
	if p.State != c.State {
		t.Errorf("State: got %q, want %q", p.State, c.State)
	}
	if len(p.Ports) != 1 || p.Ports[0] != c.Ports[0] {
		t.Errorf("Ports: got %v, want %v", p.Ports, c.Ports)
	}
}

func TestModelDockerContainerToProto_EmptyPorts(t *testing.T) {
	c := model.DockerContainer{ServiceName: "db", State: "running"}
	p := modelDockerContainerToProto(c)
	if len(p.Ports) != 0 {
		t.Errorf("expected empty ports, got %v", p.Ports)
	}
}

func TestModelStackToProto(t *testing.T) {
	created := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	st := &model.DockerStack{
		ID:      "mystack",
		Name:    "mystack",
		Compose: "services: {}",
		Env:     "FOO=bar",
		Status:  model.StackStatusRunning,
		Containers: []model.DockerContainer{
			{ServiceName: "web", State: "running", Image: "nginx"},
			{ServiceName: "db", State: "exited", Image: "postgres"},
		},
		CreatedAt: created,
	}

	p := modelStackToProto(st)

	if p.Id != "mystack" {
		t.Errorf("Id: got %q, want mystack", p.Id)
	}
	if p.Name != "mystack" {
		t.Errorf("Name: got %q, want mystack", p.Name)
	}
	if p.Compose != "services: {}" {
		t.Errorf("Compose: got %q", p.Compose)
	}
	if p.Env != "FOO=bar" {
		t.Errorf("Env: got %q", p.Env)
	}
	if p.Status != labv1.StackStatus_STACK_STATUS_RUNNING {
		t.Errorf("Status: got %v, want RUNNING", p.Status)
	}
	if len(p.Containers) != 2 {
		t.Errorf("Containers: got %d, want 2", len(p.Containers))
	}
	if p.Containers[0].ServiceName != "web" {
		t.Errorf("Containers[0].ServiceName: got %q, want web", p.Containers[0].ServiceName)
	}
	if p.CreatedAt != "2025-06-15T00:00:00Z" {
		t.Errorf("CreatedAt: got %q, want 2025-06-15T00:00:00Z", p.CreatedAt)
	}
}

func TestModelStackToProto_EmptyContainers(t *testing.T) {
	st := &model.DockerStack{
		ID:        "empty",
		Status:    model.StackStatusStopped,
		CreatedAt: time.Now(),
	}
	p := modelStackToProto(st)
	if len(p.Containers) != 0 {
		t.Errorf("expected 0 containers, got %d", len(p.Containers))
	}
	if p.Status != labv1.StackStatus_STACK_STATUS_STOPPED {
		t.Errorf("Status: got %v, want STOPPED", p.Status)
	}
}
