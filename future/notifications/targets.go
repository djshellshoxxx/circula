package notifications

import (
	"context"
	"crypto/ecdh"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// ErrRateLimited is returned when a member asks for too many verification mails.
var ErrRateLimited = errors.New("notifications: rate limited")

// ErrToken is returned for any bad, expired or tampered signed token.
var ErrToken = errors.New("notifications: invalid or expired token")

const (
	maxVerifyPerDay = 3
	verifyTTL       = 72 * time.Hour
)

// ---- sealed payloads ----

type pushSecret struct {
	Endpoint string `json:"e"`
	P256dh   string `json:"k"`
	Auth     string `json:"a"`
}

func sealAAD(kind, network, member, id string) []byte {
	parts := [4]string{kind, network, member, id}
	legacy := true
	for _, part := range parts {
		if strings.Contains(part, "|") {
			legacy = false
			break
		}
	}
	if legacy {
		// Preserve existing unambiguous targets. With pipe-free fields this
		// representation has exactly four delimiters and is injective.
		return []byte("ntf/v1|" + kind + "|" + network + "|" + member + "|" + id)
	}
	// Accepted identifiers may contain pipes. Bind their raw bytes with
	// explicit lengths, in a namespace disjoint from legacy target AAD.
	// Never fall back to ambiguous legacy AAD when opening such a target.
	aad := []byte("ntf/v2")
	for _, part := range parts {
		aad = binary.BigEndian.AppendUint64(aad, uint64(len(part)))
		aad = append(aad, part...)
	}
	return aad
}

// ---- push subscriptions ----

// PushSubscriptionInput is what the browser's PushSubscription.toJSON() gives.
type PushSubscriptionInput struct {
	Endpoint string
	P256dh   string
	Auth     string
}

// PushSubscriptionInfo is the owner-visible view: no endpoint path, no keys.
type PushSubscriptionInfo struct {
	ID           string
	Provider     string
	CreatedAt    time.Time
	SessionBound bool
}

func validatePushKeys(p256dh, auth string) error {
	k, err := decodeB64(p256dh)
	if err != nil || len(k) != 65 || k[0] != 4 {
		return invalid("push key")
	}
	if _, err := ecdh.P256().NewPublicKey(k); err != nil {
		return invalid("push key")
	}
	a, err := decodeB64(auth)
	if err != nil || len(a) != 16 {
		return invalid("push auth secret")
	}
	return nil
}

// AddPushSubscription registers a browser subscription for the actor after
// validating the endpoint against the provider policy. Re-adding an existing
// endpoint is idempotent (it reactivates a revoked one and rebinds the
// session).
func (s *Service) AddPushSubscription(ctx context.Context, a Actor, in PushSubscriptionInput) (PushSubscriptionInfo, error) {
	var zero PushSubscriptionInfo
	if err := s.requireActive(ctx, a); err != nil {
		return zero, err
	}
	u, err := s.cfg.Endpoints.Validate(in.Endpoint)
	if err != nil {
		return zero, err
	}
	if err := validatePushKeys(in.P256dh, in.Auth); err != nil {
		return zero, err
	}
	endpoint := u.String()
	hash := indexHash(s.cfg.Secrets.IndexKey, "push-endpoint", a.NetworkID, a.MemberID, endpoint)
	now := s.now()
	var info PushSubscriptionInfo
	var ab auditBatch
	err = s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		var id string
		var revoked sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT id,revoked_at FROM push_subscriptions WHERE network_id=? AND member_id=? AND endpoint_hash=?`,
			a.NetworkID, a.MemberID, hash).Scan(&id, &revoked)
		exists := err == nil
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if !exists || revoked.Valid {
			var n int
			if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE network_id=? AND member_id=? AND revoked_at IS NULL`, a.NetworkID, a.MemberID).Scan(&n); err != nil {
				return err
			}
			if n >= s.cfg.MaxPushPerMember {
				return invalid("too many push subscriptions")
			}
		}
		if !exists {
			id = newID()
		}
		blob, err := json.Marshal(pushSecret{Endpoint: endpoint, P256dh: in.P256dh, Auth: in.Auth})
		if err != nil {
			return err
		}
		sealed, err := s.cfg.Secrets.Keyring.Seal(blob, sealAAD("push", a.NetworkID, a.MemberID, id))
		if err != nil {
			return err
		}
		var sess any
		if a.SessionID != "" {
			sess = a.SessionID
		}
		if exists {
			if _, err := q.ExecContext(ctx, `UPDATE push_subscriptions SET sealed=?,session_id=?,revoked_at=NULL,revoke_reason=NULL,provider=? WHERE id=? AND network_id=? AND member_id=?`,
				sealed, sess, u.Hostname(), id, a.NetworkID, a.MemberID); err != nil {
				return err
			}
		} else if _, err := q.ExecContext(ctx, `INSERT INTO push_subscriptions(id,network_id,member_id,endpoint_hash,sealed,provider,session_id,created_at) VALUES(?,?,?,?,?,?,?,?)`,
			id, a.NetworkID, a.MemberID, hash, sealed, u.Hostname(), sess, now.Unix()); err != nil {
			return err
		}
		info = PushSubscriptionInfo{ID: id, Provider: u.Hostname(), CreatedAt: now, SessionBound: a.SessionID != ""}
		return s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, actorLabel(a), "push.add", id, "provider="+u.Hostname())
	})
	if err != nil {
		return zero, err
	}
	s.flushAudit(ctx, &ab)
	return info, nil
}

// ListPushSubscriptions returns the actor's active subscriptions.
func (s *Service) ListPushSubscriptions(ctx context.Context, a Actor) ([]PushSubscriptionInfo, error) {
	if err := a.validate(); err != nil {
		return nil, err
	}
	var out []PushSubscriptionInfo
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		rows, err := q.QueryContext(ctx, `SELECT id,provider,created_at,session_id IS NOT NULL FROM push_subscriptions WHERE network_id=? AND member_id=? AND revoked_at IS NULL ORDER BY created_at,id`, a.NetworkID, a.MemberID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var i PushSubscriptionInfo
			var at int64
			if err := rows.Scan(&i.ID, &i.Provider, &at, &i.SessionBound); err != nil {
				return err
			}
			i.CreatedAt = time.Unix(at, 0).UTC()
			out = append(out, i)
		}
		return rows.Err()
	})
	return out, err
}

// RevokePushSubscription revokes one of the actor's subscriptions. A foreign
// or unknown id is ErrNotFound. Revocation is effective for every send that has
// not already been handed to the push service.
func (s *Service) RevokePushSubscription(ctx context.Context, a Actor, id string) error {
	if err := a.validate(); err != nil {
		return err
	}
	if len(id) > 64 {
		return ErrNotFound
	}
	var ab auditBatch
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		res, err := q.ExecContext(ctx, `UPDATE push_subscriptions SET revoked_at=?,revoke_reason='user' WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL`,
			s.now().Unix(), id, a.NetworkID, a.MemberID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		return s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, actorLabel(a), "push.revoke", id, "reason=user")
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

// OnLogout revokes every push subscription bound to the actor's session. The
// host calls it when a session ends (explicit logout, expiry, revocation).
// Returns how many subscriptions were revoked.
func (s *Service) OnLogout(ctx context.Context, a Actor) (int, error) {
	if err := a.validate(); err != nil {
		return 0, err
	}
	if a.SessionID == "" {
		return 0, invalid("session id required")
	}
	var n int64
	var ab auditBatch
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		res, err := q.ExecContext(ctx, `UPDATE push_subscriptions SET revoked_at=?,revoke_reason='logout' WHERE network_id=? AND member_id=? AND session_id=? AND revoked_at IS NULL`,
			s.now().Unix(), a.NetworkID, a.MemberID, a.SessionID)
		if err != nil {
			return err
		}
		n, _ = res.RowsAffected()
		if n > 0 {
			return s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, "system", "push.revoke", "session", fmt.Sprintf("reason=logout count=%d", n))
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	s.flushAudit(ctx, &ab)
	return int(n), nil
}

// ---- e-mail target ----

// EmailInfo is the owner-visible e-mail target state.
type EmailInfo struct {
	Present  bool
	ID       string
	Hint     string // masked address
	Verified bool
	Bounced  bool
}

// normaliseAddress accepts a plain ASCII addr-spec only: no display name, no
// quoted local part, no comments, no whitespace or control characters.
func normaliseAddress(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > 254 {
		return "", invalid("email address")
	}
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if c <= 0x20 || c >= 0x7f || strings.IndexByte("\"<>(),;:\\[]", c) >= 0 {
			return "", invalid("email address")
		}
	}
	a, err := mail.ParseAddress(raw)
	if err != nil || a.Name != "" || a.Address != raw {
		return "", invalid("email address")
	}
	local, domain, ok := strings.Cut(raw, "@")
	if !ok || local == "" || len(local) > 64 || strings.Contains(domain, "@") || !strings.Contains(domain, ".") ||
		strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.Contains(domain, "..") {
		return "", invalid("email address")
	}
	for _, l := range strings.Split(domain, ".") {
		if l == "" || len(l) > 63 || l[0] == '-' || l[len(l)-1] == '-' {
			return "", invalid("email address")
		}
	}
	return local + "@" + strings.ToLower(domain), nil
}

func maskAddress(addr string) string {
	local, domain, _ := strings.Cut(addr, "@")
	return local[:1] + "***@" + domain
}

// AddEmailTarget sets the actor's e-mail address (replacing any previous one)
// and queues a confirmation message. Nothing but that confirmation is ever
// sent until the address is confirmed through the signed link (double opt-in),
// so a member cannot enrol a third party's mailbox.
func (s *Service) AddEmailTarget(ctx context.Context, a Actor, address string) (EmailInfo, error) {
	var zero EmailInfo
	if err := s.requireActive(ctx, a); err != nil {
		return zero, err
	}
	addr, err := normaliseAddress(address)
	if err != nil {
		return zero, err
	}
	hash := indexHash(s.cfg.Secrets.IndexKey, "email-address", a.NetworkID, a.MemberID, addr)
	now := s.now()
	var info EmailInfo
	var ab auditBatch
	err = s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		var curID, curHash string
		var verified, bounced sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT id,address_hash,verified_at,bounced_at FROM notification_email_targets WHERE network_id=? AND member_id=? AND revoked_at IS NULL`,
			a.NetworkID, a.MemberID).Scan(&curID, &curHash, &verified, &bounced)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && curHash == hash && verified.Valid && !bounced.Valid {
			info = EmailInfo{Present: true, ID: curID, Hint: maskAddress(addr), Verified: true}
			return nil // already confirmed: nothing to resend
		}
		var sent int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_delivery WHERE network_id=? AND member_id=? AND kind=? AND created_at>?`,
			a.NetworkID, a.MemberID, KindEmailVerification, now.Add(-24*time.Hour).Unix()).Scan(&sent); err != nil {
			return err
		}
		if sent >= maxVerifyPerDay {
			return ErrRateLimited
		}
		id := newID()
		if err == nil && curHash == hash && !bounced.Valid {
			id = curID // resend for the same unverified address
		} else {
			if curID != "" {
				if _, err := q.ExecContext(ctx, `UPDATE notification_email_targets SET revoked_at=? WHERE id=?`, now.Unix(), curID); err != nil {
					return err
				}
				if err := cancelReady(ctx, q, now, a.NetworkID, a.MemberID, ChannelEmail, "", "target_replaced"); err != nil {
					return err
				}
			}
			sealed, err := s.cfg.Secrets.Keyring.Seal([]byte(addr), sealAAD("email", a.NetworkID, a.MemberID, id))
			if err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `INSERT INTO notification_email_targets(id,network_id,member_id,address_hash,sealed,hint,created_at) VALUES(?,?,?,?,?,?,?)`,
				id, a.NetworkID, a.MemberID, hash, sealed, maskAddress(addr), now.Unix()); err != nil {
				return err
			}
		}
		ev := Event{NetworkID: a.NetworkID, EventID: "verify:" + id + ":" + newID(), MemberID: a.MemberID, Kind: KindEmailVerification, ExpiresAt: now.Add(verifyTTL)}
		if _, err := s.insertDelivery(ctx, q, now, ev, ChannelEmail, id); err != nil {
			return err
		}
		info = EmailInfo{Present: true, ID: id, Hint: maskAddress(addr)}
		return s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, actorLabel(a), "email.add", id, "verification_queued")
	})
	if err != nil {
		return zero, err
	}
	s.flushAudit(ctx, &ab)
	return info, nil
}

// GetEmailTarget returns the actor's current e-mail target state.
func (s *Service) GetEmailTarget(ctx context.Context, a Actor) (EmailInfo, error) {
	var info EmailInfo
	if err := a.validate(); err != nil {
		return info, err
	}
	err := s.st.View(ctx, func(ctx context.Context, q DBTX) error {
		var v, b sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT id,hint,verified_at,bounced_at FROM notification_email_targets WHERE network_id=? AND member_id=? AND revoked_at IS NULL`,
			a.NetworkID, a.MemberID).Scan(&info.ID, &info.Hint, &v, &b)
		if err == sql.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		info.Present, info.Verified, info.Bounced = true, v.Valid, b.Valid
		return nil
	})
	return info, err
}

// RevokeEmailTarget removes the actor's address and cancels queued mail.
func (s *Service) RevokeEmailTarget(ctx context.Context, a Actor) error {
	if err := a.validate(); err != nil {
		return err
	}
	var ab auditBatch
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := s.now()
		res, err := q.ExecContext(ctx, `UPDATE notification_email_targets SET revoked_at=? WHERE network_id=? AND member_id=? AND revoked_at IS NULL`, now.Unix(), a.NetworkID, a.MemberID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return ErrNotFound
		}
		if _, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='cancelled',last_error='target_revoked',updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE network_id=? AND member_id=? AND channel='email' AND status='ready'`, now.Unix(), a.NetworkID, a.MemberID); err != nil {
			return err
		}
		return s.addAudit(ctx, q, &ab, a.NetworkID, a.MemberID, actorLabel(a), "email.revoke", "", "reason=user")
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

func (s *Service) confirmToken(network, member, targetID string, expires time.Time) string {
	return signToken(s.cfg.Secrets.TokenKey, tokenPayload{P: tokConfirm, N: network, M: member, T: targetID, E: expires.Unix()})
}

// ConfirmEmail completes double opt-in from the signed link. It is idempotent
// and reveals nothing about why a token is rejected.
func (s *Service) ConfirmEmail(ctx context.Context, token string) error {
	p, err := verifyToken(s.cfg.Secrets.TokenKey, token, tokConfirm, s.now())
	if err != nil {
		return ErrToken
	}
	var ab auditBatch
	err = s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := s.now()
		var verified sql.NullInt64
		err := q.QueryRowContext(ctx, `SELECT verified_at FROM notification_email_targets WHERE id=? AND network_id=? AND member_id=? AND revoked_at IS NULL AND bounced_at IS NULL`,
			p.T, p.N, p.M).Scan(&verified)
		if err == sql.ErrNoRows {
			return ErrToken
		}
		if err != nil {
			return err
		}
		if verified.Valid {
			return nil
		}
		if _, err := q.ExecContext(ctx, `UPDATE notification_email_targets SET verified_at=? WHERE id=?`, now.Unix(), p.T); err != nil {
			return err
		}
		return s.addAudit(ctx, q, &ab, p.N, p.M, "email-link", "email.verify", p.T, "verified")
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

// ---- unsubscribe ----

// UnsubscribeToken returns the signed token for the one-click unsubscribe link
// of a member's channel. It carries no expiry: an old e-mail must keep working.
func (s *Service) UnsubscribeToken(network, member, channel string) string {
	return signToken(s.cfg.Secrets.TokenKey, tokenPayload{P: tokUnsubscribe, N: network, M: member, C: channel})
}

// UnsubscribeURL is the https link embedded in e-mail and List-Unsubscribe.
func (s *Service) UnsubscribeURL(network, member, channel string) string {
	return s.cfg.BaseURL + "/notifications/unsubscribe?t=" + s.UnsubscribeToken(network, member, channel)
}

// Unsubscribe disables every preference of the token's channel for its member
// and cancels queued deliveries, in one transaction: effective before the call
// returns. It needs no session (the token is the credential) and is idempotent.
func (s *Service) Unsubscribe(ctx context.Context, token string) error {
	p, err := verifyToken(s.cfg.Secrets.TokenKey, token, tokUnsubscribe, s.now())
	if err != nil || !ValidChannel(p.C) {
		return ErrToken
	}
	var ab auditBatch
	err = s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		now := s.now()
		res, err := q.ExecContext(ctx, `UPDATE notification_preferences SET enabled=0,updated_at=? WHERE network_id=? AND member_id=? AND channel=? AND enabled=1`, now.Unix(), p.N, p.M, p.C)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if err := cancelReady(ctx, q, now, p.N, p.M, p.C, "", "unsubscribed"); err != nil {
			return err
		}
		return s.addAudit(ctx, q, &ab, p.N, p.M, "unsubscribe-link", "preference.unsubscribe", p.C, fmt.Sprintf("disabled=%d", n))
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

// ---- bounces ----

const softBounceLimit = 5

// HandleBounce records a delivery-status notification for an e-mail target:
// a hard bounce suppresses the target immediately, repeated soft bounces
// suppress it after softBounceLimit. Called by the dispatcher for permanent
// recipient rejections and by operators processing DSNs (CLI).
func (s *Service) HandleBounce(ctx context.Context, networkID, targetID string, hard bool) error {
	if networkID == "" || targetID == "" || len(targetID) > 64 {
		return invalid("bounce")
	}
	var ab auditBatch
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		return s.bounceTx(ctx, q, &ab, networkID, targetID, hard)
	})
	if err != nil {
		return err
	}
	s.flushAudit(ctx, &ab)
	return nil
}

func (s *Service) bounceTx(ctx context.Context, q DBTX, ab *auditBatch, networkID, targetID string, hard bool) error {
	now := s.now()
	var member string
	var soft int
	var bounced sql.NullInt64
	err := q.QueryRowContext(ctx, `SELECT member_id,soft_bounces,bounced_at FROM notification_email_targets WHERE id=? AND network_id=?`, targetID, networkID).Scan(&member, &soft, &bounced)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if bounced.Valid {
		return nil
	}
	soft++
	suppress := hard || soft >= softBounceLimit
	if suppress {
		if _, err := q.ExecContext(ctx, `UPDATE notification_email_targets SET bounced_at=?,soft_bounces=? WHERE id=?`, now.Unix(), soft, targetID); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `UPDATE notification_delivery SET status='cancelled',last_error='bounced',updated_at=?,lease_owner=NULL,lease_until=NULL
			WHERE network_id=? AND member_id=? AND channel='email' AND status='ready'`, now.Unix(), networkID, member); err != nil {
			return err
		}
		kind := "soft_limit"
		if hard {
			kind = "hard"
		}
		return s.addAudit(ctx, q, ab, networkID, member, "system", "email.bounce", targetID, "suppressed="+kind)
	}
	_, err = q.ExecContext(ctx, `UPDATE notification_email_targets SET soft_bounces=? WHERE id=?`, soft, targetID)
	return err
}

// RotateKeys re-seals every stored target under the keyring's active key so
// retired keys can be removed from the keyring. Rows that cannot be opened with
// any retained key are reported in the error and left untouched. It returns
// the number of rows re-sealed.
func (s *Service) RotateKeys(ctx context.Context) (int, error) {
	type item struct {
		table, id, network, member, aadKind string
		sealed                              []byte
	}
	var items []item
	changed, unreadable := 0, 0
	err := s.st.Update(ctx, func(ctx context.Context, q DBTX) error {
		for _, t := range []struct{ table, kind string }{{"notification_email_targets", "email"}, {"push_subscriptions", "push"}} {
			rows, err := q.QueryContext(ctx, `SELECT id,network_id,member_id,sealed FROM `+t.table)
			if err != nil {
				return err
			}
			for rows.Next() {
				it := item{table: t.table, aadKind: t.kind}
				if err := rows.Scan(&it.id, &it.network, &it.member, &it.sealed); err != nil {
					rows.Close()
					return err
				}
				items = append(items, it)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()
		}
		for _, it := range items {
			aad := sealAAD(it.aadKind, it.network, it.member, it.id)
			plain, keyID, err := s.cfg.Secrets.Keyring.Open(it.sealed, aad)
			if err != nil {
				unreadable++
				continue
			}
			if keyID == s.cfg.Secrets.Keyring.ActiveID() {
				continue
			}
			sealed, err := s.cfg.Secrets.Keyring.Seal(plain, aad)
			if err != nil {
				return err
			}
			if _, err := q.ExecContext(ctx, `UPDATE `+it.table+` SET sealed=? WHERE id=?`, sealed, it.id); err != nil {
				return err
			}
			changed++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	if unreadable > 0 {
		return changed, fmt.Errorf("notifications: %d sealed rows could not be opened with the retained keys", unreadable)
	}
	return changed, nil
}
