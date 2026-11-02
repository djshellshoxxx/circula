# Responsive viewport checks (UX-033, UX-036)

Owned branch: `codex/circula-responsive-qa`.
Input: `9c0bc500cb19f11f18694c76d134f4bfebf896e7`, preserving draft PR #18 and
application `7643fb616208cc318939ded809e6cada66bb7415`. Read the latest tested
source in [the current handoff](../../HANDOFF.md). `git rev-parse HEAD` resolves
the checkout head, which can be a later document-only receipt; earlier
application CI does not validate later tests or receipts.

## Validated viewport checkpoint

Test source `5d2b2c219acbd0d6ba192c42372cc7a8a484ab7b` completed
[push CI 41429404139](https://github.com/djshellshoxxx/circula/actions/runs/41429404139)
with all nine jobs successful. Browser job `125223913309` reports 127 tests
passed, including all 24 new viewport cases. The filtered log contains one
successful result for each new case. Lint/pinned scanning, three OS jobs,
contract/load, fuzz, cross-build and reproducibility passed at that exact source.
No cancelled or older-source job is counted, and no local browser pass is implied.
Later text-preference tests need their own exact-source CI.

The branch is normally published and its remote commit was read back. An attempt
to create a draft against PR #18's branch returned no usable structured receipt;
the orchestration then failed with `Unable to store "responsive_pr". Only plain serializable objects can be stored.` Readback found no PR
for the branch. No permission-denial reason was captured and no create retry was
made. Inspect the original action result/permissions before another create
attempt; do not infer that a draft exists or bypass any denial.

## Scope and ownership

The current suite contains axe, ARIA, keyboard and security tests but no viewport
or orientation checks. The accepted UX-033/UX-036 requirements therefore have an
independent automated-QA slice. This reservation covers only
`e2e/tests/responsive.spec.js`, these notes, the two as-built status notes and this
branch's handoff. No existing fixture, test assertion, runtime/CSS, audit finding,
PoC, workflow, dependency or OPS-004 runbook is edited.

One coordinator is the sole local worker; no subagent or external paid-model call
is used. Current remote refs and PR #12/#17 discussions contain no explicit audit
handover. Reserved runtime modules remain with their owners. No activity status
of an external private session is inferred from unchanged published branches.

## Coverage and limits

[The test](../../e2e/tests/responsive.spec.js) registers 24 cases: 12 seeded pages
at 320×640 CSS pixels (portrait) and 640×320 (landscape): login, registration,
help, member dashboard, directory, new listing, propose trade, completed trade,
history, admin overview, admin members and admin audit.

Each case requires a 200 response at the intended route, one visible heading,
visible main content, no page-level horizontal overflow, and a labelled,
keyboard-focusable region for any horizontally scrolling table. It performs
navigation only and uses existing role fixtures. It does not post ledger entries,
alter roles/policy or change independent-founder/24-hour approval setup.

These are representative browser layout probes, not proof of every core screen,
all possible data, single-column semantics, native-device rotation, text resizing,
text-spacing or full WCAG conformance. UX-033/UX-036 remain partial. Manual
screen-reader work and independent/professional review remain open.

## Commands and evidence

With an existing built binary, cached Node dependencies and Chromium, run the
focused checks without an installation:

```sh
cd e2e
CIRCULA_BIN=/absolute/path/to/bin/circula ./node_modules/.bin/playwright test tests/responsive.spec.js
```

The existing CI route builds the binary and runs `make e2e`, which includes this
file. Its side effects were reconciled with the unchanged source-pinned workflows:
branch push/PR runs CI only; release publication requires a version tag, and
Scorecard push publication is main-only. Smoke processes are temporary loopback
instances. No manual dispatch, release or deployment is requested.

Local checks passed: `node --check e2e/tests/responsive.spec.js`, staged whitespace,
repository hygiene/action pins, 27 local links across the outgoing documents,
fences/conflict markers and a limited complete-document credential/private-data
scan across all four outgoing files, with no findings.
Node/browser dependencies are absent here; previous dependency/artifact HTTP 403
denials remain binding and are not retried. No local browser result is claimed.
Publish a normal source checkpoint and freeze it during existing CI. Record actual
exact-source jobs/results, including failures, before declaring this QA slice
validated. Keep original assertions and any failing evidence; do not weaken
coverage or edit reserved runtime/CSS to make a failure disappear.

## Screen, role and flow reconciliation

Reconciled from test source `32a8314ebd28094dd6d2500c454ed809f6b16b3c` and
handoff `04e11989b811b4541a20b4e06e7a020beabb6af3`. The existing axe page table has
57 screen/role entries; it also allows responses below 500, so it is not a
reviewed authorization or successful-flow inventory. The 34 later cases cover
12 representative positive screens, four preference cases and six section/
viewport Help checks; those counts are not interchangeable with all routes.

| Screen/role/flow group | Existing evidence | Remaining accepted coverage |
|---|---|---|
| Anonymous login, registration, Help | Axe; keyboard sign-in; viewport and Help probes | Scripting-disabled completion; registration/invitation flows remain beyond the new member slice |
| Alice/Bob ordinary members: dashboard, proposal, completed trade, history | Axe/ARIA, keyboard proposal/acceptance, viewport, proposal text preferences, Help | Combined keyboard/no-script member journeys selected below; broader flow/state coverage remains |
| Pending trade: Carol proposes to Bob; Carol is also a reviewer | Axe covers Bob accept/decline and Carol cancellation; ARIA covers Bob's pending/accept pages | These are genuine party-specific screens, not assumed 403 cases; viewport/text preference coverage remains narrower |
| Inbox, reports, time zone | Axe and specific keyboard/focus journeys | Not included in the 12-screen viewport set; wider visual/preference coverage remains |
| Review/dispute states | Axe plus selected keyboard response/decision probes | No archived PoC restoration or reserved dispute/A1 runtime remediation is assigned here |
| Founder/admin overview, members, audit | Axe plus viewport and Help probes | Invitations, policy, exports, reports, adjustments and other admin states have narrower automated coverage |
| Security/MFA/recovery/operator/erasure/dual-control | `security.spec.js` uses its own `secfixture.js` instance | Preserve that isolation and reserved ownership; these are not ordinary member fixture roles |

Roles were checked against `global-setup.js`: Alice/Bob are ordinary members,
Carol is the pending trade's proposer and a dispute reviewer, and admin is the
founder. Security tests explicitly use a separate database, so their account
changes do not invalidate the ordinary-member seed. The new QA slice does not
change that setup, any grant, the 24-hour rules or existing test assertions.

### Selected independent QA-034 slice

At the input source no browser context disabled JavaScript. The existing `as` fixture opens
its own context, so a test-level default alone would not disable scripting for
its role pages. New [no-script journeys](../../e2e/tests/zz-noscript.spec.js)
explicitly create contexts with `javaScriptEnabled: false`, using the
[documented Playwright option](https://playwright.dev/docs/api/class-browser#browser-new-context-option-java-script-enabled).
They reuse the existing `kbd.js` helpers: all page controls are operated by Tab,
typing and Enter, with no click/fill/selectOption or script-driven form changes.
Playwright's DOM inspection remains test instrumentation, not site scripting.

The two cases exercise ordinary-member sign-in/proposal/acceptance/balance/
history and proposal/decline/history flows, using only their own new trade IDs:
one small synthetic trade is completed;
the other is declined without a ledger reference. The seeded trade IDs, policy,
roles and existing security instance are preserved. Contexts are closed on
success or failure. No CSP bypass is enabled for these new contexts.

QA-034 remains partial: registration, invitation acceptance, dispute raise, all
states and manual focus/reading checks are not established by these two cases.
No manual/native or independent gate is satisfied by adding them.

With an existing built binary and cached Node/Chromium dependencies, the focused
command is `cd e2e && CIRCULA_BIN=/absolute/path/to/bin/circula ./node_modules/.bin/playwright test tests/zz-noscript.spec.js`.
Local checks passed: Node syntax, staged whitespace, repository hygiene/action
pins, 36 local links across outgoing documents, fences/conflicts and limited
credential/private-data review of all four complete outgoing files, with no
findings. Exact new-source browser/aggregate CI results remain pending.
No local browser run or retry of denied dependency/artifact downloads is implied.

### Reserved runtime handover

The audit owner must explicitly assign the finding, base, permitted files and
original regression before remediation. In particular, A1-01/A1-06 departure
work concerns `internal/service/departure.go` and its service regressions;
A1-02/A1-11 concerns currency policy/initialisation and related store methods;
C-09/A2-07 concerns API resource ETags and the archived fixed-clock regression.
Verify exact paths against the selected current packet before a handover; this
QA branch does not claim those modules. Preserve archived drafts and PR #13.
