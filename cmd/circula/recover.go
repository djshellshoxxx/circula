package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// cmdResetPassword is the operator's account recovery (SEC-008): it sets a
// random temporary password for --login, ends every session, API token and
// recovery code of that person, forces a password change at the next sign-in
// and records an audit event as the system actor with the mandatory reason.
// The temporary password goes to stdout exactly once and is never logged.
func cmdResetPassword(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var login, reason string
	cfg, _, code, done := resolveFlags("reset-password", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&login, "login", "", "login of the person whose password is reset (required)")
		fs.StringVar(&reason, "reason", "", "why the reset is done, recorded in the audit log (required)")
	})
	if done {
		return code
	}
	if login == "" || reason == "" {
		fmt.Fprintln(stderr, "usage: circula reset-password -login <login> -reason <text>")
		fmt.Fprintln(stderr, "circula: reset-password: -login and -reason are required")
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
		fmt.Fprintf(stderr, "circula: reset-password: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	temp, err := svc.OperatorResetPassword(ctx, domain.SystemActor("cli-reset-password"), login, reason)
	if err != nil {
		fmt.Fprintf(stderr, "circula: reset-password: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "temporary password for %s (shown once; the person must change it at next sign-in):\n%s\n", login, temp)
	return exitOK
}
