# Responsive viewport checks (UX-033, UX-036)

Original viewport branch: `codex/circula-responsive-qa`, frozen at
`04e11989b811b4541a20b4e06e7a020beabb6af3` for pending exact PR approval.
Current follow-up owner: `codex/circula-noscript-qa`, derived from that checkpoint.
Input: `9c0bc500cb19f11f18694c76d134f4bfebf896e7`, preserving draft PR #18 and
application `7643fb616208cc318939ded809e6cada66bb7415`. Read the latest tested
source in [the current handoff](../../HANDOFF.md). `git rev-parse HEAD` resolves
the checkout head, which can be a later document-only receipt; earlier
application CI does not validate later tests or receipts.

## Current reading guide

Latest tested source: `d75f31e76c0e1f67dcb744ff8899487b57edda6a`.
Run41451135622 browser job 125296339997: **264 passed / one failed** (265 executed).
The unchanged invitation return-link case fails; no runtime fix, full green
result, merge or review approval is claimed. Read the final execution receipt
and coverage/limits table below. Earlier sections preserve observations at their
named checkpoint and do not supersede newer exact-source evidence. The handoff
resolves tested source versus any later documentation-only head. One coordinator
is the sole worker. Local dependency/artifact403 boundaries remain binding.

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
made. This is a historical publication account, not permission to create a PR.
The parent subsequently reported that automatic review rejected conditional PR
creation because clear exact approval was absent. That approval is pending;
this continuation attempts no PR creation, retry or workaround on either branch.

## Scope and ownership

At the original input the suite contained axe, ARIA, keyboard and security tests
but no viewport/orientation checks. UX-033/UX-036 therefore supplied the initial
independent automated-QA slice. That initial reservation covered
`e2e/tests/responsive.spec.js`, these notes, the two as-built status notes and this
branch's handoff. No existing fixture, test assertion, runtime/CSS, audit finding,
PoC, workflow, dependency or OPS-004 runbook is edited.

One coordinator is the sole local worker; no subagent or external paid-model call
is used. Current remote refs and PR #12/#17 discussions contain no explicit audit
handover. Reserved runtime modules remain with their owners. No activity status
of an external private session is inferred from unchanged published branches.

## Coverage and limits

[The initial test](../../e2e/tests/responsive.spec.js) registered 24 cases: 12 seeded pages
at 320×640 CSS pixels (portrait) and 640×320 (landscape): login, registration,
help, member dashboard, directory, new listing, propose trade, completed trade,
history, admin overview, admin members and admin audit.

The no-script child branch adds four verified party-state screens at both
viewports (eight cases): Bob's pending trade detail, accept form and decline
form, plus Carol's own-proposal cancellation form. `global-setup.js` creates
`tradePending` as Carol → Bob; existing axe/ARIA cases only navigate these forms
and no current test submits a decision for that seeded ID. The new checks are
GET-only and leave the seed proposed. The current viewport matrix therefore
registers 32 cases for 16 screen/role entries; their successful browser evidence
and aggregate CI after one fuzz rerun are recorded below at source `ca908a8`.
Original entries, geometry assertions and fixtures are unchanged.
Local checks for this extension passed: Node syntax, staged whitespace/hygiene,
59 local links across the five outgoing files, fences/conflicts, limited
credential/private-data review and byte-identical existing assertion code.

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
reviewed authorization or successful-flow inventory. At that input the 34 later cases cover
12 representative positive screens, four preference cases and six section/
viewport Help checks; those counts are not interchangeable with all routes.

| Screen/role/flow group | Existing evidence | Remaining accepted coverage |
|---|---|---|
| Anonymous login, registration, Help | Axe; keyboard sign-in; viewport and Help probes | Scripting-disabled completion; registration/invitation flows remain beyond the new member slice |
| Alice/Bob ordinary members: dashboard, proposal, completed trade, history | Axe/ARIA, keyboard proposal/acceptance, viewport, proposal text preferences, Help | Combined keyboard/no-script member journeys selected below; broader flow/state coverage remains |
| Pending trade: Carol proposes to Bob; Carol is also a reviewer | Axe covers Bob accept/decline and Carol cancellation; ARIA covers Bob's pending/accept pages | Eight party-state viewport cases passed at ca908a8; text preference coverage remains narrower |
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
success or failure. These cases configure only script mode and role context
data; the suite's existing CSP configuration is unchanged.

At that checkpoint QA-034 remained partial: registration, invitation acceptance,
dispute raise, all states and manual focus/reading checks were not established by
those two cases. The later continuation below has its own validation status.
No manual/native or independent gate is satisfied by adding them.

With an existing built binary and cached Node/Chromium dependencies, the focused
command is `cd e2e && CIRCULA_BIN=/absolute/path/to/bin/circula ./node_modules/.bin/playwright test tests/zz-noscript.spec.js`.
Local checks passed: Node syntax, staged whitespace, repository hygiene/action
pins, 36 local links across outgoing documents, fences/conflicts and limited
credential/private-data review of all four complete outgoing files, with no
findings. Exact new-source browser/aggregate CI results remain pending.
No local browser run or retry of denied dependency/artifact downloads is implied.

### Failing-before evidence and bounded test correction

At `0666f4636ed70b21d3586e9f3e696f60e3907b34`,
[push run 41439165546](https://github.com/djshellshoxxx/circula/actions/runs/41439165546)
failed browser job `125256412697`: 138 passed and one failed. The new decline
journey passed. The sign-in/propose/accept/history journey reached Bob's balance
card, then its line 65 incorrectly required an always-visible `Credit limit in use`
meter. That original assertion remains preserved in the failed source.

`dashboard.html` renders the meter only inside `.Card.InDebt` and `.Card.UsedPct`.
Bob is the seeded recipient in credit. UX-011 makes that meter optional and
requires debt-use text only for a member in debt; it does not require this meter
on Bob's card. The assertion had been copied from Alice's debt-specific ARIA
check and added an unsupported requirement. No runtime defect is established.

The correction replaces only that new invalid meter assertion with UX-009's
required balance fields: direction text, credit limit, reserved outgoing amount
and available to spend. All flow assertions and the scripting-disabled setting
remain; no fixture, runtime/CSS, timeout, existing test or policy is altered.
Its subsequent exact-source validation is recorded below.

Failure artifact metadata: `e2e-test-results`, ID `13139080390`, 138629 bytes,
expires `2026-11-09T11:48:39Z`. Screenshot/error-context bytes were not retrieved;
the earlier artifact-byte HTTP 403 boundary remains binding. The decoded job log,
source/template and accepted requirement supply the diagnosis above.

At corrected source `de156e4e098991c9e393bce932f73c689c100d09`,
[push run 41440673562](https://github.com/djshellshoxxx/circula/actions/runs/41440673562)
passed all nine jobs. Browser job `125261430576` reports 139 passed, including
both no-script journeys. All previous browser cases remained intact. This
validates the bounded test correction at that exact source, not the subsequent
eight party-state viewport additions or any runtime remediation.

### Party-state viewport checkpoint and fuzz gate

At source `ca908a8b5743c7d674b4bac4b492c2a5c7c46698`,
[push run 41442176782](https://github.com/djshellshoxxx/circula/actions/runs/41442176782)
attempt 1 passed eight jobs, including browser job `125266472066`: 147 passed,
with all eight added party-state viewport cases and both no-script cases.
The source is frozen; no aggregate pass is claimed at this observation.

Fuzz job `125266472051` failed `FuzzParseAmount` at 10.10 seconds with
`context deadline exceeded`, after 779943 executions. It reported no failing
input or panic. Baseline coverage completed 21/21 inputs; the later JSON fuzz
command did not run after this failure. No dependency/artifact HTTP 403 occurred
in this job. These facts suggest a possible cutoff/scheduling issue; they do not
establish a benign flake or a runtime defect.

One failed-job-only rerun was requested at the same source, with the unchanged
command and 10-second budget. Run attempt 2 completed successfully. Fuzz job
`125272062062` passed both `FuzzParseAmount` and `FuzzStrictUnmarshal`, each with
the original 10-second budget. No code, assertion, workflow or limit changed;
there was no source push or further retry to obtain this result.

The attempt-2 API lists nine successful job entries with new IDs. Their execution
times show that eight retained their original 12:07–12:14 UTC results, while fuzz
alone executed at 12:20–12:21 UTC. The returned browser log also retains the
original 147-pass timestamps. Treat those eight as carried-forward successful
results, not fresh reruns. This corrects an initial interpretation of the job-ID
listing. Overall CI is successful at the same immutable source; the initial
cutoff failure remains a CI-reliability follow-up, not a defect claimed fixed.

### Reserved runtime handover details

The audit owner must explicitly assign the finding, base, permitted files and
original regression before remediation. In particular, A1-01/A1-06 departure
work concerns `internal/service/departure.go` and its service regressions;
A1-02/A1-11 concerns currency policy/initialisation and related store methods;
C-09/A2-07 concerns API resource ETags and the archived fixed-clock regression.
Verify exact paths against the selected current packet before a handover; this
QA branch does not claim those modules. Preserve archived drafts and PR #13.

## Registration, invitation, dispute and wider-screen continuation

Input receipt: `5337c443d3b4f5d839e4d9b6f5f0854011cef53b`; application and reserved
runtime are unchanged. On resumption the owned remote matched that receipt,
the earlier responsive branch still matched `04e1198`, and the external integrated
branch still matched `827e651fe151f49659cc20aefe85d4af67bdaa97`. PR #12's published
comments supplied no explicit runtime handover. One worker continues only the
two owned browser-test files and these existing notes/handoff. No PR is created.

QA-034 explicitly lists registration, invitation acceptance and dispute raising.
Two additional combined keyboard/no-script cases now register in
`zz-noscript.spec.js` (four total), with validation pending at this source:

- A founder creates a new synthetic invitation through the UI with its existing
  seven-day default. An anonymous visitor follows its account-creation link,
  registers, signs in, returns to that exact invitation, joins and finds their
  display name in Directory. The original unused invitation is not redeemed;
  its token is neither hard-coded nor logged. No role/policy/grant changes occur.
- Alice and Bob complete their own new 0.03 synthetic trade. Alice reviews and
  opens a dispute, Bob responds once, both can read that response, and Alice's
  balance card is unchanged by the dispute. Their own history row is Disputed.
  No seeded dispute is mutated, decision/reversal made or reserved PoC run.

These cases use the same explicit `javaScriptEnabled: false` contexts and keyboard
helpers as the original cases, with context cleanup in `finally`. Runtime/CSS,
original assertions/helpers/fixtures and workflow budgets are unchanged.

The expanded viewport table registers 55 requested screen/role/query entries,
110 cases at the existing two sizes, drawn from the original 57-entry axe inventory
apart from the deliberate not-found page and duplicate admin-overview entry.
The later runtime observation below corrects the root-path entry: with one
membership it redirects to the dashboard, not a distinct network-list screen.
It uses only GET navigation, requires a 200 response, intended path, visible main
and heading, no page overflow and the original scrolling-table accessibility
assertions. This is a screen/role sample, not all possible data/state permutations.
The 39 added entries produce 78 cases; with the two new journeys the expected
suite grows from 147 to 227, subject to actual execution. It is not a pass count.

Local checks passed: Node syntax, 110/four unique static registrations, unchanged
existing test assertions, whitespace, hygiene/action pins and scope checks.
Local browser dependencies remain absent; no denied download is retried. Existing
branch-push CI is the permitted execution route. Exact-source results, any new
failure and unexecuted gates must be recorded before calling these cases passed.

Remaining coverage includes 200-percent-text/spacing beyond the proposal sample,
negative/empty/loading states, every-step visible-focus evidence, wider browser/
native-device permutations and manual screen-reader/independent review. Adding
positive layout probes does not validate submissions of admin/policy/reversal
forms. Any failure requiring reserved runtime/CSS changes is preserved and parked
for a bounded owner handover, without weakening its assertion. The earlier invalid
meter assertion and unchanged fuzz deadline failure above remain original evidence;
no repeated fuzz retry or reliability fix is authorised by this continuation.

### First execution: four preserved failures

Source `d93a97b314e4ced620a05a0eeae5c59cb4ad920e`,
[run 41447665095](https://github.com/djshellshoxxx/circula/actions/runs/41447665095),
attempt 1: eight jobs succeeded, including both original 10s fuzz commands.
Browser job `125284693199` reported 223 passed and four failed (227 executed).
No job was retried. All 147 earlier browser cases passed. The failures were:

| Cases | Observed failure | Diagnosis and disposition |
|---|---|---|
| Root path as Carol, both sizes | Requested `/`, received `/n/demo/`; expected exact root | Test assumption error: `h_home.go` intentionally redirects a single membership. Correct only the new entry's label and explicitly expected destination. Preserve exact-path, status and layout assertions; no arbitrary redirect allowance. Original failures remain at d93a97b. |
| New no-script invitation/account return path | After Create an account, `next` contains percent-encoded `/join/...` after URLSearchParams already decodes it once | Runtime candidate reproduced in normal synthetic UI. `join_anon.html` calls `qs`, which is `url.QueryEscape` in `render.go`, inside html/template's URL query context. No runtime fix or extra decode is applied. Original failing case stays active and byte-identical. Registration/sign-in/join after this assertion were unexecuted in this case. |
| New no-script dispute balance comparison | Rendered innerText was compared to default textContent; the latter includes a hidden meter fallback plus different element separators | Test instrumentation error: compare innerText on both sides (`useInnerText: true`), preserving full balance equality, amounts and limits. Proposal/accept/open/response completed before the assertion failed; final History assertion was unexecuted. Original source and diff remain at d93a97b. |

Failure artifact, from the job upload receipt: `e2e-test-results`, ID
`13143965652`, 324482 bytes, SHA256
`117cffc98de01c8561fd6177f578b37c18ab9a4ab246536c3b7cd8c771f891e5`.
Workflow retention is seven days. No artifact bytes were requested; the existing
403 boundary remains. Synthetic invitation token values are omitted from notes.

The network-list label in the original axe inventory therefore was not evidence
of that screen. The corrected 55-request matrix has 54 distinct final positive
role/path/query destinations; the one-network root redirect is separately tested.
An empty network-list state is instead exercised by a fresh account in the next
text-preference cases. This does not establish multi-network switching coverage.

### Next independent source: validation and text preferences

New-source browser/aggregate results are pending. The new source retains the
invitation failure with its exact original assertion and contains only two
justified test corrections above, plus these independent accepted checks:

- One no-script invalid-proposal case: alert and retained amount/note, native
  keyboard error-link navigation to Amount, a visible focus indicator, correction
  and successful preview. It never sends the trade; no automatic focus move is
  claimed with scripting disabled.
- Sixteen text-preference variants (four flow groups × two sizes × two settings),
  added to the original four proposal probes. Plain registration/sign-in then
  explicit invitation acceptance; fresh proposal/acceptance/dispute raise/balance/
  history; fresh proposal/decline/history with no posting; and dispute preview/
  edit retaining the statement. Every new document gets the same preference,
  and actual heading/control font or spacing changes are asserted. The plain
  registration/explicit-invitation paths do not click or validate the broken
  return link; that separate original regression remains active. Only own new
  accounts/invitations/trades are mutated. Seeded dispute preview/edit is not
  submitted with Open dispute; no reviewer decision or reversal occurs.
- Eighteen keyboard/no-script focus probes for the nine named core-flow entry
  screens at both viewports. They walk Tab stops without activating controls,
  check no positive tabindex, a visible ring, viewport intersection and a sampled
  unoccluded point, and require traversal to terminate/wrap within 120 stops.
  Sampling is not a manual focus/reading-order or full-occlusion proof.

Static enumeration registers 110 viewport, 20 text-preference, five no-script
journey/validation and 18 focus cases. The expected aggregate is 262; actual CI
must replace that expectation. Existing baseline tests, helper/fixture code,
workflow budgets and runtime/CSS remain unchanged. The known return-path failure
means this source cannot be called green or merge-ready even if independent cases
pass. Do not skip/xfail/delete/decode around that regression to obtain a green gate.

Smallest runtime-owner action: explicitly hand over the invitation return-path
finding at source d93a97b, bounded production files (at least
`internal/web/templates/join_anon.html` and `internal/web/render.go`, verifying
other `next` templates with the owner), and this original regression. A reviewed
runtime fix must pass the unchanged case and preserve local-path validation and
security gates. Neither generic continuation nor QA publication supplies that
handover. PR creation remains separately paused pending exact approval.

### Independent checks validated; return-link failure preserved

At source `6a4aa6190dcb001fad4ba6aa5bb5e9fd366ddc02`,
[run 41449741715](https://github.com/djshellshoxxx/circula/actions/runs/41449741715),
browser job `125291709252` executed 262 cases: 261 passed and one failed.
All 110 viewport, 20 preference and 18 no-script focus probes passed. Four of
five no-script journey/validation cases passed, including the corrected balance
comparison and the new native error-link/correction case. The exact original
return-link assertion at line108 failed again with a double-encoded `next`;
its later registration/sign-in/join steps did not execute. This is reproducible
across two sources/runs with different synthetic invitations, not a fixed defect.
No job rerun was requested. Aggregate platform-job completion is recorded in the
current handoff when known; browser success is not claimed.

Failure upload receipt: artifact `13143893566`, 116744 bytes, SHA256
`5779f04c8c76fffd394e6bdfe7d4fa06684c7cb6a9e81721c456a17b214059bb`.
Seven-day retention; no artifact bytes fetched. The later next-path token is
omitted. This record corrects the earlier pending status for this exact source.

### Final prepared independent state coverage

The next source adds one keyboard/no-script plain-registration → sign-in →
explicit invitation join → balance → empty-history case, using its own fresh
synthetic account and invitation. Empty network membership is asserted before
joining, then empty pending/posted history is asserted afterwards. Direct URL
entry is explicit setup for this separate invitation-acceptance path; it does not
activate or repair the defective return link. The original failing case remains
active, unchanged and separately reported. No membership/policy fixture is changed.

Two read-only not-found-state probes add 404 status, exact route, visible content,
no horizontal overflow and an operable Help recovery link at both viewports.
They extend negative-state coverage without broadening authorization or testing
hidden resources. All 110 original positive viewport assertions remain intact.
The next source statically registers 112 viewport/error cases, 20 preference,
six no-script and 18 focus cases: 265 expected total, not a pass count.
Exact-source runtime/aggregate results remain pending.

| Accepted QA slice | Recorded coverage | Remaining boundary |
|---|---|---|
| QA-034 named core flows | Five combined key-only/script-disabled journey/validation cases pass at d75f31e, including plain registration and explicit join | Original invitation return-link regression fails; manual/transition review remains |
| QA-034/UX-040 focus |18 key-only/script-disabled probes on nine entry screens at both sizes passed at d75f31e | Entry-screen sampling does not establish every subsequent state, reading order, screen-reader or native/browser combination |
| UX-033/UX-036 |110 positive layout probes across 55 requests/54 final destinations plus two 404/Help probes passed at d75f31e | Other data permutations and native orientation remain unverified |
| UX-034 |20 enlargement/spacing probes passed at d75f31e, including complete own-trade and explicit onboarding paths | DOM style injection in Chromium is not native browser/OS text-settings certification or all admin/mutation functions |
| UX-022/UX-028 | Script-disabled invalid-proposal recovery and fresh-account empty membership/history, plus 404/Help checks pass at d75f31e | All refusal codes and loading implementations are not established |
| Manual and ownership gates | No runtime/CSS edits, PoC restore, independent review or deployment | Reserved return-path/runtime handover, manual/native/screen-reader/professional review, operator/target authority and exact PR approval remain required |

The executable continuation is bounded to these accepted ordinary-core UX slices.
Do not interpret this ledger as a whole-codebase/security audit or launch approval,
restore archived tests, or invent grant/policy/financial scenarios to inflate coverage.
Further reserved runtime/security/lifecycle work requires its existing owner;
manual/native gates require the named human/device/tool evidence and cannot be
marked complete by these browser probes. Preserve every prior failure and threshold.

### Final execution receipt: 265 cases, one preserved runtime gate

Source `d75f31e76c0e1f67dcb744ff8899487b57edda6a`,
[run 41451135622](https://github.com/djshellshoxxx/circula/actions/runs/41451135622),
attempt1, browser job `125296339997`: **264 passed, one failed** (5.4 minutes).
All nine jobs completed: the other eight passed (lint/scanning, three OS jobs,
cross-build, reproducibility, contract/load and fuzz). Overall run conclusion is
failure, with no job reruns. Both original 10-second fuzz commands passed.
The decoded successful-case list contains 112 viewport/error, 20 preference,
18 no-script focus and five of six no-script journey/validation cases. The three
final state additions all passed, including direct no-script onboarding and both
404/Help recovery probes. This supersedes their earlier pending observations.
All earlier baseline browser cases still pass. The suite has grown from 147 at
the input receipt to 265: 118 added cases,117 passing and one preserved failure.

The sole failing case is the byte-identical original return-link regression at
`e2e/tests/zz-noscript.spec.js:108`. Its `next` query value is still encoded after
URLSearchParams parsing; its later registration/sign-in/join steps remain
unexecuted. Plain registration and explicit invitation acceptance now have their
own passing keyboard/no-script evidence; they do not repair or certify that
return-link path. Neither extra decoding nor skip/xfail/deletion was introduced.
The same failure now reproduces across all three genuinely changed source
checkpoints and independent synthetic invitations. No unchanged workflow/fuzz job
was rerun to obtain a green status; there is no green aggregate claim.

Final failure upload receipt: artifact `13145300826`, 116458 bytes, SHA256
`e271b1feebf28e932ad12febb92c07a917643d9a4ecdcf285ee8327a3442557c`.
Retention remains seven days. No artifact bytes were fetched or denied route
retried; no synthetic invitation token is copied into these notes. The earlier
invalid-meter assertion and fuzz deadline failure remain preserved above and in
Git, with no reliability/runtime fix claimed.

The executable named ordinary-core UX continuation is complete as tested, apart
from the explicitly reserved return-link path. This is a bounded automated-QA
milestone. Other data/state permutations, admin/security/lifecycle mutations,
loading behaviour, multi-network fixtures, native browser/device settings and
manual/screen-reader/independent/professional review are not established. No full
feature, accessibility, security, integration or release-completion claim follows.
The smallest next action remains the explicit bounded runtime-owner handover
specified above, then a reviewed fix against the unchanged original regression.
PR creation has its own pending exact approval and is not attempted here.

Final ref/PR readback: PR12 is closed/merged (audit evidence); PR17 remains open/
draft, head 7643fb616208cc318939ded809e6cada66bb7415, base external 827e651. PR18
remains open/draft, head 9c0bc500cb19f11f18694c76d134f4bfebf896e7, base PR17.
The earlier responsive branch is still frozen at 04e11989b811b4541a20b4e06e7a020beabb6af3.
No PR creation, comment, approval, merge or deployment occurred in this continuation.
Merged audit evidence and an unchanged external head do not transfer runtime ownership
or establish an external private writer's inactivity. The final receipt is docs-only;
CI evidence above belongs to d75f31e. Remote receipt SHA is verified after publication.

## UX-002 continuation: measured paths from Home

Resumed from clean published receipt
`f51295dd9bcce68467fcf928ff9d3de235db21ef`. The owned, frozen responsive,
application, PR18 and external integrated refs still matched their recorded
heads. Published PR12/PR17 comments supplied no explicit runtime handover;
external private writer activity remains unverified. Generic continuation does
not transfer the reserved return-link/runtime/CSS/A1/pagination work or approve
PR creation. One coordinator continues in the existing executor.

UX-002's normative text in spec08 requires screens to be reachable **from Home**
in no more than three link/button activations. The older as-built row's shorthand
was ambiguous; the new test measures this direction, not a return-to-Home path.
No existing test recorded this activation budget. This is an accepted independent
QA gap; it creates no new product feature or runtime behaviour.

[The new navigation test](../../e2e/tests/navigation.spec.js) registers60 cases:
29 core/adjacent destinations at both existing viewports, plus two fresh-trade
routes to the dispute form. Each starts with a successful Home page and balance
card, then clicks only visible, actually rendered anchors. One to three explicit
link destinations are allowed; every navigation requires status200, the exact
intended path/query and a visible single heading. No direct goto, form submission
or programmatic routing occurs inside the measured path.

The ordinary matrix only navigates GET pages as Alice, Bob, Carol or admin.
For the dispute-form path, bounded UI setup first creates/completes a new0.07
synthetic Alice→Bob trade for that variant. It never mutates a seeded transaction.
The measured path then starts afresh: Home→History→that trade→Raise a dispute,
exactly three links. It stops at the form; no dispute/decision/reversal is posted.
Existing helpers, fixtures, baseline assertions, runtime/CSS, dependencies,
workflows, roles, financial policy and original budgets remain unchanged.

Local Node syntax,60 unique static registrations and whitespace passed. New-source
browser/aggregate results are pending;325 is the expected suite count (265+60),
not an executed/pass count. Existing local dependency/artifact403 boundaries
remain binding and are not retried. Reviewed branch-push CI is the execution
route; release requires a version tag and Scorecard push publication is main-only.
No PR creation, dispatch, merge, release or deployment is requested or attempted.

UX-002 remains partial even if the matrix passes: this is the seeded active-role
core/adjacent sample, not every signup/join mode, authentication/session/error
state, multi-network fixture or primary-action submission. It does not prove a
missing chosen link implies no other valid three-activation path; any failure
requires the current UI/fixture and accepted requirement to be checked first.
Reserved runtime fixes still require handover. The original invitation-return
regression remains active and unmodified; the source cannot be called green or
merge-ready while it fails. Earlier invalid-meter and fuzz-deadline evidence is
preserved; no unchanged fuzz/workflow rerun is used to obtain green.
