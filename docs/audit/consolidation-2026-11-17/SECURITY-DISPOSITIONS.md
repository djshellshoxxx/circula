# Current security finding disposition

Scope: integrated main `b4398074ae4ca72559235ca50242cfcb7a934eec` (tree `1e2923efdff1adcf60c2718fe2a6a56edb3c1ed9`). Reviewed candidate `17152dadad91d4e1c994ea807e7d6e2497513aa8` (tree `1e47c9c30165ca27866887533586c90323e1630e`) is separate; unmerged fixes retain that status. Exact integration is recorded in [the execution report](README.md).

Of the 55 historical findings, the initial review classified 49 as addressed and six as residual. A3-F21 now has corrected implementation documentation. Later comparisons with the original findings withdraw the broad fixed classifications for A2-08 and A3-F10: their catalogue and retention/logging portions remain unmerged. **48 findings are addressed by reviewed code/tests or documentation; five retain explicit policy/design residuals and two remain partially resolved.** These counts do not include the separate E-02, NEW-01 or later AUD4 findings. They are not a claim that every requirement or possible vulnerability is covered.

Qualification of merged PRs and the separate, still-unmerged combined candidate is described in the execution report. Original and concurrent runs, provider checks and local process terminations retain their exact source identities; the candidate pass is not attributed to the current main tree. Focused reproductions and independent mutations are recorded in [evidence](evidence/) and the [independent review](source-reviews/INDEPENDENT-REVIEW.md). Exact per-finding source/test mappings and original notes are preserved in [security-disposition.json](security-disposition.json).

| Finding | Severity | Disposition | Current evidence / residual |
|---|---|---|---|
| A1-01 | Medium | fixed | Network-state gate refuses settlement postings after closure, during suspension and when an administrator is required. |
| A1-02 | Medium | fixed | Scale changes refused at creation boundary; scheduled-change bypass regression covered. Historical databases with already-scheduled scale changes still require migration policy. |
| A1-03 | Medium | fixed | Re-review posting refused while another live dispute owns the transaction. |
| A1-04 | Low | fixed | networkOpenErr gates reverse; PR69 adds entry-point regression, independently mutation-tested in this review. |
| A1-05 | Low | fixed | Frozen networks excluded before the bounded sweep; healthy expiry does not starve. |
| A1-06 | Low | fixed | Departure cancels proposals beyond the first 500. |
| A1-07 | Low | fixed | Closure with an absent final checksum remains eligible for retry. |
| A1-08 | Info | fixed | Operator changes preserve scheduled policy values and allocate the next revision. |
| A1-09 | Info | fixed | Reversal-target audit events and ledger postings share the transaction. |
| A1-10 | Info | fixed | Frozen pending adjustments excluded before bounded selection. |
| A1-11 | Low | fixed | Current tests enforce scale immutability; remediation record amended FR-005/GOV-004. DOM-031/DAT-049 remain separately qualified. |
| A2-01 | Low | fixed | Canonical JSON field names reject case aliases; strict body tests remain. |
| A2-02 | Low | fixed | Network-bound tokens cannot manage person-wide tokens/preferences. |
| A2-03 | Low | fixed | Replay reauthorizes scope, role, MFA, break-glass expiry and current membership. |
| A2-04 | Low | fixed | Adapters use live effective admin authority including authorized break-glass grants. |
| A2-05 | Low | fixed | Administrator reads/navigation enforce the currently required MFA. |
| A2-06 | Low | fixed | Failed server revocation does not clear retryable session or claim success. |
| A2-07 | Low | fixed | Conditional writes include effective, pending, override and expiry state, with atomic service recheck. |
| A2-08 | Low | partly_fixed_documentation | Runtime errors are validated against OpenAPI responses. The original finding also requires four missing problem-code catalogue rows. PR #84 supplies these rows; its supplied regression independently fails on the exact baseline and passes on the candidate. It is not yet counted as integrated. |
| A2-09 | Low | fixed | Bounded signed pagination for policy logs and challenges. |
| A2-10 | Low | fixed | Invitation-only networks remain opaque to outsiders; public join modes retained. |
| A2-11 | Low | fixed | Navigation reflects readable suspended/departed sections. |
| A2-12 | Low | fixed | Re-review and dual-control approval use signed reviewed confirmation, rechecking live state. |
| A2-13 | Info | fixed | Notice times render as time elements rather than escaped markup. |
| A2-14 | Low | fixed | Rendered invitation/login/register links retain one correctly escaped local return path. |
| A2-15 | Info | fixed | Cacheable public assets issue no CSRF cookies. |
| A2-16 | Info | fixed | Record identifiers rejected before repository calls after authorization, preserving opaque failures. |
| A2-17 | Info | fixed | Home and invitation entry points enforce forced password change. |
| A2-18 | Info | fixed | Decoder messages hide Go internals; unknown/malformed queries refused. |
| A2-19 | Info | fixed | POST/tokens correctly documents no Idempotency-Key support; token secrets are never cached. Canonical API-066 SHOULD remains design guidance, not a runtime replay promise. |
| A2-20 | Info | fixed | Edge and adapter request identifiers are correlated; untrusted client identifiers do not replace the server audit identifier. |
| A2-21 | Info | deferred_design | hashToken still uses plain SHA-256, so API-009 salted/keyed-hash wording is unmet. Tokens contain 256 random bits; credential format migration and key/salt lookup compatibility need explicit design. No unapproved token revocation or format change. |
| A3-F01 | Medium | fixed | Encrypted pre-migration backups decrypted before database integrity verification. |
| A3-F02 | Medium | owner_policy | DEC-014 remains unresolved. Current original PoC re-run separately; no break-glass/governance policy changed. |
| A3-F16 | Medium | fixed | Session-side operations cannot renew/consume the owner sign-in MFA budget. |
| A3-F03 | Low | fixed | requireDecider enforces admin MFA; PR69 service-entry regression independently mutation-tested. |
| A3-F05 | Low | partly_fixed_policy | Recovery and sign-in throttle families erased, including Unicode/IP variants. Challenge reasons still survive; SEC-014/018 retention policy unresolved. |
| A3-F06 | Low | fixed | SQLite character-length cleanup replaces mismatched Go-byte length and respects literal prefixes. |
| A3-F09 | Low | fixed | Restore invalidates bearer sessions/tokens and break-glass credentials before swap. Password/recovery-code/invitation state still reflects the backup; separate operational limitation. |
| A3-F10 | Low | partly_fixed_implementation | Cadence exclusion is fixed. The original finding also covered premature retention pruning and missing clock-anomaly logs. PR #84's retention tests fail with the prior algorithm restored and pass on the candidate; its latest logging change is independently reviewed and regression-verified; it remains unmerged while required provider CI is incomplete. The unmerged candidate is not counted as integrated. |
| A3-F11 | Low | fixed | Same correction as A1-08. |
| A3-F13 | Low | fixed | Explicit audited revoke-sessions command and corrected key-rotation documentation; no claim that key rotation alone revokes opaque sessions. |
| A3-F14 | Low | fixed | Protected append-only UPDATE/REPLACE/UPSERT and DROP TRIGGER checked, with lexical/comment/exemption regressions. |
| A3-F15 | Low | fixed | Scheduled ledger verification is read-only while maintaining in-memory freeze/thaw behavior. |
| A3-F17 | Low | fixed | Manifest/record/anchor writers use unpredictable exclusively-created files. Plaintext snapshot staging residual is NEW-01, handled separately. |
| A3-F18 | Low | owner_policy | Per-member limit cuts still apply immediately. GOV-003/default-policy notice versus per-member notice requires owner decision; no financial-rule change. |
| A3-F19 | Low | fixed | Operator authentication settings bounded, validated and wired into service. |
| A3-F25 | Low | fixed | Notification erasure targets exact resource relationships, preserving unrelated name substrings. |
| A3-F07 | Info | deferred_design | PASSIVE checkpoint can retain erased plaintext in WAL frames. Forced TRUNCATE policy and concurrency/availability consequences unresolved; secure_delete alone does not prove historical-page removal. |
| A3-F12 | Info | fixed | Nonpositive migration metadata refuses cleanly instead of indexing out of bounds. |
| A3-F20 | Info | fixed | ZIP-specific listing/extraction implemented. This review independently ran clean and forbidden-name ZIP fixtures; exits 0 and 1 respectively. |
| A3-F21 | Info | fixed | PR75. Existing focused throttle/recovery tests pass; independent source/documentation review approves. SEC-006 normative wording is retained with an explicit implementation/deviation note; no owner decision or authentication policy changed. |
| A3-F22 | Info | fixed | Break-glass target login normalized consistently with sign-in. |
| A3-F23 | Info | fixed | Operator registration audited as the operator, rather than self-registration. |
| A3-F24 | Info | fixed | Erasure invalidates affected archives and running exports; admitted downloads pinned/revoked safely. |

## Explicitly retained boundaries

**A3-F02 / DEC-014 remains unresolved.** No owner policy was inferred from this assignment. The reproduced challenge-text/WAL/limit-cut residuals remain open. API-009 token-hash format, historical pending-scale changes and restore rollback of password/recovery/invitation state require compatible design or owner policy. PRIV-01–05 retention and legal-hold choices and the external SEC-033 gate remain documented.

The E-02 and NEW-01 regression notes define their demonstrated threat boundaries. The release-threshold parser validates recorded decisions without approving the real DEC-023 table. The privacy-test fix preserves all-column PII scanning and adds explicit credential-preservation checks; it does not change runtime erasure.

## Additional findings from the concurrent review

These records supplement the historical table; they are not additional entries in its 55-finding denominator. Exact merged status and current qualification are in [the integration receipts](integration-receipts.json).

| Finding | Reviewed repair or retained decision |
|---|---|
| AUD4-D01 / #81 | Escape SQLite URI metacharacters in operator-selected paths. Seventeen independent before/after cases cover literal filenames, pragmas and schema inspection. The main database setting already rejected those characters. |
| AUD4-D02 / #81 | Open-handle identity checks, Unix no-follow/link-count rules, trusted parent ownership, leased-work identity and accurately scoped platform documentation. Review findings G03/G04 were corrected at09ef8a5. Windows relies on destination ACLs; ancestor and same-user/administrator interference remain outside the check. Root-only foreign-owner test skipped locally. |
| AUD4-F04 / #82 | Use the existing durable re-authentication budget for current-password guesses, preserving the configured throttle policy. |
| AUD4-F06 / #82 | Recheck the verified identity/hash during session issuance; replace only the verified current hash; invalidate pending MFA challenges after password change. Three supplied regressions independently fail against baseline source. |
| MFA/password-proof follow-up / #82 | **Repaired in the reviewed, unmerged candidate.** Twelve baseline cases fail and the same twelve pass after transaction-bound password-hash checks; all 21 selected top-level race tests pass. The new full combined candidate also passes. Confirmation/disabling still require valid factors. Existing reset authority, throttles and errors remain unchanged. [Independent review](source-reviews/MFA-PASSWORD-INDEPENDENT-REVIEW.md). |
| AUD4-F02 / #83 | **Owner decision, not integrated.** The reviewer-reset policy extension was removed from #82 and preserved in a separate draft. It is distinct from A3-F02/DEC-014. |

The initial #82 policy-bearing review, the corrected compatible slice, and the independent baseline assertion failures remain separate source records. No fixed-head compilation termination is relabeled a test pass.

AUD4-F03 remains an additional partial finding: ordinary forward-time dispute updates are covered by #87, but independently reproduced same-millisecond and host-clock-rollback cases still require a durable opening-policy binding and compatible legacy-data handling. The proposed improvement remains unmerged while required CI is queued. It does not change the historical 55-finding denominator.
