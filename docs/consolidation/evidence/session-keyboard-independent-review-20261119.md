# Independent review: session-confirmation keyboard targeting

## Scope and source identity

The review compares test-only baseline
`e0da8bb22b0499deac8f0bfcc170c6fee6696238` (tree
`ea1968e9a8bf465e33a7e557b6d94b5f8df8903b`) with correction
`bf4261e1f4ead7f58675f0f95d514333c746cf4b` (tree
`28b81bbfc39690ceb227247e5bd8749d30c1ba02`). The relevant exact blobs are:

| Path | Baseline blob | Corrected blob |
| --- | --- | --- |
| `e2e/kbd.js` | `cd69c353d8b122a879a4b30e84a150b3d7473962` | `66a4dc3d235a12e26e5d3088385a11fe950c82c6` |
| `scripts/test_e2e_kbd.cjs` | `7ed6beaf73835912e9d86295679dc56786f448e9` | `7ed6beaf73835912e9d86295679dc56786f448e9` |

The corrected CI workflow blob is
`c5db955cec9987dfd6cf3abc5a4df0821211b2e6` at `.github/workflows/ci.yml`.

## Findings

**No defects identified in the bounded correction.**

The captured provider page has a validation-summary link before a native
checkbox with the same name. The former helper matched the name alone before
pressing Space. That source behavior explains how the confirmation could stay
unchecked after the keyboard helper submitted the validation page.

The correction enables the new constraint only for `kbSubmit` fields whose
existing value is exactly `true`, which already denotes a checkbox toggle.
After the name match, it checks the actually focused element's native tag and
type. A same-name link therefore cannot satisfy that field. Traversal still
uses Tab, the toggle still uses Space, and submission still uses Enter. The
80-press bound and failure-before-submission behavior remain explicit.

The optional constraint defaults to false. Existing generic `tabTo` callers
can still intentionally target links, regions, radios, and selects; text
entry, button targeting, and `kbActivate` keep their existing behavior.
There are no application-template, authorization, CSRF, service, or session
policy changes in the reviewed correction.

The Node regression imports the actual helper and executes its own browser
callbacks in the adapter's VM context. It does not reimplement the helper's
name or native-control predicate. The same test blob is present before and
after the correction. Its useful controls include both colliding confirmation
labels, a missing-checkbox rejection before submission, an ordinary checkbox,
text entry, intentional link activation, and the explicit Tab-search budget.
Assertions check the focused target and resulting key sequence. CI adds the
new helper tests to the existing progressive-enhancement Node command without
changing workflow events, permissions, or external actions.

## Evidence boundaries

The reviewer read the exact source diff, regression source, provider-page
evidence, baseline and corrected execution receipts, and CI change. The author
recorded three failing and three passing helper tests before the correction,
then thirteen passing tests in the combined helper/progressive-enhancement
command. These are the author's executions, not an independent rerun.

The adapter models focus order and keyboard effects. It does not exercise
native browser rendering or keyboard behavior. The reviewer ran no tests,
builds, browser, or HTTP probe and made no source edits. This source review
does not establish a browser after-pass; that remains a separate qualification.
