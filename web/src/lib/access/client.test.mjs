import assert from "node:assert/strict";
import test from "node:test";

import { browserTokenName, exchangeKey, fragmentParam, InviteError, redeemInvite, rejectMessage, rejectReason } from "./client.ts";

/** A fetch stub that records calls and answers with status/body. */
function stub(status, body) {
  const calls = [];
  const fn = async (url, init = {}) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });
  };
  return { fn, calls };
}

test("fragmentParam reads and decodes fragment values", () => {
  assert.equal(fragmentParam("#invite=sparrow_inv_abc", "invite"), "sparrow_inv_abc");
  assert.equal(fragmentParam("#x=1&token=a%2Bb", "token"), "a+b");
  assert.equal(fragmentParam("#token=abc", "invite"), "");
  assert.equal(fragmentParam("", "invite"), "");
});

test("rejectReason only accepts known reasons", () => {
  assert.equal(rejectReason({ reason: "revoked" }), "revoked");
  assert.equal(rejectReason({ reason: "haxx" }), undefined);
  assert.equal(rejectReason(null), undefined);
});

test("rejectMessage explains each case", () => {
  assert.match(rejectMessage("revoked", true), /revoked/);
  assert.match(rejectMessage("expired", true), /expired/);
  assert.match(rejectMessage(undefined, false), /requires an API key/);
  assert.match(rejectMessage(undefined, true), /rejected/);
});

test("browserTokenName labels common browsers", () => {
  const mac = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36";
  assert.equal(browserTokenName(mac), "Browser sign-in (Chrome on macOS)");
  assert.equal(browserTokenName("Mozilla/5.0 (X11; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0"), "Browser sign-in (Firefox on Linux)");
  assert.equal(browserTokenName(""), "Browser sign-in (browser)");
});

test("redeemInvite posts the invite and returns the token", async () => {
  const { fn, calls } = stub(200, { token: "sparrow_tk_x", token_id: "tok_1", name: "bob", scope: null, expires_at: null });
  const res = await redeemInvite("https://api.example.com", "sparrow_inv_1", fn);
  assert.equal(res.token, "sparrow_tk_x");
  assert.equal(calls[0].url, "https://api.example.com/invite/redeem");
  assert.deepEqual(JSON.parse(calls[0].init.body), { invite: "sparrow_inv_1" });
});

test("redeemInvite: a spent invite is an InviteError, an outage is not", async () => {
  await assert.rejects(redeemInvite("", "x", stub(400, { error: "invalid_invite" }).fn), InviteError);
  await assert.rejects(redeemInvite("", "x", stub(500, {}).fn), (e) => !(e instanceof InviteError));
});

test("exchangeKey swaps a master key for a browser token", async () => {
  const { fn, calls } = stub(201, { secret: "sparrow_tk_new", token: { id: "tok_9" } });
  const res = await exchangeKey("", "master-key", "Browser sign-in (x)", fn);
  assert.deepEqual(res, { kind: "token", secret: "sparrow_tk_new", tokenId: "tok_9" });
  assert.equal(calls[0].url, "/v1/tokens");
  assert.equal(calls[0].init.headers["X-API-Key"], "master-key");
  assert.deepEqual(JSON.parse(calls[0].init.body), { name: "Browser sign-in (x)" });
});

test("exchangeKey reports a wrong key with its reason", async () => {
  assert.deepEqual(await exchangeKey("", "bad", "n", stub(401, { reason: "invalid" }).fn), { kind: "rejected", reason: "invalid" });
});

test("exchangeKey keeps a pasted token after checking it", async () => {
  const ok = stub(200, { auth_enabled: true });
  assert.deepEqual(await exchangeKey("", "sparrow_tk_abc", "n", ok.fn), { kind: "keep" });
  assert.equal(ok.calls[0].url, "/v1/whoami");
  assert.deepEqual(await exchangeKey("", "sparrow_tk_abc", "n", stub(401, { reason: "revoked" }).fn), { kind: "rejected", reason: "revoked" });
});

test("exchangeKey keeps the key when the server cannot mint tokens", async () => {
  assert.deepEqual(await exchangeKey("", "master-key", "n", stub(404, {}).fn), { kind: "keep" });
});
