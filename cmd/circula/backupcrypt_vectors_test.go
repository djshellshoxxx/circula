package main

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"
)

// Container layout is a stable on-disk format: header size, nonce layout and
// the AAD (header | index | final) are pinned here so a refactor cannot change
// them silently (NIST SP 800-38D 8.2.1 deterministic counter field; STREAM
// final-block flag).
func TestBackupContainerLayoutPinned(t *testing.T) {
	if backupHeaderLen != 7+1+4+16+8 {
		t.Fatalf("header length %d", backupHeaderLen)
	}
	prefix := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	n := chunkNonce(prefix, 0x0a0b0c0d)
	if want := []byte{1, 2, 3, 4, 5, 6, 7, 8, 0x0a, 0x0b, 0x0c, 0x0d}; !bytes.Equal(n, want) || len(n) != 12 {
		t.Errorf("nonce %x", n)
	}
	hdr := []byte("HEADER")
	a := chunkAAD(hdr, 5, false)
	b := chunkAAD(hdr, 5, true)
	if !bytes.Equal(a[:len(hdr)], hdr) || binary.BigEndian.Uint64(a[len(hdr):]) != 5 || a[len(a)-1] != 0 || b[len(b)-1] != 1 || len(a) != len(hdr)+9 {
		t.Errorf("aad a=%x b=%x", a, b)
	}
}

// Two files from the same secret must use different keys and nonce prefixes.
func TestBackupFreshSaltAndPrefixPerFile(t *testing.T) {
	key, err := newBackupKey()
	if err != nil {
		t.Fatal(err)
	}
	var h1, h2 bytes.Buffer
	for _, buf := range []*bytes.Buffer{&h1, &h2} {
		w, err := newEncryptWriter(buf, []byte(key), 0)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("same plaintext"))
		_ = w.Close()
	}
	a, b := h1.Bytes()[:backupHeaderLen], h2.Bytes()[:backupHeaderLen]
	if bytes.Equal(a[13:29], b[13:29]) {
		t.Error("salt reused")
	}
	if bytes.Equal(a[29:], b[29:]) {
		t.Error("nonce prefix reused")
	}
	if bytes.Equal(h1.Bytes()[backupHeaderLen:], h2.Bytes()[backupHeaderLen:]) {
		t.Error("identical ciphertext for identical plaintext")
	}
}

// RFC 9106 section 4 second recommended option is t=3, m=64 MiB, p=4; the
// passphrase path must derive a 32-byte (AES-256) key deterministically.
func TestBackupPassphraseKeyDerivation(t *testing.T) {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	k1, err := deriveBackupKey([]byte("a long passphrase, not base64"), kdfArgon2, salt)
	if err != nil || len(k1) != 32 {
		t.Fatalf("argon2 key: %v len %d", err, len(k1))
	}
	k2, _ := deriveBackupKey([]byte("a long passphrase, not base64"), kdfArgon2, salt)
	if !bytes.Equal(k1, k2) {
		t.Error("argon2 derivation not deterministic")
	}
	raw, _ := newBackupKey()
	k3, err := deriveBackupKey([]byte(raw), kdfHKDF, salt)
	if err != nil || len(k3) != 32 {
		t.Fatalf("hkdf key: %v len %d", err, len(k3))
	}
	if _, err := deriveBackupKey([]byte(raw), kdfArgon2, salt); err == nil {
		t.Error("kdf byte / secret kind mismatch must fail")
	}
}
