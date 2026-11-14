package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/store"
)

// Read-only methods for operator tooling (ARC-002): the CLI and its jobs ask
// the service instead of querying the database themselves.

// NetworkRef identifies a network to operator tooling.
type NetworkRef struct{ ID, Slug string }

// OperatorNetworks lists every network in id order, or only the network whose
// id or slug equals filter when filter is not empty. It is reserved to the
// system actor (operator CLI and scheduled jobs) and returns ids and slugs
// only, so tooling can address each network through the service.
func (s *Service) OperatorNetworks(ctx context.Context, actor domain.Actor, filter string) ([]NetworkRef, error) {
	if err := requireSystem(actor); err != nil {
		return nil, err
	}
	var out []NetworkRef
	err := s.st.View(ctx, func(tx store.Tx) error {
		ids, err := tx.ListNetworkIDs()
		if err != nil {
			return err
		}
		for _, id := range ids {
			n, err := tx.GetNetwork(id)
			if err != nil {
				return err
			}
			if filter == "" || filter == n.ID || filter == n.Slug {
				out = append(out, NetworkRef{ID: n.ID, Slug: n.Slug})
			}
		}
		return nil
	})
	return out, err
}

// sqlBacked is implemented by stores built on database/sql (the SQLite store).
type sqlBacked interface{ DB() *sql.DB }

// errStorageUncheckable is returned when the store exposes no settings to check.
var errStorageUncheckable = errors.New("storage settings cannot be checked for this store")

// CheckStorage verifies the storage settings the design depends on (OPS-011):
// WAL journal mode and foreign-key enforcement. It reads settings only. A
// store whose settings cannot be read fails the check rather than passing it.
func (s *Service) CheckStorage(ctx context.Context) error {
	b, ok := s.st.(sqlBacked)
	if !ok {
		return errStorageUncheckable
	}
	return store.CheckPragmas(ctx, b.DB())
}
