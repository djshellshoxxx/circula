// Command circula runs and operates a Circula instance: the HTTP service, the
// database lifecycle (migrate, backup, restore, ledger verification) and
// operator administration. Configuration is documented in docs/ops/configuration.md.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
)

// Set at build time with -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Exit codes.
const (
	exitOK           = 0
	exitFailure      = 1
	exitUsage        = 2 // bad flags, invalid configuration
	exitLedgerFailed = 3 // ledger verification reported a fault (the operation itself completed)
)

func main() {
	restrictUmask()
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	ctx := context.Background()
	rest := args[1:]
	switch args[0] {
	case "serve":
		return cmdServe(rest, stdout, stderr)
	case "migrate":
		return cmdMigrate(ctx, rest, stdout, stderr)
	case "backup":
		return cmdBackup(ctx, rest, stdout, stderr)
	case "restore":
		return cmdRestore(ctx, rest, stdout, stderr)
	case "verify-ledger":
		return cmdVerifyLedger(ctx, rest, stdout, stderr)
	case "maintenance":
		return cmdMaintenance(ctx, rest, stdout, stderr)
	case "keygen":
		return cmdKeygen(rest, stdout, stderr)
	case "setup-reset":
		return cmdSetupReset(ctx, rest, stdout, stderr)
	case "admin":
		return cmdAdmin(ctx, rest, stdout, stderr)
	case "reset-password":
		return cmdResetPassword(ctx, rest, stdout, stderr)
	case "version", "--version":
		fmt.Fprintf(stdout, "circula %s (commit %s, built %s, %s %s/%s)\n",
			version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return exitOK
	case "help", "-h", "--help":
		usage(stdout)
		return exitOK
	default:
		if fn, ok := extraCommands[args[0]]; ok {
			return fn(ctx, rest, stdout, stderr)
		}
		fmt.Fprintf(stderr, "circula: unknown command %q\n\n", args[0])
		usage(stderr)
		return exitUsage
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: circula <command> [flags]

commands:
  serve                       run the HTTP service (use -check-config to validate only)
  migrate [-dry-run]          back up, then apply pending migrations, then verify the ledger
  backup <dest>               write a verified online snapshot and manifest (file or directory)
  restore <backup-file>       restore a backup into the configured database (stop serve first)
  verify-ledger [-network X]  recompute ledger invariants (read-only except the recorded run)
  maintenance on|off|status   toggle maintenance mode of a running instance (503 for web and API)
  setup-reset                 re-arm the one-time setup secret after it created a network (stop serve first)
  keygen                      print a fresh random key for backup-encryption-key
  admin create-operator       create a person from --login; password from env or stdin
  reset-password              set a temporary password: -login X -reason "why" (audited)
  version                     print version information
`)
	for _, l := range extraUsage {
		fmt.Fprintln(w, l)
	}
	fmt.Fprint(w, `
Every command accepts -config, and every setting may be given as a flag,
an environment variable (CIRCULA_<NAME>) or in the JSON config file.
Precedence: flags > environment > config file > defaults.
See docs/ops/configuration.md for the settings reference.
`)
}
