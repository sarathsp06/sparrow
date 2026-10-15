import assert from "node:assert/strict";
import test from "node:test";

import { groupByPrefix, hasSchema, schemaFields } from "./schema-fields.ts";

test("flattens nested objects and arrays of objects", () => {
  const rows = schemaFields({
    type: "object",
    required: ["id"],
    properties: {
      id: { type: "string", format: "uuid", description: "Order id" },
      customer: { type: "object", required: ["email"], properties: { email: { type: "string" } } },
      items: { type: "array", items: { type: "object", properties: { sku: { type: "string" } } } },
      tags: { type: "array", items: { type: "string" } },
      status: { enum: ["paid", "refunded"] },
    },
  });
  assert.deepEqual(rows.map((r) => [r.path, r.depth, r.type, r.required]), [
    ["id", 0, "string (uuid)", true],
    ["customer", 0, "object", false],
    ["customer.email", 1, "string", true],
    ["items", 0, "object[]", false],
    ["items[].sku", 1, "string", false],
    ["tags", 0, "string[]", false],
    ["status", 0, "enum", false],
  ]);
  assert.equal(rows[0].description, "Order id");
  assert.deepEqual(rows[6].enumValues, ["paid", "refunded"]);
});

test("refs, unions and junk degrade to plain types", () => {
  const rows = schemaFields({
    properties: {
      a: { $ref: "#/$defs/Money" },
      b: { oneOf: [{ type: "string" }, { type: "null" }] },
      c: { type: ["integer", "null"] },
      d: {},
      e: "not a schema",
    },
  });
  assert.deepEqual(rows.map((r) => [r.path, r.type]), [
    ["a", "Money"],
    ["b", "string | null"],
    ["c", "integer | null"],
    ["d", "any"],
  ]);
  assert.deepEqual(schemaFields(null), []);
  assert.deepEqual(schemaFields([1]), []);
});

test("hasSchema treats empty objects as no schema", () => {
  assert.equal(hasSchema({}), false);
  assert.equal(hasSchema(undefined), false);
  assert.equal(hasSchema({ type: "object" }), true);
});

test("groupByPrefix keeps first-seen order, ungrouped last", () => {
  const g = groupByPrefix([{ name: "order.created" }, { name: "ping" }, { name: "order.paid" }, { name: "invoice.sent" }]);
  assert.deepEqual(g.map((x) => [x.prefix, x.items.map((i) => i.name)]), [
    ["order", ["order.created", "order.paid"]],
    ["invoice", ["invoice.sent"]],
    ["", ["ping"]],
  ]);
});
