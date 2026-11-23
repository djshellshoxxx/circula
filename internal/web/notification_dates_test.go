package web

import (
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/repository"
	"github.com/djshellshoxxx/circula/internal/service"
)

func TestInboxEventDatesFollowCurrentViewerZone(t *testing.T) {
	w := newWorld(t)
	at := time.Date(2026, 3, 1, 0, 15, 0, 0, time.UTC)
	const reason = "voted <yes> & 100% %!"
	events := []struct{ kind, event string }{
		{service.NotifPolicyNotice, domain.NoticePolicyEffective},
		{service.NotifClosureScheduled, domain.NoticeClosureScheduled},
		{service.NotifDisputeOverdue, domain.NoticeResponseDue},
		{service.NotifDisputeOverdue, domain.NoticeDecisionDue},
		{service.NotifPolicyNotice, ""},
	}
	for _, e := range events {
		n := repository.Notification{ID: w.env.Key(), NetworkID: w.net.ID, MemberID: w.aliceM.ID,
			Kind: e.kind, Title: "Announcement", Body: "Archived wording on 1 Mar 2026", CreatedAt: at}
		if e.event != "" {
			n.EventDate = &domain.NotificationDate{Event: e.event, At: at}
			if e.event == domain.NoticeClosureScheduled {
				n.EventDate.ClosureReason = reason
				n.EventDate.ReadOnlyDays = 90
			}
		}
		if err := w.env.Store.Update(w.env.Ctx, func(tx repository.Tx) error {
			_, err := tx.InsertNotification(n)
			return err
		}); err != nil {
			t.Fatal(err)
		}
	}
	al := w.browser(t)
	al.login("alice@example.org")
	for _, tc := range []struct{ zone, iso, date string }{{"", "2026-03-01", "1 Mar 2026"}, {"America/Vancouver", "2026-02-28", "28 Feb 2026"}, {"Pacific/Kiritimati", "2026-03-01", "1 Mar 2026"}} {
		if err := w.env.Svc.SetTimeZone(w.env.Ctx, w.alice, tc.zone); err != nil {
			t.Fatal(err)
		}
		p := al.get(w.base() + "/inbox")
		date := `<time datetime="` + tc.iso + `">` + tc.date + `</time>`
		al.mustContain(p, "takes effect on "+date+".", "closes on "+date+": "+reason+".", "90 days afterwards.",
			"did not respond by "+date+".", "decision target of "+date+" has passed.", "Archived wording on 1 Mar 2026")
		if strings.Contains(p.Body, "<yes>") {
			t.Fatal("closure reason became markup")
		}
	}
}
