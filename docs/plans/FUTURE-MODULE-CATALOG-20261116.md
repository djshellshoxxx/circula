# Future feature development catalog

**HOLD: no feature is approved for integration, activation or release.**

Owner direction, 2026-11-16: develop all 15 future roadmap features independently; complete both each feature and Circula core first, then integrate one explicitly selected feature at a time. Do not merge these branches as a batch. This instruction supersedes earlier general consolidation authorization for these features.

## Current milestone

All 15 have independently tested initial core logic, not complete product features. Each feature has its own branch, standalone nested Go module, tests, historical spec snapshots, validation record, feature manifest, and README listing completed scope and remaining work. There are no cross-feature imports or required future-feature dependencies. No application registration, routes, migrations, root dependencies, jobs or release workflows were changed. Nothing is enabled.

Every branch starts directly from core baseline `2eacf7718f5bb3da3be5a288024af879076a9597`, rather than stacking atop another future feature. Each commit changes only its own `future/<feature>/` directory and can be reviewed/cherry-picked independently. The catalog branch contains documentation only.

| Feature | Independent branch and module | Initial commit |
|---|---|---|
| FUX-101 | [Installable PWA](https://github.com/djshellshoxxx/Circula/tree/future/fux-101-pwa-20261116/future/pwa) | `43966749941e` |
| FUX-102 | [QR trading](https://github.com/djshellshoxxx/Circula/tree/future/fux-102-qr-20261116/future/qr) | `510c26e2afa4` |
| FUX-103 | [Marketplace discovery](https://github.com/djshellshoxxx/Circula/tree/future/fux-103-discovery-20261116/future/discovery) | `2faa1940da9c` |
| FUX-104 | [Email and push notifications](https://github.com/djshellshoxxx/Circula/tree/future/fux-104-notifications-20261116/future/notifications) | `0e5bbdcc69c3` |
| FUX-105 | [Private member messaging](https://github.com/djshellshoxxx/Circula/tree/future/fux-105-messaging-20261116/future/messaging) | `a78af66c2135` |
| FUX-106 | [Guided onboarding and practice](https://github.com/djshellshoxxx/Circula/tree/future/fux-106-onboarding-20261116/future/onboarding) | `e7e3af0ada2e` |
| FNET-201 | [Advisory trade matching](https://github.com/djshellshoxxx/Circula/tree/future/fnet-201-matching-20261116/future/matching) | `d285fa917803` |
| FNET-202 | [Saved trading partners](https://github.com/djshellshoxxx/Circula/tree/future/fnet-202-favorites-20261116/future/favorites) | `3e30c15ecc92` |
| FNET-203 | [Optional digital trade receipts](https://github.com/djshellshoxxx/Circula/tree/future/fnet-203-receipts-20261116/future/receipts) | `ac4927581e3d` |
| FNET-204 | [Community activity feed](https://github.com/djshellshoxxx/Circula/tree/future/fnet-204-feed-20261116/future/feed) | `d28879a8bf8c` |
| FNET-205 | [Credit-circulation analytics](https://github.com/djshellshoxxx/Circula/tree/future/fnet-205-analytics-20261116/future/analytics) | `9593ee38698d` |
| FNET-206 | [Coarse geographic areas](https://github.com/djshellshoxxx/Circula/tree/future/fnet-206-areas-20261116/future/areas) | `94e521725bec` |
| FNET-207 | [Recurring trade templates](https://github.com/djshellshoxxx/Circula/tree/future/fnet-207-recurring-20261116/future/recurring) | `6729dadf934d` |
| FNET-208 | [Community governance](https://github.com/djshellshoxxx/Circula/tree/future/fnet-208-governance-20261116/future/governance) | `417d169762b6` |
| FNET-209 | [Opt-in discovery federation](https://github.com/djshellshoxxx/Circula/tree/future/fnet-209-federation-20261116/future/federation) | `e41375537096` |

## Tests actually run

All 15 modules: `go test -race -cover ./...` and `go vet ./...`, Go 1.25.4 linux/amd64, pass. QR additionally passed a short fuzz run (~37,000 inputs); review added a regression for empty URL fragments. Per-feature coverage/results are in VALIDATION.md. All packages use only the Go standard library.

Root Circula requires Go 1.26.11 and was not built/tested in this work. No root application files changed. Its completion/release qualification remains independent and is not advanced by this milestone. Nested modules are excluded from normal root build/test traversal; test each explicitly from its directory.

## Completion and integration gates

1. **Module core tested (current):** isolated logic and negative tests pass. Not feature completion.
2. **Feature complete (pending):** fulfill the entire adopted spec, durable adapter contracts, migrations where needed, UI/CLI/help, restart/concurrency/auth/tenant tests, accessibility and security audit. Record accepted scope deviations explicitly.
3. **Core complete (pending independent qualification):** Circula's own current acceptance/release gates pass.
4. **Single-feature integration (not authorized now):** rebase on the qualified core, implement/review the narrow service adapter, run full regression, and obtain the owner's selection of that feature. Never infer approval from a feature's tests passing.
5. **Opt-in rollout:** after single-feature approval only; preserve no-JS core behavior and documented disable/data-retention procedures.

Branch independence prevents deliberate coupling; it cannot guarantee future source compatibility. Revalidate adapter contracts against the core version chosen at integration time. Do not add global flags, pending migrations or stubs to main in anticipation.

## Boundaries that remain explicit

- QR claiming and proposal creation need one durable atomic transaction; a validator is not a claim store.
- Messaging, favorites, feeds and notification preferences must use fresh service authorization and network-scoped persistence.
- Receipt TXT/JSON is independent of certificates; certificate algorithms and key lifecycle remain undecided. No automatic receipt generation.
- Analytics currently calculates gross historical transfer volume, not net/reversal-aware flow or balance stock. k-suppression alone does not prevent differencing.
- Recurrence currently supports daily local reminders only; no auto-posting or auto-acceptance.
- Governance outcomes are advisory only, never automatic policy changes.
- Federation is experimental discovery-only validation. Replay persistence, transport, rotation and revocation ordering remain unfinished; settlement is excluded pending separate accounting approval.

## Continuation rule

Continue each feature on its own branch. Read that module's README, feature.json and validation record before changing it. Keep its remaining-work list accurate; never change HOLD merely because unit tests pass. Add no dependency on another unfinished feature: supply an adapter interface and test fixture instead. If another worker advances a branch, fetch and reconcile that exact branch; never force-push over their work. No merge, auto-merge PR, release tag or deployment is authorized for these future features.
