# Federation upstream reuse matrix

| Component | Exact pin | Adopted behavior | Local changes | Notices |
|---|---|---|---|---|
| [go-jose](https://github.com/go-jose/go-jose) | v4.1.9; commit3c8a86c0b60170109f319d96b05da1e77cc8dbb4 | Standard compact JWS EdDSA signing and restricted verification | None; native protocol policy wraps public APIs | Apache-2.0; third_party/github.com_go-jose_go-jose_v4-v4.1.9-LICENSE |
| [modernc SQLite](https://gitlab.com/cznic/sqlite) | v1.60.1 | Test-only SQLite database/sql driver; transaction, restart, corruption and rollback fixtures | None | Full graph/pins/notices in THIRD_PARTY_NOTICES.md and go.sum |

The native Store defines consent, network scope, trust, replay, withdrawal, quota and audit rules. Neither upstream library supplies those application guarantees. No root dependency or other future-module dependency is introduced. Prior research is retained in UPSTREAM-RESEARCH.md. The continuation adds no dependency.
