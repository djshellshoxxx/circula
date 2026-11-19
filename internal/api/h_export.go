package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/djshellshoxxx/circula/internal/service"
)

type exportEntry struct {
	net, person string
	at          time.Time
	dto         exportDTO
	spool       *spoolFile // the archive on disk (FR-095)
	ctype       string
	name        string
	// Lifecycle fields are protected by jobStore.mu.
	readers int
	expired bool
	timer   *time.Timer
	revoked chan struct{}
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
// through the zip writer into a spool file, never whole in memory (FR-095).
// The export's slot and every byte it writes are reserved before they are
// used (BR-06); without a usable spool directory it is refused.
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
	id, err := jobID("")
	if err != nil {
		return err
	}
	// Refuse before running: the service audits every export it performs, and
	// one member must not be able to fill the shared store (SA-11). The slot
	// is held while the export runs, so parallel requests cannot all pass.
	ticket, why := s.jobs.beginExport(c.netID, c.actor.PersonID, scope != service.ExportMine)
	if why != "" {
		return storeFull(why)
	}
	defer ticket.abort() // no-op once committed
	art, err := newArtefactWriter(s.jobs.dir, ticket)
	if err != nil {
		// Fail closed: never fall back to holding the archive in memory.
		s.log.Error("export spool directory unusable; export refused", "dir", s.jobs.dir, "err", err)
		return storeFull("spool")
	}
	zs := service.NewZipSink(art)
	_, err = s.svc.Export(c.r.Context(), c.actor, c.netID, scope, format, zs)
	if err == nil {
		err = zs.Close()
	}
	if err != nil {
		art.discard()
		if art.full {
			return storeFull("full")
		}
		return err
	}
	now := s.opt.Now()
	e := &exportEntry{net: c.netID, person: c.actor.PersonID, at: now, ctype: "application/zip",
		name: "circula-export-" + in.Scope + ".zip"}
	hexsum := art.finish(e)
	url := v1 + "/exports/" + id + "/artefact"
	e.dto = exportDTO{ID: id, Scope: in.Scope, Format: string(format), Status: "complete", CreatedAt: ts(now),
		ExpiresAt: ts(now.Add(jobTTL)), SHA256: &hexsum, Bytes: e.size(), ArtefactURL: &url}
	if !ticket.commit(e) {
		e.release()
		return storeFull("erased")
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
	e := s.jobs.acquireExport(c.netID, c.actor.PersonID, c.r.PathValue("id"))
	if e == nil {
		return notFoundErr()
	}
	defer s.jobs.releaseExport(e)
	if err := s.svc.AuthorizeExportDownload(c.r.Context(), c.actor, c.netID, exportScopes[e.dto.Scope]); err != nil {
		return err // SA2-17: role is re-checked at download
	}
	// Closing the spool stops reads; a deadline also interrupts a client
	// blocked while receiving a chunk already read before erasure.
	stop, watched := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(watched)
		select {
		case <-e.revoked:
			_ = http.NewResponseController(c.w).SetWriteDeadline(time.Now())
		case <-stop:
		}
	}()
	defer func() { close(stop); <-watched }()
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
	if _, err := io.Copy(exportResponseWriter{w: c.w, revoked: e.revoked}, e.content()); err != nil {
		s.log.Error("export download interrupted", "request_id", c.r.Header.Get("X-Request-Id"), "err", err)
		// Headers have already been sent. Abort the response; appending a
		// JSON error would turn a failed archive into a successful response.
		panic(http.ErrAbortHandler)
	}
	return nil
}

type exportResponseWriter struct {
	w       io.Writer
	revoked <-chan struct{}
}

func (w exportResponseWriter) Write(p []byte) (int, error) {
	select {
	case <-w.revoked:
		return 0, errExportRevoked
	default:
	}
	n, err := w.w.Write(p)
	select {
	case <-w.revoked:
		return n, errExportRevoked
	default:
		return n, err
	}
}

// storeFull is the refusal for an export that cannot run or be kept. A person
// at their own cap gets 429 (their own exports must expire first); the rest
// are 503: the shared store is full, too many exports are running, the spool
// directory is unusable (BR-06), or a member of the network was erased while
// an export ran (BR-04).
func storeFull(why string) *apiError {
	switch why {
	case "person":
		return &apiError{status: 429, code: "rate-limited", retryAfter: "3600",
			detail: "you already hold the maximum number of unexpired exports; they expire after 24 hours"}
	case "busy":
		return &apiError{status: 503, code: "unavailable", detail: "too many exports are running; retry shortly", retryAfter: "10"}
	case "spool":
		return &apiError{status: 503, code: "unavailable", detail: "export storage is unavailable; retry later", retryAfter: "300"}
	case "erased":
		return &apiError{status: 503, code: "unavailable", detail: "a member was erased while the export ran; retry", retryAfter: "1"}
	}
	return &apiError{status: 503, code: "unavailable", detail: "export storage is full; retry later", retryAfter: "60"}
}
