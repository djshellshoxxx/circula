package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Sort and scope values.
const (
	SortRelevance = "relevance"
	SortRecent    = "recent"

	ScopePublic = ""     // active, unexpired, same-network listings
	ScopeMine   = "mine" // the caller's own listings in every state
	ScopeAll    = "all"  // network admin moderation view
)

// Filters is the complete, bounded search state. It contains no secrets, no
// cursor and no identities, so its URL form is safe to bookmark and share.
type Filters struct {
	Q            string
	Category     string
	Type         string
	Kind         string
	Condition    string
	Availability string
	Area         string
	NearArea     string
	WithinKM     int
	Tag          string
	Sort         string
	Scope        string
	Limit        int
}

var allowedParams = map[string]bool{
	"q": true, "category": true, "type": true, "kind": true, "condition": true,
	"availability": true, "area": true, "near": true, "within_km": true,
	"tag": true, "sort": true, "scope": true, "limit": true, "cursor": true,
}

var secretLikeParams = []string{"token", "csrf", "session", "secret", "key", "password", "auth"}

const (
	maxQueryBytes = 200
	maxTerms      = 8
	maxTermRunes  = 64
)

// ParseFilters validates URL query values. Unknown, duplicate or secret-like
// parameters are rejected rather than ignored, so saved views never smuggle
// state. The raw cursor string (if any) is returned separately.
func ParseFilters(v url.Values, cfg Config) (Filters, string, error) {
	cfg = cfg.withDefaults()
	var f Filters
	for k, vals := range v {
		if !allowedParams[k] {
			for _, s := range secretLikeParams {
				if strings.Contains(strings.ToLower(k), s) {
					return f, "", invalid(k, "parameter not permitted in view URLs")
				}
			}
			return f, "", invalid(k, "unknown parameter")
		}
		if len(vals) != 1 {
			return f, "", invalid(k, "must appear once")
		}
	}
	get := func(k string) string { return strings.TrimSpace(v.Get(k)) }
	var err error
	f.Q = get("q")
	if len(f.Q) > maxQueryBytes || !utf8.ValidString(f.Q) {
		return f, "", invalid("q", "too long or not UTF-8")
	}
	f.Category, f.Area, f.NearArea = get("category"), get("area"), get("near")
	for k, id := range map[string]string{"category": f.Category, "area": f.Area, "near": f.NearArea} {
		if id != "" && !validID(id) {
			return f, "", invalid(k, "invalid identifier")
		}
	}
	f.Type, f.Kind, f.Condition, f.Availability = get("type"), get("kind"), get("condition"), get("availability")
	if f.Type != "" && f.Type != TypeGoods && f.Type != TypeService {
		return f, "", invalid("type", "must be goods or service")
	}
	if f.Kind != "" && f.Kind != KindOffer && f.Kind != KindRequest {
		return f, "", invalid("kind", "must be offer or request")
	}
	if f.Condition != "" && !conditions[f.Condition] {
		return f, "", invalid("condition", "unknown condition")
	}
	switch f.Availability {
	case "", AvailAvailable, AvailLimited, AvailUnavailable:
	default:
		return f, "", invalid("availability", "unknown availability")
	}
	if t := get("tag"); t != "" {
		if f.Tag, err = NormalizeTag(t); err != nil {
			return f, "", err
		}
	}
	switch f.Sort = get("sort"); f.Sort {
	case "", SortRelevance, SortRecent:
	default:
		return f, "", invalid("sort", "must be relevance or recent")
	}
	switch f.Scope = get("scope"); f.Scope {
	case ScopePublic, ScopeMine, ScopeAll:
	default:
		return f, "", invalid("scope", "unknown scope")
	}
	if s := get("within_km"); s != "" {
		n, e := strconv.Atoi(s)
		if e != nil || n < 1 || n > cfg.MaxWithinKM {
			return f, "", invalid("within_km", "out of range")
		}
		f.WithinKM = n
	}
	if (f.WithinKM != 0) != (f.NearArea != "") {
		return f, "", invalid("near", "near and within_km must be used together")
	}
	if s := get("limit"); s != "" {
		n, e := strconv.Atoi(s)
		if e != nil || n < 1 || n > cfg.MaxPage {
			return f, "", invalid("limit", "out of range")
		}
		f.Limit = n
	}
	cur := get("cursor")
	if len(cur) > 1024 {
		return f, "", invalid("cursor", "too long")
	}
	f.normalize(cfg)
	return f, cur, nil
}

func (f *Filters) normalize(cfg Config) {
	if f.Limit == 0 {
		f.Limit = cfg.DefaultPage
	}
	if f.Sort == "" {
		if len(Terms(f.Q)) > 0 {
			f.Sort = SortRelevance
		} else {
			f.Sort = SortRecent
		}
	}
	if f.Sort == SortRelevance && len(Terms(f.Q)) == 0 {
		f.Sort = SortRecent // nothing to rank by
	}
}

// Terms folds and splits free text into at most maxTerms bounded terms.
func Terms(q string) []string {
	var out []string
	for _, w := range strings.Fields(Fold(q)) {
		if utf8.RuneCountInString(w) > maxTermRunes {
			w = string([]rune(w)[:maxTermRunes])
		}
		out = append(out, w)
		if len(out) == maxTerms {
			break
		}
	}
	return out
}

// Values renders only non-default state, so equal views have equal URLs.
func (f Filters) Values(cfg Config) url.Values {
	cfg = cfg.withDefaults()
	v := url.Values{}
	set := func(k, s string) {
		if s != "" {
			v.Set(k, s)
		}
	}
	set("q", strings.TrimSpace(f.Q))
	set("category", f.Category)
	set("type", f.Type)
	set("kind", f.Kind)
	set("condition", f.Condition)
	set("availability", f.Availability)
	set("area", f.Area)
	set("near", f.NearArea)
	if f.WithinKM > 0 {
		v.Set("within_km", strconv.Itoa(f.WithinKM))
	}
	set("tag", f.Tag)
	set("scope", f.Scope)
	def := SortRecent
	if len(Terms(f.Q)) > 0 {
		def = SortRelevance
	}
	if f.Sort != def {
		set("sort", f.Sort)
	}
	if f.Limit != cfg.DefaultPage {
		v.Set("limit", strconv.Itoa(f.Limit))
	}
	return v
}

// Canonical is the sorted query string of the view (no cursor).
func (f Filters) Canonical(cfg Config) string { return f.Values(cfg).Encode() }

func (f Filters) hash(cfg Config) string {
	h := sha256.Sum256([]byte(f.Canonical(cfg)))
	return hex.EncodeToString(h[:16])
}

// validate re-checks every field so that programmatic callers get the same
// guarantees as URL parsing. It normalizes Tag.
func (f *Filters) validate(cfg Config) error {
	if len(f.Q) > maxQueryBytes || !utf8.ValidString(f.Q) {
		return invalid("q", "too long or not UTF-8")
	}
	for k, id := range map[string]string{"category": f.Category, "area": f.Area, "near": f.NearArea} {
		if id != "" && !validID(id) {
			return invalid(k, "invalid identifier")
		}
	}
	if f.Type != "" && f.Type != TypeGoods && f.Type != TypeService {
		return invalid("type", "must be goods or service")
	}
	if f.Kind != "" && f.Kind != KindOffer && f.Kind != KindRequest {
		return invalid("kind", "must be offer or request")
	}
	if f.Condition != "" && !conditions[f.Condition] {
		return invalid("condition", "unknown condition")
	}
	switch f.Availability {
	case "", AvailAvailable, AvailLimited, AvailUnavailable:
	default:
		return invalid("availability", "unknown availability")
	}
	if f.Tag != "" {
		t, err := NormalizeTag(f.Tag)
		if err != nil {
			return err
		}
		f.Tag = t
	}
	switch f.Sort {
	case "", SortRelevance, SortRecent:
	default:
		return invalid("sort", "must be relevance or recent")
	}
	switch f.Scope {
	case ScopePublic, ScopeMine, ScopeAll:
	default:
		return invalid("scope", "unknown scope")
	}
	if f.WithinKM < 0 || f.WithinKM > cfg.MaxWithinKM {
		return invalid("within_km", "out of range")
	}
	if (f.WithinKM != 0) != (f.NearArea != "") {
		return invalid("near", "near and within_km must be used together")
	}
	if f.Limit < 0 || f.Limit > cfg.MaxPage {
		return invalid("limit", "out of range")
	}
	return nil
}
