# Recurring recovery composition — 2026-11-19

**Disabled; integration HOLD. This is an isolated engineering candidate, not a core integration or release.**

## Source identities

| Source | Exact identity | Role |
|---|---|---|
| Original isolated module target | `0437e54897c6d9cfb3f2577807b82dd038493850` | Common ancestor; original daily prototype |
| Reviewed November 19 local candidate | `cdcd2cb38b3af0275e5ceb58242384cfa916be45` | Parent of this local reconciliation branch |
| Equivalent published PR #97 head | `73a911a9bec387381633c031d8a4083c9fc724ad` | Same whole tree `c6a4c5dabdbc482967a44353f14926a24066ad56`; publication predecessor |
| Verified recovered archive | `a755732a02dfe0eaa884691d59a6f748f96039e7` | Archived service, dependency pins, notices, tests and documentation |
| Archived historical tested runtime | `b4b8aec884c53a78a8586e0d94fad95e9416dc53` | No Go/test/go.mod/go.sum delta between this historical tested source and the archived head |

The archive was recovered from the SHA-verified saved recovery bundle. Its immutable ref remains `recovery-20261116/codex/future-recurring-master-20261116`. Neither the reviewed calendar candidate nor the archived service contains the other. Original archive refs and existing published branches are unchanged while this new candidate is reviewed.

## Preserved source and interfaces

The archived service adds explicit SQLite templates/revisions, owner-scoped idempotent mutations, current transactional authorization, atomic firing/reminder insertion, bounded scans, lifecycle retention/departure, manual drafts, unregistered HTTP/native forms and CLI. Its authentication and inbox callbacks are embedding contracts; they are not the actual Circula core adapter. No automatic proposal, acceptance, payment, transport or ledger writer is introduced.

The November 19 `schedule.go` is retained in full, preserving `Template`, `Occurrence`, `Due`, `DueWithSchedule`, structured `Schedule`, exact daily keys, invalid UTF-8 and host-local rejection, gap/fold handling, historical second offsets and a valid zero instant. `schedule_regression_test.go` is retained byte-for-byte. The prior `calendar_test.go` is renamed to `due_with_schedule_test.go`; the archive's distinct `calendar_test.go` is also retained. No unique tests or helper functions were discarded to resolve the filename collision.

The service's `Calendar.Window` and `Calendar.Next` remain separate APIs. Their explicit rejection of a zero clock is preserved, because it is the archived service contract. New reconciliation tests exercise malformed identities, valid literal replacement characters/Unicode, exact historical-second boundaries and a 10,368-case independent UTC-minute oracle spanning all three cadences. The direct helper identity failure and historical second-offset iterator failures were reproduced before repair. `Window` now uses the exact interval resolver; `Next` uses cron only to enumerate eligible dates at fixed UTC noon before resolving actual local wall time. The combined calendar-only race check passes, including 12 explicit next-date acceptance cases; `VALIDATION.md` records commands and limits. Full combined-module local qualification subsequently passed at `e79cd7a3d8bc1830aeb5ee255c044ddfef72f278`; see `RECOVERY-QUALIFICATION-20261119.md` for 48 full-suite tests, both 100,000-execution fuzz budgets and exact source fingerprints.

## Separate service-review follow-up

After the first composition was frozen at `048c7156133bef0b6a7d394c4e7814a975f1d8b2`, source review identified chronological retention and Unicode duplicate-field defects. Actual-driver regressions reproduced premature/late cleanup, nanosecond precision loss in operation retirement, and the related recent-inbox ordering defect. A separate follow-up adds exact canonical-UTC ordering while retaining strict cutoffs, ID tie-breaks, limits, scope and tombstones; the parser comparison uses the decoder's Unicode simple-fold equivalence without changing valid single aliases. Four focused real-driver race tests pass; the later complete combined suite also passed at the source recorded in `RECOVERY-QUALIFICATION-20261119.md`. `VALIDATION.md` records exact results and limits.

At the November 19 recovery snapshot, external PR #110 was a pending handoff for accepted Core baseline files and the fixed-public-digest test naming correction. PR #110 subsequently merged into #97, and #97 merged into the actual recurring target at `94deb0a20df07a3b38c8a905254645d6cd3019f2`. The November 20 composition preserves that accepted history and carries its exact corrected legacy test into `due_with_schedule_test.go`; the recovered `calendar_test.go` remains intact. See `CONSOLIDATION-20261120.md` for this resolved handoff and separate current qualification.

## Dependencies and workflow

`go.mod`, `go.sum` and third-party notices come from the exact archived module. Runtime uses `github.com/robfig/cron/v3 v3.0.1`; SQLite tests use `modernc.org/sqlite v1.60.1`, with the archive's exact indirect requirements. No root module, shared workflow or other future module changes.

Only the owned `.github/workflows/future-recurring.yml` changes outside this directory. Its obsolete standard-library-only check is replaced by an exact requirement/version/indirectness allowlist plus a dependency-import boundary check. It retains the existing immutable checkout/setup-go pins, `contents: read`, disabled checkout credentials, explicit disabled/HOLD checks, bounded jobs and fuzz processes, and native Linux/macOS/Windows coverage. Windows ordinary tests are not race evidence. The full service suite must use the real pinned SQLite driver; parser-only probes cannot substitute for it. The two 100,000-execution fuzz commands explicitly name the calendar production/test files to avoid instrumenting the SQL driver; the preceding complete module suite qualifies SQL behavior separately.

## Evidence and remaining gates

`CALENDAR-VALIDATION-20261119.md` preserves the separate reviewed calendar-only result. The archived validation, audit and handoff files and their raw evidence directories preserve the earlier service history. Their coverage numbers, weighted completion percentages and passes do not qualify this composition. The calendar-only reconciliation pass does not cover SQLite, HTTP or CLI. The new combined local pass is recorded separately in `RECOVERY-QUALIFICATION-20261119.md`; no new native provider result is claimed yet.

Current-core transaction, membership/terms/capacity and same-transaction inbox adapter compatibility; explicit network opt-in; host scheduler restart/fairness orchestration; root API/CLI wiring; actual browser/keyboard/zoom/screen-reader/WCAG acceptance; operational backup/lifecycle erasure; protocol/policy adoption and release decisions remain unresolved integration or manual gates. The recovered quota, 20-hour timezone edit guard, and retention constants are existing candidate policies, not a new owner approval. These gates do not erase the independently recovered implementation or justify replacing its existing code with another prototype.

## Stacked publication and tested source

The original recovery was published as PR #118 targeting `master/fnet207-recurring-remote-20261119`, while PR #97 and external PR #110 were separate workstreams. The dedicated workflow accepts that historical stacked base as well as the actual future target. The November 20 composition is intended for `future/fnet-207-recurring-20261116` after both predecessor PRs merged. Its normal merge preserves both histories and reconciles the test rename without changing recovered production code or dependency pins. The earlier tests at `e79cd7a3d8bc1830aeb5ee255c044ddfef72f278` retain their historical scope; the corrected helper fixture and combined Core/CI tree require their own qualification. Retargeting/publication and normal current-head gates are coordinator actions, not changes performed by this local composition.
