// This crate exists only so `cargo test` and `cargo clippy` can compile
// the copy-paste module. Users copy `sparrow_verify.rs` into their project.
#[path = "../sparrow_verify.rs"]
pub mod sparrow_verify;
