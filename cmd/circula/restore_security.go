package main

import (
	"context"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/service"
)

// invalidateRestoreCopy changes only the staging copy. Closing it flushes the
// write transaction before verifyCopy checks the resulting ledger/audit state.
func invalidateRestoreCopy(ctx context.Context, path string, cfg *Config) (service.RestoredCredentialCounts, error) {
	st, err := openStore(ctx, path, cfg)
	if err != nil {
		return service.RestoredCredentialCounts{}, err
	}
	svc, err := newService(st, cfg)
	var counts service.RestoredCredentialCounts
	if err == nil {
		counts, err = svc.InvalidateRestoredCredentials(ctx, domain.SystemActor("restore"))
	}
	return counts, errors.Join(err, st.Close())
}
