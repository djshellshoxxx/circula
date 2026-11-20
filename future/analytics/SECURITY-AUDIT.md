# Analytics master-scope delta review

The prior independent full-module review and fixes remain in [AUDIT.md](AUDIT.md), at recovered source `243f7fc181a14cecb124c4b9733b86c010d53993`. This continuation starts from that exact commit. The coordinator independently reviewed source commit `5c239f0494ff9ee69c4e84f985ec15d638140bbd` (transport guard, regressions and performance fixture), found no issue, and verified that the fixture checks exact accounting rather than only timing. This is a bounded delta review, not a new full-module privacy audit.

## Confirmed finding

The HTTP handler checked its 2,048-byte query limit after invoking host authentication. A legitimate host implementation could parse an arbitrarily large query before the module enforced the bound. The guard now runs before any host callback. Oversized ordinary, repeated-field and malformed-escape queries return 400 without invoking authentication. The exact 2,048-byte boundary still reaches normal strict validation. `TestQueryLimitPrecedesHostCallbacks` covers this ordering.

No new query fields, source interface, file store, network endpoint, root dependency, production route or ledger writer was added. Existing source authorization remains mandatory for every successful analytics read.

## Open privacy and lifecycle gates

Fixed granularity, cohort/contributor thresholds, dominance checks, complementary suppression and public-only fingerprints reduce disclosure. They do not provide differential privacy or guarantee anonymity. The module still recomputes releases from trusted source snapshots; it cannot enforce an immutable release across source corrections or restarts. A reviewed immutable-release, query-budget and correction policy remains a substantive requirement. Do not treat this as completed merely because core integration is held.

Snapshots are transient and never persisted by this package. Persistence/recovery of ledger truth and retained published releases are unqualified. The actual current-membership/admin historical adapter remains separately authorized integration work. Manual browser/accessibility, privacy review under external knowledge and production performance budgets remain open.

## Upstream surface

The module uses Go's standard library only. No researched chart or statistics package was imported. Historical upstream research is preserved with exact inspected revisions; this continuation does not claim a new remote advisory scan.
