# Notification transport repair and qualification — 2026-11-19

This isolated change covers Web Push and SMTP transport behavior. The notifications module remains disabled and on integration **HOLD**. No dispatcher, live recipient, production credential, Core adapter or release is enabled.

## Runtime changes

The Web Push sender now returns a nil `error` for successful HTTP responses, avoiding a typed-nil `*SendError` inside a non-nil `error` interface. It also handles Retry-After values without integer or duration overflow and retains the existing bounded retry policy. The pre-existing nil-response guard remains unchanged. The SMTP sender honors implicit TLS when a custom dialer is supplied. The regression tests use local HTTP/TLS and SMTP servers, decode the actual Web Push payload, and exercise the configured TLS connection.

The reviewed four-file source is published in [PR #103](https://github.com/djshellshoxxx/circula/pull/103) at `13669485b641475fce526633f47055ef31997098`:

| Path | Git blob |
| --- | --- |
| `pushsend/pushsend.go` | `cf4e6d876137f5756d0bba6a57536596f5541025` |
| `pushsend/pushsend_test.go` | `e4e82c4be240f8c5314e855ae710bf395f9f8c85` |
| `smtpsend/smtpsend.go` | `e4a473915835db4d219a2118a6fa448f91921537` |
| `smtpsend/smtpsend_test.go` | `356b32bce1661324e7f2cb721d05b18589fa750a` |

This documentation and native-workflow follow-up leaves those blobs and the dependency graph unchanged. It is based on the existing isolated target `797dddf21adddfb34a49e7bda1871a202968dcda`. The separately owned Core/workflow refresh in [PR #100](https://github.com/djshellshoxxx/circula/pull/100) is not incorporated here.

## Observed qualification

Local targeted race tests, vet and 10,000 fuzz executions passed. The complete module then passed the real SQLite-tagged race and vet suites. The provider independently confirmed the same published source:

- [Module run 44511002005](https://github.com/djshellshoxxx/circula/actions/runs/44511002005) passed HOLD and isolation checks, module checksums, formatting, vet, SQLite race tests and vulnerability checking. Reported coverage was 71.0% for the library, 85.3% for Web Push and 84.2% for SMTP.
- [Core run 44511002032](https://github.com/djshellshoxxx/circula/actions/runs/44511002032) passed seven executed jobs: Linux race tests; lint, vet and vulnerability checks; release-target builds; reproducibility and local smoke; fuzz smoke; contract/load smoke; and Playwright, axe and keyboard tests.
- The optional native Core matrix was skipped in that run. It is not a Windows or macOS pass.
- The module vulnerability command reported no called or imported vulnerability and one advisory in a required module that was not called or imported. Its default output did not identify that advisory, so this record does not assign an identifier or severity.

The new `notification-transport-native.yml` workflow runs the actual transport packages on Linux, macOS and Windows. Linux and macOS retain race instrumentation; Windows runs ordinary tests with cgo disabled. Every job records its Git source and toolchain, checks the disabled module and dependency boundary, verifies dependencies and runs vet. The existing full SQLite workflow continues to qualify the module on Linux.

The native workflow and this note were prepared while the local executor was unavailable. They require independent source review and successful provider execution at the new PR head. A passing older run is retained evidence, not a substitute for the new head's jobs. The existing `ci:cross-platform` opt-in label requests the shared native Core matrix when the next PR synchronization occurs; no shared workflow is edited.

## Remaining limits

Loopback protocol tests establish adapter behavior, not acceptance by a live mail or Web Push provider, delivery policy, operational key custody or production scheduling. Native transport jobs do not qualify the separate SQLite persistence layer on Windows or macOS. Normal current-head checks and review requirements still apply before merging into the isolated module target. Core integration remains on HOLD.

## Consolidation follow-up — 2026-11-20

The accepted notification target advanced through PR #100 to
`9d861d51b6ca0f3393795a126928fdf4317ca4eb`. Its transport runtime,
tests and module dependencies already include this repair. The remaining
unique work from PR #103 at `f831c45de9137b20abe3d31590433bda6348112b`
is the native transport workflow and this qualification note.

A clean Git merge preserves both source histories and produces exactly those
two additional paths relative to the accepted target. All production module
blobs and dependency files remain identical to the target, and the manifest
continues to declare `enabled: false` and `integration: HOLD`. The source
workflow retains three native platforms, race instrumentation on Linux/macOS,
real loopback transport tests, full action pins, read-only permissions and
module-isolation checks.

The previous PR #103 source passed its nine Core, one SQLite-module and three
native transport jobs. Those results do not qualify this new composition.
The integrated source must pass its own applicable Core and module workflows
before an expected-head merge into the existing notification future branch.
No module activation, deployment or release is part of this change.
