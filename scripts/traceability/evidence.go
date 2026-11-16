package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Execution evidence is optional and separate from the committed citation inventory.
// Only terminal Go test events are recorded; a citation never becomes a test result.
type testEvent struct {
	Time                  time.Time
	Action, Package, Test string
}
type executionEvidence struct {
	Version      int         `json:"version"`
	Head         string      `json:"head"`
	Dirty        bool        `json:"dirty"`
	SourceDigest string      `json:"source_digest"`
	Command      []string    `json:"command"`
	ExitCode     int         `json:"exit_code"`
	Results      []testEvent `json:"results"`
}

func gitOutput(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...) //nolint:gosec // fixed executable; arguments are internal git operations, never a shell
	cmd.Dir = root
	return cmd.Output()
}

// Fingerprint tracked and untracked, nonignored inputs, including test fixtures.
// Generated reports/evidence are excluded explicitly to avoid self-reference.
func sourceDigest(root string, excluded ...string) (string, error) {
	names, err := gitOutput(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}
	skip := map[string]bool{"docs/TRACEABILITY.md": true}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	for _, p := range excluded {
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(root, p)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		rel, err := filepath.Rel(absRoot, abs)
		if err != nil {
			return "", err
		}
		skip[filepath.ToSlash(rel)] = true
	}
	files := uniqueStrings(strings.Split(strings.TrimSuffix(string(names), "\x00"), "\x00"))
	sort.Strings(files)
	h := sha256.New()
	for _, p := range files {
		if p == "" || skip[p] {
			continue
		}
		filename := filepath.Join(root, filepath.FromSlash(p))
		info, err := os.Lstat(filename)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} // tracked deletion is represented by absence
		if err != nil {
			return "", err
		}
		var data []byte
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(filename)
			if err != nil {
				return "", err
			}
			data = []byte(target)
		} else if info.Mode().IsRegular() {
			data, err = os.ReadFile(filename)
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("unsupported source input %s", p)
		}
		fmt.Fprintf(h, "%d:%s:%s:%d:", len(p), p, info.Mode(), len(data))
		_, _ = h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func terminalEvents(src io.Reader) ([]testEvent, error) {
	dec := json.NewDecoder(src)
	results := map[string]testEvent{}
	for {
		var ev testEvent
		if err := dec.Decode(&ev); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode go test JSON: %w", err)
		}
		if ev.Action != "pass" && ev.Action != "fail" && ev.Action != "skip" {
			continue
		}
		if ev.Time.IsZero() || ev.Package == "" {
			return nil, errors.New("test result lacks timestamp or package")
		}
		ev.Time = ev.Time.UTC()
		key := ev.Package + "\x00" + ev.Test
		if _, exists := results[key]; exists {
			return nil, fmt.Errorf("duplicate terminal result for %s %s", ev.Package, ev.Test)
		}
		results[key] = ev
	}
	if len(results) == 0 {
		return nil, errors.New("go test produced no terminal results")
	}
	keys := make([]string, 0, len(results))
	for k := range results {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]testEvent, 0, len(keys))
	for _, k := range keys {
		out = append(out, results[k])
	}
	return out, nil
}

func recordEvidence(root, output string, args []string) error {
	if len(args) == 0 {
		return errors.New("-record-results requires go test arguments after -- (for example -- -race ./...)")
	}
	for _, arg := range args {
		if strings.HasPrefix(arg, "-json") || strings.HasPrefix(arg, "-count") || strings.HasPrefix(arg, "-args") {
			return fmt.Errorf("recorder owns JSON/count flags and does not accept -args: %s", arg)
		}
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	status, err := gitOutput(root, "status", "--porcelain")
	if err != nil {
		return err
	}
	digest, err := sourceDigest(root, output)
	if err != nil {
		return err
	}
	command := append([]string{"test", "-json", "-count=1"}, args...)
	cmd := exec.Command("go", command...) //nolint:gosec // developer-invoked go test arguments; no shell, credentials or remote input
	cmd.Dir, cmd.Stderr = root, os.Stderr
	stdout, runErr := cmd.Output()
	code := 0
	if runErr != nil {
		var exit *exec.ExitError
		if !errors.As(runErr, &exit) {
			return runErr
		}
		code = exit.ExitCode()
	}
	after, err := sourceDigest(root, output)
	if err != nil {
		return err
	}
	afterHead, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if digest != after || !bytes.Equal(head, afterHead) {
		return errors.New("source tree or HEAD changed during tests; no evidence written")
	}
	events, err := terminalEvents(bytes.NewReader(stdout))
	if err != nil {
		return err
	}
	if runErr != nil {
		writeFailureOutput(stdout, events, os.Stderr)
	}
	e := executionEvidence{Version: 1, Head: strings.TrimSpace(string(head)), Dirty: len(status) > 0, SourceDigest: digest, Command: append([]string{"go"}, command...), ExitCode: code, Results: events}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	p := output
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	if err := os.WriteFile(p, append(b, '\n'), 0o644); err != nil {
		return err
	}
	if runErr != nil {
		return fmt.Errorf("go test exited %d; failure evidence saved to %s", code, output)
	}
	fmt.Fprintf(os.Stderr, "traceability: recorded %d terminal results in %s\n", len(events), output)
	return nil
}

func readEvidence(root, filename, report string) (*executionEvidence, error) {
	if filename == "" {
		return nil, nil
	}
	p := filename
	if !filepath.IsAbs(p) {
		p = filepath.Join(root, p)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var e executionEvidence
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&e); err != nil {
		return nil, err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return nil, errors.New("trailing execution evidence data")
	}
	if e.Version != 1 || len(e.Head) != 40 || len(e.SourceDigest) != 64 || len(e.Results) == 0 || len(e.Command) < 4 || strings.Join(e.Command[:4], " ") != "go test -json -count=1" {
		return nil, errors.New("invalid execution evidence provenance")
	}
	if _, err := hex.DecodeString(e.Head + e.SourceDigest); err != nil {
		return nil, errors.New("invalid execution evidence digest")
	}
	seen := map[string]bool{}
	for _, ev := range e.Results {
		key := ev.Package + "\x00" + ev.Test
		if ev.Time.IsZero() || ev.Package == "" || (ev.Action != "pass" && ev.Action != "fail" && ev.Action != "skip") || seen[key] {
			return nil, errors.New("invalid or duplicate execution result")
		}
		seen[key] = true
	}
	head, err := gitOutput(root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	digest, err := sourceDigest(root, filename, report)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(head)) != e.Head || digest != e.SourceDigest {
		return nil, errors.New("execution evidence does not match the current HEAD and source tree; rerun tests")
	}
	return &e, nil
}

func testLayers(cs []Cite) string {
	layers := map[string]bool{}
	for _, c := range cs {
		parts := strings.Split(c.Path, "/")
		layer := "other"
		if len(parts) > 1 {
			switch parts[0] {
			case "internal":
				layer = parts[1]
			case "cmd":
				layer = "CLI"
			case "e2e":
				layer = "browser"
			case "scripts":
				layer = "tooling"
			}
		}
		layers[layer] = true
	}
	if len(layers) == 0 {
		return "-"
	}
	return strings.Join(sortedKeys(layers), ", ")
}

func executionCell(cs []Cite, e *executionEvidence, module string, max int) string {
	if e == nil {
		return "not recorded"
	}
	events := map[string]testEvent{}
	for _, ev := range e.Results {
		events[ev.Package+"\x00"+ev.Test] = ev
	}
	var cells []string
	for i, c := range cs {
		if i == max {
			cells = append(cells, fmt.Sprintf("+%d more citations", len(cs)-max))
			break
		}
		pkg := module
		if dir := path.Dir(c.Path); dir != "." {
			pkg += "/" + dir
		}
		key := pkg + "\x00" + c.Symbol
		ev, found := events[key]
		if !found || !strings.HasSuffix(c.Path, "_test.go") || !strings.HasPrefix(c.Symbol, "Test") {
			cells = append(cells, "not recorded")
			continue
		}
		cells = append(cells, "`"+c.Symbol+"`: "+ev.Action+" "+ev.Time.UTC().Format(time.RFC3339))
	}
	if len(cells) == 0 {
		return "not recorded"
	}
	return strings.Join(cells, "<br>")
}

// Keep CI failures diagnosable without dumping successful test JSON into the log.
func writeFailureOutput(raw []byte, results []testEvent, dst io.Writer) {
	failed := map[string]bool{}
	for _, ev := range results {
		if ev.Action == "fail" {
			failed[ev.Package+"\x00"+ev.Test] = true
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	for {
		var ev struct{ Action, Package, Test, Output string }
		if err := dec.Decode(&ev); err != nil {
			return
		}
		if ev.Action == "output" && failed[ev.Package+"\x00"+ev.Test] {
			_, _ = io.WriteString(dst, ev.Output)
		}
	}
}
