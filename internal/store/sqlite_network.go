package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func (t *txImpl) InsertNetwork(n *domain.Network) error {
	_, err := t.exec(`INSERT INTO networks(id,name,slug,status,ledger_seq,created_at) VALUES (?,?,?,?,0,?)`,
		n.ID, n.Name, n.Slug, string(n.Status), ms(n.CreatedAt))
	return err
}

func (t *txImpl) AppendPolicy(netID string, version int, p domain.Policy, by, reason string, at time.Time) error {
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = t.exec(`INSERT INTO network_policies(network_id,version,policy_json,changed_by,reason,effective_at)
		VALUES (?,?,?,?,?,?)`, netID, version, string(b), by, reason, ms(at))
	return err
}

const networkSelect = `SELECT n.id, n.name, n.slug, n.status, n.ledger_seq, n.created_at,
	p.version, p.policy_json,
	COALESCE((SELECT a.id FROM accounts a WHERE a.network_id = n.id AND a.is_system = 1), '')
	FROM networks n JOIN network_policies p ON p.network_id = n.id
	AND p.version = (SELECT MAX(version) FROM network_policies WHERE network_id = n.id)`

func (t *txImpl) scanNetwork(r *sql.Row) (*domain.Network, error) {
	var n domain.Network
	var status, pj string
	var created int64
	if err := r.Scan(&n.ID, &n.Name, &n.Slug, &status, &n.LedgerSeq, &created,
		&n.PolicyVersion, &pj, &n.SystemAccountID); err != nil {
		return nil, notFound(err, "network")
	}
	n.Status = domain.NetworkStatus(status)
	n.CreatedAt = fromMS(created)
	n.Policy = domain.DefaultPolicy() // fields absent from older versions keep their defaults
	if err := json.Unmarshal([]byte(pj), &n.Policy); err != nil {
		return nil, err
	}
	return &n, nil
}

func (t *txImpl) GetNetwork(id string) (*domain.Network, error) {
	return t.scanNetwork(t.row(networkSelect+` WHERE n.id = ?`, id))
}

func (t *txImpl) GetNetworkBySlug(slug string) (*domain.Network, error) {
	return t.scanNetwork(t.row(networkSelect+` WHERE n.slug = ?`, slug))
}

func (t *txImpl) SetNetworkStatus(id string, s domain.NetworkStatus) error {
	_, err := t.exec(`UPDATE networks SET status = ? WHERE id = ?`, string(s), id)
	return err
}

func (t *txImpl) NextLedgerSeq(netID string) (int64, error) {
	var seq int64
	err := t.row(`UPDATE networks SET ledger_seq = ledger_seq + 1 WHERE id = ? RETURNING ledger_seq`, netID).Scan(&seq)
	return seq, notFound(err, "network")
}

func (t *txImpl) LedgerSeq(netID string) (int64, error) {
	var seq int64
	err := t.row(`SELECT ledger_seq FROM networks WHERE id = ?`, netID).Scan(&seq)
	return seq, notFound(err, "network")
}

func (t *txImpl) InsertConsistencyRun(r ConsistencyRun) error {
	_, err := t.exec(`INSERT INTO consistency_runs(id,network_id,started_at,finished_at,ok,findings) VALUES (?,?,?,?,?,?)`,
		r.ID, r.NetworkID, ms(r.Started), ms(r.Finished), b2i(r.OK), r.Findings)
	return err
}
