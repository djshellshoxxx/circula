package web

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/djshellshoxxx/circula/future/favorites"
)

type apiFn func(w http.ResponseWriter, r *http.Request, s Session)

// problem is an RFC 9457 problem details document.
type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Detail: detail})
}

func (h *Handler) jsonAPI(unsafe bool, fn apiFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.secureHeaders(w, false)
		s, err := h.auth.Authenticate(r)
		if err != nil || !s.valid() {
			writeProblem(w, http.StatusUnauthorized, "authentication required")
			return
		}
		if on, err := h.svc.Enabled(r.Context(), s.Actor); err != nil {
			writeProblem(w, http.StatusServiceUnavailable, "temporarily unavailable")
			return
		} else if !on {
			writeProblem(w, http.StatusNotFound, "not found")
			return
		}
		// X-Network scopes the request; it can only confirm the session's
		// network, never select another one.
		if n := r.Header.Get("X-Network"); n != "" && n != s.Actor.Network {
			writeProblem(w, http.StatusForbidden, "network does not match the session")
			return
		}
		if unsafe {
			if !sameOrigin(r) {
				writeProblem(w, http.StatusForbidden, "cross-site request refused")
				return
			}
			if !csrfOK(s.CSRF, r.Header.Get("X-CSRF-Token")) {
				writeProblem(w, http.StatusForbidden, "missing or invalid CSRF token")
				return
			}
		}
		fn(w, r, s)
	}
}

// decode reads one JSON object: required Content-Type, 64 KiB cap, unknown
// fields and trailing data rejected.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		writeProblem(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeProblem(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeProblem(w, http.StatusBadRequest, "malformed JSON body")
		return false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		writeProblem(w, http.StatusBadRequest, "unexpected data after JSON body")
		return false
	}
	return true
}

type apiFavorite struct {
	MemberID    string    `json:"member_id"`
	DisplayName string    `json:"display_name"`
	Nickname    string    `json:"nickname"`
	Pinned      bool      `json:"pinned"`
	Position    int       `json:"position"`
	CreatedAt   time.Time `json:"created_at"`
}

func toAPI(v favorites.View) apiFavorite {
	return apiFavorite{MemberID: v.Target, DisplayName: v.DisplayName, Nickname: v.Nickname, Pinned: v.Pinned, Position: v.Order, CreatedAt: v.CreatedAt}
}

type apiDraft struct {
	NetworkID            string  `json:"network_id"`
	RecipientMemberID    string  `json:"recipient_member_id"`
	RecipientDisplayName string  `json:"recipient_display_name"`
	Description          string  `json:"description"`
	Amount               *int64  `json:"amount"`
	IdempotencyKey       string  `json:"idempotency_key"`
	RequestKey           string  `json:"request_key"`
	SourceTradeID        *string `json:"source_trade_id"`
}

func toAPIDraft(d favorites.Draft) apiDraft {
	out := apiDraft{NetworkID: d.Network, RecipientMemberID: d.Recipient, RecipientDisplayName: d.RecipientName, Description: d.Description,
		IdempotencyKey: d.Intent, RequestKey: d.RequestKey}
	if d.TradeID != "" {
		out.SourceTradeID = &d.TradeID
	}
	return out
}

// fail maps service errors to problem responses. Refusals about other
// members are uniform so responses cannot be used to probe the directory.
// ownerOnly selects the status for ErrUnavailable: operations without a
// target parameter can only be refused because of the caller's own state.
func fail(w http.ResponseWriter, err error, ownerOnly bool) {
	var ie *favorites.InputError
	switch {
	case errors.As(err, &ie):
		writeProblem(w, http.StatusUnprocessableEntity, ie.Error())
	case errors.Is(err, favorites.ErrUnavailable):
		if ownerOnly {
			writeProblem(w, http.StatusForbidden, "saved trading partners are not available to this account")
		} else {
			writeProblem(w, http.StatusUnprocessableEntity, "that member cannot be saved")
		}
	case errors.Is(err, favorites.ErrNotFound), errors.Is(err, favorites.ErrDisabled):
		writeProblem(w, http.StatusNotFound, "not found")
	case errors.Is(err, favorites.ErrExists), errors.Is(err, favorites.ErrStale), errors.Is(err, favorites.ErrKeyReuse), errors.Is(err, favorites.ErrLimit):
		writeProblem(w, http.StatusConflict, err.Error())
	case errors.Is(err, favorites.ErrInvalidOrder), errors.Is(err, favorites.ErrTradeUnavailable):
		writeProblem(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, favorites.ErrPolicy):
		writeProblem(w, http.StatusServiceUnavailable, "temporarily unavailable")
	default:
		writeProblem(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) apiList(w http.ResponseWriter, r *http.Request, s Session) {
	q := r.URL.Query()
	limit := 0
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > favorites.MaxPageLimit {
			writeProblem(w, http.StatusBadRequest, "limit must be 1 to 200")
			return
		}
		limit = n
	}
	pg, err := h.svc.List(r.Context(), s.Actor, favorites.ListOptions{Limit: limit, Cursor: q.Get("cursor")})
	if err != nil {
		var ie *favorites.InputError
		if errors.As(err, &ie) {
			writeProblem(w, http.StatusBadRequest, ie.Error())
			return
		}
		fail(w, err, true)
		return
	}
	data := make([]apiFavorite, 0, len(pg.Items))
	for _, v := range pg.Items {
		data = append(data, toAPI(v))
	}
	var next *string
	if pg.Next != "" {
		next = &pg.Next
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "next_cursor": next, "revision": pg.Revision})
}

func (h *Handler) apiAdd(w http.ResponseWriter, r *http.Request, s Session) {
	var in struct {
		MemberID string `json:"member_id"`
		Nickname string `json:"nickname"`
		Pinned   bool   `json:"pinned"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, created, err := h.svc.Add(r.Context(), s.Actor, favorites.AddRequest{Target: in.MemberID, Nickname: in.Nickname, Pinned: in.Pinned})
	if err != nil {
		fail(w, err, false)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
		w.Header().Set("Location", h.api+"/"+url.PathEscape(v.Target))
	}
	writeJSON(w, status, toAPI(v))
}

func (h *Handler) apiPatch(w http.ResponseWriter, r *http.Request, s Session) {
	var in struct {
		Nickname *string `json:"nickname"`
		Pinned   *bool   `json:"pinned"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Nickname == nil && in.Pinned == nil {
		writeProblem(w, http.StatusUnprocessableEntity, "nickname or pinned required")
		return
	}
	id := r.PathValue("id")
	// Validate before mutating so a bad nickname cannot leave a half-applied patch.
	if in.Nickname != nil {
		if _, err := favorites.CleanNickname(*in.Nickname); err != nil {
			fail(w, err, true)
			return
		}
	}
	if in.Nickname != nil {
		if err := h.svc.SetNickname(r.Context(), s.Actor, id, *in.Nickname); err != nil {
			fail(w, err, true)
			return
		}
	}
	if in.Pinned != nil {
		if err := h.svc.SetPinned(r.Context(), s.Actor, id, *in.Pinned); err != nil {
			fail(w, err, true)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) apiDelete(w http.ResponseWriter, r *http.Request, s Session) {
	if _, err := h.svc.Remove(r.Context(), s.Actor, r.PathValue("id")); err != nil {
		fail(w, err, true)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) apiReorder(w http.ResponseWriter, r *http.Request, s Session) {
	var in struct {
		Revision int64    `json:"revision"`
		Targets  []string `json:"member_ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	rev, err := h.svc.Reorder(r.Context(), s.Actor, favorites.ReorderRequest{Revision: in.Revision, Targets: in.Targets})
	if err != nil {
		fail(w, err, true)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"revision": rev})
}

func (h *Handler) apiCreateDraft(w http.ResponseWriter, r *http.Request, s Session) {
	var in struct {
		TradeID    string `json:"trade_id"`
		MemberID   string `json:"member_id"`
		RequestKey string `json:"request_key"`
	}
	if !decode(w, r, &in) {
		return
	}
	if (in.TradeID == "") == (in.MemberID == "") {
		writeProblem(w, http.StatusUnprocessableEntity, "exactly one of trade_id or member_id is required")
		return
	}
	var d favorites.Draft
	var err error
	if in.TradeID != "" {
		d, err = h.svc.RepeatTrade(r.Context(), s.Actor, in.TradeID, in.RequestKey)
	} else {
		d, err = h.svc.DraftFromFavorite(r.Context(), s.Actor, in.MemberID, in.RequestKey)
	}
	if err != nil {
		fail(w, err, false)
		return
	}
	writeJSON(w, http.StatusCreated, toAPIDraft(d))
}

func (h *Handler) apiGetDraft(w http.ResponseWriter, r *http.Request, s Session) {
	d, err := h.svc.GetDraft(r.Context(), s.Actor, r.PathValue("key"))
	if err != nil {
		if errors.Is(err, favorites.ErrUnavailable) {
			err = favorites.ErrNotFound
		}
		fail(w, err, true)
		return
	}
	writeJSON(w, http.StatusOK, toAPIDraft(d))
}
