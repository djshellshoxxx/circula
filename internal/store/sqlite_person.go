package store

import (
	"database/sql"
	"strings"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) InsertPerson(p *domain.Person, hash string) error {
	_, err := t.exec(`INSERT INTO persons(id,login,password_hash,created_at) VALUES (?,?,?,?)`,
		p.ID, p.Login, hash, ms(p.CreatedAt))
	return err
}

func (t *txImpl) scanPerson(r *sql.Row) (*domain.Person, string, error) {
	var p domain.Person
	var hash string
	var created int64
	if err := r.Scan(&p.ID, &p.Login, &hash, &created); err != nil {
		return nil, "", notFound(err, "person")
	}
	p.CreatedAt = fromMS(created)
	return &p, hash, nil
}

func (t *txImpl) GetPersonByLogin(login string) (*domain.Person, string, error) {
	return t.scanPerson(t.row(`SELECT id,login,password_hash,created_at FROM persons WHERE login = ?`, login))
}

func (t *txImpl) GetPerson(id string) (*domain.Person, string, error) {
	return t.scanPerson(t.row(`SELECT id,login,password_hash,created_at FROM persons WHERE id = ?`, id))
}

func (t *txImpl) SetPasswordHash(id, hash string) error {
	_, err := t.exec(`UPDATE persons SET password_hash = ? WHERE id = ?`, hash, id)
	return err
}

func (t *txImpl) GetThrottle(login string) (int, time.Time, bool, error) {
	var f int
	var until int64
	err := t.row(`SELECT failures, locked_until FROM login_throttle WHERE login = ?`, login).Scan(&f, &until)
	if err == sql.ErrNoRows {
		return 0, time.Time{}, false, nil
	}
	if err != nil {
		return 0, time.Time{}, false, err
	}
	return f, fromMS(until), true, nil
}

func (t *txImpl) PutThrottle(login string, failures int, until time.Time) error {
	_, err := t.exec(`INSERT INTO login_throttle(login,failures,locked_until) VALUES (?,?,?)
		ON CONFLICT(login) DO UPDATE SET failures = excluded.failures, locked_until = excluded.locked_until`,
		login, failures, ms(until))
	return err
}

func (t *txImpl) DeleteThrottle(login string) error {
	_, err := t.exec(`DELETE FROM login_throttle WHERE login = ?`, login)
	return err
}

// ---- sessions ----

func (t *txImpl) InsertSession(s *domain.Session) error {
	_, err := t.exec(`INSERT INTO sessions(id,person_id,created_at,last_seen_at,expires_at) VALUES (?,?,?,?,?)`,
		s.IDHash, s.PersonID, ms(s.CreatedAt), ms(s.LastSeenAt), ms(s.ExpiresAt))
	return err
}

func (t *txImpl) GetSession(idHash string) (*domain.Session, error) {
	var s domain.Session
	var created, seen, exp int64
	var rev sql.NullInt64
	err := t.row(`SELECT id,person_id,created_at,last_seen_at,expires_at,revoked_at FROM sessions WHERE id = ?`,
		idHash).Scan(&s.IDHash, &s.PersonID, &created, &seen, &exp, &rev)
	if err != nil {
		return nil, notFound(err, "session")
	}
	s.CreatedAt, s.LastSeenAt, s.ExpiresAt, s.RevokedAt = fromMS(created), fromMS(seen), fromMS(exp), ptrTime(rev)
	return &s, nil
}

func (t *txImpl) TouchSession(idHash string, seen time.Time) error {
	_, err := t.exec(`UPDATE sessions SET last_seen_at = ? WHERE id = ?`, ms(seen), idHash)
	return err
}

func (t *txImpl) RevokeSession(idHash string, at time.Time) error {
	_, err := t.exec(`UPDATE sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, ms(at), idHash)
	return err
}

func (t *txImpl) RevokeSessionsOf(personID string, at time.Time) error {
	_, err := t.exec(`UPDATE sessions SET revoked_at = ? WHERE person_id = ? AND revoked_at IS NULL`, ms(at), personID)
	return err
}

// ---- API tokens ----

func joinScopes(s []domain.Scope) string {
	out := make([]string, len(s))
	for i, x := range s {
		out[i] = string(x)
	}
	return strings.Join(out, ",")
}

func splitScopes(s string) []domain.Scope {
	var out []domain.Scope
	for _, p := range strings.Split(s, ",") {
		if p != "" {
			out = append(out, domain.Scope(p))
		}
	}
	return out
}

func (t *txImpl) InsertToken(a *domain.APIToken) error {
	_, err := t.exec(`INSERT INTO api_tokens(id,person_id,network_id,name,token_hash,scopes,created_at,expires_at)
		VALUES (?,?,?,?,?,?,?,?)`, a.ID, a.PersonID, nullStr(a.NetworkID), a.Name, a.TokenHash,
		joinScopes(a.Scopes), ms(a.CreatedAt), msPtr(a.ExpiresAt))
	return err
}

const tokenCols = //nolint:gosec // column list, not a credential
`id,person_id,COALESCE(network_id,''),name,token_hash,scopes,created_at,last_used_at,expires_at,revoked_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanToken(r rowScanner) (*domain.APIToken, error) {
	var a domain.APIToken
	var scopes string
	var created int64
	var used, exp, rev sql.NullInt64
	if err := r.Scan(&a.ID, &a.PersonID, &a.NetworkID, &a.Name, &a.TokenHash, &scopes,
		&created, &used, &exp, &rev); err != nil {
		return nil, err
	}
	a.Scopes, a.CreatedAt = splitScopes(scopes), fromMS(created)
	a.LastUsedAt, a.ExpiresAt, a.RevokedAt = ptrTime(used), ptrTime(exp), ptrTime(rev)
	return &a, nil
}

func (t *txImpl) GetTokenByHash(hash string) (*domain.APIToken, error) {
	a, err := scanToken(t.row(`SELECT `+tokenCols+` FROM api_tokens WHERE token_hash = ?`, hash))
	return a, notFound(err, "token")
}

func (t *txImpl) TouchToken(id string, at time.Time) error {
	_, err := t.exec(`UPDATE api_tokens SET last_used_at = ? WHERE id = ?`, ms(at), id)
	return err
}

func (t *txImpl) RevokeToken(personID, id string, at time.Time) (bool, error) {
	res, err := t.exec(`UPDATE api_tokens SET revoked_at = ? WHERE id = ? AND person_id = ? AND revoked_at IS NULL`,
		ms(at), id, personID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (t *txImpl) ListTokens(personID string) ([]*domain.APIToken, error) {
	rows, err := t.query(`SELECT `+tokenCols+` FROM api_tokens WHERE person_id = ? ORDER BY id`, personID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []*domain.APIToken
	for rows.Next() {
		a, err := scanToken(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
