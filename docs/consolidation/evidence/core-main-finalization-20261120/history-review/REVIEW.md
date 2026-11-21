# Independent historical PR reconciliation review

Reviewed against actual Core main `87651d36558a8e421e41fbd76c514045d88c74ed`, tree `43e65ee91a90ff70b70cc69d0b517b8367a2585f`. Live GitHub source and branch readback completed at **2026-11-21 02:35:03 UTC** (November 20, 18:35 America/Los_Angeles). The evidence branch remained `c1a34476de017ab70acc99181ebbe16c337fc0e6`.

**Inclusion and recovery review passed for all four PRs.** Closure remains conditional on the coordinator's verification of the post-merge workflows on actual main. This review does not claim those workflows passed. Preserve all original source branches. PRs #104 and #111 require explicit supersession language; neither source commit is an ancestor of main.

## Disposition

| PR | Exact source | Independently verified result | Permitted reconciliation after main qualification |
|---|---|---|---|
| [#109](https://github.com/djshellshoxxx/circula/pull/109) | `5db3efc084e6a807bb4702c09d3235490d42b71d` | `git merge-base --is-ancestor` returned 0; source-only commit count is 0. | Close as already incorporated through #127; no additional merge needed. |
| [#112](https://github.com/djshellshoxxx/circula/pull/112) | `7462138fb4f7110ea4f705c6f09742a2b5a253e4` | `git merge-base --is-ancestor` returned 0; source-only commit count is 0. | Close as already incorporated through #127; no additional merge needed. |
| [#111](https://github.com/djshellshoxxx/circula/pull/111) | `45356ca10e1ecd6e6441e0171910dd918b36f5d9` | All six original documents are byte-identical. The alternative WAL witness fixture is superseded by the accepted main fixture. | Close with explicit document inclusion and fixture supersession; do not claim ancestry or a new merge. |
| [#104](https://github.com/djshellshoxxx/circula/pull/104) | `bcdc569cbc483761bf2d0e2d7594636ac1713d7d` | All 118 original paths accounted for: 14 identical, two Python alternatives superseded, 102 unique documentary paths recovered separately byte for byte. | Close with explicit runtime supersession and documentary recovery; do not claim full-tree inclusion in main. |

All four PRs were open and draft when read. Direct branch-ref reads independently confirmed all four exact source heads. Main and the evidence branch were also read directly. See `provider-readback.json`.

## PR #111 fixture review

The seven-path original delta was recomputed from original base `839e86935767288e594914ad711486616a604eca`. Six documents match actual main in full Git tree identity. The remaining file is `internal/service/sqlite_ioerr_linux_test.go`.

The entire source-to-main fixture difference consists of a `database/sql` import and the witness snapshot setup. Actual main retains a separate one-connection reader pool, an explicit read-only transaction, and an established SQLite schema snapshot through the failed writer closure. All 1,877 bytes following the snapshot setup are identical. The original failure, WAL, atomicity, retry, and replay assertions remain. No production source is changed by this alternative fixture delta.

Source fixture blob: `d1d19fb96357b5eabab43d926e89dffbbcca23e9`. Main fixture blob: `fd0aa8d98fca3539c5a010bc22afc7ff5549d09d`.

## PR #104 runtime review

The original 118-path PR delta was independently recomputed from reported original base `ead63b5ce24b2bcf8352d75fd0d1ff7be711f625` and reconciled with the preserved path inventory.

Fourteen files are identical to actual main: the native-reader workflow, all three Go verifier/test files, and ten scoped parity/qualification documents. The only differing runtime/test paths are `scripts/read-export.py` and `scripts/test_read_export.py`. Their actual-main bytes equal the previously reviewed `eef0c6d8c832ce994db1019eb5e7d6a41800b8ef` versions.

The two Python differences preserve the old manifest validation and add nested-state/repeated-array compatibility plus fail-closed compressed-stream completion. Static AST comparison confirms that all **38 original test methods are unchanged**, not merely present by name. Main contains **49** test methods, including 11 added cases. These protections must be retained; merging the older runtime wholesale is unnecessary.

## PR #104 documentary recovery

The 102 paths absent from main exactly equal the separately recovered payload set. Every payload was compared again with its original source Git bytes, mode, Git blob preimage, length, and SHA-256. The archive was read in memory; no payload was extracted or executed.

| Check | Result |
|---|---|
| Archive SHA-256 | `6d7ad0e06ef2bb9a4cd1a70626ae328a8384828dd439e07c4bf07246ff37d385` |
| Compressed bytes | 1,205,996 |
| Original files and bytes verified | 102 files; 5,206,983 bytes |
| Nested ZIP payloads verified | 205 payloads plus the internal manifest; 206 entries total |
| Internal/external manifest identity | Byte-identical |
| Parsed JSON | 56 original documents; 44 nested ZIP members |
| Package checksum entries | 9 verified |
| Entry safety | Regular, unique relative files; no extracted paths, symlinks, encrypted members, or directories |

These are historical records. Their preservation does not independently attest the original execution hosts, turn historical passes into current qualification, resolve missing historical Git objects, or change the recorded scanner false-positive limitations. The documentary paths remain outside main at their original paths, with durable recovery on the evidence branch.

## Evidence and limits

- [Pinned existing containment evidence](https://github.com/djshellshoxxx/circula/tree/c1a34476de017ab70acc99181ebbe16c337fc0e6/docs/consolidation/evidence/core-containment-20261120)
- [Pinned existing recovery package](https://github.com/djshellshoxxx/circula/tree/c1a34476de017ab70acc99181ebbe16c337fc0e6/docs/consolidation/evidence/pr104-documentary-recovery-20261120)
- `proof.json` records the independent recomputation, all 102 original payload checks, all 205 nested ZIP payload checks, source/main path identities, test-method comparison, and package hashes.
- No product tests, archived programs, migrations, workflow dispatches, PR writes, or repository mutations were performed by this reviewer. Only the requested intermediate review files were written.
- This review leaves policy PRs #83, #85, #86, #94, and #107 and all separately owned future work unchanged.

Before closure, refresh actual main, all four source heads, and their PR states. If any source moved, inspect its new delta. Verify actual-main post-merge gates through the independent CI review, then record each reconciliation in its own accurate terms.
