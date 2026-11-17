// Command circula-qr is the standalone, offline face of the QR module (help,
// parse, render). Stateful commands need a host-attached service and are not
// available here. Nothing is registered or enabled by building this program.
package main

import (
	"context"
	"os"

	"github.com/djshellshoxxx/circula/future/qr/cli"
)

func main() {
	origin := os.Getenv("CIRCULA_PUBLIC_ORIGIN")
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Env{Stdout: os.Stdout, Stderr: os.Stderr, Origin: origin}))
}
