// Package smtpsend adapts github.com/wneessen/go-mail (MIT, pinned in go.mod)
// to notifications.EmailSender. It sends fixed generic content over SMTP with
// mandatory, verified TLS. The SMTP host, credentials and From address are
// operator configuration; a member can never choose any of them.
package smtpsend

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	gomail "github.com/wneessen/go-mail"

	"github.com/djshellshoxxx/circula/future/notifications"
)

// TLSMode selects how TLS is established. There is no plaintext mode.
type TLSMode string

const (
	// StartTLS connects in clear and requires the STARTTLS upgrade (default,
	// port 587). A server that does not offer it makes the send fail; the
	// client never falls back to plaintext.
	StartTLS TLSMode = "starttls"
	// Implicit uses TLS from the first byte (port 465).
	Implicit TLSMode = "implicit"
)

// Config is the operator's SMTP configuration.
type Config struct {
	Host     string
	Port     int // default 587 (StartTLS) or 465 (Implicit)
	TLS      TLSMode
	From     string // bare address, e.g. notifications@circula.example
	Username string
	Password notifications.Secret
	// ServerName overrides the name verified in the certificate (default Host).
	ServerName string
	// RootCAs pins the trust roots (default: system roots).
	RootCAs *x509.CertPool
	// Timeout bounds connect, TLS and each SMTP exchange (default 20s).
	Timeout time.Duration
	// HELO overrides the EHLO name.
	HELO string
	// DialContext, if set, replaces the TCP dialer (tests, egress policy).
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

// Sender implements notifications.EmailSender.
type Sender struct{ cfg Config }

// New validates cfg.
func New(cfg Config) (*Sender, error) {
	if cfg.Host == "" || strings.ContainsAny(cfg.Host, " \r\n/@") {
		return nil, errors.New("smtpsend: host required")
	}
	if cfg.TLS == "" {
		cfg.TLS = StartTLS
	}
	if cfg.TLS != StartTLS && cfg.TLS != Implicit {
		return nil, errors.New("smtpsend: TLS must be starttls or implicit")
	}
	if cfg.Port == 0 {
		cfg.Port = 587
		if cfg.TLS == Implicit {
			cfg.Port = 465
		}
	}
	a, err := mail.ParseAddress(cfg.From)
	if err != nil || a.Name != "" || a.Address != cfg.From {
		return nil, errors.New("smtpsend: From must be a bare address")
	}
	if (cfg.Username == "") != !cfg.Password.Set() {
		return nil, errors.New("smtpsend: username and password go together")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 20 * time.Second
	}
	return &Sender{cfg: cfg}, nil
}

func (s *Sender) client() (*gomail.Client, error) {
	c := s.cfg
	name := c.ServerName
	if name == "" {
		name = c.Host
	}
	tc := &tls.Config{ServerName: name, MinVersion: tls.VersionTLS12, RootCAs: c.RootCAs}
	opts := []gomail.Option{
		gomail.WithPort(c.Port),
		gomail.WithTimeout(c.Timeout),
		gomail.WithTLSConfig(tc),
	}
	if c.TLS == Implicit {
		opts = append(opts, gomail.WithSSL())
	} else {
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if c.HELO != "" {
		opts = append(opts, gomail.WithHELO(c.HELO))
	}
	if c.Username != "" {
		opts = append(opts, gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover), gomail.WithUsername(c.Username), gomail.WithPassword(c.Password.Reveal()))
	}
	if c.DialContext != nil {
		opts = append(opts, gomail.WithDialContextFunc(func(ctx context.Context, network, address string) (net.Conn, error) {
			return c.DialContext(ctx, network, address)
		}))
	}
	return gomail.NewClient(c.Host, opts...)
}

// SendEmail builds a fresh immutable message per call (never reused) and sends
// it. The returned error is always a *notifications.SendError.
func (s *Sender) SendEmail(ctx context.Context, m notifications.EmailMessage) error {
	if _, err := mail.ParseAddress(m.To); err != nil || strings.ContainsAny(m.To, "\r\n") {
		return notifications.NewSendError(notifications.ClassTargetGone, "bad_recipient", 0, nil)
	}
	msg := gomail.NewMsg(gomail.WithNoDefaultUserAgent())
	if err := msg.From(s.cfg.From); err != nil {
		return notifications.NewSendError(notifications.ClassPermanent, "bad_sender", 0, err)
	}
	if err := msg.To(m.To); err != nil {
		return notifications.NewSendError(notifications.ClassTargetGone, "bad_recipient", 0, err)
	}
	if strings.ContainsAny(m.Subject, "\r\n") || strings.ContainsAny(m.MessageID, "\r\n<> ") {
		return notifications.NewSendError(notifications.ClassPermanent, "bad_header", 0, nil)
	}
	msg.Subject(m.Subject)
	msg.SetDate()
	if m.MessageID != "" {
		msg.SetMessageIDWithValue(m.MessageID)
	} else {
		msg.SetMessageID()
	}
	msg.SetBodyString(gomail.TypeTextPlain, m.Body)
	if m.UnsubscribeURL != "" {
		if err := msg.SetListUnsubscribeOneClick(m.UnsubscribeURL); err != nil {
			return notifications.NewSendError(notifications.ClassPermanent, "bad_unsubscribe_url", 0, err)
		}
	}
	cl, err := s.client()
	if err != nil {
		return notifications.NewSendError(notifications.ClassPermanent, "bad_config", 0, err)
	}
	if err := cl.DialAndSendWithContext(ctx, msg); err != nil {
		return classify(ctx, err)
	}
	return nil
}

func classify(ctx context.Context, err error) *notifications.SendError {
	if ctx.Err() != nil {
		return notifications.NewSendError(notifications.ClassTransient, "canceled", 0, ctx.Err())
	}
	var se *gomail.SendError
	if errors.As(err, &se) {
		switch {
		case se.IsTemp():
			return notifications.NewSendError(notifications.ClassTransient, fmt.Sprintf("smtp_%dxx", se.ErrorCode()/100), 0, err)
		case se.Reason == gomail.ErrSMTPRcptTo:
			return notifications.NewSendError(notifications.ClassTargetGone, "recipient_rejected", 0, err)
		default:
			return notifications.NewSendError(notifications.ClassPermanent, "smtp_5xx", 0, err)
		}
	}
	var tlsErr *tls.CertificateVerificationError
	var unk x509.UnknownAuthorityError
	var host x509.HostnameError
	var inv x509.CertificateInvalidError
	var rh tls.RecordHeaderError
	if errors.As(err, &tlsErr) || errors.As(err, &unk) || errors.As(err, &host) || errors.As(err, &inv) || errors.As(err, &rh) {
		return notifications.NewSendError(notifications.ClassTransient, "tls_verify", 0, err)
	}
	var te *textproto.Error
	if errors.As(err, &te) {
		if te.Code == 535 || te.Code == 534 || te.Code == 530 {
			return notifications.NewSendError(notifications.ClassTransient, "auth", 0, err)
		}
		if te.Code >= 400 && te.Code < 500 {
			return notifications.NewSendError(notifications.ClassTransient, "smtp_4xx", 0, err)
		}
		return notifications.NewSendError(notifications.ClassPermanent, "smtp_5xx", 0, err)
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "STARTTLS") || strings.Contains(msg, "tls:") || strings.Contains(msg, "x509:"):
		return notifications.NewSendError(notifications.ClassTransient, "tls", 0, err)
	case strings.Contains(msg, "auth") || strings.Contains(msg, "AUTH"):
		return notifications.NewSendError(notifications.ClassTransient, "auth", 0, err)
	}
	return notifications.NewSendError(notifications.ClassTransient, "network", 0, err)
}
