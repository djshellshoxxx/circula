package discovery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/djshellshoxxx/circula/future/discovery/imgsan"
)

// Service implements discovery behavior over a Repository and BlobStore. It
// holds no global state, registers nothing and starts no goroutines.
type Service struct {
	repo  Repository
	blobs BlobStore
	san   *imgsan.Sanitizer
	cfg   Config
	cur   *cursors

	rlMu sync.Mutex
	rl   map[string][]time.Time
}

// NewService validates configuration. CursorKeys are mandatory.
func NewService(repo Repository, blobs BlobStore, cfg Config, img imgsan.Limits) (*Service, error) {
	cfg = cfg.withDefaults()
	cur, err := newCursors(cfg.CursorKeys, cfg.CursorTTL)
	if err != nil {
		return nil, err
	}
	if repo == nil || blobs == nil {
		return nil, invalid("repo", "repository and blob store are required")
	}
	return &Service{repo: repo, blobs: blobs, san: imgsan.New(img), cfg: cfg, cur: cur, rl: map[string][]time.Time{}}, nil
}

// ImageLimits returns the effective image budgets.
func (s *Service) ImageLimits() imgsan.Limits { return s.san.Limits() }

// Config returns the effective configuration.
func (s *Service) Config() Config { return s.cfg }

func (s *Service) now() time.Time { return s.cfg.Now().UTC() }

func (s *Service) authz(a Actor) error {
	if !a.valid() {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireAdmin(a Actor) error {
	if err := s.authz(a); err != nil {
		return err
	}
	if !a.Admin() {
		return ErrForbidden
	}
	return nil
}

// SyncListing is the trusted adapter hook that mirrors a core listing into the
// read model. It is not reachable from HTTP clients.
func (s *Service) SyncListing(ctx context.Context, rec ListingRecord) error {
	if !validID(rec.ID) || !validID(rec.Network) || !validID(rec.OwnerID) {
		return invalid("id", "invalid identifier")
	}
	switch rec.Status {
	case StatusActive, StatusPaused, StatusArchived, StatusSuspended:
	default:
		return invalid("status", "unknown status")
	}
	if rec.Kind != KindOffer && rec.Kind != KindRequest {
		return invalid("kind", "must be offer or request")
	}
	if len(rec.Title) == 0 || len(rec.Title) > 200 || len(rec.Description) > 8000 {
		return invalid("title", "length out of range")
	}
	if rec.Created.IsZero() || rec.Updated.IsZero() {
		return invalid("updated", "timestamps required")
	}
	return s.repo.UpsertListing(ctx, rec)
}

func (s *Service) decorate(a Actor, v ListingView, now time.Time) ListingView {
	own := v.OwnerID == a.ID
	v.CanEdit = own || a.Admin()
	v.Expired = !v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now)
	v.Stale = v.Expired || (v.Status == StatusActive && now.Sub(v.Updated) > s.cfg.StaleAfter)
	v.Unclassified = v.Version == 0 || (v.CategoryID == "" && v.Type == "")
	if !v.CanEdit {
		tags := v.Tags[:0:0]
		for _, t := range v.Tags {
			if t.Status == TagApproved {
				tags = append(tags, t)
			}
		}
		v.Tags = tags
	}
	return v
}

func publiclyVisible(v ListingView, now time.Time) bool {
	if v.Status != StatusActive || !v.OwnerActive {
		return false
	}
	if !v.ExpiresAt.IsZero() && !v.ExpiresAt.After(now) {
		return false
	}
	if !v.AvailableFrom.IsZero() && v.AvailableFrom.After(now) {
		return false
	}
	return true
}

func (s *Service) visible(a Actor, v ListingView, now time.Time) bool {
	return v.OwnerID == a.ID || a.Admin() || publiclyVisible(v, now)
}

// GetListing returns a listing the actor may see. Foreign-network, hidden,
// archived, suspended and expired listings are indistinguishable from missing
// for ordinary members.
func (s *Service) GetListing(ctx context.Context, a Actor, id string) (ListingView, error) {
	if err := s.authz(a); err != nil {
		return ListingView{}, err
	}
	if !validID(id) {
		return ListingView{}, ErrNotFound
	}
	v, err := s.repo.GetListing(ctx, a.Network, id)
	if err != nil {
		return ListingView{}, err
	}
	now := s.now()
	if !s.visible(a, v, now) {
		return ListingView{}, ErrNotFound
	}
	// Only active media is ever exposed in a view.
	media := v.Media[:0:0]
	for _, m := range v.Media {
		if m.Status == MediaActive {
			media = append(media, m)
		}
	}
	v.Media = media
	return s.decorate(a, v, now), nil
}

// SearchResult is one authorized page.
type SearchResult struct {
	Items      []SearchItem
	Next       string // signed cursor, empty on the last page
	Filters    Filters
	SavedView  string // canonical query string without cursor
	Snapshot   int64
	Classified int
}

// Search runs a filtered, authorized, snapshot-stable page query.
func (s *Service) Search(ctx context.Context, a Actor, f Filters, cursor string) (SearchResult, error) {
	if err := s.authz(a); err != nil {
		return SearchResult{}, err
	}
	if err := f.validate(s.cfg); err != nil {
		return SearchResult{}, err
	}
	f.normalize(s.cfg)
	if f.Scope == ScopeAll && !a.Admin() {
		return SearchResult{}, ErrForbidden
	}
	now := s.now()
	p := SearchParams{
		Network: a.Network, Viewer: a.ID, Now: now, Scope: f.Scope,
		Terms: Terms(f.Q), Category: f.Category, Type: f.Type, Kind: f.Kind,
		Condition: f.Condition, Availability: f.Availability, Tag: f.Tag,
		Sort: f.Sort, Limit: f.Limit,
	}
	switch {
	case f.NearArea != "":
		ids, err := s.areasWithin(ctx, a.Network, f.NearArea, f.WithinKM)
		if err != nil {
			return SearchResult{}, err
		}
		p.AreaIDs, p.AreaFilter = ids, true
		if f.Area != "" {
			p.AreaIDs = intersect(p.AreaIDs, []string{f.Area})
		}
	case f.Area != "":
		p.AreaIDs, p.AreaFilter = []string{f.Area}, true
	}
	fh := f.hash(s.cfg)
	if cursor != "" {
		snap, pos, err := s.cur.open(cursor, a, fh, now)
		if err != nil {
			return SearchResult{}, err
		}
		p.After, p.Snapshot = &pos, snap
	}
	page, err := s.repo.Search(ctx, p)
	if err != nil {
		return SearchResult{}, err
	}
	res := SearchResult{Items: page.Items, Filters: f, SavedView: f.Canonical(s.cfg), Snapshot: page.Snapshot}
	if page.Next != nil {
		res.Next = s.cur.seal(a, fh, page.Snapshot, *page.Next, now)
	}
	return res, nil
}

func intersect(a, b []string) []string {
	set := map[string]bool{}
	for _, x := range a {
		set[x] = true
	}
	var out []string
	for _, x := range b {
		if set[x] {
			out = append(out, x)
		}
	}
	return out
}

// areasWithin returns the ids of enabled areas whose coarse centroid lies
// within km of the origin area's centroid (the origin included). Only
// network-admin-defined coarse cells are used; no member location is involved.
func (s *Service) areasWithin(ctx context.Context, network, origin string, km int) ([]string, error) {
	areas, err := s.repo.ListAreas(ctx, network, false)
	if err != nil {
		return nil, err
	}
	var o *Area
	for i := range areas {
		if areas[i].ID == origin {
			o = &areas[i]
		}
	}
	if o == nil || o.Cell == nil {
		return nil, invalid("near", "unknown area or area has no coarse centroid")
	}
	out := []string{}
	for _, ar := range areas {
		if ar.Cell != nil && distanceKM(*o.Cell, *ar.Cell) <= float64(km) {
			out = append(out, ar.ID)
		}
	}
	sort.Strings(out)
	if len(out) > 500 {
		out = out[:500]
	}
	return out, nil
}

func distanceKM(a, b GeoCell) float64 {
	const r = 6371.0
	rad := func(t int) float64 { return float64(t) / 10 * math.Pi / 180 }
	dLat, dLon := rad(b.Lat10)-rad(a.Lat10), rad(b.Lon10)-rad(a.Lon10)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(rad(a.Lat10))*math.Cos(rad(b.Lat10))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(h)))
}

// DiscoveryInput is the owner-editable discovery metadata.
type DiscoveryInput struct {
	CategoryID    string
	Type          string
	Condition     string
	AreaID        string
	Availability  string
	AvailableFrom time.Time
	ExpiresAt     time.Time
	Tags          []string
	// ExpectedVersion must equal the version last read (0 for a listing
	// with no discovery metadata yet).
	ExpectedVersion int64
}

// UpdateDiscovery validates and saves classification, tags, area and
// availability. Owner or network admin only.
func (s *Service) UpdateDiscovery(ctx context.Context, a Actor, id string, in DiscoveryInput) (ListingView, error) {
	v, err := s.ownerOrAdminListing(ctx, a, id)
	if err != nil {
		return ListingView{}, err
	}
	if v.Status == StatusArchived && !a.Admin() {
		return ListingView{}, invalid("status", "archived listings cannot be edited")
	}
	now := s.now()
	if in.CategoryID != "" && !validID(in.CategoryID) {
		return ListingView{}, invalid("category", "invalid identifier")
	}
	if in.AreaID != "" && !validID(in.AreaID) {
		return ListingView{}, invalid("area", "invalid identifier")
	}
	switch in.Type {
	case "", TypeGoods, TypeService:
	default:
		return ListingView{}, invalid("type", "must be goods or service")
	}
	if in.Condition != "" {
		if !conditions[in.Condition] {
			return ListingView{}, invalid("condition", "unknown condition")
		}
		if in.Type != TypeGoods {
			return ListingView{}, invalid("condition", "condition applies to goods only")
		}
	}
	switch in.Availability {
	case "", AvailAvailable, AvailLimited, AvailUnavailable:
	default:
		return ListingView{}, invalid("availability", "unknown availability")
	}
	if !in.ExpiresAt.IsZero() {
		if !in.ExpiresAt.After(now) {
			return ListingView{}, invalid("expires_at", "must be in the future")
		}
		if in.ExpiresAt.After(now.Add(s.cfg.MaxExpiry)) {
			return ListingView{}, invalid("expires_at", "too far in the future")
		}
		if !in.AvailableFrom.IsZero() && !in.AvailableFrom.Before(in.ExpiresAt) {
			return ListingView{}, invalid("available_from", "must precede expiry")
		}
	}
	if !in.AvailableFrom.IsZero() && in.AvailableFrom.After(now.Add(s.cfg.MaxExpiry)) {
		return ListingView{}, invalid("available_from", "too far in the future")
	}
	tags, err := s.normalizeTags(in.Tags)
	if err != nil {
		return ListingView{}, err
	}
	m := Meta{CategoryID: in.CategoryID, Type: in.Type, Condition: in.Condition, AreaID: in.AreaID,
		Availability: in.Availability, AvailableFrom: in.AvailableFrom.UTC(), ExpiresAt: in.ExpiresAt.UTC()}
	if _, err := s.repo.SaveDiscovery(ctx, SaveInput{Network: a.Network, ListingID: id, ActorID: a.ID,
		ActorIsAdmin: a.Admin(), Meta: m, Tags: tags, ExpectedVersion: in.ExpectedVersion, MaxNetworkTags: s.cfg.MaxTagsPerNetwork, Now: now}); err != nil {
		return ListingView{}, err
	}
	return s.GetListing(ctx, a, id)
}

func (s *Service) normalizeTags(raw []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, r := range raw {
		t, err := NormalizeTag(r)
		if err != nil {
			return nil, err
		}
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	if len(out) > s.cfg.MaxTagsPerListing {
		return nil, invalid("tags", "too many tags")
	}
	sort.Strings(out)
	return out, nil
}

func (s *Service) ownerOrAdminListing(ctx context.Context, a Actor, id string) (ListingView, error) {
	if err := s.authz(a); err != nil {
		return ListingView{}, err
	}
	if !validID(id) {
		return ListingView{}, ErrNotFound
	}
	v, err := s.repo.GetListing(ctx, a.Network, id)
	if err != nil {
		return ListingView{}, err
	}
	if v.OwnerID != a.ID && !a.Admin() {
		// Indistinguishable from missing unless the listing is publicly visible.
		if publiclyVisible(v, s.now()) {
			return ListingView{}, ErrForbidden
		}
		return ListingView{}, ErrNotFound
	}
	return v, nil
}

// Renew extends the listing's expiry without touching the underlying core
// listing identity or any transaction that references it.
func (s *Service) Renew(ctx context.Context, a Actor, id string, expectedVersion int64) (ListingView, error) {
	v, err := s.ownerOrAdminListing(ctx, a, id)
	if err != nil {
		return ListingView{}, err
	}
	if v.Status != StatusActive && v.Status != StatusPaused {
		return ListingView{}, invalid("status", "only active or paused listings can be renewed")
	}
	now := s.now()
	base := now
	if v.ExpiresAt.After(now) {
		base = v.ExpiresAt
	}
	exp := base.Add(s.cfg.RenewExtend)
	if limit := now.Add(s.cfg.MaxExpiry); exp.After(limit) {
		exp = limit
	}
	if _, err := s.repo.Renew(ctx, a.Network, id, a.ID, expectedVersion, exp.UTC(), now); err != nil {
		return ListingView{}, err
	}
	return s.GetListing(ctx, a, id)
}

// --- media ---

func (s *Service) allowUpload(actor string, now time.Time) bool {
	s.rlMu.Lock()
	defer s.rlMu.Unlock()
	cut := now.Add(-time.Hour)
	ts := s.rl[actor][:0]
	for _, t := range s.rl[actor] {
		if t.After(cut) {
			ts = append(ts, t)
		}
	}
	if len(ts) >= s.cfg.UploadsPerHour {
		s.rl[actor] = ts
		return false
	}
	s.rl[actor] = append(ts, now)
	return true
}

func networkDir(network string) string {
	h := sha256.Sum256([]byte(network))
	return hex.EncodeToString(h[:8])
}

// AddMedia sanitizes and stores one owner photo. The client's file name and
// content type are never consulted.
func (s *Service) AddMedia(ctx context.Context, a Actor, listingID, alt string, r io.Reader) (MediaInfo, error) {
	v, err := s.ownerOrAdminListing(ctx, a, listingID)
	if err != nil {
		return MediaInfo{}, err
	}
	if v.OwnerID != a.ID {
		return MediaInfo{}, ErrForbidden
	}
	if v.Status != StatusActive && v.Status != StatusPaused {
		return MediaInfo{}, invalid("status", "photos can only be added to active or paused listings")
	}
	alt, err = cleanText("alt", alt, 1, 250)
	if err != nil {
		return MediaInfo{}, err
	}
	now := s.now()
	if !s.allowUpload(a.ID, now) {
		return MediaInfo{}, ErrRateLimited
	}
	res, err := s.san.Sanitize(ctx, r)
	if err != nil {
		return MediaInfo{}, mapImageErr(err)
	}
	id := s.cfg.NewID("med_")
	ext := ".jpg"
	if res.Full.MIME == "image/png" {
		ext = ".png"
	}
	dir := "media/" + networkDir(a.Network) + "/"
	m := MediaInfo{
		ID: id, Network: a.Network, ListingID: listingID, OwnerID: a.ID,
		SHA256: res.SHA256, MIME: res.Full.MIME, Alt: alt,
		Width: res.Full.Width, Height: res.Full.Height,
		ThumbWidth: res.Thumb.Width, ThumbHeight: res.Thumb.Height,
		Bytes: int64(len(res.Full.Data)), ThumbBytes: int64(len(res.Thumb.Data)),
		StoragePath: dir + id + ext, ThumbPath: dir + id + ".thumb" + ext,
		Status: MediaPending, Created: now,
	}
	if err := s.repo.ReserveMedia(ctx, m, MediaQuota{MaxCount: s.cfg.MaxPhotos, MaxBytes: s.cfg.MaxListingBytes}); err != nil {
		return MediaInfo{}, err
	}
	fail := func(cause error) (MediaInfo, error) {
		// Hand the pending row to the retention job; best-effort blob cleanup.
		_ = s.blobs.Delete(context.WithoutCancel(ctx), m.StoragePath)
		_ = s.blobs.Delete(context.WithoutCancel(ctx), m.ThumbPath)
		_ = s.repo.RemoveMedia(context.WithoutCancel(ctx), a.Network, listingID, id, a.ID, now, now)
		return MediaInfo{}, cause
	}
	if err := s.blobs.Put(ctx, m.StoragePath, res.Full.Data); err != nil {
		return fail(err)
	}
	if err := s.blobs.Put(ctx, m.ThumbPath, res.Thumb.Data); err != nil {
		return fail(err)
	}
	if err := s.repo.ActivateMedia(ctx, a.Network, listingID, id, now); err != nil {
		return fail(err)
	}
	m.Status = MediaActive
	return m, nil
}

func mapImageErr(err error) error {
	switch {
	case errors.Is(err, imgsan.ErrBusy):
		return ErrRateLimited
	case errors.Is(err, imgsan.ErrTooLarge), errors.Is(err, imgsan.ErrDimensions), errors.Is(err, imgsan.ErrOutputTooLarge):
		return &Error{Kind: ErrQuota, Field: "file", Detail: "image is too large"}
	case errors.Is(err, imgsan.ErrEmpty), errors.Is(err, imgsan.ErrUnsupportedType), errors.Is(err, imgsan.ErrCorrupt),
		errors.Is(err, imgsan.ErrTrailingData), errors.Is(err, imgsan.ErrSuspiciousMetadata):
		return &Error{Kind: ErrInvalid, Field: "file", Detail: "not an acceptable JPEG or PNG image"}
	}
	return err
}

// RemoveMedia hides a photo immediately; blobs are reclaimed by RunRetention
// after the retention period. Owner or network admin only.
func (s *Service) RemoveMedia(ctx context.Context, a Actor, listingID, mediaID string) error {
	if _, err := s.ownerOrAdminListing(ctx, a, listingID); err != nil {
		return err
	}
	if !validID(mediaID) {
		return ErrNotFound
	}
	now := s.now()
	return s.repo.RemoveMedia(ctx, a.Network, listingID, mediaID, a.ID, now, now.Add(s.cfg.MediaRetention))
}

// MediaBlob is a servable sanitized image.
type MediaBlob struct {
	Data   []byte
	MIME   string
	SHA256 string
	Alt    string
}

// OpenMedia re-authorizes on every read: viewer must be an active member of
// the same network, the listing must be visible to them right now (not
// archived, suspended or expired unless owner/admin) and the photo active.
func (s *Service) OpenMedia(ctx context.Context, a Actor, listingID, mediaID string, thumb bool) (MediaBlob, error) {
	if _, err := s.GetListing(ctx, a, listingID); err != nil {
		return MediaBlob{}, err
	}
	if !validID(mediaID) {
		return MediaBlob{}, ErrNotFound
	}
	m, err := s.repo.GetMedia(ctx, a.Network, listingID, mediaID)
	if err != nil {
		return MediaBlob{}, err
	}
	if m.Status != MediaActive {
		return MediaBlob{}, ErrNotFound
	}
	key := m.StoragePath
	if thumb {
		key = m.ThumbPath
	}
	data, err := s.blobs.Get(ctx, key)
	if err != nil {
		return MediaBlob{}, err
	}
	if !thumb {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != m.SHA256 {
			return MediaBlob{}, errors.New("discovery: stored media failed integrity check")
		}
	}
	return MediaBlob{Data: data, MIME: m.MIME, SHA256: m.SHA256, Alt: m.Alt}, nil
}

// RetentionReport summarizes one retention run.
type RetentionReport struct {
	Claimed, RowsDeleted, BlobsDeleted int
	Errors                             []error
}

// RunRetention reclaims removed, archived-listing and abandoned media whose
// retention elapsed. It must be called explicitly (no scheduler is started).
// It is idempotent and crash-safe: rows are first marked deleting, then blobs
// are deleted (unless another row still references the path), then rows.
func (s *Service) RunRetention(ctx context.Context) (RetentionReport, error) {
	var rep RetentionReport
	rows, err := s.repo.ClaimGarbage(ctx, GCParams{Now: s.now(), ArchivedRetention: s.cfg.MediaRetention, PendingTTL: s.cfg.PendingTTL, Limit: 500})
	if err != nil {
		return rep, err
	}
	rep.Claimed = len(rows)
	for _, m := range rows {
		ok := true
		for _, p := range []string{m.StoragePath, m.ThumbPath} {
			ref, err := s.repo.PathReferenced(ctx, p, m.ID)
			if err != nil {
				rep.Errors = append(rep.Errors, err)
				ok = false
				continue
			}
			if ref {
				continue
			}
			if err := s.blobs.Delete(ctx, p); err != nil {
				rep.Errors = append(rep.Errors, err)
				ok = false
				continue
			}
			rep.BlobsDeleted++
		}
		if ok {
			if err := s.repo.DeleteMediaRow(ctx, m.ID); err != nil {
				rep.Errors = append(rep.Errors, err)
				continue
			}
			rep.RowsDeleted++
		}
	}
	return rep, nil
}

// --- taxonomy ---

// DefaultCategories seeds a controlled starting taxonomy.
var DefaultCategories = []struct{ Name, Kind string }{
	{"Food and garden", TypeGoods}, {"Household", TypeGoods}, {"Tools and equipment", TypeGoods},
	{"Clothing", TypeGoods}, {"Books and media", TypeGoods}, {"Electronics", TypeGoods},
	{"Furniture", TypeGoods}, {"Repairs and trades", TypeService}, {"Care and wellbeing", TypeService},
	{"Transport and moving", TypeService}, {"Teaching and skills", TypeService}, {"Other", TypeAny},
}

// SeedDefaultCategories adds the defaults not already present. Admin only.
func (s *Service) SeedDefaultCategories(ctx context.Context, a Actor) (int, error) {
	if err := s.requireAdmin(a); err != nil {
		return 0, err
	}
	n := 0
	for _, c := range DefaultCategories {
		_, err := s.CreateCategory(ctx, a, c.Name, c.Kind)
		switch {
		case err == nil:
			n++
		case errors.Is(err, ErrDuplicate):
		default:
			return n, err
		}
	}
	return n, nil
}

// CreateCategory adds a network category. Admin only.
func (s *Service) CreateCategory(ctx context.Context, a Actor, name, kind string) (Category, error) {
	if err := s.requireAdmin(a); err != nil {
		return Category{}, err
	}
	name, err := cleanText("name", name, 2, 40)
	if err != nil {
		return Category{}, err
	}
	switch kind {
	case TypeGoods, TypeService, TypeAny:
	default:
		return Category{}, invalid("kind", "must be goods, service or any")
	}
	c := Category{ID: s.cfg.NewID("cat_"), Network: a.Network, Name: name, Kind: kind, Enabled: true, Created: s.now()}
	if err := s.repo.CreateCategory(ctx, c, a.ID, s.cfg.MaxCategories); err != nil {
		return Category{}, err
	}
	return c, nil
}

// UpdateCategory renames and/or enables/disables a category. Admin only.
// Disabling never removes the category from existing listings.
func (s *Service) UpdateCategory(ctx context.Context, a Actor, id string, name *string, enabled *bool) error {
	if err := s.requireAdmin(a); err != nil {
		return err
	}
	if !validID(id) {
		return ErrNotFound
	}
	if name != nil {
		n, err := cleanText("name", *name, 2, 40)
		if err != nil {
			return err
		}
		name = &n
	}
	return s.repo.UpdateCategory(ctx, a.Network, id, name, enabled, a.ID)
}

// ListCategories returns enabled categories for members; admins see all.
func (s *Service) ListCategories(ctx context.Context, a Actor) ([]Category, error) {
	if err := s.authz(a); err != nil {
		return nil, err
	}
	return s.repo.ListCategories(ctx, a.Network, a.Admin())
}

// CreateArea adds a coarse area. Cell may be nil. Admin only.
func (s *Service) CreateArea(ctx context.Context, a Actor, name string, cell *GeoCell) (Area, error) {
	if err := s.requireAdmin(a); err != nil {
		return Area{}, err
	}
	name, err := cleanText("name", name, 2, 60)
	if err != nil {
		return Area{}, err
	}
	if cell != nil && (cell.Lat10 < -900 || cell.Lat10 > 900 || cell.Lon10 < -1800 || cell.Lon10 > 1800) {
		return Area{}, invalid("cell", "out of range")
	}
	ar := Area{ID: s.cfg.NewID("area_"), Network: a.Network, Name: name, Enabled: true, Cell: cell}
	if err := s.repo.CreateArea(ctx, ar, a.ID, s.cfg.MaxAreas); err != nil {
		return Area{}, err
	}
	return ar, nil
}

// UpdateArea renames and/or enables an area. Admin only.
func (s *Service) UpdateArea(ctx context.Context, a Actor, id string, name *string, enabled *bool) error {
	if err := s.requireAdmin(a); err != nil {
		return err
	}
	if !validID(id) {
		return ErrNotFound
	}
	if name != nil {
		n, err := cleanText("name", *name, 2, 60)
		if err != nil {
			return err
		}
		name = &n
	}
	return s.repo.UpdateArea(ctx, a.Network, id, name, enabled, a.ID)
}

// ListAreas returns enabled areas (admins: all).
func (s *Service) ListAreas(ctx context.Context, a Actor) ([]Area, error) {
	if err := s.authz(a); err != nil {
		return nil, err
	}
	return s.repo.ListAreas(ctx, a.Network, a.Admin())
}

// ListTags lists tags. Members see only approved tags; admins any status.
func (s *Service) ListTags(ctx context.Context, a Actor, status TagStatus) ([]Tag, error) {
	if err := s.authz(a); err != nil {
		return nil, err
	}
	if !a.Admin() {
		status = TagApproved
	}
	switch status {
	case "", TagPending, TagApproved, TagRejected:
	default:
		return nil, invalid("status", "unknown status")
	}
	return s.repo.ListTags(ctx, a.Network, status, 500)
}

// ModerateTag approves or rejects a tag network-wide. Admin only.
func (s *Service) ModerateTag(ctx context.Context, a Actor, tag string, status TagStatus) error {
	if err := s.requireAdmin(a); err != nil {
		return err
	}
	if status != TagApproved && status != TagRejected && status != TagPending {
		return invalid("status", "unknown status")
	}
	t, err := NormalizeTag(tag)
	if err != nil {
		return err
	}
	return s.repo.ModerateTag(ctx, a.Network, t, status, a.ID, s.now())
}

// Audit lists recent audit entries. Admin only.
func (s *Service) Audit(ctx context.Context, a Actor, limit int) ([]AuditEntry, error) {
	if err := s.requireAdmin(a); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	return s.repo.Audit(ctx, a.Network, limit)
}

// Repo exposes the repository for host adapters (idempotency, etc).
func (s *Service) Repo() Repository { return s.repo }
