package messaging

import "net/http"

// POST keeps search phrases out of URLs, history, and referrer headers.
func (h *handler) searchPage(w http.ResponseWriter, r *http.Request, a Actor, id string) {
	after, err := pageNumber(r.PostForm.Get("after"), 0)
	if err != nil {
		respondError(w, err, false)
		return
	}
	data := pageData{ID: id, Member: a.Member, Query: r.PostForm.Get("query")}
	data.Token, err = h.formToken(r, a)
	if err != nil {
		respondError(w, ErrDenied, false)
		return
	}
	data.Messages, err = h.store.Search(r.Context(), a, id, data.Query, after, 100)
	if err != nil {
		respondError(w, err, false)
		return
	}
	data.Key, err = newID()
	if err != nil {
		respondError(w, err, false)
		return
	}
	if len(data.Messages) == 100 {
		data.Next = data.Messages[99].Sequence
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = threadPage.Execute(w, data)
}
