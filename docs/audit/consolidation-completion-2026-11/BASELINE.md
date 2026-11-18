# Consolidation baseline (2026-11-18)

Captured at session start, before any change. Clone unshallowed (`git fetch --unshallow`) and all
`refs/pull/*/head` fetched before ancestry analysis.

## Repository

| Item | Value |
|---|---|
| `main` | `ffa2758ffc2fe29133a46b1d8d3f0a6138c404e4` (2026-11-17T23:01:50-08:00, "Merge PR #75") |
| Remote branches | 96 (excluding `HEAD`) |
| Go toolchain | go1.26.11 (`go.mod`, CI and `deploy/Dockerfile` agree) |
| Direct deps | kin-openapi v0.153.0, google/uuid v1.6.0, x/crypto v0.59.0, x/text v0.44.0, modernc sqlite v1.62.0 |
| Workflows | `ci.yml`, `qa001-manual.yml`, `release.yml`, `scorecard.yml` |

## Open pull requests

| PR | Head | Base | State at capture |
|---|---|---|---|
| [#71](https://github.com/djshellshoxxx/circula/pull/71) | `b1cd8f38` | `main@ffa2758` | E-02 `trusted_schema(0)`; contains #77 commits; CI running |
| [#72](https://github.com/djshellshoxxx/circula/pull/72) | `faed4906` | `main@ffa2758` | NEW-01 private backup staging; contains #77 commits; CI running |
| [#77](https://github.com/djshellshoxxx/circula/pull/77) | `2f6b3e4f` | `main@9c1b38d` | UX-005 colour-mode parity test clock fix; CI running |
| [#67](https://github.com/djshellshoxxx/circula/pull/67) | `7a528c61` | `future/fnet-204-feed-20261116` | Draft, future module only |

## Branches added since the 2026-11-17 branch ledger

All other branches match [BRANCH-CONSOLIDATION-2026-11-17.md](../../plans/BRANCH-CONSOLIDATION-2026-11-17.md).

| Branch | Tip | Commits not on `main` | Disposition |
|---|---|---|---|
| `claude/circula-pr-consolidation-q6q18n` | `163fe15` | 0 | merged (PR #69) |
| `integration/admin-grant-notice-pagination-20261117` | `cb25b41` | 0 | merged (PR #73) |
| `integration/erasure-fixture-stability-20261117` | `fc31dc9` | 0 | merged (PR #76) |
| `integration/release-threshold-check-20261117` | `ec98cab` | 0 | merged (PR #74) |
| `integration/throttle-documentation-20261117` | `83de536` | 0 | merged (PR #75) |
| `integration/store-untrusted-schema-20261117` | `b1cd8f3` | 4 | open PR #71 |
| `integration/backup-private-staging-20261117` | `faed490` | 7 | open PR #72 |
| `integration/browser-parity-clock-20261117` | `2f6b3e4` | 2 | open PR #77 |
| `claude/circula-consolidation-audit-0sydm1` | `ffa2758` | 0 | this session's branch |

## Open security findings at capture (from `docs/security/security-review.md`)

A3-F02 (Medium, owner decision DEC-014), PRIV-01..05, E-02 (PR #71), NEW-01 (PR #72).

## Open core requirements at capture (from `docs/TODO.md`)

SEC-026 (not built), GOV-021 (not built), UX-047 (partial), QA-042 (partial), L9 storage fault
qualification beyond disk-full and WAL-commit write failure, SEC-014/018 retention questions.
External gates: SEC-033 independent review, UX-056/QA-035 manual screen-reader run, licence, DEC-* decisions.
