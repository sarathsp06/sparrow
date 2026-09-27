/**
 * Verify Sparrow webhook delivery signatures (Standard Webhooks format).
 *
 * Every delivery carries three headers:
 *
 *     webhook-id:        msg_&lt;delivery-id&gt;
 *     webhook-timestamp: Unix seconds
 *     webhook-signature: space-delimited signatures, e.g. "v1,&lt;base64&gt; v1a,&lt;base64&gt;"
 *
 * The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".
 *
 * - "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
 *   Webhooks format ("whsec_" + base64) are decoded before use; any other
 *   secret is used as raw UTF-8 bytes.
 * - "v1a," is Ed25519 (Java 15+), verified with the hex-encoded public key
 *   from the webhook resource's {@code signing_public_key} field.
 *
 * Copy this file into your project; it has no dependencies beyond the JDK.
 * Place it in any package you like, or use the default package.
 *
 * Usage:
 *
 * <pre>{@code
 *     import sparrow.verify.SparrowVerify;
 *     import sparrow.verify.SparrowVerify.SignatureVerificationException;
 *
 *     try {
 *         SparrowVerify.verifyHmac(request.getBody(), request.getHeaders(), secret);
 *     } catch (SignatureVerificationException e) {
 *         response.setStatus(401);
 *     }
 * }</pre>
 */
package sparrow.verify;

import java.nio.charset.StandardCharsets;
import java.security.KeyFactory;
import java.security.MessageDigest;
import java.security.Signature;
import java.security.spec.X509EncodedKeySpec;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.util.Map;
import javax.crypto.Mac;
import javax.crypto.spec.SecretKeySpec;

public final class SparrowVerify {

    public static final int DEFAULT_TOLERANCE_SECONDS = 5 * 60;

    private SparrowVerify() {}

    /** Thrown when a delivery signature cannot be verified. */
    public static final class SignatureVerificationException extends Exception {
        public SignatureVerificationException(String message) { super(message); }
        public SignatureVerificationException(String message, Throwable cause) { super(message, cause); }
    }

    // ---- public API (Map<String, String> headers) ----

    /**
     * Verify the "v1," (HMAC-SHA256) signature. Throws on failure.
     *
     * @param payload raw request body bytes, exactly as received
     * @param headers request headers (case-insensitive lookup)
     * @param secret  webhook secret, with or without the "whsec_" prefix
     */
    public static void verifyHmac(byte[] payload, Map<String, String> headers,
            String secret) throws SignatureVerificationException {
        verifyHmac(payload, headers, secret, DEFAULT_TOLERANCE_SECONDS, System.currentTimeMillis() / 1000);
    }

    /**
     * Verify the "v1," (HMAC-SHA256) signature with explicit tolerance and clock.
     */
    public static void verifyHmac(byte[] payload, Map<String, String> headers,
            String secret, int toleranceSeconds, long nowSeconds) throws SignatureVerificationException {
        String[] parsed = requiredHeaders(headers);
        checkTimestamp(parsed[1], toleranceSeconds, nowSeconds);

        byte[] key;
        if (secret.startsWith("whsec_")) {
            String encoded = secret.substring("whsec_".length());
            try {
                key = Base64.getDecoder().decode(encoded);
            } catch (IllegalArgumentException e) {
                throw new SignatureVerificationException("invalid whsec_ secret", e);
            }
        } else {
            key = secret.getBytes(StandardCharsets.UTF_8);
        }
        if (key.length == 0) {
            throw new SignatureVerificationException("webhook secret is empty");
        }

        byte[] message = signedMessage(parsed[0], parsed[1], payload);
        byte[] expected;
        try {
            Mac mac = Mac.getInstance("HmacSHA256");
            mac.init(new SecretKeySpec(key, "HmacSHA256"));
            expected = mac.doFinal(message);
        } catch (Exception e) {
            throw new SignatureVerificationException("HMAC computation failed", e);
        }

        for (byte[] candidate : decodeCandidates(parsed[2], "v1,")) {
            if (MessageDigest.isEqual(candidate, expected)) {
                return;
            }
        }
        throw new SignatureVerificationException("no matching v1 (HMAC-SHA256) signature");
    }

    /**
     * Verify the "v1a," (Ed25519) signature. Throws on failure.
     *
     * @param payload      raw request body bytes, exactly as received
     * @param headers      request headers (case-insensitive lookup)
     * @param publicKeyHex hex-encoded key from the webhook resource's signing_public_key field
     */
    public static void verifyEd25519(byte[] payload, Map<String, String> headers,
            String publicKeyHex) throws SignatureVerificationException {
        verifyEd25519(payload, headers, publicKeyHex, DEFAULT_TOLERANCE_SECONDS, System.currentTimeMillis() / 1000);
    }

    /**
     * Verify the "v1a," (Ed25519) signature with explicit tolerance and clock.
     */
    public static void verifyEd25519(byte[] payload, Map<String, String> headers,
            String publicKeyHex, int toleranceSeconds, long nowSeconds) throws SignatureVerificationException {
        String[] parsed = requiredHeaders(headers);
        checkTimestamp(parsed[1], toleranceSeconds, nowSeconds);

        byte[] rawKey = hexDecode(publicKeyHex);
        if (rawKey == null || rawKey.length != 32) {
            throw new SignatureVerificationException("invalid hex public key");
        }

        // Build X.509 SubjectPublicKeyInfo DER: prefix + 32 raw bytes
        byte[] prefix = hexDecode("302a300506032b6570032100");
        byte[] der = new byte[prefix.length + rawKey.length];
        System.arraycopy(prefix, 0, der, 0, prefix.length);
        System.arraycopy(rawKey, 0, der, prefix.length, rawKey.length);

        java.security.PublicKey pubKey;
        try {
            pubKey = KeyFactory.getInstance("Ed25519").generatePublic(new X509EncodedKeySpec(der));
        } catch (Exception e) {
            throw new SignatureVerificationException("invalid Ed25519 public key", e);
        }

        byte[] message = signedMessage(parsed[0], parsed[1], payload);
        for (byte[] candidate : decodeCandidates(parsed[2], "v1a,")) {
            try {
                Signature sig = Signature.getInstance("Ed25519");
                sig.initVerify(pubKey);
                sig.update(message);
                if (sig.verify(candidate)) {
                    return;
                }
            } catch (Exception e) {
                // invalid signature bytes — skip
            }
        }
        throw new SignatureVerificationException("no matching v1a (Ed25519) signature");
    }

    // ---- internals ----

    private static String[] requiredHeaders(Map<String, String> headers) throws SignatureVerificationException {
        String msgId = null, timestamp = null, signature = null;
        for (Map.Entry<String, String> e : headers.entrySet()) {
            String lower = e.getKey().toLowerCase(java.util.Locale.ROOT);
            if (lower.equals("webhook-id")) msgId = e.getValue();
            else if (lower.equals("webhook-timestamp")) timestamp = e.getValue();
            else if (lower.equals("webhook-signature")) signature = e.getValue();
        }
        if (msgId == null || timestamp == null || signature == null) {
            throw new SignatureVerificationException("missing webhook-id, webhook-timestamp, or webhook-signature header");
        }
        return new String[]{ msgId, timestamp, signature };
    }

    private static void checkTimestamp(String timestamp, int toleranceSeconds, long nowSeconds)
            throws SignatureVerificationException {
        // Must be optional minus sign followed by ASCII digits only
        String digits = timestamp;
        if (digits.startsWith("-")) digits = digits.substring(1);
        if (digits.isEmpty()) {
            throw new SignatureVerificationException("invalid webhook-timestamp: " + timestamp);
        }
        for (int i = 0; i < digits.length(); i++) {
            char c = digits.charAt(i);
            if (c < '0' || c > '9') {
                throw new SignatureVerificationException("invalid webhook-timestamp: " + timestamp);
            }
        }
        long seconds = Long.parseLong(timestamp);
        long skew = Math.abs(nowSeconds - seconds);
        if (skew > toleranceSeconds) {
            throw new SignatureVerificationException("webhook-timestamp outside tolerance (possible replay)");
        }
    }

    private static byte[] signedMessage(String msgId, String timestamp, byte[] payload) {
        byte[] prefix = (msgId + "." + timestamp + ".").getBytes(StandardCharsets.UTF_8);
        byte[] result = new byte[prefix.length + payload.length];
        System.arraycopy(prefix, 0, result, 0, prefix.length);
        System.arraycopy(payload, 0, result, prefix.length, payload.length);
        return result;
    }

    private static List<byte[]> decodeCandidates(String sigHeader, String prefix) {
        List<byte[]> result = new ArrayList<>();
        for (String part : sigHeader.split("\\s+")) {
            if (part.isEmpty() || !part.startsWith(prefix)) continue;
            String b64 = part.substring(prefix.length());
            try {
                byte[] decoded = Base64.getDecoder().decode(b64);
                result.add(decoded);
            } catch (IllegalArgumentException e) {
                // invalid base64 — skip
            }
        }
        return result;
    }

    private static byte[] hexDecode(String hex) {
        if (hex.length() % 2 != 0) return null;
        try {
            byte[] result = new byte[hex.length() / 2];
            for (int i = 0; i < result.length; i++) {
                int hi = Character.digit(hex.charAt(i * 2), 16);
                int lo = Character.digit(hex.charAt(i * 2 + 1), 16);
                if (hi < 0 || lo < 0) return null;
                result[i] = (byte) ((hi << 4) | lo);
            }
            return result;
        } catch (Exception e) {
            return null;
        }
    }
}
