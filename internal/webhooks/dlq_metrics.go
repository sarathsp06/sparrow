package webhooks

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// RegisterDLQDepthGauge registers the sparrow.dlq.depth observable gauge:
// the number of terminal-failed deliveries per webhook, observed at each
// metric export by counting rows via the DLQ partial index.
func RegisterDLQDepthGauge(repo store.DeliveryRepository) error {
	meter := observability.GetMeter("sparrow")

	gauge, err := meter.Int64ObservableGauge(
		"sparrow.dlq.depth",
		metric.WithDescription("Number of terminal-failed (DLQ) deliveries per webhook"),
	)
	if err != nil {
		return err
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		depths, err := repo.CountFailedDeliveriesByWebhook(ctx, tenant.DefaultTenantID)
		if err != nil {
			slog.Default().ErrorContext(ctx, "DLQ depth observation failed", "error", err)
			return err
		}
		for _, d := range depths {
			o.ObserveInt64(gauge, d.Depth, metric.WithAttributes(
				attribute.String("webhook_id", d.WebhookID.String()),
				attribute.String("consumer", d.Consumer),
			))
		}
		return nil
	}, gauge)
	return err
}
