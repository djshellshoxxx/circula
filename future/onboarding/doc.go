// Package onboarding is an isolated, default-off implementation of Circula's
// guided onboarding checklist and synthetic practice network (FUX-106).
//
// The package has four independent parts:
//
//   - Progress and Tracker: a versioned, idempotent, per-network/member
//     checklist with skip/resume, persisted through the Store interface
//     (SQLite adapter in sqlstore.go, in-memory test double in memstore.go).
//   - Copy: bounded plain-text administrator copy and validated links.
//   - Handler: a server-rendered, no-JavaScript HTTP adapter that the caller
//     must wrap with its own current-authentication guard.
//   - PracticeServer: a separate, fully synthetic in-memory practice world with
//     its own cookie and origin checks. It accepts no database handle, no
//     production session and no service object.
//
// Nothing here registers itself, opens connections, starts goroutines on
// import, or writes to a ledger. Facts about authorization, active membership
// and terms are trusted adapter snapshots (Actor, Facts), never client input.
// Tutorial progress is non-authoritative: it must never be used as evidence
// that terms were accepted or that a real trade happened.
package onboarding
