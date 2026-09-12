// Package rest implements Sparrow's REST/OpenAPI interface using Huma v2 on
// top of the existing chi router. Huma generates the OpenAPI document from
// the Go operations/DTOs defined here; it is the canonical contract.
package rest

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
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
// Classification (ServiceError passthrough, storage sentinels, Internal
// default) lives in pkg/errors; this only maps Status to an HTTP code.
func mapError(ctx context.Context, err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}

	svcErr := svcerrors.Classify(err, fallbackMsg)
	if svcErr.Status == svcerrors.Internal || svcErr.Status == svcerrors.Unknown {
		slog.ErrorContext(ctx, "internal error", "fallback_msg", fallbackMsg, "error", err)
	}
	status, ok := categoryToHTTP[svcErr.Status]
	if !ok {
		status = 500
	}
	return huma.NewError(status, svcErr.ClientMessage())
}
