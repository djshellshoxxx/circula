package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) SetAsOf(at time.Time) { t.asOf = ms(at) }

func (t *txImpl) LatestPolicyVersion(netID string) (int, error) {
	var v int
	err := t.row(`SELECT COALESCE(MAX(version),0) FROM network_policies WHERE network_id = ?`, netID).Scan(&v)
	return v, mapErr(err)
}

func (t *txImpl) AppendPolicyNotice(netID string, version int, p domain.Policy, by, reason string, announced, effective time.Time, noticeDays int, noticeReason string) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = t.exec(`INSERT INTO network_policies(network_id,version,policy_json,changed_by,reason,effective_at,announced_at,notice_days,notice_reason)
		VALUES (?,?,?,?,?,?,?,?,?)`, netID, version, string(b), by, reason, ms(effective), ms(announced), noticeDays, noticeReason)
	return err
}

func (t *txImpl) ListPolicyVersions(netID string) ([]PolicyVersionRow, error) {
	rows, err := t.query(`SELECT version, policy_json, changed_by, reason, effective_at, announced_at, notice_days, notice_reason
		FROM network_policies WHERE network_id = ? ORDER BY version`, netID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []PolicyVersionRow
	for rows.Next() {
		var r PolicyVersionRow
		var pj string
		var eff, ann int64
		if err := rows.Scan(&r.Version, &pj, &r.ChangedBy, &r.Reason, &eff, &ann, &r.NoticeDays, &r.NoticeReason); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(pj), &r.Policy); err != nil {
			return nil, err
		}
		r.EffectiveAt = fromMS(eff)
		r.AnnouncedAt = r.EffectiveAt
		if ann != 0 {
			r.AnnouncedAt = fromMS(ann)
		}
		out = append(out, r)
	}
	return out, mapErr(rows.Err())
}

func (t *txImpl) InsertTerms(netID string, tm domain.Terms) error {
	_, err := t.exec(`INSERT INTO network_terms(network_id,version,body,published_by,published_at) VALUES (?,?,?,?,?)`,
		netID, tm.Version, tm.Body, tm.PublishedBy, ms(tm.PublishedAt))
	return err
}

func (t *txImpl) termsWhere(where string, args ...any) (*domain.Terms, error) {
	var tm domain.Terms
	var at int64
	err := t.row(`SELECT version, body, published_by, published_at FROM network_terms WHERE `+where, args...).
		Scan(&tm.Version, &tm.Body, &tm.PublishedBy, &at)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	tm.PublishedAt = fromMS(at)
	return &tm, nil
}

func (t *txImpl) LatestTerms(netID string) (*domain.Terms, error) {
	return t.termsWhere(`network_id = ? ORDER BY version DESC LIMIT 1`, netID)
}

func (t *txImpl) GetTerms(netID string, version int) (*domain.Terms, error) {
	return t.termsWhere(`network_id = ? AND version = ?`, netID, version)
}

func (t *txImpl) RecordTermsAcceptance(netID, memberID string, version int, at time.Time) error {
	_, err := t.exec(`INSERT OR IGNORE INTO terms_acceptances(network_id,member_id,version,accepted_at) VALUES (?,?,?,?)`,
		netID, memberID, version, ms(at))
	return err
}

func (t *txImpl) AcceptedTermsVersion(netID, memberID string) (int, error) {
	var v int
	err := t.row(`SELECT COALESCE(MAX(version),0) FROM terms_acceptances WHERE network_id = ? AND member_id = ?`, netID, memberID).Scan(&v)
	return v, mapErr(err)
}
