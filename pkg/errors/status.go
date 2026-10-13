package errors

// Status classifies a ServiceError for client-safe reporting and REST
// status-code mapping (see internal/rest/errors.go). It mirrors the subset
// of gRPC's status codes Sparrow used before the REST/OpenAPI migration,
// kept as a local enum now that no gRPC transport remains. Not related to
// ErrorCategory (category.go), which classifies webhook delivery outcomes.
type Status int

const (
	// Internal is the zero value, so an unset Status maps to a 500.
	Internal Status = iota
	InvalidArgument
	DeadlineExceeded
	NotFound
	AlreadyExists
	ResourceExhausted
	FailedPrecondition
	Unavailable
)
