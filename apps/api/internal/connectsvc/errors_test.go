package connectsvc

import (
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/doomedramen/lab/apps/api/internal/service"
)

func TestServiceErrToConnect_NotFoundErrors(t *testing.T) {
	notFoundErrs := []error{
		service.ErrNodeNotFound,
		service.ErrVMNotFound,
		service.ErrContainerNotFound,
		service.ErrStackNotFound,
		service.ErrISONotFound,
	}

	for _, err := range notFoundErrs {
		connectErr := serviceErrToConnect(err)
		var ce *connect.Error
		if !errors.As(connectErr, &ce) {
			t.Errorf("expected connect.Error for %v", err)
			continue
		}
		if ce.Code() != connect.CodeNotFound {
			t.Errorf("serviceErrToConnect(%v): code = %v, want NotFound", err, ce.Code())
		}
	}
}

func TestServiceErrToConnect_PreconditionErrors(t *testing.T) {
	precondErrs := []error{
		service.ErrNodeOffline,
		service.ErrNodeInMaintenance,
		service.ErrVMInvalidState,
		service.ErrContainerInvalidState,
		service.ErrISOInUse,
		service.ErrVMAlreadyRunning,
		service.ErrVMAlreadyStopped,
		service.ErrContainerAlreadyRunning,
		service.ErrContainerAlreadyStopped,
	}

	for _, err := range precondErrs {
		connectErr := serviceErrToConnect(err)
		var ce *connect.Error
		if !errors.As(connectErr, &ce) {
			t.Errorf("expected connect.Error for %v", err)
			continue
		}
		if ce.Code() != connect.CodeFailedPrecondition {
			t.Errorf("serviceErrToConnect(%v): code = %v, want FailedPrecondition", err, ce.Code())
		}
	}
}

func TestServiceErrToConnect_UnknownError(t *testing.T) {
	err := errors.New("something unexpected")
	connectErr := serviceErrToConnect(err)
	var ce *connect.Error
	if !errors.As(connectErr, &ce) {
		t.Fatal("expected connect.Error")
	}
	if ce.Code() != connect.CodeInternal {
		t.Errorf("code = %v, want Internal", ce.Code())
	}
}
