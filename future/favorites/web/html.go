package web

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"

	"github.com/djshellshoxxx/circula/future/favorites"
)

const maxBody = 64 << 10 // 64 KiB, matching the API body policy

var notices = map[string]string{
	"added": "Saved trading partner added.", "exists": "That member is already in your list.", "renamed": "Nickname saved.",
	"pinned": "Pinned to the top.", "unpinned": "Unpinned.", "moved": "Order saved.", "edge": "That partner is already at the edge of its group.",
	"removed": "Removed from your list.", "gone": "That partner was not in your list.",
}

type pageFn func(w http.ResponseWriter, r *http.Request, s Session)

// page wraps an HTML handler with headers, authentication and, for writes,
// same-origin and CSRF checks. Failures reveal nothing about other members.
func (h *Handler) page(unsafe bool, fn pageFn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.secureHeaders(w, true)
		s, err := h.auth.Authenticate(r)
		if err != nil || !s.valid() {
			http.Error(w, "Sign in required.", http.StatusUnauthorized)
			return
		}
		if unsafe {
			if !sameOrigin(r) {
				http.Error(w, "Cross-site request refused.", http.StatusForbidden)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Bad request.", http.StatusBadRequest)
				return
			}
			if !csrfOK(s.CSRF, r.PostForm.Get("csrf_token")) {
				http.Error(w, "Invalid or missing security token. Reload the page and try again.", http.StatusForbidden)
				return
			}
		}
		fn(w, r, s)
	}
}

func (s Session) valid() bool { return s.Actor.Network != "" && s.Actor.Member != "" && s.CSRF != "" }

type itemView struct {
	Target, Path, Name, Nickname, DraftKey string
	Pinned, First, Last                    bool
}

type listView struct {
	Base, Directory, CSRF  string
	Items                  []itemFull
	Next                   string
	Revision               int64
	Errors                 []fieldErr
	Notice                 string
	AddTarget, AddNickname string
	AddPinned              bool
}

type fieldErr struct{ Field, Msg string }

type draftView struct {
	Base, Directory, CSRF string
	D                     favorites.Draft
	P                     ProposalFields
}

var tpl = template.Must(template.New("").Funcs(template.FuncMap{"esc": url.PathEscape}).Parse(`
{{define "head"}}<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="referrer" content="no-referrer"><meta name="robots" content="noindex">
<title>{{.}} - Circula</title><style>` + css + `</style></head>
<body><a class="skip" href="#main">Skip to main content</a>
<main id="main" tabindex="-1">{{end}}
{{define "foot"}}</main></body></html>{{end}}

{{define "errors"}}{{if .}}<div class="err" role="alert" tabindex="-1" aria-labelledby="err-h"><h2 id="err-h">There is a problem</h2><ul>{{range .}}<li>{{if .Field}}<a href="#{{.Field}}">{{.Msg}}</a>{{else}}{{.Msg}}{{end}}</li>{{end}}</ul></div>{{end}}{{end}}

{{define "item"}}
<li class="item">
<h3>{{.Name}}{{if .Pinned}}<span class="badge">Pinned</span>{{end}}</h3>
{{if .Nickname}}<p class="hint">Your nickname: {{.Nickname}}</p>{{end}}
<div class="row">
<form method="post" action="{{.Base}}/{{esc .Target}}/draft"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="request_key" value="{{.DraftKey}}"><button type="submit">Start a proposal<span class="vh"> with {{.Name}}</span></button></form>
<form method="post" action="{{.Base}}/{{esc .Target}}/pin"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="pinned" value="{{if .Pinned}}0{{else}}1{{end}}"><button type="submit" class="secondary">{{if .Pinned}}Unpin{{else}}Pin{{end}}<span class="vh"> {{.Name}}</span></button></form>
{{if not .First}}<form method="post" action="{{.Base}}/{{esc .Target}}/move"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="revision" value="{{.Revision}}"><input type="hidden" name="dir" value="up"><button type="submit" class="secondary">Move up<span class="vh"> {{.Name}}</span></button></form>{{end}}
{{if not .Last}}<form method="post" action="{{.Base}}/{{esc .Target}}/move"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="revision" value="{{.Revision}}"><input type="hidden" name="dir" value="down"><button type="submit" class="secondary">Move down<span class="vh"> {{.Name}}</span></button></form>{{end}}
<form method="post" action="{{.Base}}/{{esc .Target}}/remove"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><button type="submit" class="danger">Remove<span class="vh"> {{.Name}} from saved partners</span></button></form>
</div>
<details><summary>Change nickname<span class="vh"> for {{.Name}}</span></summary>
<form method="post" action="{{.Base}}/{{esc .Target}}/nickname"><input type="hidden" name="csrf_token" value="{{.CSRF}}">
<label for="nick-{{.Path}}">Nickname</label><p class="hint" id="nick-h-{{.Path}}">Only you can see this. Up to 80 characters.</p>
<input type="text" id="nick-{{.Path}}" name="nickname" value="{{.Nickname}}" maxlength="80" aria-describedby="nick-h-{{.Path}}" autocomplete="off">
<div class="row"><button type="submit">Save nickname</button></div></form></details>
</li>{{end}}

{{define "list"}}{{template "head" "Saved trading partners"}}
<h1>Saved trading partners</h1>
<p>Only you can see this list. Saving a partner does not change credit limits, balances or reputation, and they are not told.</p>
{{template "errors" .Errors}}
{{if .Notice}}<div class="ok" role="status">{{.Notice}}</div>{{end}}
<h2 id="add-h">Add a trading partner</h2>
<form method="post" action="{{.Base}}/add" aria-labelledby="add-h">
<input type="hidden" name="csrf_token" value="{{.CSRF}}">
<label for="target">Member ID</label><p class="hint" id="target-h">Copy it from the member's profile in the directory.</p>
<input type="text" id="target" name="target" value="{{.AddTarget}}" aria-describedby="target-h" autocomplete="off" autocapitalize="none" spellcheck="false" required>
<label for="nickname" style="margin-top:.8rem">Nickname (optional)</label><p class="hint" id="nickname-h">Only you can see this. Up to 80 characters.</p>
<input type="text" id="nickname" name="nickname" value="{{.AddNickname}}" maxlength="80" aria-describedby="nickname-h" autocomplete="off">
<div class="check"><input type="checkbox" id="pinned" name="pinned" value="1"{{if .AddPinned}} checked{{end}}><label for="pinned">Pin to the top of my list</label></div>
<div class="row"><button type="submit">Save trading partner</button></div>
</form>
<h2 id="list">Your saved partners</h2>
{{if .Items}}<ul class="fav">{{range .Items}}{{template "item" .}}{{end}}</ul>
{{else}}<p>You have no saved trading partners yet.</p>{{end}}
{{if .Next}}<nav class="pager" aria-label="Pagination"><a href="{{.Base}}?cursor={{.Next}}#list">Next page</a></nav>{{end}}
<p><a href="{{.Directory}}">Find members in the directory</a></p>
{{template "foot"}}{{end}}

{{define "draft"}}{{template "head" "Review proposal draft"}}
<h1>Review proposal draft</h1>
<p>This only prepares a proposal form. Nothing has been proposed or sent yet, and no amount is filled in.</p>
<dl><dt>Recipient</dt><dd>{{.D.RecipientName}}</dd></dl>
<form method="post" action="{{.P.Action}}">
<input type="hidden" name="{{.P.CSRF}}" value="{{.CSRF}}">
<input type="hidden" name="{{.P.Recipient}}" value="{{.D.Recipient}}">
<input type="hidden" name="{{.P.IdempotencyKey}}" value="{{.D.Intent}}">
<p style="margin-top:1rem"><label for="amount">Amount</label></p><p class="hint" id="amount-h">Enter the amount for this new proposal.</p>
<input type="text" id="amount" name="{{.P.Amount}}" value="" inputmode="decimal" autocomplete="off" aria-describedby="amount-h" required>
<p style="margin-top:.8rem"><label for="description">Description (optional)</label></p><p class="hint" id="description-h">Copied from your earlier trade. You can change it. Up to 2000 characters.</p>
<textarea id="description" name="{{.P.Description}}" rows="4" maxlength="2000" aria-describedby="description-h">{{.D.Description}}</textarea>
<div class="row"><button type="submit">Continue to review proposal</button></div>
</form>
<p><a href="{{.Base}}">Back to saved trading partners</a></p>
{{template "foot"}}{{end}}

{{define "picker"}}<nav aria-labelledby="fav-picker-h" class="fav-picker"><h2 id="fav-picker-h">Saved trading partners</h2>
{{if .Items}}<ul class="fav">{{range .Items}}<li><form method="post" action="{{.Base}}/{{esc .Target}}/draft"><input type="hidden" name="csrf_token" value="{{.CSRF}}"><input type="hidden" name="request_key" value="{{.DraftKey}}"><button type="submit">{{.Name}}<span class="vh"> - start a proposal</span></button></form></li>{{end}}</ul>
<p><a href="{{.Base}}">Manage saved partners</a></p>{{else}}<p>No saved partners yet.</p>{{end}}
<p><a href="{{.Directory}}">Search the directory instead</a></p></nav>{{end}}
`))

func render(w http.ResponseWriter, status int, name string, data any) {
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, name, data); err != nil {
		http.Error(w, "Internal error.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(b.Bytes())
}

// itemViews flattens items; each carries Base/CSRF/Revision for its forms.
type itemFull struct {
	itemView
	Base, CSRF string
	Revision   int64
}

func (h *Handler) fullItems(items []favorites.View, s Session, rev int64, first, last bool) []itemFull {
	out := make([]itemFull, len(items))
	for i, v := range items {
		name := v.DisplayName
		if name == "" {
			name = v.Target
		}
		out[i] = itemFull{itemView: itemView{Target: v.Target, Path: url.PathEscape(v.Target), Name: name, Nickname: v.Nickname, DraftKey: randomKey(),
			Pinned: v.Pinned, First: first && i == 0, Last: last && i == len(items)-1}, Base: h.base, CSRF: s.CSRF, Revision: rev}
	}
	return out
}

func (h *Handler) renderList(w http.ResponseWriter, r *http.Request, s Session, status int, errs []fieldErr, notice string, mod func(*listView)) {
	cursor := r.URL.Query().Get("cursor")
	if r.Method != http.MethodGet {
		cursor = ""
	}
	v := listView{Base: h.base, Directory: h.directoryURL, CSRF: s.CSRF, Errors: errs, Notice: notice}
	pg, err := h.svc.List(r.Context(), s.Actor, favorites.ListOptions{Cursor: cursor})
	if err != nil {
		st, msg := listError(err)
		v.Errors = append(v.Errors, fieldErr{Msg: msg})
		status = st
	} else {
		v.Revision, v.Next = pg.Revision, pg.Next
		v.Items = h.fullItems(pg.Items, s, pg.Revision, cursor == "", pg.Next == "")
	}
	if mod != nil {
		mod(&v)
	}
	render(w, status, "list", v)
}

func listError(err error) (int, string) {
	switch {
	case errors.Is(err, favorites.ErrPolicy):
		return http.StatusServiceUnavailable, "Saved trading partners are temporarily unavailable. Try again later."
	case errors.Is(err, favorites.ErrUnavailable):
		return http.StatusForbidden, "Saved trading partners are not available to your account."
	}
	var ie *favorites.InputError
	if errors.As(err, &ie) {
		return http.StatusBadRequest, "The page address is not valid."
	}
	return http.StatusInternalServerError, "Something went wrong. Try again later."
}

func (h *Handler) redirect(w http.ResponseWriter, r *http.Request, notice string) {
	loc := h.base
	if _, ok := notices[notice]; ok {
		loc += "?n=" + notice
	}
	http.Redirect(w, r, loc+"#list", http.StatusSeeOther)
}

func (h *Handler) htmlList(w http.ResponseWriter, r *http.Request, s Session) {
	h.renderList(w, r, s, http.StatusOK, nil, notices[r.URL.Query().Get("n")], nil)
}

func (h *Handler) htmlAdd(w http.ResponseWriter, r *http.Request, s Session) {
	target, nick := r.PostForm.Get("target"), r.PostForm.Get("nickname")
	pinned := r.PostForm.Get("pinned") == "1"
	_, created, err := h.svc.Add(r.Context(), s.Actor, favorites.AddRequest{Target: target, Nickname: nick, Pinned: pinned})
	if err == nil {
		if created {
			h.redirect(w, r, "added")
		} else {
			h.redirect(w, r, "exists")
		}
		return
	}
	status, msg, field := http.StatusUnprocessableEntity, "", "target"
	var ie *favorites.InputError
	switch {
	case errors.As(err, &ie):
		status, msg, field = http.StatusBadRequest, "Nickname: "+ie.Reason+".", "nickname"
	case errors.Is(err, favorites.ErrUnavailable):
		msg = "That member can't be saved. Check the member ID, or search the directory."
	case errors.Is(err, favorites.ErrExists):
		status, msg = http.StatusConflict, "You already saved this member with a different nickname or pin setting. Change it from your list below."
	case errors.Is(err, favorites.ErrLimit):
		status, msg, field = http.StatusConflict, "You have reached the limit of saved partners. Remove one first.", ""
	case errors.Is(err, favorites.ErrPolicy):
		status, msg, field = http.StatusServiceUnavailable, "Saved trading partners are temporarily unavailable. Try again later.", ""
	default:
		status, msg, field = http.StatusInternalServerError, "Something went wrong. Try again later.", ""
	}
	h.renderList(w, r, s, status, []fieldErr{{Field: field, Msg: msg}}, "", func(v *listView) { v.AddTarget, v.AddNickname, v.AddPinned = target, nick, pinned })
}

// mutateErr renders the list with a status derived from err.
func (h *Handler) mutateErr(w http.ResponseWriter, r *http.Request, s Session, err error) {
	var ie *favorites.InputError
	switch {
	case errors.As(err, &ie):
		h.renderList(w, r, s, http.StatusBadRequest, []fieldErr{{Msg: "Nickname: " + ie.Reason + "."}}, "", nil)
	case errors.Is(err, favorites.ErrNotFound):
		h.renderList(w, r, s, http.StatusNotFound, []fieldErr{{Msg: "That trading partner is not in your list."}}, "", nil)
	case errors.Is(err, favorites.ErrStale), errors.Is(err, favorites.ErrInvalidOrder):
		h.renderList(w, r, s, http.StatusConflict, []fieldErr{{Msg: "Your list changed since you opened this page. It has been reloaded. Try again."}}, "", nil)
	default:
		st, msg := listError(err)
		h.renderList(w, r, s, st, []fieldErr{{Msg: msg}}, "", nil)
	}
}

func (h *Handler) htmlNickname(w http.ResponseWriter, r *http.Request, s Session) {
	if err := h.svc.SetNickname(r.Context(), s.Actor, r.PathValue("id"), r.PostForm.Get("nickname")); err != nil {
		h.mutateErr(w, r, s, err)
		return
	}
	h.redirect(w, r, "renamed")
}

func (h *Handler) htmlPin(w http.ResponseWriter, r *http.Request, s Session) {
	pin := r.PostForm.Get("pinned") == "1"
	if err := h.svc.SetPinned(r.Context(), s.Actor, r.PathValue("id"), pin); err != nil {
		h.mutateErr(w, r, s, err)
		return
	}
	if pin {
		h.redirect(w, r, "pinned")
	} else {
		h.redirect(w, r, "unpinned")
	}
}

func (h *Handler) htmlRemove(w http.ResponseWriter, r *http.Request, s Session) {
	ok, err := h.svc.Remove(r.Context(), s.Actor, r.PathValue("id"))
	if err != nil {
		h.mutateErr(w, r, s, err)
		return
	}
	if ok {
		h.redirect(w, r, "removed")
	} else {
		h.redirect(w, r, "gone")
	}
}

// allVisible pages through every currently usable favorite.
func (h *Handler) allVisible(r *http.Request, s Session) ([]favorites.View, int64, error) {
	var all []favorites.View
	cur := ""
	var rev int64
	for i := 0; i < 100; i++ {
		pg, err := h.svc.List(r.Context(), s.Actor, favorites.ListOptions{Limit: favorites.MaxPageLimit, Cursor: cur})
		if err != nil {
			return nil, 0, err
		}
		if i == 0 {
			rev = pg.Revision
		}
		all = append(all, pg.Items...)
		if pg.Next == "" {
			return all, rev, nil
		}
		cur = pg.Next
	}
	return nil, 0, errors.New("web: list did not terminate")
}

func (h *Handler) htmlMove(w http.ResponseWriter, r *http.Request, s Session) {
	id, dir := r.PathValue("id"), r.PostForm.Get("dir")
	formRev, err := strconv.ParseInt(r.PostForm.Get("revision"), 10, 64)
	if err != nil || (dir != "up" && dir != "down") {
		h.renderList(w, r, s, http.StatusBadRequest, []fieldErr{{Msg: "The request was not valid."}}, "", nil)
		return
	}
	all, rev, err := h.allVisible(r, s)
	if err != nil {
		h.mutateErr(w, r, s, err)
		return
	}
	if rev != formRev {
		h.mutateErr(w, r, s, favorites.ErrStale)
		return
	}
	idx := -1
	for i, v := range all {
		if v.Target == id {
			idx = i
		}
	}
	if idx < 0 {
		h.mutateErr(w, r, s, favorites.ErrNotFound)
		return
	}
	j := idx - 1
	if dir == "down" {
		j = idx + 1
	}
	if j < 0 || j >= len(all) || all[j].Pinned != all[idx].Pinned {
		h.redirect(w, r, "edge")
		return
	}
	all[idx], all[j] = all[j], all[idx]
	order := make([]string, len(all))
	for i, v := range all {
		order[i] = v.Target
	}
	if _, err := h.svc.Reorder(r.Context(), s.Actor, favorites.ReorderRequest{Revision: rev, Targets: order}); err != nil {
		h.mutateErr(w, r, s, err)
		return
	}
	h.redirect(w, r, "moved")
}

func (h *Handler) draftErr(w http.ResponseWriter, r *http.Request, s Session, err error) {
	switch {
	case errors.Is(err, favorites.ErrPolicy):
		h.renderList(w, r, s, http.StatusServiceUnavailable, []fieldErr{{Msg: "Saved trading partners are temporarily unavailable. Try again later."}}, "", nil)
	case errors.Is(err, favorites.ErrLimit):
		h.renderList(w, r, s, http.StatusConflict, []fieldErr{{Msg: "You have too many unfinished drafts. Try again later."}}, "", nil)
	case errors.Is(err, favorites.ErrKeyReuse), errors.Is(err, favorites.ErrNotFound), errors.Is(err, favorites.ErrUnavailable), errors.Is(err, favorites.ErrTradeUnavailable):
		h.renderList(w, r, s, http.StatusUnprocessableEntity, []fieldErr{{Msg: "A proposal can't be started with that member or trade right now."}}, "", nil)
	default:
		var ie *favorites.InputError
		if errors.As(err, &ie) {
			h.renderList(w, r, s, http.StatusBadRequest, []fieldErr{{Msg: "The request was not valid."}}, "", nil)
			return
		}
		h.renderList(w, r, s, http.StatusInternalServerError, []fieldErr{{Msg: "Something went wrong. Try again later."}}, "", nil)
	}
}

func (h *Handler) draftRedirect(w http.ResponseWriter, r *http.Request, d favorites.Draft) {
	http.Redirect(w, r, h.base+"/drafts/"+url.PathEscape(d.RequestKey), http.StatusSeeOther)
}

func (h *Handler) htmlStartDraft(w http.ResponseWriter, r *http.Request, s Session) {
	d, err := h.svc.DraftFromFavorite(r.Context(), s.Actor, r.PathValue("id"), r.PostForm.Get("request_key"))
	if err != nil {
		h.draftErr(w, r, s, err)
		return
	}
	h.draftRedirect(w, r, d)
}

// htmlRepeat is the target of a "repeat this trade" button on the host's trade
// history: fields trade_id, request_key and csrf_token.
func (h *Handler) htmlRepeat(w http.ResponseWriter, r *http.Request, s Session) {
	d, err := h.svc.RepeatTrade(r.Context(), s.Actor, r.PostForm.Get("trade_id"), r.PostForm.Get("request_key"))
	if err != nil {
		h.draftErr(w, r, s, err)
		return
	}
	h.draftRedirect(w, r, d)
}

func (h *Handler) htmlDraft(w http.ResponseWriter, r *http.Request, s Session) {
	d, err := h.svc.GetDraft(r.Context(), s.Actor, r.PathValue("key"))
	if err != nil {
		if errors.Is(err, favorites.ErrPolicy) {
			http.Error(w, "Temporarily unavailable.", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "This draft is no longer available.", http.StatusNotFound)
		return
	}
	render(w, http.StatusOK, "draft", draftView{Base: h.base, Directory: h.directoryURL, CSRF: s.CSRF, D: d, P: h.prop})
}

// Picker returns a small no-JavaScript fragment for the host's Find and Trade
// page: up to ten current favorites as one-click "start a proposal" buttons,
// plus a fallback link to directory search. It authenticates and re-reads the
// Policy like every other call; on any failure it returns the fallback link only.
func (h *Handler) Picker(r *http.Request) template.HTML {
	s, err := h.auth.Authenticate(r)
	type pickerData struct {
		Items           []itemFull
		Base, Directory string
	}
	d := pickerData{Base: h.base, Directory: h.directoryURL}
	if err == nil && s.valid() {
		if pg, err := h.svc.List(r.Context(), s.Actor, favorites.ListOptions{Limit: 10}); err == nil {
			d.Items = h.fullItems(pg.Items, s, pg.Revision, false, false)
		}
	}
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, "picker", d); err != nil {
		return template.HTML(fmt.Sprintf(`<p><a href="%s">Search the directory</a></p>`, template.HTMLEscapeString(h.directoryURL)))
	}
	return template.HTML(b.String())
}
