# Export package schema, version 1

Specification: DAT-037 to DAT-041. **Schema version: 1.** `schema_version` in `manifest.json` (and in the JSON document
header) is this number. A change that renames, removes or re-types a field raises it; adding a column at the end of a
table does not, but is listed in the change log below. The test `TestExportSchemaDocMatches` fails when this document
and the code disagree.

## Package layout

An export is a `.zip` (or an unpacked directory) holding:

| File | Content |
|---|---|
| `manifest.json` | the manifest, below |
| `<table>.csv` | CSV format: one file per table |
| `<scope>.json` | JSON format: one structured document holding all tables of the scope |

CSV rules: UTF-8, RFC 4180 quoting, one header row, `\n` or `\r\n` line ends. Text cells that start with `=`, `+`, `-`,
`@`, tab or carriage return get a leading apostrophe so a spreadsheet will not run them (SEC-017); plain numbers are
never altered. Empty cell means null. Times are RFC 3339 UTC. Money columns are integers in the minor unit of the
network currency; each also has a `<column>_formatted` twin as a decimal string using the manifest's `decimals`.

## manifest.json

| Field | Type | Meaning |
|---|---|---|
| `schema_version` | integer | this document's version (1) |
| `network_id` | string | the network exported |
| `scope` | string | `mine`, `ledger`, `members`, `audit` or `full` |
| `format` | string | `csv` or `json` |
| `decimals` | integer | decimals of the network currency, for the `_formatted` twins |
| `ledger_seq` | integer | the one ledger sequence every file was read at (DAT-039) |
| `generated_at` | string | RFC 3339 UTC |
| `files` | array | per file: `name`, `rows` (data rows, header excluded), `bytes`, `sha256` (hex) |

For the JSON format `rows` is the total number of table rows in the document. Verification (DAT-040) recomputes `bytes`,
`sha256` and `rows` for every listed file; any mismatch fails.

## JSON document

```json
{"schema_version":1,"network_id":"...","scope":"full","decimals":2,"ledger_seq":42,"generated_at":"...",
 "tables":{"members":[{"id":"...","display_name":"..."}],"entries":[...]}}
```

Each table is an array of objects whose keys are the column names below, in that order. Null is `null`.

## Tables and scopes

| Scope | Tables |
|---|---|
| `mine` | transactions, entries (the caller's own) |
| `ledger` | accounts, transactions, transaction_status_events, entries |
| `members` | members, role_assignments |
| `audit` | audit_events |
| `full` | members, role_assignments, accounts, listings, transactions, transaction_status_events, entries, disputes, network_policies, audit_events |

## Columns

Each heading is the table name; the line after it is the exact column list in file order. Credentials, tokens and
recovery material are never exported.

### members
`id,display_name,status,join_method,default_flag,joined_at,departed_at,limit_override,limit_override_formatted,limit_override_until`

### role_assignments
`id,member_id,role,granted_at,granted_by,revoked_at`

### accounts
`id,member_id,is_system,balance,balance_formatted,limit_override,limit_override_formatted,limit_override_until`

### listings
`id,member_id,type,title,description,price,price_formatted,status,created_at,updated_at`

### transactions
`id,kind,status,payer_account_id,payee_account_id,amount,amount_formatted,listing_id,note,reason,original_tx_id,created_by,accepted_by,reversed_total,reversed_total_formatted,proposed_at,expires_at,completed_at,ledger_seq`

### transaction_status_events
`id,tx_id,from_status,to_status,actor_id,request_id,occurred_at`

### entries
`id,tx_id,account_id,amount,amount_formatted,ledger_seq,posted_at`

### disputes
`id,tx_id,opened_by,category,statement,status,outcome,reversal_amount,reversal_amount_formatted,reviewer_id,rationale,opened_at,decided_at`

### network_policies
`version,policy_json,changed_by,reason,effective_at`

### audit_events
`id,seq,occurred_at,actor_type,actor_id,action,target_type,target_id,request_id,idem_ref,summary_before,summary_after,reason,prev_hash,hash`

## Reference reader

`scripts/read-export.py` is a reference reader. It needs only Python 3 and its standard library, not Circula. It reads a
`.zip` or a directory, verifies size, SHA-256 and row count of every file against the manifest and prints a summary.
Exit status 0 means the package is intact.

```sh
python3 scripts/read-export.py full-export.zip
```

The product's own `circula verify-export` performs the same integrity check.

## Change log

| Version | Change |
|---|---|
| 1 | first documented version |
