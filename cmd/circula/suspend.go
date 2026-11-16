package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["suspend"] = cmdSuspend
	extraUsage = append(extraUsage, "  suspend start|lift   platform suspension of a network (GOV-024); see docs/ops/runbook.md")
}

var suspendUsage = `usage:
  circula suspend start -network SLUG -category CATEGORY -reason TEXT -operator NAME
  circula suspend lift -network SLUG -reason TEXT -operator NAME
categories: ` + strings.Join(domain.SuspensionCategories, ", ") + `
`

func cmdSuspend(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, suspendUsage)
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	name := "suspend " + sub
	var network, reason, operator, category string
	cfg, _, code, done := resolveFlags(name, rest, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "network slug")
		fs.StringVar(&reason, "reason", "", "documented reason")
		fs.StringVar(&operator, "operator", "", "name of the operator")
		fs.StringVar(&category, "category", "", "suspension category")
	})
	if done {
		return code
	}
	if (sub != "start" && sub != "lift") || network == "" || reason == "" || operator == "" || (sub == "start" && category == "") {
		fmt.Fprint(stderr, suspendUsage)
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	svc, release, code := openOperatorService(ctx, cfg, name, stderr)
	if release == nil {
		return code
	}
	defer release()
	actor := domain.SystemActor("cli-suspend")
	var err error
	if sub == "start" {
		err = svc.PlatformSuspendNetwork(ctx, actor, service.PlatformSuspendInput{Network: network, Operator: operator, Category: category, Reason: reason})
	} else {
		err = svc.PlatformResumeNetwork(ctx, actor, network, operator, reason)
	}
	if err != nil {
		fmt.Fprintf(stderr, "circula: %s: %v\n", name, err)
		return exitFailure
	}
	fmt.Fprintf(stdout, "network %s: platform suspension %s\n", network, map[string]string{"start": "started", "lift": "lifted"}[sub])
	return exitOK
}
