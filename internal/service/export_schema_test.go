package service_test

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
	tu "github.com/djshellshoxxx/circula/internal/testutil"
)

// DAT-041: docs/api/export-schema.md lists exactly the columns the code writes,
// and the version it states is the code's version.
func TestExportSchemaDocMatches(t *testing.T) {
	doc, err := os.ReadFile("../../docs/api/export-schema.md")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{}
	re := regexp.MustCompile("(?m)^### (\\w+)\\n`([^`]+)`")
	for _, m := range re.FindAllStringSubmatch(string(doc), -1) {
		want[m[1]] = m[2]
	}
	if !strings.Contains(string(doc), "**Schema version: 1.**") || service.ExportSchemaVersion != 1 {
		t.Error("schema version in the document and the code differ")
	}
	w := newWorld(t, nil)
	sink := &service.MemSink{}
	man := tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportFull, service.FormatCSV, sink))
	if len(man.Files) != len(want) {
		t.Errorf("document has %d tables, export has %d files", len(want), len(man.Files))
	}
	for _, f := range man.Files {
		table := strings.TrimSuffix(f.Name, ".csv")
		head := strings.TrimSpace(strings.SplitN(string(sink.Bytes(f.Name)), "\n", 2)[0])
		if want[table] != head {
			t.Errorf("%s: document columns\n  %s\ncode columns\n  %s", table, want[table], head)
		}
	}
}

// DAT-041: the reference reader reads a package without the application.
func TestReferenceReader(t *testing.T) {
	py, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}
	w := newWorld(t, nil)
	for _, format := range []service.ExportFormat{service.FormatCSV, service.FormatJSON} {
		dir := t.TempDir()
		tu.Must(w.Svc.Export(w.Ctx, w.admin, w.net.ID, service.ExportFull, format, service.DirSink{Dir: dir}))
		out, err := exec.Command(py, "-I", "../../scripts/read-export.py", dir).CombinedOutput()
		if err != nil || !strings.Contains(string(out), "export ok") {
			t.Fatalf("%s: %v\n%s", format, err, out)
		}
		// Tampering is caught.
		if format == service.FormatCSV {
			f := dir + "/entries.csv"
			b := tu.Must(os.ReadFile(f))
			if err := os.WriteFile(f, append(b, []byte("x\n")...), 0o600); err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(py, "-I", "../../scripts/read-export.py", dir).CombinedOutput()
			if err == nil || !strings.Contains(string(out), "FAIL") {
				t.Fatalf("tampered package passed:\n%s", out)
			}
		}
	}
}
