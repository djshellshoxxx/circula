package qr

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Service implements alias and one-time-request behaviour over a Store and a
// Core adapter. It is not registered anywhere and starts no goroutines.
type Service struct {
	cfg   Config
	store *Store
	core  Core
}

// NewService validates cfg and builds a Service. The store must already be migrated.
func NewService(cfg Config, store *Store, core Core) (*Service, error) {
	c, err := cfg.withDefaults()
	if err != nil {
		return nil, err
	}
	if store == nil || core == nil {
		return nil, errors.New("qr: store and core are required")
	}
	return &Service{cfg: c, store: store, core: core}, nil
}

// MaxNoteRunes is the configured note length limit (for form maxlength).
func (s *Service) MaxNoteRunes() int { return s.cfg.MaxNoteRunes }

// Origin returns the trusted public origin links are built on.
func (s *Service) Origin() string { return s.cfg.Origin }

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "qrr_" + hex.EncodeToString(b), nil
}

var idemKeyRE = regexp.MustCompile(`^[A-Za-z0-9._~:-]{1,128}$`)
var idRE = regexp.MustCompile(`^qrr_[0-9a-f]{32}$`)

func validKey(k string) error {
	if !idemKeyRE.MatchString(k) {
		return fmt.Errorf("%w: idempotency key must be 1-128 characters of A-Z a-z 0-9 . _ ~ : -", ErrInvalid)
	}
	return nil
}

// fingerprint hashes length-prefixed parts so field boundaries cannot be forged.
func fingerprint(parts ...string) []byte {
	h := sha256.New()
	var n [8]byte
	for _, p := range parts {
		binary.BigEndian.PutUint64(n[:], uint64(len(p)))
		h.Write(n[:])
		h.Write([]byte(p))
	}
	return h.Sum(nil)
}

func ms(t time.Time) int64        { return t.UnixMilli() }
func fromMS(v int64) time.Time    { return time.UnixMilli(v).UTC() }
func itoa(v int64) string         { return fmt.Sprintf("%d", v) }
func (s *Service) now() time.Time { return s.cfg.Now().UTC().Truncate(time.Millisecond) }

func (s *Service) deriveAlias(network, member string, gen int) string {
	m := hmac.New(sha256.New, s.cfg.AliasSecret)
	m.Write([]byte("circula-qr-alias-v1\x00" + network + "\x00" + member + "\x00" + itoa(int64(gen))))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil)[:16])
}

func cleanNote(note string, max int) (string, error) {
	note = strings.TrimSpace(note)
	if !utf8.ValidString(note) || utf8.RuneCountInString(note) > max {
		return "", fmt.Errorf("%w: note must be valid text of at most %d characters", ErrInvalid, max)
	}
	for _, r := range note {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Co, r) || r == 0xfffd {
			return "", fmt.Errorf("%w: note contains control or formatting characters", ErrInvalid)
		}
	}
	return note, nil
}

func (s *Service) checkAmount(a int64) error {
	if a <= 0 || a > s.cfg.MaxAmountMinor {
		return fmt.Errorf("%w: amount must be a positive integer within the allowed maximum", ErrInvalid)
	}
	return nil
}

func (s *Service) active(ctx context.Context, q DBTX, network, member string) (Member, error) {
	m, err := s.core.Member(ctx, q, network, member)
	if err != nil {
		return Member{}, err
	}
	if !m.Active {
		return m, fmt.Errorf("%w: member is not active", ErrForbidden)
	}
	return m, nil
}

// ---- static aliases ----------------------------------------------------

// AliasInfo describes a member's static alias. Code is a public merchant
// identifier (no amount, no balance); treat it as shareable but not secret.
type AliasInfo struct {
	Generation int
	Code       string
	URL        string
	Compact    string
	CreatedAt  time.Time
	Revoked    bool
}

func (s *Service) aliasInfo(network, member string, gen int, created time.Time, revoked bool) (AliasInfo, error) {
	code := s.deriveAlias(network, member, gen)
	u, err := AliasLink(s.cfg.Origin, code)
	if err != nil {
		return AliasInfo{}, err
	}
	return AliasInfo{Generation: gen, Code: code, URL: u, Compact: CompactCode(KindAlias, code), CreatedAt: created, Revoked: revoked}, nil
}

func (s *Service) insertAlias(ctx context.Context, q DBTX, network, member string, gen int, now time.Time) error {
	h := sha256.Sum256([]byte(s.deriveAlias(network, member, gen)))
	_, err := q.ExecContext(ctx, `INSERT INTO qr_member_alias(network_id,member_id,generation,alias_hash,created_at) VALUES(?,?,?,?,?)`,
		network, member, gen, h[:], ms(now))
	return err
}

// MyAlias returns the caller's live alias, creating generation 1 on first use.
func (s *Service) MyAlias(ctx context.Context, p Principal) (AliasInfo, error) {
	if !p.valid() {
		return AliasInfo{}, ErrForbidden
	}
	var out AliasInfo
	err := s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := s.active(ctx, q, p.NetworkID, p.MemberID); err != nil {
			return err
		}
		var gen int
		var created int64
		err := q.QueryRowContext(ctx, `SELECT generation, created_at FROM qr_member_alias WHERE network_id=? AND member_id=? AND revoked_at IS NULL`,
			p.NetworkID, p.MemberID).Scan(&gen, &created)
		if errors.Is(err, sql.ErrNoRows) {
			now := s.now()
			gen, created = 1, ms(now)
			var maxGen sql.NullInt64
			if err := q.QueryRowContext(ctx, `SELECT MAX(generation) FROM qr_member_alias WHERE network_id=? AND member_id=?`, p.NetworkID, p.MemberID).Scan(&maxGen); err != nil {
				return err
			}
			if maxGen.Valid { // every alias was revoked earlier: continue the sequence
				gen = int(maxGen.Int64) + 1
			}
			if err := s.insertAlias(ctx, q, p.NetworkID, p.MemberID, gen, now); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		out, err = s.aliasInfo(p.NetworkID, p.MemberID, gen, fromMS(created), false)
		return err
	})
	return out, err
}

// RotateAlias revokes the live alias and issues the next generation. It is
// idempotent per (member, key): a retry with the same key returns the same
// generation and does not rotate again. The old QR stays permanently revoked.
func (s *Service) RotateAlias(ctx context.Context, p Principal, key string) (AliasInfo, error) {
	if !p.valid() {
		return AliasInfo{}, ErrForbidden
	}
	if err := validKey(key); err != nil {
		return AliasInfo{}, err
	}
	fp := fingerprint("alias.rotate", p.NetworkID, p.MemberID)
	var out AliasInfo
	err := s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := s.active(ctx, q, p.NetworkID, p.MemberID); err != nil {
			return err
		}
		prior, ok, err := s.idem(ctx, q, p, "alias.rotate", key, fp)
		if err != nil {
			return err
		}
		if ok {
			var gen int
			var created int64
			var revoked sql.NullInt64
			if err := q.QueryRowContext(ctx, `SELECT generation, created_at, revoked_at FROM qr_member_alias WHERE network_id=? AND member_id=? AND generation=?`,
				p.NetworkID, p.MemberID, prior).Scan(&gen, &created, &revoked); err != nil {
				return err
			}
			out, err = s.aliasInfo(p.NetworkID, p.MemberID, gen, fromMS(created), revoked.Valid)
			return err
		}
		var maxGen sql.NullInt64
		if err := q.QueryRowContext(ctx, `SELECT MAX(generation) FROM qr_member_alias WHERE network_id=? AND member_id=?`, p.NetworkID, p.MemberID).Scan(&maxGen); err != nil {
			return err
		}
		gen := int(maxGen.Int64) + 1
		if gen > s.cfg.MaxRotations {
			return fmt.Errorf("%w: alias rotation limit reached", ErrRateLimited)
		}
		now := s.now()
		if _, err := q.ExecContext(ctx, `UPDATE qr_member_alias SET revoked_at=? WHERE network_id=? AND member_id=? AND revoked_at IS NULL`, ms(now), p.NetworkID, p.MemberID); err != nil {
			return err
		}
		if err := s.insertAlias(ctx, q, p.NetworkID, p.MemberID, gen, now); err != nil {
			return err
		}
		if err := s.idemPut(ctx, q, p, "alias.rotate", key, fp, itoa(int64(gen)), now); err != nil {
			return err
		}
		out, err = s.aliasInfo(p.NetworkID, p.MemberID, gen, now, false)
		return err
	})
	return out, err
}

// ---- idempotency helpers ----------------------------------------------

// idem returns (result, true) when key was recorded for the same intent,
// ErrIdempotencyConflict when it was recorded for a different intent.
func (s *Service) idem(ctx context.Context, q DBTX, p Principal, scope, key string, fp []byte) (string, bool, error) {
	var stored []byte
	var result string
	err := q.QueryRowContext(ctx, `SELECT fingerprint, result FROM qr_idempotency WHERE network_id=? AND member_id=? AND scope=? AND idem_key=?`,
		p.NetworkID, p.MemberID, scope, key).Scan(&stored, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if !hmac.Equal(stored, fp) {
		return "", false, fmt.Errorf("%w: key was used for a different request", ErrIdempotencyConflict)
	}
	return result, true, nil
}

func (s *Service) idemPut(ctx context.Context, q DBTX, p Principal, scope, key string, fp []byte, result string, now time.Time) error {
	_, err := q.ExecContext(ctx, `INSERT INTO qr_idempotency(network_id,member_id,scope,idem_key,fingerprint,result,created_at) VALUES(?,?,?,?,?,?,?)`,
		p.NetworkID, p.MemberID, scope, key, fp, result, ms(now))
	return err
}

// ---- trade requests ----------------------------------------------------

// RequestStatus is the distinct lifecycle a recipient sees. Only
// StatusCompleted means the trade posted after both sides consented.
type RequestStatus string

// Request statuses.
const (
	StatusOpen      RequestStatus = "open"      // unclaimed, unexpired, unrevoked
	StatusExpired   RequestStatus = "expired"   // never claimed
	StatusRevoked   RequestStatus = "revoked"   // revoked by the recipient before any claim
	StatusPending   RequestStatus = "pending"   // claimed: proposal created, awaiting acceptance
	StatusCompleted RequestStatus = "completed" // proposal accepted and posted by the core
	StatusClosed    RequestStatus = "closed"    // proposal ended without posting
)

// RequestView is the safe (token-free) description of a request.
type RequestView struct {
	ID            string
	NetworkID     string
	RecipientID   string
	AmountMinor   int64
	Note          string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	RevokedAt     *time.Time
	Claimed       bool
	ClaimedBy     string
	TransactionID string
	Status        RequestStatus
}

type reqRow struct {
	id, network, recipient string
	hash                   []byte
	amount                 int64
	note                   string
	created, expires       int64
	revoked, claimedAt     sql.NullInt64
	claimedTx, claimedBy   sql.NullString
	claimFP                []byte
}

const reqCols = `id,network_id,recipient_id,token_hash,amount_minor,note,created_at,expires_at,revoked_at,claimed_tx_id,claimed_by,claimed_at,claim_fingerprint`

func (r *reqRow) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&r.id, &r.network, &r.recipient, &r.hash, &r.amount, &r.note, &r.created, &r.expires, &r.revoked, &r.claimedTx, &r.claimedBy, &r.claimedAt, &r.claimFP)
}

func (s *Service) requestByToken(ctx context.Context, q DBTX, network, token string) (*reqRow, error) {
	h, err := TokenHash(token)
	if err != nil {
		return nil, ErrNotFound // malformed and unknown are indistinguishable to the caller
	}
	var r reqRow
	if err := r.scan(q.QueryRowContext(ctx, `SELECT `+reqCols+` FROM qr_trade_request WHERE network_id=? AND token_hash=?`, network, h[:])); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &r, nil
}

func (s *Service) view(ctx context.Context, q DBTX, r *reqRow, now time.Time) (RequestView, error) {
	v := RequestView{ID: r.id, NetworkID: r.network, RecipientID: r.recipient, AmountMinor: r.amount, Note: r.note,
		CreatedAt: fromMS(r.created), ExpiresAt: fromMS(r.expires)}
	if r.revoked.Valid {
		t := fromMS(r.revoked.Int64)
		v.RevokedAt = &t
	}
	switch {
	case r.claimedTx.Valid:
		v.Claimed, v.ClaimedBy, v.TransactionID = true, r.claimedBy.String, r.claimedTx.String
		st, err := s.core.TxStatus(ctx, q, r.network, r.claimedTx.String)
		if err != nil {
			return v, err
		}
		switch st {
		case TxCompleted:
			v.Status = StatusCompleted
		case TxClosed:
			v.Status = StatusClosed
		default:
			v.Status = StatusPending
		}
	case r.revoked.Valid:
		v.Status = StatusRevoked
	case !now.Before(fromMS(r.expires)):
		v.Status = StatusExpired
	default:
		v.Status = StatusOpen
	}
	return v, nil
}

// CreateRequestInput describes a new one-time request.
type CreateRequestInput struct {
	AmountMinor int64
	Note        string
	TTL         time.Duration // 0 = default
}

// CreatedRequest is returned from CreateRequest. Token, URL, Compact and PNG
// are present only on the original creation: the raw token is never stored,
// so an idempotent replay returns the view alone (Replayed true).
type CreatedRequest struct {
	View     RequestView
	Token    string
	URL      string
	Compact  string
	Replayed bool
}

// CreateRequest creates a one-time request payable to the caller.
func (s *Service) CreateRequest(ctx context.Context, p Principal, key string, in CreateRequestInput) (CreatedRequest, error) {
	if !p.valid() {
		return CreatedRequest{}, ErrForbidden
	}
	if err := validKey(key); err != nil {
		return CreatedRequest{}, err
	}
	if err := s.checkAmount(in.AmountMinor); err != nil {
		return CreatedRequest{}, err
	}
	note, err := cleanNote(in.Note, s.cfg.MaxNoteRunes)
	if err != nil {
		return CreatedRequest{}, err
	}
	ttl := in.TTL
	if ttl == 0 {
		ttl = s.cfg.DefaultTTL
	}
	if ttl < s.cfg.MinTTL || ttl > s.cfg.MaxTTL {
		return CreatedRequest{}, fmt.Errorf("%w: expiry must be between %s and %s", ErrInvalid, s.cfg.MinTTL, s.cfg.MaxTTL)
	}
	fp := fingerprint("request.create", p.NetworkID, p.MemberID, itoa(in.AmountMinor), note, itoa(int64(ttl/time.Millisecond)))
	var out CreatedRequest
	err = s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := s.active(ctx, q, p.NetworkID, p.MemberID); err != nil {
			return err
		}
		now := s.now()
		prior, ok, err := s.idem(ctx, q, p, "request.create", key, fp)
		if err != nil {
			return err
		}
		if ok {
			var r reqRow
			if err := r.scan(q.QueryRowContext(ctx, `SELECT `+reqCols+` FROM qr_trade_request WHERE network_id=? AND id=?`, p.NetworkID, prior)); err != nil {
				return err
			}
			v, err := s.view(ctx, q, &r, now)
			out = CreatedRequest{View: v, Replayed: true}
			return err
		}
		var live int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM qr_trade_request WHERE network_id=? AND recipient_id=? AND claimed_tx_id IS NULL AND revoked_at IS NULL AND expires_at>?`,
			p.NetworkID, p.MemberID, ms(now)).Scan(&live); err != nil {
			return err
		}
		if live >= s.cfg.MaxActiveRequests {
			return fmt.Errorf("%w: revoke or wait for an open request first", ErrRateLimited)
		}
		token, err := s.cfg.NewToken()
		if err != nil {
			return err
		}
		h, err := TokenHash(token)
		if err != nil {
			return err
		}
		id, err := s.cfg.NewID()
		if err != nil {
			return err
		}
		exp := now.Add(ttl)
		if _, err := q.ExecContext(ctx, `INSERT INTO qr_trade_request(id,network_id,recipient_id,token_hash,amount_minor,note,created_at,expires_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, p.NetworkID, p.MemberID, h[:], in.AmountMinor, note, ms(now), ms(exp)); err != nil {
			return err
		}
		if err := s.idemPut(ctx, q, p, "request.create", key, fp, id, now); err != nil {
			return err
		}
		u, err := Link(s.cfg.Origin, token)
		if err != nil {
			return err
		}
		out = CreatedRequest{Token: token, URL: u, Compact: CompactCode(KindRequest, token), View: RequestView{
			ID: id, NetworkID: p.NetworkID, RecipientID: p.MemberID, AmountMinor: in.AmountMinor, Note: note,
			CreatedAt: now, ExpiresAt: exp, Status: StatusOpen}}
		return nil
	})
	return out, err
}

// Preview is the server-authoritative confirmation data for one scan. It never
// trusts anything encoded in the QR beyond the opaque code.
type Preview struct {
	Kind            Kind
	NetworkID       string
	NetworkName     string
	PayerID         string
	PayerName       string
	PayeeID         string
	PayeeName       string
	AmountMinor     int64 // 0 for static aliases: the payer chooses
	Note            string
	ExpiresAt       time.Time
	RequestID       string
	AvailableCredit int64 // estimate only; the proposal re-checks
	Sufficient      bool
	Own             bool // the code belongs to the caller
	Claimed         bool // request already claimed (by the caller if ClaimedByYou)
	ClaimedByYou    bool
	TransactionID   string
	Status          RequestStatus
}

func (s *Service) previewParties(ctx context.Context, q DBTX, p Principal, payeeID string) (payer, payee Member, avail int64, err error) {
	payer, err = s.active(ctx, q, p.NetworkID, p.MemberID)
	if err != nil {
		return
	}
	payee, err = s.core.Member(ctx, q, p.NetworkID, payeeID)
	if err != nil {
		return
	}
	avail, err = s.core.AvailableCredit(ctx, q, p.NetworkID, p.MemberID)
	return
}

// PreviewRequest resolves a one-time token for the confirmation screen. It does
// not claim, reserve or modify anything.
func (s *Service) PreviewRequest(ctx context.Context, p Principal, token string) (Preview, error) {
	if !p.valid() {
		return Preview{}, ErrForbidden
	}
	var out Preview
	err := s.store.View(ctx, func(ctx context.Context, q DBTX) error {
		r, err := s.requestByToken(ctx, q, p.NetworkID, token)
		if err != nil {
			return err
		}
		now := s.now()
		v, err := s.view(ctx, q, r, now)
		if err != nil {
			return err
		}
		payer, payee, avail, err := s.previewParties(ctx, q, p, r.recipient)
		if err != nil {
			return err
		}
		out = Preview{Kind: KindRequest, NetworkID: p.NetworkID, NetworkName: payee.NetworkName, PayerID: payer.ID, PayerName: payer.DisplayName,
			PayeeID: payee.ID, PayeeName: payee.DisplayName, AmountMinor: r.amount, Note: r.note, ExpiresAt: v.ExpiresAt, RequestID: r.id,
			AvailableCredit: avail, Sufficient: avail >= r.amount, Own: r.recipient == p.MemberID, Claimed: v.Claimed,
			ClaimedByYou: v.Claimed && v.ClaimedBy == p.MemberID, TransactionID: v.TransactionID, Status: v.Status}
		if out.Own || (v.Claimed && !out.ClaimedByYou) {
			out.TransactionID = "" // never leak another party's transaction id
		}
		switch v.Status {
		case StatusRevoked:
			return ErrRevoked
		case StatusExpired:
			return ErrExpired
		}
		if v.Claimed && !out.ClaimedByYou && !out.Own {
			return ErrClaimed
		}
		if !payee.Active && !v.Claimed {
			return ErrCounterpartyInactive
		}
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	return out, nil
}

// ResolveAlias resolves a static alias for the confirmation screen. The amount
// is chosen by the payer afterwards.
func (s *Service) ResolveAlias(ctx context.Context, p Principal, code string) (Preview, error) {
	if !p.valid() {
		return Preview{}, ErrForbidden
	}
	var out Preview
	err := s.store.View(ctx, func(ctx context.Context, q DBTX) error {
		recipient, err := s.aliasOwner(ctx, q, p.NetworkID, code)
		if err != nil {
			return err
		}
		payer, payee, avail, err := s.previewParties(ctx, q, p, recipient)
		if err != nil {
			return err
		}
		out = Preview{Kind: KindAlias, NetworkID: p.NetworkID, NetworkName: payee.NetworkName, PayerID: payer.ID, PayerName: payer.DisplayName,
			PayeeID: payee.ID, PayeeName: payee.DisplayName, AvailableCredit: avail, Own: recipient == p.MemberID}
		if !payee.Active {
			return ErrCounterpartyInactive
		}
		return nil
	})
	if err != nil {
		return Preview{}, err
	}
	return out, nil
}

// aliasOwner finds the member behind a live alias in this network.
func (s *Service) aliasOwner(ctx context.Context, q DBTX, network, code string) (string, error) {
	h, err := TokenHash(code)
	if err != nil {
		return "", ErrNotFound
	}
	var member string
	var revoked sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT member_id, revoked_at FROM qr_member_alias WHERE network_id=? AND alias_hash=?`, network, h[:]).Scan(&member, &revoked); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if revoked.Valid {
		return "", ErrRevoked
	}
	return member, nil
}

// ClaimResult reports the proposal a claim created or replayed. The trade is
// PENDING until the payee accepts through the ordinary flow; only
// State==TxCompleted means it posted.
type ClaimResult struct {
	TransactionID string
	State         TxState
	RequestID     string
	Replayed      bool
}

// ClaimRequest atomically claims a one-time request for the caller and creates
// the ordinary proposal "caller pays recipient". Within ONE write transaction
// it re-reads fresh state, checks membership, expiry and revocation, calls the
// Core adapter's same-transaction Propose, and records the claim. Any failure
// rolls back both the proposal and the claim, so a refused proposal never burns
// the token. key may be empty, in which case a stable key is derived from the
// intent (payer, request, amount, note).
func (s *Service) ClaimRequest(ctx context.Context, p Principal, token, key string) (ClaimResult, error) {
	if !p.valid() {
		return ClaimResult{}, ErrForbidden
	}
	if key != "" {
		if err := validKey(key); err != nil {
			return ClaimResult{}, err
		}
	}
	var out ClaimResult
	err := s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := s.active(ctx, q, p.NetworkID, p.MemberID); err != nil {
			return err
		}
		r, err := s.requestByToken(ctx, q, p.NetworkID, token)
		if err != nil {
			return err
		}
		fp := fingerprint("claim.request", p.NetworkID, p.MemberID, r.id, r.recipient, itoa(r.amount), r.note)
		if key == "" {
			key = "intent-" + hex.EncodeToString(fp[:16])
		}
		now := s.now()
		prior, ok, err := s.idem(ctx, q, p, "claim.request", key, fp)
		if err != nil {
			return err
		}
		if ok {
			return s.replay(ctx, q, p, r.network, prior, r.id, &out)
		}
		if r.claimedTx.Valid {
			if r.claimedBy.String == p.MemberID && hmac.Equal(r.claimFP, fp) { // same intent, new key
				return s.replay(ctx, q, p, r.network, r.claimedTx.String, r.id, &out)
			}
			return ErrClaimed
		}
		switch {
		case r.revoked.Valid:
			return ErrRevoked
		case !now.Before(fromMS(r.expires)):
			return ErrExpired
		case r.recipient == p.MemberID:
			return fmt.Errorf("%w: cannot pay yourself", ErrInvalid)
		}
		payee, err := s.core.Member(ctx, q, p.NetworkID, r.recipient)
		if err != nil {
			return err
		}
		if !payee.Active {
			return ErrCounterpartyInactive
		}
		res, err := s.core.Propose(ctx, q, ProposalInput{NetworkID: p.NetworkID, PayerID: p.MemberID, PayeeID: r.recipient,
			AmountMinor: r.amount, Note: r.note, IdempotencyKey: key, Origin: "qr-request:" + r.id})
		if err != nil {
			return err
		}
		if res.TransactionID == "" {
			return errors.New("qr: core returned an empty transaction id")
		}
		upd, err := q.ExecContext(ctx, `UPDATE qr_trade_request SET claimed_tx_id=?, claimed_by=?, claimed_at=?, claim_fingerprint=?
			WHERE id=? AND network_id=? AND claimed_tx_id IS NULL AND revoked_at IS NULL AND expires_at>?`,
			res.TransactionID, p.MemberID, ms(now), fp, r.id, p.NetworkID, ms(now))
		if err != nil {
			return err
		}
		if n, _ := upd.RowsAffected(); n != 1 {
			return ErrClaimed // lost a race inside the transaction: rolls the proposal back too
		}
		if err := s.idemPut(ctx, q, p, "claim.request", key, fp, res.TransactionID, now); err != nil {
			return err
		}
		out = ClaimResult{TransactionID: res.TransactionID, State: res.State, RequestID: r.id}
		return nil
	})
	return out, err
}

func (s *Service) replay(ctx context.Context, q DBTX, p Principal, network, txID, reqID string, out *ClaimResult) error {
	st, err := s.core.TxStatus(ctx, q, network, txID)
	if err != nil {
		return err
	}
	*out = ClaimResult{TransactionID: txID, State: st, RequestID: reqID, Replayed: true}
	return nil
}

// ProposeToAlias creates the ordinary proposal "caller pays alias owner" for a
// payer-chosen amount. Idempotent per (caller, key): a retry with the same
// intent returns the first proposal; changing amount, note or alias under the
// same key is ErrIdempotencyConflict.
func (s *Service) ProposeToAlias(ctx context.Context, p Principal, code string, amount int64, note, key string) (ClaimResult, error) {
	if !p.valid() {
		return ClaimResult{}, ErrForbidden
	}
	if err := validKey(key); err != nil {
		return ClaimResult{}, err
	}
	if err := s.checkAmount(amount); err != nil {
		return ClaimResult{}, err
	}
	note, err := cleanNote(note, s.cfg.MaxNoteRunes)
	if err != nil {
		return ClaimResult{}, err
	}
	h, err := TokenHash(code)
	if err != nil {
		return ClaimResult{}, ErrNotFound
	}
	fp := fingerprint("claim.alias", p.NetworkID, p.MemberID, string(h[:]), itoa(amount), note)
	var out ClaimResult
	err = s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		if _, err := s.active(ctx, q, p.NetworkID, p.MemberID); err != nil {
			return err
		}
		prior, ok, err := s.idem(ctx, q, p, "claim.alias", key, fp)
		if err != nil {
			return err
		}
		if ok {
			return s.replay(ctx, q, p, p.NetworkID, prior, "", &out)
		}
		recipient, err := s.aliasOwner(ctx, q, p.NetworkID, code)
		if err != nil {
			return err
		}
		if recipient == p.MemberID {
			return fmt.Errorf("%w: cannot pay yourself", ErrInvalid)
		}
		payee, err := s.core.Member(ctx, q, p.NetworkID, recipient)
		if err != nil {
			return err
		}
		if !payee.Active {
			return ErrCounterpartyInactive
		}
		res, err := s.core.Propose(ctx, q, ProposalInput{NetworkID: p.NetworkID, PayerID: p.MemberID, PayeeID: recipient,
			AmountMinor: amount, Note: note, IdempotencyKey: key, Origin: "qr-alias"})
		if err != nil {
			return err
		}
		if res.TransactionID == "" {
			return errors.New("qr: core returned an empty transaction id")
		}
		if err := s.idemPut(ctx, q, p, "claim.alias", key, fp, res.TransactionID, s.now()); err != nil {
			return err
		}
		out = ClaimResult{TransactionID: res.TransactionID, State: res.State}
		return nil
	})
	return out, err
}

// RevokeRequest revokes an unclaimed request owned by the caller. Revoking is
// idempotent. A claimed request cannot be revoked (ErrClaimed): its proposal
// follows the ordinary decline/withdraw flow. Requests of other members or
// networks are ErrNotFound.
func (s *Service) RevokeRequest(ctx context.Context, p Principal, id string) (RequestView, error) {
	if !p.valid() {
		return RequestView{}, ErrForbidden
	}
	if !idRE.MatchString(id) {
		return RequestView{}, ErrNotFound
	}
	var out RequestView
	err := s.store.Update(ctx, func(ctx context.Context, q DBTX) error {
		var r reqRow
		if err := r.scan(q.QueryRowContext(ctx, `SELECT `+reqCols+` FROM qr_trade_request WHERE network_id=? AND id=? AND recipient_id=?`, p.NetworkID, id, p.MemberID)); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		if r.claimedTx.Valid {
			return ErrClaimed
		}
		now := s.now()
		if !r.revoked.Valid {
			if _, err := q.ExecContext(ctx, `UPDATE qr_trade_request SET revoked_at=? WHERE id=? AND claimed_tx_id IS NULL AND revoked_at IS NULL`, ms(now), id); err != nil {
				return err
			}
			r.revoked = sql.NullInt64{Int64: ms(now), Valid: true}
		}
		var err error
		out, err = s.view(ctx, q, &r, now)
		return err
	})
	return out, err
}

// GetRequest returns a request to its recipient or to the member who claimed
// it. Everyone else (other members, other networks) gets ErrNotFound.
func (s *Service) GetRequest(ctx context.Context, p Principal, id string) (RequestView, error) {
	if !p.valid() {
		return RequestView{}, ErrForbidden
	}
	if !idRE.MatchString(id) {
		return RequestView{}, ErrNotFound
	}
	var out RequestView
	err := s.store.View(ctx, func(ctx context.Context, q DBTX) error {
		var r reqRow
		if err := r.scan(q.QueryRowContext(ctx, `SELECT `+reqCols+` FROM qr_trade_request WHERE network_id=? AND id=? AND (recipient_id=? OR claimed_by=?)`,
			p.NetworkID, id, p.MemberID, p.MemberID)); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var err error
		out, err = s.view(ctx, q, &r, s.now())
		return err
	})
	return out, err
}

// ListRequests pages the caller's own requests, newest first, with an opaque
// keyset cursor. limit is clamped to 1..100.
func (s *Service) ListRequests(ctx context.Context, p Principal, limit int, cursor string) ([]RequestView, string, error) {
	if !p.valid() {
		return nil, "", ErrForbidden
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	curMS, curID := int64(1<<62), "\xff"
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		parts := strings.SplitN(string(raw), ".", 2)
		if err != nil || len(parts) != 2 || !idRE.MatchString(parts[1]) {
			return nil, "", fmt.Errorf("%w: bad cursor", ErrInvalid)
		}
		var n int64
		if _, err := fmt.Sscanf(parts[0], "%d", &n); err != nil || n < 0 || itoa(n) != parts[0] {
			return nil, "", fmt.Errorf("%w: bad cursor", ErrInvalid)
		}
		curMS, curID = n, parts[1]
	}
	var out []RequestView
	var next string
	err := s.store.View(ctx, func(ctx context.Context, q DBTX) error {
		rows, err := q.QueryContext(ctx, `SELECT `+reqCols+` FROM qr_trade_request WHERE network_id=? AND recipient_id=? AND (created_at<? OR (created_at=? AND id<?))
			ORDER BY created_at DESC, id DESC LIMIT ?`, p.NetworkID, p.MemberID, curMS, curMS, curID, limit+1)
		if err != nil {
			return err
		}
		var rs []*reqRow
		for rows.Next() {
			var r reqRow
			if err := r.scan(rows); err != nil {
				rows.Close()
				return err
			}
			rs = append(rs, &r)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		more := len(rs) > limit
		if more {
			rs = rs[:limit]
		}
		now := s.now()
		for _, r := range rs {
			v, err := s.view(ctx, q, r, now)
			if err != nil {
				return err
			}
			out = append(out, v)
		}
		if more {
			last := rs[len(rs)-1]
			next = base64.RawURLEncoding.EncodeToString([]byte(itoa(last.created) + "." + last.id))
		}
		return nil
	})
	return out, next, err
}

// RenderAlias returns the printable PNG of the caller's live alias.
func (s *Service) RenderAlias(ctx context.Context, p Principal) ([]byte, AliasInfo, error) {
	a, err := s.MyAlias(ctx, p)
	if err != nil {
		return nil, a, err
	}
	b, err := RenderPNG(a.URL, ECHigh, s.cfg.PNGPixels)
	return b, a, err
}

// RenderURL renders a link previously returned by this service. It refuses
// anything that does not parse as a trusted-origin v1 link.
func (s *Service) RenderURL(u string) ([]byte, error) {
	if _, err := ParseLink(s.cfg.Origin, u); err != nil {
		return nil, ErrInvalid
	}
	return RenderPNG(u, ECHigh, s.cfg.PNGPixels)
}
