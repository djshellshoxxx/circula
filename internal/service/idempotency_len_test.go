package service_test

import (
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// D-03: the service accepts keys up to 255 characters, the same limit as the
// API edge (API-021), and refuses longer ones as validation-failed.
func TestIdempotencyKeyLengthMatchesEdge(t *testing.T) {
	w := newWorld(t, nil)
	in := service.ProposeInput{CounterpartyID: w.bM.ID, Direction: service.ProposerPays, Amount: 100}
	in.IdempotencyKey = strings.Repeat("k", 255)
	if _, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in); err != nil {
		t.Fatalf("255-character key refused: %v", err)
	}
	in.IdempotencyKey = strings.Repeat("k", 256)
	_, err := w.Svc.ProposeTransaction(w.Ctx, w.a, w.net.ID, in)
	if err == nil || !strings.Contains(err.Error(), "idempotency_key") {
		t.Fatalf("256-character key: err = %v, want idempotency_key validation error", err)
	}
}
