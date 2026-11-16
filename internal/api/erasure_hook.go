package api

import "sync"

// The service layer is built before the HTTP handlers, but its
// Config.OnMemberErased must reach the handler's in-memory state (replay cache,
// export artefacts). The handler registers itself here when it is constructed
// and OnMemberErased forwards to it (A3-06). With several handlers in one
// process (tests) the most recently built one is the target.
var erasureHook struct {
	mu sync.Mutex
	s  *server
}

func setErasureHook(s *server) {
	erasureHook.mu.Lock()
	erasureHook.s = s
	erasureHook.mu.Unlock()
}

// OnMemberErased drops what the API keeps in memory for the person behind an
// erased membership: cached idempotent responses, export artefacts and check
// results. It is meant for service.Config.OnMemberErased and is safe to call
// before any handler exists (it then does nothing).
func OnMemberErased(netID, personID string) {
	erasureHook.mu.Lock()
	s := erasureHook.s
	erasureHook.mu.Unlock()
	if s == nil || personID == "" {
		return
	}
	s.idem.dropPerson(personID)
	s.jobs.dropPerson(personID)
}
