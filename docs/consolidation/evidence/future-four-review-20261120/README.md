# Four isolated future PRs: independent source-review evidence

See [the review and exact disposition](../../FUTURE_FOUR_SOURCE_REVIEW_20261120.md) for PRs #113, #114, #115 and #120. Final provider readback: 2026-11-20 16:49:29 UTC.

| File | Evidence |
| --- | --- |
| `provider-initial.json` | Initial live PR metadata including bodies; exact-head PR workflow runs and latest-attempt jobs/steps. |
| `provider-final.json` | Final source/base/state metadata, submitted review lists, and refreshed exact-head PR runs/jobs/steps. |
| `source-inventory.json` | Full changed-path/commit inventory, merge-base and merge-tree outputs, Core blobs, module source hashes, preserved-file comparisons, manifest and dependency boundaries. |
| `receipt-verification.json` | Historical source/log SHA-256 comparisons and accepted-main Core restoration provenance. |
| `final-verification.json` | Initial/final source-target drift comparison and explicit import/initializer/manifest checks. |
| `SHA256SUMS` | SHA-256 of the review and all evidence files, excluding this checksum file itself. Paths are repository-relative. |

The reviewer executed only source/provider/evidence inspection and read-only Git merge simulations. It did not run Go/Node tests, fuzzing, load traffic or pre-fix reproductions, and made no remote or application-source mutations. Current provider results are distinguished from historical source-bound local qualification. Original archive commit objects were not present in this checkout; preservation comparisons use the committed per-file recovery receipts, with that limitation stated in the review.

All four candidates remain disabled and on integration HOLD. The provider returned ten successful applicable jobs per PR and one skipped conditional Core matrix placeholder per PR. The placeholder represents two configured native Core targets that do not run for PR events; separate native module jobs did execute successfully.
