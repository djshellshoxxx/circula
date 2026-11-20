# Receipts master completion criteria

Partial; integration HOLD; no production or full-feature qualification. The matrix records explicit inherited and current audit evidence; it does not establish production acceptance. Criteria have equal weight within each row. A full score cannot override a mandatory failed or unexecuted gate.

| Category | Weight | Counted / total | Earned | Explicit criteria and gaps |
|---|---:|---:|---:|---|
| Domain functionality | 25 | 5/5 | 25 | Opt-in projection; exact historical/reversal semantics; TXT/JSON/print; canonical integrity; optional JWS and offline key lifecycle tested in predecessor |
| Persistence/lifecycle | 15 | 4/5 | 12 | Ephemeral artifact contract; stateless recreation; concurrent read-only requests; retained/revoked public trust semantics. Real source restore/protected-key operations pending |
| Service/API/CLI | 15 | 3/4 | 11.25 | Read service; HTTP/no-JS; CLI. Actual authenticated read-transaction adapter remains deferred |
| UI/accessibility | 10 | 2/4 | 5 | Semantic static choice/display; no-JS/escaped print. Real browser/Tor and manual assistive-technology acceptance pending |
| Security/privacy | 15 | 5/5 | 15 | Scoped fresh read; no retained artifacts; strict cryptographic profile; predecessor independent audit; fresh continuation review with reproduced/fixed malformed-form finding |
| Automated verification | 15 | 3/4 | 11.25 | Unit/race isolation; strict parser/adversarial tests; bounded fuzz/checksums. Whole external verifier interoperability pending |
| Documentation/operations | 5 | 4/5 | 4 | Specs/architecture; validation; reuse/licenses; integration/runbook. Production key custody/trust operations acceptance pending |
| Total | 100 | | 83.5 | Explicit weighted evidence; mandatory external and operational gates remain open |

The exact recovered predecessor is `0f4ec8f50d2b356bd8192915f0b7a894f31d1624`. Prior AUDIT.md and coordinator baseline-nine evidence support the inherited rows; they do not qualify this continuation automatically. Current tested runtime is `860a55135a4d7302642829a08387a000ac1132ed`; the continuation adds no dependency or core wiring. Missing browser/manual/provider evidence remains uncounted.
