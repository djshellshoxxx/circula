package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	"time"
)

// Fuzz targets for the encrypted backup container and the manifest parser
// (SA2-11, OPS-019). Seeds live in the f.Add calls below and in
// testdata/fuzz/<Target>/. Run each for a while with, for example:
//
//	go test ./cmd/circula -run '^$' -fuzz FuzzDecryptContainer -fuzztime 30s
var fuzzSecret = []byte("MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=") // base64 of 32 bytes

func fuzzSealed(plain []byte, chunk int) []byte {
	var buf bytes.Buffer
	w, err := newEncryptWriter(&buf, fuzzSecret, chunk)
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(plain); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// FuzzDecryptContainer: arbitrary bytes never panic the decryptor, and bytes
// that were not produced by the writer under this key are never accepted
// (the only accepted inputs are the seeds themselves, which are valid).
func FuzzDecryptContainer(f *testing.F) {
	valid := fuzzSealed([]byte("hello backup"), 4)
	f.Add(valid)
	f.Add(fuzzSealed(nil, 0))
	f.Add(valid[:len(valid)-1])
	f.Add(valid[:backupHeaderLen])
	f.Add([]byte(backupMagic))
	f.Add([]byte("SQLite format 3\x00"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		dr, err := newDecryptReader(bytes.NewReader(data), fuzzSecret)
		if err != nil {
			return
		}
		out, err := io.ReadAll(dr)
		if err != nil {
			return
		}
		// Accepted: it must be a genuine container, i.e. re-reading is stable
		// and the header magic is present.
		if !bytes.HasPrefix(data, []byte(backupMagic)) {
			t.Fatalf("accepted %d bytes without the container magic", len(data))
		}
		dr2, err := newDecryptReader(bytes.NewReader(data), fuzzSecret)
		if err != nil {
			t.Fatalf("second read failed: %v", err)
		}
		out2, err := io.ReadAll(dr2)
		if err != nil || !bytes.Equal(out, out2) {
			t.Fatalf("unstable decryption: %v", err)
		}
	})
}

// FuzzTamperRejected: any change to a valid container (byte edit, truncation,
// appended data) is rejected; none yields plaintext.
func FuzzTamperRejected(f *testing.F) {
	f.Add(0, byte(1), 0, []byte(nil))
	f.Add(backupHeaderLen, byte(0x80), 0, []byte(nil))
	f.Add(1<<20, byte(0), 1, []byte(nil))
	f.Add(0, byte(0), 0, []byte("x"))
	f.Add(0, byte(0), 17, []byte(nil))
	plain := bytes.Repeat([]byte("ledger "), 40)
	f.Fuzz(func(t *testing.T, pos int, xor byte, cut int, extra []byte) {
		ct := fuzzSealed(plain, 64)
		mod := append([]byte(nil), ct...)
		if len(mod) > 0 && xor != 0 {
			if pos < 0 {
				pos = -(pos + 1)
			}
			mod[pos%len(mod)] ^= xor
		}
		if cut < 0 {
			cut = -(cut + 1)
		}
		if cut > 0 && len(mod) > 0 {
			mod = mod[:len(mod)-(cut%len(mod))]
		}
		mod = append(mod, extra...)
		if bytes.Equal(mod, ct) {
			return // unchanged
		}
		dr, err := newDecryptReader(bytes.NewReader(mod), fuzzSecret)
		if err != nil {
			return
		}
		out, err := io.ReadAll(dr)
		if err == nil {
			t.Fatalf("tampered container accepted (%d bytes of plaintext, %d-byte input)", len(out), len(mod))
		}
	})
}

// FuzzWrongKey: a container never opens under another secret.
func FuzzWrongKey(f *testing.F) {
	f.Add([]byte("another-secret-passphrase-123"), []byte("data"))
	f.Add([]byte("MTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTE="), []byte{})
	f.Fuzz(func(t *testing.T, secret, plain []byte) {
		if bytes.Equal(secret, fuzzSecret) || len(secret) == 0 {
			return
		}
		ct := fuzzSealed(plain, 16)
		dr, err := newDecryptReader(bytes.NewReader(ct), secret)
		if err != nil {
			return
		}
		if _, err := io.ReadAll(dr); err == nil {
			t.Fatal("container opened under a different secret")
		}
	})
}

// FuzzManifest: manifest parsing and signature checking never panic, and an
// unsigned or foreign-signed manifest is never authenticated.
func FuzzManifest(f *testing.F) {
	key := []byte("0123456789abcdef-backup-key")
	m := &Manifest{Format: manifestFormat, CreatedAt: time.Unix(1700000000, 0).UTC(), SchemaVersion: 7, File: "x.db", Bytes: 10, SHA256: "ab"}
	m.HMAC = manifestMAC(m, key)
	good, _ := json.Marshal(m)
	f.Add(good)
	f.Add([]byte(`{"format":1}`))
	f.Add([]byte(`{"format":1,"hmac_sha256":"zz"}`))
	f.Add([]byte(`null`))
	f.Add([]byte(``))
	f.Fuzz(func(t *testing.T, data []byte) {
		var got Manifest
		if err := json.Unmarshal(data, &got); err != nil {
			return
		}
		ok, err := checkManifestAuth(&got, key)
		if ok && err != nil {
			t.Fatal("authenticated with an error")
		}
		if ok2, _ := checkManifestAuth(&got, []byte("a-completely-different-key-1234")); ok2 {
			t.Fatal("manifest authenticated under another key")
		}
		if ok3, err := checkManifestAuth(&got, nil); ok3 || err != nil {
			t.Fatalf("no key: %v %v", ok3, err)
		}
	})
}
