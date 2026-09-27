<?php
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
 *   webhook resource's `signing_public_key` field. Requires the sodium
 *   extension (bundled with PHP 7.2+).
 *
 * Copy this file into your project — it has no Composer dependencies.
 * Requires PHP 8.1+.
 *
 * Usage (Laravel):
 *
 *     // routes/api.php
 *     use Illuminate\Http\Request;
 *
 *     Route::post('/webhook', function (Request $request) {
 *         // IMPORTANT: use $request->getContent(), not $request->json() —
 *         // the raw body must match what was signed.
 *         require_once __DIR__ . '/../lib/SparrowVerify.php';
 *
 *         try {
 *             \Sparrow\Verify\SparrowVerify::verifyHmac(
 *                 $request->getContent(),
 *                 $request->headers->all(),   // or getallheaders()
 *                 config('services.sparrow.secret')
 *             );
 *         } catch (\Sparrow\Verify\SignatureVerificationException $e) {
 *             return response('', 401);
 *         }
 *
 *         // process event ...
 *         return response('', 200);
 *     });
 *
 * Usage (plain PHP):
 *
 *     require_once __DIR__ . '/SparrowVerify.php';
 *
 *     $payload = file_get_contents('php://input');  // raw body
 *     $headers = getallheaders();
 *
 *     try {
 *         \Sparrow\Verify\SparrowVerify::verifyHmac($payload, $headers, $secret);
 *     } catch (\Sparrow\Verify\SignatureVerificationException $e) {
 *         http_response_code(401);
 *         exit;
 *     }
 */

declare(strict_types=1);

namespace Sparrow\Verify;

class SignatureVerificationException extends \RuntimeException {}

class SparrowVerify
{
    public const DEFAULT_TOLERANCE_SECONDS = 300;

    /**
     * Verify the "v1," (HMAC-SHA256) signature. Throws on failure.
     *
     * @param string $payload  Raw request body, exactly as received.
     * @param array  $headers  Request headers (case-insensitive lookup).
     * @param string $secret   Webhook secret, with or without the "whsec_" prefix.
     * @param int    $toleranceSeconds  Maximum clock skew in seconds.
     * @param int|float|null $now  Override the clock (Unix seconds), e.g. in tests.
     */
    public static function verifyHmac(
        string $payload,
        array $headers,
        string $secret,
        int $toleranceSeconds = self::DEFAULT_TOLERANCE_SECONDS,
        int|float|null $now = null,
    ): void {
        [$msgId, $timestamp, $sigHeader] = self::requiredHeaders($headers);
        self::checkTimestamp($timestamp, $toleranceSeconds, $now);

        if (str_starts_with($secret, 'whsec_')) {
            $encoded = substr($secret, 6);
            $key = base64_decode($encoded, true);
            if ($key === false) {
                throw new SignatureVerificationException('invalid whsec_ secret');
            }
        } else {
            $key = $secret;
        }
        if ($key === '') {
            throw new SignatureVerificationException('webhook secret is empty');
        }

        $message = "{$msgId}.{$timestamp}.{$payload}";
        $expected = hash_hmac('sha256', $message, $key, true);

        foreach (self::decodeSignatures($sigHeader, 'v1,') as $candidate) {
            if (hash_equals($expected, $candidate)) {
                return;
            }
        }
        throw new SignatureVerificationException('no matching v1 (HMAC-SHA256) signature');
    }

    /**
     * Verify the "v1a," (Ed25519) signature. Throws on failure.
     *
     * @param string $payload        Raw request body, exactly as received.
     * @param array  $headers        Request headers (case-insensitive lookup).
     * @param string $publicKeyHex   Hex-encoded key from the webhook resource's signing_public_key field.
     * @param int    $toleranceSeconds  Maximum clock skew in seconds.
     * @param int|float|null $now    Override the clock (Unix seconds), e.g. in tests.
     */
    public static function verifyEd25519(
        string $payload,
        array $headers,
        string $publicKeyHex,
        int $toleranceSeconds = self::DEFAULT_TOLERANCE_SECONDS,
        int|float|null $now = null,
    ): void {
        if (!function_exists('sodium_crypto_sign_verify_detached')) {
            throw new SignatureVerificationException(
                'Ed25519 verification requires the sodium extension (bundled with PHP 7.2+)'
            );
        }

        [$msgId, $timestamp, $sigHeader] = self::requiredHeaders($headers);
        self::checkTimestamp($timestamp, $toleranceSeconds, $now);

        $rawKey = @hex2bin($publicKeyHex);
        if ($rawKey === false || strlen($rawKey) !== 32) {
            throw new SignatureVerificationException('invalid hex public key');
        }
        // Verify hex round-trips (rejects mixed-case ambiguity won't matter, but
        // catches truncation from odd-length input that hex2bin silently handles)
        if (bin2hex($rawKey) !== strtolower($publicKeyHex)) {
            throw new SignatureVerificationException('invalid hex public key');
        }

        $message = "{$msgId}.{$timestamp}.{$payload}";

        foreach (self::decodeSignatures($sigHeader, 'v1a,') as $candidate) {
            try {
                if (\sodium_crypto_sign_verify_detached($candidate, $message, $rawKey)) {
                    return;
                }
            } catch (\SodiumException) {
                continue;
            }
        }
        throw new SignatureVerificationException('no matching v1a (Ed25519) signature');
    }

    /**
     * Extract required headers (case-insensitive).
     *
     * @return array{string, string, string}  [msgId, timestamp, sigHeader]
     */
    private static function requiredHeaders(array $headers): array
    {
        $lowered = [];
        foreach ($headers as $k => $v) {
            // Some frameworks pass headers as arrays of values
            $lowered[strtolower((string) $k)] = is_array($v) ? ($v[0] ?? '') : (string) $v;
        }

        $msgId = $lowered['webhook-id'] ?? '';
        $timestamp = $lowered['webhook-timestamp'] ?? '';
        $sigHeader = $lowered['webhook-signature'] ?? '';

        if ($msgId === '' || $timestamp === '' || $sigHeader === '') {
            throw new SignatureVerificationException(
                'missing webhook-id, webhook-timestamp, or webhook-signature header'
            );
        }

        return [$msgId, $timestamp, $sigHeader];
    }

    private static function checkTimestamp(string $timestamp, int $toleranceSeconds, int|float|null $now): void
    {
        if (!preg_match('/\A-?\d+\z/', $timestamp)) {
            throw new SignatureVerificationException("invalid webhook-timestamp: {$timestamp}");
        }
        $seconds = (int) $timestamp;
        $current = $now ?? time();
        if (abs($current - $seconds) > $toleranceSeconds) {
            throw new SignatureVerificationException('webhook-timestamp outside tolerance (possible replay)');
        }
    }

    /**
     * Decode signatures with the given prefix, skipping invalid base64.
     *
     * @return list<string>
     */
    private static function decodeSignatures(string $sigHeader, string $prefix): array
    {
        $results = [];
        foreach (preg_split('/\s+/', $sigHeader, -1, PREG_SPLIT_NO_EMPTY) as $part) {
            if (!str_starts_with($part, $prefix)) {
                continue;
            }
            $encoded = substr($part, strlen($prefix));
            // Strict base64 validation: only A-Za-z0-9+/ and trailing =, length % 4 == 0
            if (!preg_match('#\A[A-Za-z0-9+/]*={0,2}\z#', $encoded) || strlen($encoded) % 4 !== 0) {
                continue;
            }
            $decoded = base64_decode($encoded, true);
            if ($decoded === false || $decoded === '') {
                continue;
            }
            $results[] = $decoded;
        }
        return $results;
    }
}
