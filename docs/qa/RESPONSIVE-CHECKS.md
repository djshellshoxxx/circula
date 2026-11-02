# Responsive viewport checks (UX-033, UX-036)

Owned branch: `codex/circula-responsive-qa`.
Input: `9c0bc500cb19f11f18694c76d134f4bfebf896e7`, preserving draft PR #18 and
application `7643fb616208cc318939ded809e6cada66bb7415`. Resolve the current test
commit with `git rev-parse HEAD`; earlier application CI does not validate it.

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
the orchestration then failed while storing the response. Readback found no PR
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

## Runtime handover still needed

The audit owner must explicitly assign the finding, base, permitted files and
original regression before remediation. In particular, A1-01/A1-06 departure
work concerns `internal/service/departure.go` and its service regressions;
A1-02/A1-11 concerns currency policy/initialisation and related store methods;
C-09/A2-07 concerns API resource ETags and the archived fixed-clock regression.
Verify exact paths against the selected current packet before a handover; this
QA branch does not claim those modules. Preserve archived drafts and PR #13.
