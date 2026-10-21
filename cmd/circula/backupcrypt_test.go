package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/djshellshoxxx/circula/internal/store"
)

const testChunk = 1024

func testKey(t *testing.T) []byte {
	t.Helper()
	k, err := newBackupKey()
	if err != nil {
		t.Fatal(err)
	}
	return []byte(k)
}

func encryptBytes(t *testing.T, plain, secret []byte, chunk int) []byte {
	t.Helper()
	var buf bytes.Buffer
	w, err := newEncryptWriter(&buf, secret, chunk)
	if err != nil {
		t.Fatal(err)
	}
	// Write in awkward pieces to exercise the buffering.
	for rest := plain; len(rest) > 0; {
		n := min(len(rest), 777)
		if _, err := w.Write(rest[:n]); err != nil {
			t.Fatal(err)
		}
		rest = rest[n:]
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func decryptBytes(ct, secret []byte) ([]byte, error) {
	r, err := newDecryptReader(bytes.NewReader(ct), secret)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func TestBackupCryptRoundTripSizes(t *testing.T) {
	key := testKey(t)
	for _, n := range []int{0, 1, testChunk - 1, testChunk, testChunk + 1, 3 * testChunk, 5*testChunk + 7} {
		plain := randBytes(n)
		ct := encryptBytes(t, plain, key, testChunk)
		if !bytes.HasPrefix(ct, []byte(backupMagic)) {
			t.Fatal("missing magic")
		}
		got, err := decryptBytes(ct, key)
		if err != nil || !bytes.Equal(got, plain) {
			t.Fatalf("size %d: err=%v equal=%v", n, err, bytes.Equal(got, plain))
		}
	}
}

func TestBackupCryptPassphrase(t *testing.T) {
	pass := []byte("correct horse battery staple")
	plain := randBytes(3*testChunk + 5)
	ct := encryptBytes(t, plain, pass, testChunk)
	if ct[len(backupMagic)] != kdfArgon2 {
		t.Error("passphrase must use argon2id")
	}
	got, err := decryptBytes(ct, pass)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("passphrase round trip: %v", err)
	}
	if _, err := decryptBytes(ct, []byte("correct horse battery stapl3")); !errors.Is(err, errBackupWrongKey) {
		t.Errorf("wrong passphrase: %v", err)
	}
	// A base64 key cannot open a passphrase backup and vice versa.
	if _, err := decryptBytes(ct, testKey(t)); !errors.Is(err, errBackupWrongKey) {
		t.Errorf("key kind mismatch: %v", err)
	}
}

func TestBackupCryptLarge(t *testing.T) {
	key := testKey(t)
	plain := randBytes(3*defaultChunk + 123) // several default-size chunks
	ct := encryptBytes(t, plain, key, 0)
	got, err := decryptBytes(ct, key)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("large round trip: %v", err)
	}
}

func TestBackupCryptWrongKey(t *testing.T) {
	plain := randBytes(2*testChunk + 3)
	ct := encryptBytes(t, plain, testKey(t), testChunk)
	got, err := decryptBytes(ct, testKey(t))
	if !errors.Is(err, errBackupWrongKey) || len(got) != 0 {
		t.Fatalf("wrong key: err=%v released=%d bytes", err, len(got))
	}
}

func TestBackupCryptBitFlipAnywhere(t *testing.T) {
	key := testKey(t)
	plain := randBytes(3*testChunk + 10)
	ct := encryptBytes(t, plain, key, testChunk)
	// Header bytes after the magic, the first, a middle and the final chunk.
	for _, off := range []int{len(backupMagic) + 1, len(backupMagic) + 6, backupHeaderLen - 1,
		backupHeaderLen, backupHeaderLen + testChunk + gcmTagLen + 5, len(ct) - 1} {
		bad := append([]byte(nil), ct...)
		bad[off] ^= 0x01
		if got, err := decryptBytes(bad, key); err == nil {
			t.Errorf("flip at %d accepted (%d bytes)", off, len(got))
		}
	}
	bad := append([]byte(nil), ct...)
	bad[0] ^= 0xff
	if _, err := decryptBytes(bad, key); !errors.Is(err, errBackupNotEncrypted) {
		t.Errorf("bad magic: %v", err)
	}
}

func TestBackupCryptTruncation(t *testing.T) {
	key := testKey(t)
	plain := randBytes(3*testChunk + 10) // 4 chunks
	ct := encryptBytes(t, plain, key, testChunk)
	full := testChunk + gcmTagLen
	// Cut exactly at chunk boundaries: the file looks well-formed but lacks the final chunk.
	for k := 1; k <= 3; k++ {
		_, err := decryptBytes(ct[:backupHeaderLen+k*full], key)
		if !errors.Is(err, errBackupTruncated) {
			t.Errorf("cut after chunk %d: %v", k, err)
		}
	}
	// Only the header, and a cut inside the header.
	if _, err := decryptBytes(ct[:backupHeaderLen], key); !errors.Is(err, errBackupTruncated) {
		t.Errorf("header only: %v", err)
	}
	if _, err := decryptBytes(ct[:backupHeaderLen-3], key); err == nil {
		t.Error("cut header accepted")
	}
	// Cut inside a chunk and inside a tag.
	for _, cut := range []int{backupHeaderLen + 100, backupHeaderLen + full + full/2, len(ct) - 1, len(ct) - gcmTagLen - 1} {
		if _, err := decryptBytes(ct[:cut], key); err == nil {
			t.Errorf("cut at %d accepted", cut)
		}
	}
	// A whole file exactly a multiple of the chunk size: dropping the final chunk is caught too.
	exact := encryptBytes(t, randBytes(2*testChunk), key, testChunk)
	if _, err := decryptBytes(exact[:backupHeaderLen+full], key); !errors.Is(err, errBackupTruncated) {
		t.Errorf("exact-multiple truncation: %v", err)
	}
}

func TestBackupCryptReorderAndAppend(t *testing.T) {
	key := testKey(t)
	plain := randBytes(4*testChunk + 10) // chunks 0..3 full, 4 final
	ct := encryptBytes(t, plain, key, testChunk)
	full := testChunk + gcmTagLen
	chunk := func(i int) []byte { return ct[backupHeaderLen+i*full : backupHeaderLen+(i+1)*full] }

	swapped := append([]byte(nil), ct[:backupHeaderLen]...)
	for _, i := range []int{0, 2, 1, 3} {
		swapped = append(swapped, chunk(i)...)
	}
	swapped = append(swapped, ct[backupHeaderLen+4*full:]...)
	if _, err := decryptBytes(swapped, key); !errors.Is(err, errBackupWrongKey) {
		t.Errorf("reordered chunks: %v", err)
	}

	dup := append([]byte(nil), ct[:backupHeaderLen+full]...)
	dup = append(dup, ct[backupHeaderLen:]...) // chunk 0 twice
	if _, err := decryptBytes(dup, key); err == nil {
		t.Error("duplicated chunk accepted")
	}

	// Trailing bytes after the final chunk.
	if _, err := decryptBytes(append(append([]byte(nil), ct...), 0), key); err == nil {
		t.Error("trailing byte accepted")
	}
	// A full valid chunk appended after the final one.
	if _, err := decryptBytes(append(append([]byte(nil), ct...), chunk(0)...), key); err == nil {
		t.Error("appended chunk accepted")
	}
	// Splicing chunks of another backup under this header.
	other := encryptBytes(t, plain, key, testChunk)
	spliced := append([]byte(nil), ct[:backupHeaderLen+full]...)
	spliced = append(spliced, other[backupHeaderLen+full:]...)
	if _, err := decryptBytes(spliced, key); err == nil {
		t.Error("spliced backups accepted")
	}
}

func TestBackupCryptNoncesDiffer(t *testing.T) {
	key := testKey(t)
	plain := randBytes(100)
	a, b := encryptBytes(t, plain, key, testChunk), encryptBytes(t, plain, key, testChunk)
	if bytes.Equal(a, b) {
		t.Fatal("two encryptions of the same data are identical (salt or nonce reuse)")
	}
}

func TestEncryptDecryptFiles(t *testing.T) {
	dir := t.TempDir()
	key := testKey(t)
	src := filepath.Join(dir, "plain")
	plain := randBytes(5*defaultChunk/2 + 1)
	if err := os.WriteFile(src, plain, 0o600); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(dir, "x.enc")
	sum, size, err := encryptFile(src, enc, key)
	if err != nil {
		t.Fatal(err)
	}
	if want, wsize, _ := fileSHA256(enc); want != sum || wsize != size {
		t.Errorf("ciphertext digest mismatch")
	}
	if fi, _ := os.Stat(enc); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 { // Windows has no POSIX modes
		t.Errorf("mode = %o", fi.Mode().Perm())
	}
	out := filepath.Join(dir, "out")
	psum, psize, err := decryptFile(enc, out, key)
	if err != nil {
		t.Fatal(err)
	}
	if wsum, _, _ := fileSHA256(src); psum != wsum || psize != int64(len(plain)) {
		t.Error("plaintext digest mismatch")
	}
	// Wrong key leaves no plaintext file behind.
	out2 := filepath.Join(dir, "out2")
	if _, _, err := decryptFile(enc, out2, testKey(t)); !errors.Is(err, errBackupWrongKey) {
		t.Fatalf("wrong key: %v", err)
	}
	if _, err := os.Stat(out2); err == nil {
		t.Error("partial plaintext left after failed decryption")
	}
	// A plaintext file is not an encrypted backup.
	if _, _, err := decryptFile(src, filepath.Join(dir, "out3"), key); !errors.Is(err, errBackupNotEncrypted) {
		t.Errorf("plaintext input: %v", err)
	}
}

func TestParseBackupSecret(t *testing.T) {
	k := testKey(t)
	if kind, ikm := parseBackupSecret(k); kind != kdfHKDF || len(ikm) != 32 {
		t.Error("base64 key not recognised")
	}
	if kind, _ := parseBackupSecret([]byte("this is a long passphrase")); kind != kdfArgon2 {
		t.Error("passphrase not recognised")
	}
}

// ---- integration ----

func writeKeyFile(t *testing.T, dir, name string, key []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, append(append([]byte(nil), key...), '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func encConfig(db string, key []byte) *Config {
	cfg := testConfigFor(db)
	cfg.BackupEncryptionKey = key
	return cfg
}

// listDir returns the names in dir.
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	return names
}

func TestEncryptedBackupAndRestoreIntegration(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	key := testKey(t)
	cfg := encConfig(db, key)
	ctx := context.Background()
	bk := filepath.Join(dir, "bk")

	mf, path, err := backupTo(ctx, cfg, bk, "manual", nil)
	if err != nil {
		t.Fatalf("backupTo: %v", err)
	}
	if !strings.HasSuffix(path, ".db.enc") || !mf.Encrypted || mf.Cipher == "" || mf.Ledger.Status != "ok" {
		t.Fatalf("manifest %+v path %s", mf, path)
	}
	// No plaintext remnant: only the ciphertext and its manifest.
	names := listDir(t, bk)
	if len(names) != 2 {
		t.Fatalf("backup dir = %v", names)
	}
	raw, _ := os.ReadFile(path)
	if bytes.Contains(raw, []byte("SQLite format 3")) || bytes.Contains(raw, []byte("founder01")) {
		t.Error("ciphertext contains plaintext database content")
	}
	if fi, _ := os.Stat(path); runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Errorf("backup mode %o", fi.Mode().Perm())
	}
	if mf.SHA256 == mf.PlaintextSHA256 || mf.Bytes <= mf.PlaintextBytes {
		t.Errorf("manifest conflates ciphertext and plaintext: %+v", mf)
	}
	if got, _, _ := fileSHA256(path); got != mf.SHA256 {
		t.Error("manifest sha256 is not the ciphertext digest")
	}

	// Restore into a fresh location with the right key.
	keyFile := writeKeyFile(t, dir, "enc.key", key)
	target := filepath.Join(dir, "restored", "circula.db")
	if err := os.Mkdir(filepath.Dir(target), 0o700); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", target, "-backup-encryption-key-file", keyFile, path}, &out, &errOut)
	if code != exitOK {
		t.Fatalf("restore: %d %s", code, errOut.String())
	}
	st, err := store.Open(ctx, target)
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	if entries := listDir(t, filepath.Dir(target)); len(entries) != 2 { // db + lock/backups dir only
		for _, e := range entries {
			if strings.Contains(e, "partial") || strings.Contains(e, "verify") {
				t.Errorf("leftover %s", e)
			}
		}
	}

	// Wrong key: refused with a clear error, database not created.
	bad := writeKeyFile(t, dir, "bad.key", testKey(t))
	target2 := filepath.Join(dir, "restored", "second.db")
	out.Reset()
	errOut.Reset()
	code = run([]string{"restore", "-database", target2, "-backup-encryption-key-file", bad, path}, &out, &errOut)
	if code == exitOK || !strings.Contains(errOut.String(), "wrong backup-encryption-key") {
		t.Fatalf("wrong key restore: %d %s", code, errOut.String())
	}
	if _, err := os.Stat(target2); err == nil {
		t.Error("database created from a wrong-key restore")
	}
	for _, n := range listDir(t, filepath.Dir(target2)) {
		if strings.Contains(n, "restore-partial") {
			t.Errorf("plaintext partial left: %s", n)
		}
	}

	// No key configured at all.
	out.Reset()
	errOut.Reset()
	code = run([]string{"restore", "-database", target2, path}, &out, &errOut)
	if code == exitOK || !strings.Contains(errOut.String(), "backup is encrypted") {
		t.Fatalf("keyless restore: %d %s", code, errOut.String())
	}

	// A flipped ciphertext bit is caught by the manifest checksum, and, with
	// the checksum bypassed, by GCM.
	tampered := append([]byte(nil), raw...)
	tampered[len(tampered)/2] ^= 1
	tpath := filepath.Join(bk, "circula-20200101T000000Z.db.enc")
	if err := os.WriteFile(tpath, tampered, 0o600); err != nil {
		t.Fatal(err)
	}
	mb, _ := os.ReadFile(path + ".manifest.json")
	if err := os.WriteFile(tpath+".manifest.json", mb, 0o600); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if code := run([]string{"restore", "-database", target2, "-backup-encryption-key-file", keyFile, tpath}, &out, &errOut); code == exitOK ||
		!strings.Contains(errOut.String(), "checksum") {
		t.Fatalf("tampered restore: %d %s", code, errOut.String())
	}
	var tm Manifest
	_ = json.Unmarshal(mb, &tm)
	tm.SHA256, _, _ = fileSHA256(tpath)
	if err := writeManifest(tpath+".manifest.json", &tm); err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if code := run([]string{"restore", "-database", target2, "-backup-encryption-key-file", keyFile, tpath}, &out, &errOut); code == exitOK ||
		!strings.Contains(errOut.String(), "wrong backup-encryption-key, or the backup was altered") {
		t.Fatalf("GCM-detected tamper: %d %s", code, errOut.String())
	}
}

func TestEncryptedBackupEmptyDatabase(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	db := filepath.Join(dir, "circula.db")
	st, err := store.Open(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	_ = st.Close()
	key := testKey(t)
	cfg := encConfig(db, key)
	mf, path, err := backupTo(ctx, cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil || !mf.Encrypted {
		t.Fatalf("empty db backup: %v %+v", err, mf)
	}
	target := filepath.Join(dir, "r.db")
	var out, errOut bytes.Buffer
	kf := writeKeyFile(t, dir, "k", key)
	if code := run([]string{"restore", "-database", target, "-backup-encryption-key-file", kf, path}, &out, &errOut); code != exitOK {
		t.Fatalf("restore empty: %d %s", code, errOut.String())
	}
}

func TestPlaintextBackupRefusedWhenEncryptionRequired(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	ctx := context.Background()
	_, path, err := backupTo(ctx, testConfigFor(db), filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	kf := writeKeyFile(t, dir, "k", testKey(t))
	target := filepath.Join(dir, "r.db")
	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", target, "-backup-encryption-key-file", kf, "-backup-require-encryption", "true", path}, &out, &errOut)
	if code == exitOK || !strings.Contains(errOut.String(), "not encrypted") {
		t.Fatalf("plaintext restore under require: %d %s", code, errOut.String())
	}
	if _, err := os.Stat(target); err == nil {
		t.Error("database created")
	}
	// Without the requirement, a plaintext backup still restores (migration of old backups).
	errOut.Reset()
	if code := run([]string{"restore", "-database", target, "-backup-encryption-key-file", kf, path}, &out, &errOut); code != exitOK {
		t.Fatalf("plaintext restore: %d %s", code, errOut.String())
	}
	// The requirement without a key is a configuration error, and backups refuse too.
	errOut.Reset()
	if code := run([]string{"backup", "-database", db, "-backup-require-encryption", "true", filepath.Join(dir, "x.db")}, &out, &errOut); code != exitUsage {
		t.Errorf("require without key: %d %s", code, errOut.String())
	}
	cfg := testConfigFor(db)
	cfg.BackupRequireEncryption = true
	if _, _, err := backupTo(ctx, cfg, filepath.Join(dir, "bk2"), "manual", nil); err == nil {
		t.Error("backupTo wrote a plaintext backup under require-encryption")
	}
}

func TestManifestHMACCoversEncryption(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := encConfig(db, testKey(t))
	sign := []byte("manifest-signing-key-0123456789")
	cfg.BackupKey = sign
	mf, path, err := backupTo(context.Background(), cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := checkManifestAuth(mf, sign); !ok || err != nil {
		t.Fatalf("auth: %v %v", ok, err)
	}
	for name, mutate := range map[string]func(*Manifest){
		"plaintext sha": func(m *Manifest) { m.PlaintextSHA256 = strings.Repeat("0", 64) },
		"plaintext len": func(m *Manifest) { m.PlaintextBytes++ },
		"encrypted off": func(m *Manifest) { m.Encrypted = false },
		"cipher":        func(m *Manifest) { m.Cipher = "none" },
	} {
		cp := *mf
		mutate(&cp)
		if ok, err := checkManifestAuth(&cp, sign); ok || err == nil {
			t.Errorf("%s: forged manifest accepted", name)
		}
	}
	// A manifest of an encrypted file that claims plaintext is rejected before any work.
	cp := *mf
	cp.Encrypted = false
	if err := checkEncryptionPolicy(path, &cp, cfg); err == nil {
		t.Error("encrypted file with plaintext manifest accepted")
	}
}

func TestFailedEncryptedBackupLeavesNothing(t *testing.T) {
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	raw, err := openRaw(db)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = raw.Close() }()
	bk := filepath.Join(dir, "bk")
	if err := os.Mkdir(bk, 0o700); err != nil {
		t.Fatal(err)
	}
	boom := func(context.Context, string) (LedgerSummary, error) { return LedgerSummary{}, errors.New("boom") }
	_, err = snapshotWith(context.Background(), raw, filepath.Join(bk, "circula-20260101T000000Z.db.enc"), "manual", time.Now(), boom,
		snapshotKeys{Encrypt: testKey(t)})
	if err == nil {
		t.Fatal("expected failure")
	}
	if names := listDir(t, bk); len(names) != 0 {
		t.Errorf("failed backup left %v", names)
	}
}

func TestStaleTempDirSweptAndFresh(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, tempDirPrefix+"old")
	fresh := filepath.Join(dir, tempDirPrefix+"new")
	for _, d := range []string{stale, fresh} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "snapshot.db"), []byte("secret rows"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	sweepStaleTemp(dir)
	if _, err := os.Stat(stale); err == nil {
		t.Error("stale plaintext work dir not removed")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("fresh work dir (possibly another process's) was removed")
	}
}

func TestPruneAndListHandleEncryptedBackups(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 10, 20, 3, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, encryptedBackupFileName(base.AddDate(0, 0, -i)))
		for _, p := range []string{name, name + ".manifest.json"} {
			if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	got, err := listBackups(dir)
	if err != nil || len(got) != 5 {
		t.Fatalf("list: %v %v", got, err)
	}
	removed, err := pruneBackups(dir, 2, 0)
	if err != nil || len(removed) != 3 {
		t.Fatalf("prune: %v %v", removed, err)
	}
	if names := listDir(t, dir); len(names) != 4 { // 2 backups + 2 manifests
		t.Errorf("after prune: %v", names)
	}
}

func TestEncryptionKeyConfig(t *testing.T) {
	dir := t.TempDir()
	key := "sUper-secret-passphrase-that-must-not-leak"
	kf := filepath.Join(dir, "k")
	if err := os.WriteFile(kf, []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-encryption-key-file"): kf}))
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg.BackupEncryptionKey) != key {
		t.Error("key not loaded")
	}
	var out bytes.Buffer
	writeSecretStatus(&out, cfg)
	if strings.Contains(out.String(), key) || !strings.Contains(out.String(), "backup-encryption-key = <set>") {
		t.Errorf("status output: %q", out.String())
	}
	_, vals, _ := resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-encryption-key-file"): kf}))
	var eff bytes.Buffer
	writeEffective(&eff, vals)
	if strings.Contains(eff.String(), key) {
		t.Error("effective settings leak the key")
	}
	// Env and file together is ambiguous; a short key is refused; the same key as backup-key is refused.
	if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{
		envName("backup-encryption-key-file"): kf, envName("backup-encryption-key"): key})); err == nil {
		t.Error("env+file accepted")
	}
	if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-encryption-key"): "short"})); err == nil {
		t.Error("short key accepted")
	}
	if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{
		envName("backup-encryption-key"): key, envName("backup-key"): key})); err == nil {
		t.Error("identical backup-key and backup-encryption-key accepted")
	}
	if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-require-encryption"): "true"})); err == nil {
		t.Error("require-encryption without a key accepted")
	}
	// Group-readable key file refused (POSIX modes only).
	if runtime.GOOS != "windows" {
		if err := os.Chmod(kf, 0o640); err != nil {
			t.Fatal(err)
		}
		if _, _, err := resolveWith(t, nil, baseEnv(t, map[string]string{envName("backup-encryption-key-file"): kf})); err == nil {
			t.Error("group-readable key file accepted")
		}
	}
}

func TestKeygen(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"keygen"}, &out, &errOut); code != exitOK {
		t.Fatalf("keygen: %d %s", code, errOut.String())
	}
	k := strings.TrimSpace(out.String())
	if kind, ikm := parseBackupSecret([]byte(k)); kind != kdfHKDF || len(ikm) != 32 {
		t.Errorf("keygen output %q is not a 32-byte key", k)
	}
	out2 := bytes.Buffer{}
	_ = run([]string{"keygen"}, &out2, &errOut)
	if out.String() == out2.String() {
		t.Error("keygen repeats")
	}
	if code := run([]string{"keygen", "extra"}, &out, &errOut); code != exitUsage {
		t.Errorf("keygen with args: %d", code)
	}
}
