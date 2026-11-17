package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/cli"
	"github.com/djshellshoxxx/circula/future/favorites/internal/testkit"
	"github.com/djshellshoxxx/circula/future/favorites/memstore"
)

func setup(t *testing.T) (*testkit.Directory, func(args ...string) (int, string, string)) {
	dir := testkit.NewDirectory()
	for _, id := range []string{"alice", "bob", "carol", "dave"} {
		dir.Add("n1", id)
	}
	dir.AddTrade("t1", favorites.HistoricalTrade{Network: "n1", Payer: "alice", Payee: "bob", Description: "bike", Completed: true})
	svc, err := favorites.NewService(favorites.Config{Store: memstore.New(), Policy: dir, Enabled: dir.Enabled, Trades: dir})
	if err != nil {
		t.Fatal(err)
	}
	return dir, func(args ...string) (int, string, string) {
		var o, e bytes.Buffer
		code := cli.Run(context.Background(), args, &o, &e, svc)
		return code, o.String(), e.String()
	}
}

func TestCLIHelpAndUsage(t *testing.T) {
	_, run := setup(t)
	if c, o, _ := run("help"); c != 0 || !strings.Contains(o, "Usage:") || !strings.Contains(o, "never change credit limits") && !strings.Contains(o, "never change") {
		t.Errorf("help: %d %q", c, o)
	}
	if c, _, e := run(); c != 2 || !strings.Contains(e, "Usage:") {
		t.Errorf("no args: %d", c)
	}
	for _, args := range [][]string{
		{"bogus"}, {"list"}, {"list", "--network", "n1"}, {"add", "--network", "n1", "--member", "alice"}, {"move", "--network", "n1", "--member", "alice", "--target", "bob"},
		{"move", "--network", "n1", "--member", "alice", "--target", "bob", "--up", "--down"}, {"draft", "--network", "n1", "--member", "alice"},
		{"draft", "--network", "n1", "--member", "alice", "--trade", "t1", "--target", "bob"}, {"list", "--network", "n1", "--member", "alice", "extra"},
		{"list", "--nope"}, {"departed", "--network", "n1"},
	} {
		if c, _, _ := run(args...); c != 2 {
			t.Errorf("%v: exit %d want 2", args, c)
		}
	}
}

func TestCLIWorkflow(t *testing.T) {
	dir, run := setup(t)
	base := []string{"--network", "n1", "--member", "alice"}
	call := func(cmd string, extra ...string) (int, string, string) {
		return run(append(append([]string{cmd}, base...), extra...)...)
	}
	for _, id := range []string{"bob", "carol", "dave"} {
		if c, _, e := call("add", "--target", id, "--nickname", "n-"+id); c != 0 {
			t.Fatalf("add %s: %s", id, e)
		}
	}
	if c, o, _ := call("add", "--target", "bob", "--nickname", "n-bob"); c != 0 || !strings.Contains(o, "already saved") {
		t.Errorf("idempotent add: %d %q", c, o)
	}
	if c, _, e := call("add", "--target", "alice"); c != 1 || !strings.Contains(e, "cannot be used") {
		t.Errorf("self: %d %q", c, e)
	}
	if c, _, e := call("add", "--target", "bob", "--nickname", "other"); c != 1 || !strings.Contains(e, "different settings") {
		t.Errorf("conflict: %d %q", c, e)
	}
	if c, _, e := call("move", "--target", "dave", "--up"); c != 0 {
		t.Fatal(e)
	}
	c, o, _ := call("list", "--json")
	var js struct {
		Items []struct{ Target string }
	}
	if err := json.Unmarshal([]byte(o), &js); err != nil || c != 0 || js.Items[1].Target != "dave" {
		t.Fatalf("order after move: %v %s", err, o)
	}
	if c, o, _ := call("move", "--target", "bob", "--up"); c != 0 || !strings.Contains(o, "edge") {
		t.Errorf("edge: %d %q", c, o)
	}
	if c, _, _ := call("pin", "--target", "carol"); c != 0 {
		t.Fatal("pin")
	}
	if _, o, _ := call("list"); !strings.Contains(o, "carol") || strings.Index(o, "carol") > strings.Index(o, "bob") {
		t.Errorf("pinned first: %q", o)
	}
	if c, _, _ := call("unpin", "--target", "carol"); c != 0 {
		t.Fatal("unpin")
	}
	if c, _, _ := call("rename", "--target", "carol", "--nickname", "a‮b"); c != 1 {
		t.Error("bad nickname accepted")
	}
	c, o, _ = call("draft", "--trade", "t1", "--key", "k1", "--json")
	var d map[string]any
	if err := json.Unmarshal([]byte(o), &d); err != nil || c != 0 || d["amount"] != nil || d["recipient_member_id"] != "bob" || d["description"] != "bike" {
		t.Fatalf("draft: %v %s", err, o)
	}
	_, o2, _ := call("draft", "--trade", "t1", "--key", "k1", "--json")
	_, o3, _ := call("draft", "--trade", "t1", "--key", "k2", "--json")
	if o2 != o && !strings.Contains(o2, d["idempotency_key"].(string)) || strings.Contains(o3, d["idempotency_key"].(string)) {
		t.Error("key semantics")
	}
	if c, o, _ := call("draft", "--target", "bob"); c != 0 || !strings.Contains(o, "nothing has been proposed") {
		t.Errorf("fav draft %d %q", c, o)
	}
	dir.Block("n1", "alice", "bob")
	if c, _, _ := call("draft", "--trade", "t1"); c != 1 {
		t.Error("blocked partner draft")
	}
	if _, o, _ := call("list"); strings.Contains(o, "Member bob") {
		t.Error("blocked listed")
	}
	if c, o, _ := call("remove", "--target", "bob"); c != 0 || !strings.Contains(o, "removed") {
		t.Errorf("remove blocked: %d %q", c, o)
	}
	if c, o, _ := call("remove", "--target", "bob"); c != 0 || !strings.Contains(o, "not in your list") {
		t.Errorf("second remove: %d %q", c, o)
	}
	dir.Fail.Store(true)
	if c, _, e := call("list"); c != 1 || !strings.Contains(e, "nothing was changed") || strings.Contains(e, "backend") {
		t.Errorf("policy failure: %d %q", c, e)
	}
	dir.Fail.Store(false)
	if c, o, _ := run("departed", "--network", "n1", "--member", "dave"); c != 0 || !strings.Contains(o, "removed 1 favorites") {
		t.Errorf("departed: %d %q", c, o)
	}
	if c, o, _ := run("purge-drafts"); c != 0 || !strings.Contains(o, "expired drafts") {
		t.Errorf("purge: %d %q", c, o)
	}
}
