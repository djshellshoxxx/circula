# Browser colour-parity clock boundary

## Reproduction

PR #71, workflow run [44247112400](https://github.com/djshellshoxxx/circula/actions/runs/44247112400), browser job 134608611808, completed 583 of 584 browser tests successfully. The remaining case was `UX-005: trade detail, pending is understandable without colour`. The two independently requested pages had otherwise identical visible text, but their generation times were 06:33 and 06:34 UTC. An exact comparison across those server responses therefore failed at the minute boundary.

## Correction

`e2e/tests/greyscale.spec.js` now captures each fresh page's visible text immediately before applying its colour mode and compares it with that same page after the mode is applied. Every visible word, including the generation timestamp, remains part of that exact comparison. A separate comparison against the first page substitutes only the visible text of `#figure-freshness time`, requiring that node and a nonempty value. All other words, including the freshness statement, must match across requests. This preserves detection of redirected, blank or incomplete fresh pages; independent review caught that gap in the initial same-page-only repair before merge.

Each mode still uses a fresh page. The initial expected balance states and populated history/trade rows remain checked, as do all per-mode amount signs, spoken directions, status words, visibility and contrast requirements. No production code, clocks, timeout or retry count changes.

## Local verification and limits

A deterministic assertion probe executes the actual pending-trade test body with controlled page snapshots and the pinned Playwright assertion library. The original body passes with equal generation minutes and fails when requests cross a minute boundary; the corrected body passes all three modes across that boundary. Mutating the amount or timestamp during the colour change still fails, and hiding a status through insufficient contrast still fails. Independent probes also check non-target later pages, omitted nonfinancial words and a missing generation-time node. These check assertion wiring and time-text normalization; they do not qualify DOM rendering or browser emulation.

JavaScript syntax and whitespace checks pass. Local Chromium cannot launch in this workspace because its process-singleton socket creation returns `Operation not permitted`; no local browser pass is claimed. The pull request's real browser and accessibility workflow must pass before merge. The consolidation evidence package retains the original provider failure, probe source/results and local launch limitation.
