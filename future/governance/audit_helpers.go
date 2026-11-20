package governance

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"time"
)

func writeOne(ctx context.Context, tx *sql.Tx, q string, args ...any) (sql.Result, error) {
	result, err := tx.ExecContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n != 1 {
		return nil, errors.New("governance write was not persisted")
	}
	return result, nil
}

var schemaName = regexp.MustCompile(`(?i)^CREATE (TABLE|INDEX) IF NOT EXISTS ([a-z_]+)`)

func canonicalSQL(q string) string {
	var out strings.Builder
	quoted := false
	for i := 0; i < len(q); i++ {
		c := q[i]
		if c == '\'' {
			out.WriteByte(c)
			if quoted && i+1 < len(q) && q[i+1] == '\'' {
				i++
				out.WriteByte(q[i])
				continue
			}
			quoted = !quoted
			continue
		}
		if !quoted && c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		out.WriteByte(c)
	}
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.TrimSuffix(out.String(), ";"), "if not exists ", "")), " ")
}
func validateSchema(ctx context.Context, tx *sql.Tx) error {
	for _, expected := range schemaStatements {
		parts := schemaName.FindStringSubmatch(expected)
		if len(parts) != 3 {
			return errors.New("invalid module schema")
		}
		var kind, actual string
		if err := tx.QueryRowContext(ctx, `SELECT type,sql FROM sqlite_schema WHERE name=?`, parts[2]).Scan(&kind, &actual); err != nil {
			return err
		}
		if kind != strings.ToLower(parts[1]) || canonicalSQL(actual) != canonicalSQL(expected) {
			return errors.New("incompatible governance schema")
		}
	}
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM future_gov_schema`).Scan(&count); err != nil {
		return err
	}
	if count > 1 {
		return errors.New("ambiguous governance schema version")
	}
	return nil
}
func validProposal(p Proposal) bool {
	if !ident(p.ID) || !ident(p.Network) || !ident(p.Author) || !text(p.Title, 200, true) || !text(p.Body, 12000, false) || !text(p.Category, 64, true) || p.Revision < 1 || p.Eligible < 0 || p.Eligible > 10000 {
		return false
	}
	switch p.State {
	case "draft":
		return p.Eligible == 0 && p.Opens.IsZero() && p.Closes.IsZero()
	case "open", "closed":
		return p.Eligible > 0 && p.Opens.Before(p.Closes) && p.Closes.Sub(p.Opens) <= 90*24*time.Hour
	case "withdrawn":
		return text(p.Withdrawal, 1000, false) && (p.Eligible == 0 || p.Opens.Before(p.Closes))
	default:
		return false
	}
}
func validResult(r Result) bool {
	o := r.Outcome
	if !o.Advisory || o.Eligible < 1 || o.Eligible > 10000 || o.Yes < 0 || o.No < 0 || o.Abstain < 0 || o.Yes > o.Eligible || o.No > o.Eligible || o.Abstain > o.Eligible || o.Yes+o.No+o.Abstain > o.Eligible || r.ClosedAt.IsZero() {
		return false
	}
	if r.Certified != nil && (!ident(r.Certified.Member) || r.Certified.At.Before(r.ClosedAt)) {
		return false
	}
	return r.PolicyVersion == "" || (r.Certified != nil && ident(r.PolicyVersion))
}
func safetyAction(action string) bool {
	switch action {
	case "hide", "withdraw", "resolve", "close", "certify", "link":
		return true
	}
	return false
}
func authorizeReplay(ctx context.Context, tx *sql.Tx, id Identity, principal Principal, c Command, p Proposal) error {
	switch c.Action {
	case "create", "edit", "open":
		if p.Author != id.Member {
			return ErrDenied
		}
	case "withdraw", "close":
		if p.Author != id.Member && !principal.Admin {
			return ErrDenied
		}
	case "certify", "link", "resolve":
		if !principal.Admin {
			return ErrDenied
		}
	case "hide":
		if !principal.Admin {
			comments, err := loadComments(ctx, tx, id.Network, p.ID)
			if err != nil {
				return err
			}
			found := false
			for _, row := range comments {
				if row.ID == c.Comment && row.Author == id.Member {
					found = true
				}
			}
			if !found {
				return ErrDenied
			}
		}
	case "vote":
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM future_gov_ballots WHERE network=? AND proposal=? AND member=?)`, id.Network, p.ID, id.Member).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrDenied
		}
	case "comment", "dispute":
	default:
		return ErrDenied
	}
	return nil
}
