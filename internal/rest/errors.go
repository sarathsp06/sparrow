// Package rest implements Sparrow's REST/OpenAPI interface using Huma v2 on
// top of the existing chi router. Huma generates the OpenAPI document from
// the Go operations/DTOs defined here; it is the canonical contract.
package rest

import (
	"context"
	"errors"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// categoryToHTTP maps svcerrors.Status to HTTP status codes.
var categoryToHTTP = map[svcerrors.Status]int{
	svcerrors.OK:                 200,
	svcerrors.InvalidArgument:    400,
	svcerrors.Unauthenticated:    401,
	svcerrors.PermissionDenied:   403,
	svcerrors.NotFound:           404,
	svcerrors.AlreadyExists:      409,
	svcerrors.FailedPrecondition: 409,
	svcerrors.Aborted:            409,
	svcerrors.OutOfRange:         400,
	svcerrors.ResourceExhausted:  429,
	svcerrors.Unimplemented:      501,
	svcerrors.Unavailable:        503,
	svcerrors.DeadlineExceeded:   504,
	svcerrors.Canceled:           499,
	svcerrors.Internal:           500,
	svcerrors.Unknown:            500,
	svcerrors.DataLoss:           500,
}

// mapError translates a business/storage error into a Huma HTTP error.
//
// Resolution order:
//  1. svcerrors.ServiceError — service-layer errors explicitly marked
//     client-safe with a Status, translated via categoryToHTTP.
//  2. Storage sentinel errors (not-found, conflict, fk-violation, not-null).
//  3. Default — 500, with the real error logged but not exposed.
func mapError(ctx context.Context, err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}

	var svcErr *svcerrors.ServiceError
	if errors.As(err, &svcErr) {
		status, ok := categoryToHTTP[svcErr.Status]
		if !ok {
			status = 500
		}
		return huma.NewError(status, svcErr.ClientMessage())
	}

	if errors.Is(err, storage.ErrNotFound) {
		return huma.NewError(404, fallbackMsg+": not found")
	}
	if errors.Is(err, storage.ErrForeignKeyViolation) {
		return huma.NewError(409, fallbackMsg+": a referenced resource does not exist")
	}
	if errors.Is(err, storage.ErrAlreadyExists) {
		return huma.NewError(409, fallbackMsg+": resource already exists")
	}
	if errors.Is(err, storage.ErrNotNullViolation) {
		return huma.NewError(400, fallbackMsg+": a required field is missing")
	}

	slog.ErrorContext(ctx, "internal error", "fallback_msg", fallbackMsg, "error", err)
	return huma.NewError(500, fallbackMsg)
}
