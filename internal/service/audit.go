package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// auditEntry describes one state change for the audit log. Before/After are
// minimal non-personal field maps (DAT 7.3); never put free text, secrets or
// contact details in them.
type auditEntry struct {
	Type       domain.ActorType
	Action     string
	TargetType string
	TargetID   string
	Reason     string
	IdemRef    string
	Before     map[string]any
	After      map[string]any
	Subjects   []string
}

// audit writes the event inside the caller's unit of work (LED-6). Events are
// hash-chained per network (DAT-030).
func (s *Service) audit(tx store.Tx, p *principal, e auditEntry) error {
	actorID := p.memberID()
	if e.Type == "" {
		e.Type = p.actorType(domain.ActorMember)
	}
	return s.emit(tx, p.net.ID, actorID, p.actor.RequestID, e)
}

// auditPlatform writes a platform-scope event (no network) attributed to a person.
func (s *Service) auditPlatform(tx store.Tx, actor domain.Actor, e auditEntry) error {
	if e.Type == "" {
		e.Type = domain.ActorPerson
	}
	return s.emit(tx, "", actor.PersonID, actor.RequestID, e)
}

func (s *Service) emit(tx store.Tx, netID, actorID, requestID string, e auditEntry) error {
	seq, prev, err := tx.LastAudit(netID)
	if err != nil {
		return err
	}
	ev := &domain.AuditEvent{
		ID: s.id(), NetworkID: netID, Seq: seq + 1, OccurredAt: s.now(),
		ActorType: e.Type, ActorID: actorID, Action: e.Action,
		TargetType: e.TargetType, TargetID: e.TargetID, RequestID: requestID,
		IdemRef: e.IdemRef, SummaryBefore: summarize(e.Before), SummaryAfter: summarize(e.After),
		Reason: e.Reason, SubjectIDs: dedupe(e.Subjects), PrevHash: prev,
	}
	ev.Hash = AuditHash(ev)
	return tx.InsertAudit(ev)
}

// summarize renders a map as canonical JSON (encoding/json sorts map keys).
func summarize(m map[string]any) string {
	if len(m) == 0 {
		return ""
	}
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	return string(b)
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range in {
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// AuditHash computes the chain hash of an event over its content and the
// previous hash. It detects silent edits; it is not tamper-proofing (DAT-030).
func AuditHash(e *domain.AuditEvent) string {
	subj := append([]string(nil), e.SubjectIDs...)
	sort.Strings(subj)
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s\x1f%s\x1f%s\x1f%d\x1f%d\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s\x1f%s",
		e.PrevHash, e.ID, e.NetworkID, e.Seq, e.OccurredAt.UnixMilli(), e.ActorType, e.ActorID, e.Action,
		e.TargetType, e.TargetID, e.RequestID, e.IdemRef, e.SummaryBefore, e.SummaryAfter, e.Reason,
		strings.Join(subj, ","))
	return hex.EncodeToString(h.Sum(nil))
}
