// Verify Sparrow webhook delivery signatures (Standard Webhooks format).
//
// Every delivery carries three headers:
//
//     webhook-id:        msg_<delivery-id>
//     webhook-timestamp: Unix seconds
//     webhook-signature: space-delimited signatures, e.g. "v1,<base64> v1a,<base64>"
//
// The signed message is "{webhook-id}.{webhook-timestamp}.{raw body}".
//
// - "v1,"  is HMAC-SHA256 keyed with the webhook secret. Secrets in Standard
//   Webhooks format ("whsec_" + base64) are decoded before use; any other
//   secret is used as raw bytes.
// - "v1a," is Ed25519, verified with the hex-encoded public key from the
//   webhook resource's `signing_public_key` field.
//
// Both verifiers reject deliveries whose webhook-timestamp is more than
// `tolerance` away from the current time, to prevent replay.
//
// Copy this file into your project. Add these dependencies to Cargo.toml:
//
//     [dependencies]
//     hmac = "0.12"
//     sha2 = "0.10"
//     base64 = "0.22"
//     ed25519-dalek = "2"
//
// Usage (Axum example):
//
//     use axum::{body::Bytes, extract::Extension, http::HeaderMap, response::IntoResponse};
//
//     async fn webhook(headers: HeaderMap, body: Bytes) -> impl IntoResponse {
//         // IMPORTANT: use the raw body bytes, not re-serialized JSON.
//         match sparrow_verify::verify_hmac(&body, &headers, "whsec_...") {
//             Ok(()) => (axum::http::StatusCode::OK, "ok"),
//             Err(_) => (axum::http::StatusCode::UNAUTHORIZED, "invalid signature"),
//         }
//     }

use std::fmt;
use std::time::{SystemTime, UNIX_EPOCH};

use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use ed25519_dalek::{Signature, Verifier, VerifyingKey};
use hmac::{Hmac, Mac};
use sha2::Sha256;

/// Default tolerance: 5 minutes.
pub const DEFAULT_TOLERANCE_SECS: u64 = 300;

/// Errors returned by signature verification.
#[derive(Debug)]
pub enum SignatureError {
    /// A required header (webhook-id, webhook-timestamp, webhook-signature) is missing.
    MissingHeader(&'static str),
    /// The webhook-timestamp header is malformed.
    InvalidTimestamp,
    /// The webhook-timestamp is outside the tolerance window (possible replay).
    TimestampOutsideTolerance,
    /// The webhook secret is empty.
    EmptySecret,
    /// The whsec_ secret contains invalid base64.
    InvalidSecretBase64,
    /// The hex public key is invalid or the wrong length.
    InvalidPublicKey,
    /// No signature of the requested scheme matched.
    NoMatch,
}

impl fmt::Display for SignatureError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::MissingHeader(name) => write!(f, "missing header: {name}"),
            Self::InvalidTimestamp => write!(f, "invalid webhook-timestamp"),
            Self::TimestampOutsideTolerance => {
                write!(f, "webhook-timestamp outside tolerance (possible replay)")
            }
            Self::EmptySecret => write!(f, "webhook secret is empty"),
            Self::InvalidSecretBase64 => write!(f, "invalid whsec_ secret"),
            Self::InvalidPublicKey => write!(f, "invalid hex public key"),
            Self::NoMatch => write!(f, "no matching signature"),
        }
    }
}

impl std::error::Error for SignatureError {}

/// Verify the "v1," (HMAC-SHA256) signature using the current system clock.
///
/// `payload` must be the raw request body bytes, exactly as received.
/// `secret` is the webhook secret, with or without the "whsec_" prefix.
///
/// Headers are looked up case-insensitively. Anything that iterates as
/// `(&name, &value)` pairs with `AsRef<str>` names and `AsRef<[u8]>` values
/// works: `&http::HeaderMap`, `&HashMap<String, String>`, etc.
pub fn verify_hmac<'a, H, K, V>(
    payload: &[u8],
    headers: H,
    secret: &str,
) -> Result<(), SignatureError>
where
    H: IntoIterator<Item = (&'a K, &'a V)>,
    K: AsRef<str> + ?Sized + 'a,
    V: AsRef<[u8]> + ?Sized + 'a,
{
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;
    verify_hmac_at(payload, headers, secret, DEFAULT_TOLERANCE_SECS, now)
}

/// Verify the "v1," (HMAC-SHA256) signature with an explicit clock and tolerance.
///
/// `now` is the current time as Unix seconds. `tolerance` is the maximum
/// accepted skew in seconds.
pub fn verify_hmac_at<'a, H, K, V>(
    payload: &[u8],
    headers: H,
    secret: &str,
    tolerance: u64,
    now: i64,
) -> Result<(), SignatureError>
where
    H: IntoIterator<Item = (&'a K, &'a V)>,
    K: AsRef<str> + ?Sized + 'a,
    V: AsRef<[u8]> + ?Sized + 'a,
{
    let (msg_id, timestamp, sigs) = parse_headers(headers, tolerance, now)?;

    let key = if let Some(encoded) = secret.strip_prefix("whsec_") {
        BASE64
            .decode(encoded)
            .map_err(|_| SignatureError::InvalidSecretBase64)?
    } else {
        secret.as_bytes().to_vec()
    };
    if key.is_empty() {
        return Err(SignatureError::EmptySecret);
    }

    let message = signed_message(&msg_id, &timestamp, payload);
    let mut mac = Hmac::<Sha256>::new_from_slice(&key).expect("HMAC accepts any key size");
    mac.update(&message);

    for part in &sigs {
        let encoded = match part.strip_prefix("v1,") {
            Some(e) => e,
            None => continue,
        };
        let candidate = match BASE64.decode(encoded) {
            Ok(c) => c,
            Err(_) => continue,
        };
        // Constant-time comparison via hmac crate's verify_slice.
        if Hmac::<Sha256>::new_from_slice(&key)
            .map(|mut m| {
                m.update(&message);
                m.verify_slice(&candidate)
            })
            .is_ok_and(|r| r.is_ok())
        {
            return Ok(());
        }
    }
    Err(SignatureError::NoMatch)
}

/// Verify the "v1a," (Ed25519) signature using the current system clock.
///
/// `payload` must be the raw request body bytes, exactly as received.
/// `public_key_hex` is the hex-encoded key from the webhook resource's
/// `signing_public_key` field.
pub fn verify_ed25519<'a, H, K, V>(
    payload: &[u8],
    headers: H,
    public_key_hex: &str,
) -> Result<(), SignatureError>
where
    H: IntoIterator<Item = (&'a K, &'a V)>,
    K: AsRef<str> + ?Sized + 'a,
    V: AsRef<[u8]> + ?Sized + 'a,
{
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;
    verify_ed25519_at(payload, headers, public_key_hex, DEFAULT_TOLERANCE_SECS, now)
}

/// Verify the "v1a," (Ed25519) signature with an explicit clock and tolerance.
pub fn verify_ed25519_at<'a, H, K, V>(
    payload: &[u8],
    headers: H,
    public_key_hex: &str,
    tolerance: u64,
    now: i64,
) -> Result<(), SignatureError>
where
    H: IntoIterator<Item = (&'a K, &'a V)>,
    K: AsRef<str> + ?Sized + 'a,
    V: AsRef<[u8]> + ?Sized + 'a,
{
    let (msg_id, timestamp, sigs) = parse_headers(headers, tolerance, now)?;

    let pub_bytes = hex_decode(public_key_hex).ok_or(SignatureError::InvalidPublicKey)?;
    if pub_bytes.len() != 32 {
        return Err(SignatureError::InvalidPublicKey);
    }
    let verifying_key = VerifyingKey::from_bytes(
        pub_bytes
            .as_slice()
            .try_into()
            .map_err(|_| SignatureError::InvalidPublicKey)?,
    )
    .map_err(|_| SignatureError::InvalidPublicKey)?;

    let message = signed_message(&msg_id, &timestamp, payload);
    for part in &sigs {
        let encoded = match part.strip_prefix("v1a,") {
            Some(e) => e,
            None => continue,
        };
        let raw = match BASE64.decode(encoded) {
            Ok(r) => r,
            Err(_) => continue,
        };
        let sig = match Signature::from_slice(&raw) {
            Ok(s) => s,
            Err(_) => continue,
        };
        if verifying_key.verify(&message, &sig).is_ok() {
            return Ok(());
        }
    }
    Err(SignatureError::NoMatch)
}

/// Build "{msg_id}.{timestamp}.{payload}".
fn signed_message(msg_id: &str, timestamp: &str, payload: &[u8]) -> Vec<u8> {
    let mut msg = Vec::with_capacity(msg_id.len() + 1 + timestamp.len() + 1 + payload.len());
    msg.extend_from_slice(msg_id.as_bytes());
    msg.push(b'.');
    msg.extend_from_slice(timestamp.as_bytes());
    msg.push(b'.');
    msg.extend_from_slice(payload);
    msg
}

/// Decode a hex string. Returns None on invalid input.
fn hex_decode(s: &str) -> Option<Vec<u8>> {
    if !s.len().is_multiple_of(2) {
        return None;
    }
    let mut out = Vec::with_capacity(s.len() / 2);
    for chunk in s.as_bytes().chunks(2) {
        let hi = hex_nibble(chunk[0])?;
        let lo = hex_nibble(chunk[1])?;
        out.push((hi << 4) | lo);
    }
    Some(out)
}

fn hex_nibble(b: u8) -> Option<u8> {
    match b {
        b'0'..=b'9' => Some(b - b'0'),
        b'a'..=b'f' => Some(b - b'a' + 10),
        b'A'..=b'F' => Some(b - b'A' + 10),
        _ => None,
    }
}

/// Extract and validate the three Standard Webhooks headers.
fn parse_headers<'a, H, K, V>(
    headers: H,
    tolerance: u64,
    now: i64,
) -> Result<(String, String, Vec<String>), SignatureError>
where
    H: IntoIterator<Item = (&'a K, &'a V)>,
    K: AsRef<str> + ?Sized + 'a,
    V: AsRef<[u8]> + ?Sized + 'a,
{
    let mut msg_id: Option<String> = None;
    let mut timestamp: Option<String> = None;
    let mut sig_header: Option<String> = None;

    for (k, v) in headers {
        // Non-UTF-8 values can't be valid Standard Webhooks headers: skip them.
        let Ok(value) = std::str::from_utf8(v.as_ref()) else {
            continue;
        };
        match k.as_ref().to_ascii_lowercase().as_str() {
            "webhook-id" if msg_id.is_none() => msg_id = Some(value.to_string()),
            "webhook-timestamp" if timestamp.is_none() => timestamp = Some(value.to_string()),
            "webhook-signature" if sig_header.is_none() => sig_header = Some(value.to_string()),
            _ => {}
        }
    }

    let msg_id = msg_id.ok_or(SignatureError::MissingHeader("webhook-id"))?;
    let timestamp = timestamp.ok_or(SignatureError::MissingHeader("webhook-timestamp"))?;
    let sig_header = sig_header.ok_or(SignatureError::MissingHeader("webhook-signature"))?;

    // Validate timestamp: optional leading '-' followed by ASCII digits only.
    let ts_digits = timestamp.strip_prefix('-').unwrap_or(&timestamp);
    if ts_digits.is_empty() || !ts_digits.bytes().all(|b| b.is_ascii_digit()) {
        return Err(SignatureError::InvalidTimestamp);
    }
    let seconds: i64 = timestamp
        .parse()
        .map_err(|_| SignatureError::InvalidTimestamp)?;
    let skew = (now - seconds).unsigned_abs();
    if skew > tolerance {
        return Err(SignatureError::TimestampOutsideTolerance);
    }

    let sigs: Vec<String> = sig_header
        .split_whitespace()
        .filter(|s| !s.is_empty())
        .map(|s| s.to_string())
        .collect();

    Ok((msg_id, timestamp, sigs))
}
