package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/djshellshoxxx/circula/internal/service"
)

func init() {
	extraCommands["verify-export"] = cmdVerifyExport
	extraUsage = append(extraUsage, "  verify-export <file|dir>    check an export package against its manifest (sizes, checksums, row counts)")
}

// maxExportFile bounds one file read by the verifier.
const maxExportFile = 1 << 32

// cmdVerifyExport re-hashes every file of an export package (a .zip or an
// unpacked directory) and compares size, SHA-256 and row count with
// manifest.json. Any mismatch fails (DAT-040). Read-only; needs no database.
func cmdVerifyExport(_ context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify-export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: circula verify-export <file.zip|directory>")
		return exitUsage
	}
	problems, man, err := verifyExportPackage(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "verify-export: %v\n", err)
		return exitFailure
	}
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintf(stdout, "FAIL %s\n", p)
		}
		fmt.Fprintf(stdout, "export verification failed: %d problem(s)\n", len(problems))
		return exitFailure
	}
	fmt.Fprintf(stdout, "export ok: %d file(s), scope %s, ledger_seq %d\n", len(man.Files), man.Scope, man.LedgerSeq)
	return exitOK
}

// exportReader gives name-addressed access to a package.
type exportReader struct {
	names []string
	read  func(name string) ([]byte, error)
}

func openExport(path string) (*exportReader, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		ents, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		r := &exportReader{read: func(n string) ([]byte, error) { return readCapped(filepath.Join(path, n)) }}
		for _, e := range ents {
			if !e.IsDir() {
				r.names = append(r.names, e.Name())
			}
		}
		return r, nil
	}
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("not a zip file or directory: %w", err)
	}
	files := map[string]*zip.File{}
	r := &exportReader{}
	for _, f := range zr.File {
		files[f.Name] = f
		r.names = append(r.names, f.Name)
	}
	r.read = func(n string) ([]byte, error) {
		f, ok := files[n]
		if !ok {
			return nil, os.ErrNotExist
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(io.LimitReader(rc, maxExportFile))
	}
	return r, nil
}

func readCapped(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(io.LimitReader(f, maxExportFile))
}

// verifyExportPackage returns one line per problem found. An error means the
// package could not be examined at all (missing, unreadable, no manifest).
func verifyExportPackage(path string) ([]string, *service.Manifest, error) {
	r, err := openExport(path)
	if err != nil {
		return nil, nil, err
	}
	mb, err := r.read("manifest.json")
	if err != nil {
		return nil, nil, errors.New("manifest.json is missing from the package")
	}
	var man service.Manifest
	if err := json.Unmarshal(mb, &man); err != nil {
		return nil, nil, fmt.Errorf("manifest.json is not valid: %w", err)
	}
	var problems []string
	listed := map[string]bool{"manifest.json": true}
	for _, f := range man.Files {
		listed[f.Name] = true
		if filepath.Base(f.Name) != f.Name {
			problems = append(problems, fmt.Sprintf("%s: unsafe file name in manifest", f.Name))
			continue
		}
		b, err := r.read(f.Name)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: listed in the manifest but not readable (%v)", f.Name, err))
			continue
		}
		if int64(len(b)) != f.Bytes {
			problems = append(problems, fmt.Sprintf("%s: size %d, manifest says %d", f.Name, len(b), f.Bytes))
		}
		if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != f.SHA256 {
			problems = append(problems, fmt.Sprintf("%s: checksum mismatch", f.Name))
		}
		rows, err := countExportRows(f.Name, b)
		switch {
		case err != nil:
			problems = append(problems, fmt.Sprintf("%s: cannot count rows (%v)", f.Name, err))
		case rows != f.Rows:
			problems = append(problems, fmt.Sprintf("%s: %d rows, manifest says %d", f.Name, rows, f.Rows))
		}
	}
	extra := []string{}
	for _, n := range r.names {
		if !listed[n] {
			extra = append(extra, n)
		}
	}
	sort.Strings(extra)
	for _, n := range extra {
		problems = append(problems, fmt.Sprintf("%s: present but not listed in the manifest", n))
	}
	return problems, &man, nil
}

// countExportRows counts data rows: CSV records minus the header, or the
// elements of every table array in a JSON document.
func countExportRows(name string, b []byte) (int64, error) {
	if filepath.Ext(name) == ".csv" {
		recs, err := csv.NewReader(bytes.NewReader(b)).ReadAll()
		if err != nil {
			return 0, err
		}
		if len(recs) == 0 {
			return 0, errors.New("empty csv")
		}
		return int64(len(recs) - 1), nil
	}
	var doc struct {
		Tables map[string][]json.RawMessage `json:"tables"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return 0, err
	}
	var n int64
	for _, rows := range doc.Tables {
		n += int64(len(rows))
	}
	return n, nil
}
