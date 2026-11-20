# Independent Linux qualification-budget review

Reviewer: `loading_independent_review`; review returned 2026-11-20. The reviewer
made no source edits, remote mutations or new Go test-execution claim. This
record transcribes the source identities, executed comparisons and conclusions
returned to the coordinator; it is not a GitHub account's formal approval.

## Reviewed source

- Local commit: `a4becfda0f10d5fa3790bf0ee4aeff6552dfeca1`.
- Parent: `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef`.
- Complete tree: `43e65ee91a90ff70b70cc69d0b517b8367a2585f`.
- Only changed file: `.github/workflows/ci.yml`.
- Workflow blob: `34d4f6ed758f1c685c1a38244a422c409687e70a`.
- Workflow SHA-256: `c4ebd18c99870c16d40bea63c0195b28955d93a908453273dd4f5a4b2e82e47b`.
- Complete patch SHA-256: `7539a262d36df1470f37091efa0d4d71231f3bad421c79c7914ae0198d2dcf48`.

The reviewer independently parsed YAML and reversed the intended two settings.
Every other parsed value was identical. Linux job timeout changes from 30 to 45
minutes; its Go package timeout changes from 20 to 30 minutes. The complete
`./...` selection, race flag, traceability recording, property environment,
matrices, action pins, permissions, artifacts and all other jobs remain.

## Failure and timing evidence

The reviewer read the entire 71,307-byte Linux failure log and verified its
SHA-256 `e8079e8309dd92fbf03c990c0d8902b80fb2e95fa10577f2f39e95f385cb3aba`.
The explicit failure is the 20-minute package timeout; the service package
reports `1200.102s`. QA001 had run 23 seconds and its current seeds 1–3 seconds.
Four shards had advanced to seeds 100020, 100025, 100022 and 100023. Stacks show
SQLite migration/parser activity, allocation contention and test-parent waits.
No assertion-failure line, data-race warning or fatal-deadlock diagnosis appears.
A stack snapshot cannot establish universal absence of deadlocks; this evidence
supports package deadline exhaustion without demonstrating a deadlock cause.

The reviewer decompressed and verified the original complete race log against
SHA-256 `31b2f4b9ed3b6b6750bef6052abb28261d0d1e9db33d04ed4042b9204bfeba52`.
Actual terminal events show service PASS at `1294.982s`, QA001 PASS at
`104.690s`, and all 50 configured race-mode QA001 seed subtests passing. No Go
source or root module/checksum file differs between that race source and the
reviewed repair parent.

## Conclusion and publication boundary

The reviewer found no blocking issue or comment-wording problem. Thirty minutes
leaves about 505 seconds beyond the observed successful service duration; the
45-minute job allows bounded setup and reporting headroom. No coverage reduction
or failure suppression was introduced. Fresh provider success is still required.

The coordinator subsequently published normal successor
`11739c7f8f59455b4dc480294cab4d79f7933cb9`, fetched it, and verified the identical
complete tree, expected sole parent and empty difference from the reviewed local
source. The [publication receipt](linux-budget-publication.json) records that
separate action; review alone did not qualify the new whole head.
