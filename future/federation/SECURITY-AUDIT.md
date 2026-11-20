# FNET-209 master continuation security record

Predecessor `e09f1598c32b11922982517b7f64a4b5a0f0b5c9` has a fresh-team audit preserved in AUDIT.md. Current source checkpoint `41d2041` changes only HTTP adapters and regression tests. This implementation review does not claim an external security certification or final protocol approval.

## Confirmed and corrected findings

1. Member pagination was parsed after host authentication without a size cap or strict duplicate/unknown-field rejection. Admin routes also accepted arbitrary query data. Query strings could be merged into form fields by ParseForm and influence mutation fields. Member pagination is now strictly parsed and bounded before host/form callbacks; admin URLs accept no query. Mutations use validated PostForm body fields only. Duplicate, unknown and malformed form fields fail before host authentication. Paginated member forms remain functional because their valid pagination context is accepted but cannot supply action/remote/listing/CSRF fields.
2. Both member and admin pages read private data before CSRF token callbacks. A callback could revoke or replace the host session while old data still rendered. Token callbacks now precede a fresh same-actor HostAuth check and the transactional Store read. Callback cancellation also denies output. Ten regressions cover revoke, replacement, member revocation, cancellation and unchanged sessions.

Current-session lookup must be a read-only operation returning consistent identity. Arbitrary concurrent revocation after the bounded recheck remains a host transaction/lifecycle contract; this patch does not claim global atomicity across external session providers.

## Review disposition

The coordinator independently reviewed runtime commit `41d2041` and found no issue in the bounded query/form/callback correction. Final source a833a8636724ac437d9680f9689f28a7257c7419 also preserves the established 413 oversized-body response, while malformed and ambiguous forms return 400. Current test results are recorded in VALIDATION.md; the coordinator also independently reviewed the final response-contract correction and adversarial/native-form regressions, with no issue found. No protocol format, database schema, dependency, signing-key behavior, transport or ledger behavior changed. Existing whole-module review remains bounded to its recorded source; the delta review must not be mislabeled a new full cryptographic audit.
