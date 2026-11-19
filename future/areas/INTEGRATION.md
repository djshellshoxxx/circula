# Areas integration contract — HOLD

No core integration, activation or publication is authorized. Construct Service and Web explicitly, with current transaction-scoped Policy, authenticated HostAuth, host CSRF verification/token generation and a private cursor key. The module does not register routes, open a database or create a background job. See CONTRACT.md for operation types and OPERATIONS.md for schema/retention/backup details.

## Current HTTP contract

Queries are capped at 8192 raw bytes and validated before host callbacks. GET catalog (or root) and assignment accept at most one `listing`; filter accepts at most one `area` and `cursor`; admin and POST accept none. Unknown, duplicate and malformed fields return400. Existing single-empty cursor/listing behavior remains. Local canonical routes remain required.

Catalog/admin form token construction is followed by a second read-only host authentication call requiring the same identity. Then current transaction policy is evaluated before private values are read. Hosts must return current identity, honor context cancellation and not mutate state in these reads. A future concrete host must define revocation linearization across its session store and database.

## Persistence and disable

Only the explicit Service.Migrate entry installs feature-owned schema. No root migration is added. Existing listings start unclassified. Preserve audit and replay fingerprints when disabling; stop registrations/calls rather than destructively rolling back ledger or feature history. Archival removes current selections without reactivating them on restore. Listing cleanup retires cached coarse-location payloads while preserving replay refusal.

Production principal/MFA/listing adapters, replay-preserving retention, consistent backup/restore, deployment/session/CSRF and full core regression remain separate gates. Browser/manual accessibility is pending. This module accepts no image uploads or precise location data; media EXIF validation belongs to any later separately qualified media adapter.

## Form parser ordering

The adapter applies its64KiB cap and strictly parses URL-encoded bodies before host Auth/CSRF callbacks can call ParseForm or PostFormValue. Host callbacks may inspect the validated PostForm. Earlier middleware must either pass the original body through or reject its own parsing errors; it must not pass a partial form cache after silently discarding malformed body fields. JSON processing remains unchanged.
