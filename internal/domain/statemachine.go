package domain

import "time"

// TxStatus is the transaction status (glossary, DOM section 5).
type TxStatus string

// Transaction statuses. "accepted" is transient inside the atomic accept unit
// and is never persisted (DOM-010 note, OQ-DOM-1).
const (
	TxProposed  TxStatus = "proposed"
	TxAccepted  TxStatus = "accepted"
	TxCompleted TxStatus = "completed"
	TxDeclined  TxStatus = "declined"
	TxCancelled TxStatus = "cancelled"
	TxExpired   TxStatus = "expired"
	TxDisputed  TxStatus = "disputed"
	TxReversed  TxStatus = "reversed"
)

// AllTxStatuses lists every status (used by exhaustive tests).
var AllTxStatuses = []TxStatus{TxProposed, TxAccepted, TxCompleted, TxDeclined,
	TxCancelled, TxExpired, TxDisputed, TxReversed}

// TxCommand is an input to the transaction state machine.
type TxCommand string

// Transaction commands.
const (
	CmdAccept         TxCommand = "accept"
	CmdDecline        TxCommand = "decline"
	CmdCancel         TxCommand = "cancel" // withdraw, departure, admin cancel
	CmdExpire         TxCommand = "expire"
	CmdOpenDispute    TxCommand = "open_dispute"
	CmdCloseDispute   TxCommand = "close_dispute"   // dismissed, withdrawn, partial resolved
	CmdReverseFull    TxCommand = "reverse_full"    // cumulative reversal == amount
	CmdReversePartial TxCommand = "reverse_partial" // cumulative reversal < amount
)

// AllTxCommands lists every command (used by exhaustive tests).
var AllTxCommands = []TxCommand{CmdAccept, CmdDecline, CmdCancel, CmdExpire,
	CmdOpenDispute, CmdCloseDispute, CmdReverseFull, CmdReversePartial}

var txTransitions = map[TxStatus]map[TxCommand]TxStatus{
	TxProposed: {
		CmdAccept:  TxCompleted, // via transient accepted, atomically
		CmdDecline: TxDeclined,
		CmdCancel:  TxCancelled,
		CmdExpire:  TxExpired,
	},
	TxCompleted: {
		CmdOpenDispute:    TxDisputed,
		CmdReverseFull:    TxReversed,
		CmdReversePartial: TxCompleted,
	},
	TxDisputed: {
		CmdCloseDispute:   TxCompleted,
		CmdReverseFull:    TxReversed,
		CmdReversePartial: TxDisputed,
	},
}

// NextTxStatus applies cmd to from. Any pair not in the DOM-010 table is
// refused with CodeInvalidState and must not change state.
func NextTxStatus(from TxStatus, cmd TxCommand) (TxStatus, error) {
	if to, ok := txTransitions[from][cmd]; ok {
		return to, nil
	}
	return from, InvalidState("transaction is %s; %s not allowed", from, cmd)
}

// IsTerminal reports whether no command can leave the status.
func (s TxStatus) IsTerminal() bool { return len(txTransitions[s]) == 0 }

var memberTransitions = map[MemberStatus][]MemberStatus{
	MemberPending:   {MemberActive, MemberDeparted},
	MemberActive:    {MemberSuspended, MemberDeparted},
	MemberSuspended: {MemberActive, MemberDeparted},
}

// AllMemberStatuses lists every membership status.
var AllMemberStatuses = []MemberStatus{MemberPending, MemberActive, MemberSuspended, MemberDeparted}

// CheckMemberTransition validates a DOM-003 transition.
func CheckMemberTransition(from, to MemberStatus) error {
	for _, t := range memberTransitions[from] {
		if t == to {
			return nil
		}
	}
	return InvalidState("membership cannot change from %s to %s", from, to)
}

// AccountState is derived from membership status (DOM-004).
type AccountState string

// Account states.
const (
	AccountActive AccountState = "active"
	AccountFrozen AccountState = "frozen"
	AccountClosed AccountState = "closed"
)

// AccountStateFor maps a membership status to the account state.
func AccountStateFor(s MemberStatus) AccountState {
	switch s {
	case MemberActive:
		return AccountActive
	case MemberSuspended:
		return AccountFrozen
	default: // pending and departed carry no live account
		return AccountClosed
	}
}

// DisputeStatus is the dispute lifecycle state (DOM-018).
type DisputeStatus string

// Dispute statuses.
const (
	DisputeOpen           DisputeStatus = "open"
	DisputeUnderReview    DisputeStatus = "under_review"
	DisputePostingPending DisputeStatus = "posting_pending"
	DisputeResolved       DisputeStatus = "resolved"
	DisputeWithdrawn      DisputeStatus = "withdrawn"
)

// AllDisputeStatuses lists every dispute status.
var AllDisputeStatuses = []DisputeStatus{DisputeOpen, DisputeUnderReview,
	DisputePostingPending, DisputeResolved, DisputeWithdrawn}

// DisputeOutcome is the reviewer decision.
type DisputeOutcome string

// Dispute outcomes.
const (
	OutcomeDismissed       DisputeOutcome = "dismissed"
	OutcomeUpheld          DisputeOutcome = "upheld"
	OutcomePartiallyUpheld DisputeOutcome = "partially_upheld"
)

// DisputeCommand is an input to the dispute state machine.
type DisputeCommand string

// Dispute commands.
const (
	DCmdStartReview DisputeCommand = "start_review"
	DCmdWithdraw    DisputeCommand = "withdraw"
	DCmdResolve     DisputeCommand = "resolve"    // decision posted or dismissed
	DCmdBlock       DisputeCommand = "block"      // upheld but LED-5 blocks posting
	DCmdRetryPost   DisputeCommand = "retry_post" // posting_pending -> resolved
)

// AllDisputeCommands lists every dispute command.
var AllDisputeCommands = []DisputeCommand{DCmdStartReview, DCmdWithdraw, DCmdResolve, DCmdBlock, DCmdRetryPost}

var disputeTransitions = map[DisputeStatus]map[DisputeCommand]DisputeStatus{
	DisputeOpen: {
		DCmdStartReview: DisputeUnderReview,
		DCmdWithdraw:    DisputeWithdrawn,
		DCmdResolve:     DisputeResolved,
		DCmdBlock:       DisputePostingPending,
	},
	DisputeUnderReview: {
		DCmdWithdraw: DisputeWithdrawn,
		DCmdResolve:  DisputeResolved,
		DCmdBlock:    DisputePostingPending,
	},
	DisputePostingPending: {
		DCmdRetryPost: DisputeResolved,
	},
}

// NextDisputeStatus applies cmd to from; withdrawal is only possible before a
// decision (DOM-018).
func NextDisputeStatus(from DisputeStatus, cmd DisputeCommand) (DisputeStatus, error) {
	if to, ok := disputeTransitions[from][cmd]; ok {
		return to, nil
	}
	return from, InvalidState("dispute is %s; %s not allowed", from, cmd)
}

// Dispute contests one completed trade.
type Dispute struct {
	ID             string
	NetworkID      string
	TxID           string
	OpenedBy       string // member id
	Category       string
	Statement      string
	Status         DisputeStatus
	Outcome        DisputeOutcome
	ReversalAmount int64 // amount to reverse for upheld / partially_upheld
	ReviewerID     string
	Rationale      string
	OpenedAt       time.Time
	DecidedAt      *time.Time
}
