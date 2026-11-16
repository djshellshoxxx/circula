# Circula Wiki — Home

> **Status as of 2026-11-16.** These pages describe the Circula source on branch `claude/cool-gauss-vdxlod` of [djshellshoxxx/circula](https://github.com/djshellshoxxx/circula). Feature state is taken from [`docs/plans/circula-completion-matrix.md`](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/plans/circula-completion-matrix.md) (dated 2026-11-11). This is **not a release**: no tag exists, and owner decisions and manual checks listed below remain open.

## Choose your guide

- **[User Guide](User-Guide)**: members and network administrators. Accounts, joining a network, trades, history, disputes, privacy limits.
- **[Developer Guide](Developer-Guide)**: contributors and operators. Module map, CLI commands, local build, tests, CI, release gate, open work.

## What Circula is

Circula is a self-hosted mutual-credit ledger for trading networks. Members of a network propose, accept and settle trades that move credit between their accounts. Every posting is an append-only double entry, every command writes an audit event, and `circula verify-ledger` recomputes the ledger invariants.

A Circula credit is a network accounting unit. Circula does **not** transfer fiat money, hold deposits, or provide cryptocurrency custody.

The repository contains the Go core (domain, service layer, SQLite store), a server-rendered web UI, a JSON API under `/api/v1`, and the `circula` command.

## Feature state at a glance

Summary of [the completion matrix](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/plans/circula-completion-matrix.md). Rows marked COMPLETE are backed by tests named in that file; the matrix is the reference for each row.

| Area | State (matrix, 2026-11-11) | Open items |
|---|---|---|
| Ledger: two-step trades, credit limit, reversal, immutable records, integer money, reconciliation | COMPLETE | Fault injection is at the store interface, not real SQLite I/O errors |
| Identity, sessions, CSRF, IDOR, cookies, password policy, TOTP MFA, account recovery, registration and invitations | COMPLETE | Per-login throttle values await owner choice (SEC-006); timing equivalence for account enumeration not measured |
| Dashboard, trade flow, history, account, administration, audit views, JSON API contract | COMPLETE | Admin without the reviewer role may decide disputes (DEC-072) |
| Accessibility (WCAG 2.2 AA, automated) | COMPLETE (automated) | Manual screen-reader run pending (UX-056, QA-035) |
| Data export | COMPLETE (in memory) | Streaming/spool slice not integrated |
| Migrations, constraints and triggers, backup and restore drill | COMPLETE | None listed |
| Observability (health, readiness, metrics, shutdown, logs) | COMPLETE | None listed |
| Container and deployment files (systemd, Dockerfile, compose, reverse-proxy docs) | UNVERIFIED | Container build not run in the matrix evidence; smoke script run |
| Release pipeline (SBOM, scans, release-notes gate) | COMPLETE (not exercised on a tag) | First tagged release |
| CI | PARTIAL | Remove duplicate runs; move macOS/Windows tests to main or nightly |

The matrix estimates about 98 % engineering completion over its 45 listed rows. That count excludes owner gates.

## Open owner gates

These are decisions or external dependencies. Engineering cannot close them:

- No licence file yet (`LICENSE` is absent). Do not redistribute until one is added.
- Independent security review (SEC-033) and the owner's security and financial-policy choices (DEC-005, DEC-011, DEC-013, DEC-014, DEC-021, DEC-022, DEC-023, DEC-025; break-glass scope A3-F02).
- Manual screen-reader test (UX-056, QA-035).
- First tagged release and the beta merge.

No instance is implied by this repository. An operator must install and configure one (see the [Developer Guide](Developer-Guide)).

## Source and evidence

- [README](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/README.md): quickstart, command table, status and limits
- [Specifications](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/specs/README.md). Several status columns are stale; the completion matrix takes precedence.
- [Operations guide](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/README.md) and [configuration reference](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/ops/configuration.md)
- [Member guide](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/guides/user-guide.md) and [network administrator guide](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/guides/admin-guide.md)
- [Decision log](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/docs/DECISIONS.md)
- [CI workflow](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/.github/workflows/ci.yml). Run results are on the Actions tab; they were not re-checked for this page (UNVERIFIED).
- [Security reporting policy](https://github.com/djshellshoxxx/circula/blob/claude/cool-gauss-vdxlod/SECURITY.md). Report vulnerabilities privately, not in public Issues or Discussions.

Pull requests and issues are the place for live work status. This wiki does not track them, and PR-level status claims here were not re-verified.
