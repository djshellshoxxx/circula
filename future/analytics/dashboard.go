package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Posting and Balance are trusted, transactionally consistent ledger snapshots.
// Source must read under one database snapshot, never accept these from clients.
type Posting struct {
	ID, Network, Trade, Member string
	Delta                      int64
}
type Balance struct {
	Member  string
	Amount  int64
	Network string
	System  bool
}
type Completed struct {
	Trade
	ReversalOf string
	Kind       string
}
type Offer struct {
	Member      string
	Active      bool
	Updated     time.Time
	Network, ID string
	Visible     bool
}
type Snapshot struct {
	Revision string
	Scale    int
	AsOf     time.Time
	Trades   []Completed
	Postings []Posting
	Balances []Balance
	Offers   []Offer
}
type Actor struct{ Network, Member string }

// Source must enforce fresh actor/network membership and admin permission inside
// the same read-only transaction as all facts. Read is half-open (< cutoff).
// Balances are authoritative ledger-derived stock at cutoff, not period flow.
// Every relevant posting kind must be supplied for own balance reconstruction,
// including originals needed by partial reversals. Completed means an immutable
// posting exists, never a current status filter that drops later-reversed trades.
// Account System classification must be immutable. Never accept client snapshots.
type Source interface {
	Read(context.Context, Actor, bool, time.Time) (Snapshot, error)
}
type Period struct {
	Label    string
	From, To time.Time
}
type Flow struct {
	Period                     Period
	Suppressed                 bool
	Trades, Traders, Reversals int
	Gross, Reversed, Net       int64 `json:",string"`
}
type Distribution struct {
	Suppressed                                       bool
	PositiveAccounts, NegativeAccounts, ZeroAccounts int
	Positive, Negative                               int64 `json:",string"`
}
type Dashboard struct {
	Revision, Timezone string
	Scale              int
	AsOf               time.Time
	Flows              []Flow
	Stock              Distribution
	DormantSuppressed  bool
	DormantOffers      int
	DormantDays        int
}
type DashboardService struct {
	// Granularity is fixed per network by the host; default month. Admin callers
	// cannot combine overlapping weekly/monthly releases to bypass suppression.
	Granularity string
	Source      Source
	Now         func() time.Time
	Zone        *time.Location
	K           int
	DormantDays int
}

// Periods only returns completed calendar periods. Fixed non-overlapping periods
// and no caller-chosen slices reduce opportunities for differencing queries.
func Periods(now time.Time, zone *time.Location, kind string, count int) ([]Period, error) {
	if zone == nil || count < 1 || count > 12 {
		return nil, errors.New("invalid period request")
	}
	local := now.In(zone)
	end := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	switch kind {
	case "month":
		end = time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, zone)
	case "week":
		end = end.AddDate(0, 0, -(int(end.Weekday())+6)%7)
	default:
		return nil, errors.New("period must be week or month")
	}
	out := make([]Period, count)
	for i := count - 1; i >= 0; i-- {
		start := end.AddDate(0, -1, 0)
		if kind == "week" {
			start = end.AddDate(0, 0, -7)
		}
		out[i] = Period{start.Format("2006-01-02") + " / " + end.Format("2006-01-02"), start, end}
		end = start
	}
	return out, nil
}
func plus(a, b int64) (int64, error) {
	if b > 0 && a > math.MaxInt64-b || b < 0 && a < math.MinInt64-b {
		return 0, errors.New("amount overflow")
	}
	return a + b, nil
}
func eventKind(t Completed) string {
	if t.Kind != "" {
		return t.Kind
	}
	if t.ReversalOf != "" {
		return "reversal"
	}
	return "trade"
}
func identifier(value string) bool {
	return value != "" && len(value) <= 256 && utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}
func validate(network string, s Snapshot) error {
	if s.Scale < 0 || s.Scale > 8 || len(s.Revision) > 256 || s.Revision == "" || s.AsOf.IsZero() || len(s.Trades) > 100000 || len(s.Postings) > 200000 || len(s.Balances) > 100000 || len(s.Offers) > 100000 {
		return errors.New("invalid snapshot bounds")
	}
	trades := map[string]Completed{}
	for _, t := range s.Trades {
		if t.Network != network || !t.Completed || !identifier(t.ID) || t.Amount <= 0 || !identifier(t.Payer) || !identifier(t.Payee) || t.Payer == t.Payee || trades[t.ID].ID != "" || t.PostedAt.IsZero() || !t.PostedAt.Before(s.AsOf) {
			return errors.New("invalid completed transaction")
		}
		kind := eventKind(t)
		if kind != "trade" && kind != "reversal" && kind != "grant" && kind != "adjustment" {
			return errors.New("unknown posting kind")
		}
		if (kind == "reversal") != (t.ReversalOf != "") {
			return errors.New("invalid reversal kind")
		}
		trades[t.ID] = t
	}
	entries := map[string][]Posting{}
	seen := map[string]bool{}
	for _, p := range s.Postings {
		if p.Network != network || !identifier(p.ID) || seen[p.ID] || trades[p.Trade].ID == "" {
			return errors.New("invalid ledger posting")
		}
		seen[p.ID] = true
		entries[p.Trade] = append(entries[p.Trade], p)
	}
	reversed := map[string]int64{}
	for _, t := range s.Trades {
		es := entries[t.ID]
		if len(es) != 2 {
			return errors.New("transaction must have two postings")
		}
		debit, credit := false, false
		for _, p := range es {
			if p.Member == t.Payer && p.Delta == -t.Amount {
				debit = true
			}
			if p.Member == t.Payee && p.Delta == t.Amount {
				credit = true
			}
		}
		if !debit || !credit {
			return errors.New("ledger mismatch")
		}
		if t.ReversalOf != "" {
			original, ok := trades[t.ReversalOf]
			if !ok || original.ReversalOf != "" || t.Amount > original.Amount-reversed[original.ID] || original.Payer != t.Payee || original.Payee != t.Payer || original.PostedAt.After(t.PostedAt) {
				return errors.New("invalid reversal")
			}
			reversed[original.ID] += t.Amount
		}
	}
	members := map[string]bool{}
	sum := new(big.Int)
	for _, b := range s.Balances {
		if b.Network != network || !identifier(b.Member) || members[b.Member] {
			return errors.New("invalid balance snapshot")
		}
		members[b.Member] = true
		sum.Add(sum, big.NewInt(b.Amount))
	}
	if sum.Sign() != 0 {
		return errors.New("stock not sum-zero")
	}
	for _, t := range s.Trades {
		if !members[t.Payer] || !members[t.Payee] {
			return errors.New("unknown posting account")
		}
	}
	seenOffers := map[string]bool{}
	for _, o := range s.Offers {
		if o.Network != network || !identifier(o.ID) || seenOffers[o.ID] || !members[o.Member] || o.Updated.IsZero() || o.Updated.After(s.AsOf) {
			return errors.New("invalid offer snapshot")
		}
		seenOffers[o.ID] = true
	}
	return nil
}
func (s DashboardService) load(ctx context.Context, a Actor, admin bool, kind string, count int) (Snapshot, []Period, error) {
	if !identifier(a.Network) || !identifier(a.Member) || s.DormantDays < 0 || s.DormantDays > 365 || s.Source == nil || s.Now == nil || s.Zone == nil || s.K < 5 {
		return Snapshot{}, nil, errors.New("invalid analytics context")
	}
	if err := ctx.Err(); err != nil {
		return Snapshot{}, nil, err
	}
	now := s.Now()
	if now.IsZero() {
		return Snapshot{}, nil, errors.New("invalid clock")
	}
	periods, err := Periods(now, s.Zone, kind, count)
	if err != nil {
		return Snapshot{}, nil, err
	}
	cutoff := now
	if admin {
		granularity := s.Granularity
		if granularity == "" {
			granularity = "month"
		}
		if kind != granularity {
			return Snapshot{}, nil, errors.New("unapproved aggregate granularity")
		}
		cutoff = periods[len(periods)-1].To
	}
	snap, err := s.Source.Read(ctx, a, admin, cutoff)
	if err != nil {
		return Snapshot{}, nil, err
	}
	if err = ctx.Err(); err != nil {
		return Snapshot{}, nil, err
	}
	if !snap.AsOf.Equal(cutoff) {
		return Snapshot{}, nil, errors.New("future snapshot")
	}
	if err = validate(a.Network, snap); err != nil {
		return Snapshot{}, nil, err
	}
	if periods[len(periods)-1].To.After(snap.AsOf) {
		return Snapshot{}, nil, errors.New("incomplete period snapshot")
	}
	return snap, periods, nil
}

// dominance uses each account's total contribution, so splitting into many
// transfers cannot evade the policy. Exact integer arithmetic avoids rounding.
func dominanceLimit(total int64) int64 { return total/5*4 + (total%5)*4/5 }
func dominant(amounts map[string]int64, total int64) bool {
	for _, amount := range amounts {
		if amount > dominanceLimit(total) {
			return true
		}
	}
	return false
}
func contributions(m map[string]int64, a, b string, amount int64) error {
	for _, id := range []string{a, b} {
		next, err := plus(m[id], amount)
		if err != nil {
			return err
		}
		m[id] = next
	}
	return nil
}
func (s DashboardService) Admin(ctx context.Context, a Actor, kind string, count int) (Dashboard, error) {
	snap, periods, err := s.load(ctx, a, true, kind, count)
	if err != nil {
		return Dashboard{}, err
	}
	result := Dashboard{Timezone: s.Zone.String(), Scale: snap.Scale, AsOf: snap.AsOf, Flows: []Flow{}}
	kinds := map[string]string{}
	for _, event := range snap.Trades {
		kinds[event.ID] = eventKind(event)
	}
	system := map[string]bool{}
	for _, b := range snap.Balances {
		system[b.Member] = b.System
	}
	for _, p := range periods {
		if err = ctx.Err(); err != nil {
			return Dashboard{}, err
		}
		f := Flow{Period: p}
		members := map[string]bool{}
		originalMembers, reversalMembers := map[string]int64{}, map[string]int64{}
		for _, t := range snap.Trades {
			if t.PostedAt.Before(p.From) || !t.PostedAt.Before(p.To) {
				continue
			}
			k := eventKind(t)
			if k == "reversal" {
				k = kinds[t.ReversalOf]
				if k != "trade" {
					continue
				}
			} else if k != "trade" {
				continue
			}
			if system[t.Payer] || system[t.Payee] {
				continue
			}
			members[t.Payer] = true
			members[t.Payee] = true
			if t.ReversalOf == "" {
				f.Trades++
				f.Gross, err = plus(f.Gross, t.Amount)
				if err == nil {
					err = contributions(originalMembers, t.Payer, t.Payee, t.Amount)
				}
			} else {
				f.Reversals++
				f.Reversed, err = plus(f.Reversed, t.Amount)
				if err == nil {
					err = contributions(reversalMembers, t.Payer, t.Payee, t.Amount)
				}
			}
			if err != nil {
				return Dashboard{}, err
			}
		}
		f.Net, err = plus(f.Gross, -f.Reversed)
		if err != nil {
			return Dashboard{}, err
		}
		f.Traders = len(members)
		if f.Traders < s.K || (f.Gross > 0 && (len(originalMembers) < s.K || dominant(originalMembers, f.Gross))) || (f.Reversed > 0 && (len(reversalMembers) < s.K || dominant(reversalMembers, f.Reversed))) {
			f = Flow{Period: p, Suppressed: true}
		}
		result.Flows = append(result.Flows, f)
	}
	d := Distribution{}
	largestPositive, largestNegative := int64(0), int64(0)
	for _, b := range snap.Balances {
		if b.System {
			continue
		}
		switch {
		case b.Amount > 0:
			d.PositiveAccounts++
			d.Positive, err = plus(d.Positive, b.Amount)
			if b.Amount > largestPositive {
				largestPositive = b.Amount
			}
		case b.Amount < 0:
			if b.Amount == math.MinInt64 {
				return Dashboard{}, errors.New("stock magnitude overflow")
			}
			d.NegativeAccounts++
			d.Negative, err = plus(d.Negative, b.Amount)
			if -b.Amount > largestNegative {
				largestNegative = -b.Amount
			}
		default:
			d.ZeroAccounts++
		}
		if err != nil {
			return Dashboard{}, err
		}
	}
	if d.PositiveAccounts < s.K || d.NegativeAccounts < s.K || (d.ZeroAccounts > 0 && d.ZeroAccounts < s.K) || largestPositive > dominanceLimit(d.Positive) || largestNegative > dominanceLimit(-d.Negative) || result.Flows[len(result.Flows)-1].Suppressed {
		d = Distribution{Suppressed: true}
	}
	result.Stock = d
	days := s.DormantDays
	if days == 0 {
		days = 30
	}
	result.DormantDays = days
	dormant := map[string]int64{}
	for _, o := range snap.Offers {
		if o.Active && o.Visible && !system[o.Member] && o.Updated.Before(snap.AsOf.AddDate(0, 0, -days)) {
			result.DormantOffers++
			dormant[o.Member]++
		}
	}
	if len(dormant) < s.K || dominant(dormant, int64(result.DormantOffers)) {
		result.DormantOffers = 0
		result.DormantSuppressed = true
	}
	if err = ctx.Err(); err != nil {
		return Dashboard{}, err
	}
	// Never expose a source/global database revision: publish only a fingerprint
	// of the privacy-filtered result. Hidden and foreign updates cannot affect it.
	encoded, err := json.Marshal(result)
	if err != nil {
		return Dashboard{}, err
	}
	hash := sha256.Sum256(encoded)
	result.Revision = hex.EncodeToString(hash[:])
	return result, nil
}

type OwnTrend struct {
	Period             Period
	Timezone           string
	Scale              int
	Incoming, Outgoing int64 `json:",string"`
	Opening, Closing   int64 `json:",string"`
	Trades             int
	HasPostings        bool
}

func (s DashboardService) Own(ctx context.Context, a Actor, kind string, count int) ([]OwnTrend, error) {
	snap, periods, err := s.load(ctx, a, false, kind, count)
	if err != nil {
		return nil, err
	}
	var balance int64
	found := false
	for _, b := range snap.Balances {
		if b.Member == a.Member && !b.System {
			balance = b.Amount
			found = true
		}
	}
	if !found {
		return nil, errors.New("own account unavailable")
	}
	result := []OwnTrend{}
	for _, p := range periods {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		r := OwnTrend{Period: p, Timezone: s.Zone.String(), Scale: snap.Scale}
		closing, opening := big.NewInt(balance), big.NewInt(balance)
		for _, t := range snap.Trades {
			sign := int64(0)
			if t.Payer == a.Member {
				sign = -t.Amount
			}
			if t.Payee == a.Member {
				sign = t.Amount
			}
			if sign == 0 {
				continue
			}
			if !t.PostedAt.Before(p.From) {
				opening.Sub(opening, big.NewInt(sign))
			}
			if !t.PostedAt.Before(p.To) {
				closing.Sub(closing, big.NewInt(sign))
			}
			if t.PostedAt.Before(p.From) || !t.PostedAt.Before(p.To) {
				continue
			}
			r.HasPostings = true
			r.Trades++
			if sign > 0 {
				r.Incoming, err = plus(r.Incoming, sign)
			} else {
				r.Outgoing, err = plus(r.Outgoing, -sign)
			}
			if err != nil {
				return nil, err
			}
		}
		if !opening.IsInt64() || !closing.IsInt64() {
			return nil, errors.New("historical balance overflow")
		}
		r.Opening, r.Closing = opening.Int64(), closing.Int64()
		result = append(result, r)
	}
	if err = ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

var dashboardTemplate = template.Must(template.New("dashboard").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Credit circulation</title></head><body><main><h1>Credit circulation</h1><p>Time zone: {{.Timezone}}. Snapshot {{.Revision}}. Exact integer minor credit units; decimal scale {{.Scale}}. Flow is turnover; stock is outstanding balance. Suppressed values are unavailable.</p><table><caption>Completed calendar periods</caption><thead><tr><th scope="col">Period</th><th scope="col">Trade count</th><th scope="col">Gross turnover</th><th scope="col">Reversed turnover</th><th scope="col">Net turnover</th></tr></thead><tbody>{{range .Flows}}<tr><th scope="row">{{.Period.Label}}</th>{{if .Suppressed}}<td colspan="4">Suppressed</td>{{else}}<td>{{.Trades}}</td><td>{{.Gross}}</td><td>{{.Reversed}}</td><td>{{.Net}}</td>{{end}}</tr>{{end}}</tbody></table><h2>Outstanding balance stock</h2>{{if .Stock.Suppressed}}<p>Suppressed</p>{{else}}<table><caption>Ordinary member balances at the snapshot cutoff</caption><tr><th scope="col">Band</th><th scope="col">Accounts</th><th scope="col">Total minor units</th></tr><tr><th scope="row">Positive</th><td>{{.Stock.PositiveAccounts}}</td><td>{{.Stock.Positive}}</td></tr><tr><th scope="row">Negative</th><td>{{.Stock.NegativeAccounts}}</td><td>{{.Stock.Negative}}</td></tr><tr><th scope="row">Zero</th><td>{{.Stock.ZeroAccounts}}</td><td>0</td></tr></table>{{end}}<h2>Dormant offers</h2><p>{{if .DormantSuppressed}}Suppressed{{else}}{{.DormantOffers}} active visible offers with no update for over {{.DormantDays}} calendar days. This does not measure demand.{{end}}</p></main></body></html>`))
var ownTemplate = template.Must(template.New("own").Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>My credit trend</title></head><body><main><h1>My credit trend</h1><p>Only your own account is shown. All values are exact integer minor credit units.</p><table><caption>Completed periods, including grants, adjustments and reversals</caption><thead><tr><th scope="col">Period and zone</th><th scope="col">Opening</th><th scope="col">Incoming</th><th scope="col">Outgoing</th><th scope="col">Closing</th><th scope="col">Activity</th></tr></thead><tbody>{{range .}}<tr><th scope="row">{{.Period.Label}} {{.Timezone}} (scale {{.Scale}})</th><td>{{.Opening}}</td><td>{{.Incoming}}</td><td>{{.Outgoing}}</td><td>{{.Closing}}</td><td>{{if .HasPostings}}{{.Trades}} postings{{else}}No postings{{end}}</td></tr>{{end}}</tbody></table></main></body></html>`))

func RenderDashboard(w io.Writer, d Dashboard) error { return dashboardTemplate.Execute(w, d) }
func RenderOwn(w io.Writer, rows []OwnTrend) error   { return ownTemplate.Execute(w, rows) }

// Export serializes only already-suppressed release data. Textual cells are
// generated dates and fixed labels, so no untrusted formula cells are accepted.
func csvText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed != "" && strings.ContainsRune("=+-@", rune(trimmed[0])) {
		return "'" + value
	}
	return value
}
func Export(w io.Writer, d Dashboard) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"period", "gross_minor_units", "reversed_minor_units", "net_minor_units", "trades", "traders", "decimal_scale", "timezone"}); err != nil {
		return err
	}
	flows := append([]Flow(nil), d.Flows...)
	sort.Slice(flows, func(i, j int) bool { return flows[i].Period.From.Before(flows[j].Period.From) })
	for _, f := range flows {
		row := []string{f.Period.From.Format("2006-01-02"), "suppressed", "", "", "", "", fmt.Sprint(d.Scale), csvText(d.Timezone)}
		if !f.Suppressed {
			row[1], row[2], row[3], row[4], row[5] = fmt.Sprint(f.Gross), fmt.Sprint(f.Reversed), fmt.Sprint(f.Net), fmt.Sprint(f.Trades), fmt.Sprint(f.Traders)
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
func ExportOwn(w io.Writer, rows []OwnTrend) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"period", "opening", "incoming", "outgoing", "closing", "postings", "has_postings", "decimal_scale", "timezone"}); err != nil {
		return err
	}
	for _, r := range rows {
		if err := writer.Write([]string{r.Period.From.Format("2006-01-02"), fmt.Sprint(r.Opening), fmt.Sprint(r.Incoming), fmt.Sprint(r.Outgoing), fmt.Sprint(r.Closing), fmt.Sprint(r.Trades), fmt.Sprint(r.HasPostings), fmt.Sprint(r.Scale), csvText(r.Timezone)}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
