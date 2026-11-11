package api

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/djshellshoxxx/circula/internal/service"
)

type exportEntry struct {
	net, person string
	at          time.Time
	dto         exportDTO
	file        []byte
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
	Bytes       int     `json:"bytes"`
	ArtefactURL *string `json:"artefact_url"`
	Error       *string `json:"error"`
}

var exportScopes = map[string]service.ExportScope{"mine": service.ExportMine, "ledger": service.ExportLedger,
	"members": service.ExportMembers, "audit": service.ExportAudit, "full": service.ExportFull}

// createExport runs the export synchronously (the service streams from one
// read snapshot) and keeps the artefact in memory for 24 h, so the job is
// already `complete` when the 201 returns.
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
	sink := &service.MemSink{}
	_, err := s.svc.Export(c.r.Context(), c.actor, c.netID, scope, format, sink)
	if err != nil {
		return err
	}
	var zb bytes.Buffer
	zw := zip.NewWriter(&zb)
	for _, n := range sink.Names() {
		f, _ := zw.Create(n)
		_, _ = f.Write(sink.Bytes(n))
	}
	if err := zw.Close(); err != nil {
		return err
	}
	data := zb.Bytes()
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	id := uuid.NewString()
	now := s.opt.Now()
	url := v1 + "/exports/" + id + "/artefact"
	d := exportDTO{ID: id, Scope: in.Scope, Format: string(format), Status: "complete", CreatedAt: ts(now),
		ExpiresAt: ts(now.Add(jobTTL)), SHA256: &hexsum, Bytes: len(data), ArtefactURL: &url}
	if !s.jobs.putExport(&exportEntry{net: c.netID, person: c.actor.PersonID, at: now, dto: d, file: data,
		ctype: "application/zip", name: "circula-export-" + in.Scope + ".zip"}) {
		return storeFull("full")
	}
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
	sum := sha256.Sum256(e.file)
	if hex.EncodeToString(sum[:]) != *e.dto.SHA256 { // DAT-040: only serve verified artefacts
		return errInternal
	}
	h := c.w.Header()
	h.Set("Content-Type", e.ctype)
	h.Set("Content-Disposition", `attachment; filename="`+e.name+`"`)
	h.Set("X-Content-SHA256", *e.dto.SHA256)
	c.w.WriteHeader(http.StatusOK)
	_, _ = c.w.Write(e.file)
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
