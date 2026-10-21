package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

const maxKeyLen = 255 // matches the API edge and API-021 (D-03)

func requireKey(key string) error {
	if key == "" {
		return domain.Errf(domain.CodeKeyRequired, "idempotency key required")
	}
	if len(key) > maxKeyLen {
		return domain.Invalid("idempotency_key", "too long")
	}
	return nil
}

func fingerprint(payload any) string {
	b, _ := json.Marshal(payload)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// keyRef is the non-reversible reference stored in audit events (DAT 7.1).
func keyRef(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:8])
}

// idempotent runs create at most once per (network, actor, operation, key).
// A retry with the same payload returns load(originalRef) without running
// create; the same key with a different payload fails with ErrKeyReused
// (DAT-025..027). The record is written in the caller's unit so it commits or
// rolls back together with the change. Failures are never recorded (API-025):
// the whole unit rolls back, so the key stays reusable. Concurrent duplicates
// serialise on the write gate and the loser replays.
func idempotent[T any](s *Service, tx store.Tx, netID string, actor domain.Actor, op, key string,
	payload any, create func() (string, error), load func(string) (T, error)) (T, error) {

	var zero T
	if err := requireKey(key); err != nil {
		return zero, err
	}
	actorID := actor.PersonID
	if actor.System {
		actorID = "system"
	}
	fp := fingerprint(payload)
	now := s.now()
	rec, err := tx.GetIdem(netID, actorID, op, key)
	if err != nil {
		return zero, err
	}
	if rec != nil {
		if rec.ExpiresAt.After(now) {
			if rec.Fingerprint != fp {
				return zero, domain.Errf(domain.CodeKeyReused, "idempotency key reused with a different request")
			}
			s.unit.replays++
			return load(rec.ResultRef)
		}
		if err := tx.DeleteIdem(netID, actorID, op, key); err != nil { // expired: key is free (DAT-028)
			return zero, err
		}
	}
	ref, err := create()
	if err != nil {
		return zero, err
	}
	if err := tx.PutIdem(store.IdemRecord{NetworkID: netID, ActorID: actorID, Operation: op, Key: key,
		Fingerprint: fp, ResultRef: ref, CreatedAt: now, ExpiresAt: now.Add(s.cfg.IdempotencyTTL)}); err != nil {
		return zero, err
	}
	return load(ref)
}
