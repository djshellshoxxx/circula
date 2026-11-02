# Text preference probes (UX-034)

Owned QA scope: `e2e/tests/resize.spec.js`, this record, the UX-034 as-built note
and current handoff. The input browser test source is
`5d2b2c219acbd0d6ba192c42372cc7a8a484ab7b`; its exact results are recorded in
[the viewport checks](RESPONSIVE-CHECKS.md). No runtime/CSS, fixture, old assertion,
dependency, workflow or reserved audit file is changed.

## Four browser cases

The proposal form and review step are exercised in 320×640 portrait and 640×320
landscape with two separate preferences:

- A doubled root font size. The test checks that heading, input and button text
  actually double before entering an amount and opening review.
- User spacing overrides: line height 1.5 times the font, paragraph spacing 2em,
  letter spacing 0.12em and word spacing 0.16em. Computed styles prove the override
  took effect before the form interaction.

The spacing values follow [W3C's SC 1.4.12 explanation](https://www.w3.org/WAI/WCAG22/Understanding/text-spacing.html).
The text-size probe follows the 200% target in
[SC 1.4.4](https://www.w3.org/WAI/WCAG22/Understanding/resize-text.html). These user
style overrides are browser test inputs; they do not change the site's default
styles or security settings. Existing test contexts already permit injected
styles/scripts for accessibility checks.

The review page receives the same preference after navigation. Its final Send
trade control must remain visible, reachable by scrolling and enabled, but is
never activated. No trade or ledger entry is posted and no grant/24-hour fixture
is altered. Horizontal scrolling under text enlargement is not itself declared
a failure of SC 1.4.4; the separate default-viewport reflow tests remain intact.

This is one representative form flow, not all functionality or a complete WCAG
certification. It does not simulate every browser zoom mechanism, native device
rotation, every resize increment, manual reading or screen-reader behaviour.
UX-034 remains partial; manual and independent gates remain open.

## Validation

At `2e323a18e4f3af5e080d58e941f8cc2f74f0842f`,
[push CI 41431266623](https://github.com/djshellshoxxx/circula/actions/runs/41431266623)
completed all nine jobs successfully. Browser job `125230148678` reports
131 passed, including all four new preference cases and the previous 24 viewport
cases. The filtered log confirms both preferences at both orientations. This
is exact-source CI evidence, not a local browser run or proof for later changes.

With the existing built binary and cached browser dependencies, run:

```sh
cd e2e
CIRCULA_BIN=/absolute/path/to/bin/circula ./node_modules/.bin/playwright test tests/resize.spec.js
```

Local checks passed: Node syntax, staged whitespace, hygiene/action pins,
source/fixture preservation, 29 local links across the outgoing documents,
fences/conflict markers and a limited credential/private-data scan across all
five complete outgoing files, with no findings. Local browser
dependencies remain absent; no denied downloads are retried. Browser results
must come from the existing reviewed CI route at the exact new test source.
Preserve failures and original assertions. A failing probe is a candidate for
diagnosis, not permission to alter a reserved runtime module or weaken the test.
