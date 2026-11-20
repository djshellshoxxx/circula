package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// SEC-018 / A3-F25: related notifications must be erased by their resource
// links. Matching names (including a single character) must not erase someone
// else's unrelated inbox entries.
func TestNotificationErasureUsesResourceRelations(t *testing.T) {
	for _, name := range []string{"e", "ana@example.org"} {
		t.Run(name, func(t *testing.T) {
			w := newWorld(t, nil)
			other, otherMember := w.Member2(w.admin, "same-name@example.org")
			if err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
				for _, member := range []string{w.aM.ID, otherMember.ID} {
					if err := tx.SetMemberDisplayName(w.net.ID, member, name); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			related := w.Trade(w.net.ID, w.a, w.b, w.aM, w.bM, 100)
			unrelated := w.Trade(w.net.ID, other, w.b, otherMember, w.bM, 100)
			relatedDispute := tu.Must(w.Svc.OpenDispute(w.Ctx, w.a, w.net.ID,
				service.OpenDisputeInput{TxID: related.ID, Category: "other", IdempotencyKey: w.Key()}))
			tu.Must(w.Svc.WithdrawDispute(w.Ctx, w.a, w.net.ID, relatedDispute.ID))
			otherDispute := tu.Must(w.Svc.OpenDispute(w.Ctx, other, w.net.ID,
				service.OpenDisputeInput{TxID: unrelated.ID, Category: "other", IdempotencyKey: w.Key()}))
			tu.Must(w.Svc.WithdrawDispute(w.Ctx, other, w.net.ID, otherDispute.ID))
			foreign, foreignAdmin := w.Network(w.admin, "other-network", nil)
			relatedLinks := map[string]bool{
				"/trades/" + related.ID:          true,
				"/disputes/" + relatedDispute.ID: true,
				"/members/" + w.aM.ID:            true,
				"/admin/members/" + w.aM.ID:      true,
			}
			if err := w.Store.Update(w.Ctx, func(tx store.Tx) error {
				for _, n := range []store.Notification{
					{ID: w.Key(), NetworkID: w.net.ID, MemberID: w.bM.ID, Link: "/members/" + w.aM.ID},
					{ID: w.Key(), NetworkID: w.net.ID, MemberID: w.bM.ID, Link: "/admin/members/" + w.aM.ID},
					{ID: w.Key(), NetworkID: w.net.ID, MemberID: w.bM.ID, Link: "/"},
					{ID: w.Key(), NetworkID: w.net.ID, MemberID: w.bM.ID, Link: "/trades/" + related.ID + "-different"},
					{ID: w.Key(), NetworkID: w.net.ID, MemberID: w.bM.ID, Link: "/members/" + otherMember.ID},
					{ID: w.Key(), NetworkID: foreign.ID, MemberID: foreignAdmin.ID, Link: "/trades/" + related.ID},
				} {
					n.Kind, n.Title, n.Body, n.CreatedAt = "test.notice", "Notice", "Message containing "+name, w.Clock.Now()
					if _, err := tx.InsertNotification(n); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			w.depart(w.admin, w.aM)
			type beforeRow struct {
				id, body, link string
				redact         bool
			}
			var before []beforeRow
			rows, err := w.Store.DB().Query(`SELECT id, network_id, member_id, body, link FROM notifications`)
			if err != nil {
				t.Fatal(err)
			}
			var relatedNames, unrelatedNames int
			for rows.Next() {
				var n beforeRow
				var net, member string
				if err := rows.Scan(&n.id, &net, &member, &n.body, &n.link); err != nil {
					t.Fatal(err)
				}
				n.redact = net == w.net.ID && (member == w.aM.ID || relatedLinks[n.link])
				if strings.Contains(n.body, name) {
					if n.redact {
						relatedNames++
					} else {
						unrelatedNames++
					}
				}
				before = append(before, n)
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			_ = rows.Close()
			if relatedNames < 2 || unrelatedNames < 2 {
				t.Fatalf("fixture needs related and unrelated matching names: %d / %d", relatedNames, unrelatedNames)
			}
			tu.Must(w.Svc.AnonymiseMember(w.Ctx, domain.SystemActor("test"), w.net.ID, w.aM.ID, "erasure request"))
			for _, n := range before {
				var body, link string
				if err := w.Store.DB().QueryRow(`SELECT body, link FROM notifications WHERE id = ?`, n.id).Scan(&body, &link); err != nil {
					t.Fatal(err)
				}
				want := n.body
				if n.redact {
					want = ""
				}
				if body != want || link != n.link {
					t.Errorf("notification %s (%s), redact=%v: body %q, want %q; link %q", n.id, n.link, n.redact, body, want, link)
				}
			}
		})
	}
}
