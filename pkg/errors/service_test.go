package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ServiceError
		want string
	}{
		{
			name: "without cause",
			err:  Error(InvalidArgument, "namespace is required"),
			want: "namespace is required",
		},
		{
			name: "with cause",
			err:  Wrap(fmt.Errorf("dial tcp: connection refused"), Internal, "failed to validate URL"),
			want: "failed to validate URL: dial tcp: connection refused",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("ServiceError.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestServiceError_ClientMessage(t *testing.T) {
	// ClientMessage should always return only the safe message, never the cause
	err := Wrapf(fmt.Errorf("SQL: relation \"foo\" does not exist"), Internal, "failed to create webhook")
	if got := err.ClientMessage(); got != "failed to create webhook" {
		t.Errorf("ClientMessage() = %q, want %q", got, "failed to create webhook")
	}

	err2 := Error(InvalidArgument, "loopback addresses are not allowed")
	if got := err2.ClientMessage(); got != "loopback addresses are not allowed" {
		t.Errorf("ClientMessage() = %q, want %q", got, "loopback addresses are not allowed")
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Wrap(cause, Internal, "wrapper")

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}

	// Without cause, Unwrap returns nil
	err2 := Error(InvalidArgument, "no cause")
	if err2.Unwrap() != nil {
		t.Error("Unwrap should return nil when there is no cause")
	}
}

func TestServiceError_ErrorsAs(t *testing.T) {
	// Wrapping a ServiceError in fmt.Errorf should still be extractable via errors.As
	inner := Error(InvalidArgument, "bad input")
	wrapped := fmt.Errorf("service call failed: %w", inner)

	var svcErr *ServiceError
	if !errors.As(wrapped, &svcErr) {
		t.Fatal("errors.As should find ServiceError through fmt.Errorf wrapping")
	}
	if svcErr.Status != InvalidArgument {
		t.Errorf("Status = %v, want %v", svcErr.Status, InvalidArgument)
	}
	if svcErr.ClientMessage() != "bad input" {
		t.Errorf("ClientMessage() = %q, want %q", svcErr.ClientMessage(), "bad input")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServiceError
		wantCode Status
		wantMsg  string
	}{
		{"Error InvalidArgument", Error(InvalidArgument, "bad"), InvalidArgument, "bad"},
		{"Errorf InvalidArgument", Errorf(InvalidArgument, "field %q is invalid", "name"), InvalidArgument, `field "name" is invalid`},
		{"Error FailedPrecondition", Error(FailedPrecondition, "event is inactive"), FailedPrecondition, "event is inactive"},
		{"Errorf FailedPrecondition", Errorf(FailedPrecondition, "batch %s expired", "abc"), FailedPrecondition, "batch abc expired"},
		{"Error NotFound", Error(NotFound, "webhook not found"), NotFound, "webhook not found"},
		{"Errorf NotFound", Errorf(NotFound, "event %q not found", "click"), NotFound, `event "click" not found`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Status != tt.wantCode {
				t.Errorf("Status = %v, want %v", tt.err.Status, tt.wantCode)
			}
			if tt.err.ClientMessage() != tt.wantMsg {
				t.Errorf("ClientMessage() = %q, want %q", tt.err.ClientMessage(), tt.wantMsg)
			}
		})
	}
}
