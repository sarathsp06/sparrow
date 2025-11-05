// Package testutils have helper functions to ease testing
package testutils

import (
	"github.com/stretchr/testify/assert"
)

// ErrorIs asserts that a specified error is the same as a target error.
func ErrorIs(targetError error, msgAndArgs ...any) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, msgAndArgs ...any) bool {
		return assert.ErrorIs(t, err, targetError, msgAndArgs...)
	}

}

// ErrorContains asserts that a specified error contains a substring.
func ErrorContains(contains string, msgAndArgs ...any) assert.ErrorAssertionFunc {
	return func(t assert.TestingT, err error, msgAndArgs ...any) bool {
		return assert.ErrorContains(t, err, contains, msgAndArgs...)
	}
}

// Equal asserts that two objects are equal.
func Equal(expect any, msg string) assert.ValueAssertionFunc {
	return func(t assert.TestingT, got any, args ...any) bool {
		return assert.Equal(t, expect, got, append([]any{msg}, args...)...)
	}
}
