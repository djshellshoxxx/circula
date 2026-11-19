package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var manifestVersionKey = []byte("manifest-version-regression-key-0123456789")

func manifestVersionFixture() Manifest {
	return Manifest{
		Format: 1, CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		Reason: "manual", AppVersion: "v1.0", SchemaVersion: 16,
		File: "snapshot.db", Bytes: 123, SHA256: "abcdef", Integrity: "ok",
		Ledger: LedgerSummary{Status: "not-run", Networks: 2, Reason: "older schema"},
	}
}

// CRYPTO-02: the signature authenticates the attestation, including each
// summary field, metadata and fractional time, for plain and encrypted files.
func TestManifestSignatureV2CoversAllAttestationFields(t *testing.T) {
	for _, encrypted := range []bool{false, true} {
		m := manifestVersionFixture()
		m.HMACVersion = 2
		m.Encrypted = encrypted
		if encrypted {
			m.Cipher, m.PlaintextBytes, m.PlaintextSHA256 = "AES-256-GCM", 100, "012345"
		}
		m.HMAC = manifestMAC(&m, manifestVersionKey)
		if ok, err := checkManifestAuth(&m, manifestVersionKey); !ok || err != nil {
			t.Fatalf("encrypted=%v valid signature: %v %v", encrypted, ok, err)
		}
		for name, mutate := range map[string]func(*Manifest){
			"format":          func(m *Manifest) { m.Format++ },
			"fractional time": func(m *Manifest) { m.CreatedAt = m.CreatedAt.Add(time.Nanosecond) },
			"reason":          func(m *Manifest) { m.Reason += "\nmanual" },
			"app version":     func(m *Manifest) { m.AppVersion = "forged" },
			"schema":          func(m *Manifest) { m.SchemaVersion++ },
			"bytes":           func(m *Manifest) { m.Bytes++ },
			"digest":          func(m *Manifest) { m.SHA256 = "000000" },
			"integrity":       func(m *Manifest) { m.Integrity = "failed" },
			"ledger status":   func(m *Manifest) { m.Ledger.Status = "ok" },
			"ledger networks": func(m *Manifest) { m.Ledger.Networks++ },
			"ledger failing":  func(m *Manifest) { m.Ledger.Failing++ },
			"ledger reason":   func(m *Manifest) { m.Ledger.Reason = "verified" },
			"encrypted":       func(m *Manifest) { m.Encrypted = !m.Encrypted },
			"cipher":          func(m *Manifest) { m.Cipher = "other" },
			"plaintext bytes": func(m *Manifest) { m.PlaintextBytes++ },
			"plaintext hash":  func(m *Manifest) { m.PlaintextSHA256 = "changed" },
		} {
			cp := m
			mutate(&cp)
			if ok, err := checkManifestAuth(&cp, manifestVersionKey); ok || err == nil {
				t.Errorf("encrypted=%v: %s tampering authenticated", encrypted, name)
			}
		}
		m.File = "renamed.db"
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		var roundTrip Manifest
		if err := json.Unmarshal(b, &roundTrip); err != nil {
			t.Fatal(err)
		}
		if ok, err := checkManifestAuth(&roundTrip, manifestVersionKey); !ok || err != nil {
			t.Errorf("renamed, serialized manifest: %v %v", ok, err)
		}
	}
}

func TestManifestSignatureVersionCannotBeDowngraded(t *testing.T) {
	m := manifestVersionFixture()
	m.HMACVersion = 2
	m.HMAC = manifestMAC(&m, manifestVersionKey)
	b, _ := json.Marshal(m)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		t.Fatal(err)
	}
	delete(fields, "hmac_version")
	b, _ = json.Marshal(fields)
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	stripped, err := readManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := checkManifestAuth(stripped, manifestVersionKey); ok || err == nil {
		t.Fatal("stripped v2 version authenticated as legacy")
	}
	for _, version := range []int{-1, 1, 3} {
		m.HMACVersion = version
		if ok, err := checkManifestAuth(&m, manifestVersionKey); ok || err == nil {
			t.Errorf("unsupported version %d authenticated", version)
		}
		if err := writeManifest(path, &m); err != nil {
			t.Fatal(err)
		}
		// Loading also rejects unknown versions before an allow-unsigned
		// restore can proceed without a configured authentication key.
		if _, err := readManifest(path); err == nil {
			t.Errorf("unsupported version %d loaded", version)
		}
	}
}

// Fixed vectors use the pre-versioning payload, independently of manifestMAC.
// They retain compatibility for historical plain and encrypted manifests.
func TestManifestSignatureLegacyCompatibility(t *testing.T) {
	for _, tc := range []struct {
		encrypted bool
		mac       string
	}{
		{false, "fdf59c2c7af1465dddbfeaf04db37335ba78ea7dec2a486673bc9f5c15829a5b"},
		{true, "9d081a1bcc5c6d07b0a20d4e79362e281916e1247820fb7e4f18f737cf8864ba"},
	} {
		m := manifestVersionFixture()
		m.Encrypted, m.HMAC = tc.encrypted, tc.mac
		if tc.encrypted {
			m.Cipher, m.PlaintextBytes, m.PlaintextSHA256 = "AES-256-GCM", 100, "012345"
		}
		path := filepath.Join(t.TempDir(), "legacy.manifest.json")
		if err := writeManifest(path, &m); err != nil {
			t.Fatal(err)
		}
		got, err := readManifest(path)
		if err != nil {
			t.Fatal(err)
		}
		if ok, err := checkManifestAuth(got, manifestVersionKey); !ok || err != nil {
			t.Errorf("encrypted=%v legacy signature: %v %v", tc.encrypted, ok, err)
		}
		got.HMACVersion = 2
		if ok, err := checkManifestAuth(got, manifestVersionKey); ok || err == nil {
			t.Error("legacy signature accepted as a v2 attestation")
		}
	}
}

func TestSnapshotWritesVersionedManifestSignature(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	dest := filepath.Join(dir, "signed.db")
	m, err := snapshot(context.Background(), raw, dest, "manual", time.Now(), nil, manifestVersionKey)
	if err != nil {
		t.Fatal(err)
	}
	if m.HMACVersion != 2 {
		t.Fatalf("new snapshot HMAC version = %d, want 2", m.HMACVersion)
	}
	got, err := readManifest(dest + ".manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := checkManifestAuth(got, manifestVersionKey); !ok || err != nil {
		t.Fatalf("snapshot signature: %v %v", ok, err)
	}
}
