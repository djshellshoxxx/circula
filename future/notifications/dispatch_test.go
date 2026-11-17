package notifications_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	n "github.com/djshellshoxxx/circula/future/notifications"
	"github.com/djshellshoxxx/circula/future/notifications/notifytest"
)

var ctx = context.Background()

func ev(id, member, kind string) n.Event {
	return n.Event{NetworkID: net1, EventID: id, MemberID: member, Kind: kind}
}

func (e *env) run(d *n.Dispatcher) n.RunStats {
	e.t.Helper()
	st, err := d.RunOnce(ctx)
	if err != nil {
		e.t.Fatal(err)
	}
	return st
}

func TestEmailDoubleOptInThenGenericDelivery(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "email", n.KindProposalReceived)
	info, err := e.svc.AddEmailTarget(ctx, a, "Alice.Secret@Example.ORG")
	if err != nil || info.Verified || info.Hint != "A***@example.org" {
		t.Fatalf("%+v %v", info, err)
	}
	d := e.disp("w1", n.Hooks{})
	// An event before confirmation must not reach the unconfirmed address.
	pre := e.enqueue(ev("pre", "alice", n.KindProposalReceived))
	e.run(d)
	sent := e.email.Sent()
	if len(sent) != 1 || sent[0].Subject != n.VerifySubject || sent[0].To != "Alice.Secret@example.org" {
		t.Fatalf("expected only the verification mail, got %+v", sent)
	}
	if s, why := e.status(pre[0].DeliveryID); s != "cancelled" || why != "no_target" {
		t.Fatalf("pre-confirmation event: %s/%s", s, why)
	}
	tok := tokenFrom(t, sent[0].Body, "/notifications/confirm-email?t=")
	if err := e.svc.ConfirmEmail(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.ConfirmEmail(ctx, tok); err != nil {
		t.Fatal("confirm must be idempotent:", err)
	}
	e.enqueue(ev("ev1", "alice", n.KindProposalReceived))
	e.run(d)
	sent = e.email.Sent()
	if len(sent) != 2 {
		t.Fatalf("sent %d", len(sent))
	}
	m := sent[1]
	if m.Subject != n.GenericSubject || !strings.Contains(m.Body, n.GenericBody) || m.UnsubscribeURL == "" || !strings.HasPrefix(m.UnsubscribeURL, baseURL+"/notifications/unsubscribe?t=") {
		t.Fatalf("message: %+v", m)
	}
	for _, leak := range []string{"ev1", "net-1", "alice", "proposal", "received", "Secret"} {
		if strings.Contains(m.Subject+m.Body, leak) {
			t.Fatalf("message leaks %q: %s", leak, m.Subject+m.Body)
		}
	}
	var prov sql.NullString
	if err := e.db.QueryRow(`SELECT provider_message_id FROM notification_delivery WHERE event_id='ev1'`).Scan(&prov); err != nil || prov.String != m.MessageID {
		t.Fatalf("provider id %v %v", prov, err)
	}
}

func TestConfirmTokenRejections(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	if _, err := e.svc.AddEmailTarget(ctx, a, "a@example.org"); err != nil {
		t.Fatal(err)
	}
	d := e.disp("w", n.Hooks{})
	e.run(d)
	tok := tokenFrom(t, e.email.Sent()[0].Body, "/notifications/confirm-email?t=")
	for _, bad := range []string{"", "garbage", tok + "x", tok[:len(tok)-2], strings.Replace(tok, "v1.", "v2.", 1), e.svc.UnsubscribeToken(net1, "alice", "email")} {
		if err := e.svc.ConfirmEmail(ctx, bad); err != n.ErrToken {
			t.Fatalf("%q accepted: %v", bad, err)
		}
	}
	e.clk.Add(73 * time.Hour)
	if err := e.svc.ConfirmEmail(ctx, tok); err != n.ErrToken {
		t.Fatalf("expired token accepted: %v", err)
	}
	if info, _ := e.svc.GetEmailTarget(ctx, a); info.Verified {
		t.Fatal("verified by a bad token")
	}
}

func TestConfirmTokenCannotCrossMembers(t *testing.T) {
	e := newEnv(t)
	if _, err := e.svc.AddEmailTarget(ctx, actor(net1, "alice"), "a@example.org"); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := e.db.QueryRow(`SELECT id FROM notification_email_targets`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	// Tokens are signed over network+member+target: a valid token for bob in the same
	// network cannot confirm alice's target.
	d := e.disp("w", n.Hooks{})
	e.run(d)
	if _, err := e.svc.AddEmailTarget(ctx, actor(net1, "bob"), "b@example.org"); err != nil {
		t.Fatal(err)
	}
	e.run(d)
	sent := e.email.Sent()
	bobTok := tokenFrom(t, sent[len(sent)-1].Body, "/notifications/confirm-email?t=")
	if err := e.svc.ConfirmEmail(ctx, bobTok); err != nil {
		t.Fatal(err)
	}
	if info, _ := e.svc.GetEmailTarget(ctx, actor(net1, "alice")); info.Verified {
		t.Fatal("alice verified through bob's token")
	}
}

func TestEmailAddressValidation(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	for _, bad := range []string{"", "no-at", "a@b", "A Name <a@example.org>", "a@example.org\r\nBcc: x@y.z", "a b@example.org", "\"quoted\"@example.org",
		"a@@example.org", "a@example..org", "a@-example.org", "é@example.org", "a@example.org,b@example.org", "<a@example.org>", strings.Repeat("a", 65) + "@example.org",
		"a@" + strings.Repeat("b", 250) + ".org"} {
		if _, err := e.svc.AddEmailTarget(ctx, a, bad); err == nil {
			t.Fatalf("%q accepted", bad)
		}
	}
	if e.count(`SELECT COUNT(*) FROM notification_email_targets`) != 0 || e.count(`SELECT COUNT(*) FROM notification_delivery`) != 0 {
		t.Fatal("rejected address left rows behind")
	}
}

func TestVerificationRateLimitAndReplacement(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	for i := 0; i < 3; i++ {
		if _, err := e.svc.AddEmailTarget(ctx, a, "a@example.org"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := e.svc.AddEmailTarget(ctx, a, "a@example.org"); err != n.ErrRateLimited {
		t.Fatalf("got %v", err)
	}
	e.clk.Add(25 * time.Hour)
	if _, err := e.svc.AddEmailTarget(ctx, a, "other@example.org"); err != nil {
		t.Fatal(err)
	}
	// Replacing the address revokes the old target; only one stays active.
	if e.count(`SELECT COUNT(*) FROM notification_email_targets WHERE revoked_at IS NULL`) != 1 {
		t.Fatal("two active targets")
	}
}

func TestPushDeliveryPayloadAndFanOut(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.pushSub(a, "https://updates.push.services.mozilla.com/wpush/v2/xyz")
	e.enqueue(ev("ev1", "alice", n.KindDispute))
	st := e.run(e.disp("w", n.Hooks{}))
	if st.Sent != 1 || len(e.push.Sent()) != 2 {
		t.Fatalf("stats %+v sent %d", st, len(e.push.Sent()))
	}
	for _, m := range e.push.Sent() {
		if string(m.Payload) != string(n.PushPayload()) {
			t.Fatalf("payload %s", m.Payload)
		}
		for _, leak := range []string{"ev1", "net-1", "alice", "dispute", "amount", "@"} {
			if strings.Contains(string(m.Payload), leak) {
				t.Fatalf("payload leaks %q", leak)
			}
		}
		if m.TTL > 24*time.Hour || m.TTL < time.Minute {
			t.Fatalf("ttl %v", m.TTL)
		}
	}
}

func TestFanOutRetryDoesNotRepeatSuccessfulTargets(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.pushSub(a, "https://updates.push.services.mozilla.com/wpush/v2/xyz")
	failMozilla := true
	e.push.fn = func(m n.PushMessage) error {
		if strings.Contains(m.Endpoint, "mozilla") && failMozilla {
			return n.NewSendError(n.ClassTransient, "provider_5xx", 0, nil)
		}
		return nil
	}
	e.enqueue(ev("ev1", "alice", n.KindDispute))
	d := e.disp("w", n.Hooks{})
	if st := e.run(d); st.Retried != 1 {
		t.Fatalf("%+v", st)
	}
	failMozilla = false
	e.clk.Add(2 * time.Minute)
	if st := e.run(d); st.Sent != 1 {
		t.Fatalf("%+v", st)
	}
	counts := map[string]int{}
	for _, m := range e.push.Sent() {
		counts[m.Endpoint]++
	}
	if counts[fcm] != 1 || counts["https://updates.push.services.mozilla.com/wpush/v2/xyz"] != 1 {
		t.Fatalf("successful target repeated: %v", counts)
	}
}

func TestRetryBackoffCapAndExhaustion(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassTransient, "provider_5xx", 0, nil) }
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(48 * time.Hour)})
	d := e.disp("w", n.Hooks{})
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute}
	for i, w := range want {
		e.run(d)
		var next int64
		var att int
		if err := e.db.QueryRow(`SELECT next_attempt_at,attempts FROM notification_delivery WHERE id=?`, got[0].DeliveryID).Scan(&next, &att); err != nil {
			t.Fatal(err)
		}
		if att != i+1 || time.Unix(next, 0).Sub(e.clk.Now()) != w {
			t.Fatalf("attempt %d: attempts=%d delay=%v want %v", i+1, att, time.Unix(next, 0).Sub(e.clk.Now()), w)
		}
		e.clk.Add(w)
	}
	e.run(d) // sixth failure exhausts
	if s, why := e.status(got[0].DeliveryID); s != "failed" || why != "exhausted" {
		t.Fatalf("%s/%s", s, why)
	}
	calls := len(e.push.Sent())
	e.clk.Add(time.Hour)
	e.run(d)
	if len(e.push.Sent()) != calls {
		t.Fatal("sent after exhaustion")
	}
	if got := e.svc.Metrics().Failed.Load(); got != 1 {
		t.Fatalf("failed metric %d", got)
	}
}

func TestExpiryBeatsRetry(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassTransient, "network", 0, nil) }
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(90 * time.Second)})
	d := e.disp("w", n.Hooks{})
	e.run(d) // first failure schedules t+1m, inside expiry
	e.clk.Add(time.Minute)
	e.run(d) // second failure would schedule t+3m > expiry
	if s, why := e.status(got[0].DeliveryID); s != "expired" || why != "expired" {
		t.Fatalf("%s/%s", s, why)
	}
	n0 := len(e.push.Sent())
	e.clk.Add(time.Hour)
	e.run(d)
	if len(e.push.Sent()) != n0 {
		t.Fatal("sent after expiry")
	}
}

func TestRetryAfterIsHonouredAndCapped(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	ra := 30 * time.Minute
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassRateLimited, "rate_limited", ra, nil) }
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(48 * time.Hour)})
	d := e.disp("w", n.Hooks{})
	e.run(d)
	var next int64
	_ = e.db.QueryRow(`SELECT next_attempt_at FROM notification_delivery WHERE id=?`, got[0].DeliveryID).Scan(&next)
	if time.Unix(next, 0).Sub(t0) != 30*time.Minute {
		t.Fatalf("Retry-After ignored: %v", time.Unix(next, 0).Sub(t0))
	}
	e.clk.Add(30 * time.Minute)
	ra = 1000 * time.Hour
	e.run(d)
	_ = e.db.QueryRow(`SELECT next_attempt_at FROM notification_delivery WHERE id=?`, got[0].DeliveryID).Scan(&next)
	if time.Unix(next, 0).Sub(e.clk.Now()) != n.MaxRetryAfter {
		t.Fatalf("Retry-After not capped: %v", time.Unix(next, 0).Sub(e.clk.Now()))
	}
}

func TestPermanentFailureAndGoneTargets(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	sub := e.pushSub(a, fcm)
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassTargetGone, "subscription_gone", 0, nil) }
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.run(e.disp("w", n.Hooks{}))
	if s, why := e.status(got[0].DeliveryID); s != "cancelled" || why != "no_target" {
		t.Fatalf("%s/%s", s, why)
	}
	subs, _ := e.svc.ListPushSubscriptions(ctx, a)
	if len(subs) != 0 {
		t.Fatalf("gone subscription %s still active", sub.ID)
	}
	var reason string
	_ = e.db.QueryRow(`SELECT revoke_reason FROM push_subscriptions WHERE id=?`, sub.ID).Scan(&reason)
	if reason != "provider_gone" {
		t.Fatal(reason)
	}
	// A later event finds no target and sends nothing.
	e.enqueue(ev("ev2", "alice", n.KindDispute))
	before := len(e.push.Sent())
	e.run(e.disp("w2", n.Hooks{}))
	if len(e.push.Sent()) != before {
		t.Fatal("sent to revoked subscription")
	}
	// A permanent (non-target) failure is not retried.
	e.pushSub(a, fcm) // revives it
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassPermanent, "vapid_rejected", 0, nil) }
	got = e.enqueue(ev("ev3", "alice", n.KindDispute))
	e.run(e.disp("w3", n.Hooks{}))
	if s, why := e.status(got[0].DeliveryID); s != "failed" || why != "permanent" {
		t.Fatalf("%s/%s", s, why)
	}
}

func TestHardBounceSuppressesTarget(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "email", n.KindDispute)
	e.verifiedEmail(a, "gone@example.org")
	e.email.fn = func(n.EmailMessage) error { return n.NewSendError(n.ClassTargetGone, "recipient_rejected", 0, nil) }
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.run(e.disp("w", n.Hooks{}))
	info, _ := e.svc.GetEmailTarget(ctx, a)
	if !info.Bounced {
		t.Fatal("target not suppressed")
	}
	if s, _ := e.status(got[0].DeliveryID); s != "cancelled" {
		t.Fatal(s)
	}
	e.email.fn = nil
	before := len(e.email.Sent())
	e.enqueue(ev("ev2", "alice", n.KindDispute))
	e.run(e.disp("w2", n.Hooks{}))
	if len(e.email.Sent()) != before {
		t.Fatal("mail to a bounced target")
	}
	// A bounced address can be replaced by re-adding it and confirming again.
	if _, err := e.svc.AddEmailTarget(ctx, a, "gone@example.org"); err != nil {
		t.Fatal(err)
	}
	if e.count(`SELECT COUNT(*) FROM notification_email_targets WHERE revoked_at IS NULL AND bounced_at IS NULL`) != 1 {
		t.Fatal("re-add after bounce did not create a fresh target")
	}
}

func TestSoftBouncesSuppressAfterLimit(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.verifiedEmail(a, "soft@example.org")
	var id string
	_ = e.db.QueryRow(`SELECT id FROM notification_email_targets`).Scan(&id)
	for i := 0; i < 4; i++ {
		if err := e.svc.HandleBounce(ctx, net1, id, false); err != nil {
			t.Fatal(err)
		}
	}
	if info, _ := e.svc.GetEmailTarget(ctx, a); info.Bounced {
		t.Fatal("suppressed too early")
	}
	_ = e.svc.HandleBounce(ctx, net1, id, false)
	if info, _ := e.svc.GetEmailTarget(ctx, a); !info.Bounced {
		t.Fatal("not suppressed at limit")
	}
	// Foreign network cannot touch the target.
	if err := e.svc.HandleBounce(ctx, net2, id, true); err != n.ErrNotFound {
		t.Fatalf("cross-network bounce: %v", err)
	}
}

func TestQuietHoursDeferWithoutConsumingAttempts(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	if err := e.svc.SetPreferences(ctx, a, []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push", Enabled: true, QuietStart: ptr(22 * 60), QuietEnd: ptr(8 * 60), TZ: "America/Los_Angeles"}}); err != nil {
		t.Fatal(err)
	}
	e.pushSub(a, fcm)
	e.clk.Set(time.Date(2026, 11, 17, 7, 0, 0, 0, time.UTC)) // 23:00 PST on the 16th
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	d := e.disp("w", n.Hooks{})
	st := e.run(d)
	if st.Deferred != 1 || len(e.push.Sent()) != 0 {
		t.Fatalf("%+v", st)
	}
	var next int64
	var att, claims int
	_ = e.db.QueryRow(`SELECT next_attempt_at,attempts,claims FROM notification_delivery WHERE id=?`, got[0].DeliveryID).Scan(&next, &att, &claims)
	if want := time.Date(2026, 11, 17, 16, 0, 0, 0, time.UTC); time.Unix(next, 0).UTC() != want || att != 0 || claims != 0 {
		t.Fatalf("next=%v attempts=%d claims=%d", time.Unix(next, 0).UTC(), att, claims)
	}
	e.run(d)
	if len(e.push.Sent()) != 0 {
		t.Fatal("sent before the quiet window ended")
	}
	e.clk.Set(time.Date(2026, 11, 17, 16, 0, 1, 0, time.UTC))
	e.run(d)
	if len(e.push.Sent()) != 1 {
		t.Fatal("not sent after quiet hours")
	}
}

func TestQuietHoursPastExpiryExpires(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	_ = e.svc.SetPreferences(ctx, a, []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push", Enabled: true, QuietStart: ptr(22 * 60), QuietEnd: ptr(8 * 60), TZ: "America/Los_Angeles"}})
	e.pushSub(a, fcm)
	e.clk.Set(time.Date(2026, 11, 17, 7, 0, 0, 0, time.UTC))
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "ev", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: e.clk.Now().Add(2 * time.Hour)})
	e.run(e.disp("w", n.Hooks{}))
	if s, why := e.status(got[0].DeliveryID); s != "expired" || why != "quiet_hours" {
		t.Fatalf("%s/%s", s, why)
	}
}

func TestOptOutBeforeDispatchPreventsSend(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	// Simulate an opt-out that races past the cancelReady path: flip the row directly
	// so only the dispatch-time gate can stop the send.
	if _, err := e.db.Exec(`UPDATE notification_preferences SET enabled=0`); err != nil {
		t.Fatal(err)
	}
	e.run(e.disp("w", n.Hooks{}))
	if len(e.push.Sent()) != 0 {
		t.Fatal("sent after opt-out")
	}
	if s, why := e.status(got[0].DeliveryID); s != "cancelled" || why != "opt_out" {
		t.Fatalf("%s/%s", s, why)
	}
}

func TestOptOutRaceWindows(t *testing.T) {
	cases := map[string]struct {
		hook      func(e *env) n.Hooks
		wantSends int
	}{
		"after claim, before authorize": {func(e *env) n.Hooks {
			return n.Hooks{AfterClaim: func(context.Context, string) {
				_ = e.svc.SetPreferences(ctx, actor(net1, "alice"), []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push"}})
			}}
		}, 0},
		"after authorize, before send": {func(e *env) n.Hooks {
			return n.Hooks{AfterAuthorize: func(context.Context, string) {
				_ = e.svc.SetPreferences(ctx, actor(net1, "alice"), []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push"}})
			}}
		}, 0},
		// Once the provider has accepted the message nothing can recall it. This case
		// documents the residual window rather than hiding it.
		"after send": {func(e *env) n.Hooks {
			return n.Hooks{AfterSend: func(context.Context, string, string) error {
				_ = e.svc.SetPreferences(ctx, actor(net1, "alice"), []n.PrefUpdate{{EventType: n.KindDispute, Channel: "push"}})
				return nil
			}}
		}, 1},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			e := newEnv(t)
			a := actor(net1, "alice")
			e.optIn(a, "push", n.KindDispute)
			e.pushSub(a, fcm)
			e.enqueue(ev("ev1", "alice", n.KindDispute))
			e.run(e.disp("w", c.hook(e)))
			if got := len(e.push.Sent()); got != c.wantSends {
				t.Fatalf("sends=%d want %d", got, c.wantSends)
			}
			// Whatever happened, a later event after the opt-out is never sent.
			e.enqueue(ev("ev2", "alice", n.KindDispute))
			e.run(e.disp("w2", n.Hooks{}))
			if got := len(e.push.Sent()); got != c.wantSends {
				t.Fatalf("post-opt-out send: %d", got)
			}
		})
	}
}

func TestInactiveMemberNeverReceives(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.dir.SetActive(net1, "alice", false) // left the network after the event committed
	e.run(e.disp("w", n.Hooks{}))
	if len(e.push.Sent()) != 0 {
		t.Fatal("inactive member received a notification")
	}
	if s, why := e.status(got[0].DeliveryID); s != "cancelled" || why != "member_inactive" {
		t.Fatalf("%s/%s", s, why)
	}
}

func TestDirectoryOutageFailsClosed(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.dir.Err = context.DeadlineExceeded
	d := e.disp("w", n.Hooks{})
	if st := e.run(d); st.Deferred != 1 {
		t.Fatalf("%+v", st)
	}
	if len(e.push.Sent()) != 0 {
		t.Fatal("sent while directory unavailable")
	}
	e.dir.Err = nil
	e.clk.Add(2 * time.Minute)
	e.run(d)
	if len(e.push.Sent()) != 1 {
		t.Fatal("not delivered after directory recovered")
	}
	if s, _ := e.status(got[0].DeliveryID); s != "sent" {
		t.Fatal(s)
	}
}

func TestLogoutStopsPushes(t *testing.T) {
	e := newEnv(t)
	a := n.Actor{NetworkID: net1, MemberID: "alice", SessionID: "sess-1"}
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.pushSub(n.Actor{NetworkID: net1, MemberID: "alice", SessionID: "sess-2"}, "https://updates.push.services.mozilla.com/wpush/v2/xyz")
	cnt, err := e.svc.OnLogout(ctx, a)
	if err != nil || cnt != 1 {
		t.Fatalf("%d %v", cnt, err)
	}
	e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.run(e.disp("w", n.Hooks{}))
	sent := e.push.Sent()
	if len(sent) != 1 || !strings.Contains(sent[0].Endpoint, "mozilla") {
		t.Fatalf("logged-out subscription still receives: %+v", sent)
	}
	// Session ended without OnLogout being called (e.g. expiry): the dispatch gate catches it.
	e.dir.SetSession(net1, "alice", "sess-2", false)
	e.enqueue(ev("ev2", "alice", n.KindDispute))
	e.run(e.disp("w2", n.Hooks{}))
	if len(e.push.Sent()) != 1 {
		t.Fatal("push sent for an invalid session")
	}
	if subs, _ := e.svc.ListPushSubscriptions(ctx, a); len(subs) != 0 {
		t.Fatal("subscription of a dead session stays active")
	}
	if _, err := e.svc.OnLogout(ctx, actor(net1, "alice")); err == nil {
		t.Fatal("OnLogout without session id must fail")
	}
}

func TestRevocationIsOwnerScoped(t *testing.T) {
	e := newEnv(t)
	sub := e.pushSub(actor(net1, "alice"), fcm)
	for _, a := range []n.Actor{actor(net1, "bob"), actor(net2, "alice")} {
		if err := e.svc.RevokePushSubscription(ctx, a, sub.ID); err != n.ErrNotFound {
			t.Fatalf("%v could revoke: %v", a, err)
		}
	}
	if err := e.svc.RevokePushSubscription(ctx, actor(net1, "alice"), sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.RevokePushSubscription(ctx, actor(net1, "alice"), sub.ID); err != n.ErrNotFound {
		t.Fatal("double revoke should be not found")
	}
}

func TestUnsubscribeLinkIsImmediateAndScoped(t *testing.T) {
	e := newEnv(t)
	a, b := actor(net1, "alice"), actor(net1, "bob")
	e.optIn(a, "email", n.KindDispute, n.KindProposalReceived)
	e.optIn(a, "push", n.KindDispute)
	e.optIn(b, "email", n.KindDispute)
	e.verifiedEmail(a, "a@example.org")
	e.verifiedEmail(b, "b@example.org")
	got := e.enqueue(ev("ev1", "alice", n.KindDispute))
	bob := e.enqueue(ev("ev2", "bob", n.KindDispute))
	tok := e.svc.UnsubscribeToken(net1, "alice", "email")
	if err := e.svc.Unsubscribe(ctx, tok); err != nil {
		t.Fatal(err)
	}
	if err := e.svc.Unsubscribe(ctx, tok); err != nil {
		t.Fatal("must be idempotent:", err)
	}
	for _, id := range []string{got[0].DeliveryID} {
		if s, why := e.status(id); s != "cancelled" || why != "unsubscribed" {
			t.Fatalf("%s/%s", s, why)
		}
	}
	if s, _ := e.status(bob[0].DeliveryID); s != "ready" {
		t.Fatal("unsubscribe touched another member")
	}
	prefs, _ := e.svc.ListPreferences(ctx, a)
	for _, p := range prefs {
		if p.Channel == "email" && p.Enabled {
			t.Fatal("email preference still on")
		}
		if p.Channel == "push" && p.EventType == n.KindDispute && !p.Enabled {
			t.Fatal("unsubscribe disabled the push channel")
		}
	}
	for _, bad := range []string{"", "x", tok + "A", e.svc.UnsubscribeToken(net1, "alice", "sms")} {
		if err := e.svc.Unsubscribe(ctx, bad); err != n.ErrToken {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	// A confirm token is not an unsubscribe token.
	if err := e.svc.Unsubscribe(ctx, "v1.e30.AAAA"); err != n.ErrToken {
		t.Fatal("forged token accepted")
	}
	// Unsubscribe works even after the member left the network.
	e.dir.SetActive(net1, "bob", false)
	if err := e.svc.Unsubscribe(ctx, e.svc.UnsubscribeToken(net1, "bob", "email")); err != nil {
		t.Fatal(err)
	}
}

func TestTestKindRequiresOptIn(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.pushSub(a, fcm)
	send := func(id string) {
		err := e.st.Update(ctx, func(ctx context.Context, q n.DBTX) error {
			_, err := e.svc.EnqueueTestTx(ctx, q, net1, "alice", n.ChannelPush, id)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	send("t1")
	e.run(e.disp("w", n.Hooks{}))
	if len(e.push.Sent()) != 0 {
		t.Fatal("test notification sent to a member who never opted in")
	}
	e.optIn(a, "push", n.KindAccountSecurity)
	send("t2")
	e.run(e.disp("w2", n.Hooks{}))
	if len(e.push.Sent()) != 1 {
		t.Fatal("opted-in member did not get the test")
	}
}

func TestUnconfiguredChannelsAreNotClaimed(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "e", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(time.Hour)})
	d, err := e.svc.NewDispatcher(n.DispatcherOptions{Owner: "mail-only", Email: e.email})
	if err != nil {
		t.Fatal(err)
	}
	if st := e.run(d); st.Claimed != 0 {
		t.Fatalf("%+v", st)
	}
	if s, _ := e.status(got[0].DeliveryID); s != "ready" {
		t.Fatal(s)
	}
	e.clk.Add(2 * time.Hour)
	e.run(d)
	if s, why := e.status(got[0].DeliveryID); s != "expired" {
		t.Fatalf("%s/%s", s, why)
	}
	// A dispatcher with no transports at all is valid.
	if _, err := e.svc.NewDispatcher(n.DispatcherOptions{}); err != nil {
		t.Fatal(err)
	}
}

func TestMultiTenantGateAtDispatch(t *testing.T) {
	e := newEnv(t)
	e.optIn(actor(net1, "alice"), "push", n.KindDispute)
	e.pushSub(actor(net1, "alice"), fcm)
	e.pushSub(actor(net2, "alice"), "https://updates.push.services.mozilla.com/wpush/v2/net2")
	// Only net-1 opted in: a net-2 event for the same member id is never queued.
	got := e.enqueue(n.Event{NetworkID: net2, EventID: "ev", MemberID: "alice", Kind: n.KindDispute})
	if len(got) != 0 {
		t.Fatal("cross-network opt-in leak")
	}
	e.enqueue(ev("ev", "alice", n.KindDispute))
	e.run(e.disp("w", n.Hooks{}))
	for _, m := range e.push.Sent() {
		if strings.Contains(m.Endpoint, "net2") {
			t.Fatal("delivered to the other network's subscription")
		}
	}
}

func TestStatusAndInspectionAreRedacted(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	e.pushSub(a, fcm)
	e.enqueue(ev("ev1", "alice", n.KindDispute))
	e.run(e.disp("w", n.Hooks{}))
	st, err := e.svc.Status(ctx)
	if err != nil || st.ActivePushSubs != 1 || len(st.Deliveries) != 1 || st.Deliveries[0].Status != "sent" {
		t.Fatalf("%+v %v", st, err)
	}
	rows, _ := e.svc.ListDeliveries(ctx, "", 10)
	var sb strings.Builder
	e.svc.Metrics().WritePrometheus(&sb, &st)
	all := sb.String()
	for _, r := range rows {
		all += r.ID + r.Channel + r.Kind + r.Status + r.LastError
	}
	for _, leak := range []string{"alice", "net-1", "ev1", "fcm", "googleapis"} {
		if strings.Contains(all, leak) {
			t.Fatalf("diagnostics leak %q", leak)
		}
	}
	if !strings.Contains(sb.String(), "circula_notifications_sent_total 1") {
		t.Fatal(sb.String())
	}
}

func TestPurgeAndRetryDelivery(t *testing.T) {
	e := newEnv(t)
	a := actor(net1, "alice")
	e.optIn(a, "push", n.KindDispute)
	sub := e.pushSub(a, fcm)
	e.push.fn = func(n.PushMessage) error { return n.NewSendError(n.ClassPermanent, "provider_4xx", 0, nil) }
	got := e.enqueue(n.Event{NetworkID: net1, EventID: "e", MemberID: "alice", Kind: n.KindDispute, ExpiresAt: t0.Add(72 * time.Hour)})
	e.run(e.disp("w", n.Hooks{}))
	if s, _ := e.status(got[0].DeliveryID); s != "failed" {
		t.Fatal(s)
	}
	e.push.fn = nil
	if err := e.svc.RetryDelivery(ctx, got[0].DeliveryID); err != nil {
		t.Fatal(err)
	}
	e.run(e.disp("w2", n.Hooks{}))
	if s, _ := e.status(got[0].DeliveryID); s != "sent" {
		t.Fatal(s)
	}
	if err := e.svc.RetryDelivery(ctx, got[0].DeliveryID); err != n.ErrNotFound {
		t.Fatal("retry of a sent row must be refused")
	}
	if _, err := e.svc.Purge(ctx, time.Minute); err == nil {
		t.Fatal("retention floor not enforced")
	}
	_ = e.svc.RevokePushSubscription(ctx, a, sub.ID)
	e.clk.Add(40 * 24 * time.Hour)
	cnt, err := e.svc.Purge(ctx, 30*24*time.Hour)
	if err != nil || cnt != 1 {
		t.Fatalf("%d %v", cnt, err)
	}
	if e.count(`SELECT COUNT(*) FROM push_subscriptions`) != 0 || e.count(`SELECT COUNT(*) FROM notification_delivery_target`) != 0 {
		t.Fatal("purge left rows")
	}
	if e.count(`SELECT COUNT(*) FROM notification_audit`) == 0 {
		t.Fatal("audit must be retained")
	}
}

var _ = notifytest.NewClock
