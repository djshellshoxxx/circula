# Erasure of a departed member (SEC-018)

A departed member's personal data can be removed while the ledger stays exactly as it is. Decisions and open points:
[`docs/specs/addenda/SEC-018.md`](../specs/addenda/SEC-018.md).

## Who can do it

| Route | Who | Reason |
|---|---|---|
| Web `/n/{slug}/admin/members/{id}/erase` (link on the departed members list) | network administrator | mandatory |
| Web `/n/{slug}/erase-my-data` | the departed member, for themselves | optional |
| `circula erase-member -network SLUG -member ID -reason TEXT -confirm` | operator (stop `serve` first) | mandatory |
| Service `AnonymiseMember` / `EraseMyData` | the above | |

Preconditions: the member is `departed`, was not anonymised before, and nothing is in flight (no proposed or disputed
transaction, no unresolved dispute). Otherwise the call is refused and nothing changes; the refusal lists the blockers.
The action cannot be undone.

## What changes

| Data | After |
|---|---|
| `members.display_name` | `Former member <last 8 characters of the member id>` |
| the member's listings | archived, title `Erased listing`, description empty |
| `transactions.note` of transactions the member proposed | empty |
| `disputes.statement` of disputes the member opened | empty |
| person login | `erased-<person id>`, only if the person has no other non-departed membership |
| person password hash, sessions, API tokens | unusable hash, all revoked, tokens renamed (same condition) |

Unchanged: all ids, accounts, balances, entries, transaction rows and amounts, status events, dispute rows (category,
outcome, rationale), `audit_events` and their hashes, `audit_subjects`. `VerifyLedger` passes after erasure.
`member_erasures` records that it happened (no personal data).

## The audit chain

Audit events are append-only and hashed over their content including `reason`. The code never puts a name, login or
contact detail into an audit summary, so a name appears in the audit log only if a person typed it into a free-text
reason (a departure reason, an adjustment reason). Those events are **not** altered, because that would break the chain.
Tell administrators not to write personal data into reasons (the erase forms say so). Erasing adds one event,
`member.anonymised`, with the counts of what was cleared.

## Exports

Exports read the live tables, so exports after erasure show `Former member ...`. Backups taken earlier keep the old data
until backup retention ends (SEC-028).

## Adding personal data to the model

If you add a column that holds a member's or person's personal text (bio, contact, profile, MFA secret, recovery
contact), clear it in `ErasureRepo` / `erasePersonIfUnused` (`internal/service/erasure.go`) and extend
`TestAnonymiseMemberKeepsLedger`: the test scans every column of every table for the original values.
