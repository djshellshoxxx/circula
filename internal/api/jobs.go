package api

import (
	"sync"
	"time"
)

// jobStore holds consistency-check results and export artefacts: the service
// has no persistent job tables. Artefacts are spooled to temporary files when
// possible (FR-095) and otherwise kept in memory. Entries expire after 24 h
// and the store is size-bounded, so memory and disk use are bounded.
type jobStore struct {
	now    func() time.Time
	mu     sync.Mutex
	checks map[string]checkEntry
	exps   map[string]*exportEntry
	bytes  int64
	// inflight counts bytes of exports still being written (BR-06). They
	// share limit with the stored artefacts, so parallel exports cannot
	// exceed it before their size is known.
	inflight int64
	limit    int64 // byte cap for stored plus in-flight artefacts
	// dir is where export artefacts are spooled; "" is os.TempDir().
	dir string
}

type checkEntry struct {
	net, person string
	at          time.Time
	d           checkDTO
}

const (
	jobTTL      = 24 * time.Hour
	maxChecks   = 200
	maxExports  = 50
	maxExpBytes = 64 << 20
	// maxExportsPerPerson caps the live artefacts one person can hold, so a
	// single member cannot fill the shared store for everyone (SA-11).
	maxExportsPerPerson = 5
)

func newJobStore(now func() time.Time) *jobStore {
	return &jobStore{now: now, checks: map[string]checkEntry{}, exps: map[string]*exportEntry{}, limit: maxExpBytes}
}

// dropLocked removes one export and releases its artefact.
func (j *jobStore) dropLocked(id string, e *exportEntry) {
	j.bytes -= e.size()
	delete(j.exps, id)
	e.release()
}

// dropPerson removes the export artefacts and check results of one person.
func (j *jobStore) dropPerson(person string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for k, e := range j.exps {
		if e.person == person {
			j.dropLocked(k, e)
		}
	}
	for k, e := range j.checks {
		if e.person == person {
			delete(j.checks, k)
		}
	}
}

func (j *jobStore) putCheck(net, person string, d checkDTO) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := j.now()
	for k, e := range j.checks {
		if now.Sub(e.at) > jobTTL {
			delete(j.checks, k)
		}
	}
	for len(j.checks) >= maxChecks {
		var oldest string
		var ot time.Time
		for k, e := range j.checks {
			if oldest == "" || e.at.Before(ot) {
				oldest, ot = k, e.at
			}
		}
		delete(j.checks, oldest)
	}
	j.checks[d.ID] = checkEntry{net: net, person: person, at: now, d: d}
}

func (j *jobStore) getCheck(net, id string) (checkDTO, bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	e, ok := j.checks[id]
	if !ok || e.net != net || j.now().Sub(e.at) > jobTTL {
		return checkDTO{}, false
	}
	return e.d, true
}

// exportSlot says why a new export cannot be stored: "" when it can,
// "person" when the caller holds too many, "full" when the store is full.
func (j *jobStore) exportSlot(person string) string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.slotLocked(person, 0)
}

func (j *jobStore) slotLocked(person string, size int64) string {
	now := j.now()
	own := 0
	for k, x := range j.exps {
		if now.Sub(x.at) > jobTTL {
			j.dropLocked(k, x)
			continue
		}
		if x.person == person {
			own++
		}
	}
	switch {
	case own >= maxExportsPerPerson:
		return "person"
	case len(j.exps) >= maxExports || j.bytes+j.inflight+size > j.limit:
		return "full"
	}
	return ""
}

// putExport stores e; on false the caller still owns (and releases) it.
func (j *jobStore) putExport(e *exportEntry) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.inflight -= e.charged // the reservation ends here, stored or not
	e.charged = 0
	if j.slotLocked(e.person, e.size()) != "" {
		return false
	}
	j.exps[e.dto.ID] = e
	j.bytes += e.size()
	return true
}

// getExport returns an export only to the person and network that created it.
func (j *jobStore) getExport(net, person, id string) *exportEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	e := j.exps[id]
	if e == nil || e.net != net || e.person != person || j.now().Sub(e.at) > jobTTL {
		return nil
	}
	return e
}

// charge reserves n more bytes for an export being written; false when the
// stored and in-flight artefacts would exceed the cap (BR-06).
func (j *jobStore) charge(n int64) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.bytes+j.inflight+n > j.limit {
		return false
	}
	j.inflight += n
	return true
}

// uncharge returns a reservation that will not be stored.
func (j *jobStore) uncharge(n int64) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.inflight -= n
}

// networkWideScopes are the export scopes that hold other members' data.
var networkWideScopes = map[string]bool{"ledger": true, "members": true, "audit": true, "full": true}

// dropNetworkWide removes every network-wide export artefact of net, whoever
// made it, so an erased member's data does not outlive the erasure in an
// administrator's export (BR-04).
func (j *jobStore) dropNetworkWide(net string) {
	if net == "" {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	for k, e := range j.exps {
		if e.net == net && networkWideScopes[e.dto.Scope] {
			j.dropLocked(k, e)
		}
	}
}
