package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	qr "github.com/djshellshoxxx/circula/future/qr"
	"github.com/djshellshoxxx/circula/future/qr/cli"
)

const origin = "https://circula.example"

func run(t *testing.T, env cli.Env, args ...string) (int, string, string) {
	t.Helper()
	var o, e bytes.Buffer
	env.Stdout, env.Stderr = &o, &e
	if env.Origin == "" {
		env.Origin = origin
	}
	code := cli.Run(context.Background(), args, env)
	return code, o.String(), e.String()
}

func TestHelpAndUsageErrors(t *testing.T) {
	code, out, _ := run(t, cli.Env{}, "help")
	if code != 0 || !strings.Contains(out, "HOLD") || !strings.Contains(out, "PENDING") || !strings.Contains(out, "bearer token") {
		t.Fatal(code, out)
	}
	for _, args := range [][]string{{}, {"bogus"}, {"parse"}, {"parse", "a", "b"}, {"render"}, {"render", "--out", "x"}} {
		if c, _, _ := run(t, cli.Env{}, args...); c != 2 {
			t.Fatalf("%v -> %d", args, c)
		}
	}
	// stateful commands without an attached service fail clearly instead of touching anything
	code, _, errOut := run(t, cli.Env{}, "alias", "show", "--network", "n", "--member", "m")
	if code != 1 || !strings.Contains(errOut, "host application") {
		t.Fatal(code, errOut)
	}
}

func TestParseDoesNotEchoTheCode(t *testing.T) {
	tok, _ := qr.NewToken()
	link, _ := qr.Link(origin, tok)
	for _, in := range []string{link, " r." + tok + " ", "a." + tok} {
		code, out, errOut := run(t, cli.Env{}, "parse", in)
		if code != 0 || strings.Contains(out+errOut, tok) || !strings.Contains(out, "valid=true") {
			t.Fatalf("%q: %d %s %s", in, code, out, errOut)
		}
	}
	code, out, _ := run(t, cli.Env{}, "parse", "a."+tok)
	if !strings.Contains(out, "kind=alias") || code != 0 {
		t.Fatal(out)
	}
	for _, bad := range []string{"https://evil.example/qr/v1/" + tok, link + "?x=1", "javascript:alert(1)", "r.short"} {
		code, out, errOut := run(t, cli.Env{}, "parse", bad)
		if code != 1 || out != "" || strings.Contains(errOut, tok) {
			t.Fatalf("%q: %d %q %q", bad, code, out, errOut)
		}
	}
}

func TestRenderWritesPNGAndRefusesOverwriteAndHostileLinks(t *testing.T) {
	tok, _ := qr.NewToken()
	link, _ := qr.Link(origin, tok)
	dir := t.TempDir()
	out := filepath.Join(dir, "q.png")
	code, stdout, errOut := run(t, cli.Env{}, "render", "--out", out, "--level", "m", "--pixels", "300", link)
	if code != 0 || !strings.Contains(stdout, "wrote") {
		t.Fatal(code, stdout, errOut)
	}
	b, err := os.ReadFile(out)
	if err != nil || !bytes.HasPrefix(b, []byte("\x89PNG")) {
		t.Fatal(err)
	}
	if c, _, e := run(t, cli.Env{}, "render", "--out", out, link); c != 1 || !strings.Contains(e, "cannot write") {
		t.Fatalf("overwrote existing file: %d %s", c, e)
	}
	fresh := filepath.Join(dir, "f.png")
	for _, bad := range [][]string{
		{"render", "--out", fresh, "https://evil.example/qr/v1/" + tok},
		{"render", "--out", fresh, "not a url"},
		{"render", "--out", fresh, "--pixels", "99999", link},
	} {
		if c, _, _ := run(t, cli.Env{}, bad...); c != 1 {
			t.Fatalf("%v -> %d", bad, c)
		}
	}
	if c, _, _ := run(t, cli.Env{}, "render", "--out", fresh, "--level", "z", link); c != 2 {
		t.Fatal("bad level")
	}
	if _, err := os.Stat(fresh); err == nil {
		t.Fatal("file created for a rejected render")
	}
}
