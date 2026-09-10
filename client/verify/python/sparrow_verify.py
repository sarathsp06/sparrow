"""Verify Sparrow webhook delivery signatures (Standard Webhooks format).

Every delivery carries three headers:

    webhook-id:        msg_<delivery-id>
    webhook-timestamp: Unix seconds
    webhook-signature: space-delimited signatures, e.g. "v1,<base64> v1a,<base64>"

The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".

- "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
  Webhooks format ("whsec_" + base64) are decoded before use; any other
  secret is used as raw bytes. Verification is stdlib-only.
- "v1a," is Ed25519, verified with the hex-encoded public key from the
  webhook resource's ``signing_public_key`` field. Requires the
  ``cryptography`` package (imported lazily).

Copy this file into your project or vendor it as-is; it has no dependency
on the generated Sparrow client.

Usage::

    from sparrow_verify import verify_hmac, verify_ed25519, SignatureVerificationError

    try:
        verify_hmac(request.body, request.headers, secret)   # raw bytes, not re-serialized
    except SignatureVerificationError:
        return 401
"""

from __future__ import annotations

import base64
import hashlib
import hmac
import time
from typing import Mapping

DEFAULT_TOLERANCE_SECONDS = 5 * 60

__all__ = ["SignatureVerificationError", "verify_hmac", "verify_ed25519"]


class SignatureVerificationError(Exception):
    """Raised when a delivery signature cannot be verified."""


def verify_hmac(
    payload: bytes,
    headers: Mapping[str, str],
    secret: str,
    tolerance_seconds: int = DEFAULT_TOLERANCE_SECONDS,
) -> None:
    """Verify the ``v1,`` (HMAC-SHA256) signature. Raises on failure.

    ``payload`` must be the raw request body bytes, exactly as received.
    ``secret`` is the webhook secret, with or without the ``whsec_`` prefix.
    """
    msg_id, timestamp, sig_header = _required_headers(headers)
    _check_timestamp(timestamp, tolerance_seconds)

    if secret.startswith("whsec_"):
        try:
            key = base64.b64decode(secret[len("whsec_"):], validate=True)
        except Exception as exc:
            raise SignatureVerificationError(f"invalid whsec_ secret: {exc}") from exc
    else:
        key = secret.encode()

    message = f"{msg_id}.{timestamp}.".encode() + payload
    expected = hmac.new(key, message, hashlib.sha256).digest()

    for candidate in _decode_signatures(sig_header, "v1,"):
        if hmac.compare_digest(candidate, expected):
            return
    raise SignatureVerificationError("no matching v1 (HMAC-SHA256) signature")


def verify_ed25519(
    payload: bytes,
    headers: Mapping[str, str],
    public_key_hex: str,
    tolerance_seconds: int = DEFAULT_TOLERANCE_SECONDS,
) -> None:
    """Verify the ``v1a,`` (Ed25519) signature. Raises on failure.

    ``payload`` must be the raw request body bytes, exactly as received.
    ``public_key_hex`` is the hex-encoded key from the webhook resource's
    ``signing_public_key`` field.
    """
    try:
        from cryptography.exceptions import InvalidSignature
        from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PublicKey
    except ImportError as exc:  # pragma: no cover
        raise SignatureVerificationError(
            "Ed25519 verification requires the 'cryptography' package"
        ) from exc

    msg_id, timestamp, sig_header = _required_headers(headers)
    _check_timestamp(timestamp, tolerance_seconds)

    try:
        public_key = Ed25519PublicKey.from_public_bytes(bytes.fromhex(public_key_hex))
    except Exception as exc:
        raise SignatureVerificationError(f"invalid hex public key: {exc}") from exc

    message = f"{msg_id}.{timestamp}.".encode() + payload
    for candidate in _decode_signatures(sig_header, "v1a,"):
        try:
            public_key.verify(candidate, message)
            return
        except InvalidSignature:
            continue
    raise SignatureVerificationError("no matching v1a (Ed25519) signature")


def _required_headers(headers: Mapping[str, str]) -> tuple[str, str, str]:
    lowered = {k.lower(): v for k, v in headers.items()}
    try:
        return (
            lowered["webhook-id"],
            lowered["webhook-timestamp"],
            lowered["webhook-signature"],
        )
    except KeyError as exc:
        raise SignatureVerificationError(f"missing header: {exc.args[0]}") from exc


def _check_timestamp(timestamp: str, tolerance_seconds: int) -> None:
    try:
        seconds = int(timestamp)
    except ValueError as exc:
        raise SignatureVerificationError(f"invalid webhook-timestamp: {timestamp!r}") from exc
    if abs(time.time() - seconds) > tolerance_seconds:
        raise SignatureVerificationError("webhook-timestamp outside tolerance (possible replay)")


def _decode_signatures(sig_header: str, prefix: str):
    for part in sig_header.split():
        if not part.startswith(prefix):
            continue
        try:
            yield base64.b64decode(part[len(prefix):], validate=True)
        except Exception:
            continue
