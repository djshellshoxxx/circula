package api

import (
	"sync"
	"time"
)

// jobStore holds consistency-check results and export artefacts in memory:
// the service has no persistent job tables. Entries expire after 24 h and the
// store is size-bounded, so memory use is bounded.
type jobStore struct {
	now    func() time.Time
	mu     sync.Mutex
	checks map[string]checkEntry
	exps   map[string]*exportEntry
	bytes  int64
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
	return &jobStore{now: now, checks: map[string]checkEntry{}, exps: map[string]*exportEntry{}}
}

// dropPerson removes the export artefacts and check results of one person.
func (j *jobStore) dropPerson(person string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	for k, e := range j.exps {
		if e.person == person {
			j.bytes -= int64(len(e.file))
			delete(j.exps, k)
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

func (j *jobStore) slotLocked(person string, size int) string {
	now := j.now()
	own := 0
	for k, x := range j.exps {
		if now.Sub(x.at) > jobTTL {
			j.bytes -= int64(len(x.file))
			delete(j.exps, k)
			continue
		}
		if x.person == person {
			own++
		}
	}
	switch {
	case own >= maxExportsPerPerson:
		return "person"
	case len(j.exps) >= maxExports || j.bytes+int64(size) > maxExpBytes:
		return "full"
	}
	return ""
}

func (j *jobStore) putExport(e *exportEntry) bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.slotLocked(e.person, len(e.file)) != "" {
		return false
	}
	j.exps[e.dto.ID] = e
	j.bytes += int64(len(e.file))
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
