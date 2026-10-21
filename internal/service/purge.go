package service

import (
	"context"
	"regexp"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

const (
	purgeBatch         = 1000
	throttleRetention  = 24 * time.Hour
	revokedSessionKeep = 24 * time.Hour
)

// PurgeReport counts the rows Purge removed.
type PurgeReport struct {
	Sessions        int64
	Throttles       int64
	IdempotencyKeys int64
}

// Purge deletes rows that can no longer have any effect (SA-12, DATA-01, DAT
// section 9 retention): sessions that are absolutely expired, idle past
// Config.SessionIdle, or revoked more than a day ago; login throttle rows whose
// lock (or last failure) ended more than 24 hours ago; and idempotency records
// past their expiry. It never touches ledger, audit or membership data.
//
// Each category is deleted in bounded batches, one short write unit per batch,
// so the write lock is never held for a long scan and postings interleave.
// Purge is idempotent and safe to run concurrently with traffic; run it from
// the job runner (for example hourly). It takes no actor: it is an internal
// maintenance call and must not be exposed to users.
func (s *Service) Purge(ctx context.Context) (PurgeReport, error) {
	var rep PurgeReport
	steps := []struct {
		n   *int64
		del func(tx store.Tx, now time.Time) (int64, error)
	}{
		{&rep.Sessions, func(tx store.Tx, now time.Time) (int64, error) {
			return tx.DeleteDeadSessions(now, now.Add(-s.cfg.SessionIdle), now.Add(-revokedSessionKeep), purgeBatch)
		}},
		{&rep.Throttles, func(tx store.Tx, now time.Time) (int64, error) {
			return tx.DeleteStaleThrottles(now.Add(-throttleRetention), purgeBatch)
		}},
		{&rep.IdempotencyKeys, func(tx store.Tx, now time.Time) (int64, error) {
			return tx.DeleteExpiredIdem(now, purgeBatch)
		}},
	}
	for _, st := range steps {
		for {
			var n int64
			err := s.update(ctx, func(tx store.Tx) (err error) {
				n, err = st.del(tx, s.now())
				return err
			})
			if err != nil {
				return rep, err
			}
			*st.n += n
			if n < purgeBatch {
				break
			}
		}
	}
	return rep, nil
}

// secretToken matches the bearer-style credentials that appear in URLs and
// messages: invitation tokens (inv_...), API tokens (cir_...).
var secretToken = regexp.MustCompile(`\b(inv|cir)_[A-Za-z0-9_-]{8,}`)

// RedactSecrets replaces invitation and API tokens in s with a fixed marker.
// Adapters must pass any request path, URL or error text through it before
// logging (SA-13, API-07, SEC-020); better still, log the route pattern.
func RedactSecrets(s string) string { return secretToken.ReplaceAllString(s, "${1}_REDACTED") }
