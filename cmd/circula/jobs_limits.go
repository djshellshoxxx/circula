package main

import (
	"context"
	"errors"
)

// purgeAndWarn is the hourly maintenance job: it purges expired rows (SA-12)
// and then sends credit-limit warnings (UX-012). Both always run; a failure in
// one does not skip the other, and both errors are reported.
func (j *jobRunner) purgeAndWarn(ctx context.Context) error {
	perr := j.purge(ctx)
	n, werr := j.svc.RunLimitWarnings(ctx)
	if werr == nil && n > 0 {
		j.logger.Info("limit warnings sent", "count", n)
	}
	return errors.Join(perr, werr)
}
