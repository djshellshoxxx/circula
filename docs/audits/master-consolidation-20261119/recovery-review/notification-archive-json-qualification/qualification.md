# Archived notification JSON field-alias repair

## Frozen source

Base `ba454e1c14d35bdd90dca8379e890f2bca6281d7`, local commit `3c734042159899c9d1dc737f70c535e82010c40f`, tree `f3668a89acb046d159525cddf8cd5192467a26de`. Only `future/notifications/strictjson.go` and the new `archive_json_simplefold_test.go` change. A portable patch is saved as `0001-notifications-archive-json-simplefold.patch`. The clone/branch remains isolated and local. No PR103, PR100 or Core source is changed.

## Actual boundary and repair

The archived `uniqueJSON` parser promises rejection of duplicate fields including alternate casing, but only folded ASCII before typed decoding. The actual production function accepted seven regression subcases involving raw/escaped Kelvin sign (`K`/`K`), long s (`s`/`ſ`), Greek sigma variants and a nested-object occurrence. These are Go simple-fold aliases; typed Go field matching can treat their spellings as the same key. This is a parser-conformance defect; no authorization bypass is claimed.

The repair compares each field name using the minimum rune in the `unicode.SimpleFold` cycle, matching Go's equivalence without expanding characters. The existing duplicate policy is unchanged. Single Kelvin/long-s/sigma aliases still pass and decode into their Go fields. Valid Unicode and literal U+FFFD remain accepted. Sharp-s versus `SS`, ligatures versus expanded letters, dotted/dotless I versus ordinary I, and identical aliases in separate objects remain distinct as Go requires.

## Evidence

| Check | Exact scope and result |
|---|---|
| Fail-before | `go test -p 1 -count=1 -timeout=45s strictjson.go errors.go archive_json_simplefold_test.go`; original production source, seven failing alias subcases; package0.011s, wall1.915s. Existing ASCII duplicate and all valid controls passed. |
| Green race | `go test -race -p 1 -count=1 -timeout=45s strictjson.go errors.go archive_json_simplefold_test.go`; PASS1.029s, wall2.646s. |
| Green vet | `go vet -p 1 strictjson.go errors.go archive_json_simplefold_test.go`; PASS, wall0.298s. |

The three test methods cover eight duplicate cases, six valid single-alias/value cases and six distinct-field/object controls. Every receipt captures exact source hashes, command, environment and full output. The frozen source equals those green hashes; dependency files are unchanged and `git diff --check` passed. These are actual source-file checks of the production parser using only the standard library. They are not full module, database, public HTTP handler or composed-runtime qualification. No competing heavy build ran in the shared slot.

## Review and composition boundary

Independent source review by `/root/branch_recovery` approved the full frozen two-file diff with no blocking finding. The review confirmed Go simple-fold equivalence, single-alias compatibility and nonexpanding Unicode controls. This patch is separately based on exact archived source and does not contain identity patch `b96100d2a53cf696f16b8c34305eb132c673863c`; the two patches touch disjoint source paths. They can be considered independently during deliberate runtime reconciliation. Neither authorizes a folder replacement or changes API/schema/events/retention/unsubscribe design choices. The future module remains on HOLD.

Both portable patches also pass `git apply --check` against the other frozen archival repair worktree. No patch was applied and neither worktree changed; this confirms textual applicability only, not composed-runtime qualification.
