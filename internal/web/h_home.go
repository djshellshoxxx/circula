package web

import (
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

const perPage = 25

func (s *server) homeRoutes() {
	s.mux.HandleFunc("GET /{$}", s.home)
	s.mux.HandleFunc("GET /n/{slug}/{$}", s.netH(s.dashboard))
	s.mux.HandleFunc("GET /n/{slug}/history", s.netH(s.history))
	s.mux.HandleFunc("POST /n/{slug}/export", s.netH(s.exportMine))
	s.mux.HandleFunc("GET /n/{slug}/directory", s.netH(s.directory))
	s.mux.HandleFunc("GET /n/{slug}/members/{id}", s.netH(s.profile))
	s.mux.HandleFunc("GET /n/{slug}/listings", s.netH(s.listings))
	s.mux.HandleFunc("GET /n/{slug}/listings/new", s.netH(s.listingForm))
	s.mux.HandleFunc("POST /n/{slug}/listings/new", s.netH(s.listingCreate))
	s.mux.HandleFunc("GET /n/{slug}/listings/{id}/edit", s.netH(s.listingEditForm))
	s.mux.HandleFunc("POST /n/{slug}/listings/{id}/edit", s.netH(s.listingEdit))
}

type netLink struct{ Name, Slug, Status string }

func (s *server) home(w http.ResponseWriter, r *http.Request) {
	a, _, ok := s.authenticate(r)
	if !ok {
		s.toLogin(w, r)
		return
	}
	mems, err := s.svc.MyMemberships(r.Context(), a)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var links []netLink
	for _, m := range mems {
		if m.Status == domain.MemberDeparted && len(mems) > 1 {
			continue
		}
		n, err := s.svc.GetNetwork(r.Context(), a, m.NetworkID)
		if err != nil {
			continue
		}
		links = append(links, netLink{n.Name, n.Slug, string(m.Status)})
	}
	if len(links) == 1 {
		http.Redirect(w, r, "/n/"+links[0].Slug, http.StatusSeeOther)
		return
	}
	s.render(w, r, nil, "networks", 200, &page{Title: "Your networks", D: links})
}

type dash struct {
	Card     *balanceCard
	Needs    []txRow
	Outgoing []txRow
	Recent   []txRow
	Notices  []annRow
	Err      string
}

// annRow is one announcement on the dashboard.
type annRow struct {
	ID, Title, Body string
	At              template.HTML
}

// announcementRows loads the announcements in force. A failure shows none: they are
// secondary to the balance and the trades waiting for the member.
func (c *rc) announcementRows() []annRow {
	as, err := c.s.svc.ListAnnouncements(c.ctx, c.actor, c.netID(), service.Page{Limit: 5})
	if err != nil {
		return nil
	}
	var out []annRow
	for _, a := range as {
		out = append(out, annRow{ID: a.ID, Title: a.Title, Body: a.Body, At: c.when(a.EffectiveAt)})
	}
	return out
}

// allPending returns every open proposal of the caller, oldest included, by paging the filtered
// query to its end (WEB-02): the dashboard never stops at a fixed window.
func (c *rc) allPending() ([]*domain.Transaction, error) {
	const page = 500
	var out []*domain.Transaction
	after := ""
	for {
		ts, err := c.s.svc.ListPendingForActor(c.ctx, c.actor, c.netID(), service.PendingQuery{Page: service.Page{AfterID: after, Limit: page}})
		if err != nil {
			return out, err
		}
		out = append(out, ts...)
		if len(ts) < page {
			return out, nil
		}
		after = ts[len(ts)-1].ID
	}
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request, c *rc) {
	d := &dash{}
	b, err := s.svc.GetBalance(r.Context(), c.actor, c.netID())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	d.Card = makeCard(b, c.Net.Policy.CurrencyName)
	// Open proposals come from a filtered query, so an old proposal is never
	// hidden behind a window of newer activity (WEB-02, FR-053).
	pending, perr := c.allPending()
	d.Notices = c.announcementRows()
	recent, rerr := s.svc.ListTransactions(r.Context(), c.actor, c.netID(), nil, service.Page{Limit: 30})
	if perr != nil || rerr != nil {
		d.Err = "Your recent activity could not be loaded just now. The figures above are current as of this page load; reload to try again."
	}
	n := c.memberNames()
	for _, t := range pending {
		row := c.row(t, n)
		if row.Needs {
			d.Needs = append(d.Needs, row)
		} else {
			d.Outgoing = append(d.Outgoing, row)
		}
	}
	for _, t := range recent {
		row := c.row(t, n)
		if !row.Needs && !row.Pending && len(d.Recent) < 5 {
			d.Recent = append(d.Recent, row)
		}
	}
	s.render(w, r, c, "dashboard", 200, &page{Title: "Home", D: d, Warn: c.limitNotice()})
}

type historyData struct {
	Pending, Done []txRow
	Filter        string
	Next          string
	Statuses      []string
}

func (s *server) history(w http.ResponseWriter, r *http.Request, c *rc) {
	f := r.URL.Query().Get("status")
	var st []domain.TxStatus
	for _, x := range domain.AllTxStatuses {
		if string(x) == f && x != domain.TxAccepted {
			st = []domain.TxStatus{x}
		}
	}
	txs, err := s.svc.ListTransactions(r.Context(), c.actor, c.netID(), st, service.Page{AfterID: r.URL.Query().Get("after"), Limit: 50})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	h := &historyData{Filter: f}
	for _, t := range txs {
		row := c.row(t, n)
		if row.Pending {
			h.Pending = append(h.Pending, row)
		} else {
			h.Done = append(h.Done, row)
		}
	}
	if len(txs) == 50 {
		h.Next = txs[len(txs)-1].ID
	}
	s.render(w, r, c, "history", 200, &page{Title: "History", D: h, V: map[string]string{"status": f}})
}

// activeMembers returns every active member, paging through the directory without a cap.
func (c *rc) activeMembers() ([]service.DirectoryEntry, error) {
	var out []service.DirectoryEntry
	after := ""
	for {
		pg, err := c.s.svc.ListDirectoryPage(c.ctx, c.actor, c.netID(), service.MemberPageQuery{AfterID: after, Limit: 500})
		if err != nil {
			return nil, err
		}
		out = append(out, pg.Entries...)
		if !pg.HasNext || len(pg.Entries) == 0 {
			return out, nil
		}
		after = pg.Entries[len(pg.Entries)-1].MemberID
	}
}

// activeOther returns the display name of an active member other than the caller. It reads
// one member, so it does not depend on any list.
func (c *rc) activeOther(id string) (string, bool) {
	if id == "" || id == c.Me.ID {
		return "", false
	}
	m, err := c.s.svc.GetMember(c.ctx, c.actor, c.netID(), id)
	if err != nil || m.Status != domain.MemberActive {
		return "", false
	}
	return m.DisplayName, true
}

type dirData struct {
	Entries []service.DirectoryEntry
	Q       string
	Pages   pageNav
}

func (s *server) directory(w http.ResponseWriter, r *http.Request, c *rc) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	total, err := s.svc.CountDirectory(r.Context(), c.actor, c.netID(), q)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	mq := service.MemberPageQuery{Name: q, Limit: perPage}
	cur := r.URL.Query()
	paged := false
	switch {
	case cur.Get("after") != "":
		mq.AfterID, paged = cur.Get("after"), true
	case cur.Get("before") != "":
		mq.BeforeID, paged = cur.Get("before"), true
	}
	pg, err := s.svc.ListDirectoryPage(r.Context(), c.actor, c.netID(), mq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	base := c.base() + "/directory?" + url.Values{"q": {q}}.Encode()
	d := &dirData{Entries: pg.Entries, Q: q, Pages: pageNav{Total: total}}
	if paged {
		d.Pages.First = base
	}
	if n := len(pg.Entries); n > 0 {
		if pg.HasPrev {
			d.Pages.Prev = base + "&before=" + url.QueryEscape(pg.Entries[0].MemberID)
		}
		if pg.HasNext {
			d.Pages.Next = base + "&after=" + url.QueryEscape(pg.Entries[n-1].MemberID)
		}
	}
	s.render(w, r, c, "directory", 200, &page{Title: "Directory", D: d, V: map[string]string{"q": q}})
}

type profileData struct {
	ID, Name string
	Listings []*domain.Listing
	Self     bool
}

func (s *server) profile(w http.ResponseWriter, r *http.Request, c *rc) {
	id := r.PathValue("id")
	name, ok := c.activeOther(id)
	if id == c.Me.ID {
		name, ok = c.Me.DisplayName, true
	}
	if !ok {
		s.fail(w, r, domain.NotFound("member"))
		return
	}
	ls, err := s.svc.ListListings(r.Context(), c.actor, c.netID(), service.ListingQuery{MemberID: id, Page: service.Page{Limit: 100}})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.render(w, r, c, "profile", 200, &page{Title: name, D: &profileData{ID: id, Name: name, Listings: ls, Self: id == c.Me.ID}})
}

type listingsData struct {
	Items []listingRow
	Mine  bool
	Next  string
}

type listingRow struct {
	*domain.Listing
	Owner string
	Price string
	Own   bool
}

func (s *server) listings(w http.ResponseWriter, r *http.Request, c *rc) {
	q := r.URL.Query()
	in := service.ListingQuery{Type: domain.ListingType(q.Get("type")), Query: strings.TrimSpace(q.Get("q")),
		Page: service.Page{AfterID: q.Get("after"), Limit: 50}}
	mine := q.Get("mine") == "1"
	if mine {
		in.MemberID = c.Me.ID
	}
	ls, err := s.svc.ListListings(r.Context(), c.actor, c.netID(), in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	n := c.memberNames()
	d := &listingsData{Mine: mine}
	for _, l := range ls {
		row := listingRow{Listing: l, Owner: n.of(l.MemberID), Own: l.MemberID == c.Me.ID}
		if l.Price != nil {
			row.Price = domain.FormatAmount(*l.Price, c.Net.Policy.Decimals)
		}
		d.Items = append(d.Items, row)
	}
	if len(ls) == 50 {
		d.Next = ls[49].ID
	}
	s.render(w, r, c, "listings", 200, &page{Title: "Listings", D: d,
		V: map[string]string{"type": q.Get("type"), "q": in.Query}})
}

func (s *server) listingForm(w http.ResponseWriter, r *http.Request, c *rc) {
	s.render(w, r, c, "listing_form", 200, &page{Title: "New listing", V: map[string]string{"type": "offer"}})
}

func (s *server) listingCreate(w http.ResponseWriter, r *http.Request, c *rc) {
	v := formValues(r)
	p := &page{Title: "New listing", V: v}
	in := service.ListingInput{Type: domain.ListingType(v["type"]), Title: v["title"], Description: v["description"],
		IdempotencyKey: idemKey(r)}
	if strings.TrimSpace(v["price"]) != "" {
		pr, err := domain.ParseAmount(v["price"], c.Net.Policy.Decimals)
		if err != nil || pr <= 0 {
			p.Errors = []fieldErr{{"price", priceHelp(c)}}
			p.Summary = "Not saved. Fix the problems below. Nothing changed."
			s.render(w, r, c, "listing_form", 422, p)
			return
		}
		in.Price = &pr
	}
	if _, err := s.svc.CreateListing(r.Context(), c.actor, c.netID(), in); err != nil {
		s.formFail(w, r, c, "listing_form", p, err)
		return
	}
	s.setFlash(w, r, "Listing published.")
	http.Redirect(w, r, c.base()+"/listings?mine=1", http.StatusSeeOther)
}

func priceHelp(c *rc) string {
	return "Price: enter a positive amount with up to " + strconv.Itoa(c.Net.Policy.Decimals) + " decimal places, for example " + domain.FormatAmount(1250, c.Net.Policy.Decimals) + "."
}

// formFail re-renders a form with the refusal, or falls back to a page error.
func (s *server) formFail(w http.ResponseWriter, r *http.Request, c *rc, tmpl string, p *page, err error) {
	sum, fe, ok := c.refuse(err)
	if !ok {
		s.fail(w, r, err)
		return
	}
	p.Summary, p.Errors = sum, append(p.Errors, fe...)
	st := http.StatusUnprocessableEntity
	if domain.CodeOf(err) == domain.CodeInvalidState || domain.CodeOf(err) == domain.CodeBlocked {
		st = http.StatusConflict
	}
	s.render(w, r, c, tmpl, st, p)
}

func (c *rc) myListing(id string) *domain.Listing {
	ls, err := c.s.svc.ListListings(c.ctx, c.actor, c.netID(), service.ListingQuery{MemberID: c.Me.ID, Page: service.Page{Limit: 500}})
	if err != nil {
		return nil
	}
	for _, l := range ls {
		if l.ID == id {
			return l
		}
	}
	return nil
}

func (s *server) listingEditForm(w http.ResponseWriter, r *http.Request, c *rc) {
	l := c.myListing(r.PathValue("id"))
	if l == nil {
		s.fail(w, r, domain.NotFound("listing"))
		return
	}
	v := map[string]string{"type": string(l.Type), "title": l.Title, "description": l.Description, "status": string(l.Status)}
	if l.Price != nil {
		v["price"] = domain.FormatAmount(*l.Price, c.Net.Policy.Decimals)
	}
	s.render(w, r, c, "listing_edit", 200, &page{Title: "Edit listing", V: v, D: l.ID})
}

func (s *server) listingEdit(w http.ResponseWriter, r *http.Request, c *rc) {
	id := r.PathValue("id")
	v := formValues(r)
	p := &page{Title: "Edit listing", V: v, D: id}
	st := domain.ListingStatus(v["status"])
	title, desc := v["title"], v["description"]
	up := service.ListingUpdate{ListingID: id, Title: &title, Description: &desc, Status: &st}
	if strings.TrimSpace(v["price"]) == "" {
		up.ClearPrice = true
	} else {
		pr, err := domain.ParseAmount(v["price"], c.Net.Policy.Decimals)
		if err != nil || pr <= 0 {
			p.Errors = []fieldErr{{"price", priceHelp(c)}}
			p.Summary = "Not saved. Fix the problems below. Nothing changed."
			s.render(w, r, c, "listing_edit", 422, p)
			return
		}
		up.Price = &pr
	}
	if _, err := s.svc.UpdateListing(r.Context(), c.actor, c.netID(), up); err != nil {
		s.formFail(w, r, c, "listing_edit", p, err)
		return
	}
	s.setFlash(w, r, "Listing saved.")
	http.Redirect(w, r, c.base()+"/listings?mine=1", http.StatusSeeOther)
}
