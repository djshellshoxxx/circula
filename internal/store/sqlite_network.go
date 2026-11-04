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

// networkSelect reads a network with its effective policy version and, as
// the last column, the policy stored at creation (the lowest version), whose
// scale the network keeps for good (DOM-031).
const networkSelect = `SELECT n.id, n.name, n.slug, n.status, n.ledger_seq, n.created_at,
	p.version, p.policy_json,
	COALESCE((SELECT a.id FROM accounts a WHERE a.network_id = n.id AND a.is_system = 1), ''),
	(SELECT p0.policy_json FROM network_policies p0 WHERE p0.network_id = n.id ORDER BY p0.version LIMIT 1)
	FROM networks n JOIN network_policies p ON p.network_id = n.id
	AND p.version = (SELECT MAX(version) FROM network_policies WHERE network_id = n.id
		AND (? = 0 OR effective_at <= ?))`

// creationScale is the decimal scale of the policy stored at creation, read
// the same way as the effective policy (absent keys take their defaults).
func creationScale(pj string) (int, error) {
	p := domain.DefaultPolicy()
	if err := json.Unmarshal([]byte(pj), &p); err != nil {
		return 0, err
	}
	return p.Decimals, nil
}

func (t *txImpl) scanNetwork(r *sql.Row) (*domain.Network, error) {
	var n domain.Network
	var status, pj, pj0 string
	var created int64
	if err := r.Scan(&n.ID, &n.Name, &n.Slug, &status, &n.LedgerSeq, &created,
		&n.PolicyVersion, &pj, &n.SystemAccountID, &pj0); err != nil {
		return nil, notFound(err, "network")
	}
	n.Status = domain.NetworkStatus(status)
	n.CreatedAt = fromMS(created)
	n.Policy = domain.DefaultPolicy() // fields absent from older versions keep their defaults
	if err := json.Unmarshal([]byte(pj), &n.Policy); err != nil {
		return nil, err
	}
	// DOM-031: the scale is fixed at creation. A later version stored with
	// another scale (an earlier build accepted one with notice) applies with
	// its other keys, but never with its scale: amounts already stored keep
	// the meaning they were written with.
	dec, err := creationScale(pj0)
	if err != nil {
		return nil, err
	}
	n.Policy.Decimals = dec
	sus, err := t.ListSuspensions(n.ID)
	if err != nil {
		return nil, err
	}
	n.Suspensions = sus
	if n.Closure, err = t.GetClosure(n.ID); err != nil {
		return nil, err
	}
	return &n, nil
}

func (t *txImpl) GetNetwork(id string) (*domain.Network, error) {
	return t.scanNetwork(t.row(networkSelect+` WHERE n.id = ?`, t.asOf, t.asOf, id))
}

func (t *txImpl) GetNetworkBySlug(slug string) (*domain.Network, error) {
	return t.scanNetwork(t.row(networkSelect+` WHERE n.slug = ?`, t.asOf, t.asOf, slug))
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

func (t *txImpl) LastConsistencyRun(netID string) (*ConsistencyRun, error) {
	var run ConsistencyRun
	var started, finished int64
	var ok int
	err := t.row(`SELECT id,network_id,started_at,finished_at,ok,findings FROM consistency_runs
		WHERE network_id = ? ORDER BY finished_at DESC, id DESC LIMIT 1`, netID).
		Scan(&run.ID, &run.NetworkID, &started, &finished, &ok, &run.Findings)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	run.Started, run.Finished, run.OK = fromMS(started), fromMS(finished), ok == 1
	return &run, nil
}
