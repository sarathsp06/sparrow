package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPtrIfNotZeroInt(t *testing.T) {
	tests := []struct {
		name string
		args int
		want *int
	}{
		{
			name: "zero",
			args: 0,
			want: nil,
		},
		{
			name: "non-zero",
			args: 42,
			want: Ptr(42),
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, PtrIfNotZero(tt.args))
	}
}

func TestPtrIfNotZeroString(t *testing.T) {
	tests := []struct {
		name string
		args string
		want *string
	}{
		{
			name: "empty",
			args: "",
			want: nil,
		},
		{
			name: "non-empty",
			args: "hello",
			want: Ptr("hello"),
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, PtrIfNotZero(tt.args))
	}
}

type isZeroImpl string

func (i isZeroImpl) IsZero() bool {
	return i == "zero"
}

func TestPtrIfNotZeroIsZero(t *testing.T) {
	tests := []struct {
		name string
		args isZeroImpl
		want *isZeroImpl
	}{
		{
			name: "zero",
			args: "zero",
			want: nil,
		},
		{
			name: "non-zero",
			args: "non-zero",
			want: Ptr(isZeroImpl("non-zero")),
		},
		{
			name: "non-zero empty string",
			args: "",
			want: Ptr(isZeroImpl("")),
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, PtrIfNotZero(tt.args), tt.name)
	}
}
