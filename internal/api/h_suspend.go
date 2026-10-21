package api

import "github.com/djshellshoxxx/circula/internal/domain"

type suspensionDTO struct {
	Kind     string `json:"kind"`
	Category string `json:"category,omitempty"`
	Reason   string `json:"reason"`
	At       string `json:"at"`
}

func toSuspensions(n *domain.Network) []suspensionDTO {
	out := []suspensionDTO{}
	for _, s := range n.Suspensions {
		out = append(out, suspensionDTO{Kind: string(s.Kind), Category: s.Category, Reason: s.Reason, At: ts(s.At)})
	}
	return out
}

func (s *server) reasonOnly(c *reqCtx) (string, error) {
	var in struct {
		Reason string `json:"reason"`
	}
	if err := c.decode(&in); err != nil {
		return "", err
	}
	return in.Reason, nil
}

func (s *server) suspendNetwork(c *reqCtx) error { // GOV-023
	reason, err := s.reasonOnly(c)
	if err != nil {
		return err
	}
	if err := s.svc.SuspendNetwork(c.r.Context(), c.actor, c.netID, reason); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}

func (s *server) resumeNetwork(c *reqCtx) error { // GOV-023
	reason, err := s.reasonOnly(c)
	if err != nil {
		return err
	}
	if err := s.svc.ResumeNetwork(c.r.Context(), c.actor, c.netID, reason); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}

func (s *server) closureStatus(c *reqCtx) error { // FR-068
	pv, err := s.svc.PreviewClosure(c.r.Context(), c.actor, c.netID)
	if err != nil {
		return err
	}
	bl := pv.Blockers
	if bl == nil {
		bl = []string{}
	}
	return ok(c, 200, map[string]any{"blockers": bl, "non_zero_balances": pv.NonZeroBalances, "members": pv.Members})
}

func (s *server) closeNetwork(c *reqCtx) error { // FR-068, GOV-025
	reason, err := s.reasonOnly(c)
	if err != nil {
		return err
	}
	if err := s.svc.CloseNetwork(c.r.Context(), c.actor, c.netID, reason); err != nil {
		return err
	}
	c.w.WriteHeader(204)
	return nil
}
