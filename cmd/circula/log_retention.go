package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Log retention (SEC-021, DEC-019). Circula writes logs to standard error; the
// service manager or a log shipper stores them. `prune-logs` removes stored log
// files older than the log-retention setting (default 90 days) so that logs
// kept on the host do not grow without bound. Run it daily from a timer.

func init() {
	extraCommands["prune-logs"] = cmdPruneLogs
	extraUsage = append(extraUsage, "  prune-logs -dir <dir>       delete log files older than log-retention (default 90 days)")
}

// isLogFile reports whether name looks like a log file or a rotated one.
func isLogFile(name string) bool {
	n := strings.ToLower(name)
	return strings.HasSuffix(n, ".log") || strings.Contains(n, ".log.") || strings.HasSuffix(n, ".jsonl")
}

// pruneLogs removes regular log files in dir whose modification time is more
// than retention before now. Directories, symbolic links and other files are
// left alone. It returns the names removed.
func pruneLogs(dir string, retention time.Duration, now time.Time) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, e := range ents {
		if !e.Type().IsRegular() || !isLogFile(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return removed, err
		}
		if now.Sub(info.ModTime()) <= retention {
			continue
		}
		if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
			return removed, err
		}
		removed = append(removed, e.Name())
	}
	return removed, nil
}

func cmdPruneLogs(_ context.Context, args []string, stdout, stderr io.Writer) int {
	var dir string
	cfg, rest, code, done := resolveFlags("prune-logs", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&dir, "dir", "", "directory holding log files")
	})
	if done {
		return code
	}
	if dir == "" || len(rest) != 0 {
		fmt.Fprintln(stderr, "usage: circula prune-logs [flags] -dir <directory>")
		return exitUsage
	}
	removed, err := pruneLogs(dir, cfg.LogRetention, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "circula: prune-logs: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "removed %d log file(s) older than %s\n", len(removed), cfg.LogRetention)
	return exitOK
}
