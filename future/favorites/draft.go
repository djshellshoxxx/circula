package favorites

import (
	"context"
	"fmt"
)

// RepeatTrade creates (or re-reads) a quick-repeat draft from a historical
// trade. The trade is loaded authoritatively through the TradeSource, the
// owner and the recipient are re-validated against the current Policy, only
// the recipient and description are copied, and Amount stays nil.
//
// requestKey names one deliberate user intent (the host form generates a new
// one per page render). The same key returns the same draft and idempotency
// Intent for DraftTTL; a different key, or an empty one, yields a new random
// Intent, so two repeats of one old trade are two independent proposals. A key
// reused for a different trade is ErrKeyReuse. Nothing is posted or accepted.
func (s *Service) RepeatTrade(ctx context.Context, a Actor, tradeID, requestKey string) (Draft, error) {
	if s.trades == nil || !validID(tradeID) {
		return Draft{}, ErrTradeUnavailable
	}
	key, err := s.resolveKey(requestKey)
	if err != nil {
		return Draft{}, err
	}
	if err := s.checkActor(ctx, a); err != nil {
		return Draft{}, err
	}
	trade, err := s.trades.Trade(ctx, a, tradeID)
	if err != nil {
		return Draft{}, fmt.Errorf("%w: %v", ErrTradeUnavailable, err)
	}
	if trade.ID == "" {
		trade.ID = tradeID
	}
	if trade.ID != tradeID || !validID(trade.Payee) || trade.Payer != a.Member || trade.Network != a.Network {
		return Draft{}, ErrTradeUnavailable
	}
	owner, m, err := s.owner(ctx, a, trade.Payee)
	if err != nil {
		return Draft{}, err
	}
	target, ok := m[trade.Payee]
	if !ok {
		return Draft{}, ErrUnavailable
	}
	d, err := Repeat(owner, target, trade)
	if err != nil {
		return Draft{}, err
	}
	d, err = s.storeDraft(ctx, a, key, d)
	d.RecipientName = target.DisplayName
	return d, err
}

// DraftFromFavorite starts an unpriced draft toward a current favorite, with
// no description. Same key semantics as RepeatTrade.
func (s *Service) DraftFromFavorite(ctx context.Context, a Actor, target, requestKey string) (Draft, error) {
	if !validID(target) {
		return Draft{}, ErrUnavailable
	}
	key, err := s.resolveKey(requestKey)
	if err != nil {
		return Draft{}, err
	}
	owner, m, err := s.owner(ctx, a, target)
	if err != nil {
		return Draft{}, err
	}
	t, ok := m[target]
	if !ok {
		return Draft{}, ErrUnavailable
	}
	if _, err := Save(owner, t, ""); err != nil {
		return Draft{}, err
	}
	var found bool
	if err := s.store.View(ctx, func(tx Tx) error {
		_, found, err = tx.GetFavorite(ctx, a.Network, a.Member, target)
		return err
	}); err != nil {
		return Draft{}, err
	}
	if !found {
		return Draft{}, ErrNotFound
	}
	intent, err := newIntent()
	if err != nil {
		return Draft{}, err
	}
	d, err := s.storeDraft(ctx, a, key, Draft{Network: a.Network, Recipient: target, Intent: intent})
	d.RecipientName = t.DisplayName
	return d, err
}

// GetDraft re-reads a stored draft by its RequestKey, after re-validating the
// owner and recipient against the current Policy. Unknown or expired keys are
// ErrNotFound; other members' keys are indistinguishable from unknown ones.
func (s *Service) GetDraft(ctx context.Context, a Actor, requestKey string) (Draft, error) {
	if !validKey(requestKey) {
		return Draft{}, ErrNotFound
	}
	if err := s.checkActor(ctx, a); err != nil {
		return Draft{}, err
	}
	var in Intent
	var ok bool
	if err := s.store.View(ctx, func(tx Tx) (err error) {
		in, ok, err = tx.GetIntent(ctx, a.Network, a.Member, requestKey)
		return err
	}); err != nil {
		return Draft{}, err
	}
	if !ok || !in.ExpiresAt.After(s.utcNow()) {
		return Draft{}, ErrNotFound
	}
	owner, m, err := s.owner(ctx, a, in.Target)
	if err != nil {
		return Draft{}, err
	}
	t, ok := m[in.Target]
	if !ok {
		return Draft{}, ErrUnavailable
	}
	if _, err := Save(owner, t, ""); err != nil {
		return Draft{}, err
	}
	d := draftOf(in)
	d.RecipientName = t.DisplayName
	return d, nil
}

func (s *Service) resolveKey(k string) (string, error) {
	if k == "" {
		return newIntent()
	}
	if !validKey(k) {
		return "", invalid("request_key", "must be 1 to 64 letters, digits, hyphen or underscore")
	}
	return k, nil
}

func draftOf(in Intent) Draft {
	return Draft{Network: in.Network, Recipient: in.Target, Description: in.Description, Intent: in.Intent, RequestKey: in.Key, TradeID: in.TradeID}
}

func (s *Service) storeDraft(ctx context.Context, a Actor, key string, d Draft) (Draft, error) {
	var out Draft
	err := s.store.Update(ctx, func(tx Tx) error {
		now := s.utcNow()
		in, ok, err := tx.GetIntent(ctx, a.Network, a.Member, key)
		if err != nil {
			return err
		}
		if ok {
			if in.ExpiresAt.After(now) {
				if in.TradeID != d.TradeID || in.Target != d.Recipient {
					return ErrKeyReuse
				}
				out = draftOf(in)
				return nil
			}
			if err := tx.DeleteIntent(ctx, a.Network, a.Member, key); err != nil {
				return err
			}
		}
		n, err := tx.CountIntents(ctx, a.Network, a.Member, now)
		if err != nil {
			return err
		}
		if n >= s.maxDrafts {
			return ErrLimit
		}
		in = Intent{Network: a.Network, Owner: a.Member, Key: key, TradeID: d.TradeID, Target: d.Recipient,
			Description: d.Description, Intent: d.Intent, CreatedAt: now, ExpiresAt: now.Add(s.draftTTL)}
		if err := tx.InsertIntent(ctx, in); err != nil {
			return err
		}
		out = draftOf(in)
		return nil
	})
	if err != nil {
		return Draft{}, err
	}
	return out, nil
}
