/**
 * Verify Sparrow webhook delivery signatures (Standard Webhooks format).
 *
 * Every delivery carries three headers:
 *
 *     webhook-id:        msg_<delivery-id>
 *     webhook-timestamp: Unix seconds
 *     webhook-signature: space-delimited signatures, e.g. "v1,<base64> v1a,<base64>"
 *
 * The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".
 *
 * - "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
 *   Webhooks format ("whsec_" + base64) are decoded before use; any other
 *   secret is used as raw bytes.
 * - "v1a," is Ed25519, verified with the hex-encoded public key from the
 *   webhook resource's `signing_public_key` field.
 *
 * Uses node:crypto — works on Node.js >= 16 and Bun. Copy this file into
 * your project or vendor it as-is; it has no other dependencies.
 *
 * Usage:
 *
 *     import { verifyHmac, verifyEd25519, SignatureVerificationError } from "./sparrow-verify";
 *
 *     // payload must be the raw request body (string or bytes), not re-serialized JSON.
 *     try {
 *       verifyHmac(rawBody, req.headers, secret);
 *     } catch (err) {
 *       res.status(401).end();
 *     }
 */

import {
  createHmac,
  createPublicKey,
  timingSafeEqual,
  verify as cryptoVerify,
} from "node:crypto";

export const DEFAULT_TOLERANCE_SECONDS = 5 * 60;

export class SignatureVerificationError extends Error {}

export type Headers = Record<string, string | string[] | undefined>;

/**
 * Verify the "v1," (HMAC-SHA256) signature. Throws SignatureVerificationError on failure.
 *
 * @param payload raw request body, exactly as received
 * @param headers request headers (case-insensitive lookup)
 * @param secret webhook secret, with or without the "whsec_" prefix
 */
export function verifyHmac(
  payload: string | Uint8Array,
  headers: Headers,
  secret: string,
  toleranceSeconds: number = DEFAULT_TOLERANCE_SECONDS,
): void {
  const { msgId, timestamp, signatures } = parseHeaders(headers, toleranceSeconds);

  const key = secret.startsWith("whsec_")
    ? Buffer.from(secret.slice("whsec_".length), "base64")
    : Buffer.from(secret);

  const message = Buffer.concat([
    Buffer.from(`${msgId}.${timestamp}.`),
    typeof payload === "string" ? Buffer.from(payload) : Buffer.from(payload),
  ]);
  const expected = createHmac("sha256", key).update(message).digest();

  for (const candidate of decodeSignatures(signatures, "v1,")) {
    if (candidate.length === expected.length && timingSafeEqual(candidate, expected)) {
      return;
    }
  }
  throw new SignatureVerificationError("no matching v1 (HMAC-SHA256) signature");
}

/**
 * Verify the "v1a," (Ed25519) signature. Throws SignatureVerificationError on failure.
 *
 * @param payload raw request body, exactly as received
 * @param headers request headers (case-insensitive lookup)
 * @param publicKeyHex hex-encoded key from the webhook resource's `signing_public_key` field
 */
export function verifyEd25519(
  payload: string | Uint8Array,
  headers: Headers,
  publicKeyHex: string,
  toleranceSeconds: number = DEFAULT_TOLERANCE_SECONDS,
): void {
  const { msgId, timestamp, signatures } = parseHeaders(headers, toleranceSeconds);

  const raw = Buffer.from(publicKeyHex, "hex");
  if (raw.length !== 32 || raw.toString("hex") !== publicKeyHex.toLowerCase()) {
    throw new SignatureVerificationError("invalid hex public key");
  }
  const publicKey = createPublicKey({
    key: { kty: "OKP", crv: "Ed25519", x: raw.toString("base64url") },
    format: "jwk",
  });

  const message = Buffer.concat([
    Buffer.from(`${msgId}.${timestamp}.`),
    typeof payload === "string" ? Buffer.from(payload) : Buffer.from(payload),
  ]);
  for (const candidate of decodeSignatures(signatures, "v1a,")) {
    if (cryptoVerify(null, message, publicKey, candidate)) {
      return;
    }
  }
  throw new SignatureVerificationError("no matching v1a (Ed25519) signature");
}


function parseHeaders(
  headers: Headers,
  toleranceSeconds: number,
): { msgId: string; timestamp: string; signatures: string[] } {
  const get = (name: string): string => {
    for (const [key, value] of Object.entries(headers)) {
      if (key.toLowerCase() === name && value !== undefined) {
        return Array.isArray(value) ? value[0] : value;
      }
    }
    throw new SignatureVerificationError(`missing header: ${name}`);
  };

  const msgId = get("webhook-id");
  const timestamp = get("webhook-timestamp");
  const signatureHeader = get("webhook-signature");

  const seconds = Number(timestamp);
  if (!Number.isInteger(seconds)) {
    throw new SignatureVerificationError(`invalid webhook-timestamp: ${timestamp}`);
  }
  if (Math.abs(Date.now() / 1000 - seconds) > toleranceSeconds) {
    throw new SignatureVerificationError(
      "webhook-timestamp outside tolerance (possible replay)",
    );
  }

  return { msgId, timestamp, signatures: signatureHeader.split(/\s+/) };
}

function* decodeSignatures(signatures: string[], prefix: string): Generator<Buffer> {
  for (const part of signatures) {
    if (!part.startsWith(prefix)) continue;
    const decoded = Buffer.from(part.slice(prefix.length), "base64");
    if (decoded.length > 0) yield decoded;
  }
}
