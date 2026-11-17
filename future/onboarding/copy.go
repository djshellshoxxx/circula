package onboarding

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Bounds for administrator-supplied copy.
const (
	MaxIntroRunes = 1000
	MaxURLBytes   = 512
)

// Copy is per-network, per-curriculum administrator copy. It is plain text and
// two trusted links. It can never alter credit rules, steps or limits.
type Copy struct {
	NetworkID     string
	Version       int
	Intro         string
	GovernanceURL string
	TermsURL      string
	Revision      int64
	UpdatedBy     string
	UpdatedAt     time.Time
}

// LinkPolicy states where admin links may point. Relative links must start with
// one of PathPrefixes; absolute links must be https on one of Origins' hosts.
// A zero policy rejects every link.
type LinkPolicy struct {
	PathPrefixes []string // e.g. "/governance/", "/terms"
	Hosts        []string // lowercase hostnames (no port) allowed for https links
}

func (p LinkPolicy) validate() error {
	for _, pre := range p.PathPrefixes {
		if !strings.HasPrefix(pre, "/") || strings.HasPrefix(pre, "//") || strings.ContainsAny(pre, "\\?#") {
			return fmt.Errorf("onboarding: bad link path prefix %q", pre)
		}
	}
	for _, h := range p.Hosts {
		if h == "" || h != strings.ToLower(h) || strings.ContainsAny(h, "/:@ ?#") {
			return fmt.Errorf("onboarding: bad link host %q", h)
		}
	}
	return nil
}

var errLink = errors.New("link is not allowed by this network's link policy")

// Check validates one link. Empty is allowed (link not configured).
func (p LinkPolicy) Check(raw string) error {
	if raw == "" {
		return nil
	}
	if len(raw) > MaxURLBytes {
		return fmt.Errorf("link too long")
	}
	for _, r := range raw {
		if r <= 0x20 || r == 0x7f || r == '\\' || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return errLink
		}
	}
	if strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") {
			return errLink
		}
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "" || u.Host != "" {
			return errLink
		}
		clean := u.Path
		if strings.Contains(clean, "/../") || strings.HasSuffix(clean, "/..") || strings.Contains(clean, "/./") || strings.Contains(strings.ToLower(raw), "%2e") || strings.Contains(strings.ToLower(raw), "%2f") || strings.Contains(strings.ToLower(raw), "%5c") {
			return errLink
		}
		for _, pre := range p.PathPrefixes {
			if clean == strings.TrimSuffix(pre, "/") || strings.HasPrefix(clean, pre) {
				return nil
			}
		}
		return errLink
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Opaque != "" || u.Host == "" || u.Port() != "" {
		return errLink
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range p.Hosts {
		if host == h {
			return nil
		}
	}
	return errLink
}

// ValidateText checks admin plain text: bounded, valid UTF-8, no control or
// format characters other than newline. HTML is not interpreted anywhere; the
// renderer escapes everything.
func ValidateText(s string, maxRunes int) error {
	if !utf8.ValidString(s) {
		return errors.New("text is not valid UTF-8")
	}
	if utf8.RuneCountInString(s) > maxRunes {
		return fmt.Errorf("text longer than %d characters", maxRunes)
	}
	for _, r := range s {
		if r == '\n' {
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || r == ' ' || r == ' ' {
			return errors.New("text contains control characters")
		}
	}
	return nil
}

// Validate checks c against policy.
func (c Copy) Validate(p LinkPolicy) error {
	if !validID(c.NetworkID) {
		return ErrInvalidIdentifier
	}
	if c.Version < 1 {
		return errors.New("copy version required")
	}
	if err := ValidateText(c.Intro, MaxIntroRunes); err != nil {
		return fmt.Errorf("intro: %w", err)
	}
	if err := p.Check(c.GovernanceURL); err != nil {
		return fmt.Errorf("governance link: %w", err)
	}
	if err := p.Check(c.TermsURL); err != nil {
		return fmt.Errorf("terms link: %w", err)
	}
	return nil
}

// Copy returns the network's copy for the tracker's curriculum, or a zero Copy
// (no custom text) when unset. Any active member may read it.
func (t *Tracker) Copy(ctx context.Context, a Actor) (Copy, error) {
	if _, err := a.key(); err != nil {
		return Copy{}, err
	}
	c, ok, err := t.store.LoadCopy(ctx, a.NetworkID, t.version)
	if err != nil || !ok {
		return Copy{NetworkID: a.NetworkID, Version: t.version}, err
	}
	// Re-validate on load: a row written under an older policy must not
	// produce a link the current policy rejects.
	if err := c.Validate(t.links); err != nil {
		return Copy{NetworkID: a.NetworkID, Version: t.version, Revision: c.Revision}, nil
	}
	return c, nil
}

// SaveCopy lets a network admin set the copy for their own network. The network
// and version are taken from the admin Actor and the tracker, never from c.
// expectedRevision is the Revision the form was rendered with (0 = none).
func (t *Tracker) SaveCopy(ctx context.Context, admin Actor, c Copy, expectedRevision int64) (Copy, error) {
	if _, err := admin.key(); err != nil {
		return Copy{}, err
	}
	if !admin.Admin {
		return Copy{}, ErrForbidden
	}
	c.NetworkID, c.Version, c.UpdatedBy = admin.NetworkID, t.version, admin.MemberID
	c.Intro = strings.ReplaceAll(c.Intro, "\r\n", "\n")
	if err := c.Validate(t.links); err != nil {
		return Copy{}, err
	}
	return t.store.SaveCopy(ctx, c, expectedRevision, t.now())
}
