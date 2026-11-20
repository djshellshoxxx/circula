# Independent review: blocked-required-CI checkpoint

Read-only source/evidence review on 2026-11-18. No finalizer execution, source
edits, Go processes, runtime probes or remote mutations.

Initial reviewed finalizer SHA-256:
`8bcb54de1d98ab0a2c63d065b37ef71ac08339a9d3ffb97b99bc1a66113d6de4`.
Follow-up reviewed finalizer SHA-256:
`1993968ca72a2e9b6ffa020ae0c0ca200852036598cb0bd4970ba4b1b391ce15`.
Final source follow-up reviewed SHA-256:
`835e4eca566cc6f707fcd4e85c5354d76dbb3b1fe25bb9067827739091ebcb04`.
Reviewed refresh-script SHA-256:
`2517e36b9f65d0ee85e6a618228cddc68ef0ad04f3c0d10da342707f8f494f43`.
Publication worktree: `final-evidence-delivery`, branch
`integration/consolidation-execution-record-20261118`, HEAD
`b4398074ae4ca72559235ca50242cfcb7a934eec`.

## Follow-up result

The parent corrected the receipt/log binding and exact package-set guards in
the follow-up revisions above, added the blocked-mode security-paragraph
rewrite, and corrected every noted "concurrent drafts" phrase to "concurrent
PRs". The tested PR constituent guard remains intact. All required source and
wording corrections from this review are resolved.

The original aggregate producer did **not** write a log hash. Its completed
JSON must remain unmodified. The parent will create a separate paired receipt
after the run, preserving every original field and adding `log_binding` and
`producer_receipt_binding`, including source hashes/provenance and the pairing
time. This is after-run evidence packaging, not a producer-generated or
contemporaneous log-hash assertion. Both original and paired receipts and the
log must be retained with manifest hashes. The finalizer checks the retained
original receipt's hash, equality of all original producer fields in the paired
receipt, and the paired log hash/source filename, preserving any normalization
distinction.

The original producer JSON now records a passed run at head `17152dad`, tree
`1e47c9c30165ca27866887533586c90323e1630e`, exit zero, `1063.886s`, finished
`2026-11-18T10:07:00.462200+00:00`, with identical pre/post head and tree and a
clean post-status. It contains neither binding field. This source review does
not assert that the forthcoming paired receipt, final provider/API snapshot,
generated reports, or remote publication have been completed or reviewed.

## Findings raised in the initial review

1. `finalize_consolidation_evidence.py:167–181`: the aggregate metadata and log
   are independently hash-checked against the manifest, but nothing binds them
   to the same execution. The retained older `8865b3c` full-pass log has the
   same fourteen success lines and two helper lines and would satisfy the
   current parser alongside a different candidate's passed metadata. Record
   the actual producer's log SHA-256 in the completed aggregate receipt and
   validate it against the selected retained log's original/source hash (or
   retained hash when unchanged). Do not infer pairing solely from filenames.
2. `finalize_consolidation_evidence.py:178–180`: compare the successful and
   helper package-name sets with the fourteen/two expected names, retaining
   the no-duplicate checks, rather than validating only cardinality. Counts
   alone do not prove which packages the recorded complete-suite claim covers.
3. `finalize_consolidation_evidence.py:78–80,365`: use "reviewed concurrent PRs"
   rather than calling every open concurrent PR a draft. The inventory includes
   ready functional candidates as well as policy-held drafts.
4. `SECURITY-DISPOSITIONS.md:7`: blocked mode rewrites the scope to distinguish
   main from the candidate, but leaves "aggregate qualification for the exact
   integrated tree" in the following paragraph. State that main's provider
   qualification and the unmerged candidate's local aggregate are recorded
   separately. Do not imply the `17152dad` local receipt is a `b4398074` receipt.

The expected test-bearing suffixes are `cmd/circula`, `internal/api`,
`internal/contract`, `internal/domain`, `internal/histbal`, `internal/httpx`,
`internal/logx`, `internal/repository`, `internal/service`, `internal/store`,
`internal/web`, `internal/web/messages`, `scripts/sbom`, and
`scripts/traceability`; the two helpers are `internal/storetest` and
`internal/testutil`, all under `github.com/djshellshoxxx/circula/`.

## Guard corrected during this review

The initial version (`9dbe065a...`) did not connect queued PR heads to the
qualified aggregate. The parent added lines 183–199: each published head must
be an ancestor of the aggregate, or an explicitly mapped local authoring head
must have the identical tree and ordered parents and be an aggregate ancestor.
The new guard closes that gap. Read-only checks confirm that the PR #82 local
head `d3fcc683` and published head `e127acc9` have identical tree
`84ea230a25cf30a56f4f43b1a401abea6dc1b7fb`, and the local head plus PR #84's
`eb4bb08b` and PR #87's `679a3016` are ancestors of `17152dad`.

## Verified safeguards and truthful scope

- Every merged PR still requires retained exact-PR/head/run completed-success
  proof, all seven or nine named jobs completed successfully with unique names,
  an exact API merge receipt, main as its base, and merge ancestry in main.
  Blocked mode does not bypass any of those gates.
- Main and candidate heads/trees are separate in the generated checkpoint,
  qualification paragraph and security data. Queued candidates cannot also
  be merged receipts. Additional findings retain unmerged or partial status.
- Queue evidence is retained and hash-checked, must be no more than fifteen
  minutes old, matches reviewed open PR heads/run IDs, and must show the seven
  required jobs queued without assigned runners or executed steps. The text
  states the cause is not established.
- A candidate aggregate must be passed with exit zero, exact pre/post head and
  tree, a clean post-status, and the precise full-suite race command. A running
  or incomplete receipt is rejected. The current running task and unfinished
  final endpoint/state inputs are intentional pending gates, not defects.
- The fixed main endpoint, complete API/ref inventory, preserved historical
  branches, reviewed forward-only branch advances, allowed publication paths
  and comment-only workflow diff are checked. Branch analysis uses captured
  immutable SHAs, not moving refs.
- All 228 currently retained manifest files have matching hashes and sizes,
  unique paths and no path escaping the evidence root. Refresh normalization
  now operates on bytes and accurately limits itself to the documented ASCII
  trailing-space/tab and final-LF changes.
- The current original-audit ledger remains 48 addressed, five policy/design
  residuals and two partial findings until PR #84 actually merges. AUD4-F03 is
  a separate partial residual, and A3-F02 / DEC-014 and PR #83/#85/#86 policy
  holds remain unresolved. No automatic original-finding promotion was found.
- The older two-package race narrative is explicitly about the original
  `c07838e` checkpoint, and the provider-lint replacement sentence is about the
  already-qualified backup follow-up. These do not falsely claim a new
  candidate or queued provider pass in their current surrounding context.

The current README's new-MFA and latest review-status prose still needs the
parent's planned final reconciliation; the metadata already distinguishes the
new password-proof candidate and preserves the older qualified checkpoint.
The finalizer's source safeguards are approved subject to the intentionally
outstanding paired receipt, fresh provider/API snapshot, and final
generated-document review. No further source correction remains from this
review.

## Separate PR #88 correction delivered

Clean local commit `6a9e9775fb46ca44b4500c28e1b4d1565b1e78f6`, tree
`ac4fd86166e89ec55b8526cabc01eeab481a0981`, directly follows reviewed PR #88
head `f4cbaddd4d40f690529ab33c88f302be9d0a1752`. It changes exactly one line in
`docs/wiki/Developer-Guide.md`: identifies main at `b4398074` and pins the CI
workflow link to that full revision. `git diff --check` passed; no remote write.
