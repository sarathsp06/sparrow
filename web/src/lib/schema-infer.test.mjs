import assert from "node:assert/strict";
import test from "node:test";

import { analyzeSamples, detectStringFormat, inferSchema, jsonToJsonSchema } from "./schema-infer.ts";

test("a single sample makes every property required", () => {
  assert.deepEqual(jsonToJsonSchema({ id: 7, name: "a", ok: true, tags: ["x"], meta: null }), {
    type: "object",
    properties: {
      id: { type: "integer" },
      name: { type: "string" },
      ok: { type: "boolean" },
      tags: { type: "array", items: { type: "string" } },
      meta: { type: "null" },
    },
    required: ["id", "name", "ok", "tags", "meta"],
  });
});

test("a property missing from some samples is optional", () => {
  const schema = inferSchema([{ a: 1, b: 2 }, { a: 3 }]);
  assert.deepEqual(schema.required, ["a"]);
  assert.deepEqual(Object.keys(schema.properties), ["a", "b"]);
});

test("types seen across samples are unioned", () => {
  const schema = inferSchema([{ v: "x", n: 1 }, { v: null, n: 1.5 }]);
  assert.deepEqual(schema.properties.v, { type: ["string", "null"] });
  assert.deepEqual(schema.properties.n, { type: "number" });
});

test("array items merge every element, not just the first", () => {
  const schema = jsonToJsonSchema({ items: [{ sku: "a", qty: 1 }, { sku: "b" }] });
  assert.deepEqual(schema.properties.items.items, {
    type: "object",
    properties: { sku: { type: "string" }, qty: { type: "integer" } },
    required: ["sku"],
  });
});

test("an empty array has no items schema", () => {
  assert.deepEqual(jsonToJsonSchema([]), { type: "array" });
});

test("string formats are detected", () => {
  assert.equal(detectStringFormat("3f2b8c1e-9a4d-4e2f-8b7a-1c2d3e4f5a6b"), "uuid");
  assert.equal(detectStringFormat("2026-10-05T12:30:00Z"), "date-time");
  assert.equal(detectStringFormat("2026-10-05T12:30:00.123+05:30"), "date-time");
  assert.equal(detectStringFormat("2026-10-05"), "date");
  assert.equal(detectStringFormat("jane.doe+test@example.co.uk"), "email");
  assert.equal(detectStringFormat("https://example.com/hooks?x=1"), "uri");
  assert.equal(detectStringFormat("10.0.0.255"), "ipv4");
});

test("near misses get no format", () => {
  for (const s of ["2026-13-05", "2026-10-05 12:30:00", "not an email@", "a@b", "example.com", "256.1.1.1", "12345", ""]) {
    assert.equal(detectStringFormat(s), undefined, s);
  }
});

test("a format is kept only when every sample agrees", () => {
  const same = inferSchema([{ at: "2026-10-05T12:30:00Z" }, { at: "2026-10-06T08:00:00Z" }]);
  assert.equal(same.properties.at.format, "date-time");
  const mixed = inferSchema([{ at: "2026-10-05T12:30:00Z" }, { at: "yesterday" }]);
  assert.equal(mixed.properties.at.format, undefined);
});

test("no samples accepts anything", () => {
  assert.deepEqual(inferSchema([]), {});
});

test("field coverage counts how often each property appeared", () => {
  const { fields } = analyzeSamples([
    { id: "3f2b8c1e-9a4d-4e2f-8b7a-1c2d3e4f5a6b", lines: [{ sku: "a", qty: 1 }, { sku: "b" }], note: "x" },
    { id: "9c1d2e3f-4a5b-4c6d-8e7f-0a1b2c3d4e5f", lines: [], note: null },
  ]);
  assert.deepEqual(fields, [
    { path: "id", types: ["string"], format: "uuid", seen: 2, of: 2, required: true },
    { path: "lines", types: ["array"], format: undefined, seen: 2, of: 2, required: true },
    { path: "lines[].sku", types: ["string"], format: undefined, seen: 2, of: 2, required: true },
    { path: "lines[].qty", types: ["integer"], format: undefined, seen: 1, of: 2, required: false },
    { path: "note", types: ["string", "null"], format: undefined, seen: 2, of: 2, required: true },
  ]);
});
