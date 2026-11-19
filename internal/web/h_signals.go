package web

import (
	"html/template"
	"net/http"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/web/messages"
)

// SEC-026: administrators and dispute reviewers read the network's activity
// signals. The service decides who may read them and which signals a reader
// sees; this page only formats them. Nothing here changes a trade or a member.
func init() {
	extraRoutes = append(extraRoutes, func(s *server) {
		s.mux.HandleFunc("GET /n/{slug}/review/activity-signals", s.netH(s.activitySignals))
	})
}

type signalRow struct {
	Kind, Text string
	Raised     template.HTML
}

type signalsData struct {
	Rows     []signalRow
	Filter   string
	Next     string
	Settings string
	Filters  []option
}

func (s *server) activitySignals(w http.ResponseWriter, r *http.Request, c *rc) {
	kind := r.URL.Query().Get("kind")
	if kind != domain.SignalVelocity && kind != domain.SignalReciprocal {
		kind = ""
	}
	const per = 25
	vs, err := s.svc.ListActivitySignals(r.Context(), c.actor, c.netID(), service.ActivitySignalQuery{Kind: kind,
		Page: service.Page{AfterID: r.URL.Query().Get("after"), Limit: per + 1}})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d := &signalsData{Filter: kind, Filters: []option{{"", messages.Text("signals.filter_all")},
		{domain.SignalVelocity, messages.Text("signals.kind_velocity")}, {domain.SignalReciprocal, messages.Text("signals.kind_reciprocal")}}}
	if len(vs) > per {
		vs = vs[:per]
		d.Next = vs[per-1].ID
	}
	for _, v := range vs {
		row := signalRow{Raised: c.when(v.RaisedAt)}
		switch v.Kind {
		case domain.SignalReciprocal:
			row.Kind = messages.Text("signals.kind_reciprocal")
			row.Text = messages.Text("signals.reciprocal", v.Observed, v.MemberName, v.OtherMemberName, int(v.Window/(24*time.Hour)), v.Threshold)
		default:
			row.Kind = messages.Text("signals.kind_velocity")
			row.Text = messages.Text("signals.velocity", v.Observed, v.MemberName, int(v.Window/time.Minute), v.Threshold)
		}
		d.Rows = append(d.Rows, row)
	}
	if pol := c.Net.Policy; pol.ActivitySignalsEnabled {
		d.Settings = messages.Text("signals.settings", pol.VelocitySignalProposals, pol.VelocitySignalWindowMinutes,
			pol.ReciprocalSignalTrades, pol.ReciprocalSignalWindowDays)
	} else {
		d.Settings = messages.Text("signals.settings_off")
	}
	s.render(w, r, c, "review_signals", 200, &page{Title: messages.Text("signals.title"), D: d})
}
