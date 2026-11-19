# Receipts upstream reuse matrix

| Component | Exact revision | Used for | Local modifications | License |
|---|---|---|---|---|
| [gowebpki/jcs](https://github.com/gowebpki/jcs) | 7d24017834028a80af75c290d5b122d6346dc76b | Canonical JSON through Transform | None; module dependency | Apache-2.0, third_party/jcs-LICENSE |
| [go-jose/go-jose](https://github.com/go-jose/go-jose) | 3c8a86c0b60170109f319d96b05da1e77cc8dbb4, v4.1.9 | Standard JWS EdDSA opaque signing and verification | None; module dependency | Apache-2.0, third_party/go-jose-LICENSE |

Exact checksums are retained in go.sum; copyright and NOTICE disposition in THIRD_PARTY_NOTICES.md. Native receipt projection, explicit opt-in, privacy, authorization and key policy are not borrowed library guarantees. Prior research is in UPSTREAM-RESEARCH.md. No new dependency added by the master continuation.
