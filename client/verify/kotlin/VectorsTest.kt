package sparrow.verify

import java.io.File
import java.nio.charset.StandardCharsets
import java.util.Base64

/**
 * Plain-main test runner for SparrowVerify against vectors.tsv.
 * No test framework — exit code 0 on all pass, 1 on any failure.
 */
fun main(args: Array<String>) {
    val tsvPath = if (args.isNotEmpty()) args[0] else "pkg/signature/testdata/vectors.tsv"

    val lines = File(tsvPath).readLines(StandardCharsets.UTF_8)
    val headerLine = lines.firstOrNull()
        ?: error("empty vectors.tsv")
    require(headerLine.startsWith("# now ")) { "invalid vectors.tsv header" }

    // Parse "# now <n> tolerance <n>"
    val parts = headerLine.split(Regex("\\s+"))
    val now = parts[2].toLong()
    val tolerance = parts[4].toInt()

    var passed = 0
    var failed = 0

    for (line in lines.drop(1)) {
        if (line.isEmpty()) continue

        val cols = line.split("\t", limit = 6)
        if (cols.size < 6) {
            println("SKIP (malformed): $line")
            continue
        }

        val name = cols[0]
        val scheme = cols[1]
        val expectedValid = cols[2] == "1"
        val keyB64 = cols[3]
        val payloadB64 = cols[4]
        val headersEncoded = cols[5]

        // Decode key: base64 -> string
        val key = if (keyB64.isEmpty()) "" else String(Base64.getDecoder().decode(keyB64), StandardCharsets.UTF_8)
        // Decode payload: base64 -> bytes (may be empty)
        val payload = if (payloadB64.isEmpty()) ByteArray(0) else Base64.getDecoder().decode(payloadB64)
        // Decode headers: comma-separated base64(name):base64(value)
        val headers = linkedMapOf<String, String>()
        if (headersEncoded.isNotEmpty()) {
            for (pair in headersEncoded.split(",")) {
                val (hNameB64, hValueB64) = pair.split(":", limit = 2)
                val hName = String(Base64.getDecoder().decode(hNameB64), StandardCharsets.UTF_8)
                val hValue = String(Base64.getDecoder().decode(hValueB64), StandardCharsets.UTF_8)
                headers[hName] = hValue
            }
        }

        val actualValid: Boolean = try {
            if (scheme == "hmac") {
                SparrowVerify.verifyHmac(payload, headers, key, tolerance, now)
            } else {
                SparrowVerify.verifyEd25519(payload, headers, key, tolerance, now)
            }
            true
        } catch (_: SparrowVerify.SignatureVerificationException) {
            false
        }

        if (actualValid == expectedValid) {
            println("PASS  $name")
            passed++
        } else {
            println("FAIL  $name  (expected ${if (expectedValid) "valid" else "invalid"}, got ${if (actualValid) "valid" else "invalid"})")
            failed++
        }
    }

    println()
    println("$passed passed, $failed failed, ${passed + failed} total")
    if (failed > 0) {
        System.exit(1)
    }
}
