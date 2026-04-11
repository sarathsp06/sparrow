package grpc

import (
	"math"
	"testing"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	pb "github.com/sarathsp06/sparrow/proto"
)

// --- float32/float64 pointer conversion helpers ---

func TestFloat32PtrToFloat64Ptr(t *testing.T) {
	tests := []struct {
		name string
		in   *float32
		want *float64
	}{
		{"nil returns nil", nil, nil},
		{"positive value", float32Ptr(10.5), float64Ptr(float64(float32(10.5)))},
		{"zero", float32Ptr(0), float64Ptr(0)},
		{"very small", float32Ptr(0.001), float64Ptr(float64(float32(0.001)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float32PtrToFloat64Ptr(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}
			if math.Abs(*got-*tt.want) > 1e-6 {
				t.Fatalf("got %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestFloat64PtrToFloat32Ptr(t *testing.T) {
	tests := []struct {
		name string
		in   *float64
		want *float32
	}{
		{"nil returns nil", nil, nil},
		{"positive value", float64Ptr(10.5), float32Ptr(10.5)},
		{"zero", float64Ptr(0), float32Ptr(0)},
		{"very small", float64Ptr(0.001), float32Ptr(0.001)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := float64PtrToFloat32Ptr(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil, got nil")
			}
			if math.Abs(float64(*got)-float64(*tt.want)) > 1e-6 {
				t.Fatalf("got %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestFloat64_Float32_Roundtrip(t *testing.T) {
	original := float64Ptr(5.25)
	asFloat32 := float64PtrToFloat32Ptr(original)
	back := float32PtrToFloat64Ptr(asFloat32)

	if back == nil {
		t.Fatal("roundtrip returned nil")
	}
	if math.Abs(*back-*original) > 1e-6 {
		t.Fatalf("roundtrip: got %v, want %v", *back, *original)
	}
}

func TestFloat64_Float32_Roundtrip_Nil(t *testing.T) {
	asFloat32 := float64PtrToFloat32Ptr(nil)
	back := float32PtrToFloat64Ptr(asFloat32)
	if back != nil {
		t.Fatalf("roundtrip of nil should be nil, got %v", *back)
	}
}

// --- ConvertProtoHTTPConfig / ConvertInternalHTTPConfig ---

func TestConvertProtoHTTPConfig_NilReturnsDefaults(t *testing.T) {
	config := ConvertProtoHTTPConfig(nil)
	if config == nil {
		t.Fatal("expected default config, got nil")
	}
	if config.RateLimitRPS != nil {
		t.Fatalf("expected nil RateLimitRPS in default config, got %v", *config.RateLimitRPS)
	}
}

func TestConvertProtoHTTPConfig_WithRateLimitRPS(t *testing.T) {
	rps := float32(10.0)
	pbConfig := &pb.WebhookHTTPConfig{
		RateLimitRps: &rps,
	}
	config := ConvertProtoHTTPConfig(pbConfig)
	if config.RateLimitRPS == nil {
		t.Fatal("expected RateLimitRPS to be set")
	}
	if math.Abs(*config.RateLimitRPS-10.0) > 1e-6 {
		t.Fatalf("got RateLimitRPS %v, want 10.0", *config.RateLimitRPS)
	}
}

func TestConvertProtoHTTPConfig_WithoutRateLimitRPS(t *testing.T) {
	protoConfig := &pb.WebhookHTTPConfig{}
	config := ConvertProtoHTTPConfig(protoConfig)
	if config.RateLimitRPS != nil {
		t.Fatalf("expected nil RateLimitRPS, got %v", *config.RateLimitRPS)
	}
}

func TestConvertInternalHTTPConfig_NilReturnsNil(t *testing.T) {
	got := ConvertInternalHTTPConfig(nil)
	if got != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestConvertInternalHTTPConfig_WithRateLimitRPS(t *testing.T) {
	rps := 5.0
	config := &webhooks.WebhookHTTPConfig{
		RateLimitRPS: &rps,
	}
	result := ConvertInternalHTTPConfig(config)
	if result.RateLimitRps == nil {
		t.Fatal("expected RateLimitRps to be set in proto")
	}
	if math.Abs(float64(*result.RateLimitRps)-5.0) > 1e-6 {
		t.Fatalf("got %v, want 5.0", *result.RateLimitRps)
	}
}

func TestConvertInternalHTTPConfig_WithoutRateLimitRPS(t *testing.T) {
	config := &webhooks.WebhookHTTPConfig{}
	result := ConvertInternalHTTPConfig(config)
	if result.RateLimitRps != nil {
		t.Fatalf("expected nil RateLimitRps in proto, got %v", *result.RateLimitRps)
	}
}

func TestHTTPConfig_Roundtrip_WithRateLimitRPS(t *testing.T) {
	rps := 25.0
	original := &webhooks.WebhookHTTPConfig{
		MaxRetries:            3,
		RetryBackoffSeconds:   60,
		RequestTimeoutSeconds: 30,
		UserAgent:             "Sparrow-Webhook/1.0",
		ContentType:           "application/json",
		ExpectedStatusCodes:   webhooks.IntArray{200, 201},
		RateLimitRPS:          &rps,
	}

	// Internal -> Proto -> Internal
	pbResult := ConvertInternalHTTPConfig(original)
	back := ConvertProtoHTTPConfig(pbResult)

	if back.RateLimitRPS == nil {
		t.Fatal("roundtrip lost RateLimitRPS")
	}
	if math.Abs(*back.RateLimitRPS-rps) > 1e-6 {
		t.Fatalf("roundtrip: got %v, want %v", *back.RateLimitRPS, rps)
	}
	if back.MaxRetries != original.MaxRetries {
		t.Fatalf("roundtrip: MaxRetries got %d, want %d", back.MaxRetries, original.MaxRetries)
	}
}

func TestHTTPConfig_Roundtrip_WithoutRateLimitRPS(t *testing.T) {
	original := &webhooks.WebhookHTTPConfig{
		MaxRetries:            5,
		RetryBackoffSeconds:   120,
		RequestTimeoutSeconds: 60,
		UserAgent:             "Sparrow-Webhook/1.0",
		ContentType:           "application/json",
	}

	pbResult := ConvertInternalHTTPConfig(original)
	back := ConvertProtoHTTPConfig(pbResult)

	if back.RateLimitRPS != nil {
		t.Fatalf("roundtrip should keep RateLimitRPS nil, got %v", *back.RateLimitRPS)
	}
}

// --- helpers ---

func float32Ptr(f float32) *float32 { return &f }
func float64Ptr(f float64) *float64 { return &f }
