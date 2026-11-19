package api

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"os"
)

// spoolFile is an export archive on disk (FR-095). It is written once while
// the export streams and read with ReadAt for every download, so neither step
// holds the archive in memory. Where the platform allows it (POSIX), the name
// is unlinked as soon as the file exists: the open descriptor keeps the data
// readable and nothing is left on disk if the process stops. Elsewhere the
// file is removed when its export is dropped.
type spoolFile struct {
	f    *os.File
	path string // name still on disk, removed on release; "" once unlinked
	size int64
}

func (s *spoolFile) release() {
	_ = s.f.Close()
	if s.path != "" {
		_ = os.Remove(s.path)
	}
}

// errExportFull stops an export whose archive would not fit in the shared
// byte cap (BR-06).
var errExportFull = errors.New("export storage is full")
var errExportRevoked = errors.New("export access was revoked")

// artefactWriter receives the archive as the export produces it, reserving
// every chunk against the store's byte cap before writing it to the spool
// file, and hashing and counting it on the way (BR-06). There is no in-memory
// fallback: without a spool file the export is refused.
type artefactWriter struct {
	spool *spoolFile
	t     *exportTicket
	h     hash.Hash
	n     int64
	full  bool // a write was refused by the byte cap
}

// newArtefactWriter creates the spool file in dir ("" is os.TempDir()).
func newArtefactWriter(dir string, t *exportTicket) (*artefactWriter, error) {
	f, err := os.CreateTemp(dir, "circula-export-*.zip") // mode 0600
	if err != nil {
		return nil, err
	}
	a := &artefactWriter{spool: &spoolFile{f: f, path: f.Name()}, t: t, h: sha256.New()}
	if os.Remove(f.Name()) == nil {
		a.spool.path = ""
	}
	return a, nil
}

func (a *artefactWriter) Write(p []byte) (int, error) {
	if !a.t.reserve(int64(len(p))) {
		a.full = true
		return 0, errExportFull
	}
	n, err := a.spool.f.Write(p)
	a.h.Write(p[:n])
	a.n += int64(n)
	return n, err
}

// discard releases what was written; the writer is not used again.
func (a *artefactWriter) discard() { a.spool.release() }

// finish hands the complete archive to e and returns its hex SHA-256.
func (a *artefactWriter) finish(e *exportEntry) string {
	a.spool.size = a.n
	e.spool = a.spool
	return hex.EncodeToString(a.h.Sum(nil))
}

func (e *exportEntry) size() int64 {
	if e.spool == nil {
		return 0
	}
	return e.spool.size
}

// release frees the artefact once its export has left the store.
func (e *exportEntry) release() {
	if e.spool != nil {
		e.spool.release()
	}
}

// content reads the artefact from its start; each call is independent, so
// downloads may run side by side.
func (e *exportEntry) content() *io.SectionReader {
	return io.NewSectionReader(e.spool.f, 0, e.spool.size)
}
