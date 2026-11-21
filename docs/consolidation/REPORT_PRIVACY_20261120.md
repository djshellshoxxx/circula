# Report subject confidentiality — 2026-11-19 Pacific

## Scope and contract

This follow-up completes the ordinary role-overlap protection in WAPI-01 from
the [read-only web/API audit](https://github.com/djshellshoxxx/circula/blob/896d69f472652ed54c051ec82efcb7f162d1d2c4/docs/audits/read-only-security-2026-11-19/WEB_API.md).
[FR-024](../specs/04-functional-requirements.md#fr-024--p1--report-a-listing-or-member)
and [SEC-025](../specs/09-security-privacy.md) require reporter confidentiality
from the reported party. Gaining an administrator role does not remove that
restriction. For a listing report, the reported party is the listing owner.

This privacy checkpoint covered the existing listing and member report types.
The [transaction-report follow-up](TRANSACTION_REPORTS_20261120.md) extends the
same protection to ordinary two-member transactions and adds the exact
transaction's dispute relation. The verification counts below describe this
earlier checkpoint.

## Implemented behavior

- The administrator report queue excludes reports about the caller or their
  listings, for both open and handled reports. The repository applies this
  restriction before its pagination limit, so hidden rows do not truncate the
  visible queue.
- Report handling requires current administrator authority and an independent
  subject relationship within the same transaction. Durable idempotency replay
  and cached HTTP replay repeat the authorization check. An independent
  administrator can still replay a successful handling operation without
  repeating its audit entry or notification.
- Administrator audit reads exclude the corresponding report events before
  pagination. Other audit events retain their existing visibility. The stored
  records, sequence numbers, hashes, subjects, and handling history are intact.
- Audit and full exports promise complete audit history. A member who is the
  subject of any retained report cannot create or download these scopes; the
  generic authorization refusal contains no report identity or details. The
  check is shared by initial admission, the export snapshot, retained artifact
  downloads, and cached export creation replay. Independent administrators and
  existing System authority continue to receive complete exports. Mine,
  ledger, and member exports retain their existing authorization rules.
- Handling a report does not lift its subject's audit/full export restriction:
  the retained historical audit still identifies the reporter. Unknown or
  unresolved report subjects also fail closed for member authority. Existing
  System authority is unchanged.

The shared repository predicate resolves member and listing subjects within
the report's network. The service independently checks the subject before
returning or handling an individual report. A later role assignment therefore
does not grant access to earlier reports about that member. Reporter-only
notifications, target listing/member state, and existing administrator MFA
requirements remain governed by their existing implementations.

## Held operational decision

If every available administrator is a report subject, the application cannot
provide an independent administrator to handle that report. Existing trusted
System service authority can read and handle reports, but there is no dedicated
operator report-handling command or established complaint escalation procedure.
This change adds neither an escalation privilege nor a role-grant, quorum, or
support policy. The operational owner must define who is authorized to handle
this case and through which approved procedure. Until that decision is made,
the report remains retained and unhandled by its subjects. This operational
sub-scope of WAPI-01 remains held.

## Verification

Focused race-enabled tests passed on Go 1.26.11 across the service, API, and HTML
packages: **22 top-level tests and 44 subtest passes**, with no failures, skips,
or race reports. These include both subject types, administrator promotion,
pagination, handling and successful replay, role revocation/restoration,
snapshot and retained-export authorization, complete independent/System
exports, and reporter-only notifications. Existing report behavior and the
24-case export snapshot authorization matrix also passed.

New regression coverage is in `internal/service/report_privacy_test.go`,
`internal/api/report_privacy_test.go`, and
`internal/web/report_privacy_test.go`. Verification used ordinary local
role-overlap fixtures on the corrected implementation. This focused run is not
a claim of full repository or remote CI qualification.
