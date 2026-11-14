package api

import (
	"bytes"
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

// errExportBudget stops an export whose bytes would exceed the job store's
// cap while it is still being written (BR-06).
var errExportBudget = errors.New("export storage budget exceeded")

// artefactWriter receives the archive as the export produces it, hashing and
// counting the bytes on the way. It spools to a new temporary file in dir
// ("" is os.TempDir()); when none can be created it collects the archive in
// memory, the DEC-047 interim behaviour. With a budget every byte is
// reserved before it is written, so memory and disk stay within the cap.
type artefactWriter struct {
	spool  *spoolFile
	mem    bytes.Buffer
	h      hash.Hash
	n      int64
	budget *jobStore // nil: no accounting (tests that build entries by hand)
}

// newArtefactWriter is newArtefactWriter(j.dir) charged against j's cap.
func (j *jobStore) newArtefactWriter() *artefactWriter {
	a := newArtefactWriter(j.dir)
	a.budget = j
	return a
}

func newArtefactWriter(dir string) *artefactWriter {
	a := &artefactWriter{h: sha256.New()}
	f, err := os.CreateTemp(dir, "circula-export-*.zip") // mode 0600
	if err != nil {
		return a
	}
	a.spool = &spoolFile{f: f, path: f.Name()}
	if os.Remove(f.Name()) == nil {
		a.spool.path = ""
	}
	return a
}

func (a *artefactWriter) Write(p []byte) (int, error) {
	if a.budget != nil && !a.budget.charge(int64(len(p))) {
		return 0, errExportBudget
	}
	var n int
	var err error
	if a.spool != nil {
		n, err = a.spool.f.Write(p)
	} else {
		n, err = a.mem.Write(p)
	}
	a.h.Write(p[:n])
	a.n += int64(n)
	if a.budget != nil && n < len(p) {
		a.budget.uncharge(int64(len(p) - n))
	}
	return n, err
}

// discard releases what was written; the writer is not used again.
func (a *artefactWriter) discard() {
	if a.budget != nil {
		a.budget.uncharge(a.n)
		a.budget, a.n = nil, 0
	}
	if a.spool != nil {
		a.spool.release()
	}
	a.mem = bytes.Buffer{}
}

// finish hands the complete archive to e and returns its hex SHA-256.
func (a *artefactWriter) finish(e *exportEntry) string {
	if a.budget != nil {
		e.charged = a.n // putExport ends the reservation
	}
	if a.spool != nil {
		a.spool.size = a.n
		e.spool = a.spool
	} else {
		e.file = a.mem.Bytes()
	}
	return hex.EncodeToString(a.h.Sum(nil))
}

func (e *exportEntry) size() int64 {
	if e.spool != nil {
		return e.spool.size
	}
	return int64(len(e.file))
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
	if e.spool != nil {
		return io.NewSectionReader(e.spool.f, 0, e.spool.size)
	}
	return io.NewSectionReader(bytes.NewReader(e.file), 0, int64(len(e.file)))
}
