package main

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
)

// applyRequest is how a recipe is applied. The CLI (sparrow use) and the MCP
// apply_recipe tool share it, so both validate, detect an existing webhook and
// report the same way.
type applyRequest struct {
	Events []string
	Labels map[string]string
	// DryRun reads but creates and changes nothing: the result says what
	// would happen.
	DryRun bool
	// Update re-applies the recipe to an existing webhook with the same URL
	// (its headers, transform and subscriptions) instead of refusing.
	Update bool
}

type appliedSubscription struct {
	SubscriptionID string            `json:"subscription_id,omitempty"`
	EventName      string            `json:"event_name"`
	Action         string            `json:"action"` // created, updated, or would_ with either
	Transform      bool              `json:"transform"`
	LabelFilters   map[string]string `json:"label_filters,omitempty"`
}

type appliedRecipe struct {
	Recipe   string `json:"recipe"`
	Consumer string `json:"consumer"`
	DryRun   bool   `json:"dry_run,omitempty"`
	// Action is created, updated, would_create or would_update.
	Action    string `json:"action"`
	WebhookID string `json:"webhook_id,omitempty"`
	// URL is the destination with secret params masked.
	URL           string                `json:"url"`
	Subscriptions []appliedSubscription `json:"subscriptions"`
	// Rendered is the transform's output for the sample event, and
	// SampleSource where that sample came from.
	Rendered     string   `json:"rendered,omitempty"`
	SampleSource string   `json:"sample_source,omitempty"`
	Warnings     []string `json:"warnings"`
}

// recipeInvalidError means the recipe failed validation; nothing was created.
type recipeInvalidError struct{ report recipeReport }

func (e *recipeInvalidError) Error() string {
	return fmt.Sprintf("recipe %s did not validate, nothing was created: %s", e.report.Recipe, strings.Join(e.report.Errors, "; "))
}

// webhookExistsError means the consumer already has a webhook for the
// recipe's destination; applying again needs an explicit update.
type webhookExistsError struct {
	WebhookID, Description, URL string
}

func (e *webhookExistsError) Error() string {
	desc := ""
	if e.Description != "" {
		desc = fmt.Sprintf(" (%s)", e.Description)
	}
	return fmt.Sprintf("a webhook for %s already exists: %s%s", e.URL, e.WebhookID, desc)
}

// eventSample returns the registered sample payload of an event type, so a
// recipe is checked against what will really be delivered, and where the
// sample came from. Without one, validation falls back to a built-in sample.
func eventSample(ctx context.Context, client *apiClient, event string) (map[string]any, string) {
	if event != "" && event != "*" {
		if et, err := client.getEventType(ctx, event); err == nil && len(et.SamplePayload) > 0 {
			return et.SamplePayload, "the registered sample payload of " + event
		}
	}
	if event != "" && event != "*" {
		return nil, "the built-in sample payload, since " + event + " has no registered sample"
	}
	return nil, "the built-in sample payload"
}

// applyRecipeTo validates r with values, then creates (or, with req.Update,
// re-applies to) the consumer's webhook for its destination. On a partial
// failure the result still names what was created.
func applyRecipeTo(ctx context.Context, client *apiClient, consumer string, r recipe, values map[string]string, req applyRequest) (appliedRecipe, error) {
	res := appliedRecipe{Recipe: r.Name, Consumer: consumer, DryRun: req.DryRun, Subscriptions: []appliedSubscription{}, Warnings: []string{}}
	if len(req.Events) == 0 {
		return res, errors.New("at least one event type is required")
	}
	rr, err := substituteRecipe(r, values)
	if err != nil {
		return res, err
	}

	payload, source := eventSample(ctx, client, req.Events[0])
	rep := validateRecipe(r, values, req.Events[0], payload)
	res.URL, res.Rendered, res.SampleSource = rep.URL, rep.Rendered, source
	res.Warnings = append(res.Warnings, applyWarnings(rep.Warnings)...)
	if !rep.Valid {
		return res, &recipeInvalidError{report: rep}
	}
	if r.Consumer != nil && r.Consumer.Name != consumer {
		res.Warnings = append(res.Warnings, fmt.Sprintf("recipe is meant for consumer %q, applied in %q: %s", r.Consumer.Name, consumer, r.Consumer.Note))
	}

	existing, err := findWebhookByURL(ctx, client, consumer, rr.URL)
	if err != nil {
		return res, fmt.Errorf("look up existing webhooks: %w", err)
	}
	if existing != nil && !req.Update {
		return res, &webhookExistsError{WebhookID: existing.WebhookID, Description: existing.Description, URL: res.URL}
	}
	if existing != nil {
		return updateRecipeWebhook(ctx, client, consumer, r, rr, *existing, req, res)
	}

	res.Action = "created"
	if req.DryRun {
		res.Action = "would_create"
		for _, e := range req.Events {
			res.Subscriptions = append(res.Subscriptions, appliedSubscription{EventName: e, Action: "would_create", Transform: rr.Template != "", LabelFilters: req.Labels})
		}
		return res, nil
	}
	hook, err := client.registerWebhook(ctx, consumer, webhookRequest{
		URL:               rr.URL,
		Events:            req.Events,
		Active:            true,
		Description:       recipeDescription(r),
		Headers:           rr.Headers,
		SecretHeaders:     rr.SecretHeaders,
		RequiresTransform: r.Webhook.RequiresTransform,
		TransformTemplate: rr.Template,
	})
	if err != nil {
		return res, fmt.Errorf("register webhook: %w", err)
	}
	res.WebhookID = hook.WebhookID

	subs, err := client.listSubscriptions(ctx, consumer, hook.WebhookID)
	if err != nil {
		return res, fmt.Errorf("list subscriptions of webhook %s: %w", hook.WebhookID, err)
	}
	for _, sub := range subs {
		if err := client.patchSubscription(ctx, consumer, sub.SubscriptionID, recipePatch(rr, req.Labels)); err != nil {
			return res, fmt.Errorf("update subscription %s of webhook %s: %w", sub.SubscriptionID, hook.WebhookID, err)
		}
		res.Subscriptions = append(res.Subscriptions, appliedSubscription{SubscriptionID: sub.SubscriptionID, EventName: sub.EventName, Action: "created", Transform: rr.Template != "", LabelFilters: req.Labels})
	}
	return res, nil
}

// updateRecipeWebhook re-applies a recipe to the webhook already pointing at
// its destination: the subscriptions for req.Events get the recipe's
// transform and labels (missing ones are created), then the webhook gets the
// recipe's headers. Subscriptions to other events are left alone.
func updateRecipeWebhook(ctx context.Context, client *apiClient, consumer string, r recipe, rr renderedRecipe, hook webhookOut, req applyRequest, res appliedRecipe) (appliedRecipe, error) {
	res.WebhookID, res.Action = hook.WebhookID, "updated"
	if req.DryRun {
		res.Action = "would_update"
	}
	subs, err := client.listSubscriptions(ctx, consumer, hook.WebhookID)
	if err != nil {
		return res, fmt.Errorf("list subscriptions of webhook %s: %w", hook.WebhookID, err)
	}
	byEvent := make(map[string]subscriptionItem, len(subs))
	for _, s := range subs {
		byEvent[s.EventName] = s
	}
	for _, e := range req.Events {
		row := appliedSubscription{EventName: e, Transform: rr.Template != "", LabelFilters: req.Labels}
		sub, ok := byEvent[e]
		switch {
		case req.DryRun && ok:
			row.SubscriptionID, row.Action = sub.SubscriptionID, "would_update"
		case req.DryRun:
			row.Action = "would_create"
		case ok:
			if err := client.patchSubscription(ctx, consumer, sub.SubscriptionID, recipePatch(rr, req.Labels)); err != nil {
				return res, fmt.Errorf("update subscription %s: %w", sub.SubscriptionID, err)
			}
			row.SubscriptionID, row.Action = sub.SubscriptionID, "updated"
		default:
			id, err := client.createSubscription(ctx, consumer, hook.WebhookID, e, rr.Template, req.Labels)
			if err != nil {
				return res, fmt.Errorf("subscribe webhook %s to %s: %w", hook.WebhookID, e, err)
			}
			row.SubscriptionID, row.Action = id, "created"
		}
		res.Subscriptions = append(res.Subscriptions, row)
	}
	if req.DryRun {
		return res, nil
	}
	// After the subscriptions: requires_transform can only be turned on once
	// every subscription of the webhook has a template.
	patch := map[string]any{"headers": rr.Headers, "description": recipeDescription(r)}
	if len(rr.SecretHeaders) > 0 {
		patch["secret_headers"] = rr.SecretHeaders
	}
	if r.Webhook.RequiresTransform {
		patch["requires_transform"] = true
	}
	if err := client.patchWebhook(ctx, consumer, hook.WebhookID, patch); err != nil {
		return res, fmt.Errorf("update webhook %s: %w", hook.WebhookID, err)
	}
	return res, nil
}

// applyWarnings drops the built-in catalog's authoring rules (description,
// guidance, param prompt and help) from a validation report: they matter for
// shipping a recipe, not for applying one. Warnings about what gets
// delivered stay.
func applyWarnings(ws []string) []string {
	out := []string{}
	for _, w := range ws {
		if strings.HasPrefix(w, "description is empty") || strings.HasPrefix(w, "guidance is empty") || strings.HasPrefix(w, "param ") {
			continue
		}
		out = append(out, w)
	}
	return out
}

func recipeDescription(r recipe) string {
	return fmt.Sprintf("recipe %s: %s", r.Name, r.Description)
}

func recipePatch(rr renderedRecipe, labels map[string]string) subscriptionPatch {
	patch := subscriptionPatch{LabelFilters: labels}
	if rr.Template != "" {
		patch.TransformEnabled = true
		patch.TransformTemplate = rr.Template
	}
	return patch
}

// findWebhookByURL returns the consumer's webhook delivering to rawURL, or
// nil. The server allows one webhook per URL, so a match means applying the
// recipe again would collide.
func findWebhookByURL(ctx context.Context, client *apiClient, consumer, rawURL string) (*webhookOut, error) {
	hooks, err := client.listWebhooks(ctx, consumer, false)
	if err != nil {
		return nil, err
	}
	for _, h := range hooks {
		if h.URL == rawURL {
			return &h, nil
		}
	}
	return nil, nil
}

func (c *apiClient) createSubscription(ctx context.Context, consumer, webhookID, event, tmpl string, labels map[string]string) (string, error) {
	body := map[string]any{"webhook_id": webhookID, "event_name": event}
	if tmpl != "" {
		body["transform_enabled"] = true
		body["transform_template"] = tmpl
	}
	if len(labels) > 0 {
		body["label_filters"] = maps.Clone(labels)
	}
	var out struct {
		SubscriptionID string `json:"subscription_id"`
	}
	err := c.do(ctx, http.MethodPost, "/v1/consumers/"+url.PathEscape(consumer)+"/subscriptions", body, &out)
	return out.SubscriptionID, err
}

func (c *apiClient) patchWebhook(ctx context.Context, consumer, webhookID string, patch map[string]any) error {
	return c.do(ctx, http.MethodPatch, "/v1/consumers/"+url.PathEscape(consumer)+"/webhooks/"+url.PathEscape(webhookID), patch, nil)
}
