# Circula vX.Y.Z

Copy this file to `docs/releases/vX.Y.Z.md` before tagging. The release workflow refuses to publish without it (OPS-026).
Replace every bracketed item or write "none".

## Breaking changes
- [none]

## Schema changes
- [none] If the release changes the schema, state the migration version and whether a backup is required before upgrade (required: yes/no).

## Configuration changes
- [none] New, renamed or removed settings (see docs/ops/configuration.md).

## API changes
- [none] Per the compatibility rules in docs/specs/07-api-contract.md.

## Security fixes
- [none]

## Required operator actions
- [none] For example: take a backup, run `circula migrate`, rotate a secret.

## Verification
- Checksums: `SHA256SUMS` attached to the release.
- Checks run for this tag: [list, and any target not run and why].
