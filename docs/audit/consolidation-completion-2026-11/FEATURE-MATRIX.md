# Feature-completion matrix (2026-11-18)

States come from code, tests and PR evidence. The table does not give percentages: acceptance evidence counts,
estimates do not. Outside the rows below, the core baseline is the 45-row beta matrix in
[circula-completion-matrix.md](../../plans/circula-completion-matrix.md) and the generated
[TRACEABILITY.md](../../TRACEABILITY.md): 531 IDs, of which 19 are marked built but have no test citation.

## Core requirements touched in this pass

| ID | Before | Now | Evidence | Remaining |
|---|---|---|---|---|
| SEC-026 velocity / reciprocal signals | not built | built, not merged | [#86](https://github.com/djshellshoxxx/circula/pull/86): 26 `TestSEC026*` tests across all three acceptance criteria; full race run passes locally | Owner gates: default thresholds (spec 12 / open question 10), the GOV-002 exception for the settings log, hiding a signal from its subject, retention (DEC-020); no reviewed/dismissed state |
| GOV-021 administrator-unavailable recovery | not built | built, not merged | [#85](https://github.com/djshellshoxxx/circula/pull/85): acceptance tests for second approval, network-visible audit and no ledger change; abuse, boundary and operator-listing audit tests; full race run passes locally | Owner gates: SEC-013 decision 7 versus a permanent role, authenticated operator identities, an objection period, the 7-day expiry |
| UX-047 message catalogue | partial | partial, larger | [#79](https://github.com/djshellshoxxx/circula/pull/79) merged: 121 authentication and account-security entries | Setup and join pages, shared refusal text, admin reset pages, other templates, JavaScript, API/service errors, locale scope (owner) |
| E-02 schema trust | open | fixed | #71 | none (no schema fingerprint, by design) |
| NEW-01 backup staging | open | fixed | #72 + #81 | Windows relies on NTFS ACLs; ancestor directories are not checked |
| SEC-003 password change invalidates sessions | built, but racy | fixed, pending CI | [#82](https://github.com/djshellshoxxx/circula/pull/82) (AUD4-F06) | — |
| GOV-001 prospective dispute timelines | violated (AUD4-F03) | fixed, pending CI | [#87](https://github.com/djshellshoxxx/circula/pull/87) | Policy changes in the same millisecond as an opening (residual) |
| DOM-038 reviewer independence | gap (AUD4-F02) | owner decision | [#83](https://github.com/djshellshoxxx/circula/pull/83) draft | Recovery-policy choice |
| SEC-011 dual control | gap (AUD4-F01) | owner decision | [SECURITY-AUDIT](SECURITY-AUDIT.md) | Quorum, operator-rooted approver, or proposer delay |
| SEC-013 / DEC-014 break-glass scope | open (A3-F02) | open | Still reproducible, with more affected actions listed | Owner decision |

## Core work not started in this pass

QA-042 (remaining manual and external citations), L9 storage fault qualification (read, fsync, truncate,
checkpoint and uncertain-commit cases beyond the existing disk-full and WAL-commit coverage), the SEC-014/018
retention questions (PRIV-01..05), and the release thresholds (DEC-023). See [BLOCKERS](BLOCKERS.md).

## Future modules (isolated; `enabled:false`, `integration:HOLD`)

Branch tips were re-verified on 2026-11-18. Each is unchanged since 2026-11-16 or 2026-11-17, and all fork from
`main@2eacf77`. Their core code is stale: the FNET-204 base fails `go vet` at `cmd/circula/auditanchor.go:189`.
Any integration first needs those bases rebased onto current `main`, which is an owner-scheduled step. Nothing
here is part of the core beta decision.

| Module | Branch tip | State |
|---|---|---|
| FUX-101 PWA | `a6326b87` | Isolated implementation tested; Chromium passed; platform qualification pending |
| FUX-102 QR | `9db8a323` | Initial core tested; incomplete |
| FUX-103 discovery | `34d0f889` | Initial core tested; incomplete |
| FUX-104 notifications | `3bdf8e92` | Initial core tested; incomplete |
| FUX-105 messaging | `7f92251c` | Initial core tested; incomplete |
| FUX-106 onboarding | `473cab2f` | Initial core tested; incomplete |
| FNET-201 matching | `12850290` | Initial core tested; incomplete |
| FNET-202 saved partners | `efc9d696` | Initial core tested; incomplete |
| FNET-203 receipts | `bd2fbdbd` | Initial core tested; incomplete |
| FNET-204 feed | `5d3819c3`, plus [#67](https://github.com/djshellshoxxx/circula/pull/67) and [#80](https://github.com/djshellshoxxx/circula/pull/80) | Signed cursors with expiry, key ID/rotation interface, viewer and filter binding, domain separation, wired into `Page()`; 98.8% coverage; 60 s fuzz run (1.36 M executions); owner decisions on key custody, TTL, mute semantics and joins |
| FNET-205 analytics | `0b296510` | Initial core tested; incomplete |
| FNET-206 areas | `8b444b20` | Initial core tested; incomplete |
| FNET-207 recurring | `0437e548` | Initial core tested; incomplete |
| FNET-208 governance | `b4f51137` | Initial core tested; incomplete |
| FNET-209 federation | `1b3c92e0` | Initial core tested; incomplete |

FNET-204 is the only future module advanced in this pass. The other 14 need per-module specification review and
feature work. That is out of reach for this pass alongside the core security work, and is recorded as remaining.
