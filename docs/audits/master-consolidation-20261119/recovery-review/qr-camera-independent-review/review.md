# Independent QR camera lifecycle review

Final source reviewed: `7a33e6e1ce18c04f0b0713a4895934f547e45bb9`, full tree `e2259c2a13ad546ee0f0f817e886e0e3e61ae451`, from target `9db8a323d7dcf430d1a58a4a4d10ca3628aeb401`. Exactly four changed paths: the scanner script, lifecycle tests, dedicated native Node workflow and qualification note. Vendor, Go, dependencies, feature metadata and server APIs are unchanged.

The complete initial change and exact follow-up delta were read. Two independently reproduced stale-result boundaries are now closed: selecting an oversized replacement cancels the older decoder before validation; manual form submission cancels pending camera/image results without preventing the normal preview submission. Empty file-selection events preserve the prior pending attempt. Automatic scanner submissions remain covered by successful preview controls.

Both independent regressions were replayed against the exact final source and passed. The author additionally recorded the unchanged expanded test file failing three regressions and passing24 controls before the follow-up, then passing all27 with no skips afterward. Earlier23-case before/after evidence remains separately identified. These are synthetic lifecycle tests, including the exact bundled Nimiq implementation in two existing cases; they do not establish native camera, browser, CSP, mobile or accessibility acceptance.

No blocking source finding remains in the reviewed scope. The dedicated workflow preserves read-only permissions, pinned checkout, module HOLD/isolation checks, three operating systems and bounded Node test execution. Successful current-head provider checks and ordinary merge approval remain required.

## Accepted-main baseline follow-up

Publication candidate `3c1ce7a7dc190271a9a830abc924f4f938049801`, tree `6c5f55e7fe658d5144c3c09c010f74923397c7df`, adds exactly the two already-accepted main blobs for the missing `os` import and the 501-proposal test fixture cap, plus an explicit provenance paragraph. Both introducing commits are ancestors of accepted main `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`; the final files equal main exactly. The complete candidate now changes six paths. Camera runtime, lifecycle tests and dedicated workflow remain byte-identical to the reviewed and qualified7a33e6e source. No new Core test result is claimed. The exact three-path follow-up is source-approved, subject to current provider checks and ordinary approval.
