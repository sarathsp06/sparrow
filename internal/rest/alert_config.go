package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type createAlertConfigBody struct {
	WebhookID  string   `json:"webhook_id,omitempty" doc:"Scope the alert to one webhook. Omit for a consumer-wide alert covering every webhook."`
	Email      string   `json:"email" required:"true" format:"email" doc:"Recipient email address."`
	EventTypes []string `json:"event_types" required:"true" doc:"Sparrow system event types to alert on: sparrow.webhook.health_changed, sparrow.webhook.delivery_failed."`
}

type createAlertConfigInput struct {
	Consumer string `path:"consumer"`
	Body     createAlertConfigBody
}

type alertConfigItem struct {
	ID         string   `json:"id" doc:"Alert config id (UUID)."`
	Consumer   string   `json:"consumer" doc:"Consumer this alert config belongs to."`
	WebhookID  string   `json:"webhook_id,omitempty" doc:"Webhook this alert is scoped to; omitted for consumer-wide alerts."`
	Email      string   `json:"email" doc:"Recipient email address."`
	EventTypes []string `json:"event_types" doc:"Sparrow system event types this recipient is alerted on."`
	CreatedAt  string   `json:"created_at" doc:"Creation timestamp, RFC3339."`
}

func toAlertConfigItem(c *store.AlertConfig) alertConfigItem {
	item := alertConfigItem{
		ID:         c.ID.String(),
		Consumer:   c.Consumer,
		Email:      c.Email,
		EventTypes: []string(c.EventTypes),
		CreatedAt:  c.CreatedAt.Format(time.RFC3339Nano),
	}
	if c.WebhookID != nil {
		item.WebhookID = c.WebhookID.String()
	}
	return item
}

type alertConfigOutput struct {
	Body alertConfigItem
}

type listAlertConfigsInput struct {
	Consumer  string `path:"consumer" doc:"Consumer to list alert configs in."`
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to alert configs scoped to one webhook. Omit to list every config in the consumer, including consumer-wide ones."`
}

type listAlertConfigsOutput struct {
	Body struct {
		Items []alertConfigItem `json:"items"`
	}
}

type alertConfigIDInput struct {
	Consumer      string `path:"consumer"`
	AlertConfigID string `path:"alert_config_id"`
}

func registerAlertConfigRoutes(api huma.API, svc webhooks.AlertConfigManager) {
	huma.Register(api, huma.Operation{
		OperationID:   "createAlertConfig",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/alert-configs",
		Summary:       "Register an email alert recipient",
		Description:   "Opts an email address into Sparrow's self-generated health-change and permanent-delivery-failure notifications, scoped to one webhook or the whole consumer.",
		Errors:        []int{400, 404},
		Tags:          []string{"Alert Configs"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createAlertConfigInput) (*alertConfigOutput, error) {
		cfg, err := svc.CreateAlertConfig(ctx, in.Consumer, in.Body.WebhookID, in.Body.Email, in.Body.EventTypes)
		if err != nil {
			return nil, mapError(ctx, err, "failed to create alert config")
		}
		return &alertConfigOutput{Body: toAlertConfigItem(cfg)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listAlertConfigs",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/alert-configs",
		Summary:     "List email alert recipients",
		Description: "Lists alert configs in a consumer, optionally filtered to one webhook's own configs.",
		Tags:        []string{"Alert Configs"},
	}, func(ctx context.Context, in *listAlertConfigsInput) (*listAlertConfigsOutput, error) {
		configs, err := svc.ListAlertConfigs(ctx, in.Consumer, in.WebhookID)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list alert configs")
		}
		out := &listAlertConfigsOutput{}
		out.Body.Items = make([]alertConfigItem, 0, len(configs))
		for _, c := range configs {
			out.Body.Items = append(out.Body.Items, toAlertConfigItem(c))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteAlertConfig",
		Method:        http.MethodDelete,
		Path:          "/v1/consumers/{consumer}/alert-configs/{alert_config_id}",
		Summary:       "Delete an email alert recipient",
		Description:   "Removes an alert config. The webhook and its delivery history are unaffected.",
		Errors:        []int{404},
		Tags:          []string{"Alert Configs"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *alertConfigIDInput) (*emptyOutput, error) {
		if err := svc.DeleteAlertConfig(ctx, in.Consumer, in.AlertConfigID); err != nil {
			return nil, mapError(ctx, err, "failed to delete alert config")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}
