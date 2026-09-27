// Run the shared signature vectors (pkg/signature/testdata/vectors.json):
//
//   node --test --experimental-strip-types client/verify/js/vectors.test.ts
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { verifyEd25519, verifyHmac, SignatureVerificationError } from "./sparrow-verify.ts";

const path = fileURLToPath(new URL("../../../pkg/signature/testdata/vectors.json", import.meta.url));
const data = JSON.parse(readFileSync(path, "utf8"));

for (const c of data.cases) {
  test(c.name, () => {
    const payload = Buffer.from(c.payload_b64, "base64");
    const verify = c.scheme === "hmac" ? verifyHmac : verifyEd25519;
    let ok = true;
    try {
      verify(payload, c.headers, c.key, data.tolerance_seconds, data.now);
    } catch (err) {
      if (!(err instanceof SignatureVerificationError)) throw err;
      ok = false;
    }
    assert.equal(ok, c.valid);
  });
}
