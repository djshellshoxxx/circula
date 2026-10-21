package main

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

func init() {
	extraCommands["dual-control"] = cmdDualControl
	extraUsage = append(extraUsage, "  dual-control   operator: set or disable a network's dual-control threshold (SEC-011); administrators can only tighten it")
}

func cmdDualControl(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var network, reason string
	var threshold int64
	cfg, _, code, done := resolveFlags("dual-control", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "network slug")
		fs.StringVar(&reason, "reason", "", "documented reason")
		fs.Int64Var(&threshold, "threshold-minor", -1, "new threshold in minor units (e.g. cents); 0 disables")
	})
	if done {
		return code
	}
	if network == "" || reason == "" || threshold < 0 {
		fmt.Fprintln(stderr, "circula: dual-control: -network, -threshold-minor and -reason are required")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	svc, release, code := openOperatorService(ctx, cfg, "dual-control", stderr)
	if release == nil {
		return code
	}
	defer release()
	actor := domain.SystemActor("cli-dual-control")
	if err := svc.OperatorSetDualControl(ctx, actor, network, threshold, reason); err != nil {
		fmt.Fprintf(stderr, "circula: dual-control: %v\n", err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "dual-control threshold of %s set to %d minor units\n", network, threshold)
	return exitOK
}
