package domain_test

import (
	"errors"
	"testing"

	d "github.com/djshellshoxxx/circula/internal/domain"
)

// permitted lists every allowed (status, command) pair from DOM-010; the test
// enumerates the full cross product so any other pair must be refused.
var permitted = map[d.TxStatus]map[d.TxCommand]d.TxStatus{
	d.TxProposed: {
		d.CmdAccept: d.TxCompleted, d.CmdDecline: d.TxDeclined, d.CmdCancel: d.TxCancelled, d.CmdExpire: d.TxExpired,
	},
	d.TxCompleted: {
		d.CmdOpenDispute: d.TxDisputed, d.CmdReverseFull: d.TxReversed, d.CmdReversePartial: d.TxCompleted,
	},
	d.TxDisputed: {
		d.CmdCloseDispute: d.TxCompleted, d.CmdReverseFull: d.TxReversed, d.CmdReversePartial: d.TxDisputed,
	},
}

func TestTxStateMachineExhaustive(t *testing.T) {
	n := 0
	for _, from := range d.AllTxStatuses {
		for _, cmd := range d.AllTxCommands {
			n++
			want, ok := permitted[from][cmd]
			got, err := d.NextTxStatus(from, cmd)
			if ok {
				if err != nil || got != want {
					t.Errorf("%s --%s--> got %s err=%v, want %s", from, cmd, got, err, want)
				}
				continue
			}
			if err == nil || !errors.Is(err, d.ErrInvalidState) {
				t.Errorf("%s --%s--> must be refused with invalid-state, got %s err=%v", from, cmd, got, err)
			}
			if got != from {
				t.Errorf("%s --%s--> refused transition changed state to %s", from, cmd, got)
			}
		}
	}
	if n != len(d.AllTxStatuses)*len(d.AllTxCommands) {
		t.Fatal("cross product incomplete")
	}
}

func TestTerminalStatuses(t *testing.T) {
	for _, s := range []d.TxStatus{d.TxDeclined, d.TxCancelled, d.TxExpired, d.TxReversed, d.TxAccepted} {
		if !s.IsTerminal() {
			t.Errorf("%s should have no outgoing transitions", s)
		}
	}
	for _, s := range []d.TxStatus{d.TxProposed, d.TxCompleted, d.TxDisputed} {
		if s.IsTerminal() {
			t.Errorf("%s must not be terminal", s)
		}
	}
}

func TestMemberTransitionsExhaustive(t *testing.T) {
	ok := map[[2]d.MemberStatus]bool{
		{d.MemberPending, d.MemberActive}: true, {d.MemberPending, d.MemberDeparted}: true,
		{d.MemberActive, d.MemberSuspended}: true, {d.MemberSuspended, d.MemberActive}: true,
		{d.MemberActive, d.MemberDeparted}: true, {d.MemberSuspended, d.MemberDeparted}: true,
	}
	for _, from := range d.AllMemberStatuses {
		for _, to := range d.AllMemberStatuses {
			err := d.CheckMemberTransition(from, to)
			if ok[[2]d.MemberStatus{from, to}] != (err == nil) {
				t.Errorf("%s -> %s: permitted=%v err=%v", from, to, ok[[2]d.MemberStatus{from, to}], err)
			}
		}
	}
}

func TestAccountStateDerivedFromMembership(t *testing.T) {
	want := map[d.MemberStatus]d.AccountState{d.MemberActive: d.AccountActive, d.MemberSuspended: d.AccountFrozen,
		d.MemberDeparted: d.AccountClosed, d.MemberPending: d.AccountClosed}
	for s, a := range want {
		if got := d.AccountStateFor(s); got != a {
			t.Errorf("%s -> %s want %s", s, got, a)
		}
	}
}

func TestDisputeStateMachineExhaustive(t *testing.T) {
	permittedD := map[d.DisputeStatus]map[d.DisputeCommand]d.DisputeStatus{
		d.DisputeOpen: {d.DCmdStartReview: d.DisputeUnderReview, d.DCmdWithdraw: d.DisputeWithdrawn,
			d.DCmdResolve: d.DisputeResolved, d.DCmdBlock: d.DisputePostingPending},
		d.DisputeUnderReview:    {d.DCmdWithdraw: d.DisputeWithdrawn, d.DCmdResolve: d.DisputeResolved, d.DCmdBlock: d.DisputePostingPending},
		d.DisputePostingPending: {d.DCmdRetryPost: d.DisputeResolved},
	}
	for _, from := range d.AllDisputeStatuses {
		for _, cmd := range d.AllDisputeCommands {
			want, ok := permittedD[from][cmd]
			got, err := d.NextDisputeStatus(from, cmd)
			if ok && (err != nil || got != want) {
				t.Errorf("%s --%s--> got %s err=%v want %s", from, cmd, got, err, want)
			}
			if !ok && (err == nil || got != from) {
				t.Errorf("%s --%s--> must be refused unchanged, got %s err=%v", from, cmd, got, err)
			}
		}
	}
	// Withdrawal is impossible once a decision exists (DOM-018).
	if _, err := d.NextDisputeStatus(d.DisputePostingPending, d.DCmdWithdraw); err == nil {
		t.Error("withdrawal after decision must be refused")
	}
}

func TestPolicyBounds(t *testing.T) {
	ok := func(mod func(*d.Policy)) bool { p := d.DefaultPolicy(); mod(&p); return p.Validate() == nil }
	if err := d.DefaultPolicy().Validate(); err != nil {
		t.Fatalf("defaults invalid: %v", err)
	}
	cases := []struct {
		name     string
		edgeOK   func(*d.Policy)
		beyondNO func(*d.Policy)
	}{
		{"decimals", func(p *d.Policy) { p.Decimals = 0 }, func(p *d.Policy) { p.Decimals = 5 }},
		{"decimals-low", func(p *d.Policy) { p.Decimals = d.MaxDecimals }, func(p *d.Policy) { p.Decimals = -1 }},
		{"credit-limit", func(p *d.Policy) { p.DefaultCreditLimit = 0 }, func(p *d.Policy) { p.DefaultCreditLimit = -1 }},
		{"credit-limit-max", func(p *d.Policy) { p.DefaultCreditLimit = d.MaxPolicyAmount }, func(p *d.Policy) { p.DefaultCreditLimit = d.MaxPolicyAmount + 1 }},
		{"max-tx", func(p *d.Policy) { p.MaxTxAmount = 1 }, func(p *d.Policy) { p.MaxTxAmount = 0 }},
		{"expiry-min", func(p *d.Policy) { p.ProposalExpiryDays = 1 }, func(p *d.Policy) { p.ProposalExpiryDays = 0 }},
		{"expiry-max", func(p *d.Policy) { p.ProposalExpiryDays = d.MaxProposalExpiryDays }, func(p *d.Policy) { p.ProposalExpiryDays = d.MaxProposalExpiryDays + 1 }},
		{"window-min", func(p *d.Policy) { p.DisputeWindowDays = 1 }, func(p *d.Policy) { p.DisputeWindowDays = 0 }},
		{"window-max", func(p *d.Policy) { p.DisputeWindowDays = d.MaxDisputeWindowDays }, func(p *d.Policy) { p.DisputeWindowDays = d.MaxDisputeWindowDays + 1 }},
		{"join-mode", func(p *d.Policy) { p.JoinMode = d.JoinOpen }, func(p *d.Policy) { p.JoinMode = "anyone" }},
		{"departure", func(p *d.Policy) { p.DepartureRule = d.DepartSettleToZero }, func(p *d.Policy) { p.DepartureRule = "burn" }},
		{"grant", func(p *d.Policy) { p.SignupGrant = 1 }, func(p *d.Policy) { p.SignupGrant = -1 }},
		{"warning", func(p *d.Policy) { p.LimitWarningDays = 0 }, func(p *d.Policy) { p.LimitWarningDays = -1 }},
		{"visibility", func(p *d.Policy) {}, func(p *d.Policy) { p.DirectoryVisibility = "public" }},
		{"currency", func(p *d.Policy) { p.CurrencyName = "x" }, func(p *d.Policy) { p.CurrencyName = "" }},
	}
	for _, c := range cases {
		if !ok(c.edgeOK) {
			t.Errorf("%s: edge value must be accepted", c.name)
		}
		p := d.DefaultPolicy()
		c.beyondNO(&p)
		err := p.Validate()
		var e *d.Error
		if !errors.As(err, &e) || e.Code != d.CodeValidation || len(e.Fields) == 0 {
			t.Errorf("%s: value beyond bound must be refused with named fields, got %v", c.name, err)
		}
	}
}

func TestEffectiveLimit(t *testing.T) {
	now := testNow
	lim := int64(500)
	a := &d.Account{}
	if got := a.EffectiveLimit(100, now); got != 100 {
		t.Errorf("default = %d", got)
	}
	a.LimitOverride = &lim
	if got := a.EffectiveLimit(100, now); got != 500 {
		t.Errorf("override = %d", got)
	}
	exp := now.Add(-1)
	a.LimitOverrideExpires = &exp
	if got := a.EffectiveLimit(100, now); got != 100 {
		t.Errorf("expired override must fall back to default, got %d", got)
	}
	exp = now.Add(1)
	if got := a.EffectiveLimit(100, now); got != 500 {
		t.Errorf("unexpired override = %d", got)
	}
}
