package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func init() {
	extraCommands["unlock"] = cmdUnlock
	extraUsage = append(extraUsage, "  unlock -login X             clear the sign-in and recovery lockouts of a login, all addresses (audited, SA2-10)")
}

// cmdUnlock is the operator's answer to a locked-out person: it deletes every
// login, recovery and second-factor throttle row of --login and records an
// audit event as the system actor. It prints a count only, never a secret.
func cmdUnlock(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var login string
	cfg, _, code, done := resolveFlags("unlock", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&login, "login", "", "login whose lockouts are cleared (required)")
	})
	if done {
		return code
	}
	if login == "" {
		fmt.Fprintln(stderr, "usage: circula unlock -login <login>")
		fmt.Fprintln(stderr, "circula: unlock: -login is required")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	lock, err := acquireLock(lockPath(cfg.Database))
	if err != nil {
		return reportLockError(stderr, err, cfg.Database)
	}
	defer func() { _ = lock.Release() }()
	st, svc, err := openDatabase(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "circula: unlock: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	n, err := svc.OperatorUnlockLogin(ctx, domain.SystemActor("cli-unlock"), login)
	if err != nil {
		fmt.Fprintf(stderr, "circula: unlock: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "cleared %d lockout record(s) for %s\n", n, login)
	return exitOK
}
