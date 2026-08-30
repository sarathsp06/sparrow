package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/sarathsp06/sparrow"
)

// metricExportInterval is how often the OTLP metric reader flushes.
const metricExportInterval = 30 * time.Second

// Config holds OpenTelemetry configuration
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	OTLPEndpoint   string
}

// DefaultConfig returns a default OpenTelemetry configuration.
// OTLPEndpoint is empty by default -- set it to enable export.
// When OTLPEndpoint is empty, Setup() is a no-op (no exporters created).
func DefaultConfig() *Config {
	return &Config{
		ServiceName:    "sparrow",
		ServiceVersion: sparrow.Version,
		Environment:    "development",
		OTLPEndpoint:   "", // empty = OTel export disabled
	}
}

// Setup initializes OpenTelemetry with the provided configuration.
// When config.OTLPEndpoint is empty, no exporters are created and a no-op
// shutdown function is returned. This avoids noisy connection errors when
// no collector is available.
func Setup(ctx context.Context, config *Config) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }

	// No endpoint configured -- skip all OTLP export.
	if config.OTLPEndpoint == "" {
		// Still install the propagator so trace context propagation works
		// even without an exporter (e.g. inbound headers are parsed).
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		return noop, nil
	}

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

	tracerProvider, err := setupTracing(ctx, res, config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup tracing: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	meterProvider, err := setupMetrics(ctx, res, config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup metrics: %w", err)
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	loggerProvider, err := newLoggerProvider(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

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
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(config.OTLPEndpoint),
		otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create tracer provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	return tracerProvider, nil
}

func newLoggerProvider(ctx context.Context, config *Config) (*log.LoggerProvider, error) {
	exporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint(config.OTLPEndpoint),
		otlploghttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)
	return loggerProvider, nil
}

// setupMetrics configures OpenTelemetry metrics
func setupMetrics(ctx context.Context, res *resource.Resource, config *Config) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(config.OTLPEndpoint),
		otlpmetrichttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP metric exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter,
			sdkmetric.WithInterval(metricExportInterval))),
	)

	return meterProvider, nil
}

// GetTracer returns a tracer for the given name
func GetTracer(name string) trace.Tracer {
	return otel.Tracer(name, trace.WithInstrumentationVersion(sparrow.Version))
}

// GetMeter returns a meter for the given name
func GetMeter(name string) metric.Meter {
	return otel.Meter(name, metric.WithInstrumentationVersion(sparrow.Version))
}

// SparrowMetrics holds application-specific metrics
type SparrowMetrics struct {
	WebhookRegistrations metric.Int64Counter
	EventsPushed         metric.Int64Counter
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
		ActiveWebhooks:       activeWebhooks,
	}, nil
}
