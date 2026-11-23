package web

import (
	"strconv"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
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
	return messages.Text("unit.plural", name)
}

func makeBalance(v int64, dec int, unit string) balanceVM {
	u := unitFor(unit, v, dec)
	n := domain.FormatAmount(abs64(v), dec)
	switch {
	case v > 0:
		return balanceVM{"+" + n, messages.Text("balance.credit"), messages.Text("balance.spoken_credit", n, u)}
	case v < 0:
		return balanceVM{"−" + n, messages.Text("balance.debt"), messages.Text("balance.spoken_debt", n, u)}
	}
	return balanceVM{n, messages.Text("balance.level"), messages.Text("balance.spoken_level", u)}
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
}

func makeCard(b *service.BalanceView, unit string) *balanceCard {
	d := b.Decimals
	amount := func(v int64) string {
		return minus(plainAmt(v, d)) + " " + unitFor(unit, v, d)
	}
	c := &balanceCard{Balance: makeBalance(b.Balance, d, unit), Limit: amount(b.CreditLimit),
		Reserved: amount(b.ReservedOutgoing), Available: amount(b.Available), Awaiting: amount(b.AwaitingFromOthers)}
	if b.Balance < 0 {
		c.InDebt = true
		c.Used = messages.Text("balance.limit_used", plainAmt(-b.Balance, d), plainAmt(b.CreditLimit, d))
		if b.CreditLimit > 0 {
			// Credit limits are bounded by MaxPolicyAmount. Clamp the potentially
			// much larger balance before scaling to avoid int64 overflow.
			c.UsedPct = int(min(-b.Balance, b.CreditLimit) * 100 / b.CreditLimit)
		}
	}
	return c
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
	// UX-017: a reversal or adjustment line shows its own reason; a reversal links to the
	// line it reverses (Original); an original links to each reversal posted against it
	// ("Reversed by …"), oldest first.
	Reason     string
	Original   string
	Related    string
	ReversedBy []txLink
}

// txLink is a link from one transaction line to a related one.
type txLink struct {
	ID   string
	Text string
}

// names resolves member display names for one request. Names the request already holds are
// kept in known; any other id is fetched on first use through the service, which applies the
// usual visibility rules, so no list cap can hide a name. A miss is remembered as nil.
type names struct {
	c     *rc
	known map[string]*string
}

// of returns the display name for id, or a neutral label when the caller may not see it.
func (n names) of(id string) string {
	if v, ok := n.lookup(id); ok {
		return v
	}
	if id == "" {
		return messages.Text("member.network")
	}
	return messages.Text("member.hidden")
}

// lookup returns the display name for id when the caller may see it.
func (n names) lookup(id string) (string, bool) {
	if v, ok := n.known[id]; ok {
		if v == nil {
			return "", false
		}
		return *v, true
	}
	if id == "" || n.c == nil {
		return "", false
	}
	m, err := n.c.s.svc.GetMember(n.c.ctx, n.c.actor, n.c.netID(), id)
	if err != nil || m == nil {
		n.known[id] = nil
		return "", false
	}
	name := m.DisplayName
	n.known[id] = &name
	return name, true
}

// reversals returns the reversals posted against transaction id, oldest first, under the
// service's visibility rules. A failed lookup shows no links rather than failing the page.
func (n names) reversals(id string) []*domain.Transaction {
	if n.c == nil {
		return nil
	}
	revs, err := n.c.s.svc.ReversalsOf(n.c.ctx, n.c.actor, n.c.netID(), id)
	if err != nil {
		return nil
	}
	return revs
}

// reversedBy returns the "Reversed by …" links of t, oldest first (UX-017). Only the pages
// that show them call it (History and trade detail), so other lists do not pay for one
// service lookup per reversed row.
func (n names) reversedBy(t *domain.Transaction, unit string, dec int) []txLink {
	if t.ReversedTotal <= 0 {
		return nil
	}
	var out []txLink
	for _, rev := range n.reversals(t.ID) {
		by := messages.Text("reversal.later")
		if rev.LedgerSeq > 0 {
			by = messages.Text("reversal.ledger", strconv.FormatInt(rev.LedgerSeq, 10))
		}
		out = append(out, txLink{ID: rev.ID, Text: messages.Text("reversal.by", by,
			domain.FormatAmount(rev.Amount, dec), unitFor(unit, rev.Amount, dec))})
	}
	return out
}

// memberRef is the stable member reference shown beside a display name in confirmation
// steps (SEC-016): display names are not unique, member ids are and never change. It is the
// last eight letters or digits of the id, upper-cased and split in two ("0A1B-2C3D"), or ""
// when there is no member (the network's own account).
func memberRef(id string) string {
	var b []byte
	for i := 0; i < len(id); i++ {
		if ch := id[i]; ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' {
			b = append(b, ch)
		}
	}
	if len(b) > 8 {
		b = b[len(b)-8:]
	}
	ref := strings.ToUpper(string(b))
	if len(ref) == 8 {
		ref = ref[:4] + "-" + ref[4:]
	}
	return ref
}

// MemberRef gives templates memberRef for a member id.
func (p *page) MemberRef(id string) string { return memberRef(id) }

func statusLabel(t *domain.Transaction, me *domain.Member, n names) (label string, needsMe bool) {
	switch t.Status {
	case domain.TxProposed:
		if t.CreatedBy == me.ID {
			other := t.PayeeMemberID
			if other == me.ID {
				other = t.PayerMemberID
			}
			return messages.Text("status.waiting_other", n.of(other)), false
		}
		return messages.Text("status.waiting_you"), true
	case domain.TxCompleted, domain.TxAccepted:
		return messages.Text("status.completed"), false
	case domain.TxDeclined:
		return messages.Text("status.declined"), false
	case domain.TxCancelled:
		return messages.Text("status.cancelled"), false
	case domain.TxExpired:
		return messages.Text("status.expired"), false
	case domain.TxDisputed:
		return messages.Text("status.disputed"), false
	case domain.TxReversed:
		return messages.Text("status.reversed"), false
	}
	return string(t.Status), false
}

func statusExplain(s domain.TxStatus) string {
	switch s {
	case domain.TxProposed:
		return messages.Text("status.explanation_1")
	case domain.TxCompleted, domain.TxAccepted:
		return messages.Text("status.explanation_2")
	case domain.TxDeclined:
		return messages.Text("status.explanation_3")
	case domain.TxCancelled:
		return messages.Text("status.explanation_4")
	case domain.TxExpired:
		return messages.Text("status.explanation_5")
	case domain.TxDisputed:
		return messages.Text("status.explanation_6")
	case domain.TxReversed:
		return messages.Text("status.explanation_7")
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
		r.Ledger = messages.Text("row.ledger", strconv.FormatInt(t.LedgerSeq, 10))
	}
	amt := domain.FormatAmount(t.Amount, dec)
	u := unitFor(unit, t.Amount, dec)
	kind := "trade"
	switch t.Kind {
	case domain.KindReversal:
		kind = "reversal"
	case domain.KindAdjustment:
		kind = "adjustment"
	case domain.KindGrant:
		kind = "grant"
	}
	if t.Kind == domain.KindReversal || t.Kind == domain.KindAdjustment {
		r.Reason = t.Reason
	}
	if t.Kind == domain.KindReversal {
		r.Original = t.OriginalTxID
	}
	if t.Kind == domain.KindAdjustment {
		r.Related = t.RelatedTxID
	}
	if t.PayerMemberID == me.ID {
		r.Counter = n.of(t.PayeeMemberID)
		r.Text = messages.Text("financial.row."+kind+".paid", r.Counter, amt, u)
		r.Signed = "−" + amt
		if t.Status == domain.TxProposed {
			r.Text = messages.Text("financial.row."+kind+".pay", r.Counter, amt, u)
		}
	} else {
		r.Counter = n.of(t.PayerMemberID)
		r.Text = messages.Text("financial.row."+kind+".received", r.Counter, amt, u)
		r.Signed = "+" + amt
		if t.Status == domain.TxProposed {
			r.Text = messages.Text("financial.row."+kind+".receive", r.Counter, amt, u)
		}
	}
	return r
}

// memberNames returns a resolver for member display names. The caller's own name is known up
// front; everything else is resolved on demand (see names.lookup).
func (c *rc) memberNames() names {
	n := names{c: c, known: map[string]*string{}}
	name := c.Me.DisplayName
	n.known[c.Me.ID] = &name
	return n
}

func itoa(i int) string { return strconv.Itoa(i) }
