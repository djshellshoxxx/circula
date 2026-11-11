# Root build-artifact hygiene — 2026-10-30

Owner: the coordinator on `codex/circula-consolidate-20261030`, draft PR #17.
Source: consolidated checkpoint `d9c8747280e42f1efeedfea48d0a9c9735bc62dd`.
This is a source-control hygiene slice, not an A1/A2/A3 runtime remediation.

## Reservation and scope

The live published branch/PR/issue inventory showed no competing hygiene work or
reservation. Before editing, the coordinator reserved only `.gitignore`,
`scripts/check-hygiene.sh`, deletion of tracked root `circula`, and the handoff/
receipt paths. Original executor state is unknown; external runtime audit/WIP
ownership remains protected. No finding ownership was inferred from inactivity.

The original root executable is blob
`b9b7b95b2cfdd18e423500705fa83e1e0a0ba810`, 21,703,878 bytes. It remains in Git
history; `git rm --cached -- circula` removes it only from the new source tree,
leaving the local copy ignored. No history rewrite or binary execution occurred.

`.gitignore` adds only `/circula` and `/circula.exe`, the Unix and Windows default
root output names for `go build ./cmd/circula`. Existing `/bin/` exclusion remains.
The hygiene script checks only these two exact tracked root paths and fails if
either reappears. It inspects the Git index, so ordinary ignored local builds do
not fail. Existing QA-043, QA-037 and GOV-026 checks are unchanged. No broad
binary/size ban or new requirement identifier is introduced.

## Regression evidence

After adding the guard but before removing the tracked source artifact, running
`scripts/check-hygiene.sh` at the source above produced:

```text
hygiene: generated root executables must not be tracked:
circula
exit_status=1
```

After the removal and ignore changes:

```text
hygiene checks: ok
action pins and permissions: ok
PASS: ignored local build is allowed
PASS: force-tracked circula is rejected (exit 1)
PASS: force-tracked circula.exe is rejected (exit 1)
PASS: original executable blob remains available in history
```

The two reintroduction checks used separate temporary `GIT_INDEX_FILE` state,
seeded from the actual post-removal index; the original blob was added at each
root output name with `git update-index --cacheinfo`. The guard exited 1 for each.
The real index was not changed by these negative checks. The existing ignored
local executable was present during the passing case.

`bash -n scripts/check-hygiene.sh`, `git diff --check`, both repository static
scripts and `git check-ignore --no-index circula circula.exe` passed. The original
blob's recorded size was verified with `git cat-file -s`.

No Go source, dependency, test assertion, monetary threshold, append-only ledger
rule, grant lineage or 24-hour approval behavior changed. Full local Go 1.26.10,
make check, E2E/contract/ledger and pinned-secret-scanner checks remain unavailable.
No denied download was retried. Aggregate CI must run against the actual published
hygiene head; the source consolidation's queued jobs are not a hygiene pass.

## Review and checkpoint

Independent delta review and remote publication are complete below. Aggregate
acceptance must use the latest exact-head CI in draft PR #17; keep runtime
candidate ownership and owner/release gates separate from this cleanup.

Independent read-only reviewer `spec_disposition` found no blockers and reproduced
all three guard cases. The real index remained byte-identical; the existing
QA-043/QA-037/GOV-026 script suffix is byte-identical. The reviewer also verified
raw local merge-object hashes, recorded local/remote tree equivalences and input
ancestry. Shell syntax, action pins and staged whitespace passed independently.
This assistant delta review is not formal GitHub approval or professional review.

Published fix: `66501ec55eee391b2557118636fdf6af572f7b93`, parent
`d9c8747280e42f1efeedfea48d0a9c9735bc62dd`, tree
`b81f29d928a52231be89fb017ce32f197d45136d`. The tree exactly matches local reviewed
commit `c90a76d2aeb2d6afe9da16202a81a73e98cbe742`. Following README-only correction
`b0e9d5dc03016620b541cc073ec9ac11340de0a8` has tree
`9cbfaefb45a54c69c6ae2af78740a3a429fb7777`, equal to local
`4e2a1c970b8f52931ff3333ac5151a222d9122b7`.

The connector advanced only the owned branch with `force=false` and expected head
`d9c8747280e42f1efeedfea48d0a9c9735bc62dd`. Fetch/readback confirmed the new remote
SHA, full tree equality, input ancestry, a clean tracked worktree, and passing
hygiene/action-pin scripts. Local commit lineage is retained under
`preservation/circula-local-hygiene-20261030`.

The older d9c8747 push/PR CI runs were automatically cancelled by the unchanged
per-ref concurrency policy when this source advanced; neither is a pass. At the
next observation, b0e9d5d push run 41041893777 and PR run 41041899651 were queued.
A subsequent documentation receipt may supersede these too. Current-head run
results are authoritative in PR #17; no cancelled, queued, skipped or older run
constitutes acceptance for a later source. No local full application pass is claimed.
