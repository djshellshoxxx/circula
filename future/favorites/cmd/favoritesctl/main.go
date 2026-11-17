// Command favoritesctl is a DEVELOPMENT HARNESS for the favorites module. It
// opens a SQLite file and runs the cli package with a static JSON members file
// standing in for Circula's membership, visibility and block policy. Do not use
// it against real member data: the members file is trusted input and the
// identity flags are not authentication. A production integration builds its
// own Policy over the live service layer and calls cli.Run itself.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/cli"
	"github.com/djshellshoxxx/circula/future/favorites/sqlstore"
)

// driverName/driverDSN are set by an optional build-tagged file that links a
// SQLite driver (see driver_cgo_test_tag.go). Without one the harness refuses.
var driverName string
var driverDSN = func(path string) string { return "file:" + path }

type fileDirectory struct {
	Members []struct {
		Network, ID, Name        string
		Active, Visible, Blocked bool
	} `json:"members"`
	Blocks []struct{ Network, Viewer, Other string } `json:"blocks"`
	Trades []struct {
		ID, Network, Payer, Payee, Description string
		Completed                              bool
	} `json:"trades"`
}

func (d *fileDirectory) lookup(viewer favorites.Actor, ids []string) map[string]favorites.Member {
	out := map[string]favorites.Member{}
	for _, id := range ids {
		for _, m := range d.Members {
			if m.Network == viewer.Network && m.ID == id {
				fm := favorites.Member{ID: m.ID, Network: m.Network, Active: m.Active, Visible: m.Visible, Blocked: m.Blocked, DisplayName: m.Name}
				for _, b := range d.Blocks {
					if b.Network == viewer.Network && b.Viewer == viewer.Member && b.Other == id {
						fm.Blocked = true
					}
				}
				out[id] = fm
			}
		}
	}
	return out
}

type policy struct{ d *fileDirectory }

func (p policy) Members(_ context.Context, v favorites.Actor, ids []string) (map[string]favorites.Member, error) {
	return p.d.lookup(v, ids), nil
}
func (p policy) Trade(_ context.Context, v favorites.Actor, id string) (favorites.HistoricalTrade, error) {
	for _, t := range p.d.Trades {
		if t.ID == id && t.Network == v.Network && (t.Payer == v.Member || t.Payee == v.Member) {
			return favorites.HistoricalTrade{ID: t.ID, Network: t.Network, Payer: t.Payer, Payee: t.Payee, Description: t.Description, Completed: t.Completed}, nil
		}
	}
	return favorites.HistoricalTrade{}, favorites.ErrTradeNotFound
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	// Global flags precede the command: favoritesctl --db F --members M <command> ...
	fs := flag.NewFlagSet("favoritesctl", flag.ContinueOnError)
	dbPath := fs.String("db", "", "SQLite database file (created and migrated if absent)")
	members := fs.String("members", "", "JSON file with synthetic members, blocks and trades (development only)")
	if err := fs.Parse(args); err != nil {
		return cli.ExitUsage
	}
	rest := fs.Args()
	if len(rest) == 0 || rest[0] == "help" || rest[0] == "-h" || rest[0] == "--help" {
		fmt.Print("favoritesctl --db FILE --members FILE <command> [flags]   (development harness)\n\n" + cli.Help)
		return cli.ExitOK
	}
	if *dbPath == "" || *members == "" {
		fmt.Fprintln(os.Stderr, "--db and --members are required")
		return cli.ExitUsage
	}
	if driverName == "" {
		fmt.Fprintln(os.Stderr, "no SQLite driver linked into this build; rebuild with -tags sqlite_cgo_test (test-only cgo driver) or link the host's driver")
		return cli.ExitFailure
	}
	raw, err := os.ReadFile(*members)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitFailure
	}
	var dir fileDirectory
	if err := json.Unmarshal(raw, &dir); err != nil {
		fmt.Fprintln(os.Stderr, "members file:", err)
		return cli.ExitFailure
	}
	db, err := sql.Open(driverName, driverDSN(*dbPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitFailure
	}
	defer db.Close()
	ctx := context.Background()
	if err := sqlstore.Migrate(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		return cli.ExitFailure
	}
	p := policy{&dir}
	svc, err := favorites.NewService(favorites.Config{Store: sqlstore.New(db), Policy: p, Trades: p})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return cli.ExitFailure
	}
	return cli.Run(ctx, rest, os.Stdout, os.Stderr, svc)
}
