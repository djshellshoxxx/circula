// Command discovery is the operator CLI for the FUX-103 discovery module.
package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/djshellshoxxx/circula/future/discovery/cli"
	"github.com/djshellshoxxx/circula/future/discovery/sqlstore"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, cli.Env{Open: open}))
}

func open(path string) (*sql.DB, error) {
	for _, name := range []string{"sqlite3", "sqlite"} {
		for _, have := range sql.Drivers() {
			if have == name {
				return sql.Open(name, sqlstore.DSN(path))
			}
		}
	}
	return nil, fmt.Errorf("no SQLite database/sql driver is linked into this build")
}
