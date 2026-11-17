package onboarding

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PracticeConfig configures the isolated practice origin. There is deliberately
// no field that can carry a database handle, a service object, a production
// session or a DSN (a reflection test enforces this).
type PracticeConfig struct {
	// AllowedHosts are the Host header values (host or host:port, lowercase)
	// this server answers to. It must be a distinct origin from production.
	AllowedHosts []string
	// ProductionHosts are the real application's hosts. Configuration is
	// refused if any equals an AllowedHost, so practice can never share an
	// origin (and therefore cookies/storage) with the real app.
	ProductionHosts []string
	// ProductionCookieNames are the real session cookie names. Refused if the
	// practice cookie name collides. Such cookies are never read.
	ProductionCookieNames []string
	CookieName            string // default "practice_session"
	Secure                bool   // set true behind HTTPS: adds Secure and the __Host- prefix
	// ExitURL is a trusted link back to the real app (path or https), checked
	// against Links.
	ExitURL string
	Links   LinkPolicy
	// BasePath is an optional path prefix ("" for origin root).
	BasePath    string
	MaxSessions int           // default 500
	SessionTTL  time.Duration // default 1h
	Now         func() time.Time
	MaxBody     int64 // default and maximum 16 KiB
}

// PracticeServer serves the practice network. It holds only in-memory worlds.
type PracticeServer struct {
	cfg      PracticeConfig
	cookie   string
	hosts    map[string]bool
	mu       sync.Mutex
	sessions map[string]*practiceSession
}

type practiceSession struct {
	world *World
	csrf  string
	last  time.Time
}

// NewPracticeServer validates cfg.
func NewPracticeServer(cfg PracticeConfig) (*PracticeServer, error) {
	if len(cfg.AllowedHosts) == 0 {
		return nil, errors.New("practice: AllowedHosts required")
	}
	hosts := map[string]bool{}
	for _, h := range cfg.AllowedHosts {
		if h == "" || h != strings.ToLower(h) || strings.ContainsAny(h, "/@ ?#\\") {
			return nil, fmt.Errorf("practice: bad allowed host %q", h)
		}
		hosts[h] = true
	}
	for _, p := range cfg.ProductionHosts {
		if hosts[strings.ToLower(p)] {
			return nil, fmt.Errorf("practice: origin %q is also a production host; practice needs its own origin", p)
		}
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "practice_session"
	}
	for _, c := range cfg.ProductionCookieNames {
		if strings.EqualFold(c, cfg.CookieName) || strings.EqualFold(c, "__Host-"+cfg.CookieName) {
			return nil, fmt.Errorf("practice: cookie name %q collides with a production cookie", c)
		}
	}
	name := cfg.CookieName
	if cfg.Secure {
		name = "__Host-" + name
	}
	if cfg.BasePath != "" && (!strings.HasPrefix(cfg.BasePath, "/") || strings.HasSuffix(cfg.BasePath, "/") || strings.ContainsAny(cfg.BasePath, "?#\\ ") || strings.Contains(cfg.BasePath, "//")) {
		return nil, fmt.Errorf("practice: bad base path %q", cfg.BasePath)
	}
	if err := cfg.Links.validate(); err != nil {
		return nil, err
	}
	if err := cfg.Links.Check(cfg.ExitURL); err != nil {
		return nil, fmt.Errorf("practice: exit link: %w", err)
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = 500
	}
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = time.Hour
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.MaxBody <= 0 || cfg.MaxBody > 16<<10 {
		cfg.MaxBody = 16 << 10
	}
	return &PracticeServer{cfg: cfg, cookie: name, hosts: hosts, sessions: map[string]*practiceSession{}}, nil
}

// CookieName is the exact cookie name this server uses.
func (p *PracticeServer) CookieName() string { return p.cookie }

// SessionCount is for tests and operators.
func (p *PracticeServer) SessionCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sessions)
}

func randToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // no entropy: refuse to continue rather than issue guessable tokens
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func nonce() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func validNonce(s string) bool {
	if len(s) < 16 || len(s) > 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'f') {
			return false
		}
	}
	return true
}

func (p *PracticeServer) sweepLocked(now time.Time) {
	for k, s := range p.sessions {
		if now.Sub(s.last) > p.cfg.SessionTTL {
			delete(p.sessions, k)
		}
	}
}

func (p *PracticeServer) lookup(r *http.Request) (string, *practiceSession) {
	c, err := r.Cookie(p.cookie)
	if err != nil || len(c.Value) < 40 || len(c.Value) > 64 {
		return "", nil
	}
	now := p.cfg.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	s := p.sessions[c.Value]
	if s == nil || now.Sub(s.last) > p.cfg.SessionTTL {
		delete(p.sessions, c.Value)
		return "", nil
	}
	s.last = now
	return c.Value, s
}

var errBusy = errors.New("practice: server busy")

func (p *PracticeServer) create() (string, *practiceSession, error) {
	now := p.cfg.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sweepLocked(now)
	if len(p.sessions) >= p.cfg.MaxSessions {
		return "", nil, errBusy
	}
	id := randToken(32)
	s := &practiceSession{world: NewWorld(), csrf: randToken(24), last: now}
	p.sessions[id] = s
	return id, s, nil
}

func (p *PracticeServer) drop(id string) {
	p.mu.Lock()
	delete(p.sessions, id)
	p.mu.Unlock()
}

func (p *PracticeServer) setCookie(w http.ResponseWriter, id string) {
	http.SetCookie(w, &http.Cookie{Name: p.cookie, Value: id, Path: "/", HttpOnly: true, Secure: p.cfg.Secure,
		SameSite: http.SameSiteStrictMode, MaxAge: int(p.cfg.SessionTTL / time.Second)})
}

func (p *PracticeServer) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: p.cookie, Value: "", Path: "/", HttpOnly: true, Secure: p.cfg.Secure,
		SameSite: http.SameSiteStrictMode, MaxAge: -1})
}

func (p *PracticeServer) css() string { return p.cfg.BasePath + "/assets/onboarding.css" }

func (p *PracticeServer) locale(r *http.Request) string {
	al := r.Header.Get("Accept-Language")
	if len(al) > 64 {
		al = al[:64]
	}
	if i := strings.IndexAny(al, ",;"); i >= 0 {
		al = al[:i]
	}
	return pickLocale(strings.TrimSpace(al))
}

func (p *PracticeServer) writeHTML(w http.ResponseWriter, status int, name string, data any) {
	body, err := render(name, data)
	if err != nil {
		http.Error(w, PracticeMarker+": internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write(body)
}

// fail renders an error page that still carries the practice banner.
func (p *PracticeServer) fail(w http.ResponseWriter, r *http.Request, status int, key string) {
	setPracticeHeaders(w)
	pg := newPage(p.locale(r), "", p.css(), true)
	pg.Title = pg.T["err_h"]
	p.writeHTML(w, status, "error", errorView{page: pg, Heading: pg.T["err_h"], Message: pg.T[key],
		BackHref: p.cfg.BasePath + "/", BackText: pg.T["pr_home"]})
}

func setPracticeHeaders(w http.ResponseWriter) {
	setSecureHeaders(w)
	w.Header().Set("X-Practice", "synthetic")
}

// ServeHTTP implements http.Handler.
func (p *PracticeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.hosts[strings.ToLower(r.Host)] {
		setPracticeHeaders(w)
		http.Error(w, PracticeMarker+": wrong host", http.StatusMisdirectedRequest)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, p.cfg.BasePath)
	if !ok {
		p.fail(w, r, http.StatusNotFound, "err_notfound")
		return
	}
	if rest == "/assets/onboarding.css" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			p.fail(w, r, http.StatusMethodNotAllowed, "err_method")
			return
		}
		serveCSSBytes(w, r)
		return
	}
	switch rest {
	case "", "/":
		p.route(w, r, "GET", p.home)
	case "/start":
		p.route(w, r, "POST", p.start)
	case "/reset":
		p.route(w, r, "POST", p.reset)
	case "/exit":
		p.route(w, r, "POST", p.exit)
	case "/export.json":
		p.route(w, r, "GET", p.exportJSON)
	case "/export.csv":
		p.route(w, r, "GET", p.exportCSV)
	case "/step/terms", "/step/credit", "/step/listing", "/step/partner", "/step/trade":
		step, _ := ParseStep(strings.TrimPrefix(rest, "/step/"))
		if r.Method == http.MethodPost {
			p.route(w, r, "POST", func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
				p.stepPost(w, r, s, step)
			})
		} else {
			p.route(w, r, "GET", func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
				p.stepGet(w, r, s, step, nil)
			})
		}
	case "/step/trade/propose":
		p.route(w, r, "POST", func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) { p.propose(w, r, s) })
	case "/proposal/respond":
		p.route(w, r, "POST", func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) { p.respond(w, r, s, true) })
	case "/proposal/cancel":
		p.route(w, r, "POST", func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) { p.respond(w, r, s, false) })
	default:
		p.fail(w, r, http.StatusNotFound, "err_notfound")
	}
}

type practiceHandler func(w http.ResponseWriter, r *http.Request, id string, s *practiceSession)

// route enforces method, session and (for POST) CSRF/origin/body rules. The
// start action is the only one that does not require an existing session.
func (p *PracticeServer) route(w http.ResponseWriter, r *http.Request, method string, fn practiceHandler) {
	if r.Method != method && !(method == "GET" && r.Method == http.MethodHead) {
		w.Header().Set("Allow", method)
		p.fail(w, r, http.StatusMethodNotAllowed, "err_method")
		return
	}
	setPracticeHeaders(w)
	id, s := p.lookup(r)
	if method == "POST" {
		mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mt != "application/x-www-form-urlencoded" {
			p.fail(w, r, http.StatusUnsupportedMediaType, "err_bad")
			return
		}
		if !sameOrigin(r) {
			p.fail(w, r, http.StatusForbidden, "err_csrf")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, p.cfg.MaxBody)
		if err := r.ParseForm(); err != nil {
			var mb *http.MaxBytesError
			if errors.As(err, &mb) {
				p.fail(w, r, http.StatusRequestEntityTooLarge, "err_toolarge")
			} else {
				p.fail(w, r, http.StatusBadRequest, "err_bad")
			}
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/start") {
			if s == nil || subtle.ConstantTimeCompare([]byte(s.csrf), []byte(r.PostForm.Get("csrf"))) != 1 {
				p.fail(w, r, http.StatusForbidden, "err_csrf")
				return
			}
		}
	} else if s == nil && !(r.URL.Path == p.cfg.BasePath+"/" || r.URL.Path == p.cfg.BasePath) {
		http.Redirect(w, r, p.cfg.BasePath+"/", http.StatusSeeOther)
		return
	}
	fn(w, r, id, s)
}

func (p *PracticeServer) basePage(r *http.Request, title string, s *practiceSession) page {
	pg := newPage(p.locale(r), "", p.css(), true)
	pg.Title = pg.T[title]
	if s != nil {
		pg.CSRF = s.csrf
	}
	return pg
}

type landingView struct {
	page
	StartURL, ExitURL string
}

type memberRow struct {
	Name                                string
	Balance, Limit, Reserved, Available int64
	Negative                            bool
}

type practiceHomeView struct {
	page
	Steps                                                    []stepView
	Members                                                  []memberRow
	Events                                                   []string
	AllDone                                                  bool
	JSONURL, CSVURL, ResetURL, ExitPostURL, ExitURL, HomeURL string
}

func memberRows(s Snapshot) []memberRow {
	var rows []memberRow
	for _, m := range s.Members {
		rows = append(rows, memberRow{m.Name, m.Balance, m.Limit, m.Reserved, m.Available(), m.Balance < 0})
	}
	return rows
}

func (p *PracticeServer) home(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	if s == nil {
		pg := p.basePage(r, "pr_title", nil)
		p.writeHTML(w, 200, "practice_landing", landingView{pg, p.cfg.BasePath + "/start", p.cfg.ExitURL})
		return
	}
	snap := s.world.Snapshot()
	pg := p.basePage(r, "pr_title", s)
	v := practiceHomeView{page: pg, Members: memberRows(snap), Events: snap.Events, AllDone: snap.Progress.Done(),
		JSONURL: p.cfg.BasePath + "/export.json", CSVURL: p.cfg.BasePath + "/export.csv", ResetURL: p.cfg.BasePath + "/reset",
		ExitPostURL: p.cfg.BasePath + "/exit", ExitURL: p.cfg.ExitURL, HomeURL: p.cfg.BasePath + "/"}
	next, hasNext := snap.Progress.Next()
	for st := Terms; st < StepCount; st++ {
		sv := stepView{Key: st.Key(), Name: pg.T["pr_s_"+st.Key()], Hint: pg.T["pr_s_"+st.Key()+"_hint"],
			HintID: fmt.Sprintf("onb-task-list-%d-hint", st+1), StatusID: fmt.Sprintf("onb-task-list-%d-status", st+1),
			Href: p.cfg.BasePath + "/step/" + st.Key()}
		switch {
		case snap.Progress.IsCompleted(st):
			sv.Status = pg.T["pr_done"]
		case hasNext && next == st:
			sv.Status, sv.Next = pg.T["pr_todo"], true
		default:
			sv.Status, sv.Muted = pg.T["pr_todo"], true
		}
		v.Steps = append(v.Steps, sv)
	}
	p.writeHTML(w, 200, "practice_home", v)
}

func (p *PracticeServer) start(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	if s != nil { // already practising: keep the world, never replace silently
		http.Redirect(w, r, p.cfg.BasePath+"/", http.StatusSeeOther)
		return
	}
	nid, _, err := p.create()
	if err != nil {
		p.fail(w, r, http.StatusServiceUnavailable, "err_limit")
		return
	}
	p.setCookie(w, nid)
	http.Redirect(w, r, p.cfg.BasePath+"/", http.StatusSeeOther)
}

func (p *PracticeServer) reset(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	s.world.Reset()
	http.Redirect(w, r, p.cfg.BasePath+"/", http.StatusSeeOther)
}

func (p *PracticeServer) exit(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	p.drop(id)
	p.clearCookie(w)
	dest := p.cfg.ExitURL
	if dest == "" {
		dest = p.cfg.BasePath + "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// ---- steps ----

type listingRow struct {
	Title, Kind, Owner string
	Amount             int64
}

type partnerRow struct {
	ID, Name string
	Checked  bool
}

type choiceRow struct {
	ID, Label string
	Checked   bool
}

type proposalRow struct {
	ID, State, Text, RespondURL, CancelURL, Cmd1, Cmd2, Cmd3 string
}

type previewView struct {
	YouBefore, YouAfter, PartnerBefore, PartnerAfter, Limit int64
	Negative                                                bool
	ReservedText, Listing                                   string
}

type stepPageView struct {
	page
	ActionURL, HomeURL, ProposeURL string
	Cmd                            string
	Members                        []memberRow
	Listings                       []listingRow
	Partners                       []partnerRow
	Choices                        []choiceRow
	Proposals                      []proposalRow
	Preview                        *previewView
	Kind, Title2, Amount           string
	TitleErr, AmountErr            string
}

func (p *PracticeServer) stepBase(r *http.Request, s *practiceSession, titleKey string, step string) (stepPageView, Snapshot) {
	snap := s.world.Snapshot()
	pg := p.basePage(r, titleKey, s)
	return stepPageView{page: pg, ActionURL: p.cfg.BasePath + "/step/" + step, HomeURL: p.cfg.BasePath + "/",
		ProposeURL: p.cfg.BasePath + "/step/trade/propose", Cmd: nonce(), Members: memberRows(snap)}, snap
}

func ownerName(snap Snapshot, id PracticeID) string {
	if m, ok := snap.MemberByID(id); ok {
		return m.Name
	}
	return "?"
}

func listingRows(t map[string]string, snap Snapshot) []listingRow {
	var rows []listingRow
	for _, l := range snap.Listings {
		kind := t["pr_kind_offer"]
		if l.Kind == KindRequest {
			kind = t["pr_kind_request"]
		}
		rows = append(rows, listingRow{l.Title, kind, ownerName(snap, l.Owner), l.Amount})
	}
	return rows
}

func (p *PracticeServer) stepGet(w http.ResponseWriter, r *http.Request, s *practiceSession, step Step, errs []fieldErr) {
	switch step {
	case Terms:
		v, _ := p.stepBase(r, s, "pr_terms_h", "terms")
		p.writeHTML(w, 200, "practice_terms", v)
	case Credit:
		v, _ := p.stepBase(r, s, "pr_credit_h", "credit")
		p.writeHTML(w, 200, "practice_credit", v)
	case Listing:
		v, snap := p.stepBase(r, s, "pr_listing_h", "listing")
		v.Listings, v.Kind = listingRows(v.T, snap), "offer"
		p.writeHTML(w, 200, "practice_listing", v)
	case Partner:
		v, snap := p.stepBase(r, s, "pr_partner_h", "partner")
		for _, m := range snap.Members {
			if !m.Learner {
				v.Partners = append(v.Partners, partnerRow{m.ID.String(), m.Name, m.ID == snap.Partner})
			}
		}
		v.Listings = listingRows(v.T, snap)
		v.Errors = errs
		p.writeHTML(w, 200, "practice_partner", v)
	case Trade:
		p.tradeGet(w, r, s, nil, http.StatusOK)
	}
}

func (p *PracticeServer) tradeGet(w http.ResponseWriter, r *http.Request, s *practiceSession, errs []fieldErr, status int) {
	v, snap := p.stepBase(r, s, "pr_trade_h", "trade")
	v.Errors = errs
	selected := r.URL.Query().Get("listing")
	if r.Method == http.MethodPost {
		selected = r.PostForm.Get("listing")
	}
	for _, l := range snap.Listings {
		if l.Owner == learnerID {
			continue
		}
		v.Choices = append(v.Choices, choiceRow{l.ID.String(),
			fmt.Sprintf("%s — %s — %s (%s)", l.Title, fmt.Sprintf(v.T["pr_listing_price"], l.Amount), fmt.Sprintf(v.T["pr_listing_by"], ownerName(snap, l.Owner)), kindLabel(v.T, l.Kind)),
			l.ID.String() == selected})
	}
	if lid, err := ParsePracticeID(selected); err == nil && selected != "" {
		if pv, err := s.world.PreviewProposal(lid); err == nil {
			l, _ := snap.ListingByID(lid)
			partnerBefore, partnerAfter := pv.PayeeBefore, pv.PayeeAfter
			youBefore, youAfter := pv.PayerBefore, pv.PayerAfter
			if l.Kind == KindRequest { // the learner is the payee
				youBefore, youAfter, partnerBefore, partnerAfter = pv.PayeeBefore, pv.PayeeAfter, pv.PayerBefore, pv.PayerAfter
			}
			v.Preview = &previewView{YouBefore: youBefore, YouAfter: youAfter, PartnerBefore: partnerBefore, PartnerAfter: partnerAfter,
				Limit: pv.Limit, Negative: youAfter < 0, Listing: lid.String(),
				ReservedText: fmt.Sprintf(v.T["pr_trade_reserved"], pv.Amount)}
		} else if len(v.Errors) == 0 {
			key := "err_state"
			if errors.Is(err, ErrPracticeCapacity) {
				key = "err_capacity"
			}
			v.Errors = []fieldErr{{"listing", v.T[key]}}
		}
	}
	for _, pr := range snap.Proposals {
		row := proposalRow{ID: pr.ID.String(), State: string(pr.State), RespondURL: p.cfg.BasePath + "/proposal/respond",
			CancelURL: p.cfg.BasePath + "/proposal/cancel", Cmd1: nonce(), Cmd2: nonce(), Cmd3: nonce()}
		if pr.State == ProposalPending {
			row.Text = fmt.Sprintf(v.T["pr_pending_p"], pr.Amount)
		}
		v.Proposals = append(v.Proposals, row)
	}
	p.writeHTML(w, status, "practice_trade", v)
}

func kindLabel(t map[string]string, k ListingKind) string {
	if k == KindRequest {
		return t["pr_kind_request"]
	}
	return t["pr_kind_offer"]
}

// worldErr maps engine errors to message keys.
func worldErr(err error) string {
	switch {
	case errors.Is(err, ErrPracticeCapacity):
		return "err_capacity"
	case errors.Is(err, ErrPracticeLimit):
		return "err_limit"
	case errors.Is(err, ErrPracticeAmount):
		return "err_amount"
	case errors.Is(err, ErrPracticeText):
		return "err_text"
	case errors.Is(err, ErrPracticeID), errors.Is(err, ErrPracticeNotFound):
		return "err_pick"
	}
	return "err_state"
}

func (p *PracticeServer) stepPost(w http.ResponseWriter, r *http.Request, s *practiceSession, step Step) {
	cmd := r.PostForm.Get("cmd")
	if !validNonce(cmd) {
		p.fail(w, r, http.StatusBadRequest, "err_bad")
		return
	}
	key := step.Key() + ":" + cmd
	redirect := func() { http.Redirect(w, r, p.cfg.BasePath+"/step/"+step.Key()+"#main", http.StatusSeeOther) }
	home := func() { http.Redirect(w, r, p.cfg.BasePath+"/#step-"+step.Key(), http.StatusSeeOther) }
	switch step {
	case Terms, Credit:
		var err error
		if step == Terms {
			err = s.world.AckTerms(key)
		} else {
			err = s.world.AckCredit(key)
		}
		if err != nil {
			p.fail(w, r, http.StatusConflict, worldErr(err))
			return
		}
		home()
	case Listing:
		amount, aerr := strconv.ParseInt(strings.TrimSpace(r.PostForm.Get("amount")), 10, 64)
		kind := ListingKind(r.PostForm.Get("kind"))
		title := r.PostForm.Get("title")
		v, snap := p.stepBase(r, s, "pr_listing_h", "listing")
		v.Listings, v.Kind, v.Title2, v.Amount = listingRows(v.T, snap), string(kind), title, r.PostForm.Get("amount")
		if aerr != nil {
			amount = 0
		}
		_, err := func() (PracticeID, error) {
			if kind != KindOffer && kind != KindRequest {
				return PracticeID{}, ErrPracticeText
			}
			return s.world.CreateListing(key, kind, title, amount)
		}()
		if err == nil {
			home()
			return
		}
		switch {
		case errors.Is(err, ErrPracticeAmount):
			v.AmountErr = v.T["err_amount"]
			v.Errors = append(v.Errors, fieldErr{"amount", v.AmountErr})
		case errors.Is(err, ErrPracticeText):
			v.TitleErr = v.T["err_text"]
			v.Errors = append(v.Errors, fieldErr{"title", v.TitleErr})
		default:
			v.Errors = append(v.Errors, fieldErr{"", v.T[worldErr(err)]})
		}
		if aerr != nil && v.AmountErr == "" {
			v.AmountErr = v.T["err_amount"]
			v.Errors = append(v.Errors, fieldErr{"amount", v.AmountErr})
		}
		v.Cmd = nonce()
		p.writeHTML(w, http.StatusUnprocessableEntity, "practice_listing", v)
	case Partner:
		pid, err := ParsePracticeID(r.PostForm.Get("partner"))
		if err == nil {
			err = s.world.ChoosePartner(key, pid)
		}
		if err != nil {
			p.stepGet(w, r, s, Partner, []fieldErr{{"partner", catalogs[p.locale(r)][worldErr(err)]}})
			return
		}
		home()
	case Trade:
		redirect()
	}
}

func (p *PracticeServer) propose(w http.ResponseWriter, r *http.Request, s *practiceSession) {
	cmd := r.PostForm.Get("cmd")
	lid, err := ParsePracticeID(r.PostForm.Get("listing"))
	if !validNonce(cmd) || err != nil {
		p.fail(w, r, http.StatusBadRequest, "err_bad")
		return
	}
	if _, err := s.world.Propose("propose:"+cmd, lid); err != nil {
		t := catalogs[p.locale(r)]
		p.tradeGet(w, r, s, []fieldErr{{"listing", t[worldErr(err)]}}, http.StatusUnprocessableEntity)
		return
	}
	http.Redirect(w, r, p.cfg.BasePath+"/step/trade#main", http.StatusSeeOther)
}

func (p *PracticeServer) respond(w http.ResponseWriter, r *http.Request, s *practiceSession, respond bool) {
	cmd := r.PostForm.Get("cmd")
	id, err := ParsePracticeID(r.PostForm.Get("proposal"))
	if !validNonce(cmd) || err != nil {
		p.fail(w, r, http.StatusBadRequest, "err_bad")
		return
	}
	if respond {
		accept := r.PostForm.Get("accept")
		if accept != "1" && accept != "0" {
			p.fail(w, r, http.StatusBadRequest, "err_bad")
			return
		}
		err = s.world.Respond("respond:"+cmd, id, accept == "1")
	} else {
		err = s.world.Cancel("cancel:"+cmd, id)
	}
	if err != nil {
		t := catalogs[p.locale(r)]
		p.tradeGet(w, r, s, []fieldErr{{"", t[worldErr(err)]}}, http.StatusConflict)
		return
	}
	http.Redirect(w, r, p.cfg.BasePath+"/step/trade#main", http.StatusSeeOther)
}

// ---- exports: every record is labelled practice ----

type exportDoc struct {
	Practice  bool            `json:"practice"`
	Notice    string          `json:"notice"`
	Members   []exportMember  `json:"members"`
	Listings  []exportListing `json:"listings"`
	Proposals []exportProp    `json:"proposals"`
	Events    []string        `json:"events"`
}
type exportMember struct {
	Practice bool   `json:"practice"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Balance  int64  `json:"balance"`
	Limit    int64  `json:"limit"`
	Reserved int64  `json:"reserved"`
}
type exportListing struct {
	Practice bool   `json:"practice"`
	ID       string `json:"id"`
	Owner    string `json:"owner"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Amount   int64  `json:"amount"`
}
type exportProp struct {
	Practice bool   `json:"practice"`
	ID       string `json:"id"`
	Listing  string `json:"listing"`
	Payer    string `json:"payer"`
	Payee    string `json:"payee"`
	Amount   int64  `json:"amount"`
	State    string `json:"state"`
}

func exportOf(s Snapshot) exportDoc {
	d := exportDoc{Practice: true, Notice: PracticeMarker, Events: s.Events}
	for _, m := range s.Members {
		d.Members = append(d.Members, exportMember{true, m.ID.String(), m.Name, m.Balance, m.Limit, m.Reserved})
	}
	for _, l := range s.Listings {
		d.Listings = append(d.Listings, exportListing{true, l.ID.String(), l.Owner.String(), string(l.Kind), l.Title, l.Amount})
	}
	for _, p := range s.Proposals {
		d.Proposals = append(d.Proposals, exportProp{true, p.ID.String(), p.Listing.String(), p.Payer.String(), p.Payee.String(), p.Amount, string(p.State)})
	}
	return d
}

func (p *PracticeServer) exportJSON(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="PRACTICE-synthetic-data.json"`)
	json.NewEncoder(w).Encode(exportOf(s.world.Snapshot()))
}

// csvSafe neutralises spreadsheet formula injection in user-entered titles.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func (p *PracticeServer) exportCSV(w http.ResponseWriter, r *http.Request, id string, s *practiceSession) {
	d := exportOf(s.world.Snapshot())
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="PRACTICE-synthetic-data.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"flag", "record", "id", "name_or_title", "amount_or_balance", "extra"})
	cw.Write([]string{"PRACTICE", "notice", "", csvSafe(PracticeMarker), "", ""})
	for _, m := range d.Members {
		cw.Write([]string{"PRACTICE", "member", m.ID, csvSafe(m.Name), strconv.FormatInt(m.Balance, 10), "limit=" + strconv.FormatInt(m.Limit, 10)})
	}
	for _, l := range d.Listings {
		cw.Write([]string{"PRACTICE", "listing", l.ID, csvSafe(l.Title), strconv.FormatInt(l.Amount, 10), l.Kind + " by " + l.Owner})
	}
	for _, p := range d.Proposals {
		cw.Write([]string{"PRACTICE", "proposal", p.ID, p.Payer + " -> " + p.Payee, strconv.FormatInt(p.Amount, 10), p.State})
	}
	cw.Flush()
}
