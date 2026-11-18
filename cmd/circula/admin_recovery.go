package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["admin-recovery"] = cmdAdminRecovery
	extraUsage = append(extraUsage, "  admin-recovery list|approve|refuse   dual-approved recovery of a network with no administrator (GOV-021); see docs/ops/admin-recovery.md")
}

const adminRecoveryUsage = `usage:
  circula admin-recovery list [-network SLUG] [-all]
  circula admin-recovery approve -network SLUG -request ID -operator NAME -reason TEXT
  circula admin-recovery refuse  -network SLUG -request ID -operator NAME -reason TEXT
A request is made by a member of the network. The first approval only records it; a second
approval by a different operator applies it. No command appoints an administrator otherwise.
`

// cmdAdminRecovery is the operator side of GOV-021. It never starts a request
// and never appoints anyone without a member's request and two approvals.
func cmdAdminRecovery(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, adminRecoveryUsage)
		return exitUsage
	}
	sub, rest := args[0], args[1:]
	name := "admin-recovery " + sub
	var network, request, operator, reason string
	var all bool
	cfg, _, code, done := resolveFlags(name, rest, stderr, func(fs *flag.FlagSet) {
		fs.StringVar(&network, "network", "", "network slug")
		fs.StringVar(&request, "request", "", "recovery request id")
		fs.StringVar(&operator, "operator", "", "name of the operator deciding (approvals need two different names)")
		fs.StringVar(&reason, "reason", "", "documented reason; shown to the network's administrators")
		fs.BoolVar(&all, "all", false, "list: include decided requests, not only open ones")
	})
	if done {
		return code
	}
	switch sub {
	case "list":
	case "approve", "refuse":
		if network == "" || request == "" || operator == "" || reason == "" {
			fmt.Fprintf(stderr, "circula: %s: -network, -request, -operator and -reason are required\n", name)
			return exitUsage
		}
	default:
		fmt.Fprint(stderr, adminRecoveryUsage)
		return exitUsage
	}
	ctx, stop := withInterrupt(ctx)
	defer stop()
	svc, release, code := openOperatorService(ctx, cfg, name, stderr)
	if release == nil {
		return code
	}
	defer release()
	actor := domain.SystemActor("cli-admin-recovery")
	fail := func(err error) int {
		fmt.Fprintf(stderr, "circula: %s: %v\n", name, err)
		return exitFailure
	}
	in := service.AdminRecoveryDecision{Network: network, RequestID: request, Operator: operator, Reason: reason}
	switch sub {
	case "list":
		rs, err := svc.ListAdminRecoveryOperator(ctx, actor, network, !all, service.Page{Limit: 200})
		if err != nil {
			return fail(err)
		}
		tw := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNETWORK\tSTATUS\tNO ADMIN\tREQUESTER\tNAME\tMEMBER STATUS\tJOINED\tREQUESTED\tEXPIRES\tFIRST APPROVAL")
		for _, v := range rs {
			r := v.Request
			first := "-"
			if r.FirstApprovedAt != nil {
				first = r.FirstOperator + " " + r.FirstApprovedAt.Format(time.RFC3339)
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", r.ID, v.NetworkSlug, v.Status, v.AdminRequired, r.RequesterID,
				v.RequesterName, v.RequesterStatus, v.RequesterJoined.Format("2006-01-02"), r.RequestedAt.Format(time.RFC3339),
				r.ExpiresAt.Format(time.RFC3339), first)
		}
		_ = tw.Flush()
	case "approve":
		out, err := svc.ApproveAdminRecovery(ctx, actor, in)
		if err != nil {
			return fail(err)
		}
		if out.Applied {
			fmt.Fprintf(stdout, "request %s applied: member %s is now an administrator of %s (approved by %s and %s)\n",
				out.Request.ID, out.Request.RequesterID, network, out.Request.FirstOperator, out.Request.ClosedBy)
		} else {
			fmt.Fprintf(stdout, "first approval of request %s recorded by %s; a different operator must approve before %s to apply it\n",
				out.Request.ID, out.Request.FirstOperator, out.Request.ExpiresAt.Format(time.RFC3339))
		}
	case "refuse":
		r, err := svc.RefuseAdminRecovery(ctx, actor, in)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(stdout, "request %s refused by %s; nothing changed\n", r.ID, r.ClosedBy)
	}
	return exitOK
}
