package receivercode

import "encoding/json"

// testPublicKey is a real-format Ed25519 public key for the ed25519 case.
const testPublicKey = "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29"

const orderSchema = `{
  "type": "object",
  "required": ["id", "amount", "customer"],
  "properties": {
    "id": {"type": "string"},
    "amount": {"type": "number"},
    "quantity": {"type": "integer"},
    "paid": {"type": "boolean"},
    "customer": {
      "type": "object",
      "required": ["email"],
      "properties": {
        "email": {"type": "string"},
        "name": {"type": ["string", "null"]}
      }
    },
    "items": {"type": "array", "items": {"$ref": "#/$defs/item"}},
    "metadata": {"type": "object", "additionalProperties": {"type": "string"}},
    "extra": {}
  },
  "$defs": {
    "item": {
      "type": "object",
      "required": ["sku"],
      "properties": {"sku": {"type": "string"}, "price": {"type": "number"}}
    }
  }
}`

const orderSample = `{
  "id": "ord_123",
  "amount": 42.5,
  "quantity": 2,
  "paid": true,
  "customer": {"email": "ada@example.com", "name": null},
  "items": [{"sku": "A-1", "price": 20}, {"sku": "B-2", "price": 22.5}],
  "metadata": {"source": "web"},
  "extra": {"nested": [1, "two"]}
}`

// testCases are the golden cases: every target is generated for each.
var testCases = []struct {
	name string
	in   Input
}{
	{"envelope_json", Input{
		Webhook: Webhook{ID: "wh_envelope", Consumer: "acme", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_order_paid", EventType: "order.paid", EventVersion: 2,
			Schema: json.RawMessage(orderSchema), Sample: json.RawMessage(orderSample),
		}},
	}},
	{"transform_json", Input{
		Webhook: Webhook{ID: "wh_slack", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_slack", EventType: "order.paid", EventVersion: 1,
			Sample:            json.RawMessage(orderSample),
			Transform:         `{"text": "Order {{.payload.id}} paid", "amount": {{.payload.amount}}, "blocks": [{"type": "section", "fields": {{json .payload.items}}}], "channel": null}`,
			StrictMissingKeys: true,
		}},
	}},
	{"transform_xml", Input{
		Webhook: Webhook{ID: "wh_xml", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_xml", EventType: "order.paid",
			Sample:    json.RawMessage(orderSample),
			Transform: `<order id="{{.payload.id}}" xmlns="urn:example"><amount currency="EUR">{{.payload.amount}}</amount>{{range .payload.items}}<item><sku>{{.sku}}</sku></item>{{end}}<note>paid</note></order>`,
			Headers:   map[string]string{"content-type": "application/xml"},
		}},
	}},
	{"transform_text", Input{
		Webhook: Webhook{ID: "wh_text", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_text", EventType: "order.paid",
			Sample:    json.RawMessage(orderSample),
			Transform: `Order {{.payload.id}} paid {{.payload.amount}} EUR at {{now | formatTime "2006-01-02T15:04:05Z07:00"}}`,
			Headers:   map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		}},
	}},
	{"conditional", Input{
		Webhook: Webhook{ID: "wh_cond", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_cond", EventType: "order.paid",
			Sample:    json.RawMessage(orderSample),
			Transform: `{{if .payload.paid}}{"status": "paid", "id": "{{.payload.id}}"}{{else}}{"status": "open"}{{end}}`,
		}},
	}},
	{"ed25519", Input{
		Webhook: Webhook{ID: "wh_ed25519", SigningScheme: SchemeEd25519, Ed25519PublicKey: testPublicKey},
		Subscriptions: []Subscription{{
			ID: "sub_signed", EventType: "user.created",
			Sample: json.RawMessage(`{"user_id": 7, "email": "ada@example.com", "roles": ["admin"]}`),
		}},
	}},
	{"schema_only", Input{
		Webhook: Webhook{ID: "wh_schema_only", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{{
			ID: "sub_schema", EventType: "order.paid", EventVersion: 1,
			Schema: json.RawMessage(orderSchema),
		}},
	}},
	{"multi", Input{
		Webhook: Webhook{ID: "wh_multi", Consumer: "globex", SigningScheme: SchemeHMAC},
		Subscriptions: []Subscription{
			{ID: "sub_b", EventType: "order.paid", Sample: json.RawMessage(orderSample),
				Transform: `{"order": "{{.payload.id}}", "total": {{.payload.amount}}}`},
			{ID: "sub_a", EventType: "order.paid", Schema: json.RawMessage(orderSchema), Sample: json.RawMessage(orderSample)},
			{ID: "sub_c", EventType: "invoice.failed-v2", EventVersion: 3,
				Sample: json.RawMessage(`{"invoice": "in_1", "attempts": [1, 2.5], "reason": null}`)},
			{ID: "sub_d", EventType: "invoice.voided", Sample: json.RawMessage(`{"invoice": "in_2"}`),
				Transform: `{{.payload.missing.deeper}}`, StrictMissingKeys: true},
		},
	}},
}
