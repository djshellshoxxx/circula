//go:build sqlite_cgo_test

package favorites_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/sqlitefixture"
	"github.com/djshellshoxxx/circula/future/favorites/sqlstore"
)

// Run every behavior test in this package against real SQLite as well.
func init() {
	factories = append(factories, storeFactory{"sqlite", func(t *testing.T) favorites.Store {
		db := sqlitefixture.Open(t, sqlitefixture.Path(t))
		if err := sqlstore.Migrate(ctx, db); err != nil {
			t.Fatal(err)
		}
		return sqlstore.New(db)
	}})
}
