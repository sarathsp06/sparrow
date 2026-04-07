package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// ServiceError is a client-safe error that carries a gRPC status code.
// When the gRPC layer encounters a ServiceError (via errors.As), it returns
// the message directly to the client instead of hiding it behind a generic
// fallback. This replaces the fragile string-matching allowlist in toGRPCError.
//
// Usage in service/validation code:
//
//	return errors.InvalidInput("loopback addresses are not allowed")
//	return errors.FailedPrecondition("event 'order.created' is inactive")
//	return errors.Wrap(err, codes.InvalidArgument, "invalid URL")
type ServiceError struct {
	GRPCCode codes.Code
	Msg      string
	Cause    error // underlying error (for server-side logging); nil if not wrapping
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
	}
	return e.Msg
}

// Unwrap allows errors.Is/errors.As to inspect the underlying cause.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// ClientMessage returns the message that is safe to send to the client.
// This is always Msg, never the cause chain.
func (e *ServiceError) ClientMessage() string {
	return e.Msg
}

// InvalidInput creates a ServiceError with codes.InvalidArgument.
// Use for validation failures, malformed input, missing required fields.
func InvalidInput(msg string) *ServiceError {
	return &ServiceError{GRPCCode: codes.InvalidArgument, Msg: msg}
}

// InvalidInputf creates a ServiceError with codes.InvalidArgument and a formatted message.
func InvalidInputf(format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: codes.InvalidArgument, Msg: fmt.Sprintf(format, args...)}
}

// FailedPrecondition creates a ServiceError with codes.FailedPrecondition.
// Use when the operation cannot be performed in the current state
// (e.g., event is inactive, batch expired, delivery already succeeded).
func FailedPrecondition(msg string) *ServiceError {
	return &ServiceError{GRPCCode: codes.FailedPrecondition, Msg: msg}
}

// FailedPreconditionf creates a ServiceError with codes.FailedPrecondition and a formatted message.
func FailedPreconditionf(format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: codes.FailedPrecondition, Msg: fmt.Sprintf(format, args...)}
}

// NotFoundError creates a ServiceError with codes.NotFound.
func NotFoundError(msg string) *ServiceError {
	return &ServiceError{GRPCCode: codes.NotFound, Msg: msg}
}

// NotFoundErrorf creates a ServiceError with codes.NotFound and a formatted message.
func NotFoundErrorf(format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: codes.NotFound, Msg: fmt.Sprintf(format, args...)}
}

// Wrap creates a ServiceError that wraps an underlying cause.
// The msg is sent to the client; the cause is only logged server-side.
func Wrap(cause error, code codes.Code, msg string) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: msg, Cause: cause}
}

// Wrapf creates a ServiceError with a formatted message that wraps an underlying cause.
func Wrapf(cause error, code codes.Code, format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: fmt.Sprintf(format, args...), Cause: cause}
}
