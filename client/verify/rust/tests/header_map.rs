// The Axum example in sparrow_verify.rs passes `&http::HeaderMap`; make sure
// that compiles and verifies (a vector case rebuilt as a real HeaderMap).
use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use http::{HeaderMap, HeaderName, HeaderValue};

#[path = "../sparrow_verify.rs"]
mod sparrow_verify;

#[test]
fn verifies_with_http_header_map() {
    let data = std::fs::read_to_string(
        std::path::Path::new(env!("CARGO_MANIFEST_DIR")).join("../../../pkg/signature/testdata/vectors.json"),
    )
    .unwrap();
    let v: serde_json::Value = serde_json::from_str(&data).unwrap();
    let case = &v["cases"][0];
    assert_eq!(case["name"], "hmac valid whsec secret");

    let mut headers = HeaderMap::new();
    for (k, val) in case["headers"].as_object().unwrap() {
        headers.insert(
            HeaderName::from_bytes(k.as_bytes()).unwrap(),
            HeaderValue::from_str(val.as_str().unwrap()).unwrap(),
        );
    }
    let payload = BASE64.decode(case["payload_b64"].as_str().unwrap()).unwrap();
    let now = v["now"].as_i64().unwrap();
    let tolerance = v["tolerance_seconds"].as_u64().unwrap();

    sparrow_verify::verify_hmac_at(&payload, &headers, case["key"].as_str().unwrap(), tolerance, now)
        .expect("HeaderMap should verify");
    assert!(sparrow_verify::verify_hmac_at(&payload, &headers, "whsec_b3RoZXI=", tolerance, now).is_err());
}
