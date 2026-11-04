package service

// Observer receives operational events for metrics (OPS-014). Callbacks run
// after the unit of work that caused them has committed, on the calling
// goroutine, and must be fast and must not call back into the Service.
type Observer interface {
	// Posted reports n ledger postings committed by one write unit.
	Posted(n int)
	// Replayed reports n idempotent retries answered from a stored record by
	// one write unit (no new state was created).
	Replayed(n int)
}

// SetObserver installs o (nil removes it). Call before serving starts.
func (s *Service) SetObserver(o Observer) { s.obs = o }

// unitStats counts events inside one write unit. Units are serialised by the
// store's write gate, so the fields are only touched by one unit at a time.
type unitStats struct{ posts, replays int }
