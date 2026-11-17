// Package cli implements the operator command line for the discovery module.
// It is a trusted local tool with direct database access; it performs no
// remote authentication and acts as a synthetic administrator of the network
// named on the command line. It starts no servers or background jobs.
package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	d "github.com/djshellshoxxx/circula/future/discovery"
	"github.com/djshellshoxxx/circula/future/discovery/imgsan"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
)

// Env injects process dependencies for testing.
type Env struct {
	// Open opens the SQLite database at path (a driver must be linked in).
	Open func(path string) (*sql.DB, error)
	Now  func() time.Time
}

const usage = `discovery - operator tool for the Circula marketplace discovery module (FUX-103)

HOLD: isolated development tool. It never contacts a server and registers nothing.

Usage:
  discovery <command> [flags]

Commands:
  help [command]     show this help or a command's help
  version            print the module identification
  selftest           sanitize a synthetic geotagged JPEG and verify metadata removal
  sanitize           re-encode one JPEG/PNG with the upload sanitizer
  migrate            apply (or --rollback) the module's SQLite migrations
  categories         list | add | enable | disable | rename
  areas              list | add | enable | disable
  tags               list | approve | reject
  search             run a filtered search as a member of a network
  retention          reclaim removed/archived/abandoned photos past retention
  verify             check photo rows against the blob directory
  audit              show the newest audit entries

Common flags: --db PATH (SQLite file), --network ID. Run "discovery help <command>".
Exit status: 0 success, 1 runtime failure, 2 usage error.
`

var helps = map[string]string{
	"selftest":   "discovery selftest\n  Builds a synthetic JPEG with EXIF GPS and orientation, sanitizes it and prints PASS/FAIL. No files or network are used.",
	"sanitize":   "discovery sanitize --in FILE --out FILE [--thumb FILE] [--max-edge N]\n  Validates and re-encodes a JPEG or PNG (metadata dropped, orientation applied). Refuses SVG/GIF/ZIP/polyglots/oversize images.",
	"migrate":    "discovery migrate --db PATH [--rollback]\n  Applies module-local migrations. --rollback reverses the newest one (module tables only; blob files are not touched).",
	"categories": "discovery categories <list|add|enable|disable|rename> --db PATH --network ID [--id ID] [--name NAME] [--kind goods|service|any] [--seed]\n  add --seed installs the default taxonomy.",
	"areas":      "discovery areas <list|add|enable|disable> --db PATH --network ID [--id ID] [--name NAME] [--lat DEG --lon DEG]\n  Coordinates are rounded to 0.1 degree. Never use a home address.",
	"tags":       "discovery tags <list|approve|reject> --db PATH --network ID [--tag NAME] [--status pending|approved|rejected]",
	"search":     "discovery search --db PATH --network ID --member ID [--q TEXT] [--category ID] [--type goods|service] [--kind offer|request] [--condition C] [--availability A] [--area ID] [--tag T] [--sort relevance|recent] [--scope mine|all] [--limit N] [--json]",
	"retention":  "discovery retention --db PATH --blobs DIR\n  Runs the retention job once and prints a report.",
	"verify":     "discovery verify --db PATH --blobs DIR\n  Reports missing blobs, checksum mismatches and orphan files. Exit status 1 when problems are found.",
	"audit":      "discovery audit --db PATH --network ID [--limit N]",
	"version":    "discovery version",
}

// Run executes one command and returns the process exit status.
func Run(args []string, stdout, stderr io.Writer, env Env) int {
	if env.Now == nil {
		env.Now = func() time.Time { return time.Now().UTC() }
	}
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	cmd, rest := args[0], args[1:]
	r := &runner{out: stdout, err: stderr, env: env}
	switch cmd {
	case "help", "-h", "--help":
		if len(rest) > 0 {
			if h, ok := helps[rest[0]]; ok {
				fmt.Fprintln(stdout, h)
				return 0
			}
			fmt.Fprintf(stderr, "unknown command %q\n", rest[0])
			return 2
		}
		fmt.Fprint(stdout, usage)
		return 0
	case "version":
		fmt.Fprintln(stdout, "discovery FUX-103 (isolated development, HOLD)")
		return 0
	case "selftest":
		return r.selftest()
	case "sanitize":
		return r.sanitize(rest)
	case "migrate":
		return r.migrate(rest)
	case "categories":
		return r.categories(rest)
	case "areas":
		return r.areas(rest)
	case "tags":
		return r.tags(rest)
	case "search":
		return r.search(rest)
	case "retention":
		return r.retention(rest)
	case "verify":
		return r.verify(rest)
	case "audit":
		return r.audit(rest)
	}
	fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
	return 2
}

type runner struct {
	out, err io.Writer
	env      Env
}

type common struct {
	db, network, blobs string
}

func (r *runner) flags(name string, c *common) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(r.err)
	fs.StringVar(&c.db, "db", "", "SQLite database file")
	fs.StringVar(&c.network, "network", "", "network id")
	fs.StringVar(&c.blobs, "blobs", "", "blob directory")
	return fs
}

func (r *runner) fail(format string, a ...any) int {
	fmt.Fprintf(r.err, "error: "+format+"\n", a...)
	return 1
}

func (r *runner) usageErr(format string, a ...any) int {
	fmt.Fprintf(r.err, "usage error: "+format+"\n", a...)
	return 2
}

func (r *runner) openDB(path string, migrate bool) (*sql.DB, error) {
	if path == "" {
		return nil, errors.New("--db is required")
	}
	if r.env.Open == nil {
		return nil, errors.New("no SQLite driver linked into this build (build with -tags=sqlite_cgo_test for the test-only driver, or provide one)")
	}
	db, err := r.env.Open(path)
	if err != nil {
		return nil, err
	}
	if migrate {
		if err := sqlstore.Migrate(context.Background(), db); err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

func (r *runner) service(db *sql.DB, blobDir string) (*d.Service, *sqlstore.Store, error) {
	var blobs d.BlobStore
	if blobDir == "" {
		blobs = d.NewMemBlobs()
	} else {
		b, err := d.NewFSBlobs(blobDir)
		if err != nil {
			return nil, nil, err
		}
		blobs = b
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, nil, err
	}
	st := sqlstore.New(db)
	svc, err := d.NewService(st, blobs, d.Config{CursorKeys: [][]byte{key}, Now: r.env.Now}, imgsan.Limits{})
	return svc, st, err
}

func operator(network string) d.Actor {
	return d.Actor{ID: "cli-operator", Network: network, Role: d.RoleAdmin, Active: true}
}

func (r *runner) selftest() int {
	jpg := syntheticGeotaggedJPEG()
	res, err := imgsan.New(imgsan.Limits{}).Sanitize(context.Background(), strings.NewReader(string(jpg)))
	if err != nil {
		return r.fail("selftest: %v", err)
	}
	for _, needle := range []string{"GPSLatitude", "Exif", "HOME"} {
		if strings.Contains(string(res.Full.Data), needle) || strings.Contains(string(res.Thumb.Data), needle) {
			fmt.Fprintf(r.out, "FAIL: %q survived sanitization\n", needle)
			return 1
		}
	}
	if res.Full.Width != 20 || res.Full.Height != 60 {
		fmt.Fprintf(r.out, "FAIL: orientation not applied (%dx%d)\n", res.Full.Width, res.Full.Height)
		return 1
	}
	fmt.Fprintf(r.out, "PASS: metadata removed, orientation applied, full %dx%d thumb %dx%d sha256 %s\n",
		res.Full.Width, res.Full.Height, res.Thumb.Width, res.Thumb.Height, res.SHA256[:16])
	return 0
}

func (r *runner) sanitize(args []string) int {
	var in, out, thumb string
	var maxEdge int
	fs := flag.NewFlagSet("sanitize", flag.ContinueOnError)
	fs.SetOutput(r.err)
	fs.StringVar(&in, "in", "", "input file")
	fs.StringVar(&out, "out", "", "output file")
	fs.StringVar(&thumb, "thumb", "", "thumbnail output file")
	fs.IntVar(&maxEdge, "max-edge", 0, "longest edge of the output")
	if fs.Parse(args) != nil {
		return 2
	}
	if in == "" || out == "" {
		return r.usageErr("--in and --out are required")
	}
	f, err := os.Open(in)
	if err != nil {
		return r.fail("%v", err)
	}
	defer f.Close()
	res, err := imgsan.New(imgsan.Limits{FullMaxEdge: maxEdge}).Sanitize(context.Background(), f)
	if err != nil {
		return r.fail("rejected: %v", err)
	}
	if err := os.WriteFile(out, res.Full.Data, 0o600); err != nil {
		return r.fail("%v", err)
	}
	if thumb != "" {
		if err := os.WriteFile(thumb, res.Thumb.Data, 0o600); err != nil {
			return r.fail("%v", err)
		}
	}
	fmt.Fprintf(r.out, "ok %s %dx%d %d bytes sha256 %s\n", res.Full.MIME, res.Full.Width, res.Full.Height, len(res.Full.Data), res.SHA256)
	return 0
}

func (r *runner) migrate(args []string) int {
	var c common
	var back bool
	fs := r.flags("migrate", &c)
	fs.BoolVar(&back, "rollback", false, "roll back the newest migration")
	if fs.Parse(args) != nil {
		return 2
	}
	db, err := r.openDB(c.db, false)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	if back {
		if err := sqlstore.Rollback(context.Background(), db); err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintln(r.out, "rolled back newest migration (module tables only; blob files untouched)")
		return 0
	}
	if err := sqlstore.Migrate(context.Background(), db); err != nil {
		return r.fail("%v", err)
	}
	fmt.Fprintln(r.out, "migrations applied")
	return 0
}

func (r *runner) usageOrFail(err error) int {
	if strings.Contains(err.Error(), "is required") {
		return r.usageErr("%v", err)
	}
	return r.fail("%v", err)
}

func (r *runner) categories(args []string) int {
	if len(args) == 0 {
		return r.usageErr("categories needs a subcommand")
	}
	sub := args[0]
	var c common
	var id, name, kind string
	var seed bool
	fs := r.flags("categories", &c)
	fs.StringVar(&id, "id", "", "category id")
	fs.StringVar(&name, "name", "", "category name")
	fs.StringVar(&kind, "kind", "any", "goods|service|any")
	fs.BoolVar(&seed, "seed", false, "install default categories")
	if fs.Parse(args[1:]) != nil {
		return 2
	}
	if c.network == "" {
		return r.usageErr("--network is required")
	}
	db, err := r.openDB(c.db, true)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, "")
	if err != nil {
		return r.fail("%v", err)
	}
	a, ctx := operator(c.network), context.Background()
	switch sub {
	case "list":
		cs, err := svc.ListCategories(ctx, a)
		if err != nil {
			return r.fail("%v", err)
		}
		tw := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tKIND\tENABLED\tLISTINGS")
		for _, c := range cs {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%v\t%d\n", c.ID, c.Name, c.Kind, c.Enabled, c.Listings)
		}
		tw.Flush()
	case "add":
		if seed {
			n, err := svc.SeedDefaultCategories(ctx, a)
			if err != nil {
				return r.fail("%v", err)
			}
			fmt.Fprintf(r.out, "added %d default categories\n", n)
			return 0
		}
		cat, err := svc.CreateCategory(ctx, a, name, kind)
		if err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintf(r.out, "added %s %q\n", cat.ID, cat.Name)
	case "enable", "disable", "rename":
		if id == "" {
			return r.usageErr("--id is required")
		}
		var en *bool
		var nm *string
		switch sub {
		case "enable", "disable":
			v := sub == "enable"
			en = &v
		default:
			nm = &name
		}
		if err := svc.UpdateCategory(ctx, a, id, nm, en); err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintln(r.out, "updated")
	default:
		return r.usageErr("unknown categories subcommand %q", sub)
	}
	return 0
}

func (r *runner) areas(args []string) int {
	if len(args) == 0 {
		return r.usageErr("areas needs a subcommand")
	}
	sub := args[0]
	var c common
	var id, name string
	var lat, lon float64
	fs := r.flags("areas", &c)
	fs.StringVar(&id, "id", "", "area id")
	fs.StringVar(&name, "name", "", "area name")
	fs.Float64Var(&lat, "lat", 1000, "centre latitude (optional)")
	fs.Float64Var(&lon, "lon", 1000, "centre longitude (optional)")
	if fs.Parse(args[1:]) != nil {
		return 2
	}
	if c.network == "" {
		return r.usageErr("--network is required")
	}
	db, err := r.openDB(c.db, true)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, "")
	if err != nil {
		return r.fail("%v", err)
	}
	a, ctx := operator(c.network), context.Background()
	switch sub {
	case "list":
		as, err := svc.ListAreas(ctx, a)
		if err != nil {
			return r.fail("%v", err)
		}
		tw := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tENABLED\tCOARSE CENTRE")
		for _, x := range as {
			cell := "-"
			if x.Cell != nil {
				cell = fmt.Sprintf("%.1f,%.1f", float64(x.Cell.Lat10)/10, float64(x.Cell.Lon10)/10)
			}
			fmt.Fprintf(tw, "%s\t%s\t%v\t%s\n", x.ID, x.Name, x.Enabled, cell)
		}
		tw.Flush()
	case "add":
		var cell *d.GeoCell
		if lat != 1000 || lon != 1000 {
			if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
				return r.usageErr("--lat/--lon must both be valid degrees")
			}
			cell = &d.GeoCell{Lat10: int(roundHalfAway(lat * 10)), Lon10: int(roundHalfAway(lon * 10))}
		}
		ar, err := svc.CreateArea(ctx, a, name, cell)
		if err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintf(r.out, "added %s %q\n", ar.ID, ar.Name)
	case "enable", "disable":
		if id == "" {
			return r.usageErr("--id is required")
		}
		v := sub == "enable"
		if err := svc.UpdateArea(ctx, a, id, nil, &v); err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintln(r.out, "updated")
	default:
		return r.usageErr("unknown areas subcommand %q", sub)
	}
	return 0
}

func roundHalfAway(f float64) float64 {
	if f < 0 {
		return -float64(int64(-f + 0.5))
	}
	return float64(int64(f + 0.5))
}

func (r *runner) tags(args []string) int {
	if len(args) == 0 {
		return r.usageErr("tags needs a subcommand")
	}
	sub := args[0]
	var c common
	var tag, status string
	fs := r.flags("tags", &c)
	fs.StringVar(&tag, "tag", "", "tag name")
	fs.StringVar(&status, "status", "", "filter by status")
	if fs.Parse(args[1:]) != nil {
		return 2
	}
	if c.network == "" {
		return r.usageErr("--network is required")
	}
	db, err := r.openDB(c.db, true)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, "")
	if err != nil {
		return r.fail("%v", err)
	}
	a, ctx := operator(c.network), context.Background()
	switch sub {
	case "list":
		ts, err := svc.ListTags(ctx, a, d.TagStatus(status))
		if err != nil {
			return r.fail("%v", err)
		}
		tw := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "TAG\tSTATUS\tLISTINGS")
		for _, t := range ts {
			fmt.Fprintf(tw, "%s\t%s\t%d\n", t.Name, t.Status, t.Listings)
		}
		tw.Flush()
	case "approve", "reject":
		if tag == "" {
			return r.usageErr("--tag is required")
		}
		st := d.TagApproved
		if sub == "reject" {
			st = d.TagRejected
		}
		if err := svc.ModerateTag(ctx, a, tag, st); err != nil {
			return r.fail("%v", err)
		}
		fmt.Fprintf(r.out, "tag %s %s\n", tag, st)
	default:
		return r.usageErr("unknown tags subcommand %q", sub)
	}
	return 0
}

func (r *runner) search(args []string) int {
	var c common
	var jsonOut bool
	var memberID string
	f := d.Filters{}
	fs := r.flags("search", &c)
	fs.StringVar(&memberID, "member", "", "member id to search as")
	fs.StringVar(&f.Q, "q", "", "text")
	fs.StringVar(&f.Category, "category", "", "category id")
	fs.StringVar(&f.Type, "type", "", "goods|service")
	fs.StringVar(&f.Kind, "kind", "", "offer|request")
	fs.StringVar(&f.Condition, "condition", "", "condition")
	fs.StringVar(&f.Availability, "availability", "", "availability")
	fs.StringVar(&f.Area, "area", "", "area id")
	fs.StringVar(&f.Tag, "tag", "", "tag")
	fs.StringVar(&f.Sort, "sort", "", "relevance|recent")
	fs.StringVar(&f.Scope, "scope", "", "mine|all")
	fs.IntVar(&f.Limit, "limit", 0, "page size")
	fs.BoolVar(&jsonOut, "json", false, "JSON output")
	if fs.Parse(args) != nil {
		return 2
	}
	if c.network == "" || memberID == "" {
		return r.usageErr("--network and --member are required")
	}
	db, err := r.openDB(c.db, false)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, "")
	if err != nil {
		return r.fail("%v", err)
	}
	role := d.RoleMember
	if f.Scope == d.ScopeAll {
		role = d.RoleAdmin
	}
	res, err := svc.Search(context.Background(), d.Actor{ID: memberID, Network: c.network, Role: role, Active: true}, f, "")
	if err != nil {
		return r.fail("%v", err)
	}
	if jsonOut {
		type row struct {
			ID, Title, Category, Type string
			Tags                      []string
		}
		var rows []row
		for _, it := range res.Items {
			rows = append(rows, row{it.Listing.ID, it.Listing.Title, it.Meta.CategoryID, it.Meta.Type, it.Tags})
		}
		enc := json.NewEncoder(r.out)
		enc.SetIndent("", "  ")
		enc.Encode(map[string]any{"items": rows, "more": res.Next != "", "saved_view": res.SavedView})
		return 0
	}
	if len(res.Items) == 0 {
		fmt.Fprintln(r.out, "no listings found")
		return 0
	}
	tw := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tCATEGORY\tTYPE\tTAGS")
	for _, it := range res.Items {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", it.Listing.ID, it.Listing.Title, it.Meta.CategoryID, it.Meta.Type, strings.Join(it.Tags, ","))
	}
	tw.Flush()
	if res.Next != "" {
		fmt.Fprintln(r.out, "(more results; raise --limit)")
	}
	return 0
}

func (r *runner) retention(args []string) int {
	var c common
	fs := r.flags("retention", &c)
	if fs.Parse(args) != nil {
		return 2
	}
	if c.blobs == "" {
		return r.usageErr("--blobs is required")
	}
	db, err := r.openDB(c.db, true)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, c.blobs)
	if err != nil {
		return r.fail("%v", err)
	}
	rep, err := svc.RunRetention(context.Background())
	if err != nil {
		return r.fail("%v", err)
	}
	fmt.Fprintf(r.out, "claimed %d, rows deleted %d, blobs deleted %d, errors %d\n", rep.Claimed, rep.RowsDeleted, rep.BlobsDeleted, len(rep.Errors))
	for _, e := range rep.Errors {
		fmt.Fprintf(r.err, "error: %v\n", e)
	}
	if len(rep.Errors) > 0 {
		return 1
	}
	return 0
}

func (r *runner) verify(args []string) int {
	var c common
	fset := r.flags("verify", &c)
	if fset.Parse(args) != nil {
		return 2
	}
	if c.blobs == "" {
		return r.usageErr("--blobs is required")
	}
	db, err := r.openDB(c.db, false)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	st := sqlstore.New(db)
	blobs, err := d.NewFSBlobs(c.blobs)
	if err != nil {
		return r.fail("%v", err)
	}
	ctx := context.Background()
	referenced := map[string]bool{}
	problems, checked := 0, 0
	after := ""
	for {
		rows, err := st.AllMedia(ctx, after, 200)
		if err != nil {
			return r.fail("%v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, m := range rows {
			after = m.ID
			referenced[m.StoragePath], referenced[m.ThumbPath] = true, true
			if m.Status != d.MediaActive {
				continue
			}
			checked++
			data, err := blobs.Get(ctx, m.StoragePath)
			if err != nil {
				fmt.Fprintf(r.out, "MISSING full blob for %s (%s)\n", m.ID, m.StoragePath)
				problems++
			} else if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != m.SHA256 {
				fmt.Fprintf(r.out, "CHECKSUM mismatch for %s\n", m.ID)
				problems++
			}
			if _, err := blobs.Get(ctx, m.ThumbPath); err != nil {
				fmt.Fprintf(r.out, "MISSING thumbnail blob for %s (%s)\n", m.ID, m.ThumbPath)
				problems++
			}
		}
	}
	var orphans []string
	root, _ := filepath.Abs(c.blobs)
	filepath.WalkDir(root, func(p string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if !referenced[rel] {
			orphans = append(orphans, rel)
		}
		return nil
	})
	sort.Strings(orphans)
	for _, o := range orphans {
		fmt.Fprintf(r.out, "ORPHAN file %s\n", o)
		problems++
	}
	fmt.Fprintf(r.out, "checked %d active photos, %d problem(s)\n", checked, problems)
	if problems > 0 {
		return 1
	}
	return 0
}

func (r *runner) audit(args []string) int {
	var c common
	var limit int
	fs := r.flags("audit", &c)
	fs.IntVar(&limit, "limit", 20, "entries")
	if fs.Parse(args) != nil {
		return 2
	}
	if c.network == "" {
		return r.usageErr("--network is required")
	}
	db, err := r.openDB(c.db, false)
	if err != nil {
		return r.usageOrFail(err)
	}
	defer db.Close()
	svc, _, err := r.service(db, "")
	if err != nil {
		return r.fail("%v", err)
	}
	es, err := svc.Audit(context.Background(), operator(c.network), limit)
	if err != nil {
		return r.fail("%v", err)
	}
	tw := tabwriter.NewWriter(r.out, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "TIME\tACTOR\tACTION\tTARGET")
	for _, e := range es {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.At.Format(time.RFC3339), e.Actor, e.Action, e.Target)
	}
	tw.Flush()
	return 0
}
