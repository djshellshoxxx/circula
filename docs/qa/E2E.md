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

* `tests/a11y.spec.js`: 38 pages plus three error or review states, each checked with axe-core for the tags
  `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`. Any violation fails the test and prints rule, selector and HTML. Pages:
  sign-in, register, help, invitation (signed out and in), 404, network list, network creation, password change,
  dashboard, directory, profile, listings (list, new, edit), trades (list, propose, detail, accept, decline, cancel),
  history, dispute form, disputes, dispute detail, review queue, and every admin page. A guard test confirms the harness
  does report a deliberately broken DOM.
* `tests/keyboard.spec.js`: only key presses are used. Checks: skip link is the first Tab stop, is visible when
  focused and moves focus to `main`; tab order on the sign-in page; no positive `tabindex` and a visible focus
  indicator on every Tab stop of five pages; sign in, propose a trade (select by type-ahead, amount, review, send) and
  accept it as the counterparty; the error summary takes focus on a failed sign-in and its link focuses the field.

Axe does not find everything. It covers rule-based checks only, a subset of the criteria. Screen
reader behaviour, reading order sense, zoom and reflow, forced colours and plain-language quality still need the manual
review in `docs/specs/08-ux-accessibility.md`.

The tests inject axe-core with `bypassCSP: true` because the application's `Content-Security-Policy` (correctly)
forbids inline scripts. Nothing else in the application is relaxed.

## Continuous integration note

Not wired into `.github/workflows/release.yml` (owned elsewhere). To run it in a workflow job, after the Go build
step:

```yaml
  e2e:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@<pinned sha>
      - uses: actions/setup-go@<pinned sha>
        with: { go-version-file: go.mod }
      - uses: actions/setup-node@<pinned sha>
        with: { node-version: 22, cache: npm, cache-dependency-path: e2e/package-lock.json }
      - name: Install Chromium (runners have no preinstalled Playwright browser)
        run: cd e2e && npx --yes playwright@1.63.0 install --with-deps chromium
      - run: make build
      - run: cd e2e && npm ci && CIRCULA_BIN=$PWD/../bin/circula CHROMIUM_PATH="$(node -e "console.log(require('@playwright/test').chromium.executablePath())")" npx playwright test
```

Pin every action by commit SHA to satisfy `scripts/check-action-pins.sh`. On a runner the browser is installed by
Playwright; the `CHROMIUM_PATH` shown makes the config use it. Upload `e2e/test-results` on failure to keep the
failure screenshots.
