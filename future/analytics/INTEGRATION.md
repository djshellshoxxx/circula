# Analytics integration boundary

Status: **HOLD**. Feature stays disabled. This module is not registered, merged, released or deployed. No root dependency, core route, migration, feature activation or ledger change is authorized by this continuation.

A later explicitly selected integration must provide a current authenticated historical `Source` in one read-only transaction, including immutable original and reversal facts, account classification and exact cutoff balance stock. It must qualify membership/admin revocation, own-account authorization and snapshot consistency. See CONTRACT.md for the complete port contract.

Before that integration, standalone release policy work remains: define and enforce immutable publication across restart/corrections, a network-scoped query budget and an approved correction procedure. Fixed granularity and result suppression do not substitute for those requirements. The host must not expose configuration as caller-controlled slices.

The package currently has no store, cache, worker or schema. There is no feature data migration or backup operation to perform. That absence does not qualify persistence of published releases. Future storage requires its own restart/concurrency/failure/restore evidence and retention review. Disabling a later adapter removes its registration and denies new reads without rewriting any ledger history.

Do not integrate other future modules together with this feature. Actual core contracts, full-ledger regression, browser/manual accessibility, privacy and operational qualification remain separate gates.
