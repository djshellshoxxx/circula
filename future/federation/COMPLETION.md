# Historical FNET-209 completion criteria

This is the archived assessment at `dc59d9b8349cebefb35278bd37ae5e4f207dd833`. Its 81.25 subtotal is not a score or qualification claim for the recovered composition. See `RECOVERY-20261119.md` and `VALIDATION.md` for exact current scope and pending gates.

Partial; disabled; integration HOLD. Current tests passed and the coordinator independently reviewed the runtime correction, response-contract preservation and new regressions; see SECURITY-AUDIT.md. Each item has equal weight within its category. Historical evidence is AUDIT.md and the coordinator's recovered baseline; new delta requires fresh verification.

| Category | Weight | Counted / total | Earned | Evidence and unfinished criteria |
|---|---:|---:|---:|---|
| Domain functionality |25|8/8|25|Signed projection, peer identity, key lifecycle, bilateral trust, consent, ordering, partition retry, admin/audit behavior all have standalone fixtures|
| Persistence/lifecycle |15|4/5|12|Transactional inbox/outbox; restart; withdrawal/departure; backup/corruption tests. Lifetime capacity/epoch-compaction operational acceptance remains open|
| Service/API/CLI |15|3/4|11.25|Store service, unregistered HTTP/no-JS, CLI. Actual host transaction/authorization adapter qualification remains deferred|
| UI/accessibility |10|2/4|5|Escaped semantic pages and native forms tested. Real browser automation and manual keyboard/screen-reader acceptance remain open|
| Security/privacy |15|4/5|12|Restricted crypto/profile, scoped consent/replay, bounded SSRF transport, predecessor audit plus independently reviewed delta. Production key custody/final protocol acceptance open|
| Automated verification |15|4/5|12|Unit/race; failure/restart; bounded fuzz; checksums/advisory evidence at recorded source. Live independent peer interoperability remains open|
| Documentation/operations |5|4/5|4|Contracts/ADR; validation/audit; reuse/notices; recovery handoff. Production key/peer operational exercise remains open|
| Total |100|—|81.25|Current evidence subtotal; not full-feature qualification|

No unit coverage percentage substitutes for any gate. Browser checks were not retried in the known blocked runtime. No adopted requirement for cross-network settlement is counted; settlement is explicitly out of scope. Current protocol and schema upgrade/compaction need design/decision work and cannot be called externally verified.
