# QR camera lifecycle repair — 2026-11-19

This change repairs cancellation in the current QR implementation at
`9db8a323d7dcf430d1a58a4a4d10ca3628aeb401`. The module remains disabled and on
integration **HOLD**. It does not replace the current QR service with the
divergent recovered implementation.

## Reproduced boundary and repair

The unchanged `web/static/scan.js` could attach a camera and begin detection
after a pending permission request completed following Stop, page exit or
document hiding. A decode that completed after cancellation could still fill
and submit the manual preview form. Deferred fallback imports, scanner starts
and image decoding had the same stale-completion problem. These observations
concern media lifetime and preview navigation, not a demonstrated proposal or
payment submission.

Each camera or file operation now has an attempt number. Cancellation advances
that number. Every asynchronous completion checks that it still belongs to the
current visible attempt before changing the UI, scheduling more work or
navigating to the existing preview route. Stale failures cannot stop a newer
attempt. Page exit, document hiding and the host's `circula:logout` event all
cancel pending work as well as active capture.

Independent review of the first repair reproduced two additional cancellation
boundaries. Selecting an oversized replacement image could leave an earlier
image decoder active, allowing its result to replace the size error and submit
the preview. Submitting a manual code while a camera or image decode was pending
could let the late result replace that code and submit a second preview before
page exit. A real new file selection now invalidates prior work before its size
is checked; a file event with no new selection preserves the pending attempt.
The form's submit event now cancels pending capture and decoding without
preventing the normal submission. Scanner-triggered previews retain their
existing behavior when cancellation runs again during `requestSubmit`.

The pinned Nimiq scanner delays media cleanup and retains its video reference.
Tests using that exact bundled implementation reproduced two related issues:
clearing the element too early left its acquired tracks running, and an old
scanner's delayed cleanup could stop a newer scanner's stream. The wrapper now
stops attached tracks immediately and gives each new camera attempt its own
video element. Cloning preserves the element's attributes, ID and fallback
children; the previous instance can only clean up its own detached element.

The scanner/worker distributions, encoder, QR Go source and dependencies remain
byte-for-byte unchanged. The current native-detector preference, Nimiq fallback,
file-size and decoded-text bounds, manual input, and successful automatic
navigation to the server-authorized preview remain in place. There is no new
network request destination, frame persistence or payment authority.

## Recovered source provenance

The verified recovery archive contains QR head
`08a09b2bb196403b26e87de32e3b0ac167cfa75e`. Its `web/camera.js` and
`web/camera_test.mjs` supplied the cancellation scenarios and attempt-number
pattern. Its alternate renderer, scanner distribution, service, store, APIs
and policy constants were not copied. This repair is adapted to the current
module's actual scanner and navigation contract.

## Local qualification

The initial 23 tests ran before and after the first production change under
Node `v24.22.3` on Linux. The unchanged source failed 21 cases; the ordinary
native preview and unsupported-camera controls passed. The initial repair at
`2ee089e36c9f2036c797d2be5bcce5a0650f78cb` passed all 23 cases with no skips.
The test file was unchanged between that red and green run.

After independent review, four cases were added for oversized replacement,
no-new-selection preservation, and manual submission during camera or image
decoding. The form fixture now dispatches the submit event and honors
`preventDefault`, so the existing successful preview controls also exercise
the new cancellation listener. Against the unchanged initial repair, the
expanded suite passed 24 cases and failed the three stale-result regressions.
After the follow-up production repair, the exact same 27-test file passes all
27 cases with no skips in 0.371 seconds of wall time on Linux with Node
`v24.22.3`. Both script syntax checks pass. The environment outage interrupted
the work; the actual committed files were verified intact after reattachment,
and these follow-up results come from those files rather than reconstructed
source.

```sh
node --check web/static/scan.js
node --check web/scan_lifecycle_test.mjs
node --experimental-vm-modules --test --test-timeout=15000 web/scan_lifecycle_test.mjs
```

The tests evaluate the actual application script without rewriting it. They
control synthetic DOM, media, import and decoder promises. Two tests additionally
evaluate the exact bundled Nimiq class, including its delayed stop/destroy
behavior, with synthetic media and worker boundaries. The cases cover late
permission, play, import, start, native/fallback detection, image decoding,
stale errors, overlapping attempts, Stop/leave/hide/logout, immediate media
release, replacement-file validation, manual submission, the existing preview
path and input bounds.

The dedicated `future-qr-camera.yml` workflow runs these checks on Linux, macOS
and Windows, records the hosted Node version, uses the repository's immutable
checkout pin, has a read-only token and explicit deadlines, and checks the
disabled/HOLD manifest. It installs no dependencies. Provider results for the
published commit remain a separate gate; local Linux results do not establish
native macOS or Windows results.

These are deterministic lifecycle tests, not real camera, CSP, mobile-browser,
accessibility, full Go/SQLite, host-session or ledger integration qualification.
Those existing module gates remain. The source target's old README, feature
scope and validation narrative describe an earlier prototype and are not a
complete inventory of its current service, storage, web and CLI implementation;
this narrowly scoped repair does not resolve that broader documentation debt.

## Accepted-main baseline for configured Core checks

The isolated target inherited two defects already repaired in accepted main
`ead63b5ce24b2bcf8352d75fd0d1ff7be711f625`. This candidate restores exactly
those two completed-main files so the configured Core jobs can run:

| File | Accepted-main Git blob | Original accepted repair |
| --- | --- | --- |
| `cmd/circula/auditanchor.go` | `e2a0d8ee5d2d340b5222fcc2b86ad50104255587` | `dc97f7d5b0ae64306a8e2b1e57ec83f233d3d09d` |
| `internal/service/departure_pagination_regression_test.go` | `3b117ecfe2f01e7e2363b9d201503cde6e0f5595` | `7457b618057a8b1757ec643806e9afbfc5ba907e` |

The first adds the missing `os` import. The second increases only the test's
fixture cap to permit its unchanged 501-proposal assertions; the production cap
is unchanged. No unfinished external-worker source is copied. Shared workflows,
root dependencies, migrations and other Core paths remain unchanged.

The final 27-case camera qualification was run at
`7a33e6e1ce18c04f0b0713a4895934f547e45bb9`. This baseline follow-up leaves
the exact scanner script, lifecycle test and dedicated workflow blobs unchanged.
It is not a new local Core test result. The resulting publication still requires
its configured provider jobs and normal review requirements.
