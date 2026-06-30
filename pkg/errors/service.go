package errors

import (
	"fmt"

	"google.golang.org/grpc/codes"
)

// ServiceError is a client-safe error that carries a gRPC status code.
type ServiceError struct {
	GRPCCode codes.Code
	Msg      string
	Cause    error
}

func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.Cause)
	}
	return e.Msg
}

func (e *ServiceError) Unwrap() error {
	return e.Cause
}

func (e *ServiceError) ClientMessage() string {
	return e.Msg
}

func Error(code codes.Code, msg string) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: msg}
}

func Errorf(code codes.Code, format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: fmt.Sprintf(format, args...)}
}

func Wrap(cause error, code codes.Code, msg string) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: msg, Cause: cause}
}

func Wrapf(cause error, code codes.Code, format string, args ...any) *ServiceError {
	return &ServiceError{GRPCCode: code, Msg: fmt.Sprintf(format, args...), Cause: cause}
}
