package web

import (
	"net/http"
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
	Err      string
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
	pending, perr := s.svc.ListPendingForActor(r.Context(), c.actor, c.netID(), service.PendingQuery{Page: service.Page{Limit: 100}})
	recent, rerr := s.svc.ListTransactions(r.Context(), c.actor, c.netID(), nil, service.Page{Limit: 30})
	if perr != nil || rerr != nil {
		d.Err = "Your recent activity could not be loaded just now. The figures above are current as of this page load; reload to try again."
	}
	n := c.memberNames()
	for _, t := range pending {
		row := makeRow(t, c.Me, n, c.Net.Policy.CurrencyName, c.Net.Policy.Decimals)
		if row.Needs {
			d.Needs = append(d.Needs, row)
		} else {
			d.Outgoing = append(d.Outgoing, row)
		}
	}
	for _, t := range recent {
		row := makeRow(t, c.Me, n, c.Net.Policy.CurrencyName, c.Net.Policy.Decimals)
		if !row.Needs && !row.Pending && len(d.Recent) < 5 {
			d.Recent = append(d.Recent, row)
		}
	}
	s.render(w, r, c, "dashboard", 200, &page{Title: "Home", D: d})
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
		row := makeRow(t, c.Me, n, c.Net.Policy.CurrencyName, c.Net.Policy.Decimals)
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

func (c *rc) directoryAll() []service.DirectoryEntry {
	var all []service.DirectoryEntry
	after := ""
	for i := 0; i < 20; i++ {
		es, err := c.s.svc.ListDirectory(c.ctx, c.actor, c.netID(), service.Page{AfterID: after, Limit: 500})
		if err != nil || len(es) == 0 {
			break
		}
		all = append(all, es...)
		after = es[len(es)-1].MemberID
		if len(es) < 500 {
			break
		}
	}
	return all
}

type dirData struct {
	Entries    []service.DirectoryEntry
	Q          string
	Prev, Next int
	Total      int
}

func (s *server) directory(w http.ResponseWriter, r *http.Request, c *rc) {
	if _, err := s.svc.ListDirectory(r.Context(), c.actor, c.netID(), service.Page{Limit: 1}); err != nil {
		s.fail(w, r, err)
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	var match []service.DirectoryEntry
	for _, e := range c.directoryAll() {
		if q == "" || strings.Contains(strings.ToLower(e.DisplayName), strings.ToLower(q)) {
			match = append(match, e)
		}
	}
	pg, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if pg < 0 || pg*perPage > len(match) {
		pg = 0
	}
	end := min(len(match), (pg+1)*perPage)
	d := &dirData{Entries: match[pg*perPage : end], Q: q, Total: len(match), Prev: -1, Next: -1}
	if pg > 0 {
		d.Prev = pg - 1
	}
	if end < len(match) {
		d.Next = pg + 1
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
	name := ""
	for _, e := range c.directoryAll() {
		if e.MemberID == id {
			name = e.DisplayName
		}
	}
	if name == "" {
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
