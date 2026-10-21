package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 default algorithm for authenticator-app interoperability; HMAC-SHA1 is not collision-sensitive
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// RFC 6238 parameters. Authenticator apps assume exactly these defaults.
const (
	totpPeriod   = 30
	totpDigits   = 6
	totpSecretLn = 20 // bytes (160 bits, RFC 4226 recommendation)
	totpSkew     = 1  // accepted steps either side of the current one
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// hotp is RFC 4226 with HMAC-SHA1 and 6 digits.
func hotp(secret []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	m := hmac.New(sha1.New, secret)
	_, _ = m.Write(msg[:])
	sum := m.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, v%1000000)
}

func totpStep(t time.Time) int64 { return t.Unix() / totpPeriod }

func isDigits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// matchTOTP checks code against the steps now-1..now+1 and returns the newest
// matching step that is greater than last (replay guard: a step at or below the
// last accepted one is never accepted again). All candidate steps are compared
// in constant time without early exit.
func matchTOTP(secret []byte, code string, now time.Time, last int64) (int64, bool) {
	if !isDigits(code, totpDigits) {
		return 0, false
	}
	cur := totpStep(now)
	var best int64
	found := 0
	for d := int64(-totpSkew); d <= totpSkew; d++ {
		step := cur + d
		if step < 0 {
			continue
		}
		eq := subtle.ConstantTimeCompare([]byte(hotp(secret, uint64(step))), []byte(code)) //nolint:gosec // step >= 0 checked above
		fresh := 0
		if step > last {
			fresh = 1
		}
		if eq&fresh == 1 && (found == 0 || step > best) {
			best, found = step, 1
		}
	}
	return best, found == 1
}

func newTOTPSecret() ([]byte, error) {
	b := make([]byte, totpSecretLn)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// otpauthURI is the provisioning URI authenticator apps import.
func otpauthURI(login string, secret []byte) string {
	label := url.PathEscape("Circula:" + login)
	q := url.Values{}
	q.Set("secret", b32.EncodeToString(secret))
	q.Set("issuer", "Circula")
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// ---- sealing of the TOTP seed at rest ----

const (
	sealPrefix = "v1:"  // AES-256-GCM, key derived from Config.SecretKey
	rawPrefix  = "raw:" // base32, no key configured
)

func (s *Service) sealKey() []byte {
	if len(s.cfg.SecretKey) == 0 {
		return nil
	}
	h := sha256.Sum256(append([]byte("circula-mfa-seal-v1\x00"), s.cfg.SecretKey...))
	return h[:]
}

// sealSecret protects the seed. With Config.SecretKey it is AES-GCM encrypted
// (the person id is bound as additional data, so a sealed value cannot be moved
// to another person); without it the seed is stored as base32 and protected
// only by database file permissions and backup encryption.
func (s *Service) sealSecret(personID string, secret []byte) (string, error) {
	key := s.sealKey()
	if key == nil {
		return rawPrefix + b32.EncodeToString(secret), nil
	}
	blk, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	g, err := cipher.NewGCM(blk)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := g.Seal(nonce, nonce, secret, []byte(personID))
	return sealPrefix + base64.RawStdEncoding.EncodeToString(ct), nil
}

func (s *Service) openSecret(personID, stored string) ([]byte, error) {
	switch {
	case strings.HasPrefix(stored, rawPrefix):
		return b32.DecodeString(strings.TrimPrefix(stored, rawPrefix))
	case strings.HasPrefix(stored, sealPrefix):
		key := s.sealKey()
		if key == nil {
			return nil, errors.New("mfa: secret is sealed but no SecretKey is configured")
		}
		ct, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, sealPrefix))
		if err != nil {
			return nil, err
		}
		blk, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		g, err := cipher.NewGCM(blk)
		if err != nil {
			return nil, err
		}
		if len(ct) < g.NonceSize() {
			return nil, errors.New("mfa: sealed secret too short")
		}
		return g.Open(nil, ct[:g.NonceSize()], ct[g.NonceSize():], []byte(personID))
	}
	return nil, errors.New("mfa: unknown secret format")
}

// ---- recovery codes ----

const (
	recoveryCount = 10
	recoveryLen   = 10 // base32 characters = 50 bits
)

// normalizeCode upper-cases and strips spaces and hyphens.
func normalizeCode(c string) string {
	c = strings.ToUpper(c)
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '-' || r == '\t' {
			return -1
		}
		return r
	}, c)
}

func validRecoveryFormat(c string) bool {
	if len(c) != recoveryLen {
		return false
	}
	for i := 0; i < len(c); i++ {
		ch := c[i]
		if (ch < 'A' || ch > 'Z') && (ch < '2' || ch > '7') {
			return false
		}
	}
	return true
}

// newRecoveryCodes returns recoveryCount codes shown as XXXXX-XXXXX.
func newRecoveryCodes() ([]string, error) {
	out := make([]string, recoveryCount)
	for i := range out {
		b := make([]byte, 7)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		c := b32.EncodeToString(b)[:recoveryLen]
		out[i] = c[:5] + "-" + c[5:]
	}
	return out, nil
}

// hashRecoveryCode binds the person id into the hash so equal codes of two
// people never share a stored value.
func hashRecoveryCode(personID, normalized string) string {
	return hashToken("circula-recovery-v1\x00" + personID + "\x00" + normalized)
}
