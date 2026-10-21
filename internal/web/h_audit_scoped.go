package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// myAuditPage shows the role-scoped audit view: a member sees events about their
// own account (actor names reduced), a reviewer sees dispute events (FR-082,
// DAT-033, DAT-035). The service enforces the scope.
func (s *server) myAuditPage(w http.ResponseWriter, r *http.Request, c *rc) {
	q := r.URL.Query()
	after, _ := strconv.ParseInt(q.Get("after"), 10, 64)
	evs, err := s.svc.ListAudit(r.Context(), c.actor, c.netID(), service.AuditQuery{Action: strings.TrimSpace(q.Get("action")), AfterSeq: after, Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	var rows []auditRow
	for _, e := range evs {
		who := "Network staff"
		if nm, ok := n.lookup(e.ActorID); ok {
			who = nm
		} else if e.ActorType == domain.ActorSystem {
			who = "System"
		}
		rows = append(rows, auditRow{Seq: e.Seq, When: c.when(e.OccurredAt), Actor: who, Action: e.Action, Target: e.TargetType, Reason: e.Reason})
	}
	next := ""
	if len(evs) == 50 {
		next = strconv.FormatInt(evs[49].Seq, 10)
	}
	s.render(w, r, c, "my_audit", 200, &page{Title: "Account activity log", D: map[string]any{"Rows": rows, "Next": next},
		V: map[string]string{"action": q.Get("action")}})
}
