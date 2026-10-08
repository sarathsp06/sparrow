# Documenting the REST API

For contributors. The [API reference](https://sarathsp06.github.io/sparrow/reference/api/)
on the docs site is rendered from `api/openapi.yaml`, and that file is
**generated**: [Huma](https://huma.rocks) builds the OpenAPI 3.1 spec from the
handler structs in `internal/rest`, and `cmd/openapi-export` writes it out.
Never edit `api/openapi.{yaml,json}` by hand. Change the Go code, then run:

```bash
make generate   # exports the spec, regenerates the Python client, runs go generate
```

`internal/rest/openapi_drift_test.go` fails CI when the committed spec no
longer matches the code.

## Where each part of the reference comes from

| In the reference | In the Go code |
|------------------|----------------|
| Operation summary and description | `Summary` and `Description` on `huma.Operation` |
| Operation ID (also the generated client's method name) | `OperationID`, camelCase and verb-first: `registerWebhook`, `listDeliveries` |
| Grouping | `Tags` |
| Documented error responses | `Errors: []int{400, 404}` |
| Success status | `DefaultStatus` (200 when unset) |
| Field description | `doc:"..."` struct tag |
| Required marker | `required:"true"` (body fields), or a non-`omitempty` path parameter |
| Default value | `default:"..."` |
| Allowed values | `enum:"fail,fallback"` (a trailing comma also allows the empty string) |
| Ranges and lengths | `minimum:"1"`, `maximum:"1000"`, `maxLength:"2000"` |

```go
type listEventTypesInput struct {
	ActiveOnly bool  `query:"active_only" default:"false" doc:"Only return active event types."`
	Limit      int32 `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return."`
}
```

## Defaults that live in the service

A `default` tag only applies when the field is a value type Huma can fill in.
Optional fields that are pointers, so that "not sent" differs from "zero"
(`max_retries`, `verify_ssl`), get their default in the service layer instead.
State it in the `doc` text, and keep it in sync with the code:

```go
MaxRetries *int `json:"max_retries,omitempty" doc:"Maximum delivery attempts before a delivery is marked failed. 0 means no retries. Between 0 and 10; defaults to 3."`
```

## Writing descriptions

- Say what the field does and what a caller should send, in a sentence or two.
- Name the unit (`seconds`, `bytes`) and the format (`RFC3339`).
- For enums, say what each value does.
- Mention the error a caller gets when a constraint is broken, if it isn't
  obvious from the tags.
