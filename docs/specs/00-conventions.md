# 00 Conventions, Glossary and Shared Decisions

Status: Draft v0.2 · Owner: Product architect · Last updated: 2026-10-24 · Scope: all specs · Depends on: none · Open questions: see 13-assumptions-risks-decisions.md





## Historical build status snapshot

The table below records the historical implementation snapshot at HEAD `925f637` on branch `claude/circula-build`; statuses were first read at `31b1015` and re-checked at `925f637` (see [TRACEABILITY.md](../TRACEABILITY.md)). It is not a status assessment of the current branch. **built** = behaviour exists in that snapshot; **partial** = exists with the listed gaps, or required verification was not run; **not built**; **deferred** = out of MVP by the spec or a recorded decision. Code paths and test names are in [TRACEABILITY.md](../TRACEABILITY.md). Current checkpoint evidence is tracked in [RESUME-2026-10-24](../audit/RESUME-2026-10-24.md). Neither historical nor current status is a compliance claim; no release gate is implied.

Summary: built 7, decided 2, open 2.

| ID | Priority | Status | Note |
|---|---|---|---|
| LED-1 | P0 | built | Two postings per trade in one unit; DB trigger; network sum checked by verify |
| LED-2 | P0 | built | Append-only triggers on entries, audit, status/dispute events, policies, runs; cache row unguarded (CODE-AUDIT LED-2) |
| LED-3 | P0 | built | AddBalance only inside postTx; VerifyLedger compares cache with sum |
| LED-4 | P0 | built | One store.Update per command; idempotent() writes key in same unit |
| LED-5 | P0 | built | checkCapacity inside write unit; service-level only, no DB constraint |
| LED-6 | P0 | built | Audit written in every command unit; login and session events not audited |
| LED-7 | P0 | built | int64 minor units; AddChecked overflow guard; MaxPolicyAmount; DB CHECK amount>0 |
| DEC-001 | - | decided, built | Single Go binary; embedded SQLite (pure-Go driver) behind store; PostgreSQL not built |
| DEC-002 | - | decided, built | Server-rendered HTML and /api/v1 both call service layer |
| DEC-003 | - | open, partly built | CI builds the five targets via package-release.sh; smoke run on each target not performed |
| DEC-004 | - | open, not built | Licence undecided; no LICENSE file in repo (see 13 register) |

## 1. Normative terms
**MUST** = absolute requirement. **SHOULD** = recommended; deviation needs a recorded reason. **MAY** = optional. Lowercase "must/should/may" carry no normative weight.

## 2. Requirement IDs
`<PREFIX>-<NNN>`, three digits, never reused. Existing identifiers with a trailing letter (currently `API-062a` and `API-067a` in [07](07-api-contract.md)) are legacy identifiers retained for stable references; this exception does not authorize renumbering or creating new suffixed IDs. Prefixes and home documents:

| Prefix | Meaning | Home |
|---|---|---|
| VIS | Vision/scope | 01-vision-scope.md |
| PER | Personas/journeys | 02-personas-journeys.md |
| DOM | Domain/ledger rules | 03-domain-model.md |
| FR | Functional requirements | 04-functional-requirements.md |
| ARC | Architecture | 05-architecture.md |
| DAT | Data/ledger/audit | 06-data-ledger-audit.md |
| API | API contract | 07-api-contract.md |
| UX | UX/accessibility | 08-ux-accessibility.md |
| SEC | Security/privacy | 09-security-privacy.md |
| GOV | Governance/operations | 10-governance-operations.md |
| OPS | Deployment/operations | 11-deployment-operations.md |
| QA | Quality/test plan | 12-quality-test-plan.md |
| DEC / RSK / ASM | Decisions / risks / assumptions | 13-assumptions-risks-decisions.md |

Each requirement states: ID, priority (P0 = MVP-blocking, P1 = MVP, P2 = post-MVP), rationale, testable acceptance criteria. One authoritative home per rule; others link by ID.

## 3. Glossary (authoritative)
- **Network**: an independent community running its own rules. Tenant boundary: no data crosses networks.
- **Member**: a person with membership in one network. **Account**: the ledger account of a member (one per member per network). A person in two networks has two members and two accounts.
- **Credit** (unit): accounting unit of a network. NOT a cryptocurrency, security, or redeemable for national currency. Stored as integer **minor units** (default 2 decimals => 1.00 credit = 100 units).
- **Balance**: sum of posted entries on an account (integer, may be negative). **Credit limit**: max negative balance allowed (>= 0, stored as positive magnitude). **Available credit** = balance + credit limit − reserved outgoing pending (for payer view).
- **Offer / Request**: listing of goods/services given / wanted.
- **Transaction**: proposal that payer pays payee amount. Persisted statuses are `proposed`, `completed` (posted), `declined`, `cancelled`, `expired`, `disputed`, and `reversed`. `accepted` means the counterparty's consent has been received; in the MVP it is a transient step in the same atomic operation that posts and changes the persisted status to `completed`, not a persisted transaction status. The authoritative transition rules are in [03](03-domain-model.md#5-transaction-status-state-machine); the UI does not present `accepted` as a separate status ([08](08-ux-accessibility.md)).
- **Posting / Entry**: immutable ledger line (account, signed amount, transaction id). A completed transaction MUST have exactly two entries summing to zero.
- **Adjustment**: administrator-created transaction with kind `adjustment`, reason mandatory, audited, itself posts equal-and-opposite entries against the network's system account.
- **Reversal**: new transaction with kind `reversal` referencing the original; originals are never edited.
- **Dispute**: case raised against a transaction; reviewed by Dispute reviewer.
- **Audit event**: append-only record of a state change (who, what, when, target, before/after summary, request id).
- **Roles**: Member, Network administrator, Dispute reviewer, Platform operator.
- **System account**: one per network, holds adjustment counter-entries and sign-up grants; balance may be any sign; invariant: sum of all account balances in a network = 0.

## 4. Platform-wide invariants (not configurable)
1. LED-1: Sum of all entries per transaction = 0; sum of all balances per network = 0.
2. LED-2: Entries are append-only; no UPDATE/DELETE on entries; corrections are new transactions.
3. LED-3: Balance is derivable by summing entries; cached balance MUST equal the sum (verified by a check command).
4. LED-4: Posting both sides is atomic; retry with same idempotency key returns the original result.
5. LED-5: A posting MUST NOT worsen a non-system account's position beyond its effective credit limit: a debit MUST be refused if it would take the account below `−credit limit`, including when the account is already below a newly lowered limit. An incoming credit that improves an already over-limit account is permitted; it does not authorize any further debit. The system account has no credit limit (DOM-026). Check capacity inside the same atomic unit as posting and serialize per account.
6. LED-6: Every state change writes an audit event in the same atomic unit.
7. LED-7: Amounts are integers > 0 for trades; no floating point anywhere in money handling.

## 5. Configurable network policies (defaults)
Default credit limit (default 0 for new members; recommended starting value set by admin), per-member override, max transaction amount, proposal expiry (default 7 days), decimals (default 2), member directory visibility (members-only), join mode (invite | approval | open), dispute window (default 30 days), departure rule (see 03).

## 6. Shared technical decisions (recommended defaults, see DEC log)
- DEC-001: Single self-hostable binary + optional hosted deployment of the same binary. Implementation language **Go**; the module's required Go version follows the `go` directive in [`go.mod`](../../go.mod) (currently `1.26.10`). Use stdlib `net/http`, embedded **SQLite** (WAL, pure-Go driver, no cgo) as default store; storage behind an interface so PostgreSQL MAY be added later.
- DEC-002: Server-rendered HTML (progressive enhancement, no JS required for core flows) + versioned JSON API `/api/v1`. Both call the same domain service layer.
- DEC-003: Release targets: windows/amd64, linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, plus SHA-256 checksums.
- DEC-004: Product license undecided (owner decision); repo ships without LICENSE until decided; recommendation: AGPL-3.0 or MIT.
- Legal/tax/money-services/privacy compliance: unresolved, jurisdiction-specific, for qualified counsel. No spec asserts compliance.
