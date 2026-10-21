package web

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// balanceVM presents a balance with sign, word and spoken text (UX-004..006).
type balanceVM struct {
	Signed string // +12.50, −12.50, 0.00
	State  string // In credit | In debt | Level
	Spoken string
}

func unitFor(name string, v int64, dec int) string {
	one := int64(1)
	for i := 0; i < dec; i++ {
		one *= 10
	}
	if abs64(v) == one {
		return name
	}
	if strings.HasSuffix(strings.ToLower(name), "s") {
		return name
	}
	return name + "s"
}

func makeBalance(v int64, dec int, unit string) balanceVM {
	u := unitFor(unit, v, dec)
	n := domain.FormatAmount(abs64(v), dec)
	switch {
	case v > 0:
		return balanceVM{"+" + n, "In credit", fmt.Sprintf("In credit by %s %s", n, u)}
	case v < 0:
		return balanceVM{"−" + n, "In debt", fmt.Sprintf("In debt by %s %s", n, u)}
	}
	return balanceVM{n, "Level", fmt.Sprintf("Level, zero %s", unit+"s")}
}

// balanceCard is the Home balance card data (UX-009..011).
type balanceCard struct {
	Balance   balanceVM
	Limit     string
	Reserved  string
	Available string
	AvailVM   balanceVM
	Awaiting  string
	Used      string // text for credit in use
	UsedPct   int
	InDebt    bool
	Unit      string
}

func makeCard(b *service.BalanceView, unit string) *balanceCard {
	d := b.Decimals
	c := &balanceCard{Balance: makeBalance(b.Balance, d, unit), Limit: plainAmt(b.CreditLimit, d),
		Reserved: plainAmt(b.ReservedOutgoing, d), Available: minus(plainAmt(b.Available, d)),
		Awaiting: plainAmt(b.AwaitingFromOthers, d), Unit: unitFor(unit, 0, d)}
	if b.Balance < 0 {
		c.InDebt = true
		c.Used = fmt.Sprintf("You are using %s of your %s credit limit.", plainAmt(-b.Balance, d), plainAmt(b.CreditLimit, d))
		if b.CreditLimit > 0 {
			c.UsedPct = int(min64(100, (-b.Balance*100)/b.CreditLimit))
		}
	}
	return c
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// txRow is one transaction line from the viewer's side (UX-013..016).
type txRow struct {
	ID        string
	Text      string // You paid Alex 5.00 credits
	Signed    string // −5.00
	Status    string
	StatusKey domain.TxStatus
	Pending   bool
	Needs     bool // waiting for me
	When      time.Time
	Ledger    string
	Kind      string
	Counter   string
}

type names map[string]string

func (n names) of(id string) string {
	if v, ok := n[id]; ok {
		return v
	}
	if id == "" {
		return "the network"
	}
	return "another member"
}

func statusLabel(t *domain.Transaction, me *domain.Member, n names) (label string, needsMe bool) {
	switch t.Status {
	case domain.TxProposed:
		if t.CreatedBy == me.ID {
			other := t.PayeeMemberID
			if other == me.ID {
				other = t.PayerMemberID
			}
			return "Waiting for " + n.of(other) + " to respond", false
		}
		return "Waiting for you to respond", true
	case domain.TxCompleted, domain.TxAccepted:
		return "Completed", false
	case domain.TxDeclined:
		return "Declined", false
	case domain.TxCancelled:
		return "Cancelled", false
	case domain.TxExpired:
		return "Expired", false
	case domain.TxDisputed:
		return "Disputed", false
	case domain.TxReversed:
		return "Reversed", false
	}
	return string(t.Status), false
}

func statusExplain(s domain.TxStatus) string {
	switch s {
	case domain.TxProposed:
		return "Nothing has been posted yet. The amount is held aside from the payer's available credit until it is accepted, declined, cancelled or expires."
	case domain.TxCompleted, domain.TxAccepted:
		return "Posted to both accounts."
	case domain.TxDeclined:
		return "The other party declined. Nothing was posted."
	case domain.TxCancelled:
		return "Withdrawn before it was accepted. Nothing was posted."
	case domain.TxExpired:
		return "Not answered in time. Nothing was posted."
	case domain.TxDisputed:
		return "A dispute is open. Postings stay as they were until a reviewer decides."
	case domain.TxReversed:
		return "Fully reversed by a later posting."
	}
	return ""
}

func makeRow(t *domain.Transaction, me *domain.Member, n names, unit string, dec int) txRow {
	label, needs := statusLabel(t, me, n)
	r := txRow{ID: t.ID, Status: label, StatusKey: t.Status, Pending: t.Status == domain.TxProposed, Needs: needs,
		When: t.ProposedAt, Kind: string(t.Kind)}
	if t.CompletedAt != nil && !r.Pending {
		r.When = *t.CompletedAt
	}
	if t.LedgerSeq > 0 && (t.Status == domain.TxCompleted || t.Status == domain.TxDisputed || t.Status == domain.TxReversed) {
		r.Ledger = fmt.Sprintf("Ledger reference %d", t.LedgerSeq)
	}
	amt := domain.FormatAmount(t.Amount, dec)
	u := unitFor(unit, t.Amount, dec)
	prefix := ""
	switch t.Kind {
	case domain.KindReversal:
		prefix = "Reversal: "
	case domain.KindAdjustment:
		prefix = "Adjustment: "
	case domain.KindGrant:
		prefix = "Sign-up grant: "
	}
	if t.PayerMemberID == me.ID {
		r.Counter = n.of(t.PayeeMemberID)
		r.Text = fmt.Sprintf("%sYou paid %s %s %s", prefix, r.Counter, amt, u)
		r.Signed = "−" + amt
		if t.Status == domain.TxProposed {
			r.Text = fmt.Sprintf("%sYou pay %s %s %s", prefix, r.Counter, amt, u)
		}
	} else {
		r.Counter = n.of(t.PayerMemberID)
		r.Text = fmt.Sprintf("%sYou received %s %s from %s", prefix, amt, u, r.Counter)
		r.Signed = "+" + amt
		if t.Status == domain.TxProposed {
			r.Text = fmt.Sprintf("%sYou will receive %s %s from %s", prefix, amt, u, r.Counter)
		}
	}
	return r
}

// memberNames loads display names for active members plus the caller.
func (c *rc) memberNames() names {
	n := names{}
	after := ""
	for i := 0; i < 20; i++ {
		es, err := c.s.svc.ListDirectory(c.ctx, c.actor, c.netID(), service.Page{AfterID: after, Limit: 500})
		if err != nil || len(es) == 0 {
			break
		}
		for _, e := range es {
			n[e.MemberID] = e.DisplayName
		}
		after = es[len(es)-1].MemberID
		if len(es) < 500 {
			break
		}
	}
	if c.isAdmin() {
		if ms, err := c.s.svc.AdminListMembers(c.ctx, c.actor, c.netID(), nil, service.Page{Limit: 500}); err == nil {
			for _, m := range ms {
				n[m.ID] = m.DisplayName
			}
		}
	}
	n[c.Me.ID] = c.Me.DisplayName
	return n
}

func itoa(i int) string { return strconv.Itoa(i) }
