"""Run the shared signature vectors (pkg/signature/testdata/vectors.json).

    python3 client/verify/python/test_vectors.py

Ed25519 cases need the ``cryptography`` package; without it they are skipped.
"""

import base64
import json
import pathlib
import sys
import unittest

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

from sparrow_verify import SignatureVerificationError, verify_ed25519, verify_hmac  # noqa: E402

VECTORS = HERE.parent.parent.parent / "pkg" / "signature" / "testdata" / "vectors.json"

try:
    import cryptography  # noqa: F401

    HAVE_CRYPTOGRAPHY = True
except ImportError:  # pragma: no cover
    HAVE_CRYPTOGRAPHY = False


class Vectors(unittest.TestCase):
    def test_vectors(self):
        data = json.loads(VECTORS.read_text())
        for case in data["cases"]:
            with self.subTest(case["name"]):
                if case["scheme"] == "ed25519" and not HAVE_CRYPTOGRAPHY:
                    self.skipTest("cryptography not installed")
                verify = verify_hmac if case["scheme"] == "hmac" else verify_ed25519
                payload = base64.b64decode(case["payload_b64"])
                try:
                    verify(payload, case["headers"], case["key"],
                           tolerance_seconds=data["tolerance_seconds"], now=data["now"])
                    ok = True
                except SignatureVerificationError:
                    ok = False
                self.assertEqual(ok, case["valid"], case["name"])


if __name__ == "__main__":
    unittest.main()
