package governance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixtureAuth struct {
	mu         sync.Mutex
	members    map[string]Principal
	electorate []string
	fail       bool
}

func (f *fixtureAuth) Principal(_ context.Context, _ *sql.Tx, id Identity) (Principal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if id.Network != "n" || f.fail {
		return Principal{}, ErrDenied
	}
	return f.members[id.Member], nil
}
func (f *fixtureAuth) Electorate(context.Context, *sql.Tx, string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.electorate...), nil
}
func (f *fixtureAuth) PolicyExists(_ context.Context, _ *sql.Tx, id Identity, version string) (bool, error) {
	return id.Network == "n" && version == "policy-2", nil
}
func newFixture(t *testing.T) (Service, *fixtureAuth, *time.Time, string) {
	t.Helper()
	now := time.Unix(1000, 0).UTC()
	path := filepath.Join(t.TempDir(), "governance.db")
	auth := &fixtureAuth{members: map[string]Principal{"a": {true, false, true}, "b": {true, false, true}, "admin": {true, true, true}, "reviewer": {true, true, true}}, electorate: []string{"a", "b", "admin", "reviewer"}}
	s := openService(t, path, auth, &now)
	return s, auth, &now, path
}
func openService(t *testing.T, path string, auth *fixtureAuth, now *time.Time) Service {
	t.Helper()
	db, e := sql.Open("sqlite", "file:"+path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if e != nil {
		t.Fatal(e)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	s := Service{DB: db, Authority: auth, Now: func() time.Time { return *now }}
	if e = s.Migrate(context.Background()); e != nil {
		t.Fatal(e)
	}
	return s
}
func command(t *testing.T, s Service, member string, c Command) Reply {
	t.Helper()
	r, e := s.Execute(context.Background(), Identity{"n", member}, c)
	if e != nil {
		t.Fatalf("%s %+v: %v", member, c, e)
	}
	return r
}
func opened(t *testing.T, s Service, now time.Time) Reply {
	t.Helper()
	r := command(t, s, "a", Command{Action: "create", Intent: "create", Title: "Should we update?", Body: "Plain context", Category: "policy", Comments: true})
	return command(t, s, "a", Command{Action: "open", Intent: "open", Proposal: r.Proposal, Revision: r.Revision, Opens: now, Closes: now.Add(time.Hour)})
}
func TestGovernanceLifecycleIdempotencyPrivacy(t *testing.T) {
	s, auth, now, path := newFixture(t)
	r := opened(t, s, *now)
	vote := Command{Action: "vote", Intent: "vote-a", Proposal: r.Proposal, Revision: r.Revision, Choice: Yes}
	reply := command(t, s, "a", vote)
	again := command(t, s, "a", vote)
	if reply != again {
		t.Fatal("unstable retry")
	}
	vote.Choice = No
	if _, e := s.Execute(context.Background(), Identity{"n", "a"}, vote); !errors.Is(e, ErrConflict) {
		t.Fatal("changed intent", e)
	}
	vote.Intent = "duplicate"
	if _, e := s.Execute(context.Background(), Identity{"n", "a"}, vote); e == nil {
		t.Fatal("duplicate fresh vote")
	}
	command(t, s, "b", Command{Action: "vote", Intent: "vote-b", Proposal: r.Proposal, Revision: r.Revision, Choice: Abstain})
	d, e := s.Read(context.Background(), Identity{"n", "a"}, r.Proposal)
	if e != nil || d.OwnVote == nil || d.OwnVote.Choice != Yes || d.Result != nil {
		t.Fatal(d, e)
	}
	raw, _ := jsonBytes(d)
	if strings.Contains(string(raw), "vote-b") || strings.Contains(string(raw), `"Member":"b"`) {
		t.Fatal("other ballot leaked", string(raw))
	}
	if _, e = s.Read(context.Background(), Identity{"other", "a"}, r.Proposal); e == nil {
		t.Fatal("tenant read")
	}
	if e = s.DB.Close(); e != nil {
		t.Fatal(e)
	}
	s = openService(t, path, auth, now)
	*now = now.Add(time.Hour)
	closeReply := command(t, s, "a", Command{Action: "close", Intent: "close", Proposal: r.Proposal, Revision: r.Revision})
	d, e = s.Read(context.Background(), Identity{"n", "b"}, r.Proposal)
	if e != nil || d.Result == nil || d.Result.Outcome.Yes != 1 || d.Result.Outcome.Abstain != 1 || d.Result.Outcome.Eligible != 4 || !d.Result.Outcome.Advisory {
		t.Fatal(d, e)
	}
	command(t, s, "admin", Command{Action: "certify", Intent: "cert", Proposal: r.Proposal, Revision: closeReply.Revision})
	command(t, s, "admin", Command{Action: "link", Intent: "link", Proposal: r.Proposal, Revision: closeReply.Revision, PolicyVersion: "policy-2"})
	d, e = s.Read(context.Background(), Identity{"n", "b"}, r.Proposal)
	if e != nil || d.Result.PolicyVersion != "policy-2" {
		t.Fatal(d, e)
	}
}
func jsonBytes(v any) ([]byte, error) { return json.Marshal(v) }
func TestFrozenEligibilitySuspensionAndDraftEdits(t *testing.T) {
	s, auth, now, _ := newFixture(t)
	r := opened(t, s, *now)
	auth.mu.Lock()
	auth.electorate = []string{"a"}
	auth.members["new"] = Principal{true, false, true}
	auth.members["b"] = Principal{false, false, true}
	auth.mu.Unlock()
	for _, member := range []string{"b", "new"} {
		if _, e := s.Execute(context.Background(), Identity{"n", member}, Command{Action: "vote", Intent: "v-" + member, Proposal: r.Proposal, Revision: r.Revision, Choice: Yes}); e == nil {
			t.Fatal("unauthorized voter", member)
		}
	}
	if _, e := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "edit", Intent: "edit", Proposal: r.Proposal, Revision: r.Revision, Title: "changed", Body: "changed", Category: "other"}); e == nil {
		t.Fatal("mutable open question")
	}
	command(t, s, "a", Command{Action: "withdraw", Intent: "withdraw", Proposal: r.Proposal, Revision: r.Revision, Text: "withdrawn"})
	if _, e := s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "vote", Intent: "late", Proposal: r.Proposal, Revision: r.Revision + 1, Choice: Yes}); e == nil {
		t.Fatal("withdrawn vote")
	}
}
func TestDisputesModerationAndIndependentCertification(t *testing.T) {
	s, _, now, _ := newFixture(t)
	r := opened(t, s, *now)
	comment := command(t, s, "b", Command{Action: "comment", Intent: "comment", Proposal: r.Proposal, Revision: r.Revision, Text: "<script>untrusted</script>"})
	command(t, s, "admin", Command{Action: "hide", Intent: "hide", Proposal: r.Proposal, Revision: r.Revision, Comment: comment.Reference, Text: "moderation reason"})
	dispute := command(t, s, "b", Command{Action: "dispute", Intent: "dispute", Proposal: r.Proposal, Revision: r.Revision, Challenged: "admin", Text: "challenge moderation"})
	*now = now.Add(time.Hour)
	closed := command(t, s, "a", Command{Action: "close", Intent: "close", Proposal: r.Proposal, Revision: r.Revision})
	if _, e := s.Execute(context.Background(), Identity{"n", "reviewer"}, Command{Action: "certify", Intent: "cert", Proposal: r.Proposal, Revision: closed.Revision}); e == nil {
		t.Fatal("unresolved dispute certified")
	}
	if _, e := s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "resolve", Intent: "resolve-own", Proposal: r.Proposal, Revision: closed.Revision, Dispute: dispute.Reference, Text: "own action valid"}); e == nil {
		t.Fatal("reviewer conflict")
	}
	command(t, s, "reviewer", Command{Action: "resolve", Intent: "resolve", Proposal: r.Proposal, Revision: closed.Revision, Dispute: dispute.Reference, Text: "independent finding"})
	if _, e := s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "certify", Intent: "cert-own", Proposal: r.Proposal, Revision: closed.Revision}); e == nil {
		t.Fatal("challenged certifier")
	}
	command(t, s, "reviewer", Command{Action: "certify", Intent: "cert-clean", Proposal: r.Proposal, Revision: closed.Revision})
	d, e := s.Read(context.Background(), Identity{"n", "a"}, r.Proposal)
	if e != nil || len(d.Comments) != 0 || len(d.Disputes) != 0 || d.OpenDisputes != 0 {
		t.Fatal(d, e)
	}
}
func TestConcurrentVotesAndRollback(t *testing.T) {
	s, auth, now, path := newFixture(t)
	r := opened(t, s, *now)
	other := openService(t, path, auth, now)
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			target := s
			if i%2 == 1 {
				target = other
			}
			_, e := target.Execute(context.Background(), Identity{"n", "b"}, Command{Action: "vote", Intent: fmt.Sprintf("v-%d", i), Proposal: r.Proposal, Revision: r.Revision, Choice: Yes})
			if e == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if success != 1 {
		t.Fatal("concurrent successes", success)
	}
	if _, e := s.DB.Exec(`CREATE TRIGGER fail_audit BEFORE INSERT ON future_gov_audit BEGIN SELECT RAISE(ABORT,'injected audit failure'); END`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "vote", Intent: "rollback", Proposal: r.Proposal, Revision: r.Revision, Choice: No}); e == nil {
		t.Fatal("injected failure succeeded")
	}
	var count int
	if e := s.DB.QueryRow(`SELECT COUNT(*) FROM future_gov_ballots WHERE member='admin'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("ballot leaked through rollback", count, e)
	}
}
func TestValidationAndNoJSRendering(t *testing.T) {
	s, _, _, _ := newFixture(t)
	for _, title := range []string{"", strings.Repeat("x", 201), "spoof\u202econtrol", "line\nbreak"} {
		if _, e := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "create", Intent: "bad", Title: title, Body: "context", Category: "policy"}); e == nil {
			t.Fatal("invalid title", title)
		}
	}
	var out bytes.Buffer
	d := Detail{Proposal: Proposal{ID: "p", Title: "<script>alert(1)</script>", Body: "x", State: "open", Revision: 2}}
	if e := Render(&out, Page{Detail: d, CSRF: "token", Intent: "intent", Member: "a"}); e != nil {
		t.Fatal(e)
	}
	html := out.String()
	if strings.Contains(html, "<script>") || strings.Contains(html, " checked") || strings.Count(html, `name="choice"`) != 3 || !strings.Contains(html, `aria-describedby="ballot-hint"`) {
		t.Fatal("ballot rendering", html)
	}
}

func TestMigrationRollbackAndBackupRestore(t *testing.T) {
	s, auth, now, path := newFixture(t)
	r := opened(t, s, *now)
	command(t, s, "b", Command{Action: "vote", Intent: "before-backup", Proposal: r.Proposal, Revision: r.Revision, Choice: No})
	backup := path + ".backup"
	if _, e := s.DB.Exec(`VACUUM INTO ?`, backup); e != nil {
		t.Fatal(e)
	}
	restored := openService(t, backup, auth, now)
	d, e := restored.Read(context.Background(), Identity{"n", "b"}, r.Proposal)
	if e != nil || d.OwnVote == nil || d.OwnVote.Choice != No || d.Proposal.Eligible != 4 {
		t.Fatal("restore", d, e)
	}
	bad, e := sql.Open("sqlite", "file:"+filepath.Join(t.TempDir(), "bad")+"?_pragma=foreign_keys(1)")
	if e != nil {
		t.Fatal(e)
	}
	defer bad.Close()
	bad.SetMaxOpenConns(1)
	if _, e = bad.Exec(`CREATE TABLE future_gov_schema(version INTEGER PRIMARY KEY);INSERT INTO future_gov_schema VALUES(999)`); e != nil {
		t.Fatal(e)
	}
	badService := Service{DB: bad}
	if e = badService.Migrate(context.Background()); e == nil {
		t.Fatal("unsupported migration")
	}
	var count int
	if e = bad.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name='future_gov_ballots'`).Scan(&count); e != nil || count != 0 {
		t.Fatal("failed migration left schema", count, e)
	}
}

func TestEmptyOutcomeAndForgedPolicyLink(t *testing.T) {
	s, _, now, _ := newFixture(t)
	r := opened(t, s, *now)
	if _, e := s.Execute(context.Background(), Identity{"n", "b"}, Command{Action: "close", Intent: "premature-close", Proposal: r.Proposal, Revision: r.Revision}); e == nil {
		t.Fatal("unauthorized early close")
	}
	if _, e := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "close", Intent: "early-close", Proposal: r.Proposal, Revision: r.Revision}); e == nil {
		t.Fatal("early close")
	}
	*now = now.Add(time.Hour)
	closed := command(t, s, "a", Command{Action: "close", Intent: "empty-close", Proposal: r.Proposal, Revision: r.Revision})
	d, e := s.Read(context.Background(), Identity{"n", "b"}, r.Proposal)
	if e != nil || d.Result == nil || d.Result.Outcome.Yes != 0 || d.Result.Outcome.No != 0 || d.Result.Outcome.Abstain != 0 || d.Result.Outcome.Eligible != 4 {
		t.Fatal(d, e)
	}
	if _, e = s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "link", Intent: "uncertified-link", Proposal: r.Proposal, Revision: closed.Revision, PolicyVersion: "policy-2"}); e == nil {
		t.Fatal("uncertified link")
	}
	command(t, s, "admin", Command{Action: "certify", Intent: "cert-empty", Proposal: r.Proposal, Revision: closed.Revision})
	if _, e = s.Execute(context.Background(), Identity{"n", "admin"}, Command{Action: "link", Intent: "forged-link", Proposal: r.Proposal, Revision: closed.Revision, PolicyVersion: "forged-other-network"}); e == nil {
		t.Fatal("forged policy version")
	}
	d, e = s.Read(context.Background(), Identity{"n", "b"}, r.Proposal)
	if e != nil || d.Result.PolicyVersion != "" {
		t.Fatal(d, e)
	}
}
