package repository_test

import (
	"bufio"
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ARC-004: interface declarations and domain services must compile without
// any database engine or SQL handle dependency, including indirect imports
// through helpers. database/sql/driver is an interface-only standard-library
// package used by google/uuid; it neither loads an engine nor opens databases.
// `go list` excludes test fixtures: service tests legitimately use real SQLite.
// Checking the contract suite too prevents a driver-neutral-looking harness
// from bringing the implementation back through an interface import.
func TestRepositoryServiceAndContractSuiteHaveNoDriverDependency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "list", "-deps", "-f", "{{.ImportPath}}", "./internal/repository", "./internal/service", "./internal/storetest")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("list production dependencies: %v\n%s", err, output)
	}
	denied := []string{"github.com/djshellshoxxx/circula/internal/store", "modernc.org/sqlite", "github.com/mattn/go-sqlite3"}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		dep := strings.TrimSpace(scanner.Text())
		if dep == "database/sql" {
			t.Errorf("SQL handle dependency %s in repository/service/contract graph (ARC-004)", dep)
		}
		for _, prefix := range denied {
			if dep == prefix || strings.HasPrefix(dep, prefix+"/") {
				t.Errorf("driver dependency %s in repository/service/contract graph (ARC-004)", dep)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}
