package store

import "github.com/djshellshoxxx/circula/internal/repository"

// Compatibility aliases preserve the existing storage API and exact type
// identity. Driver-neutral contracts live in internal/repository (ARC-004).
type (
	PrefsRepo          = repository.PrefsRepo
	AnnouncementFilter = repository.AnnouncementFilter
	AnnouncementRepo   = repository.AnnouncementRepo
	OverviewParams     = repository.OverviewParams
	OverviewCounts     = repository.OverviewCounts
	OverviewRepo       = repository.OverviewRepo
)

var (
	_ PrefsRepo        = (*txImpl)(nil)
	_ AnnouncementRepo = (*txImpl)(nil)
	_ OverviewRepo     = (*txImpl)(nil)
)
