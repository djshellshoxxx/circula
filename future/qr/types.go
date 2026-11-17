package qr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Domain errors. Wrap them (fmt.Errorf("...: %w", ErrX)) and test with errors.Is.
var (
	ErrNotFound             = errors.New("not found")                     // unknown, foreign-network or not visible: one indistinguishable outcome
	ErrForbidden            = errors.New("forbidden")                     // authenticated member may not do this (suspended, inactive, not the owner)
	ErrExpired              = errors.New("request expired")               // past expires_at
	ErrRevoked              = errors.New("code revoked")                  // revoked request or rotated alias
	ErrClaimed              = errors.New("request already claimed")       // claimed by someone else (or by this payer with a different intent)
	ErrIdempotencyConflict  = errors.New("idempotency key reuse")         // same key, different intent
	ErrRateLimited          = errors.New("too many active codes")         // per-member quantity limit
	ErrInsufficientCredit   = errors.New("insufficient available credit") // adapters wrap this for credit/limit failures
	ErrCounterpartyInactive = errors.New("counterparty is not active")    // adapters wrap this when the other side cannot trade
)

// Principal is the authenticated member acting in one network. The host
// authenticates it (session or token, X-Network selection); this module never
// derives it from client-supplied JSON.
type Principal struct {
	NetworkID string
	MemberID  string
}

func (p Principal) valid() bool { return p.NetworkID != "" && p.MemberID != "" }

// Member is the core adapter's fresh view of a member inside one network.
type Member struct {
	ID          string
	DisplayName string
	NetworkName string
	// Active is true only when the member may currently trade: active
	// membership, active network, accepted terms, not suspended or departed.
	Active bool
}

// TxState is the lifecycle state the module distinguishes. Only Completed
// means money moved; Proposed is merely a pending proposal.
type TxState string

// Transaction states reported by the adapter.
const (
	TxProposed  TxState = "proposed"
	TxCompleted TxState = "completed"
	TxClosed    TxState = "closed" // declined, withdrawn, expired or otherwise ended without posting
)

// ProposalInput asks the core to create an ordinary "payer pays payee"
// proposal. The payee must still accept; nothing is posted by the proposal.
type ProposalInput struct {
	NetworkID      string
	PayerID        string // the authenticated scanning member; also the proposer
	PayeeID        string
	AmountMinor    int64
	Note           string
	IdempotencyKey string // stable key bound to the intent; adapters may reuse it for core idempotency
	Origin         string // "qr-request:<id>" or "qr-alias"; audit/context only
}

// ProposalResult identifies the proposal the core created.
type ProposalResult struct {
	TransactionID string
	State         TxState
}

// DBTX is the subset of database access the module and adapters share. Inside
// Store.Update it is bound to the single open write transaction, so adapter
// work is committed or rolled back together with the QR rows.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Core is the narrow seam to the host application. Every method receives the
// open write-transaction handle and MUST read fresh state through it (never a
// cached or client-supplied snapshot) and honour tenant isolation: a member or
// transaction outside networkID is ErrNotFound. Propose MUST run the same
// membership, terms, network, limit and rate checks as the ordinary proposal
// path (for Circula: the private service propose step under the existing
// update transaction). It must not call a public method that opens its own
// transaction. See README "Core adapter contract".
type Core interface {
	Member(ctx context.Context, q DBTX, networkID, memberID string) (Member, error)
	// AvailableCredit estimates what the payer could commit now (balance +
	// limit - reserved). It is informational on confirmation screens; the
	// authoritative check is inside Propose.
	AvailableCredit(ctx context.Context, q DBTX, networkID, memberID string) (int64, error)
	Propose(ctx context.Context, q DBTX, in ProposalInput) (ProposalResult, error)
	TxStatus(ctx context.Context, q DBTX, networkID, txID string) (TxState, error)
}

// Config configures a Service. Zero values select safe defaults.
type Config struct {
	// Origin is the configured trusted public origin, e.g. "https://circula.example".
	Origin string
	// NetworkEnabled is the explicit per-network opt-in and is REQUIRED (there
	// is no default-on). It runs inside every operation's transaction handle, so
	// a host can read its own enablement setting there. A false result makes
	// the feature look absent (ErrNotFound) for that network.
	NetworkEnabled func(ctx context.Context, q DBTX, networkID string) (bool, error)
	// AliasSecret (>=32 bytes) derives static aliases so the QR can be
	// re-rendered while only hashes are stored. Keep it with other server secrets.
	AliasSecret []byte
	Now         func() time.Time
	// NewToken overrides token generation in tests only.
	NewToken func() (string, error)
	// NewID overrides request id generation in tests only.
	NewID func() (string, error)

	MaxNoteRunes      int           // default 140
	MaxAmountMinor    int64         // default 1_000_000_000_000
	DefaultTTL        time.Duration // default 24h
	MinTTL            time.Duration // default 1m
	MaxTTL            time.Duration // default 7*24h
	MaxActiveRequests int           // per recipient, default 20
	MaxRotations      int           // lifetime alias generations per member, default 1000
	PNGPixels         int           // default 512 (bounded 128..2048)
}

func (c *Config) withDefaults() (Config, error) {
	o := *c
	if err := ValidateOrigin(o.Origin); err != nil {
		return o, fmt.Errorf("qr: invalid trusted origin: %w", err)
	}
	if o.NetworkEnabled == nil {
		return o, errors.New("qr: NetworkEnabled is required (explicit per-network opt-in)")
	}
	if len(o.AliasSecret) < 32 {
		return o, errors.New("qr: AliasSecret must be at least 32 bytes")
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.NewToken == nil {
		o.NewToken = NewToken
	}
	if o.NewID == nil {
		o.NewID = newID
	}
	if o.MaxNoteRunes <= 0 {
		o.MaxNoteRunes = 140
	}
	if o.MaxAmountMinor <= 0 {
		o.MaxAmountMinor = 1_000_000_000_000
	}
	if o.DefaultTTL <= 0 {
		o.DefaultTTL = 24 * time.Hour
	}
	if o.MinTTL <= 0 {
		o.MinTTL = time.Minute
	}
	if o.MaxTTL <= 0 {
		o.MaxTTL = 7 * 24 * time.Hour
	}
	if o.MaxActiveRequests <= 0 {
		o.MaxActiveRequests = 20
	}
	if o.MaxRotations <= 0 {
		o.MaxRotations = 1000
	}
	if o.PNGPixels == 0 {
		o.PNGPixels = 512
	}
	if o.PNGPixels < 128 || o.PNGPixels > 2048 {
		return o, errors.New("qr: PNGPixels must be within 128..2048")
	}
	return o, nil
}
