# Session-warning expiry presentation — November 27, 2026 Pacific

## Bounded defect

Base main `3faeed76c41467a75b09bc75e7185a98a92f8219` formats page timestamps through
the member's saved display zone and existing message catalogue, but app.js rebuilt the
session-warning clock with browser-local getHours/getMinutes. A browser in another zone
therefore displayed a different clock from the account's selected zone.

This slice passes existing catalogue DateTime output to the enhancement: data-expires-text
on the shared shell and additive expires_text on the authenticated web /session/extend response.
The extension explicitly loads the saved zone because its auth wrapper does not render a page.
The numeric expires and extendable fields, deadlines, timers, authentication, CSRF and form data
remain unchanged. The public JSON API is untouched.

Where an existing viewer context is available, the label uses its selected zone. Existing
nil-viewer renders retain **explicitly labelled UTC**. This is a compatibility boundary, not a
claim that every authenticated page already supplies viewer context. Broader nil-context
account-page wiring remains an independent UX-024 assessment. Old cached HTML/servers without
the additive presentation value retain the legacy clock fallback. No new locale is selected.

## Regression evidence

Preserved regression-only commit: `f2095da2dbeb0ec509bb0fa378f45b9c4f45dd81`.
Native Node execution of the new script tests against original app.js: **16 pass, 2 fail**;
the failures were exactly initial and extended server-text expectations. These are real script
executions in a mock DOM, not browser results. No Go regression run is claimed for that commit.

With the corrected script, all 18 message tests passed. The same 18 passed separately under
TZ=UTC, TZ=Asia/Kolkata and TZ=America/Vancouver. The workflow's combined helper command
`node --test scripts/test_app_messages.cjs scripts/test_e2e_kbd.cjs` passed **26 tests**,
zero failures/skips. Literal percent signs, angle brackets and dollar replacement sequences
remain text, including after extension. Legacy fixtures, timers, CSRF request, failure/expiry
messages and unsent-form retention pass unchanged.

Added Go tests (source-reviewed, **not executed locally**):
- TestSessionWarningTemplateUsesDisplayZone: UTC/default, half-hour zone, date rollover,
  Berlin winter and summer, retaining epoch and extendability attributes.
- TestSessionWarningHTTPUsesSavedDisplayZone: real service preference/login, HTTP middleware,
  account-page and extension label, numeric expiry, CSRF refusal and nil-viewer UTC control.

The source and test investigators worked independently. A product review caught the extension's
missing display-zone load before publication; it is corrected. No remaining bounded product
source blockers were found. That review independently ran the 18 Node message tests. Go test
source checks verified fixture password and CSRF API, but cannot substitute for compilation.

## Gates and continuation

The local executable named go is GNU Go, not the required compiler. No Go build, gofmt, vet,
race, full HTTP or browser suite is claimed locally. The draft PR's exact-head native CI must
qualify these changes; fix actual failures without weakening tests or changing unrelated workflows.
Separate owner merge authorization and actual-main qualification remain necessary. This is not
whole UX-047/UX-024 completion, manual screen-reader qualification or independent external security acceptance.

HANDOFF and TODO identify this one active slice. No #142 inbox code, policy-held PR, future module,
locale policy, ledger/balance, release/deployment/licence or history change is included.
#142 remains source-qualified draft pending its separate owner gate. Preserve its ownership.

Fresh continuation: read live refs/current PR head and checks, finish this same session-warning
slice before selecting another. Inspect any actual Go/CI failure, repair minimally and re-review.
Do not redo #130–#139 or #142. Keep the nil-viewer UTC boundary and manual/external limits explicit.
