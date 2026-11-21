package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed" // commonpw.txt
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
	"unicode/utf8"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
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

func hashPassword(ctx context.Context, work *passwordwork.Ticket, pw string, p Argon2Params) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key, err := argon2id(ctx, work, []byte(pw), salt, p)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.MemoryKiB, p.Time,
		p.Threads, phc64.EncodeToString(salt), phc64.EncodeToString(key)), nil
}

// argonObserver, when set, is told the parameters of every Argon2id
// evaluation. Only tests set it (see ObserveArgon2 in the package's internal
// tests), to prove that two code paths do the same hashing work (SEC-005,
// BR-05) without relying on wall-clock time.
var argonObserver atomic.Pointer[func(Argon2Params)]

// argon2id is the service's single Argon2id boundary. Backup passphrase
// derivation uses the same process admission through internal/passwordwork.
func argon2id(ctx context.Context, work *passwordwork.Ticket, pw, salt []byte, p Argon2Params) ([]byte, error) {
	return work.Run(ctx, func() []byte {
		if f := argonObserver.Load(); f != nil {
			(*f)(p)
		}
		if ctx.Err() != nil {
			return nil
		}
		return argon2.IDKey(pw, salt, p.Time, p.MemoryKiB, p.Threads, p.KeyLen)
	})
}

// parsePHC decodes an Argon2id PHC string. ok is false for anything that is
// not a usable hash: another algorithm, a damaged row, the erased marker, or
// parameters outside the bounds below.
func parsePHC(encoded string) (p Argon2Params, salt, want []byte, ok bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, false
	}
	var ver int
	var th uint
	if _, err := fmt.Sscanf(parts[2], "v=%d", &ver); err != nil || ver != argon2.Version {
		return p, nil, nil, false
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Time, &th); err != nil {
		return p, nil, nil, false
	}
	// argon2.IDKey panics when t < 1; cap t so a damaged row cannot stall a login.
	if th == 0 || th > 255 || p.Time == 0 || p.Time > 64 {
		return p, nil, nil, false
	}
	p.Threads = uint8(th) //nolint:gosec // bounded above
	salt, err1 := phc64.DecodeString(parts[4])
	want, err2 := phc64.DecodeString(parts[5])
	if err1 != nil || err2 != nil || len(want) == 0 || p.Threads == 0 || p.MemoryKiB > 1<<21 {
		return p, nil, nil, false
	}
	p.KeyLen, p.SaltLen = uint32(len(want)), uint32(len(salt)) //nolint:gosec // short decoded values
	return p, salt, want, true
}

// verifyPassword checks pw against a PHC string using the parameters stored in
// the hash, in constant time. It does no hashing work for an unusable hash;
// sign-in uses Service.verifyLoginPassword, which does.
func verifyPassword(ctx context.Context, work *passwordwork.Ticket, pw, encoded string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	p, salt, want, ok := parsePHC(encoded)
	if !ok {
		return false, nil
	}
	got, err := argon2id(ctx, work, []byte(pw), salt, p)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// verifyLoginPassword is verifyPassword for sign-in. A stored hash that cannot
// be evaluated (an erased person's marker, a damaged row) still costs one
// Argon2id evaluation with the current parameters, exactly like an unknown
// login, so response time does not single such an account out (SEC-005, BR-05).
func (s *Service) verifyLoginPassword(ctx context.Context, work *passwordwork.Ticket, pw, encoded string) (bool, error) {
	if _, _, _, ok := parsePHC(encoded); !ok {
		_, err := verifyPassword(ctx, work, pw, s.dummyHash) // equalise effort
		return false, err
	}
	return verifyPassword(ctx, work, pw, encoded)
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
