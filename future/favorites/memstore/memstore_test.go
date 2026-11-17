package memstore_test

import (
	"testing"

	"github.com/djshellshoxxx/circula/future/favorites"
	"github.com/djshellshoxxx/circula/future/favorites/internal/storetest"
	"github.com/djshellshoxxx/circula/future/favorites/memstore"
)

func TestConformance(t *testing.T) {
	storetest.Run(t, func(*testing.T) favorites.Store { return memstore.New() })
}
