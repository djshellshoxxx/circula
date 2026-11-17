package web

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	qr "github.com/djshellshoxxx/circula/future/qr"
)

const (
	baseCSP = "default-src 'none'; style-src 'self'; script-src 'self'; img-src 'self' data:; form-action 'self'; base-uri 'none'; frame-ancestors 'none'"
	// The camera page additionally needs a media stream and, only for browsers
	// without BarcodeDetector, the bundled scanner's Blob worker. See README.
	scanCSP = baseCSP + "; connect-src 'self'; media-src 'self' blob:; worker-src 'self' blob:"
)

func (h *handler) loadTemplates() error {
	h.pages = map[string]*template.Template{}
	for _, name := range []string{"home", "scan", "confirm", "result", "created", "status", "print", "error"} {
		t, err := template.New(name).ParseFS(templateFS, "templates/layout.html", "templates/error.html", "templates/"+name+".html")
		if err != nil {
			return err
		}
		h.pages[name] = t
	}
	return nil
}

// pageData is the single view model shared by all templates.
type pageData struct {
	h       *handler
	Title   string
	Nav     string
	CSRF    string
	Name    string
	Network string
	Notice  string
	Err     string

	ErrTitle, ErrDetail string

	Alias        qr.AliasInfo
	PNG          template.URL
	PNGSize      int
	MaxNote      int
	Requests     []qr.RequestView
	P            qr.Preview
	Code         string
	V            qr.RequestView
	URL, Compact string
	TxID         string
	State        string
	StateText    string
	Replayed     bool
	netID        string
}

func (d *pageData) NetworkLabel() string {
	if d.Network != "" {
		return d.Network
	}
	return "your network"
}

// FormKey returns a fresh idempotency key for one rendered form so a double
// submit of that form is a retry, never a second action.
func (d *pageData) FormKey(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + "-" + hex.EncodeToString(b)
}

func (d *pageData) Amount(v int64) string { return d.h.cfg.FormatAmount(d.netID, v) }

func (d *pageData) StatusText(s qr.RequestStatus) string {
	switch s {
	case qr.StatusOpen:
		return "Open: not yet scanned or claimed. Nothing has been paid."
	case qr.StatusPending:
		return "Pending: a member confirmed and sent a proposal. It is NOT paid until you accept it."
	case qr.StatusCompleted:
		return "Completed: both members agreed and the trade was recorded."
	case qr.StatusClosed:
		return "Closed: the proposal ended without any payment."
	case qr.StatusExpired:
		return "Expired: it was never claimed."
	case qr.StatusRevoked:
		return "Revoked: it can no longer be used."
	}
	return string(s)
}

func (h *handler) newData(s Session, title, nav string) *pageData {
	return &pageData{h: h, Title: title, Nav: nav, CSRF: s.CSRF, Name: orDefault(s.DisplayName, "you"), Network: s.NetworkName,
		MaxNote: h.svc.MaxNoteRunes(), PNGSize: 256, netID: s.Principal.NetworkID}
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func (h *handler) render(w http.ResponseWriter, name string, status int, d *pageData, csp string) {
	secureHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", csp)
	var sb strings.Builder
	if err := h.pages[name].ExecuteTemplate(&sb, "layout", d); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write([]byte(sb.String()))
}

type htmlFn func(w http.ResponseWriter, r *http.Request, s Session) error

func (h *handler) html(fn htmlFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := h.cfg.Auth.Authenticate(r)
		if err != nil {
			if !errors.Is(err, ErrUnauthenticated) {
				h.fail(r, err)
			}
			if h.cfg.LoginURL != "" && r.Method == http.MethodGet {
				secureHeaders(w)
				http.Redirect(w, r, h.cfg.LoginURL, http.StatusSeeOther)
				return
			}
			h.errorPage(w, r, Session{}, ErrUnauthenticated)
			return
		}
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBody)
			ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if ct != "application/x-www-form-urlencoded" {
				h.errorPage(w, r, s, errMedia)
				return
			}
			if err := r.ParseForm(); err != nil {
				var mbe *http.MaxBytesError
				if errors.As(err, &mbe) {
					h.errorPage(w, r, s, errTooLarge)
				} else {
					h.errorPage(w, r, s, fmt.Errorf("%w: unreadable form", qr.ErrInvalid))
				}
				return
			}
			if s.CookieAuth && !csrfOK(s.CSRF, r.PostForm.Get("csrf")) {
				h.errorPage(w, r, s, errCSRF)
				return
			}
		}
		if err := fn(w, r, s); err != nil {
			h.errorPage(w, r, s, err)
		}
	}
}

func (h *handler) errorPage(w http.ResponseWriter, r *http.Request, s Session, err error) {
	m := mapError(err)
	if m.status >= 500 {
		h.fail(r, err)
	}
	d := h.newData(s, m.title, "")
	d.ErrTitle, d.ErrDetail = m.title, m.detail
	if m.status == http.StatusUnauthorized && h.cfg.LoginURL != "" {
		d.ErrDetail += " Sign in again to continue."
	}
	h.render(w, "error", m.status, d, baseCSP)
}

// ---- pages ---------------------------------------------------------------

func (h *handler) aliasData(r *http.Request, s Session) (*pageData, error) {
	a, err := h.svc.MyAlias(r.Context(), s.Principal)
	if err != nil {
		return nil, err
	}
	d := h.newData(s, "My QR and requests", "home")
	d.Alias = a
	return d, nil
}

func (h *handler) pageHome(w http.ResponseWriter, r *http.Request, s Session) error {
	d, err := h.aliasData(r, s)
	if err != nil {
		return err
	}
	if v := r.URL.Query().Get("done"); v == "rotated" {
		d.Notice = "Your QR code was rotated. The old code no longer works."
	}
	d.Requests, _, err = h.svc.ListRequests(r.Context(), s.Principal, 10, "")
	if err != nil {
		return err
	}
	h.render(w, "home", http.StatusOK, d, baseCSP)
	return nil
}

func (h *handler) pageMyPNG(w http.ResponseWriter, r *http.Request, s Session) error {
	b, _, err := h.svc.RenderAlias(r.Context(), s.Principal)
	if err != nil {
		return err
	}
	if r.URL.Query().Get("download") == "1" {
		return h.sendPNG(w, b, "circula-member-qr.png")
	}
	secureHeaders(w)
	w.Header().Set("Content-Type", "image/png")
	_, err = w.Write(b)
	return err
}

func dataURI(b []byte) template.URL {
	return template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(b))
}

func (h *handler) pagePrint(w http.ResponseWriter, r *http.Request, s Session) error {
	b, a, err := h.svc.RenderAlias(r.Context(), s.Principal)
	if err != nil {
		return err
	}
	d := h.newData(s, "Printable QR code", "home")
	d.Alias, d.PNG, d.PNGSize = a, dataURI(b), 512
	h.render(w, "print", http.StatusOK, d, baseCSP)
	return nil
}

func (h *handler) pageScan(w http.ResponseWriter, r *http.Request, s Session) error {
	h.render(w, "scan", http.StatusOK, h.newData(s, "Scan or enter a code", "scan"), scanCSP)
	return nil
}

func (h *handler) pageManual(w http.ResponseWriter, r *http.Request, s Session) error {
	parsed, err := qr.ParseManual(h.svc.Origin(), r.PostForm.Get("code"))
	if err != nil {
		d := h.newData(s, "Scan or enter a code", "scan")
		d.Err = "That is not a Circula QR link or code for this site. Check it and try again."
		h.render(w, "scan", http.StatusBadRequest, d, scanCSP)
		return nil
	}
	target := qr.LinkVersionPrefix + parsed.Code
	if parsed.Kind == qr.KindAlias {
		target = qr.LinkVersionPrefix + "a/" + parsed.Code
	}
	secureHeaders(w)
	http.Redirect(w, r, target, http.StatusSeeOther) // built from the validated canonical code only
	return nil
}

func (h *handler) confirmPage(w http.ResponseWriter, s Session, p qr.Preview, code string) {
	d := h.newData(s, "Confirm before sending", "scan")
	d.P, d.Code = p, code
	d.Network = p.NetworkName
	h.render(w, "confirm", http.StatusOK, d, baseCSP)
}

func (h *handler) pageRequest(w http.ResponseWriter, r *http.Request, s Session) error {
	code := r.PathValue("code")
	p, err := h.svc.PreviewRequest(r.Context(), s.Principal, code)
	if err != nil {
		return err
	}
	if p.Own || p.Claimed {
		// never offer a second confirmation: send the member to the status view
		if p.RequestID != "" && (p.Own || p.ClaimedByYou) {
			http.Redirect(w, r, "/qr/requests/"+url.PathEscape(p.RequestID), http.StatusSeeOther)
			return nil
		}
		return qr.ErrClaimed
	}
	h.confirmPage(w, s, p, code)
	return nil
}

func (h *handler) pageAlias(w http.ResponseWriter, r *http.Request, s Session) error {
	code := r.PathValue("code")
	p, err := h.svc.ResolveAlias(r.Context(), s.Principal, code)
	if err != nil {
		return err
	}
	if p.Own {
		return fmt.Errorf("%w: this is your own QR code", qr.ErrInvalid)
	}
	h.confirmPage(w, s, p, code)
	return nil
}

func parseAmount(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 15 || strings.Trim(v, "0123456789") != "" {
		return 0, fmt.Errorf("%w: amount must be a whole number of minor units", qr.ErrInvalid)
	}
	return strconv.ParseInt(v, 10, 64)
}

func (h *handler) pageConfirm(w http.ResponseWriter, r *http.Request, s Session) error {
	code, kind, key := r.PostForm.Get("code"), r.PostForm.Get("kind"), r.PostForm.Get("idempotency_key")
	var res qr.ClaimResult
	var err error
	switch kind {
	case string(qr.KindRequest):
		res, err = h.svc.ClaimRequest(r.Context(), s.Principal, code, key)
	case string(qr.KindAlias):
		var amt int64
		if amt, err = parseAmount(r.PostForm.Get("amount")); err != nil {
			// re-show the confirmation with the problem next to the form
			p, perr := h.svc.ResolveAlias(r.Context(), s.Principal, code)
			if perr != nil {
				return perr
			}
			d := h.newData(s, "Confirm before sending", "scan")
			d.P, d.Code, d.Network = p, code, p.NetworkName
			d.Err = "Enter the amount as a whole number of minor units, digits only."
			h.render(w, "confirm", http.StatusBadRequest, d, baseCSP)
			return nil
		}
		res, err = h.svc.ProposeToAlias(r.Context(), s.Principal, code, amt, r.PostForm.Get("note"), key)
	default:
		return fmt.Errorf("%w: unknown code kind", qr.ErrInvalid)
	}
	if err != nil {
		return err
	}
	d := h.newData(s, "Proposal sent", "scan")
	d.TxID, d.Replayed = res.TransactionID, res.Replayed
	d.State = string(res.State)
	switch res.State {
	case qr.TxCompleted:
		d.StateText = "Completed: both members agreed and the trade was recorded."
	case qr.TxClosed:
		d.StateText = "Closed: this proposal ended without any payment."
	default:
		d.StateText = "Pending: waiting for the other member to accept. Not paid yet."
	}
	h.render(w, "result", http.StatusOK, d, baseCSP)
	return nil
}

func (h *handler) pageRotate(w http.ResponseWriter, r *http.Request, s Session) error {
	if _, err := h.svc.RotateAlias(r.Context(), s.Principal, r.PostForm.Get("idempotency_key")); err != nil {
		return err
	}
	secureHeaders(w)
	http.Redirect(w, r, "/qr?done=rotated", http.StatusSeeOther)
	return nil
}

func (h *handler) pageCreate(w http.ResponseWriter, r *http.Request, s Session) error {
	amt, err := parseAmount(r.PostForm.Get("amount"))
	if err != nil {
		return err
	}
	ttl, err := strconv.ParseInt(r.PostForm.Get("ttl"), 10, 64)
	if err != nil {
		return fmt.Errorf("%w: choose how long the request stays valid", qr.ErrInvalid)
	}
	c, err := h.svc.CreateRequest(r.Context(), s.Principal, r.PostForm.Get("idempotency_key"),
		qr.CreateRequestInput{AmountMinor: amt, Note: r.PostForm.Get("note"), TTL: durationSeconds(ttl)})
	if err != nil {
		return err
	}
	if c.Replayed { // double submit: the one-time code was shown already and is not recoverable
		secureHeaders(w)
		http.Redirect(w, r, "/qr/requests/"+c.View.ID, http.StatusSeeOther)
		return nil
	}
	png, err := h.svc.RenderURL(c.URL)
	if err != nil {
		return err
	}
	d := h.newData(s, "One-time request created", "home")
	d.V, d.URL, d.Compact, d.PNG, d.PNGSize = c.View, c.URL, c.Compact, dataURI(png), 256
	h.render(w, "created", http.StatusCreated, d, baseCSP)
	return nil
}

func (h *handler) pageStatus(w http.ResponseWriter, r *http.Request, s Session) error {
	v, err := h.svc.GetRequest(r.Context(), s.Principal, r.PathValue("id"))
	if err != nil {
		return err
	}
	d := h.newData(s, "Request status", "home")
	d.V = v
	if r.URL.Query().Get("done") == "revoked" {
		d.Notice = "The request was revoked."
	}
	h.render(w, "status", http.StatusOK, d, baseCSP)
	return nil
}

func (h *handler) pageRevoke(w http.ResponseWriter, r *http.Request, s Session) error {
	if _, err := h.svc.RevokeRequest(r.Context(), s.Principal, r.PathValue("id")); err != nil {
		return err
	}
	secureHeaders(w)
	http.Redirect(w, r, "/qr/requests/"+r.PathValue("id")+"?done=revoked", http.StatusSeeOther)
	return nil
}
