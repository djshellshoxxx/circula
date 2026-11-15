package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// alertRulesPath is the Prometheus rule file shipped for operators (OPS-015).
const alertRulesPath = "../../docs/ops/alerts/circula.rules.yml"

// externalSeries are the only series a rule may use that this binary does not
// export, with where they come from (documented in docs/ops/monitoring.md).
var externalSeries = map[string]string{
	"up":                          "Prometheus scrape status of the circula job",
	"node_filesystem_avail_bytes": "node_exporter",
	"node_filesystem_size_bytes":  "node_exporter",
}

// operatorThresholds are recording rules the operator defines with the agreed
// value; until then the alert that compares against them stays silent.
var operatorThresholds = map[string]bool{
	"circula:auth_throttling_alert_threshold:rate10m": true,
}

type alertRule struct {
	name, group, expr, forDur string
	labels, annotations       map[string]string
}

// parseRules reads the rule file. It accepts the layout the shipped file uses
// (groups, rules, expr as a block or inline scalar, labels, annotations), not
// arbitrary YAML; promtool, when installed, checks the full syntax.
func parseRules(t *testing.T, path string) []alertRule {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("alert rules missing (OPS-015): %v", err)
	}
	var rules []alertRule
	var cur *alertRule
	group, section, block := "", "", false
	unquote := func(v string) string {
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
			return v[1 : len(v)-1]
		}
		return v
	}
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if block {
			if trim == "" || indent > 8 {
				cur.expr += trim + "\n"
				continue
			}
			block = false
		}
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		key, val, _ := strings.Cut(strings.TrimPrefix(trim, "- "), ":")
		val = strings.TrimSpace(val)
		switch {
		case indent == 2 && strings.HasPrefix(trim, "- name:"):
			group = unquote(val)
		case indent == 6 && strings.HasPrefix(trim, "- "):
			rules = append(rules, alertRule{group: group, labels: map[string]string{}, annotations: map[string]string{}})
			cur = &rules[len(rules)-1]
			section = ""
			if key == "alert" {
				cur.name = unquote(val)
			} else {
				cur.name = "record " + unquote(val)
			}
		case cur != nil && indent == 8:
			section = ""
			switch key {
			case "expr":
				if val == "|" || val == ">" || val == "|-" || val == ">-" {
					block = true
				} else {
					cur.expr = val
				}
			case "for":
				cur.forDur = val
			case "labels", "annotations":
				section = key
			}
		case cur != nil && indent == 10 && section == "labels":
			cur.labels[key] = unquote(val)
		case cur != nil && indent == 10 && section == "annotations":
			cur.annotations[key] = unquote(val)
		}
	}
	return rules
}

// exportedSeries lists every series name the /metrics exporter writes once each
// labelled family has at least one sample.
func exportedSeries(t *testing.T) map[string]bool {
	t.Helper()
	m := newMetrics(time.Now())
	m.observe("/healthz", 503, time.Millisecond)
	m.jobRan("backup", "failed")
	m.migrationRan("failed")
	m.ledgerResult(false, 1, time.Now())
	var buf bytes.Buffer
	m.write(&buf, dynamicMetrics{version: "test", now: time.Now()})
	names := map[string]bool{}
	for _, line := range strings.Split(buf.String(), "\n") {
		if f := strings.Fields(line); len(f) == 4 && f[0] == "#" && f[1] == "TYPE" {
			names[f[2]] = true
			if f[3] == "summary" {
				names[f[2]+"_sum"], names[f[2]+"_count"] = true, true
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		names[strings.FieldsFunc(line, func(r rune) bool { return r == '{' || r == ' ' })[0]] = true
	}
	return names
}

var (
	promString   = regexp.MustCompile(`"(?:[^"\\]|\\.)*"`)
	promMatchers = regexp.MustCompile(`\{[^}]*\}`)
	promRange    = regexp.MustCompile(`\[[^\]]*\]`)
	promLabelArg = regexp.MustCompile(`\b(?:by|without|on|ignoring|group_left|group_right)\s*\([^)]*\)`)
	promIdent    = regexp.MustCompile(`[A-Za-z_:][A-Za-z0-9_:]*`)
	promKeywords = map[string]bool{"and": true, "or": true, "unless": true, "bool": true, "offset": true, "inf": true, "nan": true}
)

// seriesIn returns the metric and recording-rule names a PromQL expression
// reads: identifiers that are not keywords, not function calls and not part of
// a number or duration.
func seriesIn(expr string) []string {
	e := promString.ReplaceAllString(expr, `""`)
	e = promMatchers.ReplaceAllString(e, " ")
	e = promRange.ReplaceAllString(e, " ")
	e = promLabelArg.ReplaceAllString(e, " ")
	var out []string
	for _, loc := range promIdent.FindAllStringIndex(e, -1) {
		if loc[0] > 0 && strings.ContainsRune("0123456789.", rune(e[loc[0]-1])) { // 5m, 1e3, 0.8
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(e[loc[1]:], " \t\n"), "(") { // a function call
			continue
		}
		if id := e[loc[0]:loc[1]]; !promKeywords[strings.ToLower(id)] {
			out = append(out, id)
		}
	}
	return out
}

func TestSeriesInParsesPromQL(t *testing.T) {
	got := seriesIn(`sum by (job) (rate(circula_x_total{route="/a b",class=~"5xx"}[5m])) > on(job) group_left() ` +
		`scalar(circula:t:rate10m) unless up == 0 and time() - circula_y > 26 * 3600 or absent(circula_z) offset 5m`)
	want := []string{"circula_x_total", "circula:t:rate10m", "up", "circula_y", "circula_z"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("seriesIn = %v, want %v", got, want)
	}
}

// OPS-015: the shipped rule file references only metrics this binary exports
// (or the documented external series), covers every alert the specification
// requires, and gives each alert a severity, an incident level from the
// incident procedure, an owner role and a first response that refers to
// SEC-030.
func TestAlertRulesUseExportedMetrics(t *testing.T) {
	rules := parseRules(t, alertRulesPath)
	exported := exportedSeries(t)
	levels := incidentLevels(t)
	alerts := map[string]alertRule{}
	recorded := map[string]bool{}
	for _, r := range rules {
		if strings.HasPrefix(r.name, "record ") {
			recorded[strings.TrimPrefix(r.name, "record ")] = true
		}
	}
	for _, r := range rules {
		if strings.TrimSpace(r.expr) == "" {
			t.Errorf("%s: no expr", r.name)
		}
		used := seriesIn(r.expr)
		if len(used) == 0 {
			t.Errorf("%s: expr reads no series: %q", r.name, r.expr)
		}
		for _, s := range used {
			switch {
			case exported[s], recorded[s], operatorThresholds[s]:
			case externalSeries[s] != "":
				if r.group == "circula" && s != "up" { // host exporters belong in circula-host
					t.Errorf("%s: %s comes from %s; keep it out of the circula group", r.name, s, externalSeries[s])
				}
			default:
				t.Errorf("%s: series %s is not exported by circula (cmd/circula/metrics.go) nor a documented external series", r.name, s)
			}
		}
		if strings.HasPrefix(r.name, "record ") {
			continue
		}
		alerts[r.name] = r
		if sev := r.labels["severity"]; sev != "critical" && sev != "warning" {
			t.Errorf("%s: severity label %q, want critical or warning", r.name, sev)
		}
		if lvl := r.annotations["incident_level"]; !levels[lvl] {
			t.Errorf("%s: incident_level %q is not a level of docs/ops/incident-response.md", r.name, lvl)
		}
		if r.labels["severity"] == "critical" && r.annotations["incident_level"] != "critical" {
			t.Errorf("%s: a critical alert must be a critical incident", r.name)
		}
		for _, k := range []string{"summary", "owner_role", "first_response", "runbook_url"} {
			if r.annotations[k] == "" {
				t.Errorf("%s: missing annotation %s", r.name, k)
			}
		}
		if !strings.Contains(r.annotations["first_response"], "SEC-030") {
			t.Errorf("%s: first_response must refer to the incident procedure (SEC-030)", r.name)
		}
	}
	// The conditions OPS-015 requires, with their severity.
	for name, sev := range map[string]string{
		"CirculaLedgerVerificationFailed": "critical", // ledger verification failure
		"CirculaHealthCheckFailing":       "critical", // health endpoint unhealthy for a set period
		"CirculaDown":                     "critical",
		"CirculaBackupStale":              "warning", // no successful backup within a set period
		"CirculaVolumeUsageHigh":          "warning", // database or backup volume above a set percentage
		"CirculaAuthThrottling":           "warning", // sustained authentication throttling
		"CirculaMigrationFailed":          "critical",
	} {
		r, ok := alerts[name]
		if !ok {
			t.Errorf("required alert %s missing", name)
			continue
		}
		if r.labels["severity"] != sev {
			t.Errorf("%s: severity %q, want %q", name, r.labels["severity"], sev)
		}
	}
	// A failed verification sets a value the critical rule matches (OPS-014).
	if !strings.Contains(alerts["CirculaLedgerVerificationFailed"].expr, "circula_ledger_verification_ok == 0") {
		t.Errorf("ledger rule does not match a failed verification: %q", alerts["CirculaLedgerVerificationFailed"].expr)
	}
	// The incident procedure lists every alert with its level.
	doc := readDoc(t, incidentDocPath)
	for name, r := range alerts {
		if !strings.Contains(doc, "| `"+name+"` | `"+r.annotations["incident_level"]+"` |") {
			t.Errorf("incident-response.md does not list %s at level %s", name, r.annotations["incident_level"])
		}
	}
	if path, err := exec.LookPath("promtool"); err == nil {
		if out, err := exec.Command(path, "check", "rules", alertRulesPath).CombinedOutput(); err != nil {
			t.Errorf("promtool check rules: %v\n%s", err, out)
		}
	}
}

const incidentDocPath = "../../docs/ops/incident-response.md"

func readDoc(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// incidentLevels reads the level names from the severity table of the
// incident procedure (OPS-030).
func incidentLevels(t *testing.T) map[string]bool {
	t.Helper()
	doc := readDoc(t, incidentDocPath)
	_, after, ok := strings.Cut(doc, "\n## Severity levels\n")
	if !ok {
		t.Fatal("incident-response.md has no Severity levels section")
	}
	section, _, _ := strings.Cut(after, "\n## ")
	levels := map[string]bool{}
	row := regexp.MustCompile("(?m)^\\| `([a-z]+)` \\|")
	for _, m := range row.FindAllStringSubmatch(section, -1) {
		levels[m[1]] = true
	}
	return levels
}

// SEC-030 and OPS-030 documentation check: the incident procedure contains
// each element SEC-030 lists, defines the levels OPS-030 names with a
// recommended target and channel each, marks the targets as not approved
// (DEC-022), and the runbook and monitoring guide refer to it with the same
// levels.
func TestIncidentProcedureDocumented(t *testing.T) {
	doc := readDoc(t, incidentDocPath)
	for _, el := range []string{"## Severity levels", "## Roles", "## Detection sources", "### 2. Contain", "frozen",
		"### 3. Preserve evidence", "Affected network administrators", "Members through their administrators",
		"### 5. Recover and verify", "post-incident review", "## Drills", "DEC-014", "DEC-022"} {
		if !strings.Contains(doc, el) {
			t.Errorf("incident-response.md lacks %q", el)
		}
	}
	levels := incidentLevels(t)
	var names []string
	for l := range levels {
		names = append(names, l)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != "critical,high,normal" {
		t.Fatalf("severity levels = %v, want critical, high, normal", names)
	}
	_, sev, _ := strings.Cut(doc, "\n## Severity levels\n")
	sev, _, _ = strings.Cut(sev, "\n## ")
	for _, l := range names {
		full := false // a row with meaning, examples, two targets and a channel, all filled
		for _, line := range strings.Split(sev, "\n") {
			cells := strings.Split(strings.Trim(line, "|"), "|")
			if !strings.HasPrefix(line, "| `"+l+"` |") || len(cells) != 6 {
				continue
			}
			full = true
			for _, c := range cells {
				if strings.TrimSpace(c) == "" {
					full = false
				}
			}
		}
		if !full {
			t.Errorf("level %s lacks a row with meaning, examples, recommended targets and channel", l)
		}
	}
	if !strings.Contains(sev, "Recommended first response (DEC-022 open)") || !strings.Contains(doc, "not service-level commitments") {
		t.Error("targets are not marked as recommendations pending DEC-022")
	}
	for _, p := range []string{"../../docs/ops/runbook.md", "../../docs/ops/monitoring.md"} {
		other := readDoc(t, p)
		if !strings.Contains(other, "incident-response.md") {
			t.Errorf("%s does not link the incident procedure", p)
		}
		for _, l := range names {
			if !strings.Contains(other, "`"+l+"`") {
				t.Errorf("%s does not use the incident level %s", p, l)
			}
		}
	}
	if !strings.Contains(readDoc(t, "../../docs/ops/monitoring.md"), "alerts/circula.rules.yml") {
		t.Error("monitoring.md does not point to the shipped alert rules")
	}
}
