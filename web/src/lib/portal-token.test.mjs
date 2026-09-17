import assert from "node:assert/strict";
import test from "node:test";

import { parsePortalToken } from "./portal-token.ts";

test("parsePortalToken rejects legacy v1 tokens", () => {
  assert.equal(parsePortalToken("spt_v1.YWNtZQ.1735689600.signature"), null);
});

test("parsePortalToken supports v2 tokens with key ids", () => {
  const session = parsePortalToken("spt_v2.new-key.YWNtZQ.1735689600.signature");
  assert.deepEqual(session, {
    token: "spt_v2.new-key.YWNtZQ.1735689600.signature",
    consumer: "acme",
    expiresAt: new Date(1735689600 * 1000),
  });
});

test("parsePortalToken rejects malformed tokens", () => {
  assert.equal(parsePortalToken("garbage"), null);
  assert.equal(parsePortalToken("spt_v2.bad"), null);
});
