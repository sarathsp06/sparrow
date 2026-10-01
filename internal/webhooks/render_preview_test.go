package webhooks

import (
	"strings"
	"testing"
)

func TestRenderTemplatePreview_StrictFlag(t *testing.T) {
	payload := map[string]any{"id": "ord_1"}
	const bad = `{{ .Payload.id }}`

	if out, err := RenderTemplatePreview("order.created", bad, payload, false); err != nil || !strings.Contains(out, "no value") {
		t.Fatalf("lenient: out=%q err=%v", out, err)
	}
	_, err := RenderTemplatePreview("order.created", bad, payload, true)
	if err == nil || !strings.Contains(err.Error(), `"Payload"`) {
		t.Fatalf("strict should fail naming the key: %v", err)
	}
	if out, err := RenderTemplatePreview("order.created", `{{ .payload.id }}-{{ .event_name }}`, payload, true); err != nil || out != "ord_1-order.created" {
		t.Fatalf("strict valid: out=%q err=%v", out, err)
	}
}
