# Reused components

- `github.com/gowebpki/jcs v1.0.3-0.20261110202536-7d2401783402`, full source commit `7d24017834028a80af75c290d5b122d6346dc76b`, https://github.com/gowebpki/jcs. Used through `jcs.Transform` in canonical.go and strict HTTP JSON parsing; no source modifications/transplants. Apache-2.0 full license retained as third_party/jcs-LICENSE. Source headers: Copyright 2021-2026 Bret Jordan & Benedikt Thoma; Copyright 2006-2019 WebPKI.org. No upstream root NOTICE file present in downloaded module. No nonstandard runtime transitive dependency. Exact checksums are in go.sum.

This does not assign a product license to Circula or permit integration/release. Packaging must carry required licenses/notices and reconsider advisories at the actual release date.

- `github.com/go-jose/go-jose/v4 v4.1.9`, full source commit `3c8a86c0b60170109f319d96b05da1e77cc8dbb4`, https://github.com/go-jose/go-jose. Used through `NewSigner`, `OpaqueSigner`, `ParseSignedCompact` and `Verify` for standard JWS EdDSA. Apache-2.0 full license retained as third_party/go-jose-LICENSE. No source copied or modified. No upstream root NOTICE file present in the downloaded module. No nonstandard runtime transitive dependency in the selected module. Neither generic ParseSigned nor JWT/all-algorithm parsing is used. Synthetic direct Ed25519 test handles exist only in test code and are generated per test.
