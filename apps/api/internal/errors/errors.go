// Package errors provides application-specific error types and utilities
// for consistent error handling throughout the API.
//
// Error Types:
//   - APIError: Base error type with code and metadata
//   - NotFoundError: Resource not found
//   - InvalidArgumentError: Invalid input parameters
//   - AlreadyExistsError: Resource already exists
//   - PermissionDeniedError: Insufficient permissions
//   - UnauthenticatedError: Authentication required
//   - InternalError: Internal server error
//   - UnavailableError: Service temporarily unavailable
//   - ConflictError: Resource conflict (e.g., VM running when stop requested)
//
// Usage:
//
//	// Create errors
//	err := errors.NewNotFoundError("vm", "100")
//	err := errors.NewInvalidArgumentError("memory", "must be greater than 0")
//	err := errors.NewInternalError(err, "failed to create VM")
//
//	// Check error types
//	if errors.IsNotFound(err) { ... }
//	if errors.IsInvalidArgument(err) { ... }
//
//	// Wrap errors with context
//	err := errors.Wrap(err, errors.CodeInternal, "failed to process request")
package errors

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ErrorCode represents application error codes
type ErrorCode string

const (
	// OK - No error
	OK ErrorCode = "OK"

	// InvalidArgument - Client specified an invalid argument
	InvalidArgument ErrorCode = "INVALID_ARGUMENT"

	// NotFound - Requested resource was not found
	NotFound ErrorCode = "NOT_FOUND"

	// AlreadyExists - Resource already exists
	AlreadyExists ErrorCode = "ALREADY_EXISTS"

	// PermissionDenied - Client does not have sufficient permissions
	PermissionDenied ErrorCode = "PERMISSION_DENIED"

	// Unauthenticated - Client is not authenticated
	Unauthenticated ErrorCode = "UNAUTHENTICATED"

	// Internal - Internal server error
	Internal ErrorCode = "INTERNAL"

	// Unavailable - Service is currently unavailable
	Unavailable ErrorCode = "UNAVAILABLE"

	// Conflict - Request conflicts with current state
	Conflict ErrorCode = "CONFLICT"

	// ResourceExhausted - Resource quota exceeded
	ResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"

	// Cancelled - Operation was cancelled
	Cancelled ErrorCode = "CANCELLED"

	// DeadlineExceeded - Deadline expired before operation completed
	DeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"

	// Unimplemented - Operation is not implemented
	Unimplemented ErrorCode = "UNIMPLEMENTED"

	// DataLoss - Unrecoverable data loss or corruption
	DataLoss ErrorCode = "DATA_LOSS"
)

// HTTPStatus returns the HTTP status code for an error code
func (c ErrorCode) HTTPStatus() int {
	switch c {
	case OK:
		return http.StatusOK
	case InvalidArgument:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case Unauthenticated:
		return http.StatusUnauthorized
	case Internal:
		return http.StatusInternalServerError
	case Unavailable:
		return http.StatusServiceUnavailable
	case Conflict:
		return http.StatusConflict
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case Cancelled:
		return http.StatusRequestTimeout
	case DeadlineExceeded:
		return http.StatusGatewayTimeout
	case Unimplemented:
		return http.StatusNotImplemented
	case DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// APIError is the base error type for all application errors
type APIError struct {
	Code       ErrorCode         `json:"code"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details,omitempty"`
	Cause      error             `json:"-"`
	StackTrace string            `json:"-"`
	Operation  string            `json:"-"` // Operation being performed
	Resource   string            `json:"-"` // Resource type (e.g., "vm", "user")
	ResourceID string            `json:"-"` // Resource identifier
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("%s: %s", e.Operation, e.Message)
	}
	if e.Resource != "" && e.ResourceID != "" {
		return fmt.Sprintf("%s %s: %s", e.Resource, e.ResourceID, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause (for errors.Is/As)
func (e *APIError) Unwrap() error {
	return e.Cause
}

// WithDetails adds metadata to the error
func (e *APIError) WithDetails(key, value string) *APIError {
	if e.Details == nil {
		e.Details = make(map[string]string)
	}
	e.Details[key] = value
	return e
}

// WithOperation adds operation context
func (e *APIError) WithOperation(op string) *APIError {
	e.Operation = op
	return e
}

// WithResource adds resource context
func (e *APIError) WithResource(resourceType, resourceID string) *APIError {
	e.Resource = resourceType
	e.ResourceID = resourceID
	return e
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) string {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var sb strings.Builder
	for {
		frame, more := frames.Next()
		// Skip internal error package frames
		if !strings.Contains(frame.File, "/internal/errors/") {
			sb.WriteString(fmt.Sprintf("\n\t%s:%d in %s", frame.File, frame.Line, frame.Function))
		}
		if !more {
			break
		}
	}
	return sb.String()
}

// New creates a new API error with the specified code and message
func New(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:       code,
		Message:    message,
		StackTrace: captureStackTrace(3),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code ErrorCode, message string) *APIError {
	if err == nil {
		return nil
	}
	
	return &APIError{
		Code:       code,
		Message:    message,
		Cause:      err,
		StackTrace: captureStackTrace(3),
	}
}

// Wrapf wraps an error with a formatted message
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *APIError {
	if err == nil {
		return nil
	}
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resourceType, resourceID string) *APIError {
	return &APIError{
		Code:       NotFound,
		Message:    fmt.Sprintf("%s not found", resourceType),
		Resource:   resourceType,
		ResourceID: resourceID,
		StackTrace: captureStackTrace(3),
	}
}

// NewNotFoundErrorf creates a not found error with a formatted message
func NewNotFoundErrorf(resourceType, resourceID string, format string, args ...interface{}) *APIError {
	err := NewNotFoundError(resourceType, resourceID)
	err.Message = fmt.Sprintf(format, args...)
	return err
}

// NewInvalidArgumentError creates an invalid argument error
func NewInvalidArgumentError(field, message string) *APIError {
	err := &APIError{
		Code:       InvalidArgument,
		Message:    fmt.Sprintf("invalid argument: %s", message),
		StackTrace: captureStackTrace(3),
	}
	if field != "" {
		err = err.WithDetails("field", field)
	}
	return err
}

// NewInvalidArgumentErrorf creates an invalid argument error with formatted message
func NewInvalidArgumentErrorf(field, format string, args ...interface{}) *APIError {
	err := NewInvalidArgumentError(field, fmt.Sprintf(format, args...))
	return err
}

// NewAlreadyExistsError creates an already exists error
func NewAlreadyExistsError(resourceType, resourceID string) *APIError {
	return &APIError{
		Code:       AlreadyExists,
		Message:    fmt.Sprintf("%s already exists", resourceType),
		Resource:   resourceType,
		ResourceID: resourceID,
		StackTrace: captureStackTrace(3),
	}
}

// NewPermissionDeniedError creates a permission denied error
func NewPermissionDeniedError(message string) *APIError {
	return &APIError{
		Code:       PermissionDenied,
		Message:    fmt.Sprintf("permission denied: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewUnauthenticatedError creates an unauthenticated error
func NewUnauthenticatedError(message string) *APIError {
	return &APIError{
		Code:       Unauthenticated,
		Message:    fmt.Sprintf("unauthenticated: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewInternalError creates an internal server error
func NewInternalError(cause error, message string) *APIError {
	return Wrap(cause, Internal, fmt.Sprintf("internal error: %s", message))
}

// NewInternalErrorf creates an internal server error with formatted message
func NewInternalErrorf(cause error, format string, args ...interface{}) *APIError {
	return Wrap(cause, Internal, fmt.Sprintf("internal error: "+format, args...))
}

// NewUnavailableError creates a service unavailable error
func NewUnavailableError(message string) *APIError {
	return &APIError{
		Code:       Unavailable,
		Message:    fmt.Sprintf("service unavailable: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewConflictError creates a conflict error
func NewConflictError(message string) *APIError {
	return &APIError{
		Code:       Conflict,
		Message:    fmt.Sprintf("conflict: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewResourceExhaustedError creates a resource exhausted error
func NewResourceExhaustedError(message string) *APIError {
	return &APIError{
		Code:       ResourceExhausted,
		Message:    fmt.Sprintf("resource exhausted: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewCancelledError creates a cancelled error
func NewCancelledError(message string) *APIError {
	return &APIError{
		Code:       Cancelled,
		Message:    fmt.Sprintf("cancelled: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewDeadlineExceededError creates a deadline exceeded error
func NewDeadlineExceededError(message string) *APIError {
	return &APIError{
		Code:       DeadlineExceeded,
		Message:    fmt.Sprintf("deadline exceeded: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// NewUnimplementedError creates an unimplemented error
func NewUnimplementedError(message string) *APIError {
	return &APIError{
		Code:       Unimplemented,
		Message:    fmt.Sprintf("unimplemented: %s", message),
		StackTrace: captureStackTrace(3),
	}
}

// IsNotFound returns true if err is a NotFound error
func IsNotFound(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == NotFound
}

// IsInvalidArgument returns true if err is an InvalidArgument error
func IsInvalidArgument(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == InvalidArgument
}

// IsAlreadyExists returns true if err is an AlreadyExists error
func IsAlreadyExists(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == AlreadyExists
}

// IsPermissionDenied returns true if err is a PermissionDenied error
func IsPermissionDenied(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == PermissionDenied
}

// IsUnauthenticated returns true if err is an Unauthenticated error
func IsUnauthenticated(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == Unauthenticated
}

// IsInternal returns true if err is an Internal error
func IsInternal(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == Internal
}

// IsUnavailable returns true if err is an Unavailable error
func IsUnavailable(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == Unavailable
}

// IsConflict returns true if err is a Conflict error
func IsConflict(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == Conflict
}

// IsResourceExhausted returns true if err is a ResourceExhausted error
func IsResourceExhausted(err error) bool {
	apiErr, ok := err.(*APIError)
	return ok && apiErr.Code == ResourceExhausted
}

// GetErrorCode returns the error code if err is an APIError
func GetErrorCode(err error) ErrorCode {
	apiErr, ok := err.(*APIError)
	if !ok {
		return Internal
	}
	return apiErr.Code
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	apiErr, ok := err.(*APIError)
	if !ok {
		return http.StatusInternalServerError
	}
	return apiErr.Code.HTTPStatus()
}
