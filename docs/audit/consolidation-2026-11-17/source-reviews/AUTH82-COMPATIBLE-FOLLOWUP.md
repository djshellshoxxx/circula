# PR82 revised compatible slice — source approval

**Source approval applies only to `5c39abdb89faa505533e51d3878b3b119957ff32`.** Tree: `302beed8428063a8df33ab68ba3256c7e1976f5f`. Merge base/current main at inspection: `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a`; ahead4/behind0. CI and aggregate gates remain the integration engineer's responsibility.

## Exact preservation checks

- `internal/service/recovery.go` blob `f3e1681c1634e2afac23f758bd3c1af0eeb0b5a2` equals current main. The reviewer reset restriction is fully removed from the candidate tree.
- `internal/service/auth.go` blob `97316e591698792ad3bece18063e4883aac4e3c2` equals the previously reviewed/tested head `0a7ef6ee8efbec15c4603bd9fbb14492c7e4ddbf` byte for byte. F04/F06 implementation did not change.
- The bodies of `TestLoginStraddlingPasswordChangeGetsNoSession`, `TestChangePasswordReplacesOnlyTheVerifiedHash`, and `TestChangePasswordGuessesAreThrottled` remain byte-identical. The only test-file removals are the policy-specific F02 test and its now-unused testutil import.
- The net diff against current main contains three files: auth implementation, its regression file and the audit note. No recovery implementation diff remains.

These three existing test bodies passed with `-race -count=1` during the prior exact-head run. No additional tests were run for this narrow follow-up; unchanged blobs/body hashes are recorded in `AUTH82-FOLLOWUP-5c39abd.json`.

## Policy language

The updated audit note explicitly states F02 is not fixed in PR82, acknowledges that an operator-only reviewer reset rule is a new recovery-policy choice, and names the alternatives. The role-revoke/reset/regrant limitation is accurately labeled source review that has not been executed. A3-F02/DEC014 remains unresolved. This resolves the source-review policy gate for the compatible F04/F06 slice.

Minor existing provenance wording remains in the note's opening: “Fixed on main@28ed716” should be read as implementation based on that main baseline, not as evidence that this unmerged branch's fixes already shipped there. The separate PR metadata and this review identify the actual candidate SHA.

## Preserved policy candidate: PR83

- Branch `claude/aud4-f02-reviewer-reset-policy`.
- Head `cb95bb1894cd3b178f99ff3c54e0b04651d68cff`.
- Tree `30dd02f228093839b331f3667f9a2515e0c6ff79`.
- Single parent/base `cdc8112cf80ca51fdd220cd2f86c5180180cbe2a`.
- Files: modified `internal/service/recovery.go` and new `internal/service/aud4_f02_test.go`.
- Draft title/body/commit clearly say policy pending and no merge without owner decision.
- The reset guard and direct regression from earlier PR82 are preserved. There is no dependency on PR82's auth implementation; its descriptive audit link points to the note introduced by PR82.

Preserve PR83 and branch. Do not integrate this policy candidate into core without the owner's recovery-policy choice. No remote or source mutations were made by this reviewer.
