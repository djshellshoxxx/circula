package receipts

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

// RunCLI uses the host CLI's authenticated actor and read-only service. No
// alternate login, token source or client-authoritative transaction input exists.
func RunCLI(ctx context.Context, s Service, actor Actor, args []string, out, help io.Writer) error {
	flags := flag.NewFlagSet("receipt", flag.ContinueOnError)
	flags.SetOutput(help)
	ref := flags.String("transaction", "", "Completed transaction reference")
	format := flags.String("format", "txt", "Output format: txt, json, html")
	explicit := flags.Bool("request-receipt", false, "Explicitly request this optional private receipt")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 || !*explicit || !safe(*ref) {
		return errors.New("receipt requires --transaction and --request-receipt")
	}
	return s.Download(ctx, actor, DownloadRequest{*ref, *format, true}, out)
}

// RunCertificateCLI is a separate, explicit server-side action. The receipt-only
// CLI above has no dependency on keys or this issuer.
func RunCertificateCLI(ctx context.Context, issuer Issuer, actor Actor, args []string, out, help io.Writer) error {
	flags := flag.NewFlagSet("certificate", flag.ContinueOnError)
	flags.SetOutput(help)
	ref := flags.String("transaction", "", "Completed transaction reference")
	explicit := flags.Bool("request-certificate", false, "Explicitly request this optional private signed receipt")
	if e := flags.Parse(args); e != nil {
		return e
	}
	if flags.NArg() != 0 || !*explicit || !safe(*ref) {
		return errors.New("certificate requires --transaction and --request-certificate")
	}
	token, e := issuer.Issue(ctx, actor, CertificateRequest{*ref, true})
	if e != nil {
		return e
	}
	_, e = io.WriteString(out, token)
	return e
}

// RunVerifierCLI reads only the explicitly named local certificate. Trust and
// expected scope are injected from authenticated local configuration, never a
// key embedded in the file. Output is a reason and interpretation, no payload.
func RunVerifierCLI(ctx context.Context, args []string, expectedIssuer, expectedNetwork string, trust TrustStore, now time.Time, out, help io.Writer) error {
	flags := flag.NewFlagSet("verify-certificate", flag.ContinueOnError)
	flags.SetOutput(help)
	file := flags.String("file", "", "Local certificate file; verification makes no network request")
	if e := flags.Parse(args); e != nil {
		return e
	}
	if flags.NArg() != 0 || *file == "" {
		return errors.New("verification requires --file")
	}
	input, e := os.Open(*file)
	if e != nil {
		return errors.New("certificate file unavailable")
	}
	defer input.Close()
	result := VerifyFile(ctx, input, expectedIssuer, expectedNetwork, trust, now)
	if _, e = fmt.Fprintln(out, result.Reason); e != nil {
		return e
	}
	if result.Manifest == nil {
		return errors.New("certificate verification failed")
	}
	_, e = fmt.Fprintln(out, "Valid issuer signature over a historical snapshot; does not prove current status, delivery, legal identity or cash settlement.")
	return e
}
