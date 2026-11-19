package notifications

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

// Actor is a TRUSTED snapshot from the host adapter's session/authorization
// layer: who is acting, in which network, from which session. It is never
// parsed from client JSON. Every query this package runs is keyed by
// NetworkID and MemberID, so one tenant or member cannot reach another's rows
// even if a caller bug passed a foreign row id.
type Actor struct {
	NetworkID string
	MemberID  string
	SessionID string // optional; binds push subscriptions to the login session
}

func (a Actor) validate() error {
	for _, v := range []string{a.NetworkID, a.MemberID} {
		if v == "" || len(v) > 128 || !utf8.ValidString(v) || strings.ContainsAny(v, "\x00\r\n") {
			return invalid("actor")
		}
	}
	if len(a.SessionID) > 256 || !utf8.ValidString(a.SessionID) || strings.ContainsAny(a.SessionID, "\x00\r\n") {
		return invalid("actor")
	}
	return nil
}

// Directory is the adapter seam to Circula's authoritative membership and
// session state. It is consulted fresh before every dispatch, so a member who
// left, was suspended or logged out stops receiving notifications even though
// outbox rows were queued earlier. Implementations must fail closed: an error
// means "do not send now".
type Directory interface {
	MemberActive(ctx context.Context, networkID, memberID string) (bool, error)
	// SessionActive reports whether the login session that registered a push
	// subscription is still valid.
	SessionActive(ctx context.Context, networkID, memberID, sessionID string) (bool, error)
}

// Clock is injectable time.
type Clock interface{ Now() time.Time }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// AuditRecord mirrors a notification_audit row for the host's own audit sink.
type AuditRecord struct {
	NetworkID, MemberID, Actor, Action, Subject, Detail string
	At                                                  time.Time
}

// Config wires a Service.
type Config struct {
	Store     *Store
	Secrets   Secrets
	Directory Directory
	Clock     Clock
	Endpoints EndpointPolicy
	// BaseURL is the https origin used in unsubscribe/confirm links, e.g.
	// "https://circula.example". Required for e-mail with links.
	BaseURL string
	Logger  *slog.Logger
	Metrics *Metrics
	// Audit, if set, receives each audit record after its transaction commits.
	Audit AuditHook
	// MaxPushPerMember bounds active subscriptions per member/network (default 10).
	MaxPushPerMember int
}

// AuditHook is called after commit; it must not block for long.
type AuditHook func(ctx context.Context, r AuditRecord)

// Service is the preference, target and outbox API.
type Service struct {
	cfg Config
	st  *Store
	log *slog.Logger
	clk Clock
	met *Metrics
}

// NewService validates cfg. Secrets must validate; SMTP/VAPID are optional.
func NewService(cfg Config) (*Service, error) {
	if cfg.Store == nil || cfg.Directory == nil {
		return nil, errors.New("notifications: store and directory are required")
	}
	if err := cfg.Secrets.Validate(); err != nil {
		return nil, err
	}
	if cfg.BaseURL != "" && !strings.HasPrefix(cfg.BaseURL, "https://") {
		return nil, errors.New("notifications: BaseURL must be https")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Clock == nil {
		cfg.Clock = systemClock{}
	}
	if cfg.MaxPushPerMember <= 0 {
		cfg.MaxPushPerMember = 10
	}
	lg := cfg.Logger
	if lg == nil {
		lg = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	m := cfg.Metrics
	if m == nil {
		m = &Metrics{}
	}
	return &Service{cfg: cfg, st: cfg.Store, log: lg, clk: cfg.Clock, met: m}, nil
}

// Metrics returns the in-process counters.
func (s *Service) Metrics() *Metrics { return s.met }

// Store returns the underlying store.
func (s *Service) Store() *Store { return s.st }

func (s *Service) now() time.Time { return s.clk.Now().UTC() }

// audit writes one record inside tx and queues it for the post-commit hook.
type auditBatch struct {
	recs []AuditRecord
}

func (s *Service) addAudit(ctx context.Context, q DBTX, b *auditBatch, network, member, actor, action, subject, detail string) error {
	at := s.now()
	_, err := q.ExecContext(ctx, `INSERT INTO notification_audit(network_id,member_id,actor,action,subject,detail,at) VALUES(?,?,?,?,?,?,?)`,
		network, member, actor, action, subject, detail, at.Unix())
	if err != nil {
		return err
	}
	b.recs = append(b.recs, AuditRecord{network, member, actor, action, subject, detail, at})
	return nil
}

func (s *Service) flushAudit(ctx context.Context, b *auditBatch) {
	if s.cfg.Audit == nil {
		return
	}
	for _, r := range b.recs {
		s.cfg.Audit(ctx, r)
	}
}

func (s *Service) requireActive(ctx context.Context, a Actor) error {
	if err := a.validate(); err != nil {
		return err
	}
	ok, err := s.cfg.Directory.MemberActive(ctx, a.NetworkID, a.MemberID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func actorLabel(a Actor) string { return "member" }
