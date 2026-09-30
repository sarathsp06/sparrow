package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// maxBundleBytes caps an import request body. The item count is capped
// separately (webhooks.MaxBundleItems).
const maxBundleBytes = 5 << 20

// bundleItem is one event type definition in a bundle file. It carries no
// version number, timestamps or sample payload: those differ per environment
// and would show a false difference on every promotion.
type bundleItem struct {
	Name        string            `json:"name" minLength:"1" maxLength:"255" doc:"Event type name."`
	Description string            `json:"description,omitempty" doc:"Human-readable summary. Omitted means empty: an import replaces the definition."`
	JSONSchema  map[string]any    `json:"event_schema,omitempty" doc:"JSON Schema. Omitted means no schema, which removes an existing schema (a breaking change)."`
	Metadata    map[string]string `json:"metadata,omitempty" doc:"Arbitrary key/value metadata. Omitted means none."`
	Active      *bool             `json:"active,omitempty" doc:"Whether events of this type can be pushed. Omitted means true."`
}

type bundleStampBody struct {
	SparrowVersion string `json:"sparrow_version" doc:"Version of the Sparrow server that exported the bundle."`
	Format         int    `json:"format" doc:"Bundle format number."`
	SHA256         string `json:"sha256" doc:"sha256 of the canonical encoding of items, to tell whether they changed after export."`
}

func toBundleItems(defs []webhooks.EventTypeDefinition) []bundleItem {
	out := make([]bundleItem, 0, len(defs))
	for _, d := range defs {
		active := d.Active
		out = append(out, bundleItem{Name: d.Name, Description: d.Description, JSONSchema: d.Schema, Metadata: d.Metadata, Active: &active})
	}
	return out
}

func fromBundleItems(items []bundleItem) []webhooks.EventTypeDefinition {
	out := make([]webhooks.EventTypeDefinition, 0, len(items))
	for _, it := range items {
		active := true
		if it.Active != nil {
			active = *it.Active
		}
		out = append(out, webhooks.EventTypeDefinition{Name: it.Name, Description: it.Description, Schema: it.JSONSchema, Metadata: it.Metadata, Active: active})
	}
	return out
}

type exportEventTypesInput struct {
	Body struct {
		Names  []string `json:"names,omitempty" doc:"Export these event types. Each must exist."`
		Prefix string   `json:"prefix,omitempty" doc:"Export every event type whose name starts with this prefix."`
		All    bool     `json:"all,omitempty" doc:"Export every event type."`
	}
}

type eventTypeBundle struct {
	APIVersion string          `json:"apiVersion" enum:"sparrow/v1" doc:"Bundle schema version."`
	Kind       string          `json:"kind" enum:"EventTypeList" doc:"Bundle kind."`
	Stamp      bundleStampBody `json:"stamp" doc:"Which Sparrow produced this bundle. A compatibility hint, not a signature."`
	Items      []bundleItem    `json:"items" doc:"Event type definitions, sorted by name."`
}

type exportEventTypesOutput struct {
	Body eventTypeBundle
}

type importEventTypesInput struct {
	Body struct {
		APIVersion    string           `json:"apiVersion,omitempty" enum:"sparrow/v1" doc:"Bundle schema version, if the body is an exported bundle."`
		Kind          string           `json:"kind,omitempty" enum:"EventTypeList" doc:"Bundle kind, if the body is an exported bundle."`
		Stamp         *bundleStampBody `json:"stamp,omitempty" doc:"The bundle's stamp. Absent for a hand-written bundle, which imports with a notice."`
		Items         []bundleItem     `json:"items" minItems:"1" maxItems:"500" doc:"Definitions to import. Each replaces the named event type; types not listed are never touched."`
		DryRun        bool             `json:"dry_run,omitempty" doc:"Compute the full result without writing anything."`
		Acknowledge   []string         `json:"acknowledge,omitempty" enum:"version_differs,format_unsupported,items_changed" doc:"Stamp warnings you accept. Each unacknowledged warning blocks the write."`
		AllowBreaking bool             `json:"allow_breaking,omitempty" doc:"Apply breaking schema changes to event types that subscriptions receive."`
	}
}

type schemaCompatibilityOut = schemaCompatibilityItem

type templateFailureItem struct {
	SubscriptionID string `json:"subscription_id" doc:"Subscription whose template failed."`
	Consumer       string `json:"consumer" doc:"Consumer the subscription belongs to."`
	WebhookID      string `json:"webhook_id" doc:"Webhook the subscription delivers to."`
	Payload        string `json:"payload" enum:"sample,required_only" doc:"Which generated payload failed: every field, or only required fields (a template reading an optional field without guarding for it)."`
	Error          string `json:"error" doc:"The render error."`
}

type templateCheckItem struct {
	Failures         []templateFailureItem `json:"failures,omitempty" doc:"Subscriptions whose transform no longer renders against the new version."`
	Passed           int                   `json:"passed" doc:"Subscriptions whose transform renders."`
	WithoutTransform int                   `json:"without_transform" doc:"Subscriptions with no transform. Sparrow passes the payload through and cannot tell whether the receiver copes."`
	CatchAll         int                   `json:"catch_all" doc:"How many of the affected subscriptions are catch-all (\"*\")."`
}

type importItemResult struct {
	Name            string                  `json:"name" doc:"Event type name."`
	Action          string                  `json:"action" enum:"created,new_version,updated,unchanged" doc:"What the import does to this event type. See updateEventType's change.action."`
	Version         int                     `json:"version" doc:"Version after the import (or that it would have, for a dry run)."`
	PreviousVersion int                     `json:"previous_version,omitempty" doc:"Version before the import."`
	Changes         []string                `json:"changes,omitempty" doc:"Fields that change: schema, schema_defined, description, metadata, active."`
	ActiveChange    string                  `json:"active_change,omitempty" enum:"deactivates,reactivates," doc:"Set when the import flips the active flag."`
	Compatibility   *schemaCompatibilityOut `json:"compatibility,omitempty" doc:"For new_version: whether the change could break a subscription's payload transformation."`
	Blocked         bool                    `json:"blocked,omitempty" doc:"The change is breaking for subscriptions and allow_breaking was not set."`
	Subscriptions   *templateCheckItem      `json:"subscriptions,omitempty" doc:"For new_version: every affected subscription's transform rendered strictly against the new version."`
}

type stampCheckItem struct {
	Status     string   `json:"status" enum:"valid,unsigned,warning" doc:"valid: stamp present and matching. unsigned: no stamp (hand-written file). warning: see warnings."`
	ExportedBy string   `json:"exported_by,omitempty" doc:"Sparrow version that exported the file."`
	Server     string   `json:"server" doc:"This server's Sparrow version."`
	Warnings   []string `json:"warnings,omitempty" doc:"version_differs, format_unsupported, items_changed. Each must be acknowledged for the import to write."`
}

type importEventTypesOutput struct {
	Body struct {
		Applied    bool               `json:"applied" doc:"True when the import was written. False for a dry run or when blocked_by is non-empty."`
		DryRun     bool               `json:"dry_run" doc:"Whether this was a dry run."`
		ImportedAt string             `json:"imported_at,omitempty" doc:"When the import was written, RFC3339. Use it to find deliveries that failed after the change."`
		Stamp      stampCheckItem     `json:"stamp" doc:"The bundle's stamp compared with this server."`
		Items      []importItemResult `json:"items" doc:"One result per bundle item, sorted by name."`
		BlockedBy  []string           `json:"blocked_by,omitempty" doc:"Why nothing was written: unacknowledged stamp warnings, and breaking when a breaking change needs allow_breaking."`
	}
}

func registerEventBundleRoutes(api huma.API, svc eventRouteService) {
	huma.Register(api, huma.Operation{
		OperationID: "exportEventTypes",
		Method:      http.MethodPost,
		Path:        "/v1/event-types:export",
		Summary:     "Export event type definitions as a bundle",
		Description: "Returns the current definitions of the selected event types as one JSON bundle, ready to import into another environment with importEventTypes. Choose exactly one of names, prefix or all. An unknown name fails the whole export. Sparrow's own sparrow.* event types are never exported.",
		Errors:      []int{400, 404},
		Tags:        []string{"Event Types"},
	}, func(ctx context.Context, in *exportEventTypesInput) (*exportEventTypesOutput, error) {
		defs, stamp, err := svc.ExportEventTypes(ctx, webhooks.EventTypeExportSelection{Names: in.Body.Names, Prefix: in.Body.Prefix, All: in.Body.All})
		if err != nil {
			return nil, mapError(ctx, err, "failed to export event types")
		}
		return &exportEventTypesOutput{Body: eventTypeBundle{
			APIVersion: webhooks.BundleAPIVersion,
			Kind:       webhooks.BundleKind,
			Stamp:      bundleStampBody{SparrowVersion: stamp.SparrowVersion, Format: stamp.Format, SHA256: stamp.SHA256},
			Items:      toBundleItems(defs),
		}}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:  "importEventTypes",
		Method:       http.MethodPost,
		Path:         "/v1/event-types:import",
		Summary:      "Import a bundle of event type definitions",
		Description:  "Applies a bundle in one transaction: each item replaces the named event type using the usual version rules, or nothing is written. Event types not in the bundle are never touched; an import does not delete. The full per-item result is returned either way. Nothing is written for a dry run, when a stamp warning is not acknowledged, or when a change is breaking for subscriptions and allow_breaking is not set; blocked_by says why. The body may be an exported bundle as-is, plus the options.",
		Errors:       []int{400},
		Tags:         []string{"Event Types"},
		MaxBodyBytes: maxBundleBytes,
	}, func(ctx context.Context, in *importEventTypesInput) (*importEventTypesOutput, error) {
		var stamp *webhooks.BundleStamp
		if in.Body.Stamp != nil {
			stamp = &webhooks.BundleStamp{SparrowVersion: in.Body.Stamp.SparrowVersion, Format: in.Body.Stamp.Format, SHA256: in.Body.Stamp.SHA256}
		}
		res, err := svc.ImportEventTypes(ctx, fromBundleItems(in.Body.Items), stamp, webhooks.EventTypeImportOptions{
			DryRun:        in.Body.DryRun,
			Acknowledge:   in.Body.Acknowledge,
			AllowBreaking: in.Body.AllowBreaking,
		})
		if err != nil {
			return nil, mapError(ctx, err, "failed to import event types")
		}
		return toImportOutput(res), nil
	})
}

func toImportOutput(res *webhooks.EventTypeImportResult) *importEventTypesOutput {
	out := &importEventTypesOutput{}
	out.Body.Applied = res.Applied
	out.Body.DryRun = res.DryRun
	if res.Applied {
		out.Body.ImportedAt = res.ImportedAt.Format(time.RFC3339Nano)
	}
	out.Body.Stamp = stampCheckItem{Status: res.Stamp.Status, ExportedBy: res.Stamp.ExportedBy, Server: res.Stamp.Server, Warnings: res.Stamp.Warnings}
	out.Body.BlockedBy = res.BlockedBy
	out.Body.Items = make([]importItemResult, 0, len(res.Items))
	for _, it := range res.Items {
		item := importItemResult{
			Name:            it.Name,
			Action:          string(it.Action),
			Version:         it.Version,
			PreviousVersion: it.PreviousVersion,
			Changes:         it.Changes,
			ActiveChange:    it.ActiveChange,
			Blocked:         it.Blocked,
		}
		if it.Compatibility != nil {
			item.Compatibility = &schemaCompatibilityOut{Result: it.Compatibility.Result(), Reasons: it.Compatibility.Reasons}
		}
		if t := it.Templates; t != nil {
			sc := &templateCheckItem{Passed: t.Passed, WithoutTransform: t.WithoutTransform, CatchAll: t.CatchAll}
			for _, f := range t.Failures {
				sc.Failures = append(sc.Failures, templateFailureItem{SubscriptionID: f.SubscriptionID, Consumer: f.Consumer, WebhookID: f.WebhookID, Payload: f.Payload, Error: f.Error})
			}
			item.Subscriptions = sc
		}
		out.Body.Items = append(out.Body.Items, item)
	}
	return out
}
