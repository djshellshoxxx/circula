// Package pushsend adapts github.com/SherClockHolmes/webpush-go (MIT, pinned
// in go.mod) to notifications.PushSender. webpush-go provides RFC 8291
// encryption and RFC 8292 VAPID; this package adds everything Circula must own:
// the provider-host policy, the SSRF-safe HTTP client (injected into
// webpush.Options.HTTPClient so upstream's unconstrained default client is
// never used), response classification, bounded body reads and secret hygiene.
package pushsend

import (
	"context"
	"crypto/ecdh"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/djshellshoxxx/circula/future/notifications"
)

// MaxPayload bounds the plaintext handed to encryption. The fixed Circula
// payload is far smaller; the cap exists so no caller can stuff content in.
const MaxPayload = 512

// Config is the operator's Web Push configuration.
type Config struct {
	VAPIDPublic  string
	VAPIDPrivate notifications.Secret
	// Subject is the VAPID contact: "mailto:ops@example.org" or an https URL.
	Subject string
	// Policy is the provider-host allowlist applied again at send time.
	Policy notifications.EndpointPolicy
	// Client must be the SSRF-safe client from notifications.Egress.Client().
	Client webpush.HTTPClient
	// AuthScheme selects "vapid" (default) or "webpush"; set it explicitly to
	// what the tested provider matrix requires.
	AuthScheme webpush.AuthScheme
}

// Sender implements notifications.PushSender.
type Sender struct{ cfg Config }

// New validates the VAPID key pair so a bad secret fails at start-up.
func New(cfg Config) (*Sender, error) {
	if cfg.Client == nil {
		return nil, errors.New("pushsend: an SSRF-safe HTTP client is required")
	}
	if !(strings.HasPrefix(cfg.Subject, "mailto:") || strings.HasPrefix(cfg.Subject, "https://")) {
		return nil, errors.New("pushsend: subject must be mailto: or https:")
	}
	priv, err := decodeKey(cfg.VAPIDPrivate.Reveal())
	if err != nil || len(priv) != 32 {
		return nil, errors.New("pushsend: VAPID private key must be 32 bytes, base64url")
	}
	pub, err := decodeKey(cfg.VAPIDPublic)
	if err != nil || len(pub) != 65 {
		return nil, errors.New("pushsend: VAPID public key must be a 65-byte P-256 point, base64url")
	}
	pk, err := ecdh.P256().NewPrivateKey(priv)
	if err != nil || string(pk.PublicKey().Bytes()) != string(pub) {
		return nil, errors.New("pushsend: VAPID public key does not match the private key")
	}
	return &Sender{cfg: cfg}, nil
}

func decodeKey(v string) ([]byte, error) {
	return decodeB64URL(strings.TrimSpace(v))
}

// SendPush encrypts and posts one message. The returned error is always a
// *notifications.SendError whose text never contains the endpoint or tokens.
func (s *Sender) SendPush(ctx context.Context, m notifications.PushMessage) error {
	if len(m.Payload) == 0 || len(m.Payload) > MaxPayload {
		return notifications.NewSendError(notifications.ClassPermanent, "payload_size", 0, nil)
	}
	if _, err := s.cfg.Policy.Validate(m.Endpoint); err != nil {
		return notifications.NewSendError(notifications.ClassTargetGone, "endpoint_policy", 0, err)
	}
	ttl := int(m.TTL / time.Second)
	if ttl < 60 {
		ttl = 60
	}
	subject := strings.TrimPrefix(s.cfg.Subject, "mailto:") // webpush-go adds mailto: itself
	resp, err := webpush.SendNotificationWithContext(ctx, m.Payload, &webpush.Subscription{
		Endpoint: m.Endpoint,
		Keys:     webpush.Keys{P256dh: m.P256dh, Auth: m.Auth},
	}, &webpush.Options{
		AuthScheme:      s.cfg.AuthScheme,
		HTTPClient:      s.cfg.Client,
		Subscriber:      subject,
		Topic:           m.Topic,
		TTL:             ttl,
		Urgency:         webpush.UrgencyNormal,
		VAPIDPublicKey:  s.cfg.VAPIDPublic,
		VAPIDPrivateKey: s.cfg.VAPIDPrivate.Reveal(),
	})
	if err != nil {
		return classifyErr(ctx, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 2048)) // bounded; provider text is never kept
	return classifyStatus(resp)
}

func classifyErr(ctx context.Context, err error) *notifications.SendError {
	if ctx.Err() != nil {
		return notifications.NewSendError(notifications.ClassTransient, "canceled", 0, ctx.Err())
	}
	if errors.Is(err, notifications.ErrBlockedDestination) {
		return notifications.NewSendError(notifications.ClassPermanent, "egress_blocked", 0, err)
	}
	if errors.Is(err, webpush.ErrMaxPadExceeded) {
		return notifications.NewSendError(notifications.ClassPermanent, "payload_size", 0, err)
	}
	// Key decoding / encryption failures happen before any request is made and
	// are not retryable; network failures are url.Error values.
	msg := err.Error()
	if strings.Contains(msg, "Post ") || strings.Contains(msg, "dial") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509") || strings.Contains(msg, "timeout") || strings.Contains(msg, "EOF") || strings.Contains(msg, "connection") {
		code := "network"
		if strings.Contains(msg, "x509") || strings.Contains(msg, "tls") {
			code = "tls"
		}
		return notifications.NewSendError(notifications.ClassTransient, code, 0, err)
	}
	return notifications.NewSendError(notifications.ClassPermanent, "encrypt", 0, err)
}

func classifyStatus(resp *http.Response) error {
	c := resp.StatusCode
	switch {
	case c >= 200 && c < 300:
		return nil
	case c == http.StatusNotFound || c == http.StatusGone:
		return notifications.NewSendError(notifications.ClassTargetGone, "subscription_gone", 0, nil)
	case c == http.StatusTooManyRequests:
		return notifications.NewSendError(notifications.ClassRateLimited, "rate_limited", retryAfter(resp), nil)
	case c == http.StatusRequestEntityTooLarge:
		return notifications.NewSendError(notifications.ClassPermanent, "payload_too_large", 0, nil)
	case c >= 300 && c < 400:
		return notifications.NewSendError(notifications.ClassPermanent, "redirect_refused", 0, nil)
	case c == http.StatusUnauthorized || c == http.StatusForbidden:
		return notifications.NewSendError(notifications.ClassPermanent, "vapid_rejected", 0, nil)
	case c >= 500:
		return notifications.NewSendError(notifications.ClassTransient, "provider_5xx", retryAfter(resp), nil)
	default:
		return notifications.NewSendError(notifications.ClassPermanent, "provider_4xx", 0, nil)
	}
}

// retryAfter parses seconds or an HTTP date, capped by the dispatcher.
func retryAfter(resp *http.Response) time.Duration {
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	numeric := true
	for _, c := range v {
		if c < '0' || c > '9' {
			numeric = false
			break
		}
	}
	if numeric {
		n, err := strconv.ParseUint(v, 10, 64)
		// Clamp seconds before multiplying: even a valid decimal header can
		// exceed the integer or time.Duration range. Numeric overflow caps at
		// the existing maximum instead of becoming a negative/zero delay.
		if err != nil || n >= uint64(notifications.MaxRetryAfter/time.Second) {
			return notifications.MaxRetryAfter
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return capDur(d)
		}
	}
	return 0
}

func capDur(d time.Duration) time.Duration {
	if d > notifications.MaxRetryAfter {
		return notifications.MaxRetryAfter
	}
	return d
}
