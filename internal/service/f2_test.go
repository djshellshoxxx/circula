package service_test

import (
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// FR-067: every count comes from live data, with the thresholds stated.
func TestAdminOverviewCounts(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) {
		p.DefaultCreditLimit = 2000
		p.DisputeWindowDays = 30
		p.JoinMode = domain.JoinApproval
	})
	ov := tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.PendingMembers != 0 || ov.StaleProposals != 0 || ov.OpenDisputes != 0 || ov.PostingPending != 0 ||
		ov.NearLimit != 0 || ov.ExpiringInvites != 0 || ov.LastCheck != nil {
		t.Fatalf("fresh network overview = %+v", ov)
	}
	if !ov.AsOf.Equal(w.Clock.Now().Truncate(time.Millisecond)) || ov.StaleAfter != service.OverviewStaleAfter {
		t.Errorf("asof/threshold = %v %v", ov.AsOf, ov.StaleAfter)
	}

	// near limit: a (limit 20.00) pays 20.00 and is at its limit.
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2000)
	// stale proposal: ben proposes to ana, nobody answers for 49 hours (proposal expiry is longer).
	tu.Must(w.propose(w.b, w.aM, service.ProposerPays, 100))
	// pending member + expiring invitation + dispute
	pat := w.Person("pat@example.org")
	if m := tu.Must(w.Svc.JoinNetwork(w.Ctx, pat, w.net.ID, "Pat")); m.Status != domain.MemberPending {
		t.Fatalf("join status = %s", m.Status)
	}
	tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{TTL: 60 * time.Hour}))
	c, cM := w.Member2(w.admin, "carol@example.org")
	tx := w.Trade(w.net.ID, c, w.b, cM, w.bM, 100)
	tu.Must(w.open(w.b, tx.ID))

	w.Clock.Advance(49 * time.Hour)
	ov = tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.NearLimit != 1 || ov.PendingMembers != 1 {
		t.Fatalf("near limit = %d pending = %d", ov.NearLimit, ov.PendingMembers)
	}
	if ov.StaleProposals != 1 {
		t.Errorf("stale proposals = %d, want 1", ov.StaleProposals)
	}
	if ov.OpenDisputes != 1 {
		t.Errorf("open disputes = %d, want 1", ov.OpenDisputes)
	}
	if ov.ExpiringInvites != 1 { // 60h invite, 49h later: 11h left, inside the 72h window
		t.Errorf("expiring invites = %d, want 1", ov.ExpiringInvites)
	}

	// a recorded check shows up with its time and result
	tu.Must(w.Svc.VerifyAndRecordLedger(w.Ctx, w.admin, w.net.ID))
	ov = tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.LastCheck == nil || !ov.LastCheck.OK {
		t.Errorf("last check = %+v", ov.LastCheck)
	}

	// negative: members and outsiders are refused
	_, err := w.Svc.AdminOverview(w.Ctx, w.a, w.net.ID)
	wantCode(t, err, domain.ErrForbidden)
	other := w.Person("out@example.org")
	_, err = w.Svc.AdminOverview(w.Ctx, other, w.net.ID)
	wantCode(t, err, domain.ErrNotFound)
}

func TestOverviewNearLimitAndPending(t *testing.T) {
	w := newWorld(t, func(p *domain.Policy) { p.DefaultCreditLimit = 2000 })
	w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 2000)
	ov := tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.NearLimit != 1 {
		t.Fatalf("at the limit: near limit = %d, want 1", ov.NearLimit)
	}
	// an audited override above the debt takes ana out of the count
	lim := int64(5000)
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lim, Reason: "more room"}))
	ov = tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID))
	if ov.NearLimit != 0 {
		t.Errorf("after raising the limit: near limit = %d, want 0", ov.NearLimit)
	}
	// lowering below the debt puts ana beyond the limit
	lim = 1000
	tu.Must(0, w.Svc.SetCreditLimit(w.Ctx, w.admin, w.net.ID, service.SetLimitInput{MemberID: w.aM.ID, Limit: &lim, Reason: "less room"}))
	if ov = tu.Must(w.Svc.AdminOverview(w.Ctx, w.admin, w.net.ID)); ov.NearLimit != 1 {
		t.Errorf("beyond the limit: near limit = %d, want 1", ov.NearLimit)
	}
}

func TestAnnouncements(t *testing.T) {
	w := newWorld(t, nil)
	key := func() string { return w.Key() }
	a := tu.Must(w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, service.AnnouncementInput{Title: " Market day ", Body: "Saturday at ten.", IdempotencyKey: key()}))
	if a.Title != "Market day" || a.CreatedBy != w.adminM.ID || a.DeletedAt != nil {
		t.Fatalf("announcement = %+v", a)
	}
	// members read it; a scheduled one stays hidden from them but not from the administrator
	future := tu.Must(w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, service.AnnouncementInput{Title: "Later", Body: "soon",
		EffectiveAt: w.Clock.Now().Add(48 * time.Hour), IdempotencyKey: key()}))
	got := tu.Must(w.Svc.ListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{}))
	if len(got) != 1 || got[0].ID != a.ID {
		t.Fatalf("member list = %+v", got)
	}
	if adm := tu.Must(w.Svc.AdminListAnnouncements(w.Ctx, w.admin, w.net.ID, service.Page{})); len(adm) != 2 || adm[0].ID != future.ID {
		t.Fatalf("admin list = %+v", adm)
	}
	w.Clock.Advance(49 * time.Hour)
	if got = tu.Must(w.Svc.ListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{})); len(got) != 2 {
		t.Fatalf("after effective time: %d", len(got))
	}

	// idempotent replay creates nothing new; a reused key with another body is refused
	k := key()
	in := service.AnnouncementInput{Title: "Once", Body: "only once", IdempotencyKey: k}
	first := tu.Must(w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, in))
	again := tu.Must(w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, in))
	if first.ID != again.ID {
		t.Error("replay created a second announcement")
	}
	in.Body = "different"
	_, err := w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, in)
	wantCode(t, err, domain.ErrKeyReused)

	// delete retracts for everyone; a second delete is not-found; the row stays for audit
	tu.Must(0, w.Svc.DeleteAnnouncement(w.Ctx, w.admin, w.net.ID, a.ID))
	for _, x := range tu.Must(w.Svc.ListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{})) {
		if x.ID == a.ID {
			t.Error("deleted announcement still listed")
		}
	}
	wantCode(t, w.Svc.DeleteAnnouncement(w.Ctx, w.admin, w.net.ID, a.ID), domain.ErrNotFound)
	var n int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM announcements WHERE id = ? AND deleted_at IS NOT NULL`, a.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("retracted row kept: n=%d err=%v", n, err)
	}
	if _, err := w.Store.DB().Exec(`DELETE FROM announcements WHERE id = ?`, a.ID); err == nil {
		t.Error("raw delete of an announcement succeeded")
	}
	if _, err := w.Store.DB().Exec(`UPDATE announcements SET title = 'x' WHERE id = ?`, future.ID); err == nil {
		t.Error("raw edit of an announcement succeeded")
	}

	// validation and authorisation
	for name, in := range map[string]service.AnnouncementInput{
		"empty title": {Title: " ", Body: "b", IdempotencyKey: key()},
		"empty body":  {Title: "t", Body: "", IdempotencyKey: key()},
		"long title":  {Title: string(make([]rune, 121)) + "x", Body: "b", IdempotencyKey: key()},
		"past":        {Title: "t", Body: "b", EffectiveAt: w.Clock.Now().Add(-time.Hour), IdempotencyKey: key()},
		"far future":  {Title: "t", Body: "b", EffectiveAt: w.Clock.Now().Add(400 * 24 * time.Hour), IdempotencyKey: key()},
	} {
		if _, err := w.Svc.CreateAnnouncement(w.Ctx, w.admin, w.net.ID, in); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	_, err = w.Svc.CreateAnnouncement(w.Ctx, w.a, w.net.ID, service.AnnouncementInput{Title: "t", Body: "b", IdempotencyKey: key()})
	wantCode(t, err, domain.ErrForbidden)
	wantCode(t, w.Svc.DeleteAnnouncement(w.Ctx, w.a, w.net.ID, future.ID), domain.ErrForbidden)
	_, err = w.Svc.AdminListAnnouncements(w.Ctx, w.a, w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrForbidden)
	_, err = w.Svc.ListAnnouncements(w.Ctx, w.Person("out2@example.org"), w.net.ID, service.Page{})
	wantCode(t, err, domain.ErrNotFound)
	// another network's administrator cannot delete it
	other := w.Person("boss2@example.org")
	n2, _ := w.Network(other, "other", nil)
	wantCode(t, w.Svc.DeleteAnnouncement(w.Ctx, other, n2.ID, future.ID), domain.ErrNotFound)
}

func TestTimeZonePreference(t *testing.T) {
	w := newWorld(t, nil)
	if z := tu.Must(w.Svc.GetTimeZone(w.Ctx, w.a)); z != "" {
		t.Fatalf("default zone = %q", z)
	}
	tu.Must(0, w.Svc.SetTimeZone(w.Ctx, w.a, "Europe/Berlin"))
	if z := tu.Must(w.Svc.GetTimeZone(w.Ctx, w.a)); z != "Europe/Berlin" {
		t.Errorf("zone = %q", z)
	}
	if z := tu.Must(w.Svc.GetTimeZone(w.Ctx, w.b)); z != "" {
		t.Errorf("another person's zone leaked: %q", z)
	}
	tu.Must(0, w.Svc.SetTimeZone(w.Ctx, w.a, "America/Argentina/Buenos_Aires"))
	tu.Must(0, w.Svc.SetTimeZone(w.Ctx, w.a, "")) // back to UTC
	if z := tu.Must(w.Svc.GetTimeZone(w.Ctx, w.a)); z != "" {
		t.Errorf("cleared zone = %q", z)
	}
	for _, bad := range []string{"Mars/Olympus", "Local", "../etc/passwd", "europe/berlin ", "Europe\\Berlin", "UTC+2", "Europe/Berlin\x00", "/UTC"} {
		if err := w.Svc.SetTimeZone(w.Ctx, w.a, bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if service.ValidTimeZone("Local") || !service.ValidTimeZone("UTC") || !service.ValidTimeZone("Etc/GMT+5") {
		t.Error("ValidTimeZone")
	}
	if service.Location("Asia/Kolkata").String() != "Asia/Kolkata" || service.Location("junk") != time.UTC {
		t.Error("Location")
	}
	// the change is audited at platform scope
	var n int
	if err := w.Store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE action = ?`, domain.ActTimeZoneChanged).Scan(&n); err != nil || n != 3 {
		t.Errorf("audit events = %d err=%v", n, err)
	}
	// read-only tokens cannot change it
	ro := w.a
	ro.Scopes = []domain.Scope{domain.ScopeRead}
	wantCode(t, w.Svc.SetTimeZone(w.Ctx, ro, "Asia/Tokyo"), domain.ErrForbidden)
}

func TestDirectorySearchFoldsCaseAndAccents(t *testing.T) {
	w := newWorld(t, nil)
	names := []string{"Émile Zola", "İstanbul Ayşe", "ISPARTA Işık", "Zoë Ångström", "Ana Maria", "Straße Fritz"}
	for i, n := range names {
		a := w.Person("n" + string(rune('a'+i)) + "@example.org")
		tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
		tu.Must(w.Svc.RedeemInvitation(w.Ctx, a, tok, n))
	}
	// a decomposed name: "Jose" + U+0301
	a := w.Person("nz@example.org")
	tok, _ := tu.Must2(w.Svc.CreateInvitation(w.Ctx, w.admin, w.net.ID, service.InviteInput{}))
	tu.Must(w.Svc.RedeemInvitation(w.Ctx, a, tok, "José Peña"))

	find := func(q string) []string {
		var out []string
		pg := tu.Must(w.Svc.ListDirectoryPage(w.Ctx, w.a, w.net.ID, service.MemberPageQuery{Name: q, Limit: 50}))
		for _, e := range pg.Entries {
			out = append(out, e.DisplayName)
		}
		if n := tu.Must(w.Svc.CountDirectory(w.Ctx, w.a, w.net.ID, q)); n != len(out) {
			t.Errorf("%q: count %d != %d rows", q, n, len(out))
		}
		return out
	}
	cases := []struct {
		q    string
		want string
	}{
		{"emile", "Émile Zola"}, {"ÉMILE", "Émile Zola"}, {"émile", "Émile Zola"}, // typed with a combining mark
		{"istanbul", "İstanbul Ayşe"}, {"İSTANBUL", "İstanbul Ayşe"}, {"ıstanbul", "İstanbul Ayşe"}, {"ayse", "İstanbul Ayşe"},
		{"isparta", "ISPARTA Işık"}, {"ısparta", "ISPARTA Işık"}, {"isik", "ISPARTA Işık"},
		{"zoe angstrom", "Zoë Ångström"}, {"ÅNGSTRÖM", "Zoë Ångström"},
		{"strasse", "Straße Fritz"}, {"STRASSE", "Straße Fritz"},
		{"jose", "José Peña"}, {"pena", "José Peña"}, {"JOSÉ", "José Peña"},
		{"ana m", "Ana Maria"},
	}
	for _, c := range cases {
		got := find(c.q)
		ok := false
		for _, g := range got {
			ok = ok || g == c.want
		}
		if !ok {
			t.Errorf("search %q = %v, want it to contain %q", c.q, got, c.want)
		}
	}
	// negatives: a different letter never matches, and no match is an empty page
	if got := find("emilio"); len(got) != 0 {
		t.Errorf("emilio = %v", got)
	}
	if got := find("zzz"); len(got) != 0 {
		t.Errorf("zzz = %v", got)
	}
	// an unaccented query must not match across unrelated names ("i" in many) but stays a substring test
	if got := find("ana m"); len(got) != 1 {
		t.Errorf("ana m = %v", got)
	}
}

func TestFindDisputesByTxAndStatus(t *testing.T) {
	w := newWorld(t, nil)
	t1 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
	t2 := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 200)
	d1 := tu.Must(w.open(w.a, t1.ID))
	d2 := tu.Must(w.open(w.b, t2.ID))

	got := tu.Must(w.Svc.FindDisputes(w.Ctx, w.admin, w.net.ID, service.DisputeQuery{TxID: t1.ID}))
	if len(got) != 1 || got[0].ID != d1.ID {
		t.Fatalf("by tx = %+v", got)
	}
	// paging with no window: 1 per page walks both, newest first
	p1 := tu.Must(w.Svc.FindDisputes(w.Ctx, w.admin, w.net.ID, service.DisputeQuery{Page: service.Page{Limit: 1}}))
	p2 := tu.Must(w.Svc.FindDisputes(w.Ctx, w.admin, w.net.ID, service.DisputeQuery{Page: service.Page{Limit: 1, AfterID: p1[0].ID}}))
	p3 := tu.Must(w.Svc.FindDisputes(w.Ctx, w.admin, w.net.ID, service.DisputeQuery{Page: service.Page{Limit: 1, AfterID: p2[0].ID}}))
	if len(p1) != 1 || len(p2) != 1 || len(p3) != 0 || p1[0].ID != d2.ID || p2[0].ID != d1.ID {
		t.Fatalf("pages = %v %v %v", p1, p2, p3)
	}
	// status filter
	if got = tu.Must(w.Svc.FindDisputes(w.Ctx, w.admin, w.net.ID, service.DisputeQuery{Statuses: []domain.DisputeStatus{domain.DisputeResolved}})); len(got) != 0 {
		t.Errorf("resolved = %v", got)
	}
	// a non-party member sees nothing about someone else's trade
	c, _ := w.Member2(w.admin, "carol@example.org")
	if got = tu.Must(w.Svc.FindDisputes(w.Ctx, c, w.net.ID, service.DisputeQuery{TxID: t1.ID})); len(got) != 0 {
		t.Errorf("outsider saw %v", got)
	}
	// a party sees theirs
	if got = tu.Must(w.Svc.FindDisputes(w.Ctx, w.a, w.net.ID, service.DisputeQuery{TxID: t1.ID})); len(got) != 1 {
		t.Errorf("party saw %v", got)
	}
	// a person outside the network is refused
	_, err := w.Svc.FindDisputes(w.Ctx, w.Person("out@example.org"), w.net.ID, service.DisputeQuery{})
	wantCode(t, err, domain.ErrNotFound)
}
