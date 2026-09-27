use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use serde::Deserialize;
use std::collections::HashMap;

#[path = "../sparrow_verify.rs"]
mod sparrow_verify;

use sparrow_verify::{verify_ed25519_at, verify_hmac_at};

#[derive(Deserialize)]
struct Vectors {
    now: i64,
    tolerance_seconds: u64,
    cases: Vec<Case>,
}

#[derive(Deserialize)]
struct Case {
    name: String,
    scheme: String,
    key: String,
    headers: HashMap<String, String>,
    payload_b64: String,
    valid: bool,
}

#[test]
fn run_vectors() {
    let data = std::fs::read_to_string(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("../../..")
            .join("pkg/signature/testdata/vectors.json"),
    )
    .expect("read vectors.json");
    let vectors: Vectors = serde_json::from_str(&data).expect("parse vectors.json");

    let mut pass = 0;
    let mut fail = 0;

    for case in &vectors.cases {
        let payload = BASE64.decode(&case.payload_b64).unwrap_or_default();
        let result = match case.scheme.as_str() {
            "hmac" => verify_hmac_at(
                &payload,
                &case.headers,
                &case.key,
                vectors.tolerance_seconds,
                vectors.now,
            ),
            "ed25519" => verify_ed25519_at(
                &payload,
                &case.headers,
                &case.key,
                vectors.tolerance_seconds,
                vectors.now,
            ),
            other => panic!("unknown scheme: {other}"),
        };
        let got_valid = result.is_ok();
        if got_valid == case.valid {
            println!("PASS: {}", case.name);
            pass += 1;
        } else {
            println!(
                "FAIL: {} (expected valid={}, got valid={}, err={:?})",
                case.name,
                case.valid,
                got_valid,
                result.err()
            );
            fail += 1;
        }
    }
    println!("\n{pass} passed, {fail} failed out of {} cases", pass + fail);
    assert_eq!(fail, 0, "{fail} test(s) failed");
}
