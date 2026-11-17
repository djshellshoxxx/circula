package notifications

import (
	"fmt"
	"io"
	"sync/atomic"
)

// Metrics are in-process counters; labels are limited to channel and a fixed
// outcome vocabulary, never member, network, address or endpoint values.
type Metrics struct {
	Enqueued   atomic.Int64
	Duplicates atomic.Int64
	Sent       atomic.Int64
	Retried    atomic.Int64
	Failed     atomic.Int64
	Cancelled  atomic.Int64
	Expired    atomic.Int64
	Deferred   atomic.Int64
	LeaseLost  atomic.Int64
	Poisoned   atomic.Int64
}

// WritePrometheus renders the counters (plus an optional DB snapshot) in the
// Prometheus text exposition format.
func (m *Metrics) WritePrometheus(w io.Writer, st *StatusReport) {
	c := func(name, help string, v int64) {
		fmt.Fprintf(w, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, v)
	}
	c("circula_notifications_enqueued_total", "Outbox rows created.", m.Enqueued.Load())
	c("circula_notifications_duplicates_total", "Enqueue calls that matched an existing row.", m.Duplicates.Load())
	c("circula_notifications_sent_total", "Deliveries accepted by a provider.", m.Sent.Load())
	c("circula_notifications_retried_total", "Deliveries scheduled for retry.", m.Retried.Load())
	c("circula_notifications_failed_total", "Deliveries failed permanently or exhausted.", m.Failed.Load())
	c("circula_notifications_cancelled_total", "Deliveries cancelled (opt-out, inactive, no target).", m.Cancelled.Load())
	c("circula_notifications_expired_total", "Deliveries expired before sending.", m.Expired.Load())
	c("circula_notifications_deferred_total", "Deliveries deferred by quiet hours.", m.Deferred.Load())
	c("circula_notifications_lease_lost_total", "Acks refused because the lease moved.", m.LeaseLost.Load())
	c("circula_notifications_poisoned_total", "Rows failed after too many abandoned claims.", m.Poisoned.Load())
	if st != nil {
		fmt.Fprint(w, "# HELP circula_notifications_deliveries Current outbox rows.\n# TYPE circula_notifications_deliveries gauge\n")
		for _, r := range st.Deliveries {
			fmt.Fprintf(w, "circula_notifications_deliveries{channel=%q,status=%q} %d\n", r.Channel, r.Status, r.Count)
		}
		fmt.Fprintf(w, "# HELP circula_notifications_oldest_ready_age_seconds Age of the oldest due ready row.\n# TYPE circula_notifications_oldest_ready_age_seconds gauge\ncircula_notifications_oldest_ready_age_seconds %d\n", st.OldestReadyAgeSeconds)
	}
}
