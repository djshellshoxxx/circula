# Recurring deferred integration contract

**Integration HOLD; enabled=false.** No route, job, root dependency or migration registration is authorized. The isolated module exposes Service, Handler and RunCLI only for explicitly constructed adapters.

A later authorized host must supply current transaction-aware membership/target/terms/capacity and network opt-in policy; current-session authentication and cookie CSRF; a bounded SQLite connection with foreign_keys; and an explicitly invoked scheduler with durable continuation/fairness policy. Optional inbox insertion must use the supplied SQL transaction and never a remote delivery or ledger write.

## HTTP query contract

GET accepts one `after` and one `limit` only. A single empty after is the first page; limit is 1–100 when nonempty. Query bytes are capped at 8192 before host callbacks. Duplicate, unknown, malformed or invalid values return 400. POST accepts no nonempty query: request facts come from the bounded strict body. Every generated form explicitly posts to the query-free local path, including forms reached from paginated GET URLs. The invalid-query decision occurs before session callbacks and before mutation.

## Lifecycle and disable

GUIDE.md specifies explicit additive module migration, rollback, closed-writer/consistent SQLite backup, 90-day minimum maintenance retention and Depart semantics. Minimal fingerprints/firing tombstones intentionally remain to prevent replay. Production owner/network archival/erasure policy remains unresolved. Disable by withholding construction and host registration or stopping the separately owned scheduler. No ledger rollback or automated trade occurs.

Actual core adapter conformance, scheduler recovery/fairness, live browser/assistive technology, operational backup/erasure and full ledger regression require separate qualification. Core integration remains outside this branch.
