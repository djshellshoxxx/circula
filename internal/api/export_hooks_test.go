package api

import "net/http"

// Hooks for the external export tests (package api_test).

// SetExportByteCap lowers the export store's byte cap of handler h.
func SetExportByteCap(h http.Handler, n int64) {
	j := h.(*server).jobs
	j.mu.Lock()
	j.capBytes = n
	j.mu.Unlock()
}

// ExportStoreUsage reports the bytes stored and reserved by running exports,
// and the number of running exports, of handler h.
func ExportStoreUsage(h http.Handler) (stored, inflight int64, running int) {
	j := h.(*server).jobs
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.bytes, j.inflight, j.nRunning
}
