package main

import (
	"context"
	"errors"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// closures is the hourly governance job: it completes closures whose date has
// passed and records the final export checksum (GOV-025), applies the erasure
// rules after the read-only period (GOV-025 f, SEC-018) and reminds reviewers
// of disputes past their response deadline or decision target (GOV-014). The
// steps are independent; every error is reported.
func (j *jobRunner) closures(ctx context.Context) error {
	actor := domain.SystemActor("job-closure")
	done, ferr := j.svc.FinaliseClosures(ctx, actor)
	if ferr == nil && done > 0 {
		j.logger.Info("network closures completed", "count", done)
	}
	erased, eerr := j.svc.EraseClosedNetworks(ctx, actor)
	if eerr == nil && erased > 0 {
		j.logger.Info("closed networks erased", "count", erased)
	}
	sent, rerr := j.svc.RemindOverdueDisputes(ctx, actor)
	if rerr == nil && sent > 0 {
		j.logger.Info("overdue dispute reminders sent", "count", sent)
	}
	return errors.Join(ferr, eerr, rerr)
}
