# Core acceptance continuation — operator JSON files

Reviewed source: `acdc3485ad9c7355050f12b426c973b45a5f42b8`. Repair commit: `acaf8610b870d2ff913cebcd350f0a15c209e2a1`.

## Confirmed gap and repair

`cmd/circula/config.go:readConfigFile` used a single streaming JSON decode. It accepted the first configuration object while silently ignoring trailing data, and treated `null` as an empty configuration. The real `serve -check-config` command therefore reported success for concatenated or null files.

`cmd/circula/auditanchor.go:cmdAuditAnchor` had the same unread-tail behavior. A correctly signed first object followed by unrelated content still reported a successful file verification. A null file reached the semantic version check instead of being rejected as a malformed document.

The bounded repair requires a nonnull object and EOF after decoding. Existing numeric precision, booleans/strings, known-field rules, signing behavior, ordinary surrounding whitespace and unsupported-version status remain covered. No new financial, privilege or governance semantics are introduced.

## Actual verification

- Before: six direct config cases and three command preflight cases failed. That initial combined process was killed when starting the anchor test; it is not a complete suite result.
- Separate unchanged-source anchor retry: six failing boundary cases, with valid signed/whitespace and other existing refusal controls passing.
- After: six top-level tests and 26 subcases pass with race instrumentation and default vet, zero failures/skips/race reports, package duration 3.415 seconds. The test also verifies that config preflight does not create its database and anchor verification preserves the database checksum.
- Formatting and whitespace checks pass. Worktree clean after commit.

Receipts: `operator_json_verification.json`, `operator-json-before.log`, `operator-anchor-before.log`, `operator-json-after.log`.

## Scope

Reviewed the current Core completion notes and inspected command configuration, audit verification, recovery command entry points, and selected API query/input handling. Historical spec status rows are not treated as current missing implementation. Future modules and recorded policy/external gates remain separate. This review stops after the bounded operator-file repair and does not assert exhaustive specification acceptance or full Core completion.
