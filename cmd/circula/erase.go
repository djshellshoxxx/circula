package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["erase-member"] = cmdEraseMember
	extraUsage = append(extraUsage, "  erase-member                irreversibly anonymise a departed member (SEC-018); see docs/dev/erasure.md")
}

// cmdEraseMember anonymises one departed member. It cannot be undone, so it
// needs -confirm and a reason, and it writes the member.anonymised audit event.
func cmdEraseMember(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	var network, member, reason string
	var confirm bool
	cfg, _, code, done := resolveFlags("erase-member", args, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "network slug")
		fs.StringVar(&member, "member", "", "member id of a departed member")
		fs.StringVar(&reason, "reason", "", "reason, recorded in the audit log; do not put personal data in it")
		fs.BoolVar(&confirm, "confirm", false, "acknowledge that this is irreversible")
	})
	if done {
		return code
	}
	if network == "" || member == "" || reason == "" {
		fmt.Fprintln(stderr, "usage: circula erase-member -network SLUG -member ID -reason TEXT -confirm")
		return exitUsage
	}
	if !confirm {
		fmt.Fprintln(stderr, "circula: erase-member: this cannot be undone; re-run with -confirm")
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	svc, release, code := openOperatorService(ctx, cfg, "erase-member", stderr)
	if release == nil {
		return code
	}
	defer release()
	info, err := svc.NetworkBySlug(ctx, network)
	if err != nil {
		fmt.Fprintf(stderr, "circula: erase-member: %v\n", err)
		return exitFailure
	}
	rep, err := svc.AnonymiseMember(ctx, domain.SystemActor("cli-erase-member"), info.ID, member, reason)
	if err != nil {
		fmt.Fprintf(stderr, "circula: erase-member: %v\n", err)
		var de *domain.Error
		if errors.As(err, &de) {
			for _, b := range de.Blockers {
				fmt.Fprintf(stderr, "  blocked by: %s\n", b)
			}
		}
		return exitFailure
	}
	printErasure(stdout, rep)
	return exitOK
}

func printErasure(w io.Writer, r *service.ErasureReport) {
	fmt.Fprintf(w, "member %s anonymised as %q: %d listing(s), %d memo(s), %d dispute statement(s) cleared; person erased: %v\n",
		r.MemberID, r.DisplayName, r.Listings, r.Memos, r.DisputeStatements, r.PersonErased)
}
