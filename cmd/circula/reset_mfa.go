package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// cmdResetMFA is the operator's second-factor recovery (A3-03) for a person who
// lost both their authenticator and their recovery codes: it removes the
// second factor, recovery codes and pending challenges of --login, ends every
// session, tells the person with a security notice and records an audit event
// as the system actor with the mandatory reason. The password is not changed;
// use reset-password as well if it may be known to someone else.
func cmdResetMFA(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var login, reason string
	cfg, _, code, done := resolveFlags("reset-mfa", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&login, "login", "", "login of the person whose second factor is removed (required)")
		fs.StringVar(&reason, "reason", "", "why it is done, recorded in the audit log (required)")
	})
	if done {
		return code
	}
	if login == "" || reason == "" {
		fmt.Fprintln(stderr, "usage: circula reset-mfa -login <login> -reason <text>")
		fmt.Fprintln(stderr, "circula: reset-mfa: -login and -reason are required")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	// Writes: refuse while `serve` (or another command) holds the database.
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()
	st, svc, err := openDatabase(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "circula: reset-mfa: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	if err := svc.OperatorResetMFA(ctx, domain.SystemActor("cli-reset-mfa"), login, reason); err != nil {
		fmt.Fprintf(stderr, "circula: reset-mfa: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "second factor of %s removed; every session ended. The person can sign in with the password and turn two-step sign-in on again.\n", login)
	return exitOK
}
