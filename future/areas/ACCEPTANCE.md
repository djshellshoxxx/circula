# Requirement-to-evidence matrix

| Requirement | Standalone implementation and evidence | Remaining gate |
|---|---|---|
| Admin-defined named areas | `Edit`, `AdminCatalog`, transactional Policy admin/MFA contract; member/MFA/tenant negatives | Actual core principal/MFA adapter |
| Optional zero/one listing area | Composite scoped FK, nullable selection, owner-authorized `Assign`/`Current`; two A/one B/unset fixture | Actual core listing ownership adapter |
| Optional approximate point/distance | Absent by explicit initial named-area scope; no coordinate/location fields or estimates | Separately adopt only if needed |
| Optional map with equivalent list | Map absent; native list/select/filter forms are primary, no JS/GPS dependency | Browser keyboard/zoom/screen reader |
| Exact location absent; image GPS removed | No location/image intake; strict unknown-field rejection and no external browser code | Media sanitizer/GPS fixture when actual photos integrate |
| Federation sharing opt-in | Persistent network policy defaults false; admin-only revisions/idempotency; association-free export | Actual federation adapter/privacy review |
| Hidden/archived area lifecycle | Hidden projection omitted immediately; archive deactivates with revisions; restore does not reactivate | Fresh independent audit |
| Auth, tenant and stale write isolation | Trusted policy checked in transaction; fresh retry authorization; current owner checks; stale/racing writes | Core concurrency/revocation race qualification |
| Atomic durability and backup | Reopen, closed backup/restore, injected audit failure rollback, retry fingerprint regression | Production backup/restore/full regression |
| Bounded filtering and cursor privacy | At most 1,000 scans/100 results, encrypted tenant/member/area/expiry cursor; tamper/scope/expiry tests | Browser pagination accessibility |
| API/CLI/no-JS forms | Exported unregistered Web, native forms, `RunCLI`/`RunPagedCLI`; CSRF/body/coordinate/XSS negatives | Actual root registration/help/wiki, deployment checks |
| Dependency notices | Exact Go pins/checksums; 25 resolved-module notice records, no missing cached notice | Distribution-level review at release |

The module is ready for a fresh independent source/security audit; this matrix does not declare the full product feature or Circula core complete. Integration remains HOLD.

Fresh independent audit is complete for the adopted standalone named-area scope; see AUDIT.md. Additional regressions prove acknowledged writes persist, schema corruption fails atomically, stale active replays cannot revive selections, listing cleanup retires cached coarse-location payloads, and network sharing can be disabled without allocating a retry record even at quota exhaustion. Existing core/browser/operational gates remain.
