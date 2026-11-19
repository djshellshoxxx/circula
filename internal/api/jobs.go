package api

import (
	"sync"
	"time"
)

// jobStore holds consistency-check results and export artefacts: the service
// has no persistent job tables. Artefacts are spooled to files in dir
// (FR-095), never held in memory. Entries expire after 24 h and the store is
// bounded: running exports count against the same caps as stored ones, and
// every byte an export writes is reserved against capBytes before it
// reaches the disk, so parallel exports cannot exceed the cap (BR-06).
type jobStore struct {
	now    func() time.Time
	mu     sync.Mutex
	checks map[string]checkEntry
	exps   map[string]*exportEntry
	bytes  int64 // sizes of the stored artefacts
	// capBytes bounds stored plus running artefacts: maxExpBytes, lowered
	// only by tests.
	capBytes int64
	closed   bool
	// Running exports (BR-06): how many, per person and in total, and the
	// bytes they have reserved so far.
	running  map[string]int
	nRunning int
	inflight int64
	// erasures counts member erasures per network, so that any
	// export running across one is not stored (BR-04).
	erasures map[string]uint64
	// dir is where export artefacts are spooled (Options.ExportSpoolDir);
	// "" is os.TempDir().
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
	// maxExportsRunning caps the exports in progress at once in the process;
	// each holds a read transaction and a compressor (BR-06).
	maxExportsRunning = 4
)

func newJobStore(now func() time.Time) *jobStore {
	return &jobStore{now: now, checks: map[string]checkEntry{}, exps: map[string]*exportEntry{},
		running: map[string]int{}, erasures: map[string]uint64{}, capBytes: maxExpBytes}
}

// dropLocked removes one export and releases its artefact.
func (j *jobStore) dropLocked(id string, e *exportEntry) {
	e.expired = true
	if e.timer != nil {
		e.timer.Stop()
	}
	// An expiry cannot interrupt a download already admitted. Keep the
	// archive counted against both limits until its final reader leaves.
	if e.readers != 0 {
		return
	}
	j.removeLocked(id, e)
}

func (j *jobStore) removeLocked(id string, e *exportEntry) {
	if j.exps[id] != e {
		return
	}
	if e.timer != nil {
		e.timer.Stop()
	}
	close(e.revoked)
	j.bytes -= e.size()
	delete(j.exps, id)
	e.release()
}

// dropErased removes what the store keeps for an erased member: the export
// artefacts and check results of the person, and every export of
// the member's network, since those may hold the member's data (BR-04).
// Member exports also contain counterparties' names and shared notes.
// Exports of that network still running are not stored when they
// finish.
func (j *jobStore) dropErased(net, person string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if net != "" {
		j.erasures[net]++
	}
	for k, e := range j.exps {
		if e.person == person || (net != "" && e.net == net) {
			// Privacy revocation interrupts readers instead of retaining data
			// for them as normal expiry does.
			j.removeLocked(k, e)
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

// beginExport admits a new export and reserves its slot, or says why it
// cannot run: "person" when the caller holds too many exports, "full" when
// the store is full, "busy" when too many exports are running. adminScope
// marks an export of network-wide data (BR-04). The caller must end the
// ticket with commit or abort.
func (j *jobStore) beginExport(net, person string, adminScope bool) (*exportTicket, string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil, "busy"
	}
	if j.nRunning >= maxExportsRunning {
		return nil, "busy"
	}
	if why := j.slotLocked(person); why != "" {
		return nil, why
	}
	j.running[person]++
	j.nRunning++
	return &exportTicket{j: j, net: net, person: person, admin: adminScope, gen: j.erasures[net]}, ""
}

// slotLocked drops expired exports and says why one more cannot start.
// Running exports count as held.
func (j *jobStore) slotLocked(person string) string {
	now := j.now()
	own := j.running[person]
	for k, x := range j.exps {
		if x.expired || now.Sub(x.at) >= jobTTL {
			j.dropLocked(k, x)
			if j.exps[k] == nil {
				continue
			}
		}
		if x.person == person {
			own++
		}
	}
	switch {
	case own >= maxExportsPerPerson:
		return "person"
	case len(j.exps)+j.nRunning >= maxExports || j.bytes+j.inflight >= j.capBytes:
		return "full"
	}
	return ""
}

// exportTicket is one running export's slot and byte reservation (BR-06).
type exportTicket struct {
	j           *jobStore
	net, person string
	admin       bool
	gen         uint64 // the network's erasure count when the export began
	reserved    int64
	done        bool
}

// reserve claims n more bytes of the shared cap before they are written. It
// fails, claiming nothing, when stored and running artefacts would exceed
// the cap.
func (t *exportTicket) reserve(n int64) bool {
	j := t.j
	j.mu.Lock()
	defer j.mu.Unlock()
	if t.done || j.closed || j.bytes+j.inflight+n > j.capBytes {
		return false
	}
	j.inflight += n
	t.reserved += n
	return true
}

func (t *exportTicket) endLocked() {
	j := t.j
	t.done = true
	j.inflight -= t.reserved
	j.nRunning--
	if j.running[t.person]--; j.running[t.person] <= 0 {
		delete(j.running, t.person)
	}
}

// abort ends the export without storing anything; after commit it does
// nothing.
func (t *exportTicket) abort() {
	t.j.mu.Lock()
	defer t.j.mu.Unlock()
	if !t.done {
		t.endLocked()
	}
}

// commit stores e, whose artefact is the bytes reserved through t. It refuses
// any export when a member of its network was erased while the
// export ran (BR-04). On false the caller still owns (and releases) e.
func (t *exportTicket) commit(e *exportEntry) bool {
	j := t.j
	j.mu.Lock()
	defer j.mu.Unlock()
	if t.done {
		return false
	}
	t.endLocked()
	if j.closed || j.erasures[t.net] != t.gen {
		return false
	}
	e.revoked = make(chan struct{})
	j.exps[e.dto.ID] = e
	j.bytes += e.size()
	e.timer = time.AfterFunc(jobTTL, func() { j.expire(e) })
	return true
}

// getExport returns an export only to the person and network that created it.
func (j *jobStore) getExport(net, person, id string) *exportEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	e := j.exps[id]
	if e == nil || e.net != net || e.person != person {
		return nil
	}
	if e.expired || j.now().Sub(e.at) >= jobTTL {
		j.dropLocked(id, e)
		return nil
	}
	return e
}

func (j *jobStore) expire(e *exportEntry) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.exps[e.dto.ID] == e {
		j.dropLocked(e.dto.ID, e)
	}
}

// acquireExport pins the archive across verification and streaming. The
// caller must releaseExport it on every exit path.
func (j *jobStore) acquireExport(net, person, id string) *exportEntry {
	j.mu.Lock()
	defer j.mu.Unlock()
	e := j.exps[id]
	if e == nil || e.net != net || e.person != person {
		return nil
	}
	if e.expired || j.now().Sub(e.at) >= jobTTL {
		j.dropLocked(id, e)
		return nil
	}
	e.readers++
	return e
}

func (j *jobStore) releaseExport(e *exportEntry) {
	j.mu.Lock()
	defer j.mu.Unlock()
	e.readers--
	if e.readers == 0 && (e.expired || j.now().Sub(e.at) >= jobTTL) {
		j.removeLocked(e.dto.ID, e)
	}
}

func (j *jobStore) close() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.closed = true
	for id, e := range j.exps {
		j.removeLocked(id, e)
	}
}
