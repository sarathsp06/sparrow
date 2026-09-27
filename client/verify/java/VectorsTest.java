package sparrow.verify;

import java.io.BufferedReader;
import java.io.FileReader;
import java.nio.charset.StandardCharsets;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Plain-main test runner for SparrowVerify against vectors.tsv.
 * No JUnit dependency — exit code 0 on all pass, 1 on any failure.
 */
public class VectorsTest {

    public static void main(String[] args) throws Exception {
        String tsvPath = args.length > 0 ? args[0] : "pkg/signature/testdata/vectors.tsv";

        BufferedReader reader = new BufferedReader(new FileReader(tsvPath, StandardCharsets.UTF_8));
        String headerLine = reader.readLine();
        if (headerLine == null || !headerLine.startsWith("# now ")) {
            System.err.println("ERROR: invalid vectors.tsv header");
            System.exit(1);
        }

        // Parse "# now <n> tolerance <n>"
        String[] parts = headerLine.split("\\s+");
        long now = Long.parseLong(parts[2]);
        int tolerance = Integer.parseInt(parts[4]);

        int passed = 0;
        int failed = 0;
        String line;
        while ((line = reader.readLine()) != null) {
            if (line.isEmpty()) continue;

            String[] cols = line.split("\t", -1);
            if (cols.length < 6) {
                System.out.println("SKIP (malformed): " + line);
                continue;
            }

            String name = cols[0];
            String scheme = cols[1];
            boolean expectedValid = cols[2].equals("1");
            String keyB64 = cols[3];
            String payloadB64 = cols[4];
            String headersEncoded = cols[5];

            // Decode key: base64 → string
            String key = keyB64.isEmpty() ? "" : new String(Base64.getDecoder().decode(keyB64), StandardCharsets.UTF_8);
            // Decode payload: base64 → bytes (may be empty)
            byte[] payload = payloadB64.isEmpty() ? new byte[0] : Base64.getDecoder().decode(payloadB64);
            // Decode headers: comma-separated base64(name):base64(value)
            Map<String, String> headers = new LinkedHashMap<>();
            if (!headersEncoded.isEmpty()) {
                for (String pair : headersEncoded.split(",")) {
                    String[] kv = pair.split(":", 2);
                    String hName = new String(Base64.getDecoder().decode(kv[0]), StandardCharsets.UTF_8);
                    String hValue = new String(Base64.getDecoder().decode(kv[1]), StandardCharsets.UTF_8);
                    headers.put(hName, hValue);
                }
            }

            boolean actualValid;
            try {
                if (scheme.equals("hmac")) {
                    SparrowVerify.verifyHmac(payload, headers, key, tolerance, now);
                } else {
                    SparrowVerify.verifyEd25519(payload, headers, key, tolerance, now);
                }
                actualValid = true;
            } catch (SparrowVerify.SignatureVerificationException e) {
                actualValid = false;
            }

            if (actualValid == expectedValid) {
                System.out.println("PASS  " + name);
                passed++;
            } else {
                System.out.println("FAIL  " + name + "  (expected " + (expectedValid ? "valid" : "invalid") + ", got " + (actualValid ? "valid" : "invalid") + ")");
                failed++;
            }
        }
        reader.close();

        System.out.println();
        System.out.println(passed + " passed, " + failed + " failed, " + (passed + failed) + " total");
        if (failed > 0) {
            System.exit(1);
        }
    }
}
