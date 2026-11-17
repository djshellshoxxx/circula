// Package cli implements the favorites command line over favorites.Service.
//
// The commands act for the member named by --network and --member. That is an
// operator-supplied identity for a local tool run by someone who already has
// access to the database; it is not authentication, and the same Service
// still re-checks every authorization fact through its Policy.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/djshellshoxxx/circula/future/favorites"
)

// Exit codes.
const (
	ExitOK      = 0
	ExitFailure = 1
	ExitUsage   = 2
)

// Help is the top-level usage text.
const Help = `favorites - owner-private saved trading partners

Usage:
  favorites <command> [flags]

Member commands (all require --network and --member):
  list      List current saved partners          [--limit N] [--cursor C] [--json]
  add       Save a partner                       --target ID [--nickname S] [--pin]
  rename    Change a private nickname            --target ID --nickname S
  pin       Pin a partner to the top             --target ID
  unpin     Remove the pin                       --target ID
  move      Move one place within its group      --target ID (--up | --down)
  remove    Delete a saved partner               --target ID
  draft     Start an unpriced proposal draft     (--trade ID | --target ID) [--key K] [--json]

Maintenance commands:
  departed      Delete all rows for a departed member   --network N --member M
  purge-drafts  Delete expired drafts

Other:
  help      Show this text

Notes:
  * Favorites never change credit limits, balances, ledger entries or reputation.
  * A draft never has an amount and carries a fresh idempotency key; nothing is proposed.
  * Exit status: 0 success, 1 failure, 2 usage error.
`

// Run executes args (without the program name) and returns an exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer, svc *favorites.Service) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, Help)
		return ExitUsage
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, Help)
		return ExitOK
	}
	fs := flag.NewFlagSet(cmd, flag.ContinueOnError)
	fs.SetOutput(stderr)
	network := fs.String("network", "", "network ID")
	member := fs.String("member", "", "acting member ID")
	target := fs.String("target", "", "target member ID")
	nick := fs.String("nickname", "", "private nickname")
	pin := fs.Bool("pin", false, "pin the new favorite")
	up := fs.Bool("up", false, "move up")
	down := fs.Bool("down", false, "move down")
	limit := fs.Int("limit", 0, "page size")
	cursor := fs.String("cursor", "", "page cursor")
	asJSON := fs.Bool("json", false, "JSON output")
	trade := fs.String("trade", "", "historical trade ID")
	key := fs.String("key", "", "request key (same key returns the same draft)")
	if err := fs.Parse(rest); err != nil {
		return ExitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "unexpected argument %q\n", fs.Arg(0))
		return ExitUsage
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "error:", message(err))
		return ExitFailure
	}
	usage := func(msg string) int {
		fmt.Fprintf(stderr, "%s: %s\n", cmd, msg)
		return ExitUsage
	}
	out := func(v any) {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(v)
	}

	switch cmd {
	case "departed":
		if *network == "" || *member == "" {
			return usage("--network and --member are required")
		}
		res, err := svc.MemberDeparted(ctx, *network, *member)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "removed %d favorites, %d drafts, %d list states\n", res.Favorites, res.Intents, res.States)
		return ExitOK
	case "purge-drafts":
		n, err := svc.PurgeExpiredDrafts(ctx)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "removed %d expired drafts\n", n)
		return ExitOK
	case "list", "add", "rename", "pin", "unpin", "move", "remove", "draft":
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, Help)
		return ExitUsage
	}
	if *network == "" || *member == "" {
		return usage("--network and --member are required")
	}
	a := favorites.Actor{Network: *network, Member: *member}
	needTarget := func() bool { return *target != "" }

	switch cmd {
	case "list":
		pg, err := svc.List(ctx, a, favorites.ListOptions{Limit: *limit, Cursor: *cursor})
		if err != nil {
			return fail(err)
		}
		if *asJSON {
			out(map[string]any{"items": pg.Items, "next_cursor": pg.Next, "revision": pg.Revision})
			return ExitOK
		}
		tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "MEMBER\tNAME\tNICKNAME\tPINNED")
		for _, v := range pg.Items {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%v\n", v.Target, v.DisplayName, v.Nickname, v.Pinned)
		}
		_ = tw.Flush()
		if pg.Next != "" {
			fmt.Fprintf(stdout, "next page: --cursor %s\n", pg.Next)
		}
	case "add":
		if !needTarget() {
			return usage("--target is required")
		}
		v, created, err := svc.Add(ctx, a, favorites.AddRequest{Target: *target, Nickname: *nick, Pinned: *pin})
		if err != nil {
			return fail(err)
		}
		if created {
			fmt.Fprintf(stdout, "saved %s\n", v.Target)
		} else {
			fmt.Fprintf(stdout, "already saved %s\n", v.Target)
		}
	case "rename":
		if !needTarget() {
			return usage("--target is required")
		}
		if err := svc.SetNickname(ctx, a, *target, *nick); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "nickname saved")
	case "pin", "unpin":
		if !needTarget() {
			return usage("--target is required")
		}
		if err := svc.SetPinned(ctx, a, *target, cmd == "pin"); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, cmd+"ned")
	case "remove":
		if !needTarget() {
			return usage("--target is required")
		}
		ok, err := svc.Remove(ctx, a, *target)
		if err != nil {
			return fail(err)
		}
		if ok {
			fmt.Fprintln(stdout, "removed")
		} else {
			fmt.Fprintln(stdout, "not in your list")
		}
	case "move":
		if !needTarget() || *up == *down {
			return usage("--target and exactly one of --up or --down are required")
		}
		var all []favorites.View
		var rev int64
		cur := ""
		for {
			pg, err := svc.List(ctx, a, favorites.ListOptions{Limit: favorites.MaxPageLimit, Cursor: cur})
			if err != nil {
				return fail(err)
			}
			if cur == "" {
				rev = pg.Revision
			}
			all = append(all, pg.Items...)
			if pg.Next == "" {
				break
			}
			cur = pg.Next
		}
		idx := -1
		for i, v := range all {
			if v.Target == *target {
				idx = i
			}
		}
		if idx < 0 {
			return fail(favorites.ErrNotFound)
		}
		j := idx - 1
		if *down {
			j = idx + 1
		}
		if j < 0 || j >= len(all) || all[j].Pinned != all[idx].Pinned {
			fmt.Fprintln(stdout, "already at the edge of its group")
			return ExitOK
		}
		all[idx], all[j] = all[j], all[idx]
		order := make([]string, len(all))
		for i, v := range all {
			order[i] = v.Target
		}
		if _, err := svc.Reorder(ctx, a, favorites.ReorderRequest{Revision: rev, Targets: order}); err != nil {
			return fail(err)
		}
		fmt.Fprintln(stdout, "moved")
	case "draft":
		if (*trade == "") == (*target == "") {
			return usage("exactly one of --trade or --target is required")
		}
		var d favorites.Draft
		var err error
		if *trade != "" {
			d, err = svc.RepeatTrade(ctx, a, *trade, *key)
		} else {
			d, err = svc.DraftFromFavorite(ctx, a, *target, *key)
		}
		if err != nil {
			return fail(err)
		}
		if *asJSON {
			out(map[string]any{"network_id": d.Network, "recipient_member_id": d.Recipient, "description": d.Description, "amount": nil,
				"idempotency_key": d.Intent, "request_key": d.RequestKey, "source_trade_id": d.TradeID})
			return ExitOK
		}
		fmt.Fprintf(stdout, "recipient:       %s\ndescription:     %s\namount:          (enter when proposing)\nidempotency key: %s\nrequest key:     %s\n", d.Recipient, d.Description, d.Intent, d.RequestKey)
		fmt.Fprintln(stdout, "nothing has been proposed")
	}
	return ExitOK
}

// message turns an error into a short, non-leaky line.
func message(err error) string {
	var ie *favorites.InputError
	switch {
	case errors.As(err, &ie):
		return ie.Error()
	case errors.Is(err, favorites.ErrUnavailable):
		return "that member cannot be used right now"
	case errors.Is(err, favorites.ErrNotFound):
		return "not found"
	case errors.Is(err, favorites.ErrExists):
		return "already saved with different settings"
	case errors.Is(err, favorites.ErrLimit):
		return "limit reached"
	case errors.Is(err, favorites.ErrStale):
		return "the list changed; try again"
	case errors.Is(err, favorites.ErrKeyReuse):
		return "request key already used for a different draft"
	case errors.Is(err, favorites.ErrTradeUnavailable):
		return "that trade cannot be repeated"
	case errors.Is(err, favorites.ErrPolicy):
		return "authorization lookup unavailable; nothing was changed"
	}
	return strings.SplitN(err.Error(), "\n", 2)[0]
}
