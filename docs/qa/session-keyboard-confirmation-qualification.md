# Session confirmation keyboard-helper qualification

## Confirmed fixture defect

PR #99's actual [browser job](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/job/135522757115)
reported **587 passed, one failed**. After the session-revocation fixture
intentionally omitted confirmation, the positive keyboard submission remained
on the confirmation URL. The downloaded
[failure artifact](https://github.com/djshellshoxxx/circula/actions/runs/44513300863/artifacts/14533534257)
has 200,778 bytes and SHA-256
`eb84858ee7a7c68a2fbdeda29845a3ad3f7be23789d714d7088f9eaf8ac33576`, matching
the provider upload log.

Its [actual accessibility snapshot](../consolidation/evidence/session-keyboard-control-target/provider-accessibility-snapshot.yml)
shows the error-summary link **“I want to end this session.”** before a native
checkbox with the same name. The checkbox remains unchecked and invalid, and
the page reports missing confirmation. The helper's `tabTo` matched only
accessible text; `kbSubmit` then pressed Space while the link was focused.
Space did not check the checkbox. The next Enter submission therefore repeated
the missing-confirmation response.

This fixture failure is distinct from a demonstrated product authorization or
CSRF defect. Authenticated form tokens are HMAC-bound to the session cookie in
`internal/web/session.go` and `internal/httpx/httpx.go`. The rejected-CSRF path
in `middleware.go` renders an error and does not call `rotateCSRF`.

PR #119 initially proposed a reload after its invalid-CSRF request. That reload
occurs before the intentional missing-confirmation submission, so the later
response still contains the ambiguous link and checkbox. It is not adopted.
PR #119's three Go fixture changes overlap the current migration, actual-message
loader and query-only WAL witnesses; those stronger current fixtures remain
unchanged. Its later audit-representation rewrites are also not adopted; the
original inventory/evidence bytes remain preserved.

## Minimal correction

For boolean fields, `kbSubmit` now asks `tabTo` to require a focused native
checkbox as well as the requested name. Traversal still uses real Tab presses,
the checkbox still uses Space, and submission still uses Enter. The bounded
80-press search remains. Ordinary `tabTo` callers can still target links,
regions, radio buttons and selects; text entry and intentional link activation
retain their previous behavior. Product templates, service calls, confirmation
requirements, CSRF handling and session policies are unchanged.

## Observed checks and limits

The database-free Node regression imports the actual `e2e/kbd.js` helper and
executes its browser callbacks with a small page/DOM adapter. The colliding
names and order come from the captured provider page. The adapter models focus
and key effects; it does not replace real browser qualification.

- Test-only baseline `e0da8bb22b0499deac8f0bfcc170c6fee6696238`: **three failed,
  three passed**. Both individual/bulk checkbox-name collisions and the
  missing-checkbox negative control fail; existing ordinary checkbox, text,
  link and bounded-search controls pass.
- Corrected helper: **all thirteen tests pass** in the combined Node command,
  comprising six helper tests and seven existing progressive-enhancement tests.
  Wall time was 0.423 seconds. The same command is wired into the existing CI
  Node step.
- `git diff --check` passes. No Go build or application change was made.

Exact commands, source blobs, runtime version and output are preserved in the
[before receipt](../consolidation/evidence/session-keyboard-control-target/helper-before.json)
and [after receipt](../consolidation/evidence/session-keyboard-control-target/helper-after.json).
The [provider receipt](../consolidation/evidence/session-keyboard-control-target/provider-browser-evidence.json)
binds the original failure and selected raw log to its exact PR head and artifact.

The first planned live HTTP probe could not start its existing binary after an
executor restart cleared `/tmp`; it returned `FileNotFoundError` before server
startup. That attempt supplies no HTTP result. The full Playwright rerun is
still required. **No browser after-pass is claimed.**

## Subsequent HTTP qualification

After the frozen binary was restored, the bounded
[HTTP probe](../consolidation/evidence/session-keyboard-control-target/probe_http.py)
passed in **1.721 seconds**. The executable is from clean source
`e0d27cc7cb73299219439919b260991b824d594a`, with SHA-256
`b93018a474007d3c2190b169ae9735121f5165632826894b94d4c6919fad3855`.
The probe starts a separate server with a fresh temporary database and uses two
real HTTP cookie jars. The preserved
[result](../consolidation/evidence/session-keyboard-control-target/http-probe-result.json)
and [receipt](../consolidation/evidence/session-keyboard-control-target/http-probe-receipt.json)
record these observed outcomes:

- Invalid CSRF returns **403**, sets no cookies, and leaves the session cookie,
  CSRF cookie and authenticated form token unchanged.
- Missing confirmation returns **422**. The actual HTML contains the matching
  error-summary link before the unchecked confirmation checkbox. A reload
  followed by another missing-confirmation POST produces the same structure.
- The original form token with confirmation succeeds with **303** to
  `/account/sessions`. The revoked session subsequently receives **401** from
  the sessions API, while the current session still receives **200**.

The [HTML control capture](../consolidation/evidence/session-keyboard-control-target/session-revoke-error-controls.json)
retains DOM order and label text with the dynamic session selector replaced by
`fixture`. It is an HTML-parser capture, includes controls under hidden
ancestors, and makes no claim about computed layout or browser focus order.
The actual provider accessibility snapshot independently establishes that the
colliding link and checkbox are exposed to the browser.

An independent source review of the test baseline and correction found no
issues. That reviewer did not run tests or the HTTP probe. The full Playwright
rerun remains a separate qualification gate; **no browser after-pass is claimed**.
