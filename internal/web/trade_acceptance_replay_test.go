package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
	"github.com/djshellshoxxx/circula/internal/store"
	"github.com/djshellshoxxx/circula/internal/testutil"
)

// Run a competing command immediately after a read completes, outside the
// read transaction. This serializes a permitted interleaving without relying
// on scheduler timing or wrapping away optional transaction interfaces.
type acceptanceReadBoundaryStore struct {
	store.Store
	afterRead func() error
	reached   bool
}

func (s *acceptanceReadBoundaryStore) View(ctx context.Context, fn func(store.Tx) error) error {
	if err := s.Store.View(ctx, fn); err != nil {
		return err
	}
	if callback := s.afterRead; callback != nil {
		s.afterRead = nil
		s.reached = true
		return callback()
	}
	return nil
}

// DOM-014, DOM-016, LED-4: a concurrent same-key acceptance must remain successful
// when its posting reaches the supported balance boundary. Exercise the
// authenticated handler directly to place the competing command at its first
// read; the ordinary trade-flow tests cover the surrounding HTTP middleware.
func TestTradeAcceptanceConcurrentReplayAtBalanceBoundary(t *testing.T) {
	var boundary *acceptanceReadBoundaryStore
	env := testutil.NewEnvWith(t, func(base store.Store) store.Store {
		boundary = &acceptanceReadBoundaryStore{Store: base}
		return boundary
	})
	w := &world{env: env}
	w.admin = env.Person("admin@example.org")
	w.net, w.adminM = env.Network(w.admin, "town", nil)
	w.alice, w.aliceM = env.Member(w.admin, w.net, "alice@example.org")
	w.bob, w.bobM = env.Member(w.admin, w.net, "bob@example.org")
	tx := testutil.Must(env.Svc.ProposeTransaction(env.Ctx, w.bob, w.net.ID, service.ProposeInput{
		CounterpartyID: w.aliceM.ID, Direction: service.ProposerPays, Amount: 500, IdempotencyKey: env.Key(),
	}))
	setPreviewBalance(t, w, w.aliceM.ID, domain.MaxBalance-500)
	s := newServer(env.Svc, Options{CSRFKey: []byte("0123456789abcdef0123456789abcdef"), Clock: env.Clock})
	c := &rc{s: s, ctx: env.Ctx, actor: w.alice, Net: w.net, Me: w.aliceM}
	key := env.Key()
	path := w.base() + "/trades/" + tx.ID
	r := httptest.NewRequest(http.MethodPost, path+"/accept", strings.NewReader(url.Values{"idem": {key}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetPathValue("id", tx.ID)
	r.SetPathValue("action", "accept")
	r.SetPathValue("slug", w.net.Slug)
	var competingErr error
	boundary.afterRead = func() error {
		_, competingErr = env.Svc.AcceptTransaction(env.Ctx, w.alice, w.net.ID, service.TxRef{TxID: tx.ID, IdempotencyKey: key})
		return competingErr
	}
	written := httptest.NewRecorder()
	s.tradeActionPost(written, r, c)
	if !boundary.reached {
		t.Fatal("the competing acceptance did not run")
	}
	if competingErr != nil {
		t.Fatalf("the competing acceptance failed: %v", competingErr)
	}
	if written.Code != http.StatusSeeOther || written.Header().Get("Location") != path {
		t.Errorf("same-key acceptance returned %d %s, want successful redirect to %s", written.Code, written.Header().Get("Location"), path)
	}
	posted := testutil.Must(env.Svc.GetTransaction(env.Ctx, w.alice, w.net.ID, tx.ID))
	balance := testutil.Must(env.Svc.GetBalance(env.Ctx, w.alice, w.net.ID))
	if posted.Tx.Status != domain.TxCompleted || len(posted.Entries) != 2 || balance.Balance != domain.MaxBalance {
		t.Fatalf("expected one committed posting: status=%s entries=%d balance=%d", posted.Tx.Status, len(posted.Entries), balance.Balance)
	}
}
