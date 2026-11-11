package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["revoke-sessions"] = cmdRevokeSessions
	extraUsage = append(extraUsage, "  revoke-sessions -reason X   sign everyone out (or one person: -login Y); -rotate-web-key also replaces the web signing key (audited, SEC-027, OPS-010)")
}

// cmdRevokeSessions is the operator's emergency sign-out and the session half
// of secret rotation (SEC-027, OPS-010). It revokes every live session, or
// those of -login, records one audit event as the system actor with the
// mandatory reason, and with -rotate-web-key replaces <state-dir>/circula-web.key
// so that CSRF tokens and signed cookies issued before stop validating too.
// Ledger data is never touched. Like the other write commands it needs the
// database lock, so `serve` must be stopped first.
func cmdRevokeSessions(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var login, reason string
	var rotate bool
	cfg, rest, code, done := resolveFlags("revoke-sessions", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&login, "login", "", "only end this person's sessions (default: every person)")
		fs.StringVar(&reason, "reason", "", "why it is done, recorded in the audit log (required)")
		fs.BoolVar(&rotate, "rotate-web-key", false, "also replace the web signing key in state-dir (not with -login)")
	})
	if done {
		return code
	}
	// flag stops at the first bare word, so an unquoted reason would silently
	// drop -login or -rotate-web-key after it and widen or weaken the
	// operation. Refuse before anything is revoked.
	if len(rest) != 0 {
		fmt.Fprintln(stderr, "usage: circula revoke-sessions -reason <text> [-login <login> | -rotate-web-key]")
		fmt.Fprintf(stderr, "circula: revoke-sessions: unexpected arguments %q (quote a reason with spaces)\n", rest)
		return exitUsage
	}
	if reason == "" || (rotate && login != "") {
		fmt.Fprintln(stderr, "usage: circula revoke-sessions -reason <text> [-login <login> | -rotate-web-key]")
		if reason == "" {
			fmt.Fprintln(stderr, "circula: revoke-sessions: -reason is required")
		} else {
			fmt.Fprintln(stderr, "circula: revoke-sessions: -rotate-web-key signs everyone out and cannot be combined with -login")
		}
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
		fmt.Fprintf(stderr, "circula: revoke-sessions: %v\n", err)
		return exitFailure
	}
	defer func() { _ = st.Close() }()
	n, err := svc.RevokeAllSessions(ctx, domain.SystemActor("cli-revoke-sessions"), service.RevokeSessionsInput{Login: login, Reason: reason, WebKeyRotated: rotate})
	if err != nil {
		fmt.Fprintf(stderr, "circula: revoke-sessions: %v\n", err)
		return exitFailure
	}
	who := "every person"
	if login != "" {
		who = login
	}
	fmt.Fprintf(stdout, "revoked %d session(s) of %s; API tokens are unchanged.\n", n, who)
	if rotate {
		if err := rotateWebKey(cfg.StateDir); err != nil {
			fmt.Fprintf(stderr, "circula: revoke-sessions: sessions were revoked, but the web signing key was not replaced: %v\n", err)
			return exitFailure
		}
		fmt.Fprintf(stdout, "web signing key %s replaced; forms and signed cookies issued before stop validating.\n",
			filepath.Join(cfg.StateDir, webKeyFile))
	}
	return exitOK
}

// rotateWebKey atomically replaces (or creates) the web signing key in
// stateDir with a fresh random key: a private temporary file is written,
// synced and renamed over the old one, so a reader sees either key, never a
// partial file. It never prints the key.
func rotateWebKey(stateDir string) error {
	if stateDir == "" {
		return fmt.Errorf("no state directory configured")
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return err
	}
	key := make([]byte, webKeyLen)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("web key: %w", err)
	}
	tmp, err := os.CreateTemp(stateDir, ".circula-web.key.*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(base64.StdEncoding.EncodeToString(key) + "\n"); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(stateDir, webKeyFile)); err != nil {
		return err
	}
	return syncDir(stateDir)
}

// syncDir makes a rename in dir durable, so a crash cannot bring the old key
// back. Directories cannot be synced on Windows; there it is a no-op.
func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	d, err := os.Open(dir) //nolint:gosec // G304: dir is the operator-configured state directory
	if err != nil {
		return err
	}
	defer func() { _ = d.Close() }()
	return d.Sync()
}
