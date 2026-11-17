package web

import (
	"errors"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

func (s *Server) funcs() template.FuncMap {
	return template.FuncMap{"inc": func(i int) int { return i + 1 }}
}

type formErr struct{ Field, Message string }

type base struct {
	Base, PageTitle, Nav string
	IsAdmin              bool
	Notice               string
	CSRF                 string
	Errors               []formErr
}

func (b base) Err(field string) string {
	for _, e := range b.Errors {
		if e.Field == field {
			return e.Message
		}
	}
	return ""
}

var notices = map[string]string{
	"saved": "Details saved.", "renewed": "Listing renewed.", "photo-added": "Photo added.", "photo-removed": "Photo removed.",
	"category-added": "Category added.", "category-updated": "Category updated.", "area-added": "Area added.", "area-updated": "Area updated.",
	"tag-moderated": "Tag updated.", "seeded": "Default categories added.",
}

func (s *Server) newBase(r *http.Request, a d.Actor, title, nav string) base {
	return base{Base: s.base, PageTitle: title, Nav: nav, IsAdmin: a.Admin(), CSRF: a.CSRF, Notice: notices[r.URL.Query().Get("notice")]}
}

type opt struct{ Value, Label string }

var conditionOpts = []opt{{"new", "New"}, {"like_new", "Like new"}, {"good", "Good"}, {"fair", "Fair"}, {"for_parts", "For parts or repair"}}

func label(opts []opt, v string) string {
	for _, o := range opts {
		if o.Value == v {
			return o.Label
		}
	}
	return ""
}

type areaView struct {
	ID, Name  string
	Enabled   bool
	Cell      bool
	CellLabel string
}

func areaViews(as []d.Area, admin bool) []areaView {
	out := make([]areaView, 0, len(as))
	for _, a := range as {
		v := areaView{ID: a.ID, Name: a.Name, Enabled: a.Enabled, Cell: a.Cell != nil}
		if a.Cell != nil && admin {
			v.CellLabel = fmt.Sprintf("%.1f, %.1f", float64(a.Cell.Lat10)/10, float64(a.Cell.Lon10)/10)
		}
		out = append(out, v)
	}
	return out
}

func (s *Server) render(w http.ResponseWriter, status int, name string, data any) {
	var buf strings.Builder
	if err := s.tpl.ExecuteTemplate(&buf, name, data); err != nil {
		s.cfg.Logf("discovery/web: template %s: %v", name, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(buf.String()))
}

// page wraps HTML endpoints: auth, CSRF, friendly errors.
func (s *Server) page(fn handlerFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a, err := s.gate(w, r, true)
		if err == nil {
			err = fn(w, r, a)
		}
		if err != nil {
			s.renderError(w, r, a, err)
		}
	}
}

type errPage struct {
	base
	Heading, Message string
}

func (s *Server) renderError(w http.ResponseWriter, r *http.Request, a d.Actor, err error) {
	status, p := s.classify(err)
	heading, msg := "Something went wrong", "Please try again."
	switch status {
	case 401:
		heading, msg = "Sign in required", "Your session is not signed in."
	case 403:
		heading, msg = "Not allowed", "You do not have permission to do that."
		if errors.Is(err, errCSRF) {
			msg = "The form expired or was not sent from this site. Go back, reload the page and try again."
		}
	case 404:
		heading, msg = "Not found", "We could not find that page or listing."
	case 400, 413, 415, 422, 409, 429:
		heading, msg = p.Title, p.Detail
		if msg == "" {
			msg = "Please check your input and try again."
		}
	}
	b := s.newBase(r, a, heading, "")
	b.Notice = ""
	s.render(w, status, "error", errPage{base: b, Heading: heading, Message: msg})
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request, path, notice string) {
	http.Redirect(w, r, s.base+path+"?notice="+url.QueryEscape(notice), http.StatusSeeOther)
}

// ---------- search ----------

type itemView struct {
	Title, URL string
	Meta, Tags []string
	Badges     []string
	CoverURL   string
	CoverAlt   string
	CoverW     int
	CoverH     int
}

type searchPage struct {
	base
	Heading                     string
	F                           d.Filters
	Categories                  []d.Category
	Areas                       []areaView
	Conditions                  []opt
	Limits                      []int
	MaxKM                       int
	BadRequest                  bool
	Items                       []itemView
	HasNext, HasCursor          bool
	NextURL, FirstURL, SavedURL string
}

func (s *Server) pageSearch(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	cfg := s.svc.Config()
	f, cur, perr := d.ParseFilters(r.URL.Query(), cfg)
	nav, heading := "browse", "Browse listings"
	if f.Scope == d.ScopeMine {
		nav, heading = "mine", "My listings"
	}
	if f.Scope == d.ScopeAll {
		nav, heading = "admin", "All listings (administrator view)"
	}
	cats, err := s.svc.ListCategories(r.Context(), a)
	if err != nil {
		return err
	}
	areas, err := s.svc.ListAreas(r.Context(), a)
	if err != nil {
		return err
	}
	pg := searchPage{base: s.newBase(r, a, heading, nav), Heading: heading, F: f, Categories: cats, Areas: areaViews(areas, false),
		Conditions: conditionOpts, Limits: []int{10, 20, 50}, MaxKM: cfg.MaxWithinKM}
	pg.Notice = ""
	if perr != nil {
		pg.BadRequest = true
		pg.F = d.Filters{Limit: cfg.DefaultPage, Sort: d.SortRecent}
		pg.Errors = []formErr{{Message: friendly(perr)}}
		s.render(w, http.StatusBadRequest, "search", pg)
		return nil
	}
	res, err := s.svc.Search(r.Context(), a, f, cur)
	if errors.Is(err, d.ErrBadCursor) {
		pg.BadRequest = true
		pg.Errors = []formErr{{Message: "That page link has expired. Start again from the first page."}}
		pg.FirstURL = s.base + "/listings?" + f.Values(cfg).Encode()
		s.render(w, http.StatusBadRequest, "search", pg)
		return nil
	}
	if err != nil {
		var de *d.Error
		if errors.As(err, &de) && errors.Is(err, d.ErrInvalid) {
			pg.BadRequest = true
			pg.F = d.Filters{Limit: cfg.DefaultPage, Sort: d.SortRecent}
			pg.Errors = []formErr{{Message: friendly(err)}}
			s.render(w, http.StatusUnprocessableEntity, "search", pg)
			return nil
		}
		return err
	}
	pg.F = res.Filters
	catName := map[string]string{}
	for _, c := range cats {
		catName[c.ID] = c.Name
	}
	areaName := map[string]string{}
	for _, x := range areas {
		areaName[x.ID] = x.Name
	}
	for _, it := range res.Items {
		pg.Items = append(pg.Items, s.itemView(a, it, catName, areaName))
	}
	q := res.Filters.Values(cfg)
	pg.SavedURL = s.base + "/listings"
	if enc := q.Encode(); enc != "" {
		pg.SavedURL += "?" + enc
	}
	pg.FirstURL = pg.SavedURL
	pg.HasCursor = cur != ""
	if res.Next != "" {
		q.Set("cursor", res.Next)
		pg.HasNext, pg.NextURL = true, s.base+"/listings?"+q.Encode()
	}
	s.render(w, 200, "search", pg)
	return nil
}

func (s *Server) itemView(a d.Actor, it d.SearchItem, catName, areaName map[string]string) itemView {
	now := s.svc.Config().Now()
	v := itemView{Title: it.Listing.Title, URL: s.base + "/listings/" + it.Listing.ID, Tags: it.Tags}
	kind := "Offer"
	if it.Listing.Kind == d.KindRequest {
		kind = "Request"
	}
	v.Meta = append(v.Meta, kind)
	m := it.Meta
	if m.Type == d.TypeGoods {
		v.Meta = append(v.Meta, "Goods")
	} else if m.Type == d.TypeService {
		v.Meta = append(v.Meta, "Service")
	}
	if n := catName[m.CategoryID]; n != "" {
		v.Meta = append(v.Meta, n)
	}
	if l := label(conditionOpts, m.Condition); l != "" {
		v.Meta = append(v.Meta, l)
	}
	if n := areaName[m.AreaID]; n != "" {
		v.Meta = append(v.Meta, n)
	}
	switch m.Availability {
	case d.AvailLimited:
		v.Meta = append(v.Meta, "Limited availability")
	case d.AvailUnavailable:
		v.Meta = append(v.Meta, "Currently unavailable")
	}
	if it.Listing.OwnerID == a.ID {
		v.Badges = append(v.Badges, "Yours")
	}
	if it.Listing.Status != d.StatusActive {
		v.Badges = append(v.Badges, strings.ToUpper(it.Listing.Status[:1])+it.Listing.Status[1:])
	}
	if !m.ExpiresAt.IsZero() && !m.ExpiresAt.After(now) {
		v.Badges = append(v.Badges, "Expired")
	}
	if it.Cover != nil {
		v.CoverURL = s.mediaURL(it.Listing.ID, it.Cover.ID, true)
		v.CoverAlt, v.CoverW, v.CoverH = it.Cover.Alt, it.Cover.ThumbWidth, it.Cover.ThumbHeight
	}
	return v
}

func friendly(err error) string {
	var de *d.Error
	if errors.As(err, &de) && de.Detail != "" {
		msg := de.Detail
		if de.Field != "" {
			msg = strings.ReplaceAll(de.Field, "_", " ") + ": " + msg
		}
		return strings.ToUpper(msg[:1]) + msg[1:] + "."
	}
	return "That request could not be completed."
}

// ---------- detail ----------

type detailPage struct {
	base
	V                                                                               d.ListingView
	Hidden                                                                          bool
	KindLabel, CategoryName, TypeLabel, ConditionLabel, AreaName, AvailabilityLabel string
	s                                                                               *Server
}

func (p detailPage) FullURL(m d.MediaInfo) string  { return p.s.mediaURL(m.ListingID, m.ID, false) }
func (p detailPage) ThumbURL(m d.MediaInfo) string { return p.s.mediaURL(m.ListingID, m.ID, true) }

func (s *Server) names(r *http.Request, a d.Actor) (map[string]string, map[string]string, []d.Category, []d.Area, error) {
	cats, err := s.svc.ListCategories(r.Context(), a)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	areas, err := s.svc.ListAreas(r.Context(), a)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	cn, an := map[string]string{}, map[string]string{}
	for _, c := range cats {
		cn[c.ID] = c.Name
	}
	for _, x := range areas {
		an[x.ID] = x.Name
	}
	return cn, an, cats, areas, nil
}

func (s *Server) pageDetail(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	v, err := s.svc.GetListing(r.Context(), a, r.PathValue("id"))
	if err != nil {
		return err
	}
	cn, an, _, _, err := s.names(r, a)
	if err != nil {
		return err
	}
	pg := detailPage{base: s.newBase(r, a, v.Title, "browse"), V: v, s: s}
	pg.Hidden = v.Status != d.StatusActive || v.Expired || !v.OwnerActive
	pg.KindLabel = map[string]string{d.KindOffer: "Offer", d.KindRequest: "Request"}[v.Kind]
	pg.CategoryName, pg.AreaName = cn[v.CategoryID], an[v.AreaID]
	pg.TypeLabel = map[string]string{d.TypeGoods: "Goods", d.TypeService: "Service"}[v.Type]
	pg.ConditionLabel = label(conditionOpts, v.Condition)
	pg.AvailabilityLabel = map[string]string{d.AvailAvailable: "Available", d.AvailLimited: "Limited", d.AvailUnavailable: "Currently unavailable"}[v.Availability]
	s.render(w, 200, "detail", pg)
	return nil
}

// ---------- edit ----------

type editForm struct {
	Category, Type, Condition, Area, Availability, AvailableFrom, ExpiresAt, Tags string
	Version                                                                       int64
}

type editPage struct {
	base
	V                                        d.ListingView
	Form                                     editForm
	Categories                               []d.Category
	Areas                                    []areaView
	Conditions                               []opt
	MaxTags, MaxPhotos, MaxFileMB, RenewDays int
	s                                        *Server
}

func (p editPage) ThumbURL(m d.MediaInfo) string { return p.s.mediaURL(m.ListingID, m.ID, true) }

func (s *Server) editData(r *http.Request, a d.Actor, v d.ListingView, form editForm) (editPage, error) {
	_, _, cats, areas, err := s.names(r, a)
	if err != nil {
		return editPage{}, err
	}
	cfg := s.svc.Config()
	pg := editPage{base: s.newBase(r, a, "Edit "+v.Title, "mine"), V: v, Form: form, Categories: cats, Areas: areaViews(areas, false),
		Conditions: conditionOpts, MaxTags: cfg.MaxTagsPerListing, MaxPhotos: cfg.MaxPhotos, RenewDays: int(cfg.RenewExtend / (24 * time.Hour)), s: s}
	pg.MaxFileMB = int(s.svc.ImageLimits().MaxInputBytes >> 20)
	if pg.MaxFileMB < 1 {
		pg.MaxFileMB = 1
	}
	return pg, nil
}

func formFromView(v d.ListingView) editForm {
	f := editForm{Category: v.CategoryID, Type: v.Type, Condition: v.Condition, Area: v.AreaID, Availability: v.Availability, Version: v.Version}
	if !v.AvailableFrom.IsZero() {
		f.AvailableFrom = v.AvailableFrom.Format("2006-01-02")
	}
	if !v.ExpiresAt.IsZero() {
		f.ExpiresAt = v.ExpiresAt.Format("2006-01-02")
	}
	var tags []string
	for _, t := range v.Tags {
		tags = append(tags, t.Name)
	}
	f.Tags = strings.Join(tags, ", ")
	return f
}

func (s *Server) editable(r *http.Request, a d.Actor) (d.ListingView, error) {
	v, err := s.svc.GetListing(r.Context(), a, r.PathValue("id"))
	if err != nil {
		return v, err
	}
	if !v.CanEdit {
		return v, d.ErrForbidden
	}
	return v, nil
}

func (s *Server) pageEdit(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	v, err := s.editable(r, a)
	if err != nil {
		return err
	}
	pg, err := s.editData(r, a, v, formFromView(v))
	if err != nil {
		return err
	}
	s.render(w, 200, "edit", pg)
	return nil
}

func (s *Server) rerenderEdit(w http.ResponseWriter, r *http.Request, a d.Actor, form editForm, cause error, extra ...formErr) error {
	v, err := s.editable(r, a)
	if err != nil {
		return err
	}
	pg, err := s.editData(r, a, v, form)
	if err != nil {
		return err
	}
	status, _ := s.classify(cause)
	if status >= 500 {
		return cause
	}
	pg.Notice = ""
	msg := friendly(cause)
	var de *d.Error
	field := ""
	if errors.As(cause, &de) {
		field = de.Field
	}
	switch {
	case errors.Is(cause, d.ErrConflict):
		msg, field = "This listing changed since you opened the form. Reload the page and try again.", ""
	case errors.Is(cause, d.ErrRateLimited):
		msg = "You are uploading too quickly. Please wait a few minutes."
	case errors.Is(cause, d.ErrQuota) && field == "file":
		msg = "The photo is too large or you have reached the photo limit."
	}
	formField := map[string]string{"category": "category", "type": "type", "condition": "condition", "area": "area", "availability": "availability",
		"available_from": "available_from", "expires_at": "expires_at", "tags": "tags", "file": "file", "alt": "alt"}[field]
	pg.Errors = append(pg.Errors, formErr{Field: formField, Message: msg})
	pg.Errors = append(pg.Errors, extra...)
	s.render(w, status, "edit", pg)
	return nil
}

func parseDate(field, v string, endOfDay bool) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return t, &d.Error{Kind: d.ErrInvalid, Field: field, Detail: "enter a date as YYYY-MM-DD"}
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t.UTC(), nil
}

func (s *Server) postEdit(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	ver, _ := strconv.ParseInt(r.PostFormValue("version"), 10, 64)
	form := editForm{Category: r.PostFormValue("category"), Type: r.PostFormValue("type"), Condition: r.PostFormValue("condition"),
		Area: r.PostFormValue("area"), Availability: r.PostFormValue("availability"), AvailableFrom: r.PostFormValue("available_from"),
		ExpiresAt: r.PostFormValue("expires_at"), Tags: r.PostFormValue("tags"), Version: ver}
	from, err := parseDate("available_from", form.AvailableFrom, false)
	if err != nil {
		return s.rerenderEdit(w, r, a, form, err)
	}
	exp, err := parseDate("expires_at", form.ExpiresAt, true)
	if err != nil {
		return s.rerenderEdit(w, r, a, form, err)
	}
	var tags []string
	for _, t := range strings.FieldsFunc(form.Tags, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if strings.TrimSpace(t) != "" {
			tags = append(tags, t)
		}
	}
	_, err = s.svc.UpdateDiscovery(r.Context(), a, r.PathValue("id"), d.DiscoveryInput{CategoryID: form.Category, Type: form.Type, Condition: form.Condition,
		AreaID: form.Area, Availability: form.Availability, AvailableFrom: from, ExpiresAt: exp, Tags: tags, ExpectedVersion: ver})
	if err != nil {
		return s.rerenderEdit(w, r, a, form, err)
	}
	s.redirect(w, r, "/listings/"+r.PathValue("id")+"/edit", "saved")
	return nil
}

func (s *Server) postRenew(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	ver, _ := strconv.ParseInt(r.PostFormValue("version"), 10, 64)
	if _, err := s.svc.Renew(r.Context(), a, r.PathValue("id"), ver); err != nil {
		if errors.Is(err, d.ErrConflict) || errors.Is(err, d.ErrInvalid) {
			v, gerr := s.editable(r, a)
			if gerr != nil {
				return gerr
			}
			return s.rerenderEdit(w, r, a, formFromView(v), err)
		}
		return err
	}
	s.redirect(w, r, "/listings/"+r.PathValue("id")+"/edit", "renewed")
	return nil
}

func (s *Server) postPhoto(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	alt, rd, cleanup, err := s.uploadFrom(w, r)
	defer cleanup()
	if err == nil {
		_, err = s.svc.AddMedia(r.Context(), a, r.PathValue("id"), alt, rd)
	}
	if err != nil {
		v, gerr := s.editable(r, a)
		if gerr != nil {
			return gerr
		}
		var he *httpError
		if errors.As(err, &he) {
			return err
		}
		return s.rerenderEdit(w, r, a, formFromView(v), err)
	}
	s.redirect(w, r, "/listings/"+r.PathValue("id")+"/edit", "photo-added")
	return nil
}

func (s *Server) postRemovePhoto(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	if err := s.svc.RemoveMedia(r.Context(), a, r.PathValue("id"), r.PathValue("mediaId")); err != nil {
		return err
	}
	s.redirect(w, r, "/listings/"+r.PathValue("id")+"/edit", "photo-removed")
	return nil
}

// ---------- admin ----------

type adminPage struct {
	base
	Categories []d.Category
	Areas      []areaView
	Tags       []d.Tag
}

func (s *Server) pageAdmin(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	if !a.Admin() {
		return d.ErrForbidden
	}
	return s.renderAdmin(w, r, a, 200, nil)
}

func (s *Server) renderAdmin(w http.ResponseWriter, r *http.Request, a d.Actor, status int, errs []formErr) error {
	cats, err := s.svc.ListCategories(r.Context(), a)
	if err != nil {
		return err
	}
	areas, err := s.svc.ListAreas(r.Context(), a)
	if err != nil {
		return err
	}
	tags, err := s.svc.ListTags(r.Context(), a, "")
	if err != nil {
		return err
	}
	pg := adminPage{base: s.newBase(r, a, "Categories, areas and tags", "admin"), Categories: cats, Areas: areaViews(areas, true), Tags: tags}
	if errs != nil {
		pg.Notice, pg.Errors = "", errs
	}
	s.render(w, status, "admin", pg)
	return nil
}

func (s *Server) adminResult(w http.ResponseWriter, r *http.Request, a d.Actor, err error, notice string) error {
	if err == nil {
		s.redirect(w, r, "/admin/taxonomy", notice)
		return nil
	}
	if errors.Is(err, d.ErrInvalid) || errors.Is(err, d.ErrDuplicate) || errors.Is(err, d.ErrQuota) {
		status, _ := s.classify(err)
		return s.renderAdmin(w, r, a, status, []formErr{{Message: friendly(err)}})
	}
	return err
}

func (s *Server) postAdminCategory(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	if r.PostFormValue("seed") == "1" {
		_, err := s.svc.SeedDefaultCategories(r.Context(), a)
		return s.adminResult(w, r, a, err, "seeded")
	}
	_, err := s.svc.CreateCategory(r.Context(), a, r.PostFormValue("name"), r.PostFormValue("kind"))
	return s.adminResult(w, r, a, err, "category-added")
}

func (s *Server) postAdminCategoryUpdate(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var name *string
	var en *bool
	t, f := true, false
	switch r.PostFormValue("op") {
	case "rename":
		n := r.PostFormValue("name")
		name = &n
	case "enable":
		en = &t
	case "disable":
		en = &f
	default:
		return badRequest("unknown operation")
	}
	return s.adminResult(w, r, a, s.svc.UpdateCategory(r.Context(), a, r.PathValue("cid"), name, en), "category-updated")
}

func (s *Server) postAdminArea(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var cell *d.GeoCell
	lat, lon := strings.TrimSpace(r.PostFormValue("lat")), strings.TrimSpace(r.PostFormValue("lon"))
	if lat != "" || lon != "" {
		la, e1 := strconv.ParseFloat(lat, 64)
		lo, e2 := strconv.ParseFloat(lon, 64)
		if e1 != nil || e2 != nil || math.IsNaN(la) || math.IsNaN(lo) || math.IsInf(la, 0) || math.IsInf(lo, 0) || math.Abs(la) > 90 || math.Abs(lo) > 180 {
			return s.adminResult(w, r, a, &d.Error{Kind: d.ErrInvalid, Field: "centre", Detail: "enter both latitude and longitude as decimal degrees"}, "")
		}
		cell = &d.GeoCell{Lat10: int(math.Round(la * 10)), Lon10: int(math.Round(lo * 10))}
	}
	_, err := s.svc.CreateArea(r.Context(), a, r.PostFormValue("name"), cell)
	return s.adminResult(w, r, a, err, "area-added")
}

func (s *Server) postAdminAreaUpdate(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	t, f := true, false
	var en *bool
	switch r.PostFormValue("op") {
	case "enable":
		en = &t
	case "disable":
		en = &f
	default:
		return badRequest("unknown operation")
	}
	return s.adminResult(w, r, a, s.svc.UpdateArea(r.Context(), a, r.PathValue("aid"), nil, en), "area-updated")
}

func (s *Server) postAdminTag(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	tag, _ := url.PathUnescape(r.PathValue("tag"))
	err := s.svc.ModerateTag(r.Context(), a, tag, d.TagStatus(r.PostFormValue("status")))
	return s.adminResult(w, r, a, err, "tag-moderated")
}
