package webhooks

import (
	"fmt"
	"strings"
)

// ValidationError represents a validation error with structured details
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// Error implements the error interface
func (e ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}

	var messages []string
	for _, err := range e.Errors {
		messages = append(messages, err.Error())
	}

	return strings.Join(messages, "; ")
}

// HasErrors returns true if there are validation errors
func (e ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// Add adds a validation error
func (e *ValidationErrors) Add(field, message, code string) {
	e.Errors = append(e.Errors, ValidationError{
		Field:   field,
		Message: message,
		Code:    code,
	})
}

// Validation error codes
const (
	ErrorCodeRequired      = "REQUIRED"
	ErrorCodeInvalidFormat = "INVALID_FORMAT"
	ErrorCodeTooLong       = "TOO_LONG"
	ErrorCodeTooShort      = "TOO_SHORT"
	ErrorCodeInvalid       = "INVALID"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeDuplicate     = "DUPLICATE"
	ErrorCodeForbidden     = "FORBIDDEN"
)

// Common validation messages
const (
	MsgRequired           = "field is required"
	MsgInvalidNamespace   = "namespace must be a valid identifier (alphanumeric, hyphens, underscores)"
	MsgInvalidURL         = "must be a valid HTTP or HTTPS URL"
	MsgInvalidTimeout     = "timeout must be between 1 and 300 seconds"
	MsgInvalidEvent       = "event name must be a valid identifier"
	MsgTooManyEvents      = "maximum 50 events allowed per webhook"
	MsgHeadersTooLarge    = "headers total size must not exceed 8KB"
	MsgPayloadTooLarge    = "payload size must not exceed 1MB"
	MsgInvalidJSON        = "must be valid JSON"
	MsgNamespaceTooLong   = "namespace must not exceed 64 characters"
	MsgDescriptionTooLong = "description must not exceed 500 characters"
)
