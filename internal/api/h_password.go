package api

import "github.com/djshellshoxxx/circula/internal/domain"

// changePassword changes the caller's password (AUTH-03, SEC-003). It needs a
// browser session (route flag cookieOnly): a stolen API token must not be able
// to take the account over. Every session of the person is revoked, including
// the caller's; API tokens stay valid until revoked.
func (s *server) changePassword(c *reqCtx) error {
	var in struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	if ok, retry := s.pwFail.allow("pw:"+c.actor.PersonID, 5); !ok {
		return rateLimited(retry)
	}
	err := s.svc.ChangePassword(c.r.Context(), c.actor, in.CurrentPassword, in.NewPassword)
	switch domain.CodeOf(err) {
	case "":
		c.w.WriteHeader(204)
		return nil
	case domain.CodeUnauthenticated:
		// A wrong current password is a field error, not a lost session.
		return validation("current_password", "incorrect", "the current password is not correct")
	}
	if de, isDomain := err.(*domain.Error); isDomain && de.Code == domain.CodeValidation {
		for i := range de.Fields {
			if de.Fields[i].Field == "password" {
				de.Fields[i].Field = "new_password"
			}
		}
	}
	return err
}
