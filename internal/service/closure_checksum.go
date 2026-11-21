package service

import (
	"context"
	"io"

	"github.com/djshellshoxxx/circula/internal/domain"
)

// closureExportVersion identifies the checkpoint byte representation, not the
// downloadable export schema. Version 1 uses the JSON ledger export schema 1
// with generated_at fixed to the persisted closure completion time. Future
// export layout changes must retain this representation or add a new version.
const closureExportVersion = 1

func (s *Service) closureManifest(ctx context.Context, actor domain.Actor, c *domain.Closure) (*Manifest, error) {
	if ExportSchemaVersion != 1 {
		return nil, domain.InvalidState("closure checkpoint version 1 requires export schema 1")
	}
	return s.export(ctx, actor, c.NetworkID, ExportLedger, FormatJSON, checksumSink{ctx}, c.ClosedAt.UTC())
}

// checksumSink discards serialized bytes after the ordinary export writers
// have counted and hashed them. Its memory use is independent of ledger size;
// context checks also stop serialization promptly if the caller goes away.
type checksumSink struct{ ctx context.Context }

func (s checksumSink) Create(string) (io.WriteCloser, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	return checksumFile(s), nil
}

type checksumFile struct{ ctx context.Context }

func (f checksumFile) Write(p []byte) (int, error) {
	if err := f.ctx.Err(); err != nil {
		return 0, err
	}
	return io.Discard.Write(p)
}

func (f checksumFile) Close() error { return f.ctx.Err() }
