package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
	OTLPHeaders    map[string]string
	EnableTracing  bool
	EnableMetrics  bool
	SampleRate     float64 // 0.0 to 1.0
	MetricInterval time.Duration
}

// DefaultConfig returns a default OpenTelemetry configuration
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "sparrow",
		ServiceVersion: "1.0.0",
		Environment:    "development",
		OTLPEndpoint:   "localhost:4318", // Default OTLP HTTP endpoint
		EnableTracing:  true,
		EnableMetrics:  true,
		SampleRate:     1.0, // Sample all traces in development
		MetricInterval: 30 * time.Second,
	}
}

// Setup initializes OpenTelemetry with the provided configuration
func Setup(ctx context.Context, config *Config) (func(context.Context) error, error) {
	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var shutdownFuncs []func(context.Context) error

	// Setup tracing
	if config.EnableTracing {
		tracerProvider, err := setupTracing(ctx, res, config)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		otel.SetTracerProvider(tracerProvider)
	}

	// Setup metrics
	if config.EnableMetrics {
		meterProvider, err := setupMetrics(ctx, res, config)
		if err != nil {
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
		otel.SetMeterProvider(meterProvider)
	}

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Return shutdown function
	return func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		if len(errs) > 0 {
			return fmt.Errorf("failed to shutdown OpenTelemetry: %w", errors.Join(errs...))
		}
		return nil
	}, nil
}

// setupTracing configures OpenTelemetry tracing
func setupTracing(ctx context.Context, res *resource.Resource, config *Config) (*sdktrace.TracerProvider, error) {
	// Create OTLP trace exporter
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	}

	if len(config.OTLPHeaders) > 0 {
		opts = append(opts, otlptracehttp.WithHeaders(config.OTLPHeaders))
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Configure sampler based on sample rate
	var sampler sdktrace.Sampler
	if config.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if config.SampleRate <= 0.0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(config.SampleRate)
	}

	// Create tracer provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return tracerProvider, nil
}

// setupMetrics configures OpenTelemetry metrics
func setupMetrics(ctx context.Context, res *resource.Resource, config *Config) (*sdkmetric.MeterProvider, error) {
	// Create OTLP metric exporter
	opts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint("localhost:4318"),
		otlpmetrichttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	}

	if len(config.OTLPHeaders) > 0 {
		opts = append(opts, otlpmetrichttp.WithHeaders(config.OTLPHeaders))
	}

	exporter, err := otlpmetrichttp.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(config.MetricInterval))),
	)

	return meterProvider, nil
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name, trace.WithInstrumentationVersion("1.0.0"))
}

// GetMeter returns a meter for the given name
func GetMeter(name string) metric.Meter {
	return otel.Meter(name, metric.WithInstrumentationVersion("1.0.0"))
}

// SparrowMetrics holds application-specific metrics
type SparrowMetrics struct {
	WebhookRegistrations metric.Int64Counter
	EventsPushed         metric.Int64Counter
	WebhookDeliveries    metric.Int64Counter
	DeliveryDuration     metric.Float64Histogram
	QueueDepth           metric.Int64UpDownCounter
	ActiveWebhooks       metric.Int64UpDownCounter
}

// NewSparrowMetrics creates application-specific metrics
func NewSparrowMetrics() (*SparrowMetrics, error) {
	meter := GetMeter("sparrow")

	webhookRegistrations, err := meter.Int64Counter(
		"sparrow_webhook_registrations_total",
		metric.WithDescription("Total number of webhook registrations"),
	)
	if err != nil {
		return nil, err
	}

	eventsPushed, err := meter.Int64Counter(
		"sparrow_events_pushed_total",
		metric.WithDescription("Total number of events pushed"),
	)
	if err != nil {
		return nil, err
	}

	webhookDeliveries, err := meter.Int64Counter(
		"sparrow_webhook_deliveries_total",
		metric.WithDescription("Total number of webhook delivery attempts"),
	)
	if err != nil {
		return nil, err
	}

	deliveryDuration, err := meter.Float64Histogram(
		"sparrow_webhook_delivery_duration_seconds",
		metric.WithDescription("Duration of webhook deliveries in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	queueDepth, err := meter.Int64UpDownCounter(
		"sparrow_queue_depth",
		metric.WithDescription("Current depth of job queues"),
	)
	if err != nil {
		return nil, err
	}

	activeWebhooks, err := meter.Int64UpDownCounter(
		"sparrow_active_webhooks",
		metric.WithDescription("Current number of active webhook registrations"),
	)
	if err != nil {
		return nil, err
	}

	return &SparrowMetrics{
		WebhookRegistrations: webhookRegistrations,
		EventsPushed:         eventsPushed,
		WebhookDeliveries:    webhookDeliveries,
		DeliveryDuration:     deliveryDuration,
		QueueDepth:           queueDepth,
		ActiveWebhooks:       activeWebhooks,
	}, nil
}
