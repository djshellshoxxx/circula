package governance

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAuditSilentWritesRollback(t *testing.T) {
	for _, table := range []string{"future_gov_ballots", "future_gov_intents", "future_gov_audit", "future_gov_results"} {
		t.Run(table, func(t *testing.T) {
			s, _, now, _ := newFixture(t)
			opened := opened(t, s, *now)
			if _, err := s.DB.Exec(`CREATE TRIGGER ignored_write BEFORE INSERT ON ` + table + ` BEGIN SELECT RAISE(IGNORE); END`); err != nil {
				t.Fatal(err)
			}
			command := Command{Action: "vote", Proposal: opened.Proposal, Revision: opened.Revision, Choice: Yes, Intent: "ignored"}
			if table == "future_gov_results" {
				*now = now.Add(2 * time.Hour)
				command.Action = "close"
			}
			if _, err := s.Execute(context.Background(), Identity{"n", "a"}, command); err == nil {
				t.Fatal("silently ignored write acknowledged")
			}
			var count int
			if err := s.DB.QueryRow(`SELECT COUNT(*) FROM future_gov_ballots`).Scan(&count); err != nil || count != 0 {
				t.Fatal(count, err)
			}
			if err := s.DB.QueryRow(`SELECT COUNT(*) FROM future_gov_intents WHERE intent='ignored'`).Scan(&count); err != nil || count != 0 {
				t.Fatal(count, err)
			}
		})
	}
}
func TestAuditRejectsSchemaShapeAndCorruptScope(t *testing.T) {
	s, _, _, _ := newFixture(t)
	if _, err := s.DB.Exec(`DROP INDEX future_gov_owner`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB.Exec(`CREATE INDEX future_gov_owner ON future_gov_proposals(author)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(context.Background()); err == nil {
		t.Fatal("incompatible existing index accepted")
	}
	s, _, _, _ = newFixture(t)
	r := command(t, s, "a", Command{Action: "create", Intent: "c", Title: "Private", Body: "Body", Category: "policy"})
	var raw []byte
	if err := s.DB.QueryRow(`SELECT body FROM future_gov_proposals WHERE id=?`, r.Proposal).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var p Proposal
	json.Unmarshal(raw, &p)
	p.Network = "other"
	p.State = "open"
	raw, _ = json.Marshal(p)
	if _, err := s.DB.Exec(`UPDATE future_gov_proposals SET body=? WHERE id=?`, raw, r.Proposal); err != nil {
		t.Fatal(err)
	}
	if list, err := s.List(context.Background(), Identity{"n", "b"}, "", 100); err != nil || len(list) != 0 {
		t.Fatal("private SQL draft should remain excluded", list, err)
	}
	if list, err := s.List(context.Background(), Identity{"n", "a"}, "", 100); err == nil || len(list) != 0 {
		t.Fatal("corrupt scoped body accepted", list, err)
	}
}
func TestAuditBoundedStrictInputsAndSafePath(t *testing.T) {
	for _, raw := range [][]byte{[]byte(`{"Action":"create","action":"vote"}`), []byte(`{"Body":"\ud800"}`), []byte{'{', '"', 'x', '"', ':', '"', 255, '"', '}'}} {
		var c Command
		if strictDecode(raw, &c) == nil {
			t.Fatal("ambiguous JSON accepted", string(raw))
		}
	}
	s, _, _, _ := newFixture(t)
	web := Web{s, webFixture{}, bytes.Repeat([]byte("k"), 32)}
	for _, path := range []string{"//evil.test/send", "/../send", "/\\evil.test"} {
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("X-Test-Member", "a")
		out := httptest.NewRecorder()
		web.ServeHTTP(out, r)
		if out.Code != 400 {
			t.Fatal(path, out.Code, out.Body.String())
		}
	}
	raw := `{"Action":"create","action":"vote","Intent":"i"}`
	r := httptest.NewRequest("POST", "/", strings.NewReader(raw))
	r.Header.Set("X-Test-Member", "a")
	r.Header.Set("X-Test-CSRF", "valid")
	r.Header.Set("Idempotency-Key", "i")
	out := httptest.NewRecorder()
	web.ServeHTTP(out, r)
	if out.Code != 400 {
		t.Fatal(out.Code, out.Body.String())
	}
	var b bytes.Buffer
	if err := RunCLI(context.Background(), s, Identity{"n", "a"}, []string{"command"}, strings.NewReader(raw), &b); err == nil || b.Len() != 0 {
		t.Fatal("CLI ambiguous input", err)
	}
}
func TestAuditClosedHistoryQuotaAndSafety(t *testing.T) {
	s, _, _, _ := newFixture(t)
	for i := 0; i < 100; i++ {
		r := command(t, s, "a", Command{Action: "create", Intent: fmt.Sprint("c", i), Title: "T", Body: "B", Category: "p"})
		command(t, s, "a", Command{Action: "withdraw", Proposal: r.Proposal, Revision: r.Revision, Text: "Withdraw", Intent: fmt.Sprint("w", i)})
	}
	if _, err := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "create", Intent: "overflow", Title: "T", Body: "B", Category: "p"}); err == nil {
		t.Fatal("withdraw churn bypasses proposal bound")
	}
}

type borrowedElectorate struct {
	*fixtureAuth
	shared []string
}

func (a borrowedElectorate) Electorate(context.Context, *sql.Tx, string) ([]string, error) {
	return a.shared, nil
}
func TestAuditElectorateCopyAndCurrentAdminReplay(t *testing.T) {
	s, auth, now, _ := newFixture(t)
	shared := []string{"reviewer", "a", "admin", "b"}
	s.Authority = borrowedElectorate{auth, shared}
	r := opened(t, s, *now)
	if strings.Join(shared, ",") != "reviewer,a,admin,b" {
		t.Fatal("provider-owned roster mutated", shared)
	}
	*now = now.Add(2 * time.Hour)
	r = command(t, s, "a", Command{Action: "close", Intent: "close", Proposal: r.Proposal, Revision: r.Revision})
	c := Command{Action: "certify", Intent: "certify", Proposal: r.Proposal, Revision: r.Revision}
	command(t, s, "admin", c)
	auth.mu.Lock()
	auth.members["admin"] = Principal{Active: true, Enabled: true}
	auth.mu.Unlock()
	if _, err := s.Execute(context.Background(), Identity{"n", "admin"}, c); err == nil {
		t.Fatal("revoked admin replay acknowledged")
	}
}

func TestAuditDeadlineCannotEraseResultAndDuplicateDispute(t *testing.T) {
	s, _, now, _ := newFixture(t)
	r := opened(t, s, *now)
	command(t, s, "b", Command{Action: "vote", Intent: "v", Proposal: r.Proposal, Revision: r.Revision, Choice: No})
	*now = now.Add(time.Hour)
	if _, err := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "withdraw", Intent: "late-withdraw", Proposal: r.Proposal, Revision: r.Revision, Text: "Erase losing poll"}); err == nil {
		t.Fatal("deadline outcome can be withdrawn")
	}
	r = command(t, s, "a", Command{Action: "close", Intent: "close", Proposal: r.Proposal, Revision: r.Revision})
	d := Command{Action: "dispute", Intent: "concern", Proposal: r.Proposal, Revision: r.Revision, Challenged: "a", Text: "Check process"}
	command(t, s, "b", d)
	d.Intent = "duplicate-concern"
	if _, err := s.Execute(context.Background(), Identity{"n", "b"}, d); err == nil {
		t.Fatal("identical unresolved concern consumes quota")
	}
}
func TestAuditSafetyActionSurvivesIntentQuota(t *testing.T) {
	s, _, _, _ := newFixture(t)
	r := command(t, s, "a", Command{Action: "create", Intent: "create", Title: "T", Body: "B", Category: "p"})
	if _, err := s.DB.Exec(`WITH RECURSIVE seq(n) AS(SELECT 1 UNION ALL SELECT n+1 FROM seq WHERE n<10000) INSERT INTO future_gov_intents(network,member,intent,fingerprint,reply) SELECT 'n','a','seed-'||n,'f','{}' FROM seq`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Execute(context.Background(), Identity{"n", "a"}, Command{Action: "edit", Intent: "blocked", Proposal: r.Proposal, Revision: r.Revision, Title: "New", Body: "B", Category: "p"}); err == nil {
		t.Fatal("intent bound ignored")
	}
	command(t, s, "a", Command{Action: "withdraw", Intent: "safety", Proposal: r.Proposal, Revision: r.Revision, Text: "Withdraw"})
}
func TestAuditEscapedStoredPayloadKeepsDocumentedBounds(t *testing.T) {
	s, _, _, _ := newFixture(t)
	body := strings.Repeat("<", 12000)
	r := command(t, s, "a", Command{Action: "create", Intent: "escaped", Title: "T", Body: body, Category: "p"})
	detail, err := s.Read(context.Background(), Identity{"n", "a"}, r.Proposal)
	if err != nil || detail.Proposal.Body != body {
		t.Fatal("valid escaped persistent payload rejected", err)
	}
}
func TestAuditSchemaLiteralCaseIsNotNormalizedAway(t *testing.T) {
	s, _, _, _ := newFixture(t)
	if _, err := s.DB.Exec(`DROP TABLE future_gov_ballots`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range schemaStatements {
		if strings.Contains(statement, "CREATE TABLE IF NOT EXISTS future_gov_ballots(") {
			statement = strings.Replace(statement, "'yes'", "'YES'", 1)
			if _, err := s.DB.Exec(statement); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := s.Migrate(context.Background()); err == nil {
		t.Fatal("different choice constraint accepted")
	}
}
func TestAuditListRejectsForgedElectorateCount(t *testing.T) {
	s, _, now, _ := newFixture(t)
	r := opened(t, s, *now)
	var raw []byte
	if err := s.DB.QueryRow(`SELECT body FROM future_gov_proposals WHERE id=?`, r.Proposal).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var p Proposal
	json.Unmarshal(raw, &p)
	p.Eligible++
	raw, _ = json.Marshal(p)
	if _, err := s.DB.Exec(`UPDATE future_gov_proposals SET body=? WHERE id=?`, raw, r.Proposal); err != nil {
		t.Fatal(err)
	}
	if list, err := s.List(context.Background(), Identity{"n", "b"}, "", 100); err == nil || len(list) > 0 {
		t.Fatal("forged eligibility count surfaced", list, err)
	}
}
