// Package service is the ONLY write path to Circula state (ARC-002). HTTP,
// HTML and CLI adapters call it with an authenticated domain.Actor and hold
// no business rules. Every mutating method runs in one store.Update unit that
// also writes the audit event (LED-6) and, where retryable, the idempotency
// record (LED-4).
package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/google/uuid"
)

// Config carries injected dependencies and tunables.
type Config struct {
	Clock domain.Clock // default: wall clock
	IDs   domain.IDGen // default: UUIDv7

	Argon          Argon2Params      // password hashing cost (DEC-006 pending benchmark)
	MinPasswordLen int               // default 12 (SEC-002)
	PasswordDenied func(string) bool // optional breached-password check
	SessionIdle    time.Duration     // default 30 days (DAT section 9)
	SessionMax     time.Duration     // default 90 days
	IdempotencyTTL time.Duration     // default 7 days (DAT section 9)

	// Abuse limits on open proposals (SA-08). A proposal is "open" while it is
	// proposed and unexpired. Zero selects the default; negative disables.
	MaxOpenRequestsPerPair    int // requests (proposer is payee) from one member to one payer; default 5
	MaxOpenProposalsPerMember int // all open proposals created by one member; default 200

	// ExportsPerHour caps Export calls per member per network per rolling hour
	// (SA-11). Zero selects 30; negative disables. System actors are exempt.
	ExportsPerHour int

	// SecretKey, when non-empty, seals TOTP seeds at rest with AES-256-GCM
	// (key = SHA-256 of a domain label and SecretKey). Without it seeds are
	// stored base32-encoded (SEC-007). Changing or losing the key makes sealed
	// seeds unusable: affected people must re-enrol.
	SecretKey []byte
	// RequireMFAForAdmins makes every administrator-only operation fail with
	// forbidden "mfa-enrolment-required" until that person has enabled a second factor.
	RequireMFAForAdmins bool
	// PendingAdjustmentTTL is how long a dual-control adjustment request waits
	// for a second administrator (SEC-011). Zero selects 7 days.
	PendingAdjustmentTTL time.Duration
	// NewAdminApprovalDelay is how long after being granted the administrator
	// role a member may not yet approve or reject a dual-control adjustment
	// (A3-01). Zero selects 24 hours; negative disables the wait.
	NewAdminApprovalDelay time.Duration
	// OnMemberErased, when set, runs after a member's anonymisation committed
	// (SA2-17). Adapters use it to drop what they keep in memory for that
	// person: export artefacts and cached idempotent responses. personID is
	// the person behind the erased membership.
	OnMemberErased func(netID, personID string)
}

// Service implements all use cases. It is safe for concurrent use.
type Service struct {
	st        store.Store
	cfg       Config
	dummyHash string // for constant-effort login of unknown users (SEC-005)
}

// New builds a Service over st, filling defaults.
func New(st store.Store, cfg Config) (*Service, error) {
	if cfg.Clock == nil {
		cfg.Clock = wallClock{}
	}
	if cfg.IDs == nil {
		cfg.IDs = uuidV7{}
	}
	if cfg.Argon == (Argon2Params{}) {
		cfg.Argon = DefaultArgon2Params()
	}
	if cfg.MinPasswordLen == 0 {
		cfg.MinPasswordLen = 12
	}
	if cfg.SessionIdle == 0 {
		cfg.SessionIdle = 30 * 24 * time.Hour
	}
	if cfg.SessionMax == 0 {
		cfg.SessionMax = 90 * 24 * time.Hour
	}
	if cfg.IdempotencyTTL == 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.MaxOpenRequestsPerPair == 0 {
		cfg.MaxOpenRequestsPerPair = 5
	}
	if cfg.MaxOpenProposalsPerMember == 0 {
		cfg.MaxOpenProposalsPerMember = 200
	}
	if cfg.ExportsPerHour == 0 {
		cfg.ExportsPerHour = 30
	}
	if cfg.PendingAdjustmentTTL <= 0 {
		cfg.PendingAdjustmentTTL = 7 * 24 * time.Hour
	}
	if cfg.NewAdminApprovalDelay == 0 {
		cfg.NewAdminApprovalDelay = DefaultNewAdminApprovalDelay
	}
	s := &Service{st: st, cfg: cfg}
	h, err := hashPassword("dummy-password-for-timing", cfg.Argon)
	if err != nil {
		return nil, err
	}
	s.dummyHash = h
	return s, nil
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }

type uuidV7 struct{}

func (uuidV7) New() string {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("uuid: %v", err)) // entropy source failure is unrecoverable
	}
	return id.String()
}

// now returns the injected time truncated to the stored millisecond precision.
func (s *Service) now() time.Time { return s.cfg.Clock.Now().UTC().Truncate(time.Millisecond) }

func (s *Service) id() string { return s.cfg.IDs.New() }

// randToken returns n random bytes as base64url (no padding).
func randToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b64.EncodeToString(b), nil
}

// update and view are thin wrappers so call sites read uniformly.
func (s *Service) update(ctx context.Context, fn func(store.Tx) error) error {
	return s.st.Update(ctx, fn)
}

func (s *Service) view(ctx context.Context, fn func(store.Tx) error) error {
	return s.st.View(ctx, fn)
}

// Page is a keyset pagination request: items strictly after AfterID in the
// listing's documented order. Limit 0 means the default (100); max 500.
type Page struct {
	AfterID string
	Limit   int
}
