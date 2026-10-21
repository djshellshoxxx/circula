// Package api serves the versioned JSON API (/api/v1). Placeholder; see docs/specs/07-api-contract.md.
package api

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
)

// NewHandler returns the API handler. Mount it at /api/.
func NewHandler(svc *service.Service) http.Handler { return http.NotFoundHandler() }
