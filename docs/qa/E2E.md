# End-to-end accessibility and keyboard tests

Directory: `e2e/`. Runner: Playwright Test 1.63.0 with axe-core 4.13.0 (exact versions pinned in
`e2e/package.json` and `e2e/package-lock.json`). Browser: headless Chromium that is already installed
(`/opt/pw-browsers/chromium-1194/chrome-linux/chrome` by default; override with `CHROMIUM_PATH`). No browser is
downloaded: `PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1` is set by `make e2e-install`.

## Run

```
make e2e            # builds bin/circula, runs npm ci, then playwright test
make e2e-install    # only npm ci
cd e2e && CIRCULA_BIN=../bin/circula npx playwright test tests/keyboard.spec.js   # one file
```

Needs Node 20 or later (tested with 22.22.0) and network access to the npm registry for `npm ci` the first time.

## What it does

`e2e/global-setup.js` starts the real binary (`circula serve`) on a free loopback port with a temporary database,
`CIRCULA_MIGRATE_MODE=auto`, `CIRCULA_COOKIE_SECURE=false` (plain http on loopback) and a setup secret. It then seeds
through the web forms, as a person would: it registers four accounts, creates the network `demo` through `/setup`
(using the setup secret), creates four invitations and joins three members, completes two trades, leaves one proposal
pending, publishes a listing and opens a dispute. The binary and the temporary directory are removed at the end.

* `tests/security.spec.js` (own instance and database, built by `secfixture.js`; needs Node 22 for `node:sqlite`):
  axe plus keyboard-only journeys for the Security page, two-step set-up and sign-in challenge, recovery codes and
  account recovery, administrator password reset, operator access (break-glass session opened with the CLI),
  dual-control pending adjustments (approve by a second administrator), administrator and self erasure, password
  change, and a frozen network (the ledger is broken in the stopped database, the restart freezes it): admin banner,
  member read-only notice, and the 423 page.
* `tests/a11y.spec.js`: 57 pages plus 9 error, review, saved and maintenance states, each checked with axe-core for the tags
  `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`. Any violation fails the test and prints rule, selector and HTML. Pages:
  sign-in, register, help, invitation (signed out and in), 404, network list, network creation, password change,
  dashboard, directory, profile, listings (list, new, edit), trades (list, propose, detail, accept, decline, cancel),
  history, dispute form, disputes, dispute detail, review queue, every admin page, and the pages added later: inbox, report form, time zone, dispute response, resolved and re-review states (several roles), admin overview, reports, announcements and reversal (form, error, review step), the dispute decision review step, and the 503 maintenance page (served by a second instance started with the maintenance setting, which also checks that GET still works). The setup also seeds an announcement, a report and three disputes (open, resolved with response, re-review requested). A guard test confirms the harness
  does report a deliberately broken DOM.
* `tests/keyboard.spec.js`: only key presses are used. Checks: skip link is the first Tab stop, is visible when
  focused and moves focus to `main`; tab order on the sign-in page; no positive `tabindex` and a visible focus
  indicator on every Tab stop of five pages, then of the inbox, report, time zone, dispute and admin overview, reports, announcements and reverse pages; respond to a dispute, set the time zone, mark notifications read and reverse a trade by keyboard; sign in, propose a trade (select by type-ahead, amount, review, send) and
  accept it as the counterparty; the error summary takes focus on a failed sign-in and its link focuses the field.

Axe does not find everything. It covers rule-based checks only, a subset of the criteria. Screen
reader behaviour, reading order sense, zoom and reflow, forced colours and plain-language quality still need the manual
review in `docs/specs/08-ux-accessibility.md`.

The tests inject axe-core with `bypassCSP: true` because the application's `Content-Security-Policy` (correctly)
forbids inline scripts. Nothing else in the application is relaxed.

## CI and recorded run

The source of truth for CI is [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml). Its `e2e` job installs the
locked npm dependencies and Chromium, then runs `make e2e` on Ubuntu; on failure it retains the Playwright result
artifacts. The workflow also has a separate contract/load job. Keep this document aligned with that workflow rather
than copying a second hypothetical workflow configuration here.

Recorded checkpoint result (2026-10-24): [workflow run 39750077129](https://github.com/djshellshoxxx/circula/actions/runs/39750077129)
for commit `d39bfef12d4ac1430815fa49ad799fb1c4bf9964` reports the E2E job failed and the other eight jobs succeeded.
This is the outcome of that specific run, not a current passing result or a release attestation.

## CI and ARIA proxy tests
`ci.yml` job `e2e` runs `make e2e` on ubuntu-24.04 after `npx playwright install --with-deps chromium` (the only place a
browser is downloaded; locally `/opt/pw-browsers` is used). Job `contract` runs `make contract` and a small load smoke
(`CIRCULA_LOAD=1`, 4 workers, 20 members, 25 trades each) plus the QA-008 test.
`tests/aria.spec.js` (16 tests) is a screen-reader proxy based on ARIA snapshots; it is not a screen-reader run
(docs/qa/SCREEN-READER.md). Total with it: 93 e2e tests.
