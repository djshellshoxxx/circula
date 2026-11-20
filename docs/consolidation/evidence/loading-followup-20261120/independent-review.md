# Independent loading correction review

Two fresh workers reviewed exact repair
`eef0c6d8c832ce994db1019eb5e7d6a41800b8ef`, tree
`029e5fdda2d03347389397576ec08701ae04cec2`, on 2026-11-20. One reviewed source,
history, templates, fixtures and listener ordering without product execution;
the other performed the focused checks recorded in `local-checks/`. Neither
edited source or made remote mutations. The coordinator separately inspected
the complete three-path delta and original provider log.

## Production behavior

The application synchronously sets catalogue-backed status text and main
`aria-busy` during the qualifying native click/submit event, before a pending
document response can delay timer-based feedback. The next-task
`defaultPrevented` check remains and clears feedback when a later listener
cancels an in-place interaction. Events already prevented remain excluded.

Existing same-origin HTTP(S), same-window, fragment/download, form/submitter
opt-out, allowed method, target override and native-validation boundaries are
retained. The enhancement neither cancels navigation nor invokes submission;
it does not move focus, disable controls or alter successful form values.
Text is assigned with `textContent`. `pageshow` clears pending/displayed
feedback. The live status region remains outside the busy main element.

## Browser observation and assertions

The deferred application script registers its document listeners before the
test installs its observation-only listener after page navigation. The latter
reads actual DOM state after the application handles the same event; it does
not inject expected state. The helper returns an object containing the console
event promise, allowing installation to finish before the test triggers the
event. Its one-shot listener is used immediately before the intended control
activation.

The tests retain actual held document requests and every behavioral assertion:
status text, role, busy state, focus, enabled control, native POST method and
confirmation fields, amount/counterparty/CSRF/idempotency values, completed
navigation or successful trade result, and reset status in the destination.
Native-validation and download tests remain. No retry or explicit timeout
configuration was changed.

The new Node regression checks both navigation and native form feedback before
advancing timer tasks. Existing cancellation, focus, successful-control,
exclusion and `pageshow` checks remain. The separate old-production control
demonstrates that this regression fails against the previous deferred script.

## Reviewed blobs and conclusion

| Path | Git blob |
|---|---|
| `internal/web/static/app.js` | `c74fa68ad3e597389e6c28b76f3a169917c466d2` |
| `scripts/test_app_messages.cjs` | `feb386621e477c7d65c8f246f1d8c805ee53472e` |
| `e2e/tests/loading.spec.js` | `13926245a25bac2ed47f8c204ab55a22a4a3a3d8` |

No blocking finding remains in the scoped correction. Current-head provider
browser and applicable CI qualification are still mandatory before integration.
This review is not a provider approval or a browser pass. Event-time DOM/ARIA
state does not establish physical rendering or assistive-technology acceptance.
