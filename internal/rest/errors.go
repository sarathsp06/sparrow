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
	svcerrors.InvalidArgument:    400,
	svcerrors.NotFound:           404,
	svcerrors.AlreadyExists:      409,
	svcerrors.FailedPrecondition: 409,
	svcerrors.ResourceExhausted:  429,
	svcerrors.Unavailable:        503,
	svcerrors.DeadlineExceeded:   504,
	svcerrors.Internal:           500,
}

// mapError translates a business/storage error into a Huma HTTP error.
// Classification (ServiceError passthrough, storage sentinels, Internal
// default) lives in pkg/errors; this only maps Status to an HTTP code.
func mapError(ctx context.Context, err error, fallbackMsg string) error {
	if err == nil {
		return nil
	}

	svcErr := svcerrors.Classify(err, fallbackMsg)
	if svcErr.Status == svcerrors.Internal {
		slog.ErrorContext(ctx, "internal error", "fallback_msg", fallbackMsg, "error", err)
	}
	status, ok := categoryToHTTP[svcErr.Status]
	if !ok {
		status = 500
	}
	return huma.NewError(status, svcErr.ClientMessage())
}
