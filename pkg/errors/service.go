package errors

import (
	"fmt"
)

// ServiceError is a client-safe error that carries a Status for REST
// status-code mapping.
type ServiceError struct {
	Status Status
	Msg    string
	Cause  error
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

func Error(status Status, msg string) *ServiceError {
	return &ServiceError{Status: status, Msg: msg}
}

func Errorf(status Status, format string, args ...any) *ServiceError {
	return &ServiceError{Status: status, Msg: fmt.Sprintf(format, args...)}
}

func Wrap(cause error, status Status, msg string) *ServiceError {
	return &ServiceError{Status: status, Msg: msg, Cause: cause}
}

func Wrapf(cause error, status Status, format string, args ...any) *ServiceError {
	return &ServiceError{Status: status, Msg: fmt.Sprintf(format, args...), Cause: cause}
}
