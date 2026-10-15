package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Event type tools. Event types are tenant-wide (every consumer shares them),
// unlike the rest of the MCP tools, which act on one consumer.
//
// There is deliberately no infer_schema tool: the UI infers a schema in the
// browser (web/src/lib/schema-infer.ts), and a Go copy would drift from it.
// An agent reads real payloads with list_events, writes the schema itself, and
// saves it with update_event_type; the server derives the sample payload that
// test_template and validate_recipe render against from that schema.

type getEventTypeIn struct {
	Name     string `json:"name" jsonschema:"event type name, e.g. order.created"`
	Versions bool   `json:"versions,omitempty" jsonschema:"also return every version of the type, newest first"`
}

type listEventsIn struct {
	Event       string            `json:"event,omitempty" jsonschema:"only events of this type, e.g. order.created"`
	SchemaValid string            `json:"schema_valid,omitempty" jsonschema:"only events whose payload was valid or invalid against the schema at push time: valid or invalid"`
	Labels      map[string]string `json:"labels,omitempty" jsonschema:"only events carrying all of these label key/values"`
	Limit       int               `json:"limit,omitempty" jsonschema:"max events, default 20, max 1000"`
	Cursor      string            `json:"cursor,omitempty" jsonschema:"pagination.next_cursor from the previous call, to fetch the next (older) page"`
}

type exportEventTypesIn struct {
	Names  []string `json:"names,omitempty" jsonschema:"event type names to export; give exactly one of names, prefix or all"`
	Prefix string   `json:"prefix,omitempty" jsonschema:"export every event type whose name starts with this, e.g. order."`
	All    bool     `json:"all,omitempty" jsonschema:"export every event type"`
}

type importEventTypesIn struct {
	Bundle                map[string]any `json:"bundle" jsonschema:"the bundle object exactly as export_event_types (or sparrow events export) returned it"`
	DryRun                *bool          `json:"dry_run,omitempty" jsonschema:"preview only, the default; set false to write"`
	AllowBreaking         bool           `json:"allow_breaking,omitempty" jsonschema:"apply schema changes that are breaking for existing subscriptions"`
	AcceptVersionMismatch bool           `json:"accept_version_mismatch,omitempty" jsonschema:"import a bundle exported by a different Sparrow version or format"`
	AcceptEdited          bool           `json:"accept_edited,omitempty" jsonschema:"import a bundle whose items changed after it was exported"`
	PauseAffected         bool           `json:"pause_affected,omitempty" jsonschema:"pause subscriptions whose template fails against the new schema instead of leaving them to fail"`
}

type updateEventTypeIn struct {
	Name          string            `json:"name" jsonschema:"event type to change; it must already exist (push_event creates one)"`
	EventSchema   map[string]any    `json:"event_schema,omitempty" jsonschema:"new JSON Schema for the payload; a changed schema creates a new version and regenerates the sample payload"`
	Description   string            `json:"description,omitempty" jsonschema:"new human-readable description; empty leaves it unchanged"`
	Metadata      map[string]string `json:"metadata,omitempty" jsonschema:"replace the whole metadata map"`
	Active        *bool             `json:"active,omitempty" jsonschema:"false retires the type (pushes then fail with 409); true re-enables it"`
	AllowBreaking bool              `json:"allow_breaking,omitempty" jsonschema:"apply a schema change that is breaking for subscriptions receiving this type"`
}

func addEventTypeTools(srv *mcp.Server, client *apiClient, consumer string, allowWrite bool) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true}
	c := url.PathEscape(consumer)
	call := func(ctx context.Context, method, path string, body any) (any, error) {
		var out any
		err := client.do(ctx, method, path, body, &out)
		return dropSchema(out), err
	}

	mcp.AddTool(srv, &mcp.Tool{Name: "get_event_type", Annotations: ro,
		Description: "Get one event type: description, JSON schema, sample payload and current version, optionally with its version history. Lighter than list_event_types when you know the name."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in getEventTypeIn) (*mcp.CallToolResult, any, error) {
			path := "/v1/event-types/" + url.PathEscape(in.Name)
			et, err := call(ctx, http.MethodGet, path, nil)
			if err != nil || !in.Versions {
				return nil, et, err
			}
			versions, err := call(ctx, http.MethodGet, path+"/versions", nil)
			if err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{"event_type": et, "versions": versions}, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_events", Annotations: ro,
		Description: "List events pushed to the consumer, newest first, with payload, labels, schema_valid and delivery counts. Read several payloads of one type to write its JSON schema (fields seen in every event are required), then save it with update_event_type. When pagination.has_more is true, pass pagination.next_cursor as cursor."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in listEventsIn) (*mcp.CallToolResult, any, error) {
			if in.Limit <= 0 {
				in.Limit = 20
			}
			q := url.Values{"limit": {fmt.Sprint(in.Limit)}}
			for k, v := range map[string]string{"event": in.Event, "schema_valid": in.SchemaValid, "cursor": in.Cursor, "labels": labelQuery(in.Labels)} {
				if v != "" {
					q.Set(k, v)
				}
			}
			out, err := call(ctx, http.MethodGet, "/v1/consumers/"+c+"/events?"+q.Encode(), nil)
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "export_event_types", Annotations: ro,
		Description: "Export event type definitions (schema, description, metadata, active) as one bundle, like `sparrow events export`. Pick types with exactly one of names, prefix or all. Pass the bundle unchanged to import_event_types on another Sparrow, e.g. to promote types from development to production."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in exportEventTypesIn) (*mcp.CallToolResult, any, error) {
			sel, err := exportSelection(in.Names, in.Prefix, in.All)
			if err != nil {
				return nil, nil, err
			}
			var bundle map[string]any
			if err := client.do(ctx, http.MethodPost, "/v1/event-types:export", sel, &bundle); err != nil {
				return nil, nil, err
			}
			return nil, dropSchema(bundle), nil
		})

	if !allowWrite {
		return
	}

	mcp.AddTool(srv, &mcp.Tool{Name: "import_event_types",
		Description: "Import an event type bundle, like `sparrow events import`. A dry run by default: it returns, per type, the action (new, new_version, unchanged), schema compatibility and which subscriptions' templates would fail against the new sample, and writes nothing. Set dry_run=false to apply; everything is applied in one transaction or nothing is. blocked_by names what stops it (version_differs, items_changed, breaking) and the flag that accepts it. Event types are shared by every consumer."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in importEventTypesIn) (*mcp.CallToolResult, any, error) {
			if _, ok := in.Bundle["items"]; !ok {
				return nil, nil, errors.New("bundle has no items: pass the object export_event_types returned")
			}
			body := make(map[string]any, len(in.Bundle)+4)
			for k, v := range in.Bundle {
				body[k] = v
			}
			addImportOptions(body, importFlags{
				dryRun:                in.DryRun == nil || *in.DryRun,
				allowBreaking:         in.AllowBreaking,
				acceptVersionMismatch: in.AcceptVersionMismatch,
				acceptEdited:          in.AcceptEdited,
				pauseAffected:         in.PauseAffected,
			})
			out, err := call(ctx, http.MethodPost, "/v1/event-types:import", body)
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "update_event_type",
		Description: "Change an event type's JSON schema, description, metadata or active flag; fields left out stay as they are. A changed schema creates a new version, and the server regenerates the sample payload from it, which is what test_template renders against: give an event type created by push_event a schema this way before writing templates for it. The sample's values are generated (placeholder text and numbers; enum values are used as given), so to render a template against a real payload use validate_recipe with payload. A breaking change to a type subscriptions receive is refused (409) unless allow_breaking is set. Event types are shared by every consumer."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in updateEventTypeIn) (*mcp.CallToolResult, any, error) {
			body := map[string]any{}
			if in.EventSchema != nil {
				body["event_schema"] = in.EventSchema
			}
			if in.Description != "" {
				body["description"] = in.Description
			}
			if in.Metadata != nil {
				body["metadata"] = in.Metadata
			}
			if in.Active != nil {
				body["active"] = *in.Active
			}
			if len(body) == 0 {
				return nil, nil, errors.New("nothing to change: give event_schema, description, metadata or active")
			}
			path := "/v1/event-types/" + url.PathEscape(in.Name)
			if in.AllowBreaking {
				path += "?allow_breaking=true"
			}
			out, err := call(ctx, http.MethodPatch, path, body)
			return nil, out, err
		})
}

// labelQuery renders label filters as the API's sorted k=v,k=v form.
func labelQuery(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	pairs := make([]string, 0, len(labels))
	for k, v := range labels {
		pairs = append(pairs, k+"="+v)
	}
	sort.Strings(pairs)
	return strings.Join(pairs, ",")
}
