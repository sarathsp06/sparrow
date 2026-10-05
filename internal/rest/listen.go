package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

type createListenSessionBody struct {
	Events      []string `json:"events" required:"true" minItems:"1" doc:"Event types the session receives."`
	TTLSeconds  int      `json:"ttl_seconds,omitempty" minimum:"0" doc:"Session lifetime in seconds. 0 or more than the server maximum (SPARROW_LISTEN_MAX_TTL) means the maximum."`
	Description string   `json:"description,omitempty" maxLength:"200" doc:"Shown on the session's webhook, e.g. who is listening from where."`
}

type createListenSessionInput struct {
	Consumer string `path:"consumer"`
	Body     createListenSessionBody
}

type listenSessionItem struct {
	SessionID string   `json:"session_id" doc:"Session id. It is also the id of the session's webhook (URL sparrow-cli://<session_id>), so its deliveries show up in the delivery history."`
	Consumer  string   `json:"consumer"`
	Events    []string `json:"events"`
	Secret    string   `json:"webhook_secret" doc:"Signing secret of the session's deliveries. Returned only here."`
	CreatedAt string   `json:"created_at" doc:"RFC3339."`
	ExpiresAt string   `json:"expires_at" doc:"RFC3339. The session and its webhook are deleted after this."`
}

type listenSessionOutput struct {
	Body listenSessionItem
}

type listenSessionIDInput struct {
	Consumer  string `path:"consumer"`
	SessionID string `path:"session_id"`
}

type claimListenDeliveriesInput struct {
	Consumer    string `path:"consumer"`
	SessionID   string `path:"session_id"`
	WaitSeconds int    `query:"wait_seconds" minimum:"0" maximum:"20" default:"20" doc:"How long to wait for a delivery before returning an empty list."`
}

type listenDeliveryItem struct {
	DeliveryID string            `json:"delivery_id"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers" doc:"Request headers exactly as a receiver gets them, including the webhook-id/-timestamp/-signature signing headers."`
	Body       []byte            `json:"body" doc:"Request body (base64)."`
}

type claimListenDeliveriesOutput struct {
	Body struct {
		Items []listenDeliveryItem `json:"items"`
	}
}

type respondListenDeliveryInput struct {
	Consumer   string `path:"consumer"`
	SessionID  string `path:"session_id"`
	DeliveryID string `path:"delivery_id"`
	Body       struct {
		Status  int               `json:"status" required:"true" minimum:"100" maximum:"599" doc:"HTTP status the local app answered with."`
		Headers map[string]string `json:"headers,omitempty"`
		Body    []byte            `json:"body,omitempty" doc:"Response body (base64)."`
	}
}

func registerListenRoutes(api huma.API, svc webhooks.ListenManager) {
	huma.Register(api, huma.Operation{
		OperationID: "createListenSession",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/listen-sessions",
		Summary:     "Start a listen session",
		Description: "Creates a temporary webhook that is delivered to a polling client (`sparrow listen`) instead of a URL, " +
			"so a developer can receive this server's deliveries on a machine it cannot reach. " +
			"Off unless the server runs with SPARROW_LISTEN_ENABLED=true.",
		Errors:        []int{400, 403, 429},
		Tags:          []string{"Listen Sessions"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createListenSessionInput) (*listenSessionOutput, error) {
		s, err := svc.CreateListenSession(ctx, webhooks.ListenSessionRequest{
			Consumer:    in.Consumer,
			Events:      in.Body.Events,
			TTL:         time.Duration(in.Body.TTLSeconds) * time.Second,
			Description: in.Body.Description,
		})
		if err != nil {
			return nil, mapError(ctx, err, "failed to create listen session")
		}
		return &listenSessionOutput{Body: listenSessionItem{
			SessionID: s.SessionID,
			Consumer:  s.Consumer,
			Events:    s.Events,
			Secret:    s.Secret,
			CreatedAt: s.CreatedAt.Format(time.RFC3339Nano),
			ExpiresAt: s.ExpiresAt.Format(time.RFC3339Nano),
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteListenSession",
		Method:        http.MethodDelete,
		Path:          "/v1/consumers/{consumer}/listen-sessions/{session_id}",
		Summary:       "Stop a listen session",
		Description:   "Deletes the session and its webhook, subscriptions and deliveries.",
		Errors:        []int{403, 404},
		Tags:          []string{"Listen Sessions"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *listenSessionIDInput) (*emptyOutput, error) {
		if err := svc.DeleteListenSession(ctx, in.Consumer, in.SessionID); err != nil {
			return nil, mapError(ctx, err, "failed to delete listen session")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "claimListenDeliveries",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/listen-sessions/{session_id}:claim",
		Summary:     "Wait for a listen session's deliveries",
		Description: "Long-polls for deliveries to the session and claims them. Answer each with respondListenDelivery " +
			"before the webhook's request timeout, or the attempt fails as a timeout and is retried. " +
			"Polling also keeps the session connected: a session not polled for 45 seconds fails its deliveries as offline.",
		Errors: []int{403, 404, 409},
		Tags:   []string{"Listen Sessions"},
	}, func(ctx context.Context, in *claimListenDeliveriesInput) (*claimListenDeliveriesOutput, error) {
		claimed, err := svc.ClaimListenDeliveries(ctx, in.Consumer, in.SessionID, time.Duration(in.WaitSeconds)*time.Second)
		if err != nil {
			return nil, mapError(ctx, err, "failed to claim listen deliveries")
		}
		out := &claimListenDeliveriesOutput{}
		out.Body.Items = make([]listenDeliveryItem, 0, len(claimed))
		for _, d := range claimed {
			out.Body.Items = append(out.Body.Items, listenDeliveryItem{
				DeliveryID: d.DeliveryID.String(),
				Method:     d.Method,
				Headers:    d.Headers,
				Body:       d.Body,
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "respondListenDelivery",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/listen-sessions/{session_id}/deliveries/{delivery_id}:respond",
		Summary:       "Report a listen delivery's response",
		Description:   "Records the local app's response to a claimed delivery. It becomes the delivery attempt's outcome, judged like an HTTP response (expected status codes, retries).",
		Errors:        []int{400, 403, 404},
		Tags:          []string{"Listen Sessions"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *respondListenDeliveryInput) (*emptyOutput, error) {
		if err := svc.RespondListenDelivery(ctx, in.Consumer, in.SessionID, in.DeliveryID, in.Body.Status, in.Body.Headers, in.Body.Body); err != nil {
			return nil, mapError(ctx, err, "failed to record listen delivery response")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})
}
