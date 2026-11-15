package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/djshellshoxxx/circula/internal/service"
)

type exportEntry struct {
	net, person string
	at          time.Time
	dto         exportDTO
	file        []byte     // the archive when held in memory (no temporary file could be made)
	spool       *spoolFile // the archive on disk (FR-095); nil when in memory
	charged     int64      // bytes still reserved in the job store (BR-06)
	ctype       string
	name        string
}

type exportDTO struct {
	ID          string  `json:"id"`
	Scope       string  `json:"scope"`
	Format      string  `json:"format"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	ExpiresAt   string  `json:"expires_at"`
	SHA256      *string `json:"sha256"`
	Bytes       int64   `json:"bytes"`
	ArtefactURL *string `json:"artefact_url"`
	Error       *string `json:"error"`
}

var exportScopes = map[string]service.ExportScope{"mine": service.ExportMine, "ledger": service.ExportLedger,
	"members": service.ExportMembers, "audit": service.ExportAudit, "full": service.ExportFull}

// createExport runs the export synchronously (the service streams from one
// read snapshot) and keeps the artefact for 24 h, so the job is already
// `complete` when the 201 returns. The archive streams from the service
// through the zip writer into a temporary file, never whole in memory
// (FR-095); see artefactWriter for the fallback.
func (s *server) createExport(c *reqCtx) error { // API-065
	var in struct {
		Scope  string `json:"scope"`
		Format string `json:"format"`
	}
	if err := c.decode(&in); err != nil {
		return err
	}
	scope, found := exportScopes[in.Scope]
	if !found {
		return validation("scope", "invalid-enum", "must be mine, ledger, members, audit or full")
	}
	format := service.ExportFormat(in.Format)
	if in.Format == "" {
		format = service.FormatJSON
	}
	if format != service.FormatJSON && format != service.FormatCSV {
		return validation("format", "invalid-enum", "must be csv or json")
	}
	// Refuse before running: the service audits every export it performs, and
	// one member must not be able to fill the shared store (SA-11).
	if why := s.jobs.exportSlot(c.actor.PersonID); why != "" {
		return storeFull(why)
	}
	art := s.jobs.newArtefactWriter()
	if art.spool == nil {
		s.log.Warn("export spool unavailable; holding the artefact in memory within the export byte cap",
			"dir", s.jobs.dir)
	}
	zs := service.NewZipSink(art)
	_, err := s.svc.Export(c.r.Context(), c.actor, c.netID, scope, format, zs)
	if err == nil {
		err = zs.Close()
	}
	if err != nil {
		art.discard()
		if errors.Is(err, errExportBudget) {
			return storeFull("full")
		}
		return err
	}
	id := uuid.NewString()
	now := s.opt.Now()
	e := &exportEntry{net: c.netID, person: c.actor.PersonID, at: now, ctype: "application/zip",
		name: "circula-export-" + in.Scope + ".zip"}
	hexsum := art.finish(e)
	url := v1 + "/exports/" + id + "/artefact"
	e.dto = exportDTO{ID: id, Scope: in.Scope, Format: string(format), Status: "complete", CreatedAt: ts(now),
		ExpiresAt: ts(now.Add(jobTTL)), SHA256: &hexsum, Bytes: e.size(), ArtefactURL: &url}
	if !s.jobs.putExport(e) {
		e.release()
		return storeFull("full")
	}
	d := e.dto
	c.w.Header().Set("Location", v1+"/exports/"+id)
	return ok(c, 201, d)
}

func (s *server) ownExport(c *reqCtx) (*exportEntry, error) {
	e := s.jobs.getExport(c.netID, c.actor.PersonID, c.r.PathValue("id"))
	if e == nil {
		return nil, notFoundErr()
	}
	return e, nil
}

func (s *server) getExport(c *reqCtx) error { // API-065
	e, err := s.ownExport(c)
	if err != nil {
		return err
	}
	return ok(c, 200, e.dto)
}

func (s *server) getArtefact(c *reqCtx) error { // API-065
	e, err := s.ownExport(c)
	if err != nil {
		return err
	}
	if err := s.svc.AuthorizeExportDownload(c.r.Context(), c.actor, c.netID, exportScopes[e.dto.Scope]); err != nil {
		return err // SA2-17: role is re-checked at download
	}
	// DAT-040: only serve verified artefacts. Both the check and the response
	// stream from the artefact, so a download never holds it in memory (FR-095).
	sum := sha256.New()
	if _, err := io.Copy(sum, e.content()); err != nil || hex.EncodeToString(sum.Sum(nil)) != *e.dto.SHA256 {
		return errInternal
	}
	h := c.w.Header()
	h.Set("Content-Type", e.ctype)
	h.Set("Content-Disposition", `attachment; filename="`+e.name+`"`)
	h.Set("Content-Length", strconv.FormatInt(e.size(), 10))
	h.Set("X-Content-SHA256", *e.dto.SHA256)
	c.w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(c.w, e.content())
	return nil
}

// storeFull is the refusal for a full artefact store. A person at their own
// cap gets 429 (their own exports must expire or be downloaded first); a full
// shared store is a 503.
func storeFull(why string) *apiError {
	if why == "person" {
		return &apiError{status: 429, code: "rate-limited", retryAfter: "3600",
			detail: "you already hold the maximum number of unexpired exports; they expire after 24 hours"}
	}
	return &apiError{status: 503, code: "unavailable", detail: "export storage is full; retry later", retryAfter: "60"}
}
