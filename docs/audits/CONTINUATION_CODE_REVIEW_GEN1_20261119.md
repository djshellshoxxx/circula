# Continuation code review, generation 1

## Reviewed composition

The review pins combined Core commit `82a7e744d3730f1c426e2977795bd87133a002b4`.
Its inputs are accepted main `aa01563fcc891da60fefebb8335eca27855678e8` and the
UI continuation from PR #112, `7462138fb4f7110ea4f705c6f09742a2b5a253e4`.
This review does not establish a full application-suite or release pass.

The two inputs diverged at `0a73ee42b96255601cf2227b1ddf6599ec07a0ef`.
Accepted main changed 11 paths from that point; the UI input changed 389.
Seven paths overlapped. Reviewing the current composition confirms:

- The CI evidence checksum verification step remains, alongside the new
  scanner controls and keyboard-helper checks.
- Main's native-checkbox selection, idempotent checking, and failed-check
  rejection remain. Its session POST-response assertion and diagnostics remain.
- Main's WAL read-snapshot witness remains, with its original failure assertions.
- Both the earlier migration-statement fault and the new head-migration record
  fault are present. Neither rollback test replaces the other.
- The lost-COMMIT-acknowledgment driver counters and tests remain. The deferred
  metadata fault is still delivered by `Rows.Next`, not as an immediate query error.
- The accepted post-snapshot evidence files and checksum manifest remain.

Catalogue review covered message helpers, member/invitation action state,
trade and dispute presentation, report target handling, historical balances,
and administrator reversal/dual-control presentation. Those reviewed changes
preserve integer amounts, contextual HTML escaping, and stored enum values for
authorization and action eligibility. This is a bounded source review, not a
claim that every Core path has been audited.

## Repairs in this continuation

The retained main keyboard helper uses native `matches` and Playwright's
`locator(...).isChecked()`. The older UI branch's unit adapter implemented neither.
The unmodified merged harness therefore failed four of its six tests with
`document.activeElement?.matches is not a function`.

The adapter now supports those exact browser operations without duplicating the
helper's focus-name callback. Two regressions also verify that an already-checked
confirmation remains checked and that an ineffective Space press stops submission.
The existing tests retain same-name error-link rejection, bounded searching,
text entry, and intentional link activation.

The approval-mode browser setup searched visible link text for a member name.
The updated member template intentionally places the complete action name in
`aria-label`, while the visible text is only the action. The setup now uses the
exact `link` role and accessible name. It has no first-match fallback, so
ambiguous names still fail instead of selecting a different member.

## Execution evidence

| Command or check | Observed result |
| --- | --- |
| Original `node --test scripts/test_e2e_kbd.cjs` at the combined input | FAIL: 2 pass, 4 fail; adapter lacks native `matches` |
| `node --test scripts/test_app_messages.cjs scripts/test_e2e_kbd.cjs` after repair | PASS: 15 tests, zero failures/skips |
| New eight-test keyboard harness loaded against unchanged PR #112 `e2e/kbd.js` in an in-memory Node module | Expected FAIL: 6 pass, 2 fail; restored checkbox is cleared and ineffective checking is accepted |
| `node --check e2e/global-setup.js` | PASS |
| `git diff --check` for these authored changes | PASS |
| Chromium locator/native-keyboard smoke attempt | BLOCKED before any assertion: system Chromium aborted with `socket() failed: Operation not permitted (1)` |

The mutation control replaces only the helper module import in memory; it does
not modify either Git input or the regression test bytes. The Node adapter is
not a real-browser substitute. The failed smoke used runtime Playwright 1.62.1,
system Chromium and the project's normal launch arguments; it does not qualify
the project's pinned 1.63.0 application suite. A fresh worker must still execute
the real approval setup and keyboard journeys on the final combined source.

## PR #104 uniqueness check

Against PR #104 head `bcdc569cbc483761bf2d0e2d7594636ac1713d7d`, all four of
its remaining non-documentation paths are already byte-identical here:

- `.github/workflows/export-reader.yml`
- `cmd/circula/verify_export.go`
- `cmd/circula/verify_export_ambiguity_test.go`
- `cmd/circula/verify_export_boundary_test.go`

Its Python reader and test file are preserved with strictly additive continuation
content: nine lines requiring decoder EOF and 89 lines of associated regression
coverage. There is no unmatched PR #104 runtime repair to import. Its additional
audit records remain provenance work for the coordinator to classify separately.

## Fresh-worker handoff

Review the authored helper-adapter and exact-role changes independently, then run
the final combined application browser suite. Preserve the accepted main helper
when resolving subsequent branch changes. Qualify both migration-fault tests and
the COMMIT-acknowledgment controls using the final Go dependency graph. No remote
branch or PR was modified by this worker, and no held policy or future module was
activated.
