<?php
/**
 * Run the shared signature vectors (pkg/signature/testdata/vectors.json).
 *
 *     php client/verify/php/test_vectors.php
 */

declare(strict_types=1);

require_once __DIR__ . '/SparrowVerify.php';

use Sparrow\Verify\SparrowVerify;
use Sparrow\Verify\SignatureVerificationException;

$vectorsPath = __DIR__ . '/../../../pkg/signature/testdata/vectors.json';
$data = json_decode(file_get_contents($vectorsPath), true, 512, JSON_THROW_ON_ERROR);

$now = $data['now'];
$tolerance = $data['tolerance_seconds'];
$passed = 0;
$failed = 0;

foreach ($data['cases'] as $case) {
    $payload = base64_decode($case['payload_b64'], true);
    if ($payload === false) {
        $payload = '';
    }

    $ok = true;
    try {
        if ($case['scheme'] === 'hmac') {
            SparrowVerify::verifyHmac($payload, $case['headers'], $case['key'], $tolerance, $now);
        } else {
            SparrowVerify::verifyEd25519($payload, $case['headers'], $case['key'], $tolerance, $now);
        }
    } catch (SignatureVerificationException) {
        $ok = false;
    }

    if ($ok === $case['valid']) {
        echo "PASS  {$case['name']}\n";
        $passed++;
    } else {
        $expected = $case['valid'] ? 'true' : 'false';
        $got = $ok ? 'true' : 'false';
        echo "FAIL  {$case['name']} (expected valid={$expected}, got {$got})\n";
        $failed++;
    }
}

$total = $passed + $failed;
echo "\n{$total} cases: {$passed} passed, {$failed} failed\n";
exit($failed > 0 ? 1 : 0);
