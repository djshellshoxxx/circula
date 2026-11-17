package web

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
)

type mediaJSON struct {
	ID        string `json:"id"`
	Alt       string `json:"alt"`
	MIME      string `json:"mime"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	ThumbW    int    `json:"thumb_width"`
	ThumbH    int    `json:"thumb_height"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumb_url"`
	SHA256    string `json:"sha256"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
}

type tagJSON struct {
	Name   string `json:"name"`
	Status string `json:"status,omitempty"`
}

type listingJSON struct {
	ID            string      `json:"id"`
	Kind          string      `json:"kind"`
	Title         string      `json:"title"`
	Description   string      `json:"description,omitempty"`
	Status        string      `json:"status"`
	Mine          bool        `json:"mine"`
	Type          string      `json:"type,omitempty"`
	CategoryID    string      `json:"category_id,omitempty"`
	Condition     string      `json:"condition,omitempty"`
	AreaID        string      `json:"coarse_area,omitempty"`
	Availability  string      `json:"availability,omitempty"`
	AvailableFrom string      `json:"available_from,omitempty"`
	ExpiresAt     string      `json:"expires_at,omitempty"`
	Expired       bool        `json:"expired,omitempty"`
	Stale         bool        `json:"stale,omitempty"`
	Unclassified  bool        `json:"unclassified"`
	Version       int64       `json:"version"`
	UpdatedAt     string      `json:"updated_at"`
	Tags          []tagJSON   `json:"tags"`
	Media         []mediaJSON `json:"media,omitempty"`
	Cover         *mediaJSON  `json:"cover,omitempty"`
}

func ts(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *Server) mediaURL(listing, media string, thumb bool) string {
	u := s.base + "/api/v1/listings/" + listing + "/media/" + media
	if thumb {
		u += "?variant=thumb"
	}
	return u
}

func (s *Server) toMedia(m d.MediaInfo) mediaJSON {
	return mediaJSON{ID: m.ID, Alt: m.Alt, MIME: m.MIME, Width: m.Width, Height: m.Height, ThumbW: m.ThumbWidth, ThumbH: m.ThumbHeight,
		URL: s.mediaURL(m.ListingID, m.ID, false), ThumbURL: s.mediaURL(m.ListingID, m.ID, true), SHA256: m.SHA256, Position: m.Position, CreatedAt: ts(m.Created)}
}

func (s *Server) toListing(a d.Actor, v d.ListingView) listingJSON {
	j := listingJSON{ID: v.ID, Kind: v.Kind, Title: v.Title, Description: v.Description, Status: v.Status, Mine: v.OwnerID == a.ID,
		Type: v.Type, CategoryID: v.CategoryID, Condition: v.Condition, AreaID: v.AreaID, Availability: v.Availability,
		AvailableFrom: ts(v.AvailableFrom), ExpiresAt: ts(v.ExpiresAt), Expired: v.Expired, Stale: v.Stale, Unclassified: v.Unclassified,
		Version: v.Version, UpdatedAt: ts(v.Updated), Tags: []tagJSON{}}
	for _, t := range v.Tags {
		tj := tagJSON{Name: t.Name}
		if v.CanEdit {
			tj.Status = string(t.Status)
		}
		j.Tags = append(j.Tags, tj)
	}
	for _, m := range v.Media {
		j.Media = append(j.Media, s.toMedia(m))
	}
	return j
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); ct != "application/json" {
		return &httpError{http.StatusUnsupportedMediaType, "Content-Type must be application/json"}
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, apiBodyLimit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var mbe *http.MaxBytesError
		if asMaxBytes(err, &mbe) {
			return err
		}
		return badRequest("malformed JSON body")
	}
	if dec.More() {
		return badRequest("unexpected data after JSON body")
	}
	return nil
}

func asMaxBytes(err error, target **http.MaxBytesError) bool {
	for err != nil {
		if e, ok := err.(*http.MaxBytesError); ok {
			*target = e
			return true
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

func (s *Server) apiSearch(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	f, cur, err := d.ParseFilters(r.URL.Query(), s.svc.Config())
	if err != nil {
		return &httpError{http.StatusBadRequest, friendly(err)}
	}
	res, err := s.svc.Search(r.Context(), a, f, cur)
	if err != nil {
		return err
	}
	out := struct {
		Items      []listingJSON `json:"items"`
		NextCursor string        `json:"next_cursor,omitempty"`
		SavedView  string        `json:"saved_view"`
		Limit      int           `json:"limit"`
	}{Items: []listingJSON{}, NextCursor: res.Next, SavedView: res.SavedView, Limit: res.Filters.Limit}
	for _, it := range res.Items {
		v := d.ListingView{ListingRecord: it.Listing, Meta: it.Meta, CanEdit: it.Listing.OwnerID == a.ID || a.Admin()}
		v.Unclassified = it.Meta.Version == 0 || (it.Meta.CategoryID == "" && it.Meta.Type == "")
		now := s.svc.Config().Now()
		v.Expired = !v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now)
		for _, t := range it.Tags {
			v.Tags = append(v.Tags, d.ListingTag{Name: t, Status: d.TagApproved})
		}
		j := s.toListing(a, v)
		j.Description = "" // search results stay compact
		if it.Cover != nil {
			c := s.toMedia(*it.Cover)
			j.Cover = &c
		}
		out.Items = append(out.Items, j)
	}
	body, _ := json.Marshal(out)
	sum := sha256.Sum256(body)
	etag := `W/"` + hex.EncodeToString(sum[:8]) + `"`
	w.Header().Set("ETag", etag)
	if inm := r.Header.Get("If-None-Match"); inm != "" && (inm == etag || inm == "*") {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(append(body, '\n'))
	return nil
}

func (s *Server) apiGetListing(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	v, err := s.svc.GetListing(r.Context(), a, r.PathValue("id"))
	if err != nil {
		return err
	}
	w.Header().Set("ETag", `"v`+strconv.FormatInt(v.Version, 10)+`-`+strconv.FormatInt(v.Updated.Unix(), 10)+`"`)
	s.writeJSON(w, 200, s.toListing(a, v))
	return nil
}

type saveBody struct {
	CategoryID    string   `json:"category_id"`
	Type          string   `json:"type"`
	Condition     string   `json:"condition"`
	AreaID        string   `json:"coarse_area"`
	Availability  string   `json:"availability"`
	AvailableFrom string   `json:"available_from"`
	ExpiresAt     string   `json:"expires_at"`
	Tags          []string `json:"tags"`
	Version       int64    `json:"version"`
}

func parseTime(field, v string) (time.Time, error) {
	if v == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return t, &d.Error{Kind: d.ErrInvalid, Field: field, Detail: "must be an RFC 3339 timestamp"}
	}
	return t.UTC(), nil
}

func (s *Server) apiSaveDiscovery(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b saveBody
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	from, err := parseTime("available_from", b.AvailableFrom)
	if err != nil {
		return err
	}
	exp, err := parseTime("expires_at", b.ExpiresAt)
	if err != nil {
		return err
	}
	id := r.PathValue("id")
	v, err := s.svc.UpdateDiscovery(r.Context(), a, id, d.DiscoveryInput{CategoryID: b.CategoryID, Type: b.Type, Condition: b.Condition,
		AreaID: b.AreaID, Availability: b.Availability, AvailableFrom: from, ExpiresAt: exp, Tags: b.Tags, ExpectedVersion: b.Version})
	if err != nil {
		return err
	}
	s.writeJSON(w, 200, s.toListing(a, v))
	return nil
}

func (s *Server) apiRenew(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct {
		Version int64 `json:"version"`
	}
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	v, err := s.svc.Renew(r.Context(), a, r.PathValue("id"), b.Version)
	if err != nil {
		return err
	}
	s.writeJSON(w, 200, s.toListing(a, v))
	return nil
}

// uploadFrom extracts alt text and the image reader from a multipart or raw
// image request. File name and declared image type are ignored.
func (s *Server) uploadFrom(w http.ResponseWriter, r *http.Request) (alt string, rd io.Reader, cleanup func(), err error) {
	cleanup = func() {}
	ct, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch ct {
	case "multipart/form-data":
		if r.MultipartForm == nil {
			r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxMediaBody)
			if err = r.ParseMultipartForm(1 << 20); err != nil {
				return "", nil, cleanup, err
			}
		}
		cleanup = func() { r.MultipartForm.RemoveAll() }
		files := r.MultipartForm.File["file"]
		if len(files) != 1 || len(r.MultipartForm.File) != 1 {
			return "", nil, cleanup, &d.Error{Kind: d.ErrInvalid, Field: "file", Detail: "exactly one file field named file is required"}
		}
		f, err := files[0].Open()
		if err != nil {
			return "", nil, cleanup, err
		}
		prev := cleanup
		cleanup = func() { f.Close(); prev() }
		return r.PostFormValue("alt"), f, cleanup, nil
	case "image/jpeg", "image/png":
		return r.URL.Query().Get("alt"), http.MaxBytesReader(w, r.Body, s.cfg.MaxMediaBody), cleanup, nil
	}
	return "", nil, cleanup, &httpError{http.StatusUnsupportedMediaType, "use multipart/form-data, image/jpeg or image/png"}
}

func (s *Server) apiAddMedia(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	alt, rd, cleanup, err := s.uploadFrom(w, r)
	defer cleanup()
	if err != nil {
		return err
	}
	m, err := s.svc.AddMedia(r.Context(), a, r.PathValue("id"), alt, rd)
	if err != nil {
		return err
	}
	w.Header().Set("Location", s.mediaURL(m.ListingID, m.ID, false))
	s.writeJSON(w, 201, s.toMedia(m))
	return nil
}

func (s *Server) apiGetMedia(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	variant := r.URL.Query().Get("variant")
	if variant != "" && variant != "thumb" && variant != "full" {
		return &d.Error{Kind: d.ErrInvalid, Field: "variant", Detail: "must be thumb or full"}
	}
	m, err := s.svc.OpenMedia(r.Context(), a, r.PathValue("id"), r.PathValue("mediaId"), variant == "thumb")
	if err != nil {
		return err
	}
	h := w.Header()
	h.Set("Content-Type", m.MIME)
	h.Set("Content-Length", strconv.Itoa(len(m.Data)))
	h.Set("Content-Disposition", "inline")
	h.Set("Cache-Control", "private, no-store")
	sum := sha256.Sum256(m.Data)
	h.Set("ETag", `"`+hex.EncodeToString(sum[:8])+`"`)
	w.Write(m.Data)
	return nil
}

func (s *Server) apiDeleteMedia(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	if err := s.svc.RemoveMedia(r.Context(), a, r.PathValue("id"), r.PathValue("mediaId")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type categoryJSON struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Enabled  bool   `json:"enabled"`
	Listings int    `json:"listings,omitempty"`
}

func (s *Server) apiListCategories(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	cs, err := s.svc.ListCategories(r.Context(), a)
	if err != nil {
		return err
	}
	out := []categoryJSON{}
	for _, c := range cs {
		out = append(out, categoryJSON{c.ID, c.Name, c.Kind, c.Enabled, c.Listings})
	}
	s.writeJSON(w, 200, map[string]any{"items": out})
	return nil
}

func (s *Server) apiCreateCategory(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct{ Name, Kind string }
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	c, err := s.svc.CreateCategory(r.Context(), a, b.Name, b.Kind)
	if err != nil {
		return err
	}
	s.writeJSON(w, 201, categoryJSON{c.ID, c.Name, c.Kind, c.Enabled, 0})
	return nil
}

func (s *Server) apiPatchCategory(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct {
		Name    *string `json:"name"`
		Enabled *bool   `json:"enabled"`
	}
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	if err := s.svc.UpdateCategory(r.Context(), a, r.PathValue("cid"), b.Name, b.Enabled); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type areaJSON struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Enabled bool    `json:"enabled"`
	Cell    *cellJS `json:"coarse_cell,omitempty"`
}
type cellJS struct {
	Lat10 int `json:"lat10"`
	Lon10 int `json:"lon10"`
}

func (s *Server) apiListAreas(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	as, err := s.svc.ListAreas(r.Context(), a)
	if err != nil {
		return err
	}
	out := []areaJSON{}
	for _, x := range as {
		j := areaJSON{ID: x.ID, Name: x.Name, Enabled: x.Enabled}
		if x.Cell != nil && a.Admin() {
			j.Cell = &cellJS{x.Cell.Lat10, x.Cell.Lon10}
		}
		out = append(out, j)
	}
	s.writeJSON(w, 200, map[string]any{"items": out})
	return nil
}

func (s *Server) apiCreateArea(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct {
		Name string  `json:"name"`
		Cell *cellJS `json:"coarse_cell"`
	}
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	var cell *d.GeoCell
	if b.Cell != nil {
		cell = &d.GeoCell{Lat10: b.Cell.Lat10, Lon10: b.Cell.Lon10}
	}
	ar, err := s.svc.CreateArea(r.Context(), a, b.Name, cell)
	if err != nil {
		return err
	}
	s.writeJSON(w, 201, areaJSON{ID: ar.ID, Name: ar.Name, Enabled: true})
	return nil
}

func (s *Server) apiPatchArea(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct {
		Name    *string `json:"name"`
		Enabled *bool   `json:"enabled"`
	}
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	if err := s.svc.UpdateArea(r.Context(), a, r.PathValue("aid"), b.Name, b.Enabled); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) apiListTags(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	ts_, err := s.svc.ListTags(r.Context(), a, d.TagStatus(r.URL.Query().Get("status")))
	if err != nil {
		return err
	}
	out := []map[string]any{}
	for _, t := range ts_ {
		out = append(out, map[string]any{"name": t.Name, "status": t.Status, "listings": t.Listings})
	}
	s.writeJSON(w, 200, map[string]any{"items": out})
	return nil
}

func (s *Server) apiModerateTag(w http.ResponseWriter, r *http.Request, a d.Actor) error {
	var b struct {
		Status string `json:"status"`
	}
	if err := readJSON(w, r, &b); err != nil {
		return err
	}
	tag, _ := url.PathUnescape(r.PathValue("tag"))
	if err := s.svc.ModerateTag(r.Context(), a, tag, d.TagStatus(b.Status)); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}
