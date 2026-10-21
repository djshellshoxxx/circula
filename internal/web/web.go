// Package web serves the server-rendered HTML UI. Placeholder; see docs/specs/08-ux-accessibility.md.
package web

import (
	"net/http"

	"github.com/djshellshoxxx/circula/internal/service"
)

// NewHandler returns the UI handler. Mount it at /.
func NewHandler(svc *service.Service) http.Handler { return http.NotFoundHandler() }
