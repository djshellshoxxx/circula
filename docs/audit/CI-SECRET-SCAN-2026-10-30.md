# Pinned secret-scan diagnosis — 2026-10-30

Source: `337667b54fbf252c1ceac2cebe377530bad481e0`.
[Push run 41042298049](https://github.com/djshellshoxxx/circula/actions/runs/41042298049),
lint job `124017890794`, failed at the pinned gitleaks v8.28.0 step. Workflow-pin,
hygiene and dependency-licence steps passed. Later gofmt/vet/lint/govulncheck steps
were skipped, not passed. Redacted log retrieval through the GitHub connector
succeeded; no dependency or build-artifact bytes were downloaded locally.

Observed scanner output:

```text
scanned ~4530444 bytes (4.53 MB) in 1.51s
leaks found: 1
exit status 1
```

The existing command omitted verbose output and returned no file/rule detail.
The generic connector rejected check-annotation retrieval as an unsupported URL;
that route was not retried. A limited local credential-pattern scan is not the
same detector and did not identify this finding. Do not infer a real credential
or blame a particular document without the scanner evidence.

The coordinator reserves only diagnostic CI output and its handoff/receipt.
The sole workflow change appends `--verbose` while retaining `--redact`, exact
version v8.28.0, full-directory coverage, configuration, default failure status,
permissions and action pins. The [pinned tool README](https://github.com/gitleaks/gitleaks/blob/v8.28.0/README.md)
documents verbose output and default 100% secret redaction. This adds redacted
finding location/rule diagnostics; it does not create an allowlist or suppression.
The scanner's original failure remains linked above. Runtime audit ownership and
existing PoC assertions are unchanged.

Local workflow-pin, hygiene and whitespace checks are required for this change;
the exact scanner remains unavailable locally. The next CI scan must identify
the actual finding before the smallest correction is chosen. No successful
scanner, aggregate test or runtime-remediation result is claimed here.

Independent read-only delta review found no blockers: the only workflow change
is `--verbose`, with 100% redaction, version, scope, configuration, failure
propagation and permissions unchanged. Workflow-pin, hygiene and whitespace
checks passed independently. The reviewer also checked the pinned tool README;
no scanner run or passing rescan is implied.

## Identified cause and minimal correction

Source `a02eedfbe526ef9075d376f905e1ab82607b7e79` reproduced the one-finding
failure in [push run 41042900829](https://github.com/djshellshoxxx/circula/actions/runs/41042900829),
job `124019828103`. Redacted verbose output identified:

```text
RuleID: generic-api-key
Entropy: 3.634830
File: docs/audit/CONSOLIDATION-2026-10-30.md
Line: 161
Fingerprint: docs/audit/CONSOLIDATION-2026-10-30.md:generic-api-key:161
leaks found: 1
exit status 1
```

The matched value is the plan-merge Git commit ID recorded in the consolidation receipt. Its
credential-like prose label caused the generic matcher to treat that hexadecimal
ID as key material. `git cat-file -t` confirms it is a commit and its tree matches
the recorded resolved consolidation tree. It is not a live credential. The
existing review note about an audit JSON idempotency key was only a lead and was
not this observed finding; no audit JSON or external PoC was modified.

The correction changes the two merge labels to "Receipt merge commit" and
"Plan merge commit", retaining all full SHAs and provenance. Only owned receipt
prose and the handoff change; scanner configuration/rules/coverage/version,
failure status, redaction, permissions and runtime code remain unchanged. No
allowlist, inline suppression or detector weakening was used.

The original failure, diagnostic failure and source SHAs remain preserved above.
Local whitespace, link, hygiene and action-pin checks verify the correction; the
full pinned scanner and aggregate results must be observed at the new exact head
in PR #17. Local scanner/runtime execution is still unavailable.

Independent delta review reproduced the redacted CI finding, verified the commit
object/tree, and confirmed both label changes retain every full SHA in order.
Only three Markdown paths change; hygiene, action pins and whitespace pass.
No suppression, allowlist, scanner-rule or runtime change was made.
