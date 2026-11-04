# Personal data inventory

Version: 1, 2026-11-04. Specifications: SEC-014 (data minimisation), SEC-017 (exports), SEC-018 (retention and erasure),
DAT section 9 ([06](../specs/06-data-ledger-audit.md#9-retention)). Derived from the migrations in
`internal/store/migrations/` (0001 to 0015).

This file is checked by `internal/store/pii_inventory_test.go`. The test migrates an empty database and fails when a
table column is classified neither under [Personal data columns](#personal-data-columns) nor under
[Other columns](#other-columns), when this file names a column that no longer exists, or when a personal-data row has an
empty cell. Adding a migration that adds a column therefore means adding a row here, in the same change.

Retention values are what the current build does. Where the specification gives a recommended default that the build
does not apply, the cell says so. Retention periods for personal data are an open decision (DEC-020, counsel review,
RSK-022): nothing here is a compliance claim.

## Forbidden data (SEC-014)

The product collects no national identity number, government identifier, bank account number, card data or date of
birth. `TestSchemaHasNoForbiddenPersonalData` fails if a column whose name suggests one of them is added. Free-text
fields can still receive such data if a person types it; their length is limited (FR-016 and the service validators)
and administrators are told not to write personal data into reasons ([erasure.md](../dev/erasure.md#the-audit-chain)).

## Terms used in the tables

- **Category**: `identifier` (names or sign-in identifiers that point at a person), `credential` (secret
  authentication material, stored only as a hash or sealed), `profile` (a setting the person chose), `free text`
  (anything a person typed, which may contain personal data), `activity` (when a person was last active), `governance
  marker` (a statement about a member made by the network).
- **Export**: the export packages of [07](../specs/07-api-contract.md) and DAT-037/038. "Network export" is the
  administrator export of the whole network; "self-export" is a member's export of their own data (SEC-017). Exports
  never contain credentials, password hashes, token hashes, MFA secrets or recovery material.
- **Erasure**: what the erasure of a departed member does (`AnonymiseMember`, `EraseMyData`, `circula erase-member`;
  see [docs/dev/erasure.md](../dev/erasure.md)). "Person erasure" means the person row is also anonymised, which happens
  only when the person has no other live membership and the request came from the person or the operator CLI. Closed
  networks are erased the same way after their read-only period (GOV-025). Backups taken earlier keep the old values
  until backup retention ends (SEC-028).

## Personal data columns

| Column | Category | Purpose | Retention | Export | Erasure |
|---|---|---|---|---|---|
| `persons.login` | identifier | Sign-in identifier (often an email address) | Until person erasure; no automatic period (DAT 9 recommends departure plus 90 days, DEC-020 open) | Not exported | Person erasure: replaced by `erased-<person id>` |
| `persons.password_hash` | credential | Verify the password (Argon2id hash) | Until changed or person erasure | Never | Person erasure: replaced by an unusable hash |
| `sessions.id` | credential | SHA-256 of the session cookie value | Purged when expired, idle past `session-idle`, or 24 hours after revocation | Never | Person erasure: all sessions revoked, then purged |
| `sessions.last_seen_at` | activity | Idle timeout of the session | As `sessions.id` | Never | As `sessions.id` |
| `person_activity.last_seen_at` | activity | Inactivity rule (GOV-012): newest sign-in time copied from expiring sessions | Until the person row goes; no automatic period | Never | Not cleared by erasure (gap, see [Findings](#findings)) |
| `person_prefs.time_zone` | profile | Show times in the person's zone | Until changed | Not exported | Not cleared by erasure (gap, see [Findings](#findings)) |
| `person_mfa.secret` | credential | TOTP seed (SEC-007), sealed with AES-GCM when a secret key is configured | Until MFA is disabled or reset | Never | Person erasure: row deleted |
| `recovery_codes.code_hash` | credential | Single-use MFA recovery codes, salted SHA-256 | Until used, regenerated or MFA disabled | Never | Person erasure: deleted |
| `account_recovery_codes.code_hash` | credential | Single-use account recovery codes (SEC-008), salted SHA-256 | Until used or regenerated | Never | Person erasure: deleted |
| `mfa_challenges.id_hash` | credential | SHA-256 of the short-lived second-step login token | Expires in minutes; dead rows deleted at the person's next sign-in, on recovery and on MFA reset | Never | Not deleted by erasure; sessions cannot follow because the login is erased |
| `api_tokens.name` | free text | Label the person gave the token | Until person erasure; revoked tokens stay as metadata (DAT 9 recommends 1 year) | Not exported | Person erasure: renamed `erased` |
| `api_tokens.token_hash` | credential | Verify a bearer API token; the value is shown once | Until revoked or expired; then kept as metadata | Never | Person erasure: token revoked |
| `api_tokens.last_used_at` | activity | Show when a token was last used | As `api_tokens.name` | Not exported | Kept (no longer linked to a login after person erasure) |
| `security_notices.detail` | free text | Security notice to the person (new device, MFA change; can name a token) | Until person erasure | Not exported | Person erasure: deleted |
| `login_throttle.login` | identifier | Count failed sign-ins per typed login (SEC-006) | Deleted 24 hours after the lock or last failure | Never | Person erasure: rows for the login deleted |
| `invitations.token_hash` | credential | Verify an invitation link; not linked to a person until used | Until the network goes | Never | Kept (holds no personal data once used) |
| `members.display_name` | identifier | Name shown to other members (FR-016) | Until erasure; no automatic period (DAT 9 recommends departure plus 90 days, DEC-020 open) | Network export; self-export (own name, counterparties' names on own transactions) | Replaced by `Former member <id suffix>` |
| `members.default_flag` | governance marker | Network's "default" marker on a member (DOM-025) | Lifetime of the network | Network export | Kept (pseudonymous after erasure) |
| `listings.title` | free text | Offer or request title | Until the member edits or removes it, or erasure | Network export; self-export (own listings) | Replaced by `Erased listing` |
| `listings.description` | free text | Offer or request text | As `listings.title` | As `listings.title` | Emptied |
| `transactions.note` | free text | Memo of a trade, written by the proposer | Lifetime of the network | Network export; self-export (own transactions) | Emptied on transactions the member proposed |
| `transactions.reason` | free text | Mandatory reason of an adjustment or reversal, written by an administrator | Lifetime of the network | Network export; self-export (own transactions) | Kept: an administrator's record; reasons must not name people |
| `pending_adjustments.reason` | free text | Reason of a dual-control adjustment proposal | Lifetime of the network | Not exported | Kept: an administrator's record |
| `disputes.statement` | free text | The opener's statement | Lifetime of the network | Network export (not in self-export) | Emptied on disputes the member opened |
| `disputes.rationale` | free text | The reviewer's decision rationale | Lifetime of the network | Network export | Kept: the reviewer's record |
| `dispute_responses.body` | free text | The counterparty's response to a dispute | Lifetime of the network | Not exported | Emptied for responses the member wrote |
| `dispute_rereviews.reason` | free text | Reason for a re-review request | Lifetime of the network | Not exported | Emptied for requests the member made |
| `dispute_rereviews.rationale` | free text | Re-review decision rationale | Lifetime of the network | Not exported | Kept: the reviewer's record |
| `reports.note` | free text | Abuse report written by a member | Lifetime of the network | Not exported | Emptied for reports the member filed |
| `reports.handling_note` | free text | Administrator's handling note | Lifetime of the network | Not exported | Kept: an administrator's record |
| `challenges.reason` | free text | A member's challenge of a governance decision (GOV-016) | Lifetime of the network | Not exported | Kept (gap, see [Findings](#findings)) |
| `challenges.rationale` | free text | Decision on the challenge | Lifetime of the network | Not exported | Kept: the decider's record |
| `notifications.body` | free text | Inbox text; can contain display names | Read notifications deleted after 90 days; unread kept | Not exported | Emptied when it names the member, and the member's own inbox |
| `announcements.title` | free text | Administrator announcement title | Lifetime of the network (append-only) | Not exported | Kept: an administrator's record |
| `announcements.body` | free text | Administrator announcement text | Lifetime of the network (append-only) | Not exported | Kept: an administrator's record |
| `network_terms.body` | free text | The network's published rules and terms (GOV-009) | Lifetime of the network (versioned) | Not exported | Kept: a published network record |
| `network_policies.reason` | free text | Reason of a policy change | Lifetime of the network (append-only) | Network export | Kept: an administrator's record |
| `network_policies.notice_reason` | free text | Reason for a shortened notice period | Lifetime of the network | Not exported | Kept: an administrator's record |
| `network_closures.reason` | free text | Reason of a network closure | Lifetime of the record | Not exported | Kept |
| `network_suspensions.reason` | free text | Reason of a suspension (GOV-023, GOV-024) | Until the suspension is lifted (row deleted; the audit log keeps the event) | Not exported | Kept while the suspension lasts |
| `network_suspensions.suspended_by` | identifier | Name the platform operator gave for a platform suspension | As `network_suspensions.reason` | Not exported | Not applicable: operator staff, not a member |
| `break_glass_sessions.operator_label` | identifier | Operator staff member who opened a break-glass session (SEC-013) | Lifetime of the network (append-only) | Not exported; the audit trail shows the access to the network's administrators | Not applicable: operator staff, not a member |
| `break_glass_sessions.reason` | free text | Documented reason of the break-glass access | Lifetime of the network (append-only) | Not exported; the audit trail shows the access to the network's administrators | Kept: an operator record |
| `break_glass_sessions.closed_reason` | free text | Why the session ended | As `break_glass_sessions.reason` | Not exported | Kept: an operator record |
| `audit_events.reason` | free text | Reason recorded with an audited action | Lifetime of the network; hash-chained, never changed | Network export | Kept: changing it would break the audit chain ([erasure.md](../dev/erasure.md#the-audit-chain)) |

## Other columns

These columns hold no personal data of their own: identifiers, references to members, persons, accounts and
transactions, statuses, amounts, hashes and timestamps. Ledger and audit records stay after erasure with the member
reference unchanged; once the member's name and login are erased the reference is pseudonymous (SEC-018, LED-2).
`audit_events.summary_before` and `summary_after` are structured values; the code never writes a name, login or contact
detail into them.

| Table | Columns |
|---|---|
| `account_recovery_codes` | `id`, `person_id`, `created_at`, `used_at` |
| `accounts` | `id`, `network_id`, `member_id`, `is_system`, `balance`, `limit_override`, `limit_override_until` |
| `announcements` | `id`, `network_id`, `created_by`, `published_at`, `effective_at`, `deleted_at`, `deleted_by`, `supersedes_id` |
| `api_tokens` | `id`, `person_id`, `network_id`, `scopes`, `created_at`, `expires_at`, `revoked_at` |
| `audit_events` | `id`, `network_id`, `seq`, `occurred_at`, `actor_type`, `actor_id`, `action`, `target_type`, `target_id`, `request_id`, `idem_ref`, `summary_before`, `summary_after`, `prev_hash`, `hash` |
| `audit_heads` | `network_id`, `seq`, `hash` |
| `audit_subjects` | `audit_id`, `network_id`, `member_id` |
| `break_glass_grants` | `id`, `session_id`, `network_id`, `member_id`, `role`, `granted_at` |
| `break_glass_sessions` | `id`, `network_id`, `opened_at`, `expires_at`, `closed_at` |
| `challenges` | `id`, `network_id`, `member_id`, `kind`, `subject_id`, `involved_id`, `status`, `outcome`, `decided_by`, `created_at`, `decided_at` |
| `consistency_runs` | `id`, `network_id`, `started_at`, `finished_at`, `ok`, `findings` |
| `dispute_deadline_notices` | `network_id`, `dispute_id`, `kind`, `notified_at` |
| `dispute_events` | `id`, `network_id`, `dispute_id`, `actor_id`, `action`, `outcome`, `occurred_at` |
| `dispute_rereviews` | `id`, `network_id`, `dispute_id`, `requested_by`, `requested_at`, `status`, `reviewer_id`, `outcome`, `reversal_amount`, `decided_at` |
| `dispute_responses` | `id`, `network_id`, `dispute_id`, `responder_id`, `submitted_at` |
| `disputes` | `id`, `network_id`, `tx_id`, `opened_by`, `category`, `status`, `outcome`, `reversal_amount`, `reviewer_id`, `opened_at`, `decided_at` |
| `entries` | `id`, `network_id`, `tx_id`, `account_id`, `amount`, `ledger_seq`, `posted_at` |
| `idempotency_keys` | `network_id`, `actor_id`, `operation`, `key`, `fingerprint`, `result_ref`, `created_at`, `expires_at` |
| `invitations` | `id`, `network_id`, `created_by`, `created_at`, `expires_at`, `used_at`, `used_by`, `revoked_at` |
| `listings` | `id`, `network_id`, `member_id`, `type`, `price`, `status`, `created_at`, `updated_at` |
| `login_throttle` | `failures`, `locked_until` |
| `member_erasures` | `member_id`, `network_id`, `person_id`, `erased_at`, `person_erased` |
| `members` | `id`, `network_id`, `person_id`, `status`, `join_method`, `joined_at`, `departed_at` |
| `mfa_challenges` | `person_id`, `created_at`, `expires_at`, `used_at` |
| `network_closures` | `network_id`, `scheduled_by`, `scheduled_at`, `closing_at`, `notice_days`, `read_only_days`, `closed_at`, `final_ledger_seq`, `final_sha256`, `final_balance_total`, `final_entries_sum`, `erased_at` |
| `network_policies` | `network_id`, `version`, `policy_json`, `changed_by`, `effective_at`, `announced_at`, `notice_days` |
| `network_suspensions` | `network_id`, `kind`, `category`, `suspended_at` |
| `network_terms` | `network_id`, `version`, `published_by`, `published_at` |
| `networks` | `id`, `name`, `slug`, `status`, `ledger_seq`, `created_at` |
| `notifications` | `id`, `network_id`, `member_id`, `kind`, `title`, `link`, `dedupe_key`, `created_at`, `read_at` |
| `pending_adjustments` | `id`, `network_id`, `proposer_member_id`, `target_member_id`, `amount`, `direction`, `status`, `created_at`, `expires_at`, `decided_by`, `decided_at`, `tx_id` |
| `person_activity` | `person_id` |
| `person_mfa` | `person_id`, `enabled`, `last_step`, `created_at`, `enabled_at` |
| `person_prefs` | `person_id`, `updated_at` |
| `persons` | `id`, `created_at`, `must_change_password` |
| `recovery_codes` | `person_id`, `used_at` |
| `reports` | `id`, `network_id`, `reporter_id`, `target_type`, `target_id`, `category`, `status`, `created_at`, `handled_by`, `handled_at` |
| `role_assignments` | `id`, `network_id`, `member_id`, `role`, `granted_at`, `granted_by`, `revoked_at` |
| `schema_migrations` | `version`, `name`, `applied_at` |
| `security_notices` | `id`, `person_id`, `kind`, `created_at`, `seen_at` |
| `sessions` | `person_id`, `created_at`, `expires_at`, `revoked_at` |
| `terms_acceptances` | `network_id`, `member_id`, `version`, `accepted_at` |
| `transaction_status_events` | `id`, `network_id`, `tx_id`, `from_status`, `to_status`, `actor_id`, `request_id`, `occurred_at` |
| `transactions` | `id`, `network_id`, `kind`, `status`, `payer_account_id`, `payee_account_id`, `amount`, `listing_id`, `original_tx_id`, `created_by`, `accepted_by`, `reversed_total`, `proposed_at`, `expires_at`, `completed_at`, `ledger_seq` |

## Outside the database

| Store | Personal data | Retention | Notes |
|---|---|---|---|
| Application logs | None by design: request ids, route templates, status, durations (SEC-020) | `log-retention`, default 90 days (SEC-021, DEC-019 open) | Written through `internal/logx`, which redacts credentials, masks email addresses and escapes control characters |
| Backups | Everything in the database at backup time | Backup retention (SEC-028, OPS-019) | Erasure does not reach earlier backups |
| Export artefacts | As listed in the Export column | Download expiry (SEC-017) | Formats in [07](../specs/07-api-contract.md) |
| Audit mirror file (`audit-mirror-file`) | As `audit_events` | Operator's file retention | Optional |

## Findings

Found while building this inventory; recorded for the owner, not changed here.

1. `person_prefs.time_zone` and `person_activity.last_seen_at` are not cleared by person erasure
   (`erasePersonIfUnused` in `internal/service/erasure.go`). Neither identifies the person on its own once the login is
   erased, but they stay linked to the person id.
2. `challenges.reason`, written by the member who opened a governance challenge, is not emptied by erasure, unlike the
   other member-written free text.
3. No retention period is applied automatically to member or person data (SEC-018 addendum, open point 5; DEC-020).
