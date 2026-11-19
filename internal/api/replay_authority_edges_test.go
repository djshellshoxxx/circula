package api_test

import (
	"crypto/hmac"
	"crypto/sha1" //nolint:gosec // Independent RFC 6238 test oracle.
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/api"
	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

const replayTestPassword = "correct horse battery"

func enrollReplayMFA(t *testing.T, env *testutil.Env, actor domain.Actor) []string {
	t.Helper()
	enrollment := testutil.Must(env.Svc.BeginMFAEnrolment(env.Ctx, actor, replayTestPassword))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(enrollment.Secret)
	if err != nil {
		t.Fatal(err)
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(env.Clock.Now().Unix()/30)) //nolint:gosec // Positive fixture timestamp.
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 15
	code := fmt.Sprintf("%06d", (binary.BigEndian.Uint32(digest[offset:offset+4])&0x7fffffff)%1000000)
	codes := testutil.Must(env.Svc.ConfirmMFAEnrolment(env.Ctx, actor, replayTestPassword, code))
	env.Clock.Advance(31 * time.Second)
	return codes
}

func assertOriginalReplay(t *testing.T, first, again *httptest.ResponseRecorder) {
	t.Helper()
	wantStatus(t, again, first.Code)
	if again.Body.String() != first.Body.String() || again.Header().Get("Idempotent-Replayed") != "true" {
		t.Fatalf("authorized retry changed the original response: %s", again.Body.String())
	}
}

// The cached invitation contains a credential. Neither a narrower token nor
// loss of mandatory MFA may retrieve it; restoring MFA must recover the same
// response, without creating another invitation under the reused key.
func TestReplayInvitationRequiresCurrentMFA(t *testing.T) {
	env := testutil.NewEnvConfig(t, nil, func(c *service.Config) { c.RequireMFAForAdmins = true })
	admin := env.Person("admin")
	codes := enrollReplayMFA(t, env, admin)
	net, member := env.Network(admin, "replay-mfa", nil)
	w := &world{t: t, env: env, net: net, admin: admin, adminM: member,
		h: api.NewHandler(env.Svc)}
	t.Cleanup(func() { api.CloseHandler(w.h) })
	w.adminT = w.token(admin, domain.ScopeRead, domain.ScopeWrite)
	body, key := m{}, w.key()
	first := w.post(w.adminT, "/admin/invitations", body, key)
	wantStatus(t, first, 201)
	readOnly := w.token(admin, domain.ScopeRead)
	wantProblem(t, w.post(readOnly, "/admin/invitations", body, key), 403, "forbidden")
	if err := env.Svc.DisableMFA(env.Ctx, admin, replayTestPassword, codes[0]); err != nil {
		t.Fatal(err)
	}
	p := wantProblem(t, w.post(w.adminT, "/admin/invitations", body, key), 403, "forbidden")
	if !strings.Contains(p.Detail, "mfa-enrolment-required") {
		t.Fatalf("replay did not enforce current MFA: %+v", p)
	}
	testutil.Must(env.Svc.BeginMFAEnrolment(env.Ctx, admin, replayTestPassword))
	wantProblem(t, w.post(w.adminT, "/admin/invitations", body, key), 403, "forbidden")
	enrollReplayMFA(t, env, admin)
	assertOriginalReplay(t, first, w.post(w.adminT, "/admin/invitations", body, key))
	var count int
	if err := env.Store.DB().QueryRow(`SELECT count(*) FROM invitations WHERE network_id=?`, net.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("refused retries recreated the invitation: count=%d error=%v", count, err)
	}
}

// Temporary authority expires on the next request, without the cleanup job.
// A later permanent grant restores access to the original successful result.
func TestReplayInvitationRechecksBreakGlassExpiry(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	sys := domain.SystemActor("replay-test")
	testutil.Must(w.env.Svc.OpenBreakGlass(w.env.Ctx, sys, service.OpenBreakGlassInput{
		Network: w.net.ID, Operator: "Test operator", Reason: "Replay authorization test", TTL: time.Minute}))
	testutil.Must(w.env.Svc.GrantBreakGlassAdmin(w.env.Ctx, sys, service.GrantBreakGlassInput{
		Network: w.net.ID, Login: "alice", Reason: "Temporary recovery access"}))
	body, key := m{}, w.key()
	first := w.post(w.aliceT, "/admin/invitations", body, key)
	wantStatus(t, first, 201)
	assertOriginalReplay(t, first, w.post(w.aliceT, "/admin/invitations", body, key))
	w.env.Clock.Advance(time.Minute)
	wantProblem(t, w.post(w.aliceT, "/admin/invitations", body, key), 403, "forbidden")
	if err := w.env.Svc.AssignRole(w.env.Ctx, sys, w.net.ID, w.aliceM.ID, domain.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	assertOriginalReplay(t, first, w.post(w.aliceT, "/admin/invitations", body, key))
	var count int
	if err := w.env.Store.DB().QueryRow(`SELECT count(*) FROM invitations WHERE network_id=? AND created_by=?`,
		w.net.ID, w.aliceM.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay recreated the temporary admin's invitation: count=%d error=%v", count, err)
	}
}

// A noNet command acquires its network during the first execution. Replays
// authorize that cached membership, even after the invitation is consumed or
// another network is selected in a header. Reduced authority cannot retrieve
// the old response or consume the invitation a second time.
func TestReplayInvitationAcceptanceUsesCachedMembership(t *testing.T) {
	w := newWorld(t)
	t.Cleanup(func() { api.CloseHandler(w.h) })
	guest := w.env.Person("guest")
	guestToken := w.token(guest, domain.ScopeRead, domain.ScopeWrite)
	invite, _ := testutilInvite(t, w, w.admin, w.net.ID)
	path, body, key := "/invitations/"+invite+"/accept", m{"display_name": "Guest"}, w.key()
	first := w.post(guestToken, path, body, key)
	wantStatus(t, first, 201)
	joined := decode[struct {
		MemberID string `json:"member_id"`
	}](t, first)
	assertOriginalReplay(t, first, w.post(guestToken, path, body, key))
	readOnly := w.token(guest, domain.ScopeRead)
	wantProblem(t, w.post(readOnly, path, body, key), 403, "forbidden")
	scoped, _, err := w.env.Svc.CreateAPIToken(w.env.Ctx, guest, service.TokenInput{
		Name: "restricted", NetworkID: w.net.ID, Scopes: []domain.Scope{domain.ScopeRead, domain.ScopeWrite}})
	if err != nil {
		t.Fatal(err)
	}
	wantProblem(t, w.post(scoped, path, body, key), 403, "forbidden")
	other, _ := w.env.Network(guest, "guest-other-network", nil)
	if err := w.env.Svc.SuspendMember(w.env.Ctx, w.admin, w.net.ID, joined.MemberID, "Suspend original membership"); err != nil {
		t.Fatal(err)
	}
	wantProblem(t, w.do(req{method: "POST", path: "/api/v1" + path, token: guestToken, body: body,
		hdr: map[string]string{"Idempotency-Key": key, "X-Network": other.ID}}), 403, "forbidden")
	if err := w.env.Svc.RestoreMember(w.env.Ctx, w.admin, w.net.ID, joined.MemberID, "Restore original membership"); err != nil {
		t.Fatal(err)
	}
	assertOriginalReplay(t, first, w.post(guestToken, path, body, key))
	testutil.Must(w.env.Svc.DepartMember(w.env.Ctx, guest, w.net.ID, service.DepartInput{}))
	wantProblem(t, w.post(guestToken, path, body, key), 403, "forbidden")
	var count int
	if err := w.env.Store.DB().QueryRow(`SELECT count(*) FROM members WHERE network_id=? AND person_id=?`,
		w.net.ID, guest.PersonID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("replay created duplicate membership: count=%d error=%v", count, err)
	}
}
