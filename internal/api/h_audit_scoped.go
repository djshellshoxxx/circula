package api

// listMyAudit serves GET /audit-events: the role-scoped audit read for members
// (own-account events, reduced) and dispute reviewers (dispute events). The
// service decides the scope; administrators get the full log. Actor and target
// filters are admin-only because they could probe other members' events.
func (s *server) listMyAudit(c *reqCtx) error {
	if err := c.only("limit", "cursor", "action", "from", "to"); err != nil {
		return err
	}
	return s.auditList(c)
}
