package main

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djshellshoxxx/circula/internal/service"
)

// A compressed member can yield every declared byte and still lack its final
// block. Correct lengths, CRC and manifest checksums must not certify that ZIP.
// The Go reader already refuses this case; keep the Python reader in agreement.
func TestVerifyExportRequiresCompletedDeflateStream(t *testing.T) {
	data := []byte("id\nm1\n")
	sum := sha256.Sum256(data)
	manifest, err := json.Marshal(service.Manifest{SchemaVersion: service.ExportSchemaVersion,
		Scope: service.ExportMembers, Format: service.FormatCSV,
		Files: []service.ManifestFile{{Name: "members.csv", Rows: 1, Bytes: int64(len(data)),
			SHA256: hex.EncodeToString(sum[:])}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"members.csv", "manifest.json"} {
		for _, complete := range []bool{false, true} {
			label := target + "/incomplete"
			if complete {
				label = target + "/complete"
			}
			t.Run(label, func(t *testing.T) {
				var archive bytes.Buffer
				zw := zip.NewWriter(&archive)
				for _, member := range []struct {
					name string
					data []byte
				}{{"members.csv", data}, {"manifest.json", manifest}} {
					var compressed bytes.Buffer
					fw, err := flate.NewWriter(&compressed, flate.NoCompression)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := fw.Write(member.data); err != nil {
						t.Fatal(err)
					}
					if err := fw.Flush(); err != nil {
						t.Fatal(err)
					}
					unfinished := compressed.Len()
					if err := fw.Close(); err != nil {
						t.Fatal(err)
					}
					body := compressed.Bytes()
					if member.name == target && !complete {
						body = body[:unfinished]
					}
					writer, err := zw.CreateRaw(&zip.FileHeader{Name: member.name, Method: zip.Deflate,
						CompressedSize64: uint64(len(body)), UncompressedSize64: uint64(len(member.data)),
						CRC32: crc32.ChecksumIEEE(member.data)})
					if err != nil {
						t.Fatal(err)
					}
					if _, err := writer.Write(body); err != nil {
						t.Fatal(err)
					}
				}
				if err := zw.Close(); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(t.TempDir(), "completion.zip")
				if err := os.WriteFile(path, archive.Bytes(), 0o600); err != nil {
					t.Fatal(err)
				}
				code, output := runVerifyExport(path)
				if complete {
					if code != exitOK {
						t.Fatalf("complete stream refused: code=%d output=%s", code, output)
					}
				} else if code != exitFailure || !strings.Contains(output, "unexpected EOF") {
					t.Fatalf("unfinished stream not refused as truncated: code=%d output=%s", code, output)
				}
			})
		}
	}
}
