package main

import (
	"context"
	"errors"
)

// purgeAndWarn is the hourly maintenance job: it purges expired rows (SA-12)
// and then sends credit-limit warnings (UX-012). Both always run; a failure in
// one does not skip the other, and both errors are reported.
func (j *jobRunner) purgeAndWarn(ctx context.Context) error {
	// GOV-012: record sign-in activity before purge deletes idle sessions.
	ierr := j.inactivity(ctx)
	perr := j.purge(ctx)
	n, werr := j.svc.RunLimitWarnings(ctx)
	if werr == nil && n > 0 {
		j.logger.Info("limit warnings sent", "count", n)
	}
	return errors.Join(ierr, perr, werr)
}

// inactivity flags members under each network's inactivity rule (GOV-012).
func (j *jobRunner) inactivity(ctx context.Context) error {
	n, err := j.svc.RunInactivityRule(ctx)
	if err == nil && n > 0 {
		j.logger.Info("members flagged inactive", "count", n)
	}
	return err
}
