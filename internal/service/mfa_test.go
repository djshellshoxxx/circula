package service_test

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // RFC 6238 test oracle
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// totpAt is an independent RFC 6238 implementation used as the test oracle.
func totpAt(t *testing.T, b32 string, at time.Time) string {
	t.Helper()
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(b32)
	if err != nil {
		t.Fatal(err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(at.Unix()/30)) //nolint:gosec // positive
	m := hmac.New(sha1.New, key)
	m.Write(msg[:])
	s := m.Sum(nil)
	o := s[19] & 15
	v := binary.BigEndian.Uint32(s[o:o+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", v%1000000)
}

func now(e *tu.Env) time.Time { return e.Clock.Now() }

// enrol enables MFA for a, then moves the clock one step so fresh codes work.
func enrol(t *testing.T, e *tu.Env, a domain.Actor) (secret string, codes []string) {
	t.Helper()
	en := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, a, pw))
	codes = tu.Must(e.Svc.ConfirmMFAEnrolment(e.Ctx, a, pw, totpAt(t, en.Secret, now(e))))
	e.Clock.Advance(31 * time.Second)
	return en.Secret, codes
}

func login(e *tu.Env, who string) *service.LoginResult {
	r, err := e.Svc.LoginStep(e.Ctx, who, pw)
	if err != nil {
		panic(err)
	}
	return r
}

func count(t *testing.T, e *tu.Env, q string, args ...any) int {
	t.Helper()
	var n int
	if err := e.Store.DB().QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMFAEnrolFlow(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	st := tu.Must(e.Svc.MFAStatus(e.Ctx, a))
	if st.Enabled || st.Pending {
		t.Fatalf("fresh status %+v", st)
	}
	en := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, a, pw))
	if !strings.HasPrefix(en.URI, "otpauth://totp/Circula:ana@example.org?") && !strings.HasPrefix(en.URI, "otpauth://totp/Circula:ana%40example.org?") &&
		!strings.HasPrefix(en.URI, "otpauth://totp/Circula%3Aana@example.org?") {
		t.Errorf("uri %q", en.URI)
	}
	for _, want := range []string{"secret=" + en.Secret, "issuer=Circula", "digits=6", "period=30", "algorithm=SHA1"} {
		if !strings.Contains(en.URI, want) {
			t.Errorf("uri lacks %s: %s", want, en.URI)
		}
	}
	if len(en.Secret) != 32 { // 20 bytes
		t.Errorf("secret length %d", len(en.Secret))
	}
	// pending: begin again returns the same secret and writes no second audit event
	if en2 := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, a, pw)); en2.Secret != en.Secret {
		t.Error("resume must return the pending secret")
	}
	if n := count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActMFAEnrolBegun); n != 1 {
		t.Errorf("enrol_begun events %d", n)
	}
	if !tu.Must(e.Svc.MFAStatus(e.Ctx, a)).Pending {
		t.Error("expected pending")
	}
	if _, err := e.Svc.ConfirmMFAEnrolment(e.Ctx, a, pw, "000000"); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("wrong code: %v", err)
	}
	if tu.Must(e.Svc.MFAStatus(e.Ctx, a)).Enabled {
		t.Fatal("enabled by wrong code")
	}
	codes := tu.Must(e.Svc.ConfirmMFAEnrolment(e.Ctx, a, pw, totpAt(t, en.Secret, now(e))))
	if len(codes) != 10 {
		t.Fatalf("codes %d", len(codes))
	}
	st = tu.Must(e.Svc.MFAStatus(e.Ctx, a))
	if !st.Enabled || st.RecoveryRemaining != 10 {
		t.Errorf("status %+v", st)
	}
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, pw); domain.CodeOf(err) != domain.CodeConflict {
		t.Errorf("begin when enabled: %v", err)
	}
	// SEC-007: enrolment audited and noticed
	if n := count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActMFAEnabled); n != 1 {
		t.Errorf("mfa.enabled events %d", n)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM security_notices WHERE kind = ?`, domain.NoticeMFAEnabled); n != 1 {
		t.Errorf("notices %d", n)
	}
	// the confirming code is burned: it cannot open a login a moment later
	r := login(e, "ana@example.org")
	if r.MFAChallenge == "" || r.Token != "" {
		t.Fatalf("expected challenge: %+v", r)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, en.Secret, now(e))); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("replay of enrolment code: %v", err)
	}
}

func TestMFASecretsAndCodesNotStoredInClear(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, codes := enrol(t, e, a)
	for _, table := range []string{"audit_events", "security_notices", "recovery_codes", "mfa_challenges", "login_throttle"} {
		rows, err := e.Store.DB().Query(`SELECT * FROM ` + table)
		if err != nil {
			t.Fatal(err)
		}
		cols, _ := rows.Columns()
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			_ = rows.Scan(ptrs...)
			for _, v := range vals {
				s, ok := v.(string)
				if !ok {
					continue
				}
				if strings.Contains(s, secret) {
					t.Errorf("secret in %s", table)
				}
				for _, c := range codes {
					if strings.Contains(s, c) || strings.Contains(s, strings.ReplaceAll(c, "-", "")) {
						t.Errorf("recovery code in %s", table)
					}
				}
			}
		}
		_ = rows.Close()
	}
	// no read path hands the secret back after enrolment
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, a, pw); err == nil {
		t.Error("Begin after enrolment must not return a secret")
	}
}

func TestMFASecretSealedWithKey(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.SecretKey = []byte("0123456789abcdef0123456789abcdef") })
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	var stored string
	if err := e.Store.DB().QueryRow(`SELECT secret FROM person_mfa`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, "v1:") || strings.Contains(stored, secret) {
		t.Errorf("stored %q", stored)
	}
	r := login(e, "ana@example.org")
	tok, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, secret, now(e)))
	if err != nil || tok == "" {
		t.Fatalf("login with sealed secret: %v", err)
	}
	// a sealed value moved to another person must not open (AAD binds person)
	b := e.Person("bob@example.org")
	enB := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, b, pw))
	if _, err := e.Store.DB().Exec(`UPDATE person_mfa SET secret = ? WHERE person_id = ?`, stored, b.PersonID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Svc.ConfirmMFAEnrolment(e.Ctx, b, pw, totpAt(t, enB.Secret, now(e))); err == nil {
		t.Error("moved sealed secret accepted")
	}
}

func TestLoginWithoutMFAUnchangedAndLegacyLoginRefusesMFA(t *testing.T) {
	e := tu.NewEnv(t)
	e.Person("plain@example.org")
	tok, p, err := e.Svc.Login(e.Ctx, "plain@example.org", pw)
	if err != nil || tok == "" || p == nil {
		t.Fatalf("plain login: %v", err)
	}
	a := e.Person("ana@example.org")
	enrol(t, e, a)
	before := count(t, e, `SELECT COUNT(*) FROM sessions`)
	tok, _, err = e.Svc.Login(e.Ctx, "ana@example.org", pw)
	if tok != "" || domain.CodeOf(err) != domain.CodeMFARequired {
		t.Errorf("legacy Login for MFA person: %q %v", tok, err)
	}
	if count(t, e, `SELECT COUNT(*) FROM sessions`) != before {
		t.Error("a session was created without second factor")
	}
	if _, err := e.Svc.LoginStep(e.Ctx, "ana@example.org", "wrong password!!"); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("wrong password: %v", err)
	}
}

func TestCompleteMFA(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)

	r := login(e, "ana@example.org")
	if r.Person == nil || r.Person.ID != a.PersonID {
		t.Fatal("person missing")
	}
	// the stored challenge is hashed
	if count(t, e, `SELECT COUNT(*) FROM mfa_challenges WHERE id_hash = ?`, r.MFAChallenge) != 0 {
		t.Error("challenge stored in clear")
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, "123456"); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("bad code: %v", err)
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, "garbage", totpAt(t, secret, now(e))); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("bad challenge: %v", err)
	}
	code := totpAt(t, secret, now(e))
	tok, p, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, code)
	if err != nil || p.ID != a.PersonID {
		t.Fatalf("complete: %v", err)
	}
	if got, err := e.Svc.Authenticate(e.Ctx, tok); err != nil || got.PersonID != a.PersonID {
		t.Fatalf("session: %v", err)
	}
	// challenge is single use; code is single use even with a fresh challenge
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, code); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("challenge reuse: %v", err)
	}
	r2 := login(e, "ana@example.org")
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r2.MFAChallenge, code); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("code replay: %v", err)
	}
	// previous-step skew works once the step is newer than the last used
	e.Clock.Advance(30 * time.Second)
	prev := totpAt(t, secret, now(e).Add(-30*time.Second)) // == step already used above
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r2.MFAChallenge, prev); err == nil {
		t.Error("older-or-equal step accepted")
	}
	// +1 step (client clock ahead) is accepted
	r3 := login(e, "ana@example.org")
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r3.MFAChallenge, totpAt(t, secret, now(e).Add(30*time.Second))); err != nil {
		t.Errorf("one step ahead: %v", err)
	}
	// +2 steps is not
	e.Clock.Advance(30 * time.Second)
	r4 := login(e, "ana@example.org")
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r4.MFAChallenge, totpAt(t, secret, now(e).Add(60*time.Second))); err == nil {
		t.Error("two steps ahead accepted")
	}
	// challenges expire after 5 minutes
	r5 := login(e, "ana@example.org")
	e.Clock.Advance(5*time.Minute + time.Second)
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r5.MFAChallenge, totpAt(t, secret, now(e))); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("expired challenge: %v", err)
	}
}

func TestMFAThrottle(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	r := login(e, "ana@example.org")
	for i := 0; i < 5; i++ {
		if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, "000000"); domain.CodeOf(err) != domain.CodeUnauthenticated {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}
	// locked: even the right code is refused without being evaluated
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, totpAt(t, secret, now(e))); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Errorf("locked: %v", err)
	}
	// the lock also covers disable and regenerate
	if err := e.Svc.DisableMFA(e.Ctx, a, pw, totpAt(t, secret, now(e))); domain.CodeOf(err) != domain.CodeRateLimited {
		t.Errorf("disable while locked: %v", err)
	}
	// the lock is not permanent
	e.Clock.Advance(16 * time.Minute)
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, totpAt(t, secret, now(e))); err != nil {
		t.Errorf("after lock: %v", err)
	}
	// no login string can address the throttle key
	if _, err := e.Svc.LoginStep(e.Ctx, "\x1fmfa:"+a.PersonID, pw); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("collision login: %v", err)
	}
}

func TestRecoveryCodes(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	_, codes := enrol(t, e, a)
	r := login(e, "ana@example.org")
	// lower case and no hyphen are tolerated
	use := strings.ToLower(strings.ReplaceAll(codes[0], "-", ""))
	tok, _, err := e.Svc.CompleteMFA(e.Ctx, r.MFAChallenge, use)
	if err != nil || tok == "" {
		t.Fatalf("recovery login: %v", err)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActMFARecoveryCodeUsed); n != 1 {
		t.Errorf("audit events %d", n)
	}
	if n := count(t, e, `SELECT COUNT(*) FROM security_notices WHERE kind = ?`, domain.NoticeRecoveryCodeUsed); n != 1 {
		t.Errorf("notices %d", n)
	}
	if st := tu.Must(e.Svc.MFAStatus(e.Ctx, a)); st.RecoveryRemaining != 9 {
		t.Errorf("remaining %d", st.RecoveryRemaining)
	}
	// SEC-007: a used recovery code is rejected on reuse
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, codes[0]); domain.CodeOf(err) != domain.CodeUnauthenticated {
		t.Errorf("reuse: %v", err)
	}
	// another person's code does not work for me
	b := e.Person("bob@example.org")
	_, bcodes := enrol(t, e, b)
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, bcodes[1]); err == nil {
		t.Error("cross-person recovery code accepted")
	}
	// the tx-level helper consumes exactly once
	var first, second bool
	if err := e.Store.Update(e.Ctx, func(tx store.Tx) error {
		var err error
		if first, err = e.Svc.VerifyAndConsumeRecoveryCode(tx, a.PersonID, codes[2]); err != nil {
			return err
		}
		second, err = e.Svc.VerifyAndConsumeRecoveryCode(tx, a.PersonID, codes[2])
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if !first || second {
		t.Errorf("helper first=%v second=%v", first, second)
	}
	for _, bad := range []string{"", "short", "!!!!!-!!!!!", strings.Repeat("A", 40)} {
		_ = e.Store.Update(e.Ctx, func(tx store.Tx) error {
			if ok, err := e.Svc.VerifyAndConsumeRecoveryCode(tx, a.PersonID, bad); ok || err != nil {
				t.Errorf("%q: %v %v", bad, ok, err)
			}
			return nil
		})
	}
}

func TestRegenerateRecoveryCodes(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, old := enrol(t, e, a)
	// a recovery code cannot authorise regeneration
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, a, old[0]); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("recovery code accepted: %v", err)
	}
	fresh := tu.Must(e.Svc.RegenerateRecoveryCodes(e.Ctx, a, totpAt(t, secret, now(e))))
	if len(fresh) != 10 || fresh[0] == old[0] {
		t.Fatal("codes not replaced")
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, old[3]); err == nil {
		t.Error("old recovery code still works")
	}
	if _, _, err := e.Svc.CompleteMFA(e.Ctx, login(e, "ana@example.org").MFAChallenge, fresh[3]); err != nil {
		t.Errorf("new code: %v", err)
	}
	if count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActMFARecoveryRegen) != 1 ||
		count(t, e, `SELECT COUNT(*) FROM security_notices WHERE kind = ?`, domain.NoticeRecoveryRegen) != 1 {
		t.Error("regeneration not audited and noticed")
	}
	// not enabled: invalid state
	if _, err := e.Svc.RegenerateRecoveryCodes(e.Ctx, e.Person("bob@example.org"), "123456"); domain.CodeOf(err) != domain.CodeInvalidState {
		t.Errorf("not enabled: %v", err)
	}
}

func TestDisableMFA(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, codes := enrol(t, e, a)
	if err := e.Svc.DisableMFA(e.Ctx, a, "wrong password!!", totpAt(t, secret, now(e))); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("wrong password: %v", err)
	}
	if err := e.Svc.DisableMFA(e.Ctx, a, pw, "000000"); domain.CodeOf(err) != domain.CodeValidation {
		t.Errorf("wrong code: %v", err)
	}
	if !tu.Must(e.Svc.MFAStatus(e.Ctx, a)).Enabled {
		t.Fatal("disabled by a failed attempt")
	}
	// a recovery code may authorise disabling (lost device)
	if err := e.Svc.DisableMFA(e.Ctx, a, pw, codes[0]); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if st := tu.Must(e.Svc.MFAStatus(e.Ctx, a)); st.Enabled || st.RecoveryRemaining != 0 {
		t.Errorf("status %+v", st)
	}
	if count(t, e, `SELECT COUNT(*) FROM recovery_codes`) != 0 || count(t, e, `SELECT COUNT(*) FROM person_mfa`) != 0 {
		t.Error("material left behind")
	}
	if count(t, e, `SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActMFADisabled) != 1 ||
		count(t, e, `SELECT COUNT(*) FROM security_notices WHERE kind = ?`, domain.NoticeMFADisabled) != 1 {
		t.Error("disable not audited and noticed")
	}
	if tok, _, err := e.Svc.Login(e.Ctx, "ana@example.org", pw); err != nil || tok == "" {
		t.Errorf("password-only login after disable: %v", err)
	}
	if err := e.Svc.DisableMFA(e.Ctx, a, pw, "123456"); domain.CodeOf(err) != domain.CodeInvalidState {
		t.Errorf("already off: %v", err)
	}
}

func TestMFAAuthzScopes(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	ro := domain.Actor{PersonID: a.PersonID, Scopes: []domain.Scope{domain.ScopeRead}}
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, ro, pw); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("read-only begin: %v", err)
	}
	if _, err := e.Svc.BeginMFAEnrolment(e.Ctx, domain.SystemActor("r"), pw); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("system begin: %v", err)
	}
	if _, err := e.Svc.ListSecurityNotices(e.Ctx, domain.Actor{}, false); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("anonymous notices: %v", err)
	}
}

// SEC-007: with RequireMFAForAdmins an administrator cannot complete an admin
// action until enrolled; members are unaffected.
func TestRequireMFAForAdmins(t *testing.T) {
	e := tu.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	admin := e.Person("admin@example.org")
	// set-up needs admin calls, so enrol first and drop the factor afterwards
	_, rc := enrol(t, e, admin)
	net, _ := e.Network(admin, "town", nil)
	member, _ := e.Member(admin, net, "m@example.org")
	if err := e.Svc.DisableMFA(e.Ctx, admin, pw, rc[0]); err != nil {
		t.Fatal(err)
	}

	_, _, err := e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{})
	if domain.CodeOf(err) != domain.CodeForbidden || !strings.Contains(err.Error(), "mfa-enrolment-required") {
		t.Fatalf("admin without MFA: %v", err)
	}
	// non-admin operations still work
	if _, err := e.Svc.GetBalance(e.Ctx, admin, net.ID); err != nil {
		t.Errorf("member read by admin: %v", err)
	}
	if _, err := e.Svc.GetBalance(e.Ctx, member, net.ID); err != nil {
		t.Errorf("plain member: %v", err)
	}
	// a pending (unconfirmed) enrolment does not count
	en := tu.Must(e.Svc.BeginMFAEnrolment(e.Ctx, admin, pw))
	if _, _, err := e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{}); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("pending enrolment: %v", err)
	}
	tu.Must(e.Svc.ConfirmMFAEnrolment(e.Ctx, admin, pw, totpAt(t, en.Secret, now(e))))
	if _, _, err := e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{}); err != nil {
		t.Errorf("admin with MFA: %v", err)
	}
	// a member cannot use the exemption to gain admin
	if _, _, err := e.Svc.CreateInvitation(e.Ctx, member, net.ID, service.InviteInput{}); domain.CodeOf(err) != domain.CodeForbidden {
		t.Errorf("member invite: %v", err)
	}
	// system actors (CLI, jobs) are exempt
	if _, _, err := e.Svc.CreateInvitation(e.Ctx, domain.SystemActor("r"), net.ID, service.InviteInput{}); err != nil {
		t.Errorf("system: %v", err)
	}
}

func TestRequireMFADefaultOff(t *testing.T) {
	e := tu.NewEnv(t)
	admin := e.Person("admin@example.org")
	net, _ := e.Network(admin, "town", nil)
	if _, _, err := e.Svc.CreateInvitation(e.Ctx, admin, net.ID, service.InviteInput{}); err != nil {
		t.Errorf("default must not require MFA: %v", err)
	}
}

func TestSecurityNotices(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	b := e.Person("bob@example.org")
	_, tok := tu.Must2(e.Svc.CreateAPIToken(e.Ctx, a, service.TokenInput{Name: "ci", Scopes: []domain.Scope{domain.ScopeRead}}))
	if err := e.Svc.RevokeAPIToken(e.Ctx, a, tok.ID); err != nil {
		t.Fatal(err)
	}
	secret, _ := enrol(t, e, a)
	_ = secret
	if err := e.Svc.ChangePassword(e.Ctx, a, pw, "another long password"); err != nil {
		t.Fatal(err)
	}
	ns := tu.Must(e.Svc.ListSecurityNotices(e.Ctx, a, false))
	kinds := map[string]bool{}
	for _, n := range ns {
		kinds[n.Kind] = true
		if n.PersonID != a.PersonID || n.SeenAt != nil {
			t.Errorf("notice %+v", n)
		}
	}
	for _, k := range []string{domain.NoticeTokenCreated, domain.NoticeTokenRevoked, domain.NoticeMFAEnabled, domain.NoticePasswordChanged} {
		if !kinds[k] {
			t.Errorf("missing notice %s (have %v)", k, kinds)
		}
	}
	// cross-person isolation
	if got := tu.Must(e.Svc.ListSecurityNotices(e.Ctx, b, false)); len(got) != 0 {
		t.Errorf("bob sees %d of ana's notices", len(got))
	}
	if n := tu.Must(e.Svc.AckSecurityNotices(e.Ctx, b)); n != 0 {
		t.Errorf("bob acked %d", n)
	}
	if c := tu.Must(e.Svc.CountUnseenSecurityNotices(e.Ctx, a)); c != len(ns) {
		t.Errorf("unseen %d want %d", c, len(ns))
	}
	if n := tu.Must(e.Svc.AckSecurityNotices(e.Ctx, a)); int(n) != len(ns) {
		t.Errorf("acked %d", n)
	}
	if got := tu.Must(e.Svc.ListSecurityNotices(e.Ctx, a, true)); len(got) != 0 {
		t.Errorf("unseen after ack: %d", len(got))
	}
	if got := tu.Must(e.Svc.ListSecurityNotices(e.Ctx, a, false)); len(got) != len(ns) || got[0].SeenAt == nil {
		t.Error("history lost or not marked")
	}
}

// Replay race: the same code offered concurrently on two challenges wins once.
func TestMFAConcurrentReplay(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, _ := enrol(t, e, a)
	const n = 6
	chs := make([]string, n)
	for i := range chs {
		chs[i] = login(e, "ana@example.org").MFAChallenge
	}
	code := totpAt(t, secret, now(e))
	var ok atomic.Int32
	var wg sync.WaitGroup
	for _, c := range chs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := e.Svc.CompleteMFA(e.Ctx, c, code); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Errorf("code accepted %d times", ok.Load())
	}
}

// Same challenge in parallel: one session at most.
func TestMFAConcurrentChallengeUse(t *testing.T) {
	e := tu.NewEnv(t)
	a := e.Person("ana@example.org")
	secret, codes := enrol(t, e, a)
	ch := login(e, "ana@example.org").MFAChallenge
	var ok atomic.Int32
	var wg sync.WaitGroup
	for _, code := range []string{totpAt(t, secret, now(e)), codes[0], codes[1], codes[2]} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, _, err := e.Svc.CompleteMFA(e.Ctx, ch, code); err == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Errorf("challenge used %d times", ok.Load())
	}
}
