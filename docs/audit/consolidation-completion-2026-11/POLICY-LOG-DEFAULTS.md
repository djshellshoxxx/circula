# Historical policy defaults in the rules log

`GetNetwork` and historical effective-policy reads initialize a policy with
`domain.DefaultPolicy()` before decoding persisted JSON. The two rules-log
readers instead decoded into zero values. Policies written before a field
was introduced consequently showed zero/empty values in the member log,
despite the service enforcing the defined default. A later edit could show
a change from an invented zero value.

`ListPolicyVersions` and `ListPolicyLogPage` now use the same defaults for
both current and previous rows. An explicit zero in JSON still overrides
the default. The change is read-only: historical policy JSON is untouched.

`TestPolicyLogUsesDefaultsForMissingHistoricalFields` upgrades a version-18
fixture containing partial policy JSON. It checks agreement with the live
network reader, the complete log, a one-row page whose predecessor is
outside that page, the initial version with no predecessor, an explicitly
stored zero response period, and unchanged raw policy JSON.

Formatting and diff checks passed. Go runtime qualification is delegated
to integration CI because this environment repeatedly killed the initial
SQLite compilation for memory pressure. No local runtime pass is claimed.
