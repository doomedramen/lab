package connectsvc

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/doomedramen/lab/apps/api/internal/service"
)

// serviceErrToConnect translates service-layer errors into Connect error codes.
func serviceErrToConnect(err error) error {
	switch {
	case errors.Is(err, service.ErrNodeNotFound),
		errors.Is(err, service.ErrVMNotFound),
		errors.Is(err, service.ErrContainerNotFound),
		errors.Is(err, service.ErrStackNotFound),
		errors.Is(err, service.ErrISONotFound):
		return connect.NewError(connect.CodeNotFound, err)

	case errors.Is(err, service.ErrNodeOffline),
		errors.Is(err, service.ErrNodeInMaintenance),
		errors.Is(err, service.ErrVMInvalidState),
		errors.Is(err, service.ErrContainerInvalidState),
		errors.Is(err, service.ErrISOInUse),
		errors.Is(err, service.ErrVMAlreadyRunning),
		errors.Is(err, service.ErrVMAlreadyStopped),
		errors.Is(err, service.ErrContainerAlreadyRunning),
		errors.Is(err, service.ErrContainerAlreadyStopped):
		return connect.NewError(connect.CodeFailedPrecondition, err)

	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
