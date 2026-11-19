# Remaining blockers (owner decisions and external gates only)

Engineering items that need no decision are listed in [FEATURE-MATRIX](FEATURE-MATRIX.md) and `docs/TODO.md`.

## Owner decisions

| # | Decision | Blocks | Options recorded in |
|---|---|---|---|
| 1 | Break-glass scope and second approver (DEC-014) | A3-F02. A break-glass administrator can still adjust balances, change credit limits, approve dual-control requests, reset passwords, reverse trades and create 90-day invitations | [SECURITY-AUDIT](SECURITY-AUDIT.md), `docs/specs/addenda/SEC-013.md` |
| 2 | Independence of dual-control administrators (SEC-011 quorum, operator-rooted approver, or proposer delay) | AUD4-F01: an appointer can approve requests from a puppet it appointed | [AUD4-AUTH-FIXES](AUD4-AUTH-FIXES.md) (from #82) |
| 3 | Reviewer account recovery (operator-only reset, MFA for reviewers, or ineligibility after an admin reset) | AUD4-F02; draft [#83](https://github.com/djshellshoxxx/circula/pull/83) | [AUD4-AUTH-FIXES](AUD4-AUTH-FIXES.md) (from #82) |
| 4 | GOV-021 alignment: SEC-013 decision 7 versus a permanent role; authenticated operator identities; objection period; expiry value | Merging [#85](https://github.com/djshellshoxxx/circula/pull/85) | [GOV-021.md at `59cf699`](https://github.com/djshellshoxxx/circula/blob/59cf699/docs/audit/consolidation-completion-2026-11/GOV-021.md) (#85) |
| 5 | SEC-026 defaults, the GOV-002 settings-log exception, hiding signals from their subjects, a second approver for loosening thresholds, retention | Merging [#86](https://github.com/djshellshoxxx/circula/pull/86) | [SEC-026.md at `4d349c4`](https://github.com/djshellshoxxx/circula/blob/4d349c4/docs/audit/consolidation-completion-2026-11/SEC-026.md) (#86) |
| 6 | Session rotation on role change (SEC-003): amend the spec or implement | AUD4-F07 (Info) | [SECURITY-AUDIT](SECURITY-AUDIT.md) |
| 7 | Retention and erasure of `last_seen_at`, `time_zone`, `challenges.reason`/`rationale`; free-text audit reasons; WAL checkpoint after erasure (DEC-020, SEC-014/018) | PRIV-01..05 | `docs/security/security-review.md` |
| 8 | FNET-204 key custody, rotation cadence, cursor TTL, mute/follow semantics, core joins; rebasing the stale `future/*` bases | Future integration only | [#80](https://github.com/djshellshoxxx/circula/pull/80), `future/feed/README.md` |
| 9 | Licence; DEC-005/011/013/021/022/023/025; UX-047 locale scope; release thresholds; pilot jurisdictions | Release | `docs/DECISIONS.md`, `docs/TODO.md` |

## External gates

- SEC-033: independent professional security review. The AI-assisted review in this pass does not satisfy it.
- UX-056 / QA-035: manual screen-reader qualification.
- First tagged release and deployment qualification. Not attempted: this pass made no release and no deployment.
