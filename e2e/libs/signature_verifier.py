"""
SignatureVerifier -- Verifies HMAC-SHA256 (v1,) and Ed25519 (v1a,) signatures
in the Standard Webhooks format.
"""

import base64
import hashlib
import hmac
import json

from nacl.signing import VerifyKey
from nacl.exceptions import BadSignatureError


def verify_hmac_signature(delivery: dict, secret: str):
    """Verify HMAC-SHA256 signature (v1, prefix)."""
    headers = delivery["headers"]
    msg_id = headers.get("Webhook-Id", headers.get("webhook-id", ""))
    timestamp = headers.get("Webhook-Timestamp", headers.get("webhook-timestamp", ""))
    signature_header = headers.get("Webhook-Signature", headers.get("webhook-signature", ""))

    if not all([msg_id, timestamp, signature_header]):
        raise AssertionError(f"Missing signature headers: id={msg_id}, ts={timestamp}, sig={signature_header}")

    v1_sig = None
    for part in signature_header.split(" "):
        if part.startswith("v1,"):
            v1_sig = part[3:]
            break

    if not v1_sig:
        raise AssertionError(f"No v1, signature found in: {signature_header}")

    body_str = delivery["body"] if isinstance(delivery["body"], str) else json.dumps(delivery["body"], separators=(",", ":"))
    message = f"{msg_id}.{timestamp}.{body_str}"

    secret_clean = secret.replace("whsec_", "")
    secret_bytes = base64.b64decode(secret_clean)

    expected = hmac.new(secret_bytes, message.encode(), hashlib.sha256).digest()
    expected_b64 = base64.b64encode(expected).decode()

    if not hmac.compare_digest(v1_sig, expected_b64):
        raise AssertionError(f"HMAC signature mismatch. Expected={expected_b64}, Got={v1_sig}")


def verify_ed25519_signature(delivery: dict, public_key_b64: str):
    """Verify Ed25519 signature (v1a, prefix)."""
    headers = delivery["headers"]
    msg_id = headers.get("Webhook-Id", headers.get("webhook-id", ""))
    timestamp = headers.get("Webhook-Timestamp", headers.get("webhook-timestamp", ""))
    signature_header = headers.get("Webhook-Signature", headers.get("webhook-signature", ""))

    if not all([msg_id, timestamp, signature_header]):
        raise AssertionError("Missing signature headers for Ed25519 verification.")

    v1a_sig = None
    for part in signature_header.split(" "):
        if part.startswith("v1a,"):
            v1a_sig = part[4:]
            break

    if not v1a_sig:
        raise AssertionError(f"No v1a, signature found in: {signature_header}")

    body_str = delivery["body"] if isinstance(delivery["body"], str) else json.dumps(delivery["body"], separators=(",", ":"))
    message = f"{msg_id}.{timestamp}.{body_str}".encode()

    public_key_bytes = base64.b64decode(public_key_b64)
    verify_key = VerifyKey(public_key_bytes)
    signature_bytes = base64.b64decode(v1a_sig)

    try:
        verify_key.verify(message, signature_bytes)
    except BadSignatureError:
        raise AssertionError("Ed25519 signature verification failed")


def delivery_has_signature_headers(delivery: dict):
    """Assert delivery has Standard Webhooks signature headers."""
    headers = delivery["headers"]
    lower_headers = {k.lower(): v for k, v in headers.items()}
    required = ["webhook-id", "webhook-timestamp", "webhook-signature"]
    missing = [h for h in required if h not in lower_headers]
    if missing:
        raise AssertionError(f"Missing signature headers: {missing}")
