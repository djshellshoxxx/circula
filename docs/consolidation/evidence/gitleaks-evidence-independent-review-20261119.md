# Independent review of exact Gitleaks evidence exceptions

**No concrete overbroad-exemption or CI/harness defect identified.** Reviewed
commit `dd3357d3cb155bbe29de47d1bb788bff38d7d35f`, tree
`0942998f7a50d7c25f704d55403689a9d2f33ec5`, with configuration blob
`e138f4232f4021d8c691937d3ba30b66210fa524`. This was an independent read-only
source and provenance review; the reviewer did not compile or rerun Gitleaks.

The change adds six rule-local allowlist blocks under `generic-api-key`. Each
uses `AND`, one anchored literal document path and anchored literal values with
`regexTarget = "secret"`. There are 17 path/value pairs, not a general digest,
Git-ID, line, directory or rule exemption. The default-rule extension and both
existing global fixture allowlists are unchanged. The custom rule specifies no
matcher, entropy, secret-group, keyword or path override.

Inspection of the cached source for the pinned Gitleaks v8.28.0 confirmed the
configuration merge inherits those default matcher fields and appends the new
allowlists. Its `checkCommitOrPathAllowed` defers an AND allowlist containing
regex predicates, so an allowed path alone does not skip the document. Its
`checkFindingAllowed` then requires both the path and the captured secret value.

Independent Git/source checks reproduced six source-file digests, resolved all
four claimed Git commit objects and matched both shared-page source identities.
The matched prose value is the literal phrase `CSRF/idempotency`. The detailed
provenance results are in
[gitleaks-evidence-independent-review-20261119.json](gitleaks-evidence-independent-review-20261119.json).

## Harness and CI coverage

The harness validates the exact marked rule-local section and constructs its
baseline by removing only that section. It uses the real scanner against
temporary copies, with strict report-type, exit-code and rule/path/line-count
checks. Its controls distinguish original nonsecret values from replacement
and extended values, prefixed/suffixed paths, same-line generic and GitHub-token
controls, and an independent rule matching the same values. This directly tests
the principal ways these exceptions could accidentally become broader.

Synthetic controls are generated from public labels at runtime. Scanner reports
are redacted, and diagnostic comparisons expose finding metadata rather than
matched secret text. The repository documents are read, not rewritten.

The existing CI secret-scan step runs the harness using the unchanged pinned
`github.com/zricethezav/gitleaks/v8@v8.28.0` command and then retains the full
directory scan. Events and permissions do not change. The harness does not
replace the repository scan or convert scanner failures into success.

The author's eight-control and directory-scan receipts were inspected as
supporting evidence; those runs remain attributed to the author. This review
does not claim a new scanner execution, native-provider result, scan of Git
history or absence of credentials in an unscanned future source tree.
