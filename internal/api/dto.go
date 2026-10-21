package api

import (
	"encoding/json"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

type networkDTO struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Slug                    string `json:"slug"`
	Status                  string `json:"status"`
	CurrencyName            string `json:"currency_name"`
	Decimals                int    `json:"decimals"`
	JoinMode                string `json:"join_mode"`
	DisputeWindowDays       int    `json:"dispute_window_days"`
	ProposalExpiryDays      int    `json:"proposal_expiry_days"`
	MaxTxAmountMinor        string `json:"max_tx_amount_minor"`
	DefaultCreditLimitMinor string `json:"default_credit_limit_minor"`
	PolicyVersion           int    `json:"policy_version"`
	Suspended               bool   `json:"suspended"`
	// AdminRequired is true while an active network has no active administrator (GOV-020): read-only, no value-moving action.
	AdminRequired bool            `json:"admin_required"`
	Closure       *closureDTO     `json:"closure"` // GOV-025: the scheduled or completed closure, null when none
	Suspensions   []suspensionDTO `json:"suspensions"`
	// ReadOnly is true while the network is frozen after a failed ledger check: reads work, writes return 423.
	ReadOnly    bool       `json:"read_only"`
	FrozenSince *time.Time `json:"frozen_since,omitempty"`
	// FrozenReason is shown to administrators only.
	FrozenReason string `json:"frozen_reason,omitempty"`
}

func toNetwork(n *domain.Network) networkDTO {
	p := n.Policy
	return networkDTO{ID: n.ID, Name: n.Name, Slug: n.Slug, Status: string(n.Status), CurrencyName: p.CurrencyName,
		Decimals: p.Decimals, JoinMode: string(p.JoinMode), DisputeWindowDays: p.DisputeWindowDays,
		ProposalExpiryDays: p.ProposalExpiryDays, MaxTxAmountMinor: minor(p.MaxTxAmount),
		DefaultCreditLimitMinor: minor(p.DefaultCreditLimit), PolicyVersion: n.PolicyVersion,
		Suspended: n.Suspended(), Suspensions: toSuspensions(n), AdminRequired: n.AdminRequired, Closure: toClosure(n.Closure)}
}

type membershipDTO struct {
	NetworkID   string   `json:"network_id"`
	MemberID    string   `json:"member_id"`
	DisplayName string   `json:"display_name"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
}

func roleNames(rs []domain.Role) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, string(r))
	}
	return out
}

func toMembership(m *domain.Member) membershipDTO {
	return membershipDTO{NetworkID: m.NetworkID, MemberID: m.ID, DisplayName: m.DisplayName,
		Status: string(m.Status), Roles: roleNames(m.Roles)}
}

type memberDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

type adminMemberDTO struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	JoinMethod  string   `json:"join_method"`
	DefaultFlag bool     `json:"default_flag"`
	JoinedAt    string   `json:"joined_at"`
	DepartedAt  *string  `json:"departed_at"`
}

func toAdminMember(m *domain.Member) adminMemberDTO {
	return adminMemberDTO{ID: m.ID, DisplayName: m.DisplayName, Status: string(m.Status), Roles: roleNames(m.Roles),
		JoinMethod: m.JoinMethod, DefaultFlag: m.DefaultFlag, JoinedAt: ts(m.JoinedAt), DepartedAt: tsPtr(m.DepartedAt)}
}

func memberETag(m *domain.Member) string {
	return etagOf("member", m.ID, string(m.Status), roleJoin(m.Roles), boolStr(m.DefaultFlag))
}

func roleJoin(rs []domain.Role) string {
	s := ""
	for _, r := range rs {
		s += string(r) + ","
	}
	return s
}

func boolStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

type accountDTO struct {
	MemberID                string `json:"member_id"`
	AccountState            string `json:"account_state"`
	BalanceMinor            string `json:"balance_minor"`
	CreditLimitMinor        string `json:"credit_limit_minor"`
	ReservedMinor           string `json:"reserved_minor"`
	AvailableMinor          string `json:"available_minor"`
	AwaitingFromOthersMinor string `json:"awaiting_from_others_minor"`
	Direction               string `json:"direction"`
	Decimals                int    `json:"decimals"`
	LedgerSeq               int64  `json:"ledger_seq"`
}

// direction implements API-007.
func direction(balance int64) string {
	switch {
	case balance > 0:
		return "in_credit"
	case balance < 0:
		return "in_debt"
	}
	return "zero"
}

func toAccount(b *service.BalanceView) accountDTO {
	return accountDTO{MemberID: b.MemberID, AccountState: string(b.AccountState), BalanceMinor: minor(b.Balance),
		CreditLimitMinor: minor(b.CreditLimit), ReservedMinor: minor(b.ReservedOutgoing),
		AvailableMinor: minor(b.Available), AwaitingFromOthersMinor: minor(b.AwaitingFromOthers),
		Direction: direction(b.Balance), Decimals: b.Decimals, LedgerSeq: b.LedgerSeq}
}

type listingDTO struct {
	ID          string  `json:"id"`
	MemberID    string  `json:"member_id"`
	Kind        string  `json:"kind"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	PriceMinor  *string `json:"price_minor"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toListing(l *domain.Listing) listingDTO {
	return listingDTO{ID: l.ID, MemberID: l.MemberID, Kind: string(l.Type), Title: l.Title, Description: l.Description,
		PriceMinor: minorPtr(l.Price), Status: string(l.Status), CreatedAt: ts(l.CreatedAt), UpdatedAt: ts(l.UpdatedAt)}
}

func listingETag(l *domain.Listing) string {
	p := "-"
	if l.Price != nil {
		p = minor(*l.Price)
	}
	return etagOf("listing", l.ID, ts(l.UpdatedAt), string(l.Status), l.Title, l.Description, p)
}

type txDTO struct {
	ID                 string  `json:"id"`
	Kind               string  `json:"kind"`
	Status             string  `json:"status"`
	PayerMemberID      *string `json:"payer_member_id"`
	PayeeMemberID      *string `json:"payee_member_id"`
	AmountMinor        string  `json:"amount_minor"`
	MyDirection        *string `json:"my_direction"`
	ListingID          *string `json:"listing_id"`
	Description        string  `json:"description"`
	Reason             string  `json:"reason,omitempty"`
	OriginalTxID       *string `json:"original_transaction_id"`
	ReversedTotalMinor string  `json:"reversed_total_minor"`
	ProposedAt         string  `json:"proposed_at"`
	ExpiresAt          *string `json:"expires_at"`
	CompletedAt        *string `json:"completed_at"`
	PostedLedgerSeq    *int64  `json:"posted_ledger_seq"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toTx(t *domain.Transaction, myMember string) txDTO {
	d := txDTO{ID: t.ID, Kind: string(t.Kind), Status: string(t.Status), PayerMemberID: strPtr(t.PayerMemberID),
		PayeeMemberID: strPtr(t.PayeeMemberID), AmountMinor: minor(t.Amount), ListingID: strPtr(t.ListingID),
		Description: t.Note, Reason: t.Reason, OriginalTxID: strPtr(t.OriginalTxID), ReversedTotalMinor: minor(t.ReversedTotal),
		ProposedAt: ts(t.ProposedAt), ExpiresAt: tsPtr(t.ExpiresAt), CompletedAt: tsPtr(t.CompletedAt)}
	if t.LedgerSeq > 0 {
		v := t.LedgerSeq
		d.PostedLedgerSeq = &v
	}
	switch {
	case myMember == "":
	case t.PayerMemberID == myMember:
		s := "pay"
		d.MyDirection = &s
	case t.PayeeMemberID == myMember:
		s := "receive"
		d.MyDirection = &s
	}
	return d
}

type disputeDTO struct {
	ID                  string  `json:"id"`
	TransactionID       string  `json:"transaction_id"`
	OpenedBy            string  `json:"opened_by"`
	Category            string  `json:"category"`
	Statement           string  `json:"statement"`
	Status              string  `json:"status"`
	Outcome             *string `json:"outcome"`
	ReversalAmountMinor string  `json:"reversal_amount_minor"`
	ReviewerID          *string `json:"reviewer_id"`
	Reason              string  `json:"reason"`
	OpenedAt            string  `json:"opened_at"`
	DecidedAt           *string `json:"decided_at"`
}

func toDispute(d *domain.Dispute) disputeDTO {
	return disputeDTO{ID: d.ID, TransactionID: d.TxID, OpenedBy: d.OpenedBy, Category: d.Category, Statement: d.Statement,
		Status: string(d.Status), Outcome: strPtr(string(d.Outcome)), ReversalAmountMinor: minor(d.ReversalAmount),
		ReviewerID: strPtr(d.ReviewerID), Reason: d.Rationale, OpenedAt: ts(d.OpenedAt), DecidedAt: tsPtr(d.DecidedAt)}
}

type auditDTO struct {
	ID            string          `json:"id"`
	Seq           int64           `json:"seq"`
	OccurredAt    string          `json:"occurred_at"`
	ActorType     string          `json:"actor_type"`
	ActorID       string          `json:"actor_id"`
	Action        string          `json:"action"`
	TargetType    string          `json:"target_type"`
	TargetID      string          `json:"target_id"`
	RequestID     string          `json:"request_id"`
	Reason        string          `json:"reason"`
	SummaryBefore json.RawMessage `json:"summary_before"`
	SummaryAfter  json.RawMessage `json:"summary_after"`
	Hash          string          `json:"hash"`
}

func rawJSON(s string) json.RawMessage {
	if s != "" && json.Valid([]byte(s)) {
		return json.RawMessage(s)
	}
	return json.RawMessage("null")
}

func toAudit(e *domain.AuditEvent) auditDTO {
	return auditDTO{ID: e.ID, Seq: e.Seq, OccurredAt: ts(e.OccurredAt), ActorType: string(e.ActorType), ActorID: e.ActorID,
		Action: e.Action, TargetType: e.TargetType, TargetID: e.TargetID, RequestID: e.RequestID, Reason: e.Reason,
		SummaryBefore: rawJSON(e.SummaryBefore), SummaryAfter: rawJSON(e.SummaryAfter), Hash: e.Hash}
}
