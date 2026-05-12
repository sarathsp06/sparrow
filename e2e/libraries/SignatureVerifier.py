"""
SignatureVerifier -- Robot Framework library for verifying webhook signatures.

Verifies both HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures in the
Standard Webhooks format.
"""

import base64
import hashlib
import hmac

from nacl.signing import VerifyKey
from nacl.exceptions import BadSignatureError


class SignatureVerifier:
    """Robot Framework library for verifying Sparrow webhook signatures."""

    ROBOT_LIBRARY_SCOPE = "SUITE"

    def verify_hmac_signature(self, delivery: dict, secret: str):
        """Verify that a delivery has a valid HMAC-SHA256 signature (v1, prefix).

        Args:
            delivery: Dict with 'body' and 'headers' keys
            secret: The webhook secret (base64-encoded)
        """
        headers = delivery["headers"]
        msg_id = headers.get("Webhook-Id", headers.get("webhook-id", ""))
        timestamp = headers.get("Webhook-Timestamp", headers.get("webhook-timestamp", ""))
        signature_header = headers.get("Webhook-Signature", headers.get("webhook-signature", ""))

        if not all([msg_id, timestamp, signature_header]):
            raise AssertionError(
                f"Missing signature headers. Got: webhook-id={msg_id}, "
                f"webhook-timestamp={timestamp}, webhook-signature={signature_header}"
            )

        # Find v1, signature
        v1_sig = None
        for part in signature_header.split(" "):
            if part.startswith("v1,"):
                v1_sig = part[3:]
                break

        if not v1_sig:
            raise AssertionError(f"No v1, signature found in: {signature_header}")

        # Compute expected HMAC
        body_str = delivery["body"] if isinstance(delivery["body"], str) else __import__("json").dumps(delivery["body"], separators=(",", ":"))
        message = f"{msg_id}.{timestamp}.{body_str}"

        # Secret is base64-encoded, may have "whsec_" prefix
        secret_clean = secret.replace("whsec_", "")
        secret_bytes = base64.b64decode(secret_clean)

        expected = hmac.new(secret_bytes, message.encode(), hashlib.sha256).digest()
        expected_b64 = base64.b64encode(expected).decode()

        if not hmac.compare_digest(v1_sig, expected_b64):
            raise AssertionError(f"HMAC signature mismatch. Expected={expected_b64}, Got={v1_sig}")

    def verify_ed25519_signature(self, delivery: dict, public_key_b64: str):
        """Verify that a delivery has a valid Ed25519 signature (v1a, prefix).

        Args:
            delivery: Dict with 'body' and 'headers' keys
            public_key_b64: Base64-encoded Ed25519 public key
        """
        headers = delivery["headers"]
        msg_id = headers.get("Webhook-Id", headers.get("webhook-id", ""))
        timestamp = headers.get("Webhook-Timestamp", headers.get("webhook-timestamp", ""))
        signature_header = headers.get("Webhook-Signature", headers.get("webhook-signature", ""))

        if not all([msg_id, timestamp, signature_header]):
            raise AssertionError(
                f"Missing signature headers for Ed25519 verification."
            )

        # Find v1a, signature
        v1a_sig = None
        for part in signature_header.split(" "):
            if part.startswith("v1a,"):
                v1a_sig = part[4:]
                break

        if not v1a_sig:
            raise AssertionError(f"No v1a, signature found in: {signature_header}")

        # Reconstruct message
        body_str = delivery["body"] if isinstance(delivery["body"], str) else __import__("json").dumps(delivery["body"], separators=(",", ":"))
        message = f"{msg_id}.{timestamp}.{body_str}".encode()

        # Verify
        public_key_bytes = base64.b64decode(public_key_b64)
        verify_key = VerifyKey(public_key_bytes)
        signature_bytes = base64.b64decode(v1a_sig)

        try:
            verify_key.verify(message, signature_bytes)
        except BadSignatureError:
            raise AssertionError("Ed25519 signature verification failed")

    def delivery_has_signature_headers(self, delivery: dict):
        """Assert that a delivery has the required Standard Webhooks signature headers."""
        headers = delivery["headers"]
        required = ["webhook-id", "webhook-timestamp", "webhook-signature"]
        # Case-insensitive check
        lower_headers = {k.lower(): v for k, v in headers.items()}
        missing = [h for h in required if h not in lower_headers]
        if missing:
            raise AssertionError(f"Missing signature headers: {missing}")
