# Service layer reference

Package `github.com/djshellshoxxx/circula/internal/service` is the **only write path** to Circula state (ARC-002).
HTTP (`internal/api`), HTML (`internal/web`) and CLI (`cmd/circula`) adapters translate requests into calls on
`*service.Service` and domain errors into responses; they hold no business rules (ARC-003). Specs: `docs/specs/`
(03 domain, 06 ledger, 09 security). Audit findings addressed here: `docs/audit/CODE-AUDIT.md`, `SECURITY-AUDIT.md`
(IDs such as SVC-01, SA-08 are used in code comments and tests).

Status: service and store are complete and tested; the adapters above are built on top of this surface.

## 1. Wiring

```go
st, err := store.Open(ctx, "/var/lib/circula/circula.db") // WAL, FKs on, forward-only embedded migrations
svc, err := service.New(st, service.Config{})            // defaults: wall clock, UUIDv7, Argon2id 64 MiB/t=3/p=2
```

* One `*Service` per process; safe for concurrent use. **One writer process per database file** (ARC-011). Writers are
  serialised in-process by a gate and across processes by `BEGIN IMMEDIATE`. `store.Open` uses a 5 s busy timeout
  (`store.OpenWith(ctx, path, store.Options{BusyTimeout: d})` to change it).
* `Store.View` is truly read-only: the connection runs `PRAGMA query_only=ON` for the unit, so a write inside a view
  fails and cannot bypass the write gate (SVC-02). Everything that must not write (reads, `VerifyLedger`) uses it.
* `Config.Clock` and `Config.IDs` are injected (DAT-002, DOM section 12); tests use `internal/testutil`.
* `Config.Argon` cost parameters are a placeholder until DEC-006 (benchmark on minimum hardware) is decided.

## 2. Conventions every adapter must follow

**Actor.** Every call takes `ctx` and a `domain.Actor`. The adapter authenticates and builds the actor:

| Credential | How | Actor |
|---|---|---|
| Browser session | `Login` -> opaque token (cookie) -> `Authenticate(token)` | person, scopes read+write |
| API token | `CreateAPIToken` -> `cir_...` -> `AuthenticateToken(token)` | person, token scopes, optional `NetworkScope` |
| Background job / CLI | `domain.SystemActor(requestID)` | `System: true`, no member |

Set `Actor.RequestID` from the edge request id (`X-Request-Id`); it lands in audit events and status history.
The service resolves the member and roles *per network on every call*; adapters never pass roles. Read methods need
scope `read`, mutating ones `write` (a read-only token gets `forbidden`). A `NetworkScope`d token is `not-found` outside its network.

**Network scoping.** Every network-scoped method takes `netID`. A caller who is not a member of `netID`, a foreign
object id and a missing id all return the same `not-found` (API-017, FR-006).

**Money.** `int64` minor units everywhere (`domain.ParseAmount`/`FormatAmount` for the network's `Policy.Decimals`). No floats.

**Idempotency.** Commands in the retry table of 06 section 6.1 (propose, accept, decline, cancel, reverse, adjust, open/resolve
dispute, create listing, update policy) require a non-empty `IdempotencyKey` (map the `Idempotency-Key` header; else
`idempotency-key-required`). Scope: network + person + operation. Same key + same payload returns the original result;
same key + different payload is `idempotency-key-reused`. Failed commands are not recorded, so the key stays reusable (API-025).
Records expire after 7 days (`Config.IdempotencyTTL`). Input structs carry the key as `IdempotencyKey` (excluded from the fingerprint).

**Errors.** Expected failures are `*domain.Error{Code, Msg, Fields, Blockers}`; use `errors.Is(err, domain.ErrForbidden)` or
`domain.CodeOf(err)`. Anything else is an internal error: log it, return `internal-error`, never leak the text.

| Code (`domain.Code...`) | HTTP | Meaning |
|---|---|---|
| `validation-failed` | 422 | `Fields` lists `{field, detail}` |
| `insufficient-available-credit` | 422 | LED-5 / available credit |
| `amount-exceeds-policy` | 422 | above `MaxTxAmount` |
| `counterparty-not-active` | 422 | counterparty or party not active |
| `idempotency-key-reused` | 422 | key reused with a different payload |
| `idempotency-key-required` | 400 | empty key on a command that needs one |
| `unauthenticated` | 401 | bad credentials / session / token |
| `forbidden` | 403 | role, membership state, scope or conflict of interest |
| `not-found` | 404 | missing, foreign network, or not visible |
| `invalid-transaction-state` | 409 | state machine refusal (also used for dispute/membership/network state) |
| `conflict` | 409 | duplicate (e.g. login unavailable) |
| `blocked` | 409 | `Blockers` lists what prevents departure, closure or last-admin removal |
| `rate-limited` | 429 | login throttle (per login; add per-address limits at the edge); open-proposal caps; export quota |
| `unavailable` | 503 | database busy past the busy timeout (SQLITE_BUSY/LOCKED). **Retryable**: nothing was written. Send `Retry-After`; `domain.IsRetryable(err)` |
| `internal-error` | 500 | not produced by the service; adapters map unknown errors to it |

**Pagination.** Keyset: `Page{AfterID, Limit}` (default 100, max 500). Lists are ordered by id (UUIDv7, time-ordered),
newest first for transactions, listings, disputes; oldest first for members; audit uses `AuditQuery.AfterSeq`;
`ListEntries` pages by `ledger_seq` descending with `EntryQuery.BeforeSeq`. Adapters must page with the cursor rather
than scan a fixed window; use the single-object reads (`GetMember`, `GetListing`, `GetDispute`, `GetDisputeForTx`,
`GetTransaction`) to fetch one item (API-04).

**Never log credentials.** Invitation tokens (`inv_...`) travel in URL paths; pass any path, URL or error text through
`service.RedactSecrets` before logging, or log the route pattern (SA-13, API-07, SEC-020). The service itself logs nothing.

**Secrets.** Session tokens, API tokens and invitation tokens are returned once and stored only as SHA-256. Passwords are Argon2id PHC strings.

## 3. Method reference

All methods are on `*service.Service`. Doc comments in the source are authoritative; first sentences are repeated here.

#### accounts.go

```go
func (s *Service) GetBalance(ctx context.Context, actor domain.Actor, netID string) (*BalanceView, error)
// GetBalance returns the caller's own balance view.

func (s *Service) AdminGetMemberBalance(ctx context.Context, actor domain.Actor, netID, memberID string) (*BalanceView, error)
// AdminGetMemberBalance lets an administrator see another member's balance and limit. The read is audited (account.viewed), so it takes the write gate.

func (s *Service) PeekMemberBalance(ctx context.Context, actor domain.Actor, netID, memberID string) (*BalanceView, error)
// Non-audited admin read in a View, ONLY for ETags / preconditions (SVC-05). Never render it; displaying goes through AdminGetMemberBalance.

func (s *Service) BalanceAt(ctx context.Context, actor domain.Actor, netID string, seq int64) (int64, error)
// BalanceAt reconstructs the caller's balance at a ledger_seq by summing entries (DAT-021, FR-054).
```

#### adjustments.go

```go
func (s *Service) CreateAdjustment(ctx context.Context, actor domain.Actor, netID string, in AdjustInput) (*domain.Transaction, error)
// CreateAdjustment posts one entry on the member and the equal opposite entry on the system account, completed on creation (DOM-022, FR-062).
```

#### reads.go (single-object and paged reads; all run in `Store.View`)

```go
func (s *Service) GetMember(ctx, actor, netID, memberID string) (*domain.Member, error)
// Own record or administrator: full. Active member viewing another ACTIVE member: directory subset (ID, NetworkID, DisplayName, Status). Else not-found.

func (s *Service) GetListing(ctx, actor, netID, listingID string) (*domain.Listing, error)
// Active members: active listings of active owners, plus all of their own. Else not-found.

func (s *Service) GetDisputeForTx(ctx, actor, netID, txID string) (*DisputeView, error)
// Latest dispute on a transaction, to parties, reviewers, administrators. GetDispute (disputes_decide.go) is the by-id read; both are tenant-scoped.

func (s *Service) ListEntries(ctx, actor, netID string, q EntryQuery) ([]EntryRow, error)
// The caller's own entries, ledger_seq DESC (API-045). EntryQuery{BeforeSeq, Limit}: pass the last row's Entry.LedgerSeq as the next BeforeSeq.

func (s *Service) ListPendingForActor(ctx, actor, netID string, q PendingQuery) ([]*domain.Transaction, error)
// Open (proposed, unexpired) trades the caller is party to, keyset-paged, no fixed window. PendingQuery{AwaitingMe, Page}.
```

#### purge.go

```go
func (s *Service) Purge(ctx context.Context) (PurgeReport, error)
// Retention (SA-12, DATA-01): deletes dead sessions (expired, idle past SessionIdle, revoked >24 h ago), login_throttle rows whose lock/last failure ended >24 h ago, and expired idempotency records. Batches of 1000, one short write unit each. No actor: internal maintenance only, call it from the job runner (hourly). Never touches ledger, audit or membership data.

func RedactSecrets(s string) string
// Replaces inv_... and cir_... tokens with a marker (SA-13).
```

#### audit.go

```go
func AuditHash(e *domain.AuditEvent) string
// AuditHash computes the chain hash of an event over its content and the previous hash.
```

#### audit_read.go

```go
func (s *Service) ListAudit(ctx context.Context, actor domain.Actor, netID string, q AuditQuery) ([]*domain.AuditEvent, error)
// ListAudit reads the audit log; what the caller sees depends on role: admins see everything (the read is audited), reviewers see dispute events, members see only events about their own account in reduced form.
```

#### auth.go

```go
func (s *Service) RegisterPerson(ctx context.Context, in RegisterInput) (*domain.Person, error)
// RegisterPerson creates a Person with an Argon2id password hash. UNIFORM RESPONSE (SA-10): a taken login returns success with a decoy Person (ID matches no row), leaves the account untouched and audits person.register_duplicate. Adapters must not auto-login or branch on the outcome. RegisterInput.RevealDuplicate (CLI only) returns conflict instead.

func (s *Service) Login(ctx context.Context, login, password string) (string, *domain.Person, error)
// Login verifies credentials and returns an opaque bearer session token (shown once; only its SHA-256 is stored). Atomic throttle (AUTH-01/SA-04): lock check + attempt count are ONE write unit before the password is verified, so at most 5 guesses are evaluated per lock window however many run in parallel. Malformed logins (validLogin) are refused uniformly and never stored.

func (s *Service) Authenticate(ctx context.Context, token string) (domain.Actor, error)
// Authenticate resolves a session token to an Actor. Only an unknown/revoked/expired session is `unauthenticated`; store failures (including `unavailable`) are returned as they are, so an outage never logs users out (AUTH-02). Same for AuthenticateToken.

func (s *Service) Logout(ctx context.Context, token string) error
// Logout revokes the session behind token.

func (s *Service) ChangePassword(ctx context.Context, actor domain.Actor, oldPW, newPW string) error
// ChangePassword verifies the old password, sets the new one and revokes every session of the person, including the caller's (SEC-003).
```

#### departure.go

```go
func (s *Service) DepartMember(ctx context.Context, actor domain.Actor, netID string, in DepartInput) (*domain.Member, error)
// DepartMember ends a membership (DOM-024, FR-014, FR-015).
```

#### disputes.go

```go
func (s *Service) OpenDispute(ctx context.Context, actor domain.Actor, netID string, in OpenDisputeInput) (*domain.Dispute, error)
// OpenDispute lets a party to a completed trade contest it within the network's dispute window, measured from completion (DOM-017).

func (s *Service) StartDisputeReview(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error)
// StartDisputeReview takes an open dispute into review (open -> under_review).

func (s *Service) WithdrawDispute(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error)
// WithdrawDispute lets the opener withdraw before a decision (FR-075); the transaction returns to `completed` and no entries are created.
```

#### disputes_decide.go

```go
func (s *Service) ResolveDispute(ctx context.Context, actor domain.Actor, netID string, in ResolveInput) (*domain.Dispute, error)
// ResolveDispute records dismissed / upheld / partially_upheld with a mandatory rationale and the decider's identity (DOM-018).

func (s *Service) RetryDisputePosting(ctx context.Context, actor domain.Actor, netID, disputeID string) (*domain.Dispute, error)
// RetryDisputePosting re-attempts the reversal of a `posting_pending` dispute (DOM-021).

func (s *Service) GetDispute(ctx context.Context, actor domain.Actor, netID, disputeID string) (*DisputeView, error)
// GetDispute returns a dispute to its parties, reviewers and administrators; everyone else gets not-found.

func (s *Service) ListDisputes(ctx context.Context, actor domain.Actor, netID string, statuses []domain.DisputeStatus, pg Page) ([]*domain.Dispute, error)
// ListDisputes lists disputes: all for reviewers and administrators, own for members.
```

#### export.go

```go
func (s *Service) Export(ctx context.Context, actor domain.Actor, netID string, scope ExportScope, format ExportFormat, sink ExportSink) (*Manifest, error)
// Export streams an export package to sink and returns its manifest.
```

#### join.go

```go
func (s *Service) CreateInvitation(ctx context.Context, actor domain.Actor, netID string, in InviteInput) (string, *domain.Invitation, error)
// CreateInvitation returns the one-time token (shown once; only its hash is stored) and the invitation record (FR-010, FR-063).

func (s *Service) RevokeInvitation(ctx context.Context, actor domain.Actor, netID, invID string) error
// RevokeInvitation voids an unused invitation (FR-063).

func (s *Service) ListInvitations(ctx context.Context, actor domain.Actor, netID string) ([]*domain.Invitation, error)
// ListInvitations lists the network's invitations (admin).

func (s *Service) RedeemInvitation(ctx context.Context, actor domain.Actor, token, displayName string) (*domain.Member, error)
// RedeemInvitation joins the caller to the invitation's network as an active member. A pending/active member gets the existing membership back (token not consumed); suspended/departed -> forbidden (SVC-04).

func (s *Service) JoinNetwork(ctx context.Context, actor domain.Actor, netID, displayName string) (*domain.Member, error)
// JoinNetwork joins an approval-mode network (-> pending) or open-mode network (-> active). Same existing-membership rule as RedeemInvitation.

func (s *Service) DecideJoinRequest(ctx context.Context, actor domain.Actor, netID, memberID string, approve bool, reason string) (*domain.Member, error)
// DecideJoinRequest approves (-> active) or denies (-> departed) a pending member (FR-011).
```

#### listings.go

```go
func (s *Service) CreateListing(ctx context.Context, actor domain.Actor, netID string, in ListingInput) (*domain.Listing, error)
// CreateListing publishes an offer or request.

func (s *Service) UpdateListing(ctx context.Context, actor domain.Actor, netID string, in ListingUpdate) (*domain.Listing, error)
// UpdateListing edits, pauses or archives the caller's own listing (FR-022).

func (s *Service) ListListings(ctx context.Context, actor domain.Actor, netID string, q ListingQuery) ([]*domain.Listing, error)
// ListListings returns active listings to active members (FR-023, FR-025).
```

#### membership.go

```go
func (s *Service) ListDirectory(ctx context.Context, actor domain.Actor, netID string, pg Page) ([]DirectoryEntry, error)
// ListDirectory lists active members to active members and administrators.

func (s *Service) AdminListMembers(ctx context.Context, actor domain.Actor, netID string, statuses []domain.MemberStatus, pg Page) ([]*domain.Member, error)
// AdminListMembers lists members of any status with roles and flags (admin).

func (s *Service) SuspendMember(ctx context.Context, actor domain.Actor, netID, memberID, reason string) error
// SuspendMember freezes a member (-> suspended) with a mandatory reason.

func (s *Service) RestoreMember(ctx context.Context, actor domain.Actor, netID, memberID, reason string) error
// RestoreMember returns a suspended member to active.

func (s *Service) AssignRole(ctx context.Context, actor domain.Actor, netID, memberID string, role domain.Role) error
// AssignRole grants admin or reviewer to an active member (FR-065, FR-069).

func (s *Service) RevokeRole(ctx context.Context, actor domain.Actor, netID, memberID string, role domain.Role) error
// RevokeRole removes a role.

func (s *Service) SetDefaultFlag(ctx context.Context, actor domain.Actor, netID, memberID string, flag bool, reason string) error
// SetDefaultFlag sets or clears the governance "default" marker (DOM-025, FR-064).

func (s *Service) SetCreditLimit(ctx context.Context, actor domain.Actor, netID string, in SetLimitInput) error
// SetCreditLimit changes a member's override.

func (s *Service) SetCreditLimitView(ctx context.Context, actor domain.Actor, netID string, in SetLimitInput) (*BalanceView, error)
// SetCreditLimit that returns the position after the change from the same unit: one write, one limit.changed event, no account.viewed (SVC-05).
```

#### networks.go

```go
func (s *Service) CreateNetwork(ctx context.Context, actor domain.Actor, in CreateNetworkInput) (*domain.Network, *domain.Member, error)
// CreateNetwork creates a network in `setup` status with its system account (DOM-026), policy version 1, and the caller as the first active administrator (FR-001, FR-002).

func (s *Service) ActivateNetwork(ctx context.Context, actor domain.Actor, netID string) (*domain.Network, error)
// ActivateNetwork moves setup -> active once an active administrator exists (FR-004).

func (s *Service) CloseNetwork(ctx context.Context, actor domain.Actor, netID, reason string) error
// CloseNetwork closes a network (FR-068).

func (s *Service) GetNetwork(ctx context.Context, actor domain.Actor, netID string) (*domain.Network, error)
// GetNetwork returns the network for any of its members (any membership status).

func (s *Service) NetworkBySlug(ctx context.Context, slug string) (*NetworkInfo, error)
// NetworkBySlug resolves a slug for the join flow.

func (s *Service) MyMemberships(ctx context.Context, actor domain.Actor) ([]*domain.Member, error)
// MyMemberships lists every membership of the caller across networks.

func (s *Service) UpdatePolicy(ctx context.Context, actor domain.Actor, netID string, in PolicyChange) (*domain.Network, error)
// UpdatePolicy appends a new policy version (DOM-029, DOM-030).
```

#### password.go

```go
func DefaultArgon2Params() Argon2Params
// DefaultArgon2Params follows the OWASP minimum (19 MiB, t=2) with headroom.
```

#### reversal.go

```go
func (s *Service) ReverseTransaction(ctx context.Context, actor domain.Actor, netID string, in ReverseInput) (*domain.Transaction, error)
// ReverseTransaction posts a `reversal` transaction against a completed one (FR-066, DOM-036, DOM-019/020): the original payee is debited and the original payer credited.
```

#### service.go

```go
func New(st store.Store, cfg Config) (*Service, error)
// New builds a Service over st, filling defaults.
```

#### tokens.go

```go
func (s *Service) CreateAPIToken(ctx context.Context, actor domain.Actor, in TokenInput) (string, *domain.APIToken, error)
// CreateAPIToken returns the bearer value once; only its hash is stored.

func (s *Service) AuthenticateToken(ctx context.Context, token string) (domain.Actor, error)
// AuthenticateToken resolves an API token to an Actor carrying the token's scopes and optional network restriction.

func (s *Service) RevokeAPIToken(ctx context.Context, actor domain.Actor, tokenID string) error
// RevokeAPIToken revokes one of the caller's own tokens.

func (s *Service) ListAPITokens(ctx context.Context, actor domain.Actor) ([]*domain.APIToken, error)
// ListAPITokens lists the caller's tokens (never the secret values).
```

#### transactions.go

```go
func (s *Service) ProposeTransaction(ctx context.Context, actor domain.Actor, netID string, in ProposeInput) (*domain.Transaction, error)
// ProposeTransaction creates a `proposed` trade. Reservation follows consent: a payer's own proposal reserves their credit; a request (proposer is payee) reserves NOTHING against the payer (section 6, SA-08). Open-proposal caps apply.

func (s *Service) AcceptTransaction(ctx context.Context, actor domain.Actor, netID string, in TxRef) (*domain.Transaction, error)
// AcceptTransaction is consent by the party who did not propose. Re-checks both parties active, the payer's default flag when the policy blocks flagged payers (SVC-03), and the hard credit limit.

func (s *Service) DeclineTransaction(ctx context.Context, actor domain.Actor, netID string, in TxRef) (*domain.Transaction, error)
// DeclineTransaction is the counterparty's exit.

func (s *Service) CancelTransaction(ctx context.Context, actor domain.Actor, netID string, in CancelInput) (*domain.Transaction, error)
// CancelTransaction lets the proposer withdraw (also when suspended), or an administrator who is not a party cancel with a reason (DOM-010, DOM-037).

func (s *Service) ExpireProposals(ctx context.Context, actor domain.Actor, limit int) (int, error)
// ExpireProposals moves proposals whose expiry has elapsed to `expired`, releasing their reservations (DOM-013, FR-034).
```

#### tx_queries.go

```go
func (s *Service) GetTransaction(ctx context.Context, actor domain.Actor, netID, txID string) (*TxView, error)
// GetTransaction returns one transaction to a party, an administrator or a reviewer of a dispute on it.

func (s *Service) ListTransactions(ctx context.Context, actor domain.Actor, netID string, statuses []domain.TxStatus, pg Page) ([]*domain.Transaction, error)
// ListTransactions is the caller's own history, newest first, filterable by status.

func (s *Service) AdminListTransactions(ctx context.Context, actor domain.Actor, netID string, kind domain.TxKind, statuses []domain.TxStatus, pg Page) ([]*domain.Transaction, error)
// AdminListTransactions lists every transaction of the network (administrators).
```

#### verify.go

```go
func (s *Service) VerifyLedger(ctx context.Context, actor domain.Actor, netID string) (*LedgerReport, error)
// READ-ONLY (OPS-016, SVC-01): runs in one Store.View snapshot, takes no write lock, writes no consistency run and no audit event; the database is unchanged afterwards. Also checks that the audit chain ends at the recorded head (SA-09).

func (s *Service) RecordConsistencyRun(ctx context.Context, actor domain.Actor, rep *LedgerReport) error
// Appends rep (from VerifyLedger) to consistency_runs and writes ledger.verified in one short write unit. Admin/System, write scope. Sets rep.RunID.

func (s *Service) VerifyAndRecordLedger(ctx context.Context, actor domain.Actor, netID string) (*LedgerReport, error)
// VerifyLedger then RecordConsistencyRun. Use for an explicit administrator-initiated check only; startup, migrate, restore, backup verification and hourly jobs use plain VerifyLedger.
```



#### overview.go, announcements.go, timezone.go, disputes_list.go

These use optional store surfaces (`store.OverviewRepo`, `AnnouncementRepo`, `PrefsRepo`, in `iface_f2.go`) that the SQLite transaction implements; the service asserts for them, so `store.Tx` did not change. Schema is in migration 0004.

* `AdminOverview(ctx, actor, netID) (*AdminOverview, error)`: administrator only. Exact counts read at call time (FR-067): pending members, live proposals older than `OverviewStaleAfter` (48 h), open or under-review disputes, `posting_pending` disputes, active members whose credit used is at or beyond their effective limit, unused invitations ending within `OverviewInviteWithin` (72 h), and the latest recorded consistency check. `AsOf` is the read time.
* `CreateAnnouncement` (admin, idempotent, audited), `DeleteAnnouncement` (admin, retracts, audited, second delete is not-found), `AdminListAnnouncements` (admin, scheduled included), `ListAnnouncements` (active and suspended members, only those in force). Append-only: no edit; the table refuses deletes and edits except stamping `deleted_at`.
* `GetTimeZone` / `SetTimeZone(ctx, actor, name)`: per-person display zone, stored in `person_prefs`. `name` must be an IANA zone (`ValidTimeZone`; "Local" and odd characters refused; zone data is embedded via `time/tzdata`) or "" for UTC. Audited at platform scope (`person.time_zone_changed`). `Location(name)` returns the `*time.Location` (UTC on empty or unknown). Adapters convert only for display; the API stays UTC.
* `FindDisputes(ctx, actor, netID, DisputeQuery{Statuses, TxID, Page})`: `ListDisputes` with a transaction filter and keyset paging (newest first, page by the last id). Same visibility as `ListDisputes`.
* Directory search (`MemberPageQuery.Name`, `CountDirectory`, admin member pages) folds case and accents with `domain.FoldName` (NFKD, combining marks removed, Unicode case folding, Turkish dotless i mapped to i) through the SQLite function `circula_fold`, registered by the store.

### Input and result types

```go
type BalanceView struct {
	MemberID		string
	AccountState		domain.AccountState
	Balance			int64
	CreditLimit		int64
	ReservedOutgoing	int64
	Available		int64
	AwaitingFromOthers	int64
	Decimals		int
	LedgerSeq		int64
}

type AdjustDirection string

type AdjustInput struct {
	MemberID	string
	Amount		int64	// minor units, > 0
	Direction	AdjustDirection
	Reason		string
	IdempotencyKey	string	`json:"-"`
}

type AuditQuery struct {
	Action		string
	ActorID		string
	TargetID	string
	From, To	time.Time
	AfterSeq	int64
	Limit		int
}

type DepartInput struct {
	MemberID	string	// empty = the caller departs voluntarily
	Reason		string	// mandatory when an administrator removes someone else
}

type OpenDisputeInput struct {
	TxID		string
	Category	string	// not_delivered | quality | wrong_amount | unauthorised | other
	Statement	string
	IdempotencyKey	string	`json:"-"`
}

type ResolveInput struct {
	DisputeID	string
	Outcome		domain.DisputeOutcome
	Amount		int64	// partially_upheld only: minor units to reverse
	Rationale	string
	IdempotencyKey	string	`json:"-"`
}

type DisputeView struct {
	Dispute		*domain.Dispute
	Tx		*domain.Transaction
	PayerName	string
	PayeeName	string
	Events		[]store.DisputeEvent
}

type ExportScope string

type ExportFormat string

type ExportSink interface {
	Create(name string) (io.WriteCloser, error)
}

type ManifestFile struct {
	Name	string	`json:"name"`
	Rows	int64	`json:"rows"`
	Bytes	int64	`json:"bytes"`
	SHA256	string	`json:"sha256"`
}

type Manifest struct {
	SchemaVersion	int		`json:"schema_version"`
	NetworkID	string		`json:"network_id"`
	Scope		ExportScope	`json:"scope"`
	Format		ExportFormat	`json:"format"`
	Decimals	int		`json:"decimals"`
	LedgerSeq	int64		`json:"ledger_seq"`
	GeneratedAt	time.Time	`json:"generated_at"`
	Files		[]ManifestFile	`json:"files"`
}

type DirSink struct{ Dir string }

type MemSink struct{ Files map[string]*memFile }

type InviteInput struct {
	TTL time.Duration	// default 7 days, max 90 days
}

type ListingInput struct {
	Type		domain.ListingType
	Title		string
	Description	string
	Price		*int64
	IdempotencyKey	string	`json:"-"`
}

type ListingUpdate struct {
	ListingID	string
	Title		*string
	Description	*string
	Price		*int64	// set price; use ClearPrice to remove
	ClearPrice	bool
	Status		*domain.ListingStatus	// active | paused | archived
}

type ListingQuery struct {
	Type		domain.ListingType
	Query		string
	MemberID	string	// set to browse one member; own listings show every status
	Page		Page
}

type DirectoryEntry struct {
	MemberID	string
	DisplayName	string
}

type SetLimitInput struct {
	MemberID	string
	Limit		*int64		// nil removes the override (default applies again)
	ExpiresAt	*time.Time	// optional: time-limited override (DOM-021, OQ-DOM-2)
	Reason		string
}

type CreateNetworkInput struct {
	Name			string
	Slug			string
	Policy			*domain.Policy	// nil = platform defaults
	CreatorDisplayName	string
}

type NetworkInfo struct {
	ID, Name, Slug	string
	Status		domain.NetworkStatus
	JoinMode	domain.JoinMode
}

type PolicyChange struct {
	Policy		domain.Policy
	Reason		string
	IdempotencyKey	string	`json:"-"`
}

type Argon2Params struct {
	MemoryKiB	uint32
	Time		uint32
	Threads		uint8
	KeyLen		uint32
	SaltLen		uint32
}

type ReverseInput struct {
	TxID		string
	Amount		int64	// minor units; 0 = the full remaining reversible amount
	Reason		string
	IdempotencyKey	string	`json:"-"`
}

type Config struct {
	Clock	domain.Clock	// default: wall clock
	IDs	domain.IDGen	// default: UUIDv7

	Argon		Argon2Params		// password hashing cost (DEC-006 pending benchmark)
	MinPasswordLen	int			// default 12 (SEC-002)
	PasswordDenied	func(string) bool	// optional breached-password check
	SessionIdle	time.Duration		// default 30 days (DAT section 9)
	SessionMax	time.Duration		// default 90 days
	IdempotencyTTL	time.Duration		// default 7 days (DAT section 9)

	MaxOpenRequestsPerPair		int	// open requests from one proposer to one payer; default 5, <0 off (SA-08)
	MaxOpenProposalsPerMember	int	// open proposals created by one member; default 200, <0 off (SA-08)
	ExportsPerHour			int	// Export calls per member per network per hour; default 30, <0 off (SA-11)
}

type Service struct {
	st		store.Store
	cfg		Config
	dummyHash	string	// for constant-effort login of unknown users (SEC-005)
}

type Page struct {
	AfterID	string
	Limit	int
}

type TokenInput struct {
	Name		string
	Scopes		[]domain.Scope	// subset of the caller's own scopes
	NetworkID	string		// optional: restrict the token to one network
	ExpiresAt	*time.Time
}

type Direction string

type ProposeInput struct {
	CounterpartyID	string	// member id
	Direction	Direction
	Amount		int64	// minor units, > 0
	ListingID	string
	Note		string
	IdempotencyKey	string	`json:"-"`
}

type TxRef struct {
	TxID		string
	IdempotencyKey	string	`json:"-"`
}

type CancelInput struct {
	TxID		string
	Reason		string	// mandatory for administrator cancellation
	IdempotencyKey	string	`json:"-"`
}

type TxView struct {
	Tx	*domain.Transaction
	Entries	[]*domain.Entry
	History	[]*domain.StatusEvent
}

type LedgerReport struct {
	NetworkID	string
	OK		bool
	Findings	[]string
	LedgerSeq	int64
	AuditEvents	int64
	Started		time.Time
	Finished	time.Time
	RunID		string	// set by RecordConsistencyRun / VerifyAndRecordLedger
}

type RegisterInput struct {
	Login		string
	Password	string
	RevealDuplicate	bool	// trusted operator tooling only (SA-10)
}

type EntryQuery struct {
	BeforeSeq	int64	// 0 = newest; keyset cursor
	Limit		int
}

type EntryRow struct {
	Entry	*domain.Entry
	Tx	*domain.Transaction
}

type PendingQuery struct {
	AwaitingMe	bool	// only proposals made by the other party
	Page		Page
}

type PurgeReport struct {
	Sessions, Throttles, IdempotencyKeys	int64
}
```

Domain types (`Network`, `Member`, `Transaction`, `Dispute`, `AuditEvent`, `Policy`, ...) are in `internal/domain/types.go`, `policy.go`, `statemachine.go`, `audit.go`.

## 4. Authorisation summary (spec 02 matrix, SEC-009/010)

| Capability | Who |
|---|---|
| Register, Login, RedeemInvitation, JoinNetwork, CreateNetwork | any authenticated person (adapter decides who may sign up / create) |
| GetBalance, ListTransactions, Export(`mine`), ListAudit (own) | own member, any status except `pending` (suspended and departed included) |
| ListDirectory, ListListings, Propose, Accept, CreateListing, OpenDispute | **active** member |
| Decline, Cancel (withdraw), WithdrawDispute | active or suspended party (exits, DOM-039) |
| Resolve / StartReview / RetryDisputePosting | active reviewer or admin who is not a party and did not propose, accept or reverse the transaction |
| Admin methods (adjust, reverse, limits, suspend, roles, invitations, policy, close, verify, ledger/members/audit export, AdminList*) | active admin; never on self where DOM-037 says so |
| ExpireProposals | System actor only |

## 5. Invariants and where they are enforced / tested

| Rule | Code | Test |
|---|---|---|
| LED-1 entries and balances sum to zero | `postTx`; SQL trigger `tx_complete_needs_entries` | `property_test.go`, `TestTradeScenarioBalances` |
| LED-2 append-only | SQL triggers on entries, audit, status/dispute events, policies, runs | `store/sqlite_test.go`, `TestAppendOnlyThroughRawAccess` |
| LED-3 cached == sum | `AddBalance` only inside `postTx`; `VerifyLedger` | `TestVerifyDetectsBalanceCorruptionAndBalanceAt`, property test |
| LED-4 atomic + idempotent | one `store.Update` per command; `idempotent()` | `atomicity_test.go`, `idempotency_test.go`, `concurrency_test.go` |
| LED-5 limit at post time, serialised | `checkCapacity` inside write tx | `TestParallelAcceptsNeverBreachCreditLimit`, property test |
| LED-6 audit in same unit | `Service.audit` inside every command | `TestEveryCommandWritesAuditEvent`, `TestAuditFailureRollsBackStateChange` |
| LED-7 integer money, no floats | `int64`; DB `CHECK(amount > 0)` | `TestNoFloatsInMoneyPackages`, `TestMigrationsAvoidFloatColumns` |
| DAT-005 no cross-network refs | composite FKs `(network_id, id)` | `TestCrossNetworkReferencesRejected`, `TestCrossNetworkIdsAreNotFound` |
| DOM-010 state machine | `domain.NextTxStatus` + trigger `tx_status_flow` | `statemachine_test.go` (full cross product) |
| DAT-030 audit hash chain | `AuditHash`, checked by `VerifyLedger` | `TestAuditChainDetectsTampering` |
| SA-09 audit tail truncation | `audit_heads` row advanced in the same unit as every append (trigger: head only advances); `VerifyLedger` compares | `TestAuditTailTruncationDetected`, `TestAuditHeadTracksAppendsAndIsProtected`, `TestMigration0002BackfillsAuditHeads` |
| OPS-016 verify is read-only | `VerifyLedger` on `Store.View` | `TestVerifyLedgerIsReadOnly` (database content hash unchanged), `TestVerifyLedgerDoesNotTakeWriteGate` |
| SVC-02 View is read-only | `PRAGMA query_only` per view connection | `store.TestViewIsReadOnly` |
| SEC-006 atomic login throttle | check + count in one write unit before verifying | `TestLoginThrottleIsAtomicUnderConcurrency`, `TestLoginThrottleKeyIsBounded` |
| CMD-05 busy is retryable | `isBusy` -> `domain.CodeUnavailable` in `store.run`/`mapErr` | `store.TestBusyDatabaseIsRetryable` |
| DOM-011/012 consent before reservation | `ReservedOutgoing` counts the payer's own proposals only | `TestUnsolicitedRequestsDoNotFreezeCredit`, `TestRequestExposureAndCounts` |
| SA-12 retention | `Purge` | `TestPurge`, `store.TestPurgePrimitives` |

## 6. Decisions, deviations and gaps

Deviations from the specs, with reasons:

* **DAT-012** (one reversal per original) is **not** enforced: DOM-020 requires cumulative partial reversals, so the cap is
  `reversed_total <= amount` (DB `CHECK`, `VerifyLedger` cross-check).
* **Disputes cover trades only**; reversals/adjustments/grants cannot be disputed. A reversal cannot be reversed (DOM-019).
* **Admin reversal needs status `completed`**; disputed transactions are reversed only through the dispute decision.
* **`accepted` is never persisted** (OQ-DOM-1 default); the audit trail records `proposed -> completed`.
* **OQ-DOM-2** implemented as both paths: `posting_pending` plus a time-limited, audited `SetCreditLimit` override, then `RetryDisputePosting`.
* **Departed members cannot rejoin** (UNIQUE network+person, and DOM-003 has no transition out of `departed`). Denied
  applicants, people who left and people who were removed all end as `departed`. A second join by a pending/active
  member is idempotent (FR-017, DOM-002); by a **suspended or departed** one it is refused with `forbidden` and changes
  nothing (SVC-04; before, it returned success and the web UI said "you have joined"). Re-application after denial or
  removal is therefore **not supported**; supporting it needs a new DOM-003 transition (open question for FR-011).
* Founder membership and the founder's account get no sign-up grant. The founder is active in a `setup` network (needed to bootstrap FR-004).
* Sign-up grants are the only `grant` source (DOM-027); there is no periodic grant.
* Login throttling is per login (including unknown logins); per-address limiting belongs to the HTTP adapter (SEC-006).
  Each evaluated attempt is counted before the password is checked (see `Login`), so a correct password on the 5th try
  still resets the counter but a parallel burst cannot out-run it. A locked attempt is not counted again.
* **Request proposals (SA-08).** DOM-012 reserves the payer's capacity on proposal, but for a `request` the payer has not
  consented (DOM-011), so reserving would let anyone freeze another member's credit. Decision: `ReservedOutgoing`
  counts only proposals the payer made; a request is checked against the payer's available credit when made (hopeless
  requests are refused), counted with the proposer's other open requests to that payer (their sum must fit the payer's
  available credit; at most `MaxOpenRequestsPerPair`, default 5), and the payer's acceptance re-checks the hard limit
  (DOM-014). One member may hold at most `MaxOpenProposalsPerMember` (200) open proposals. Both return `rate-limited`.
  This refines DOM-012's "reserve" for requests; the DOM-012 acceptance test (30.00 available, 20.00 then 15.00) holds
  for payer-initiated proposals.
* **Default flag at accept (SVC-03).** With `DefaultFlagBlocksPayer`, `AcceptTransaction` also refuses a payer flagged
  after the proposal (forbidden). DOM-025 names proposals as an example only; blocking the posting is the stricter
  reading and keeps a flagged member from paying through a stale proposal.
* **Sign-up is uniform (SA-10).** `RegisterPerson` never reveals a taken login (see method reference). Consequence for
  adapters: do not auto-login after sign-up and show one message for both outcomes ("sign in with the details you chose").
* **Consistency checks are read-only (SVC-01).** `VerifyLedger` never writes. Only an explicit administrator action
  records a run (`VerifyAndRecordLedger`); startup, scheduled, migrate/restore and backup-verification callers use
  `VerifyLedger` and keep their own result, so they no longer add an audit event and a run row each.
* **Audit head (SA-09).** `audit_heads` holds the newest `(seq, hash)` per scope, written in the same unit as the
  event. It detects deletion of the newest events by anyone who drops the append-only triggers but does not also edit
  the head; an attacker who can rewrite both (and recompute the unkeyed hashes) is still undetected. For stronger
  evidence publish the head outside the database (metrics, backup manifest). After tampering, appends to that scope fail
  closed (head cannot move backwards) until an operator investigates.
* Reads by admins that are audited (`ListAudit`, `AdminGetMemberBalance`) write an audit row, so they use a write transaction.
  Adapters that only need state for a precondition use `PeekMemberBalance` (or `SetCreditLimitView`) instead (SVC-05).
* `Export` audits first in its own unit, then streams from one read snapshot; the snapshot's `ledger_seq` is in the manifest.
  A per-member quota (`ExportsPerHour`, default 30) is checked before the audit write (SA-11); the adapter still owns
  its artefact store limits.
* **Retention (SA-12).** `Purge` is the only deleter of sessions, throttle rows and idempotency records. It is not
  scheduled by the service; the job runner must call it (hourly is plenty).

Not implemented (outside this core task or spec P1/P2); adapters must not assume them:

* FR-076 other-party dispute response, FR-078 re-review, FR-024 reports, FR-038 notifications,
  DOM-025.1 limit-warning job, SEC-011 dual control, SEC-006 per-address limits, account recovery (SEC-008), MFA (SEC-007),
  setup-secret bootstrap (FR-002; adapter concern), break-glass operator access (SEC-013), per-person session cap,
  platform-scope (`network_id = ''`) audit-chain verification (SVC-06).
* `store.Open` migrates. Pre-migration backups, the post-migration consistency gate (DAT-043/044), the single-writer
  lock (ARC-011), CSRF and the CLI wrapping are implemented in `cmd/circula` and `internal/web`, not here; the service
  only supplies read-only `VerifyLedger` for those gates.

## 7. Testing notes

`go vet ./... && go test -race ./internal/service/... ./internal/store/... ./internal/domain/...` (about one minute under `-race`).
IDs from `testutil.IDs` are deterministic per `Env`, so two `Env`s collide on ids; build a second network inside the same
`Env` to test tenant isolation. `testutil.NewEnvConfig(t, wrap, func(*service.Config))` adjusts limits. Tests build real SQLite files in temp dirs;
`internal/testutil` provides a fake clock, deterministic ids, a fault-injecting store wrapper and a world builder.
