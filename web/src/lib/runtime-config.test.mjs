import assert from "node:assert/strict";
import test from "node:test";

import { apiHref, portalGatewayURL, resolveApiBase } from "./runtime-config.ts";

test("resolveApiBase: runtime config beats build-time PUBLIC_API_URL", () => {
  assert.equal(resolveApiBase("https://rt.example.com", "https://build.example.com", false), "https://rt.example.com");
  assert.equal(resolveApiBase(undefined, "https://build.example.com", false), "https://build.example.com");
  assert.equal(resolveApiBase("", "https://build.example.com", false), "https://build.example.com");
});

test("resolveApiBase: defaults to localhost:8080 in dev and same origin in builds", () => {
  assert.equal(resolveApiBase(undefined, undefined, true), "http://localhost:8080");
  assert.equal(resolveApiBase(undefined, undefined, false), "");
  // The embedded build sets PUBLIC_API_URL=/ which means same origin.
  assert.equal(resolveApiBase(undefined, "/", false), "");
});

test("resolveApiBase: strips trailing slashes and whitespace", () => {
  assert.equal(resolveApiBase(" https://api.example.com/ ", undefined, false), "https://api.example.com");
  assert.equal(resolveApiBase("https://host/sparrow-api//", undefined, false), "https://host/sparrow-api");
});

test("apiHref points server pages at the API host", () => {
  assert.equal(apiHref("", "/docs"), "/docs");
  assert.equal(apiHref("https://api.example.com", "/docs"), "https://api.example.com/docs");
  assert.equal(apiHref("https://host/sparrow-api", "/docs"), "https://host/sparrow-api/docs");
});

const UI = "https://ui.example.com/portal";

test("portalGatewayURL: same origin", () => {
  assert.equal(
    portalGatewayURL("https://ui.example.com/v1/consumers/acme/webhooks?limit=5", "", "acme", UI),
    "https://ui.example.com/portal/api/webhooks?limit=5",
  );
  assert.equal(portalGatewayURL("https://ui.example.com/v1/event-types", "", "acme", UI), "https://ui.example.com/portal/api/event-types");
});

test("portalGatewayURL: unknown consumer (bare #token= link) still routes by the token", () => {
  assert.equal(
    portalGatewayURL("https://ui.example.com/v1/consumers//webhooks", "", "", UI),
    "https://ui.example.com/portal/api/webhooks",
  );
});

test("portalGatewayURL: cross-origin API keeps the API host", () => {
  assert.equal(
    portalGatewayURL("https://api.example.com/v1/consumers/acme/webhooks", "https://api.example.com", "acme", UI),
    "https://api.example.com/portal/api/webhooks",
  );
});

test("portalGatewayURL: API under a path prefix", () => {
  assert.equal(
    portalGatewayURL("https://host/sparrow-api/v1/consumers/acme/deliveries", "https://host/sparrow-api", "acme", UI),
    "https://host/sparrow-api/portal/api/deliveries",
  );
});

test("portalGatewayURL: percent-encoded consumer", () => {
  assert.equal(
    portalGatewayURL("https://api.example.com/v1/consumers/team%20a/webhooks", "https://api.example.com", "team a", UI),
    "https://api.example.com/portal/api/webhooks",
  );
});

test("portalGatewayURL: leaves non-API URLs alone", () => {
  assert.equal(portalGatewayURL("https://api.example.com/health", "https://api.example.com", "acme", UI), null);
});
