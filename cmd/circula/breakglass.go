package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["break-glass"] = cmdBreakGlass
	extraUsage = append(extraUsage, "  break-glass open|close|list|grant-admin   operator break-glass sessions (SEC-013); see docs/dev/break-glass.md")
}

const breakGlassUsage = `usage:
  circula break-glass open -network SLUG -reason TEXT -operator NAME [-ttl 4h]   (ttl is capped at 4h)
  circula break-glass close -session ID [-reason TEXT]
  circula break-glass list -network SLUG
  circula break-glass grant-admin -network SLUG -login LOGIN -reason TEXT
`

// openOperatorService takes the single-writer lock, opens the database and
// builds the service for an operator command. The returned func releases both.
func openOperatorService(ctx context.Context, cfg *Config, name string, stderr io.Writer) (*service.Service, func(), int) {
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return nil, nil, reportLockError(stderr, err, cfg.Database)
	}
	st, svc, err := openDatabase(ctx, cfg)
	if err != nil {
		_ = lock.Release()
		fmt.Fprintf(stderr, "circula: %s: %v\n", name, err)
		return nil, nil, exitFailure
	}
	return svc, func() { _ = st.Close(); _ = lock.Release() }, exitOK
}

func cmdBreakGlass(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, breakGlassUsage)
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	name := "break-glass " + sub
	var network, reason, operator, session, login string
	var ttl time.Duration
	cfg, _, code, done := resolveFlags(name, rest, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "network slug")
		fs.StringVar(&reason, "reason", "", "documented reason")
		fs.StringVar(&operator, "operator", "", "name of the operator opening the session")
		fs.StringVar(&session, "session", "", "break-glass session id")
		fs.StringVar(&login, "login", "", "login of the member to receive a temporary admin role")
		fs.DurationVar(&ttl, "ttl", domain.BreakGlassDefaultTTL, "session length; at most 4h, longer values are reduced to 4h")
	})
	if done {
		return code
	}
	switch sub {
	case "open":
		if network == "" || reason == "" || operator == "" {
			fmt.Fprintln(stderr, "circula: break-glass open: -network, -reason and -operator are required")
			return exitUsage
		}
	case "close":
		if session == "" {
			fmt.Fprintln(stderr, "circula: break-glass close: -session is required")
			return exitUsage
		}
	case "list":
		if network == "" {
			fmt.Fprintln(stderr, "circula: break-glass list: -network is required")
			return exitUsage
		}
	case "grant-admin":
		if network == "" || login == "" || reason == "" {
			fmt.Fprintln(stderr, "circula: break-glass grant-admin: -network, -login and -reason are required")
			return exitUsage
		}
	default:
		fmt.Fprint(stderr, breakGlassUsage)
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	svc, release, code := openOperatorService(ctx, cfg, name, stderr)
	if release == nil {
		return code
	}
	defer release()
	actor := domain.SystemActor("cli-break-glass")
	fail := func(err error) int {
		fmt.Fprintf(stderr, "circula: %s: %v\n", name, err)
		return exitFailure
	}
	switch sub {
	case "open":
		b, err := svc.OpenBreakGlass(ctx, actor, service.OpenBreakGlassInput{Network: network, Operator: operator, Reason: reason, TTL: ttl})
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "break-glass session %s open for %s until %s\n", b.ID, b.NetworkID, b.ExpiresAt.Format(time.RFC3339))
	case "close":
		b, err := svc.CloseBreakGlass(ctx, actor, session, reason)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "break-glass session %s closed\n", b.ID)
	case "list":
		bs, err := svc.ListBreakGlassOperator(ctx, actor, network, service.Page{Limit: 200})
		if err != nil {
			return fail(err)
		}
		now := time.Now().UTC()
		tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tSTATUS\tOPERATOR\tOPENED\tEXPIRES\tREASON")
		for _, b := range bs {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", b.ID, b.Status(now), b.OperatorLabel,
				b.OpenedAt.Format(time.RFC3339), b.ExpiresAt.Format(time.RFC3339), b.Reason)
		}
		_ = tw.Flush()
	case "grant-admin":
		b, err := svc.GrantBreakGlassAdmin(ctx, actor, service.GrantBreakGlassInput{Network: network, Login: login, Reason: reason})
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "%s holds the admin role in %s until %s (session %s)\n", login, network, b.ExpiresAt.Format(time.RFC3339), b.ID)
	}
	return exitOK
}
