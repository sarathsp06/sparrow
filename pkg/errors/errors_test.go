package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrap(t *testing.T) {
	err := fmt.Errorf("test error")
	wrappedErr := Wrap(err, "wrapped error")
	assert.Equal(t, wrappedErr.Error(), "wrapped error: test error")
}

func TestWrapf(t *testing.T) {
	err := fmt.Errorf("test error")
	wrappedErr := Wrapf(err, "wrapped error with %d argument", 1)
	assert.Equal(t, wrappedErr.Error(), "wrapped error with 1 argument: test error")
}

func TestWithStack(t *testing.T) {
	err := fmt.Errorf("test error")
	wrappedErr := WithStack(err)
	assert.Contains(t, wrappedErr.Error(), "test error")
	assert.Contains(t, fmt.Sprintf("%+v", wrappedErr), "errors_test.go:")
}

func TestMetadata(t *testing.T) {
	err := Error{
		error:    fmt.Errorf("test error"),
		metaData: map[string]string{"key": "value"},
	}
	assert.Equal(t, err.Metadata()["key"], "value")
	assert.Equal(t, err.MetadataString(), "key: value")
}

func TestWithField(t *testing.T) {
	// test WithField using table driven test
	tests := []struct {
		name     string
		err      *Error
		field    string
		value    string
		expected map[string]string
	}{
		{
			name:     "new error without field values",
			err:      New("test error"),
			field:    "key",
			value:    "value",
			expected: map[string]string{"key": "value"},
		},
		{
			name:     "error with already a field value",
			err:      &Error{error: fmt.Errorf("test error"), metaData: map[string]string{"key": "value"}},
			field:    "key2",
			value:    "value2",
			expected: map[string]string{"key": "value", "key2": "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// copy original error metadata
			var metadataCopy = make(map[string]string)
			for k, v := range tt.err.Metadata() {
				metadataCopy[k] = v
			}
			err := tt.err.WithField(tt.field, tt.value)
			assert.Equal(t, tt.expected, err.Metadata(), "error  metadata should have new field added")
			assert.Equal(t, metadataCopy, tt.err.Metadata(), "original error should not change")
		})
	}
}
