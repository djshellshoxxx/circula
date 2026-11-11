# Circula vX.Y.Z

<!--
Copy this file to docs/releases/vX.Y.Z.md before tagging, replace every bracketed item, and write "none" where nothing
applies. The release workflow runs scripts/check-release-notes.sh on it and refuses to build or publish while a
section is missing or empty, a placeholder or vX.Y.Z is left, or the evidence directory is not linked (OPS-026,
QA-039, QA-044). This comment may stay; the check ignores HTML comments.
-->

## Breaking changes
- [none]

## Schema changes
- [none] If the release changes the schema, give the migration version and keep the next line.
- Backup required before upgrade: [yes/no]

## Configuration changes
- [none] New, renamed or removed settings (see docs/ops/configuration.md).

## API changes
- [none] Per the compatibility rules in docs/specs/07-api-contract.md.

## Security fixes
- [none]

## Required operator actions
- [none] For example: take a backup, run `circula migrate`, rotate a secret.

## Evidence
Release gate items (QA-039), each linked from here. Records live in the evidence directory (QA-044, docs/qa/EVIDENCE.md).
- Evidence directory: [docs/qa/evidence/vX.Y.Z/](../qa/evidence/vX.Y.Z/)
- Release gate record (`scripts/release-gate.sh -r vX.Y.Z`): [gate-UTC.md link]
- Restore drill record (QA-029): [drill-UTC.json link]
- Traceability matrix at the release commit (QA-042): [traceability.md link]
- Quality thresholds and their decision references (QA-040): [copy the table from docs/qa/THRESHOLDS.md]
- Keyboard-only walkthrough (UX-055): [date, tester, browser; pass or fail per flow: sign-in, propose trade, accept
  trade with review, decline, history view, admin member approval; link to the record]
- Manual accessibility review and screen-reader run (QA-035, docs/qa/SCREEN-READER.md): [link]
- Exploratory session records (L14): [link]
- Independent security review (SEC-033): [link]
- Performance results on recorded reference hardware (L15, docs/qa/LOAD-TEST.md): [link; hardware, date, commit]
- Smoke run per target (L13, OPS-003): [release workflow run link; for each target not run, the target and why]
- Checksums and bills of materials (OPS-035, SEC-029): `SHA256SUMS` and one `circula_vX.Y.Z_<os>_<arch>.cdx.json`
  (CycloneDX 1.5) per target, attached to the release.
- Decisions accepted, changed or superseded since the previous release: [DEC ids, or none]
