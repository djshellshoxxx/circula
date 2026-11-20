# Evidence retention and records (QA-044, QA-029)

## What is kept per release
Test reports, gate records, drill records and the traceability matrix. Keep them for the release's supported lifetime
plus one year (QA-044 recommended value; confirm under DEC-023).

## Where
`docs/qa/evidence/<version>/` holds, for a tagged release:
* `gate-<UTC>.md`: output of `scripts/release-gate.sh` (see docs/qa/RELEASE-GATE.md for the format)
* `drill-<UTC>.json`: restore drill record from `scripts/restore-drill-full.sh` (QA-029, OPS-021)
* `traceability.md`: copy of docs/TRACEABILITY.md at the release commit
* links to manual records (screen-reader run, exploratory sessions, security review) which live outside CI

Untagged working records sit directly in `docs/qa/evidence/`. Move them into the version directory when a release is cut.
Evidence is committed to git, so it is retained as long as the repository; the release notes
(docs/releases/vX.Y.Z.md) MUST link the version directory. `scripts/release-gate.sh` does not check this link: release
notes are a MANUAL row in the gate record (see RELEASE-GATE.md).

## Restore drill record (QA-029)
Run `scripts/restore-drill-full.sh [-s SOURCE_DB]`. It backs up, restores into a separate directory, verifies the ledger on the
restored copy and writes `drill-<UTC>.json` naming the backup file, restore duration (seconds) and outcome. A `fail`
outcome blocks the gate until the cause is recorded. Without `-s` the drill uses an empty migrated database: that proves
the procedure only, not recovery of real data; for a hosted launch, drill with a copy of a real backup.
Current record: docs/qa/evidence/ (one drill from this build, empty database).

## CI evidence is temporary
The workflow uploads Go test/traceability evidence for 14 days and browser/build artifacts for 7 days. These artifacts are working evidence, not the release retention record. Before a supported release, archive the required reports under the version directory above and link them from its release notes; an expiring Actions link alone is insufficient. Confirmation of the proposed supported-lifetime-plus-one-year duration remains DEC-023.

## Retention rules
* Never delete a record for a release still supported.
* Records contain no personal data (synthetic or empty databases only; QA-043).
