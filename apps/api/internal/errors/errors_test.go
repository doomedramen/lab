package errors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestErrorCode_HTTPStatus(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want int
	}{
		{OK, http.StatusOK},
		{InvalidArgument, http.StatusBadRequest},
		{NotFound, http.StatusNotFound},
		{AlreadyExists, http.StatusConflict},
		{PermissionDenied, http.StatusForbidden},
		{Unauthenticated, http.StatusUnauthorized},
		{Internal, http.StatusInternalServerError},
		{Unavailable, http.StatusServiceUnavailable},
		{Conflict, http.StatusConflict},
		{ResourceExhausted, http.StatusTooManyRequests},
		{Cancelled, http.StatusRequestTimeout},
		{DeadlineExceeded, http.StatusGatewayTimeout},
		{Unimplemented, http.StatusNotImplemented},
		{DataLoss, http.StatusInternalServerError},
		{ErrorCode("UNKNOWN"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if got := tt.code.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "basic",
			err:  &APIError{Code: NotFound, Message: "resource not found"},
			want: "NOT_FOUND: resource not found",
		},
		{
			name: "with operation",
			err:  &APIError{Code: Internal, Message: "failed", Operation: "create VM"},
			want: "create VM: failed",
		},
		{
			name: "with resource",
			err:  &APIError{Code: NotFound, Message: "not found", Resource: "vm", ResourceID: "100"},
			want: "vm 100: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIError_WithDetails(t *testing.T) {
	err := New(InvalidArgument, "invalid input").
		WithDetails("field", "memory").
		WithDetails("value", "-1")

	if err.Details["field"] != "memory" {
		t.Errorf("WithDetails() field = %q, want %q", err.Details["field"], "memory")
	}
	if err.Details["value"] != "-1" {
		t.Errorf("WithDetails() value = %q, want %q", err.Details["value"], "-1")
	}
}

func TestAPIError_WithOperation(t *testing.T) {
	err := New(Internal, "failed").WithOperation("create VM")
	if err.Operation != "create VM" {
		t.Errorf("WithOperation() = %q, want %q", err.Operation, "create VM")
	}
}

func TestAPIError_WithResource(t *testing.T) {
	err := New(NotFound, "not found").WithResource("vm", "100")
	if err.Resource != "vm" {
		t.Errorf("WithResource() resource = %q, want %q", err.Resource, "vm")
	}
	if err.ResourceID != "100" {
		t.Errorf("WithResource() resourceID = %q, want %q", err.ResourceID, "100")
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("original error")
	err := Wrap(cause, Internal, "wrapped")

	if err.Unwrap() != cause {
		t.Error("Unwrap() did not return the cause")
	}

	// Test errors.Is
	if !errors.Is(err, cause) {
		t.Error("errors.Is() should return true for wrapped error")
	}
}

func TestNew(t *testing.T) {
	err := New(NotFound, "resource not found")

	if err.Code != NotFound {
		t.Errorf("Code = %v, want %v", err.Code, NotFound)
	}
	if err.Message != "resource not found" {
		t.Errorf("Message = %q, want %q", err.Message, "resource not found")
	}
	if err.Cause != nil {
		t.Error("Cause should be nil")
	}
}

func TestWrap(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		err := Wrap(nil, Internal, "message")
		if err != nil {
			t.Error("Wrap(nil) should return nil")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		cause := fmt.Errorf("original")
		err := Wrap(cause, Internal, "wrapped")

		if err.Code != Internal {
			t.Errorf("Code = %v, want %v", err.Code, Internal)
		}
		if err.Message != "wrapped" {
			t.Errorf("Message = %q, want %q", err.Message, "wrapped")
		}
		if err.Cause != cause {
			t.Error("Cause not preserved")
		}
	})

	t.Run("wrap APIError", func(t *testing.T) {
		original := New(NotFound, "not found")
		err := Wrap(original, Internal, "context added")

		if err.Code != NotFound {
			t.Errorf("Code = %v, want %v", err.Code, NotFound)
		}
		if err.Message != "context added" {
			t.Errorf("Message = %q, want %q", err.Message, "context added")
		}
	})
}

func TestWrapf(t *testing.T) {
	cause := fmt.Errorf("disk full")
	err := Wrapf(cause, Internal, "failed to %s", "write data")

	if err.Message != "failed to write data" {
		t.Errorf("Message = %q, want %q", err.Message, "failed to write data")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("vm", "100")

	if err.Code != NotFound {
		t.Errorf("Code = %v, want %v", err.Code, NotFound)
	}
	if err.Resource != "vm" {
		t.Errorf("Resource = %q, want %q", err.Resource, "vm")
	}
	if err.ResourceID != "100" {
		t.Errorf("ResourceID = %q, want %q", err.ResourceID, "100")
	}
}

func TestNewNotFoundErrorf(t *testing.T) {
	err := NewNotFoundErrorf("vm", "100", "VM %s not found in cluster", "100")

	if err.Message != "VM 100 not found in cluster" {
		t.Errorf("Message = %q, want %q", err.Message, "VM 100 not found in cluster")
	}
}

func TestNewInvalidArgumentError(t *testing.T) {
	err := NewInvalidArgumentError("memory", "must be greater than 0")

	if err.Code != InvalidArgument {
		t.Errorf("Code = %v, want %v", err.Code, InvalidArgument)
	}
	if err.Details["field"] != "memory" {
		t.Errorf("Details[field] = %q, want %q", err.Details["field"], "memory")
	}
}

func TestNewInvalidArgumentErrorf(t *testing.T) {
	err := NewInvalidArgumentErrorf("memory", "must be at least %d GB", 1)

	if err.Message != "invalid argument: must be at least 1 GB" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid argument: must be at least 1 GB")
	}
}

func TestNewAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("vm", "my-vm")

	if err.Code != AlreadyExists {
		t.Errorf("Code = %v, want %v", err.Code, AlreadyExists)
	}
	if err.Resource != "vm" {
		t.Errorf("Resource = %q, want %q", err.Resource, "vm")
	}
}

func TestNewPermissionDeniedError(t *testing.T) {
	err := NewPermissionDeniedError("admin role required")

	if err.Code != PermissionDenied {
		t.Errorf("Code = %v, want %v", err.Code, PermissionDenied)
	}
	if err.Message != "permission denied: admin role required" {
		t.Errorf("Message = %q, want %q", err.Message, "permission denied: admin role required")
	}
}

func TestNewUnauthenticatedError(t *testing.T) {
	err := NewUnauthenticatedError("token expired")

	if err.Code != Unauthenticated {
		t.Errorf("Code = %v, want %v", err.Code, Unauthenticated)
	}
}

func TestNewInternalError(t *testing.T) {
	cause := fmt.Errorf("database connection failed")
	err := NewInternalError(cause, "failed to save VM")

	if err.Code != Internal {
		t.Errorf("Code = %v, want %v", err.Code, Internal)
	}
	if err.Cause != cause {
		t.Error("Cause not preserved")
	}
}

func TestNewInternalErrorf(t *testing.T) {
	cause := fmt.Errorf("timeout")
	err := NewInternalErrorf(cause, "operation %s after %d seconds", "timed out", 30)

	if err.Message != "internal error: operation timed out after 30 seconds" {
		t.Errorf("Message = %q, want %q", err.Message, "internal error: operation timed out after 30 seconds")
	}
}

func TestNewUnavailableError(t *testing.T) {
	err := NewUnavailableError("maintenance in progress")

	if err.Code != Unavailable {
		t.Errorf("Code = %v, want %v", err.Code, Unavailable)
	}
}

func TestNewConflictError(t *testing.T) {
	err := NewConflictError("VM is already running")

	if err.Code != Conflict {
		t.Errorf("Code = %v, want %v", err.Code, Conflict)
	}
}

func TestNewResourceExhaustedError(t *testing.T) {
	err := NewResourceExhaustedError("rate limit exceeded")

	if err.Code != ResourceExhausted {
		t.Errorf("Code = %v, want %v", err.Code, ResourceExhausted)
	}
}

func TestNewCancelledError(t *testing.T) {
	err := NewCancelledError("operation cancelled by user")

	if err.Code != Cancelled {
		t.Errorf("Code = %v, want %v", err.Code, Cancelled)
	}
}

func TestNewDeadlineExceededError(t *testing.T) {
	err := NewDeadlineExceededError("operation timed out")

	if err.Code != DeadlineExceeded {
		t.Errorf("Code = %v, want %v", err.Code, DeadlineExceeded)
	}
}

func TestNewUnimplementedError(t *testing.T) {
	err := NewUnimplementedError("feature not yet implemented")

	if err.Code != Unimplemented {
		t.Errorf("Code = %v, want %v", err.Code, Unimplemented)
	}
}

func TestIsNotFound(t *testing.T) {
	err := NewNotFoundError("vm", "100")
	if !IsNotFound(err) {
		t.Error("IsNotFound() should return true")
	}

	standardErr := fmt.Errorf("not found")
	if IsNotFound(standardErr) {
		t.Error("IsNotFound() should return false for standard error")
	}
}

func TestIsInvalidArgument(t *testing.T) {
	err := NewInvalidArgumentError("field", "invalid")
	if !IsInvalidArgument(err) {
		t.Error("IsInvalidArgument() should return true")
	}
}

func TestIsAlreadyExists(t *testing.T) {
	err := NewAlreadyExistsError("vm", "my-vm")
	if !IsAlreadyExists(err) {
		t.Error("IsAlreadyExists() should return true")
	}
}

func TestIsPermissionDenied(t *testing.T) {
	err := NewPermissionDeniedError("access denied")
	if !IsPermissionDenied(err) {
		t.Error("IsPermissionDenied() should return true")
	}
}

func TestIsUnauthenticated(t *testing.T) {
	err := NewUnauthenticatedError("token expired")
	if !IsUnauthenticated(err) {
		t.Error("IsUnauthenticated() should return true")
	}
}

func TestIsInternal(t *testing.T) {
	err := NewInternalError(fmt.Errorf("cause"), "failed")
	if !IsInternal(err) {
		t.Error("IsInternal() should return true")
	}
}

func TestIsUnavailable(t *testing.T) {
	err := NewUnavailableError("down for maintenance")
	if !IsUnavailable(err) {
		t.Error("IsUnavailable() should return true")
	}
}

func TestIsConflict(t *testing.T) {
	err := NewConflictError("resource in use")
	if !IsConflict(err) {
		t.Error("IsConflict() should return true")
	}
}

func TestIsResourceExhausted(t *testing.T) {
	err := NewResourceExhaustedError("quota exceeded")
	if !IsResourceExhausted(err) {
		t.Error("IsResourceExhausted() should return true")
	}
}

func TestGetErrorCode(t *testing.T) {
	err := New(NotFound, "not found")
	if GetErrorCode(err) != NotFound {
		t.Errorf("GetErrorCode() = %v, want %v", GetErrorCode(err), NotFound)
	}

	standardErr := fmt.Errorf("standard error")
	if GetErrorCode(standardErr) != Internal {
		t.Error("GetErrorCode() should return Internal for non-API errors")
	}
}

func TestGetHTTPStatus(t *testing.T) {
	err := New(NotFound, "not found")
	if GetHTTPStatus(err) != http.StatusNotFound {
		t.Errorf("GetHTTPStatus() = %d, want %d", GetHTTPStatus(err), http.StatusNotFound)
	}

	standardErr := fmt.Errorf("standard error")
	if GetHTTPStatus(standardErr) != http.StatusInternalServerError {
		t.Error("GetHTTPStatus() should return 500 for non-API errors")
	}
}

func TestStackTrace(t *testing.T) {
	err := New(Internal, "test error")
	
	if err.StackTrace == "" {
		t.Error("StackTrace should not be empty")
	}
	
	// Verify stack trace contains file information
	if len(err.StackTrace) < 10 {
		t.Error("StackTrace should contain meaningful information")
	}
}

func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	cause := fmt.Errorf("root cause")
	err1 := Wrap(cause, Internal, "database error")
	err2 := Wrap(err1, Internal, "failed to save VM")
	err3 := Wrap(err2, NotFound, "VM operation failed")

	// Verify the chain
	if !errors.Is(err3, cause) {
		t.Error("Error chain should preserve root cause")
	}

	// Verify the top-level error code
	if err3.Code != NotFound {
		t.Errorf("Code = %v, want %v", err3.Code, NotFound)
	}
}
