package main

import (
	"context"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// cmdKeygen prints a fresh backup-encryption-key: 32 random bytes, base64.
// It touches neither configuration nor the database. Store the output in a
// 0600 file (backup-encryption-key-file) and in your password manager; a
// backup cannot be restored without it.
func cmdKeygen(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: circula keygen")
		return exitUsage
	}
	k, err := newBackupKey()
	if err != nil {
		fmt.Fprintf(stderr, "circula: keygen: %v\n", err)
		return exitFailure
	}
	fmt.Fprintln(stdout, k)
	return exitOK
}

// cmdSetupReset re-arms the one-time setup secret after it has created a
// network (SEC-028). It is audited as a system action and, like every writer,
// refuses to run beside `serve`.
func cmdSetupReset(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg, _, code, done := resolveFlags("setup-reset", args, stderr, nil)
	if done {
		return code
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
		fmt.Fprintf(stderr, "circula: setup-reset: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	was, err := svc.ResetSetupSecret(ctx, domain.SystemActor("cli-setup-reset"))
	if err != nil {
		fmt.Fprintf(stderr, "circula: setup-reset: %v\n", err)
		return exitFailure
	}
	if !was {
		fmt.Fprintln(stdout, "the setup secret has not been used (or was already reset); nothing to do")
		return exitOK
	}
	fmt.Fprintln(stdout, "the setup secret is valid again for exactly one more network; start serve and create it, then it is spent again")
	return exitOK
}
