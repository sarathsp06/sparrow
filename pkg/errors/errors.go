package errors

import (
	"fmt"
	"io"
	"strings"

	libErr "errors"

	"github.com/pkg/errors"
)

var (
	ErrorRetriable = errors.New("retryable error")
)

// Is checks if the given error is the same as the target error
// It is a wrapper around errors.Is
func Is(err, target error) bool {
	return libErr.Is(err, target)
}

// IsRetryable checks if the given error is a retryable error
func IsRetryable(err error) bool {
	return Is(err, ErrorRetriable)
}

// NewRetryableError wraps the given error with a retryable error
func NewRetryableError(err error) error {
	return libErr.Join(err, ErrorRetriable)
}

// Error indicates an error with more features like metedata
type Error struct {
	error
	metaData map[string]string
}

// New returns an error that formats as the given text.
// Each call to New returns a distinct error value even if the text is identical.
func New(message string) *Error {
	return &Error{error: libErr.New(message), metaData: make(map[string]string)}
}

// WithField adds a field to the error metadata and returns a new error
func (err Error) WithField(field, value string) *Error {
	if err.error == nil {
		return nil
	}
	// make sure to copy the metadata from original error if any
	m := err.Metadata()
	if m == nil {
		m = make(map[string]string)
	}
	m[field] = value
	// add the new field
	return &Error{error: err.error, metaData: m}
}

// MetaData returns copy of error metadata .
// Since it is copied no trace issue needs to be worried about
func (e Error) Metadata() map[string]string {
	if e.metaData == nil {
		return nil
	}
	m := make(map[string]string)
	for k, v := range e.metaData {
		m[k] = v
	}
	return m
}

// Wrap returns an error annotating err with a stack trace
// at the point Wrap is called, and the supplied message.
// If err is nil, Wrap returns nil.
func Wrap(err error, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{error: errors.Wrap(err, message)}
}

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is called, and the format specifier.
// If err is nil, Wrapf returns nil.
func Wrapf(err error, format string, args ...any) *Error {
	return &Error{error: errors.Wrapf(err, format, args...)}
}

// WithStack annotates err with a stack trace at the point
// WithStack is called, and returns the result.
// If err is nil, WithStack returns nil.
func WithStack(err error) *Error {
	return &Error{error: errors.WithStack(err)}
}

// Unwrap returns the underlying error.
// If the error is nil or does not implement Unwrap, Unwrap returns nil.
// Unwrap is implemented by all errors returned by this package.
// It is used by errors.Is and errors.As to check for a specific error (or target) in a Tree of errors.
func (er *Error) Unwrap() error {
	if er == nil {
		return nil
	}
	return er.error
}

// Error returns the error message.
// and adds the metadata to the error message
func (er *Error) Error() string {
	if er == nil {
		return ""
	}
	if er.error == nil {
		return ""
	}
	msg := er.error.Error()
	if len(er.Metadata()) == 0 {
		return msg
	}
	return fmt.Sprintf("%s: %s", msg, er.MetadataString())
}

// MetadataString returns the metadata as a string
// if there is no metadata it returns an empty string
// if there is metadata it returns a string with the format
// key1: value1,key2: value2, ...
func (er Error) MetadataString() string {
	if len(er.Metadata()) == 0 {
		return ""
	}
	var str strings.Builder
	// use strings.Builder
	for k, v := range er.metaData {
		str.WriteString(k)
		str.WriteString(": ")
		str.WriteString(v)
		str.WriteRune(',')
	}
	return str.String()[0 : str.Len()-1]
}

// Format implements fmt.Formatter.
// It supports the formats %+v, %v, %s and %q.
// %+v prints the error message and the stack trace.
// %v and %s print the error message.
// %q prints the error message in quotes.
func (er Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%+v", er.error)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, er.Error()) // nolint: errcheck
	case 'q':
		fmt.Fprintf(s, "%q", er.Error()) // nolint: errcheck
	}
}

func (err *Error) Is(target error) bool {
	if err == nil {
		return false
	}
	if libErr.Is(err.error, target) {
		return true
	}
	if targetErr, ok := target.(*Error); ok {
		return libErr.Is(err.error, targetErr.error)
	}
	return false
}
