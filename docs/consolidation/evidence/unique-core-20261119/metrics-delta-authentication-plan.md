# Metrics delta review and bounded authentication continuation

Date: 2026-11-20. Reviewer/implementer: `/root/review_integration` (AI).

## Independent review: final posting and latency delta

Reviewed worker commit `4630a0a7ae05f867ce5889075cb7214ef324b03f`, integrated as
`a29a0943258be14d292a0d0998c0f0661c80d248`. This supplements the earlier,
explicitly uncommitted posting-metrics review in `integration_feature_review.md`.
It does not replace that report's historical source or test provenance.

No actionable blocking finding in this bounded delta:

- The added latency summary uses the existing route template and response
  status-class key, under the existing metrics mutex. It keeps the original
  route-only sum/count series. The HTTP wrapper supplies fixed route labels
  through `routeOf`; this change does not add raw paths or identities to labels.
- The corrected credit-refusal fixtures use the actual `ProposerIsPaid`
  direction. A proposal affordable under the original limit becomes
  unaffordable after the administrator reduces the payer's limit. The acceptance
  reaches the central posting boundary and exercises its ordinary credit
  refusal. The fixture change does not relax production credit rules.
- The optional posting observer preserves the original committed-posting and
  replay observer interface and counters, as covered by the earlier review.

Evidence inspected, not re-executed by this reviewer:
`analysis/posting-metrics-tests.log`, SHA-256
`66065c133b88dd13d254cafbd08ebd543fc0c39c6545da21f08f7ef7a66f5168`.
The complete text log records ten top-level passes: five in service (4.603 s)
and five in cmd/circula (2.595 s). Parent identifies the run as focused race
testing. This is not a full-suite, deployment or external approval claim.

## Remaining OPS-014 gap and approved implementation scope

The existing HTTP counters record 401 and 429 responses. Source inspection
shows ordinary sign-in and MFA-login refusals already use 401/429; an MFA
challenge rendered with 200 is the successful first-factor step. The concrete
coverage gaps are direct service calls, session refusals that redirect to
sign-in, and MFA-management credential errors expressed as typed validation
fields and rendered with 422.

Parent approved a separate optional `AuthenticationObserver` extension with
`AuthenticationFailed(reason)` and `AuthenticationThrottled(reason)` callbacks.
It instruments only the outer credential operations after their transactions
finish, preserving the original `Observer` interface and all authentication
decisions, response messages and rate-limit budgets.

| Class | Instrumented service operation |
|---|---|
| `password` | `LoginStep`; the delegating `Login` wrapper is not instrumented again. |
| `mfa` | `CompleteMFA`. |
| `recovery` | `RecoverPassword`. |
| `session` | `Authenticate`. |
| `api_token` | `AuthenticateToken`. |
| `password_proof` | `ChangePassword`, `GenerateRecoveryCodes`. |
| `mfa_management` | `BeginMFAEnrolment`, `ConfirmMFAEnrolment`, `DisableMFA`, `RegenerateRecoveryCodes`. |
| `other` | Bounded fallback for unexpected observer class input. |

Typed `unauthenticated` errors count as failed credential operations. Typed
`rate-limited` errors count only as throttling, not a second failure. The
`mfa_management` class additionally recognizes typed validation fields named
`password` or `code`. Classification never inspects error messages. Successful
MFA challenges, new-password policy validation, authorization/state refusals,
and storage/cancellation failures do not count as failed credentials.

Both metric families retain only the fixed class and a count. They accept no
credential, identity, address or error text. The HTTP series remain unchanged.
Service counters measure credential operations rather than unique users or
requests: one request can perform multiple checks, and edge refusals can
perform none. Overlapping HTTP and service totals must not be added. Counters
are process-local and reset at restart.

Implementation worktree:
`/workspace/scratch/4d30fa819181/circula-authentication-metrics`, based on
`a29a0943258be14d292a0d0998c0f0661c80d248`. Auth source is the reviewer's own
implementation and is not represented as an independent review. Seven focused
new top-level tests cover the classes, distinct throttle counts, MFA challenge
success, typed-field filtering, storage-failure propagation, wrapper count,
bounded metric labels and the existing HTML 422 response. Fixture credentials
come from existing helpers; invalid values use `Env.Key()`. No scanner
configuration or exception was changed. Execution was pending at this planning
checkpoint; the completion addendum below records the subsequent outcome.

## Frozen-dashboard documentation correction

The missing-dashboard note in `docs/ops/README.md` was stale. Source chain:
`Service.AdminOverview` includes `FrozenInfo`; web `overview` transfers its
time and reason; `templates/admin_home.html` renders the alert and audit link.
The existing `TestFrozenNetworkVisibility` checks the administrator reason,
member notice without the reason, and both banners disappearing after thaw.
`TestAdminOverviewCatalogueControls` separately covers rendered controls.

Only documentation is changed for this item. `frozenSet` remains in memory;
start-up verification repopulates it, periodic verification freezes/thaws
networks, and the displayed freeze time belongs to the current process. This
review did not add persistent freeze history or a second dashboard mechanism.

This report is AI source review and implementation planning, not GitHub
approval, an external security qualification or a full feature-acceptance run.

## Completion addendum: authentication metrics

Committed as `1a216cb51e411bce2d8fa842a7e7881d25ecd2a0` on the isolated branch
`fix/authentication-metrics-20261120`. The worktree is clean. Parent independently
read the observer, named-error defers and fixed-label implementation and reported
no blocking source finding; that review remains an AI review.

After the shared Go caches were relocated from tmpfs to workspace disk, keeping
their original paths through symlinks, this agent ran the unchanged selection
against the full service, cmd/circula and web packages with Go 1.26.11,
`-race -p 1 -count=1`, one CPU, a 256 MiB Go memory target and a workspace
`GOTMPDIR`. All 22 selected top-level tests and two subcases passed, with no
failures, skips, non-JSON compiler output or race warning. Package elapsed times
were 14.915 s (service), 3.958 s (cmd/circula) and 7.166 s (web); total command
wall time, including compilation, was 93.842 s.

This selection includes all seven new authentication tests; existing password,
MFA and store-outage behavior; the original posting observer; existing metrics
latency/series checks; and `TestFrozenNetworkVisibility`. The earlier compiler
kills belonged to another agent's projection branch and are not represented as
authentication test failures or overwritten by this result.

Retained evidence:

- Raw events: `analysis/authentication-metrics-focused.jsonl`, SHA-256
  `694e031c5b409c856a70895213c5bba5be06773ebd1fb73879ffef90c2b900d7`.
- Original execution receipt: `analysis/authentication-metrics-focused-receipt.json`,
  SHA-256 `f3e95e1bec3d5ddd1966eb14b8dd7d8666b27b48649d45a7afec3ec659a8e4dc`.
  It retains the exact command, environment, start/end times, selected names and
  17 file hashes before and after execution. All hashes stayed unchanged.
- `analysis/authentication_metrics_verification.json` maps the final source
  commit and tree to the receipt. Every recorded file, including Go source,
  tests, operations docs and module files, matches its committed bytes.
- `analysis/run_authentication_metrics_tests.py` retains the focused command
  and evidence collection procedure, with overwrite protection.

Formatting and `git diff --check` also passed. No full integrated suite,
post-integration scanner execution or deployed acceptance result is claimed by
this focused implementation receipt; those remain parent-owned gates.
