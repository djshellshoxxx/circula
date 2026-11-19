# Governance integration contract — HOLD

No integration is authorized. This nested module remains independently disabled, with no root routes, migrations, dependencies, workflows or policy/ledger writers. All adapters are explicit constructor seams; importing the package starts no task and opens no DB.

The future host must implement current authentication, network opt-in, active membership, scoped administration/MFA and CSRF. Authority reads must use the supplied database transaction. Read-only PolicyExists confirms an existing same-network policy version; governance never applies that policy. The own-ballot-only API does not imply anonymous voting; protected database operators can inspect ballots.

The web adapter permits only bounded, strict proposal/format/cursor queries. Cursors bind member/network. Native forms intentionally submit to the current document URL; a retained proposal query must match the command body. Invalid cursor/query inputs fail before mutation. Token callbacks require fresh same-identity authorization before private form reads. Host callbacks must not mutate the request or return client-supplied authority.

Retain independent runtime flags false and do not register adapters until separately selected. CONTRACT.md specifies transaction/HTTP/CLI requirements. OPERATIONS.md specifies additive schema, backup/restore, logical quotas and retention. No destructive migration or bulk erase is introduced. Disable by withholding registration and network opt-in; preserve governance history and uniqueness records.

Required later qualification: actual core authority/admin-MFA/store/policy-history adapter, core/API/ledger regression, deployment retention/backup policy, browser/keyboard/screen-reader/translation acceptance and owner-selected rollout. These deferred actions must not be performed by this standalone branch.
