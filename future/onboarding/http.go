package onboarding

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ErrUnauthenticated is what an Authenticate function returns when there is no
// valid current session. Any other error is treated as forbidden.
var ErrUnauthenticated = errors.New("onboarding: not authenticated")

// CSRF is supplied by the host. The host's session layer is the only place
// that can bind a token to the real session.
type CSRF interface {
	Token(a Actor, r *http.Request) string
	Verify(a Actor, r *http.Request, token string) bool
}

// HMACCSRF is a ready-made CSRF implementation: tokens are an HMAC of the
// actor and a host-supplied session identifier. Session must return a value
// unique to the logged-in session (not a guessable constant).
type HMACCSRF struct {
	Key     []byte // at least 32 random bytes
	Session func(*http.Request) string
}

func (c HMACCSRF) mac(a Actor, r *http.Request) []byte {
	m := hmac.New(sha256.New, c.Key)
	fmt.Fprintf(m, "onb-csrf-v1|%s|%s|%s", a.NetworkID, a.MemberID, c.Session(r))
	return m.Sum(nil)
}

// Token implements CSRF.
func (c HMACCSRF) Token(a Actor, r *http.Request) string {
	if len(c.Key) < 32 || c.Session == nil || c.Session(r) == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(c.mac(a, r))
}

// Verify implements CSRF.
func (c HMACCSRF) Verify(a Actor, r *http.Request, token string) bool {
	want := c.Token(a, r)
	return want != "" && hmac.Equal([]byte(want), []byte(token))
}

// Facts are optional, trusted, read-only snapshots of authoritative account
// state, shown beside the checklist. They never modify tutorial progress and
// tutorial progress is never used to derive them.
type Facts struct{ TermsAccepted, HasListing, HasProposal bool }

// FactsFunc returns Facts for the actor, or an error (facts are then omitted).
type FactsFunc func(ctx context.Context, a Actor) (Facts, error)

// HandlerConfig configures Handler. Authenticate and CSRF are required and must
// perform current-state authorization on every request (membership active,
// session valid); this package never caches an Actor.
type HandlerConfig struct {
	Tracker      *Tracker
	Authenticate func(*http.Request) (Actor, error)
	CSRF         CSRF
	Facts        FactsFunc
	// BasePath is where the host mounts the handler, e.g. "/onboarding".
	BasePath string
	// StepLinks are trusted links for each checklist step (e.g. the real
	// "add a listing" page). Each must satisfy Links.
	StepLinks map[Step]string
	// PracticeURL is the trusted link to the separate-origin practice server.
	PracticeURL string
	// Links is the trusted link policy used for StepLinks and PracticeURL.
	Links LinkPolicy
	// MaxBodyBytes bounds request bodies; default and maximum 64 KiB.
	MaxBodyBytes int64
}

// Handler is the no-JavaScript HTTP adapter. Mount it behind the host's normal
// router; it registers nothing by itself.
type Handler struct {
	cfg HandlerConfig
}

const maxBody = 64 << 10

// NewHandler validates cfg.
func NewHandler(cfg HandlerConfig) (*Handler, error) {
	if cfg.Tracker == nil || cfg.Authenticate == nil || cfg.CSRF == nil {
		return nil, errors.New("onboarding: Tracker, Authenticate and CSRF are required")
	}
	b := cfg.BasePath
	if b == "" || !strings.HasPrefix(b, "/") || strings.HasSuffix(b, "/") || strings.ContainsAny(b, "?#\\ ") || strings.Contains(b, "//") {
		return nil, fmt.Errorf("onboarding: bad base path %q", b)
	}
	if cfg.MaxBodyBytes <= 0 || cfg.MaxBodyBytes > maxBody {
		cfg.MaxBodyBytes = maxBody
	}
	if err := cfg.Links.validate(); err != nil {
		return nil, err
	}
	for s, l := range cfg.StepLinks {
		if s >= StepCount {
			return nil, ErrInvalidStep
		}
		if err := cfg.Links.Check(l); err != nil {
			return nil, fmt.Errorf("onboarding: step link %s: %w", s.Key(), err)
		}
	}
	if err := cfg.Links.Check(cfg.PracticeURL); err != nil {
		return nil, fmt.Errorf("onboarding: practice link: %w", err)
	}
	return &Handler{cfg}, nil
}

func setSecureHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Cross-Origin-Opener-Policy", "same-origin")
}

func (h *Handler) css() string { return h.cfg.BasePath + "/assets/onboarding.css" }

func (h *Handler) writeHTML(w http.ResponseWriter, status int, name string, data any) {
	body, err := render(name, data)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(body)
}

func (h *Handler) fail(w http.ResponseWriter, locale string, status int, key string) {
	setSecureHeaders(w)
	pg := newPage(locale, "", h.css(), false)
	pg.Title = pg.T["err_h"]
	v := errorView{page: pg, Heading: pg.T["err_h"], Message: pg.T[key], BackHref: h.cfg.BasePath + "/", BackText: pg.T["back_checklist"]}
	h.writeHTML(w, status, "error", v)
}

// statusFor maps package errors to an HTTP status and message key.
func statusFor(err error) (int, string) {
	var mb *http.MaxBytesError
	switch {
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized, "err_unauth"
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrInvalidIdentifier):
		return http.StatusForbidden, "err_forbidden"
	case errors.Is(err, ErrStaleCurriculum):
		return http.StatusConflict, "err_stale"
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, "err_conflict"
	case errors.Is(err, ErrInvalidStep), errors.Is(err, ErrInvalidAction):
		return http.StatusBadRequest, "err_bad"
	case errors.As(err, &mb):
		return http.StatusRequestEntityTooLarge, "err_toolarge"
	}
	return http.StatusInternalServerError, "err_server"
}

func (h *Handler) auth(w http.ResponseWriter, r *http.Request) (Actor, bool) {
	a, err := h.cfg.Authenticate(r)
	if err != nil {
		st := http.StatusUnauthorized
		key := "err_unauth"
		if !errors.Is(err, ErrUnauthenticated) {
			st, key = http.StatusForbidden, "err_forbidden"
		}
		h.fail(w, "", st, key)
		return Actor{}, false
	}
	if _, err := a.key(); err != nil { // suspended or malformed actor
		h.fail(w, a.Locale, http.StatusForbidden, "err_forbidden")
		return Actor{}, false
	}
	return a, true
}

// sameOrigin rejects browser cross-site POSTs even before the token check.
func sameOrigin(r *http.Request) bool {
	if o := r.Header.Get("Origin"); o != "" {
		u, err := url.Parse(o)
		return err == nil && u.Host != "" && strings.EqualFold(u.Host, r.Host)
	}
	return r.Header.Get("Sec-Fetch-Site") != "cross-site"
}

func (h *Handler) readForm(w http.ResponseWriter, r *http.Request, a Actor) (url.Values, bool) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/x-www-form-urlencoded" {
		h.fail(w, a.Locale, http.StatusUnsupportedMediaType, "err_bad")
		return nil, false
	}
	if !sameOrigin(r) {
		h.fail(w, a.Locale, http.StatusForbidden, "err_csrf")
		return nil, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes)
	if err := r.ParseForm(); err != nil {
		st, key := statusFor(err)
		if st == http.StatusInternalServerError {
			st, key = http.StatusBadRequest, "err_bad"
		}
		h.fail(w, a.Locale, st, key)
		return nil, false
	}
	// Only the body is consulted: query-string values can never act as form input.
	if !h.cfg.CSRF.Verify(a, r, r.PostForm.Get("csrf")) {
		h.fail(w, a.Locale, http.StatusForbidden, "err_csrf")
		return nil, false
	}
	return r.PostForm, true
}

// ServeHTTP routes the handler's few fixed paths.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	base := h.cfg.BasePath
	rest, ok := strings.CutPrefix(r.URL.Path, base)
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch rest {
	case "", "/":
		h.method(w, r, http.MethodGet, h.checklist)
	case "/progress":
		h.method(w, r, http.MethodPost, h.progress)
	case "/admin":
		h.method(w, r, http.MethodGet, h.adminPage)
	case "/admin/copy":
		h.method(w, r, http.MethodPost, h.adminCopy)
	case "/admin/metrics":
		h.method(w, r, http.MethodPost, h.adminMetrics)
	case "/assets/onboarding.css":
		h.serveCSS(w, r)
	default:
		setSecureHeaders(w)
		h.fail(w, "", http.StatusNotFound, "err_notfound")
	}
}

func (h *Handler) method(w http.ResponseWriter, r *http.Request, want string, fn http.HandlerFunc) {
	if r.Method == want || (want == http.MethodGet && r.Method == http.MethodHead) {
		fn(w, r)
		return
	}
	w.Header().Set("Allow", want)
	h.fail(w, "", http.StatusMethodNotAllowed, "err_method")
}

func (h *Handler) serveCSS(w http.ResponseWriter, r *http.Request) {
	templates() // ensures cssETag
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		h.fail(w, "", http.StatusMethodNotAllowed, "err_method")
		return
	}
	serveCSSBytes(w, r)
}

func serveCSSBytes(w http.ResponseWriter, r *http.Request) {
	templates()
	hd := w.Header()
	hd.Set("Content-Type", "text/css; charset=utf-8")
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cache-Control", "public, max-age=3600")
	hd.Set("ETag", cssETag)
	if r.Header.Get("If-None-Match") == cssETag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if r.Method != http.MethodHead {
		w.Write(cssBytes)
	}
}

func (h *Handler) checklist(w http.ResponseWriter, r *http.Request) {
	setSecureHeaders(w)
	a, ok := h.auth(w, r)
	if !ok {
		return
	}
	p, err := h.cfg.Tracker.Progress(r.Context(), a)
	if err != nil {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	cp, err := h.cfg.Tracker.Copy(r.Context(), a)
	if err != nil {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	var facts Facts
	if h.cfg.Facts != nil {
		if f, err := h.cfg.Facts(r.Context(), a); err == nil {
			facts = f
		}
	}
	pg := newPage(a.Locale, "", h.css(), false)
	pg.Title = pg.T["cl_title"]
	pg.CSRF = h.cfg.CSRF.Token(a, r)
	v := checklistView{page: pg, Intro: cp.Intro, GovernanceURL: cp.GovernanceURL, TermsURL: cp.TermsURL,
		Dismissed: p.Dismissed, Share: p.Share, ActionURL: h.cfg.BasePath + "/progress",
		PracticeURL: h.cfg.PracticeURL, Curriculum: p.Version}
	if a.Admin {
		v.AdminURL = h.cfg.BasePath + "/admin"
	}
	done := 0
	next, hasNext := p.Next()
	confirmed := map[Step]bool{Terms: facts.TermsAccepted, Listing: facts.HasListing, Trade: facts.HasProposal}
	for s := Terms; s < StepCount; s++ {
		sv := stepView{Key: s.Key(), Name: stepName(pg.T, s), Hint: stepHint(pg.T, s),
			HintID: fmt.Sprintf("onb-task-list-%d-hint", s+1), StatusID: fmt.Sprintf("onb-task-list-%d-status", s+1),
			Href: h.cfg.StepLinks[s], Completed: p.IsCompleted(s), Skipped: p.IsSkipped(s)}
		switch {
		case sv.Completed:
			sv.Status = pg.T["status_done"]
			done++
		case confirmed[s]:
			sv.Status, sv.Completed = pg.T["status_confirmed"], true
			done++
		case sv.Skipped:
			sv.Status, sv.Muted = pg.T["status_skipped"], true
		case hasNext && next == s:
			sv.Status, sv.Next = pg.T["status_next"], true
		default:
			sv.Status, sv.Muted = pg.T["status_todo"], true
		}
		v.Steps = append(v.Steps, sv)
	}
	v.ProgressText = fmt.Sprintf(pg.T["cl_progress"], done, int(StepCount))
	if done == int(StepCount) {
		v.ProgressText += " " + pg.T["cl_all_done"]
	}
	h.writeHTML(w, http.StatusOK, "checklist", v)
}

func parseAction(f url.Values) (Action, int, error) {
	cur, err := strconv.Atoi(f.Get("curriculum"))
	if err != nil {
		return Action{}, 0, ErrInvalidAction
	}
	var act Action
	switch f.Get("action") {
	case "complete":
		act.Kind = ActComplete
	case "skip":
		act.Kind = ActSkip
	case "resume":
		act.Kind = ActResume
	case "dismiss":
		act.Kind = ActDismiss
	case "restore":
		act.Kind = ActRestore
	case "share":
		act.Kind = ActShare
	case "unshare":
		act.Kind = ActUnshare
	default:
		return Action{}, 0, ErrInvalidAction
	}
	if act.needsStep() {
		s, err := ParseStep(f.Get("step"))
		if err != nil {
			return Action{}, 0, err
		}
		act.Step = s
	}
	return act, cur, nil
}

func (h *Handler) progress(w http.ResponseWriter, r *http.Request) {
	setSecureHeaders(w)
	a, ok := h.auth(w, r)
	if !ok {
		return
	}
	f, ok := h.readForm(w, r, a)
	if !ok {
		return
	}
	act, cur, err := parseAction(f)
	if err == nil {
		_, err = h.cfg.Tracker.Apply(r.Context(), a, cur, act)
	}
	if err != nil {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	anchor := "#progress-summary"
	if act.needsStep() {
		anchor = "#step-" + act.Step.Key()
	} else if act.Kind == ActShare || act.Kind == ActUnshare {
		anchor = "#share-heading"
	}
	http.Redirect(w, r, h.cfg.BasePath+"/"+anchor, http.StatusSeeOther)
}

func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (Actor, bool) {
	a, ok := h.auth(w, r)
	if !ok {
		return a, false
	}
	if !a.Admin {
		h.fail(w, a.Locale, http.StatusForbidden, "err_forbidden")
		return a, false
	}
	return a, true
}

func (h *Handler) adminView(r *http.Request, a Actor) (adminView, error) {
	cp, err := h.cfg.Tracker.Copy(r.Context(), a)
	if err != nil {
		return adminView{}, err
	}
	on, err := h.cfg.Tracker.MetricsEnabled(r.Context(), a)
	if err != nil {
		return adminView{}, err
	}
	pg := newPage(a.Locale, "", h.css(), false)
	pg.Title = pg.T["ad_title"]
	pg.CSRF = h.cfg.CSRF.Token(a, r)
	v := adminView{page: pg, Intro: cp.Intro, GovernanceURL: cp.GovernanceURL, TermsURL: cp.TermsURL, Revision: cp.Revision,
		MetricsOn: on, MetricsHelp: fmt.Sprintf(pg.T["ad_metrics_help"], MinCohort),
		CopyURL: h.cfg.BasePath + "/admin/copy", MetricsURL: h.cfg.BasePath + "/admin/metrics", BackURL: h.cfg.BasePath + "/"}
	if on {
		agg, err := h.cfg.Tracker.Aggregate(r.Context(), a)
		if err != nil {
			return adminView{}, err
		}
		for _, s := range agg {
			val := strconv.FormatInt(s.Completed, 10)
			if s.Suppressed {
				val = fmt.Sprintf(pg.T["ad_counts_hidden"], MinCohort)
			}
			v.Counts = append(v.Counts, countRow{stepName(pg.T, s.Step), val})
		}
	}
	return v, nil
}

func (h *Handler) adminPage(w http.ResponseWriter, r *http.Request) {
	setSecureHeaders(w)
	a, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	v, err := h.adminView(r, a)
	if err != nil {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	v.Saved = r.URL.Query().Get("saved") == "1"
	h.writeHTML(w, http.StatusOK, "admin", v)
}

func (h *Handler) adminCopy(w http.ResponseWriter, r *http.Request) {
	setSecureHeaders(w)
	a, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	f, ok := h.readForm(w, r, a)
	if !ok {
		return
	}
	rev, err := strconv.ParseInt(f.Get("revision"), 10, 64)
	if err != nil || rev < 0 {
		h.fail(w, a.Locale, http.StatusBadRequest, "err_bad")
		return
	}
	c := Copy{Intro: f.Get("intro"), GovernanceURL: strings.TrimSpace(f.Get("governance")), TermsURL: strings.TrimSpace(f.Get("terms"))}
	_, err = h.cfg.Tracker.SaveCopy(r.Context(), a, c, rev)
	if err == nil {
		http.Redirect(w, r, h.cfg.BasePath+"/admin?saved=1", http.StatusSeeOther)
		return
	}
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrConflict) || errors.Is(err, ErrInvalidIdentifier) {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	// Validation problem: redisplay the form with what was typed and an error summary.
	v, verr := h.adminView(r, a)
	if verr != nil {
		st, key := statusFor(verr)
		h.fail(w, a.Locale, st, key)
		return
	}
	v.Intro, v.GovernanceURL, v.TermsURL = c.Intro, c.GovernanceURL, c.TermsURL
	if ValidateText(c.Intro, MaxIntroRunes) != nil {
		v.IntroErr = true
		v.Errors = append(v.Errors, fieldErr{"intro", v.T["err_text_long"]})
	}
	if h.cfg.Tracker.links.Check(c.GovernanceURL) != nil {
		v.GovErr = true
		v.Errors = append(v.Errors, fieldErr{"governance", v.T["err_link"]})
	}
	if h.cfg.Tracker.links.Check(c.TermsURL) != nil {
		v.TermsErr = true
		v.Errors = append(v.Errors, fieldErr{"terms", v.T["err_link"]})
	}
	if len(v.Errors) == 0 {
		v.Errors = append(v.Errors, fieldErr{"", v.T["err_bad"]})
	}
	h.writeHTML(w, http.StatusUnprocessableEntity, "admin", v)
}

func (h *Handler) adminMetrics(w http.ResponseWriter, r *http.Request) {
	setSecureHeaders(w)
	a, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}
	f, ok := h.readForm(w, r, a)
	if !ok {
		return
	}
	var on bool
	switch f.Get("enabled") {
	case "1":
		on = true
	case "0":
	default:
		h.fail(w, a.Locale, http.StatusBadRequest, "err_bad")
		return
	}
	if err := h.cfg.Tracker.SetMetrics(r.Context(), a, on); err != nil {
		st, key := statusFor(err)
		h.fail(w, a.Locale, st, key)
		return
	}
	http.Redirect(w, r, h.cfg.BasePath+"/admin#metrics-heading", http.StatusSeeOther)
}

// ---- JSON API (optional GET/PATCH /api/v1/me/onboarding-progress contract) ----

type apiStep struct {
	Step      string `json:"step"`
	Completed bool   `json:"completed"`
	Skipped   bool   `json:"skipped"`
}

type apiProgress struct {
	Curriculum     int       `json:"curriculum"`
	Revision       int64     `json:"revision"`
	Steps          []apiStep `json:"steps"`
	Next           string    `json:"next,omitempty"`
	Dismissed      bool      `json:"dismissed"`
	ShareAggregate bool      `json:"share_aggregate"`
	// Authoritative is always false: tutorial progress is not evidence of
	// accepted terms, listings or trades.
	Authoritative bool `json:"authoritative"`
}

func toAPI(p Progress) apiProgress {
	out := apiProgress{Curriculum: p.Version, Revision: p.Revision, Dismissed: p.Dismissed, ShareAggregate: p.Share}
	for s := Terms; s < StepCount; s++ {
		out.Steps = append(out.Steps, apiStep{s.Key(), p.IsCompleted(s), p.IsSkipped(s)})
	}
	if s, ok := p.Next(); ok {
		out.Next = s.Key()
	}
	return out
}

func problem(w http.ResponseWriter, status int, detail string) {
	setSecureHeaders(w)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": http.StatusText(status), "status": status, "detail": detail})
}

// APIHandler serves the single resource /me/onboarding-progress as JSON. The
// host mounts it wherever its API contract places that resource.
//
//	GET   -> current progress of the authenticated member
//	PATCH -> {"curriculum":1,"action":"complete|skip|resume|dismiss|restore|share|unshare","step":"terms"}
//
// PATCH needs the X-CSRF-Token header (cookie-authenticated callers) and is
// idempotent. Errors are RFC 9457 problem documents.
func (h *Handler) APIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a, err := h.cfg.Authenticate(r)
		if err != nil {
			if errors.Is(err, ErrUnauthenticated) {
				problem(w, http.StatusUnauthorized, "authentication required")
			} else {
				problem(w, http.StatusForbidden, "not permitted")
			}
			return
		}
		if _, err := a.key(); err != nil {
			problem(w, http.StatusForbidden, "not permitted")
			return
		}
		switch r.Method {
		case http.MethodGet:
			p, err := h.cfg.Tracker.Progress(r.Context(), a)
			if err != nil {
				st, _ := statusFor(err)
				problem(w, st, "could not load progress")
				return
			}
			setSecureHeaders(w)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(toAPI(p))
		case http.MethodPatch:
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				problem(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
			if !sameOrigin(r) || !h.cfg.CSRF.Verify(a, r, r.Header.Get("X-CSRF-Token")) {
				problem(w, http.StatusForbidden, "CSRF check failed")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxBodyBytes)
			var in struct {
				Curriculum int    `json:"curriculum"`
				Action     string `json:"action"`
				Step       string `json:"step"`
			}
			dec := json.NewDecoder(r.Body)
			dec.DisallowUnknownFields()
			if err := dec.Decode(&in); err != nil {
				var mb *http.MaxBytesError
				if errors.As(err, &mb) {
					problem(w, http.StatusRequestEntityTooLarge, "body too large")
				} else {
					problem(w, http.StatusBadRequest, "invalid JSON body")
				}
				return
			}
			if _, err := dec.Token(); !errors.Is(err, io.EOF) {
				problem(w, http.StatusBadRequest, "unexpected trailing data")
				return
			}
			f := url.Values{"curriculum": {strconv.Itoa(in.Curriculum)}, "action": {in.Action}, "step": {in.Step}}
			act, cur, err := parseAction(f)
			var p Progress
			if err == nil {
				p, err = h.cfg.Tracker.Apply(r.Context(), a, cur, act)
			}
			if err != nil {
				st, _ := statusFor(err)
				detail := "request not applied"
				if st == http.StatusBadRequest {
					detail = "invalid action or step"
				}
				problem(w, st, detail)
				return
			}
			setSecureHeaders(w)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(toAPI(p))
		default:
			w.Header().Set("Allow", "GET, PATCH")
			problem(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
}
