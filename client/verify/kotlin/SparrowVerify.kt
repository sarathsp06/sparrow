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
 *   secret is used as raw UTF-8 bytes.
 * - "v1a," is Ed25519 (Java 15+), verified with the hex-encoded public key
 *   from the webhook resource's `signing_public_key` field.
 *
 * Copy this file into your project; it has no dependencies beyond the JDK.
 *
 * Usage:
 *
 * ```kotlin
 *     import sparrow.verify.SparrowVerify
 *     import sparrow.verify.SparrowVerify.SignatureVerificationException
 *
 *     try {
 *         SparrowVerify.verifyHmac(request.body, request.headers, secret)
 *     } catch (e: SignatureVerificationException) {
 *         response.status = 401
 *     }
 * ```
 */
package sparrow.verify

import java.nio.charset.StandardCharsets
import java.security.KeyFactory
import java.security.MessageDigest
import java.security.Signature
import java.security.spec.X509EncodedKeySpec
import java.util.Base64
import java.util.Locale
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec

object SparrowVerify {

    const val DEFAULT_TOLERANCE_SECONDS: Int = 5 * 60

    /** Thrown when a delivery signature cannot be verified. */
    class SignatureVerificationException : Exception {
        constructor(message: String) : super(message)
        constructor(message: String, cause: Throwable) : super(message, cause)
    }

    /**
     * Verify the "v1," (HMAC-SHA256) signature. Throws on failure.
     *
     * @param payload raw request body bytes, exactly as received
     * @param headers request headers (case-insensitive lookup)
     * @param secret  webhook secret, with or without the "whsec_" prefix
     */
    @JvmStatic
    @JvmOverloads
    @Throws(SignatureVerificationException::class)
    fun verifyHmac(
        payload: ByteArray,
        headers: Map<String, String>,
        secret: String,
        toleranceSeconds: Int = DEFAULT_TOLERANCE_SECONDS,
        nowSeconds: Long = System.currentTimeMillis() / 1000
    ) {
        val (msgId, timestamp, sigHeader) = requiredHeaders(headers)
        checkTimestamp(timestamp, toleranceSeconds, nowSeconds)

        val key: ByteArray = if (secret.startsWith("whsec_")) {
            val encoded = secret.removePrefix("whsec_")
            try {
                Base64.getDecoder().decode(encoded)
            } catch (e: IllegalArgumentException) {
                throw SignatureVerificationException("invalid whsec_ secret", e)
            }
        } else {
            secret.toByteArray(StandardCharsets.UTF_8)
        }
        if (key.isEmpty()) {
            throw SignatureVerificationException("webhook secret is empty")
        }

        val message = signedMessage(msgId, timestamp, payload)
        val mac = Mac.getInstance("HmacSHA256")
        mac.init(SecretKeySpec(key, "HmacSHA256"))
        val expected = mac.doFinal(message)

        for (candidate in decodeCandidates(sigHeader, "v1,")) {
            if (MessageDigest.isEqual(candidate, expected)) {
                return
            }
        }
        throw SignatureVerificationException("no matching v1 (HMAC-SHA256) signature")
    }

    /**
     * Verify the "v1a," (Ed25519) signature. Throws on failure.
     *
     * @param payload      raw request body bytes, exactly as received
     * @param headers      request headers (case-insensitive lookup)
     * @param publicKeyHex hex-encoded key from the webhook resource's signing_public_key field
     */
    @JvmStatic
    @JvmOverloads
    @Throws(SignatureVerificationException::class)
    fun verifyEd25519(
        payload: ByteArray,
        headers: Map<String, String>,
        publicKeyHex: String,
        toleranceSeconds: Int = DEFAULT_TOLERANCE_SECONDS,
        nowSeconds: Long = System.currentTimeMillis() / 1000
    ) {
        val (msgId, timestamp, sigHeader) = requiredHeaders(headers)
        checkTimestamp(timestamp, toleranceSeconds, nowSeconds)

        val rawKey = hexDecode(publicKeyHex)
            ?: throw SignatureVerificationException("invalid hex public key")
        if (rawKey.size != 32) {
            throw SignatureVerificationException("invalid hex public key")
        }

        // Build X.509 SubjectPublicKeyInfo DER: prefix + 32 raw bytes
        val prefix = hexDecode("302a300506032b6570032100")!!
        val der = prefix + rawKey

        val pubKey = try {
            KeyFactory.getInstance("Ed25519").generatePublic(X509EncodedKeySpec(der))
        } catch (e: Exception) {
            throw SignatureVerificationException("invalid Ed25519 public key", e)
        }

        val message = signedMessage(msgId, timestamp, payload)
        for (candidate in decodeCandidates(sigHeader, "v1a,")) {
            try {
                val sig = Signature.getInstance("Ed25519")
                sig.initVerify(pubKey)
                sig.update(message)
                if (sig.verify(candidate)) {
                    return
                }
            } catch (_: Exception) {
                // invalid signature bytes — skip
            }
        }
        throw SignatureVerificationException("no matching v1a (Ed25519) signature")
    }

    // ---- internals ----

    private data class ParsedHeaders(val msgId: String, val timestamp: String, val sigHeader: String)

    private fun requiredHeaders(headers: Map<String, String>): ParsedHeaders {
        var msgId: String? = null
        var timestamp: String? = null
        var signature: String? = null
        for ((k, v) in headers) {
            when (k.lowercase(Locale.ROOT)) {
                "webhook-id" -> msgId = v
                "webhook-timestamp" -> timestamp = v
                "webhook-signature" -> signature = v
            }
        }
        if (msgId == null || timestamp == null || signature == null) {
            throw SignatureVerificationException(
                "missing webhook-id, webhook-timestamp, or webhook-signature header"
            )
        }
        return ParsedHeaders(msgId, timestamp, signature)
    }

    private fun checkTimestamp(timestamp: String, toleranceSeconds: Int, nowSeconds: Long) {
        val digits = if (timestamp.startsWith("-")) timestamp.substring(1) else timestamp
        if (digits.isEmpty() || digits.any { it < '0' || it > '9' }) {
            throw SignatureVerificationException("invalid webhook-timestamp: $timestamp")
        }
        val seconds = timestamp.toLong()
        val skew = Math.abs(nowSeconds - seconds)
        if (skew > toleranceSeconds) {
            throw SignatureVerificationException("webhook-timestamp outside tolerance (possible replay)")
        }
    }

    private fun signedMessage(msgId: String, timestamp: String, payload: ByteArray): ByteArray {
        val prefix = "$msgId.$timestamp.".toByteArray(StandardCharsets.UTF_8)
        return prefix + payload
    }

    private fun decodeCandidates(sigHeader: String, prefix: String): List<ByteArray> {
        val result = mutableListOf<ByteArray>()
        for (part in sigHeader.trim().split(Regex("\\s+"))) {
            if (part.isEmpty() || !part.startsWith(prefix)) continue
            val b64 = part.removePrefix(prefix)
            try {
                result.add(Base64.getDecoder().decode(b64))
            } catch (_: IllegalArgumentException) {
                // invalid base64 — skip
            }
        }
        return result
    }

    private fun hexDecode(hex: String): ByteArray? {
        if (hex.length % 2 != 0) return null
        return try {
            ByteArray(hex.length / 2) { i ->
                val hi = Character.digit(hex[i * 2], 16)
                val lo = Character.digit(hex[i * 2 + 1], 16)
                if (hi < 0 || lo < 0) return null
                ((hi shl 4) or lo).toByte()
            }
        } catch (_: Exception) {
            null
        }
    }
}
