package errors

import (
	"errors"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Classify converts any error into a client-safe *ServiceError.
// ServiceErrors pass through unchanged; storage sentinels get the matching
// Status and a client-safe message built from msg; anything else becomes
// Internal with msg as the client message (the cause stays wrapped).
func Classify(err error, msg string) *ServiceError {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr
	}

	switch {
	case errors.Is(err, storage.ErrNotFound):
		return Wrap(err, NotFound, msg+": not found")
	case errors.Is(err, storage.ErrForeignKeyViolation):
		return Wrap(err, FailedPrecondition, msg+": a referenced resource does not exist")
	case errors.Is(err, storage.ErrAlreadyExists):
		return Wrap(err, AlreadyExists, msg+": resource already exists")
	case errors.Is(err, storage.ErrNotNullViolation):
		return Wrap(err, InvalidArgument, msg+": a required field is missing")
	}

	return Wrap(err, Internal, msg)
}
