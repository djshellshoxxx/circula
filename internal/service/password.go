package service

import (
	"crypto/rand"
	"crypto/subtle"
	_ "embed" // commonpw.txt
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"golang.org/x/crypto/argon2"
)

var (
	b64   = base64.RawURLEncoding
	phc64 = base64.RawStdEncoding
)

// Argon2Params are the Argon2id cost parameters, recorded inside every hash
// so they can be raised later without a mass reset (SEC-001).
type Argon2Params struct {
	MemoryKiB uint32
	Time      uint32
	Threads   uint8
	KeyLen    uint32
	SaltLen   uint32
}

// DefaultArgon2Params follows the OWASP minimum (19 MiB, t=2) with headroom.
// DEC-006 requires a benchmark on the minimum supported hardware before release.
func DefaultArgon2Params() Argon2Params {
	return Argon2Params{MemoryKiB: 64 * 1024, Time: 3, Threads: 2, KeyLen: 32, SaltLen: 16}
}

func hashPassword(pw string, p Argon2Params) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, p.Time, p.MemoryKiB, p.Threads, p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.MemoryKiB, p.Time,
		p.Threads, phc64.EncodeToString(salt), phc64.EncodeToString(key)), nil
}

// verifyPassword checks pw against a PHC string using the parameters stored in
// the hash, in constant time.
func verifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var ver int
	var p Argon2Params
	var th uint
	if _, err := fmt.Sscanf(parts[2], "v=%d", &ver); err != nil || ver != argon2.Version {
		return false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Time, &th); err != nil {
		return false
	}
	// argon2.IDKey panics when t < 1; cap t so a damaged row cannot stall a login.
	if th == 0 || th > 255 || p.Time == 0 || p.Time > 64 {
		return false
	}
	p.Threads = uint8(th) //nolint:gosec // bounded above
	salt, err1 := phc64.DecodeString(parts[4])
	want, err2 := phc64.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 || p.Threads == 0 || p.MemoryKiB > 1<<21 {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, p.Time, p.MemoryKiB, p.Threads, uint32(len(want))) //nolint:gosec // want is a short decoded hash
	return subtle.ConstantTimeCompare(got, want) == 1
}

//go:embed commonpw.txt
var commonPasswordsRaw string

var (
	commonOnce sync.Once
	commonSet  map[string]struct{}
)

func commonPasswords() map[string]struct{} {
	commonOnce.Do(func() {
		commonSet = map[string]struct{}{}
		for _, l := range strings.Split(commonPasswordsRaw, "\n") {
			if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
				commonSet[l] = struct{}{}
			}
		}
	})
	return commonSet
}

var unleet = strings.NewReplacer("@", "a", "0", "o", "1", "i", "3", "e", "$", "s", "5", "s", "7", "t", "!", "i")

// commonPassword reports whether pw is, ignoring case, a listed password,
// optionally spelled with look-alike characters (p@ssw0rd) and/or followed by
// digits and punctuation (password2025!).
func commonPassword(pw string) bool {
	set := commonPasswords()
	lower := strings.ToLower(strings.TrimSpace(pw))
	for _, c := range []string{lower, unleet.Replace(lower)} {
		if _, ok := set[c]; ok {
			return true
		}
		stem := strings.TrimRightFunc(c, func(r rune) bool { return !unicode.IsLetter(r) })
		if len(stem) >= 4 {
			if _, ok := set[stem]; ok {
				return true
			}
		}
	}
	return false
}

// derivedPassword reports whether pw contains one of the related strings
// (the login, its local part) of at least four characters, ignoring case and
// look-alike spellings: a password built from the account name.
func derivedPassword(pw string, related []string) bool {
	lower := strings.ToLower(pw)
	plain := unleet.Replace(lower)
	for _, r := range related {
		r = strings.ToLower(strings.TrimSpace(r))
		cands := []string{r}
		if at := strings.IndexByte(r, '@'); at > 0 {
			r = r[:at]
			cands = append(cands, r)
		}
		cands = append(cands, strings.FieldsFunc(r, func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsDigit(c) })...)
		for _, c := range cands {
			if utf8.RuneCountInString(c) >= 4 && (strings.Contains(lower, c) || strings.Contains(plain, c)) {
				return true
			}
		}
	}
	return false
}

// validatePassword enforces SEC-002: minimum length, at least 64 accepted, no
// composition rules, plus a refusal of common passwords (embedded list) and of
// passwords derived from related strings such as the login. Config.PasswordDenied
// may add a stricter check.
func (s *Service) validatePassword(pw string, related ...string) error {
	n := utf8.RuneCountInString(pw)
	switch {
	case n < s.cfg.MinPasswordLen:
		return domain.Invalid("password", fmt.Sprintf("must be at least %d characters", s.cfg.MinPasswordLen))
	case len(pw) > 1024:
		return domain.Invalid("password", "too long")
	case commonPassword(pw):
		return domain.Invalid("password", "is too common; choose another")
	case derivedPassword(pw, related):
		return domain.Invalid("password", "must not contain your login; choose another")
	case s.cfg.PasswordDenied != nil && s.cfg.PasswordDenied(pw):
		return domain.Invalid("password", "is too common; choose another")
	}
	return nil
}
