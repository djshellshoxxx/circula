# Future specification: Circula command-line interface
Status: PROPOSED / FUTURE ONLY. Not approved for implementation; no CLI is claimed to exist.

## Purpose and scope
Provide an optional, scriptable operator and member interface for the existing mutual-credit application. Preserve the Go, SQLite, server-rendered HTML and JSON API architecture; do not introduce a blockchain or federation dependency. The CLI must reuse canonical domain services and authorization checks, not implement a second ledger.

## Proposed commands
- `circula --help|--version`; `circula doctor [--json]`; `circula config validate`
- `circula serve [--config FILE]`; `circula migrate status|up`; `circula backup create|verify`; `circula restore --from FILE --dry-run`
- `circula account list|show|create|suspend`; `circula limits show|set`
- `circula balance show ACCOUNT`; `circula transfer create --from ID --to ID --amount INTEGER --idempotency-key KEY`; `circula transfer show|reverse`
- `circula ledger list|reconcile|verify`; `circula report balances|activity`; `circula export transactions --format json|csv`
Actual naming/flags remain subject to existing API and domain-model review. Commands modifying accounts, credit limits or ledger entries require explicit authenticated roles; no CLI-only bypass.

## CLI contract
Use subcommands and contextual help, consistent `--json`, `--quiet`, `--verbose`, `--dry-run` where meaningful, UTF-8, deterministic JSON schemas with version fields, pagination and stable ordering. Stable exit codes: 0 success, 2 usage/validation, 3 operational failure, 4 no results, 5 cancellation. Never print tokens or private credentials; avoid secrets as command arguments. Support noninteractive CI, timeouts, and structured stderr diagnostics. Document POSIX and Windows use.

## Accounting invariants
Use atomic SQLite transactions and canonical ledger APIs; no mutable deletion of posted entries. Idempotent retries cannot duplicate postings. Enforce configured limits, balance arithmetic without floating-point currency errors, and double-entry conservation for a closed network. Reversals are explicit compensating records with audit provenance. Backup must be SQLite-consistent and restore must require explicit confirmation and validation.

## Acceptance tests
1. Headless help/version/config/doctor work without launching a web UI.
2. Unauthorized account/transfer/limit commands fail without mutation.
3. Duplicate idempotency key does not double-post; concurrent transfers preserve invariants.
4. Reconcile detects deliberately inconsistent fixtures; export balances match canonical API.
5. Backup, integrity check, restore to isolated location, and post-restore reconciliation pass.
6. JSON is parseable with no mixed stdout logs; stable exit codes tested on Windows/Linux.
7. Sensitive data is redacted in errors, logs and shell examples.

## Delivery phases
F0: review existing models, routes, trust boundaries and owners; finalize command/API mapping.
F1: CLI framework, help, doctor, config, read-only balance/ledger.
F2: authenticated account and transfer operations with idempotency and audit.
F3: reconciliation, backup/restore, exports, integration/CI and operator docs.
Non-goals: blockchain, federation, unaudited ledger edits, separate accounting engine. Do not displace active beta tasks; schedule as a future milestone.
