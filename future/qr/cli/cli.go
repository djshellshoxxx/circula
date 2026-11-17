// Package cli is the unregistered command-line interface of the QR module. It
// is a library (Run) so a host can wire it to its own database, Core adapter
// and authorization. The module's own binary (cmd/circula-qr) links no SQLite
// driver and therefore offers only the offline commands.
package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	qr "github.com/djshellshoxxx/circula/future/qr"
)

// Env carries everything Run needs; nothing is read from globals.
type Env struct {
	Stdout, Stderr io.Writer
	// Origin is the trusted public origin used to parse and build links.
	Origin string
	// Service is nil for offline-only use (help, parse, render).
	Service *qr.Service
	// WriteFile creates a file exclusively (never overwrites). Default: os.OpenFile with O_EXCL, mode 0600.
	WriteFile func(path string, data []byte) error
}

const usage = `circula-qr: QR identifiers and one-time trade requests (HOLD: not integrated, not enabled)

Usage: circula-qr <command> [options]

Offline commands (no database):
  help                         show this text
  parse <code>                 validate a pasted QR link or compact code (r.<code> / a.<code>)
  render --out FILE <link>     write a printable black-on-white PNG of a trusted-origin link
        --level m|q|h          error correction (default h)    --pixels N   max size 128-2048 (default 512)

Commands that need a host-attached service (acting member given by flags):
  alias show   --network N --member M [--out FILE]
  alias rotate --network N --member M --key KEY [--out FILE]
  request create  --network N --member M --amount MINOR --key KEY [--note TEXT] [--ttl 24h] [--out FILE]
  request show    --network N --member M ID
  request list    --network N --member M [--limit 25] [--cursor C]
  request revoke  --network N --member M ID
  request preview --network N --member M <code>
  claim <code>    --network N --member M --key KEY [--amount MINOR --note TEXT]

Notes:
  * A one-time link/code is printed ONCE at creation and is never stored or recoverable.
    Treat it like a bearer token: do not paste it into chat, tickets or logs.
  * Scanning or claiming only creates a PENDING proposal. A trade is COMPLETED only after the
    other member accepts it through the normal Circula flow.
  * Amounts are whole numbers of minor units. Keys are idempotency keys: retrying with the same
    key is safe; reusing one for different input is refused.
  * Exit codes: 0 ok, 1 failed, 2 usage error.
`

// Run executes args (without the program name) and returns the exit code.
func Run(ctx context.Context, args []string, env Env) int {
	if env.Stdout == nil {
		env.Stdout = io.Discard
	}
	if env.Stderr == nil {
		env.Stderr = io.Discard
	}
	if env.WriteFile == nil {
		env.WriteFile = writeExclusive
	}
	if len(args) == 0 {
		fmt.Fprint(env.Stderr, usage)
		return 2
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(env.Stdout, usage)
		return 0
	case "parse":
		return cmdParse(args[1:], env)
	case "render":
		return cmdRender(args[1:], env)
	case "alias", "request", "claim":
		if env.Service == nil {
			fmt.Fprintln(env.Stderr, "error: this command needs a service attached by the host application (not available in the standalone binary)")
			return 1
		}
		return cmdStateful(ctx, args, env)
	}
	fmt.Fprintf(env.Stderr, "unknown command %q; run 'circula-qr help'\n", args[0])
	return 2
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func newFlags(name string, env Env) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(env.Stderr)
	return fs
}

func hashPrefix(code string) string {
	h := sha256.Sum256([]byte(code))
	return hex.EncodeToString(h[:4])
}

func cmdParse(args []string, env Env) int {
	if len(args) != 1 {
		fmt.Fprintln(env.Stderr, "usage: circula-qr parse <code>")
		return 2
	}
	p, err := qr.ParseManual(env.Origin, args[0])
	if err != nil {
		fmt.Fprintln(env.Stderr, "invalid: not a Circula QR link or code for the trusted origin")
		return 1
	}
	// The code itself is not echoed: only its kind and a short non-reversible fingerprint.
	fmt.Fprintf(env.Stdout, "kind=%s\nfingerprint=%s\nvalid=true\n", p.Kind, hashPrefix(p.Code))
	return 0
}

func cmdRender(args []string, env Env) int {
	fs := newFlags("render", env)
	out := fs.String("out", "", "output PNG path (must not exist)")
	level := fs.String("level", "h", "error correction: m, q or h")
	px := fs.Int("pixels", 512, "maximum image size in pixels")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 || *out == "" {
		fmt.Fprintln(env.Stderr, "usage: circula-qr render --out FILE <link>")
		return 2
	}
	lv := map[string]qr.ECLevel{"m": qr.ECMedium, "q": qr.ECQuart, "h": qr.ECHigh}[strings.ToLower(*level)]
	if lv == 0 {
		fmt.Fprintln(env.Stderr, "error: --level must be m, q or h")
		return 2
	}
	if _, err := qr.ParseLink(env.Origin, fs.Arg(0)); err != nil {
		fmt.Fprintln(env.Stderr, "error: only links on the trusted origin can be rendered")
		return 1
	}
	b, err := qr.RenderPNG(fs.Arg(0), lv, *px)
	if err != nil {
		fmt.Fprintln(env.Stderr, "error: cannot render (size out of range?)")
		return 1
	}
	if err := env.WriteFile(*out, b); err != nil {
		fmt.Fprintf(env.Stderr, "error: cannot write %s: %v\n", *out, err)
		return 1
	}
	fmt.Fprintf(env.Stdout, "wrote %s (%d bytes)\n", *out, len(b))
	return 0
}

type actor struct {
	network, member, key, out *string
}

func actorFlags(fs *flag.FlagSet) actor {
	return actor{fs.String("network", "", "network id"), fs.String("member", "", "acting member id"), fs.String("key", "", "idempotency key"), fs.String("out", "", "write a PNG here (must not exist)")}
}

func (a actor) principal() (qr.Principal, bool) {
	p := qr.Principal{NetworkID: *a.network, MemberID: *a.member}
	return p, p.NetworkID != "" && p.MemberID != ""
}

func fail(env Env, err error) int {
	msg := "failed"
	switch {
	case errors.Is(err, qr.ErrInvalid):
		msg = strings.TrimPrefix(strings.TrimPrefix(err.Error(), qr.ErrInvalid.Error()), ": ")
		if msg == "" {
			msg = "invalid input"
		}
	case errors.Is(err, qr.ErrNotFound):
		msg = "not found in this network"
	case errors.Is(err, qr.ErrForbidden):
		msg = "forbidden"
	case errors.Is(err, qr.ErrExpired):
		msg = "request expired"
	case errors.Is(err, qr.ErrRevoked):
		msg = "code revoked"
	case errors.Is(err, qr.ErrClaimed):
		msg = "request already claimed"
	case errors.Is(err, qr.ErrIdempotencyConflict):
		msg = "idempotency key was used for different input"
	case errors.Is(err, qr.ErrRateLimited):
		msg = "too many active codes"
	case errors.Is(err, qr.ErrInsufficientCredit):
		msg = "insufficient available credit"
	case errors.Is(err, qr.ErrCounterpartyInactive):
		msg = "the other member cannot trade right now"
	}
	fmt.Fprintln(env.Stderr, "error:", msg)
	return 1
}

func (a actor) writePNG(env Env, svc *qr.Service, link string) error {
	if *a.out == "" {
		return nil
	}
	b, err := svc.RenderURL(link)
	if err != nil {
		return err
	}
	return env.WriteFile(*a.out, b)
}

func printRequest(env Env, v qr.RequestView) {
	fmt.Fprintf(env.Stdout, "id=%s\nstatus=%s\namount_minor=%d\nexpires_at=%s\n", v.ID, v.Status, v.AmountMinor, v.ExpiresAt.Format(time.RFC3339))
	if v.Note != "" {
		fmt.Fprintf(env.Stdout, "note=%s\n", v.Note)
	}
	if v.TransactionID != "" {
		fmt.Fprintf(env.Stdout, "transaction_id=%s\n", v.TransactionID)
	}
	fmt.Fprintf(env.Stdout, "completed=%t\n", v.Status == qr.StatusCompleted)
}

func cmdStateful(ctx context.Context, args []string, env Env) int {
	svc := env.Service
	sub := ""
	rest := args[1:]
	if args[0] != "claim" {
		if len(rest) == 0 {
			fmt.Fprint(env.Stderr, usage)
			return 2
		}
		sub, rest = rest[0], rest[1:]
	}
	fs := newFlags(args[0]+" "+sub, env)
	a := actorFlags(fs)
	amount := fs.Int64("amount", 0, "amount in minor units")
	note := fs.String("note", "", "note")
	ttl := fs.Duration("ttl", 0, "validity, e.g. 15m, 24h")
	limit := fs.Int("limit", 25, "page size")
	cursor := fs.String("cursor", "", "page cursor")
	// Go's flag package stops at the first positional argument; accept flags on either side.
	var pos []string
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			return 2
		}
		rest = fs.Args()
		if len(rest) > 0 {
			pos, rest = append(pos, rest[0]), rest[1:]
		}
	}
	p, ok := a.principal()
	if !ok {
		fmt.Fprintln(env.Stderr, "error: --network and --member are required")
		return 2
	}
	switch args[0] + " " + sub {
	case "alias show":
		al, err := svc.MyAlias(ctx, p)
		if err != nil {
			return fail(env, err)
		}
		if err := a.writePNG(env, svc, al.URL); err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "generation=%d\nurl=%s\ncode=%s\n", al.Generation, al.URL, al.Compact)
	case "alias rotate":
		al, err := svc.RotateAlias(ctx, p, *a.key)
		if err != nil {
			return fail(env, err)
		}
		if err := a.writePNG(env, svc, al.URL); err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "generation=%d\nurl=%s\ncode=%s\nold_codes=revoked\n", al.Generation, al.URL, al.Compact)
	case "request create":
		c, err := svc.CreateRequest(ctx, p, *a.key, qr.CreateRequestInput{AmountMinor: *amount, Note: *note, TTL: *ttl})
		if err != nil {
			return fail(env, err)
		}
		printRequest(env, c.View)
		if c.Replayed {
			fmt.Fprintln(env.Stdout, "token_available=false\nnote=this key already created the request; the one-time link was shown only then. Revoke it and create a new one if it was lost.")
			return 0
		}
		if err := a.writePNG(env, svc, c.URL); err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "url=%s\ncode=%s\nwarning=shown once; never stored; treat as a bearer token\n", c.URL, c.Compact)
	case "request show":
		if len(pos) != 1 {
			fmt.Fprintln(env.Stderr, "usage: request show ID")
			return 2
		}
		v, err := svc.GetRequest(ctx, p, pos[0])
		if err != nil {
			return fail(env, err)
		}
		printRequest(env, v)
	case "request list":
		items, next, err := svc.ListRequests(ctx, p, *limit, *cursor)
		if err != nil {
			return fail(env, err)
		}
		for _, v := range items {
			fmt.Fprintf(env.Stdout, "%s\t%s\t%d\t%s\n", v.ID, v.Status, v.AmountMinor, v.ExpiresAt.Format(time.RFC3339))
		}
		if next != "" {
			fmt.Fprintf(env.Stdout, "next_cursor=%s\n", next)
		}
	case "request revoke":
		if len(pos) != 1 {
			fmt.Fprintln(env.Stderr, "usage: request revoke ID")
			return 2
		}
		v, err := svc.RevokeRequest(ctx, p, pos[0])
		if err != nil {
			return fail(env, err)
		}
		printRequest(env, v)
	case "request preview":
		if len(pos) != 1 {
			fmt.Fprintln(env.Stderr, "usage: request preview <code>")
			return 2
		}
		parsed, err := qr.ParseManual(env.Origin, pos[0])
		if err != nil {
			return fail(env, qr.ErrInvalid)
		}
		var pv qr.Preview
		if parsed.Kind == qr.KindAlias {
			pv, err = svc.ResolveAlias(ctx, p, parsed.Code)
		} else {
			pv, err = svc.PreviewRequest(ctx, p, parsed.Code)
		}
		if err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "kind=%s\nnetwork=%s\npayer=%s\npayee=%s\namount_minor=%d\nestimated_available_credit_minor=%d\nnotice=scanning is not acceptance or payment\n",
			pv.Kind, pv.NetworkName, pv.PayerName, pv.PayeeName, pv.AmountMinor, pv.AvailableCredit)
	case "claim ":
		if len(pos) != 1 {
			fmt.Fprintln(env.Stderr, "usage: claim <code> --network N --member M --key KEY")
			return 2
		}
		parsed, err := qr.ParseManual(env.Origin, pos[0])
		if err != nil {
			return fail(env, qr.ErrInvalid)
		}
		var res qr.ClaimResult
		if parsed.Kind == qr.KindAlias {
			res, err = svc.ProposeToAlias(ctx, p, parsed.Code, *amount, *note, *a.key)
		} else if *amount != 0 || *note != "" {
			return fail(env, fmt.Errorf("%w: a one-time request fixes its own amount and note", qr.ErrInvalid))
		} else {
			res, err = svc.ClaimRequest(ctx, p, parsed.Code, *a.key)
		}
		if err != nil {
			return fail(env, err)
		}
		fmt.Fprintf(env.Stdout, "transaction_id=%s\nstatus=%s\nreplayed=%t\nnote=%s\n", res.TransactionID, statusWord(res.State), res.Replayed,
			"a proposal only: the other member must accept before any payment")
	default:
		fmt.Fprint(env.Stderr, usage)
		return 2
	}
	return 0
}

func statusWord(s qr.TxState) string {
	switch s {
	case qr.TxCompleted:
		return "completed"
	case qr.TxClosed:
		return "closed"
	}
	return "pending"
}
