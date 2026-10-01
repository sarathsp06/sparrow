package webhooks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

func TestBundleDigest_IgnoresFormattingAndKeyOrder(t *testing.T) {
	a := []EventTypeDefinition{{Name: "order.created", Active: true, Schema: js(t, `{"type":"object","required":["id"],"properties":{"id":{"type":"string"}}}`)}}
	// Same schema, different key order and whitespace in the source.
	b := []EventTypeDefinition{{Name: "order.created", Active: true, Schema: js(t, `{
		"properties": {"id": {"type": "string"}},
		"required": ["id"],
		"type": "object"
	}`)}}
	assert.Equal(t, BundleDigest(a), BundleDigest(b))

	// Nil and empty metadata or schema digest the same.
	c := []EventTypeDefinition{{Name: "x", Active: true, Metadata: map[string]string{}, Schema: map[string]any{}}}
	d := []EventTypeDefinition{{Name: "x", Active: true}}
	assert.Equal(t, BundleDigest(c), BundleDigest(d))

	// Any real change does not.
	e := []EventTypeDefinition{{Name: "order.created", Active: false, Schema: a[0].Schema}}
	assert.NotEqual(t, BundleDigest(a), BundleDigest(e))
}

func TestCheckStamp(t *testing.T) {
	items := []EventTypeDefinition{{Name: "a.b", Active: true}}
	good := &BundleStamp{SparrowVersion: sparrow.Version, Format: BundleFormat, SHA256: BundleDigest(items)}

	check := checkStamp(nil, items)
	assert.Equal(t, "unsigned", check.Status)
	assert.Empty(t, check.Warnings)

	check = checkStamp(good, items)
	assert.Equal(t, "valid", check.Status)
	assert.Empty(t, check.Warnings)

	older := *good
	older.SparrowVersion = "0.0.1"
	check = checkStamp(&older, items)
	assert.Equal(t, "warning", check.Status)
	assert.Equal(t, []string{StampVersionDiffers}, check.Warnings)
	assert.Equal(t, "0.0.1", check.ExportedBy)

	newer := *good
	newer.Format = BundleFormat + 1
	assert.Equal(t, []string{StampFormatUnsupported}, checkStamp(&newer, items).Warnings)

	edited := []EventTypeDefinition{{Name: "a.b", Active: false}}
	assert.Equal(t, []string{StampItemsChanged}, checkStamp(good, edited).Warnings)
}

func TestValidateBundle(t *testing.T) {
	assertStatus(t, validateBundle(nil), svcerrors.InvalidArgument)

	many := make([]EventTypeDefinition, MaxBundleItems+1)
	for i := range many {
		many[i] = EventTypeDefinition{Name: uuid.NewString()}
	}
	err := validateBundle(many)
	assertStatus(t, err, svcerrors.InvalidArgument)
	assert.Contains(t, err.Error(), "at most 500")

	err = validateBundle([]EventTypeDefinition{
		{Name: "ok.one"},
		{Name: "ok.one"},
		{Name: "sparrow.webhook.health_changed"},
		{Name: "bad.schema", Schema: map[string]any{"type": 12}},
		{Name: strings.Repeat("x", 256)},
	})
	assertStatus(t, err, svcerrors.InvalidArgument)
	msg := err.Error()
	assert.Contains(t, msg, `"ok.one": appears more than once`)
	assert.Contains(t, msg, "reserved prefix")
	assert.Contains(t, msg, `"bad.schema": event_schema does not compile`)
	assert.Contains(t, msg, "at most 255 characters")
	assert.Contains(t, msg, "nothing was imported")

	require.NoError(t, validateBundle([]EventTypeDefinition{{Name: "a.b", Schema: orderSchema("number")}}))
}

func TestCheckTemplates(t *testing.T) {
	def := EventTypeDefinition{Name: "order.created", Schema: js(t, `{
		"type": "object",
		"required": ["id"],
		"properties": {"id": {"type": "string"}, "coupon": {"type": "string"}}
	}`)}
	sub := func(event, tmpl string) *store.EventSubscription {
		return &store.EventSubscription{ID: uuid.New(), WebhookID: uuid.New(), Consumer: "shop", EventName: event, TransformEnabled: tmpl != "", TransformTemplate: tmpl}
	}
	subs := []*store.EventSubscription{
		sub("order.created", `{"id": {{.payload.id | json}}}`),               // reads a required field: fine
		sub("order.created", `{"total": {{.payload.total}}}`),                // reads a field the new schema lacks
		sub("order.created", `{"c": {{.payload.coupon | json}}}`),            // reads an optional field unguarded
		sub("order.created", `{"c": {{ dig "coupon" "" .payload | json }}}`), // reads it safely
		sub("order.created", ""),                                             // no transform
		sub(store.CatchAllEventName, `{"e": {{.event_name | json}}}`),        // catch-all, generic
	}

	check := checkTemplates(def, subs)
	assert.Equal(t, 3, check.Passed)
	assert.Equal(t, 1, check.WithoutTransform)
	assert.Equal(t, 1, check.CatchAll)
	require.Len(t, check.Failures, 2)

	byID := map[string]TemplateCheckFailure{}
	for _, f := range check.Failures {
		byID[f.SubscriptionID] = f
	}
	missing := byID[subs[1].ID.String()]
	assert.Equal(t, "sample", missing.Payload)
	assert.Contains(t, missing.Error, "total")
	optional := byID[subs[2].ID.String()]
	assert.Equal(t, "required_only", optional.Payload, "caught only when the optional field is absent")
	assert.Contains(t, optional.Error, "coupon")
}

func TestBundleItemJSONShape(t *testing.T) {
	// The digest encodes definitions as the bundle file does, so re-encoding
	// an exported file's items reproduces the stamp.
	b, err := json.Marshal(bundleItem{Name: "a.b", Active: true})
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"a.b","active":true}`, string(b))
}
