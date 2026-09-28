import assert from "node:assert/strict";
import test from "node:test";

import { parsePortalLink } from "./portal-link.ts";

test("parsePortalLink reads access-token links", () => {
  const session = parsePortalLink("#token=sparrow_tk_abc&consumer=acme%20corp%26co&expires=1735689600");
  assert.deepEqual(session, {
    token: "sparrow_tk_abc",
    consumer: "acme corp&co",
    expiresAt: new Date(1735689600 * 1000),
  });
});

test("parsePortalLink accepts a bare access token (consumer and expiry unknown)", () => {
  assert.deepEqual(parsePortalLink("#token=sparrow_tk_abc"), { token: "sparrow_tk_abc", consumer: "", expiresAt: null });
});

test("parsePortalLink rejects other tokens and fragments", () => {
  assert.equal(parsePortalLink("#token=spt_v2.key.YWNtZQ.1735689600.sig"), null);
  assert.equal(parsePortalLink("#token=garbage&consumer=acme"), null);
  assert.equal(parsePortalLink("#invite=sparrow_inv_abc"), null);
  assert.equal(parsePortalLink(""), null);
});
