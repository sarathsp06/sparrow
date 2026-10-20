# Vendored verification helpers

Byte-identical copies of the tested signature helpers. Generated receivers
copy them unchanged; they are never generated or edited per target.

| File | Original |
|------|----------|
| `signature.go.txt` | `pkg/signature/signature.go` |
| `sparrow_verify.py` | `client/verify/python/sparrow_verify.py` |
| `sparrow-verify.ts` | `client/verify/js/sparrow-verify.ts` |

`helpers_test.go` fails when a copy drifts from its original. To update,
copy the original over the file here.
