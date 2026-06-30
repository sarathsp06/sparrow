package errors

import (
	"errors"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *ServiceError
		want string
	}{
		{
			name: "without cause",
			err:  Error(codes.InvalidArgument, "namespace is required"),
			want: "namespace is required",
		},
		{
			name: "with cause",
			err:  Wrap(fmt.Errorf("dial tcp: connection refused"), codes.Internal, "failed to validate URL"),
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
	err := Wrapf(fmt.Errorf("SQL: relation \"foo\" does not exist"), codes.Internal, "failed to create webhook")
	if got := err.ClientMessage(); got != "failed to create webhook" {
		t.Errorf("ClientMessage() = %q, want %q", got, "failed to create webhook")
	}

	err2 := Error(codes.InvalidArgument, "loopback addresses are not allowed")
	if got := err2.ClientMessage(); got != "loopback addresses are not allowed" {
		t.Errorf("ClientMessage() = %q, want %q", got, "loopback addresses are not allowed")
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	err := Wrap(cause, codes.Internal, "wrapper")

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}

	// Without cause, Unwrap returns nil
	err2 := Error(codes.InvalidArgument, "no cause")
	if err2.Unwrap() != nil {
		t.Error("Unwrap should return nil when there is no cause")
	}
}

func TestServiceError_ErrorsAs(t *testing.T) {
	// Wrapping a ServiceError in fmt.Errorf should still be extractable via errors.As
	inner := Error(codes.InvalidArgument, "bad input")
	wrapped := fmt.Errorf("service call failed: %w", inner)

	var svcErr *ServiceError
	if !errors.As(wrapped, &svcErr) {
		t.Fatal("errors.As should find ServiceError through fmt.Errorf wrapping")
	}
	if svcErr.GRPCCode != codes.InvalidArgument {
		t.Errorf("GRPCCode = %v, want %v", svcErr.GRPCCode, codes.InvalidArgument)
	}
	if svcErr.ClientMessage() != "bad input" {
		t.Errorf("ClientMessage() = %q, want %q", svcErr.ClientMessage(), "bad input")
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name     string
		err      *ServiceError
		wantCode codes.Code
		wantMsg  string
	}{
		{"Error InvalidArgument", Error(codes.InvalidArgument, "bad"), codes.InvalidArgument, "bad"},
		{"Errorf InvalidArgument", Errorf(codes.InvalidArgument, "field %q is invalid", "name"), codes.InvalidArgument, `field "name" is invalid`},
		{"Error FailedPrecondition", Error(codes.FailedPrecondition, "event is inactive"), codes.FailedPrecondition, "event is inactive"},
		{"Errorf FailedPrecondition", Errorf(codes.FailedPrecondition, "batch %s expired", "abc"), codes.FailedPrecondition, "batch abc expired"},
		{"Error NotFound", Error(codes.NotFound, "webhook not found"), codes.NotFound, "webhook not found"},
		{"Errorf NotFound", Errorf(codes.NotFound, "event %q not found", "click"), codes.NotFound, `event "click" not found`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.GRPCCode != tt.wantCode {
				t.Errorf("GRPCCode = %v, want %v", tt.err.GRPCCode, tt.wantCode)
			}
			if tt.err.ClientMessage() != tt.wantMsg {
				t.Errorf("ClientMessage() = %q, want %q", tt.err.ClientMessage(), tt.wantMsg)
			}
		})
	}
}
