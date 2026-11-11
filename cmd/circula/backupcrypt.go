package main

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Encrypted backup container (OPS-019, SEC-028).
//
//	header  = magic "CIRCBK1" | kdf (1) | chunk size (4, BE) | salt (16) | nonce prefix (8)
//	chunk i = AES-256-GCM(key, nonce = prefix | i (4, BE), aad = header | i (8, BE) | final (1))
//
// Every chunk but the last carries exactly chunkSize plaintext bytes; the last
// carries 0..chunkSize and has final=1 in its AAD. Binding the header, the
// index and the final flag into each chunk's tag means a flipped bit, a
// reordered or dropped chunk, a truncation at a chunk boundary and appended
// data are all detected.
const (
	backupMagic     = "CIRCBK1"
	backupHeaderLen = len(backupMagic) + 1 + 4 + saltLen + noncePrefixLen
	saltLen         = 16
	noncePrefixLen  = 8
	gcmTagLen       = 16
	defaultChunk    = 64 << 10
	maxChunk        = 16 << 20

	kdfHKDF   byte = 1 // secret is 32 random bytes (base64): HKDF-SHA256
	kdfArgon2 byte = 2 // secret is a passphrase: Argon2id
)

// Errors reported to the operator by decryption.
var (
	errBackupNotEncrypted = errors.New("file is not an encrypted circula backup (missing CIRCBK1 header)")
	errBackupWrongKey     = errors.New("decryption failed: wrong backup-encryption-key, or the backup was altered")
	errBackupTruncated    = errors.New("encrypted backup is truncated (a final chunk is missing)")
	errBackupTrailing     = errors.New("encrypted backup has data after its final chunk (chunks appended or reordered)")
)

// parseBackupSecret classifies the configured secret: 32 random bytes in
// base64 (recommended) or a passphrase.
func parseBackupSecret(secret []byte) (kind byte, ikm []byte) {
	s := strings.TrimSpace(string(secret))
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(s); err == nil && len(b) == 32 {
			return kdfHKDF, b
		}
	}
	return kdfArgon2, []byte(s)
}

// deriveBackupKey turns the configured secret and the per-file salt into the
// AES-256 key.
func deriveBackupKey(secret []byte, kdf byte, salt []byte) ([]byte, error) {
	kind, ikm := parseBackupSecret(secret)
	if kind != kdf {
		return nil, errBackupWrongKey
	}
	switch kdf {
	case kdfHKDF:
		return hkdf.Key(sha256.New, ikm, salt, "circula backup encryption v1", 32)
	case kdfArgon2:
		return argon2.IDKey(ikm, salt, 3, 64<<10, 4, 32), nil
	}
	return nil, fmt.Errorf("unknown key derivation %d", kdf)
}

// newBackupKey returns a fresh recommended key (32 random bytes, base64).
func newBackupKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func chunkNonce(prefix []byte, idx uint32) []byte {
	n := make([]byte, 12)
	copy(n, prefix)
	binary.BigEndian.PutUint32(n[noncePrefixLen:], idx)
	return n
}

func chunkAAD(header []byte, idx uint32, final bool) []byte {
	aad := make([]byte, 0, len(header)+9)
	aad = append(aad, header...)
	aad = binary.BigEndian.AppendUint64(aad, uint64(idx))
	if final {
		return append(aad, 1)
	}
	return append(aad, 0)
}

// encryptWriter streams plaintext into the encrypted container.
type encryptWriter struct {
	w      io.Writer
	aead   cipher.AEAD
	header []byte
	prefix []byte
	size   int
	buf    []byte
	idx    uint32
	closed bool
}

// newEncryptWriter writes the header to w and returns the writer. chunk <= 0
// selects the default chunk size. The caller must Close it to emit the final chunk.
func newEncryptWriter(w io.Writer, secret []byte, chunk int) (io.WriteCloser, error) {
	if chunk <= 0 {
		chunk = defaultChunk
	}
	kdf, _ := parseBackupSecret(secret)
	header := make([]byte, backupHeaderLen)
	copy(header, backupMagic)
	header[len(backupMagic)] = kdf
	binary.BigEndian.PutUint32(header[len(backupMagic)+1:], uint32(chunk))
	salt := header[len(backupMagic)+5 : len(backupMagic)+5+saltLen]
	prefix := header[len(backupMagic)+5+saltLen:]
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	if _, err := rand.Read(prefix); err != nil {
		return nil, err
	}
	key, err := deriveBackupKey(secret, kdf, salt)
	if err != nil {
		return nil, err
	}
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(header); err != nil {
		return nil, err
	}
	return &encryptWriter{w: w, aead: aead, header: header, prefix: prefix, size: chunk, buf: make([]byte, 0, chunk)}, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

// Write buffers p and flushes a full chunk only when more data arrives, so
// the last chunk (possibly a full one) is always held back for Close.
func (e *encryptWriter) Write(p []byte) (int, error) {
	if e.closed {
		return 0, errors.New("write to closed encrypt writer")
	}
	n := len(p)
	for len(p) > 0 {
		if len(e.buf) == e.size {
			if err := e.flush(false); err != nil {
				return n - len(p), err
			}
		}
		take := min(e.size-len(e.buf), len(p))
		e.buf = append(e.buf, p[:take]...)
		p = p[take:]
	}
	return n, nil
}

func (e *encryptWriter) flush(final bool) error {
	if e.idx == ^uint32(0) {
		return errors.New("backup too large for the chunk counter")
	}
	ct := e.aead.Seal(nil, chunkNonce(e.prefix, e.idx), e.buf, chunkAAD(e.header, e.idx, final))
	if _, err := e.w.Write(ct); err != nil {
		return err
	}
	e.idx++
	e.buf = e.buf[:0]
	return nil
}

// Close writes the final chunk. It does not close the underlying writer.
func (e *encryptWriter) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	return e.flush(true)
}

// decryptReader streams plaintext out of the encrypted container.
type decryptReader struct {
	r      *bufio.Reader
	aead   cipher.AEAD
	header []byte
	prefix []byte
	size   int
	idx    uint32
	out    []byte
	done   bool
	err    error
}

// newDecryptReader reads and checks the header. It returns errBackupWrongKey
// when the secret is of the wrong kind for the file; a wrong secret of the
// right kind surfaces when the first chunk fails authentication.
func newDecryptReader(r io.Reader, secret []byte) (io.Reader, error) {
	br := bufio.NewReaderSize(r, 2*defaultChunk)
	header := make([]byte, backupHeaderLen)
	if _, err := io.ReadFull(br, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			if bytes.HasPrefix(header, []byte(backupMagic)) {
				return nil, errBackupTruncated
			}
			return nil, errBackupNotEncrypted
		}
		return nil, err
	}
	if !bytes.HasPrefix(header, []byte(backupMagic)) {
		return nil, errBackupNotEncrypted
	}
	size := int(binary.BigEndian.Uint32(header[len(backupMagic)+1:]))
	if size < 1 || size > maxChunk {
		return nil, errors.New("encrypted backup header is corrupt (chunk size)")
	}
	salt := header[len(backupMagic)+5 : len(backupMagic)+5+saltLen]
	key, err := deriveBackupKey(secret, header[len(backupMagic)], salt)
	if err != nil {
		return nil, err
	}
	aead, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return &decryptReader{r: br, aead: aead, header: header, prefix: header[len(backupMagic)+5+saltLen:], size: size}, nil
}

func (d *decryptReader) Read(p []byte) (int, error) {
	for len(d.out) == 0 {
		if d.err != nil {
			return 0, d.err
		}
		if d.done {
			return 0, io.EOF
		}
		d.err = d.next()
	}
	n := copy(p, d.out)
	d.out = d.out[n:]
	return n, nil
}

// next authenticates and decrypts one chunk.
func (d *decryptReader) next() error {
	ct := make([]byte, d.size+gcmTagLen)
	n, err := io.ReadFull(d.r, ct)
	switch {
	case err == nil:
	case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
		ct = ct[:n]
	default:
		return err
	}
	// A chunk is final when the file ends right after it.
	_, peekErr := d.r.Peek(1)
	final := errors.Is(peekErr, io.EOF)
	if len(ct) < gcmTagLen {
		return errBackupTruncated
	}
	nonce := chunkNonce(d.prefix, d.idx)
	pt, err := d.aead.Open(nil, nonce, ct, chunkAAD(d.header, d.idx, final))
	if err != nil {
		// Explain the failure: does the chunk verify with the other final flag?
		if _, err2 := d.aead.Open(nil, nonce, ct, chunkAAD(d.header, d.idx, !final)); err2 == nil {
			if final {
				return errBackupTruncated
			}
			return errBackupTrailing
		}
		return errBackupWrongKey
	}
	d.idx++
	d.out = pt
	d.done = final
	return nil
}

// encryptFile encrypts src into dst (created exclusively, 0600, synced) and
// returns the SHA-256 and size of the ciphertext.
func encryptFile(src, dst string, secret []byte) (sum string, size int64, err error) {
	in, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		if err != nil {
			_ = out.Close()
			_ = os.Remove(dst)
		}
	}()
	h := sha256.New()
	cw := &countHash{w: out, h: h}
	ew, err := newEncryptWriter(cw, secret, 0)
	if err != nil {
		return "", 0, err
	}
	if _, err = io.Copy(ew, in); err != nil {
		return "", 0, err
	}
	if err = ew.Close(); err != nil {
		return "", 0, err
	}
	if err = out.Sync(); err != nil {
		return "", 0, err
	}
	if err = out.Close(); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), cw.n, nil
}

// countHash writes to w while hashing and counting the bytes.
type countHash struct {
	w io.Writer
	h hash.Hash
	n int64
}

func (c *countHash) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.h.Write(p[:n])
	c.n += int64(n)
	return n, err
}

// decryptTo decrypts the container in src into w and returns the SHA-256 and
// size of the plaintext. Plaintext released to w before an error is reported
// must be discarded by the caller.
func decryptTo(src string, w io.Writer, secret []byte) (sum string, size int64, err error) {
	in, err := os.Open(src)
	if err != nil {
		return "", 0, err
	}
	defer func() { _ = in.Close() }()
	dr, err := newDecryptReader(in, secret)
	if err != nil {
		return "", 0, err
	}
	cw := &countHash{w: w, h: sha256.New()}
	if _, err := io.Copy(cw, dr); err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(cw.h.Sum(nil)), cw.n, nil
}

// decryptFile decrypts src into dst (0600, synced). On any error dst is
// securely removed, so no partial plaintext is left behind.
func decryptFile(src, dst string, secret []byte) (sum string, size int64, err error) {
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", 0, err
	}
	defer func() {
		if err != nil {
			_ = out.Close()
			secureRemove(dst)
		}
	}()
	if sum, size, err = decryptTo(src, out, secret); err != nil {
		return "", 0, err
	}
	if err = out.Sync(); err != nil {
		return "", 0, err
	}
	if err = out.Close(); err != nil {
		return "", 0, err
	}
	return sum, size, nil
}

// isEncryptedBackup reports whether the file starts with the container magic.
func isEncryptedBackup(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	b := make([]byte, len(backupMagic))
	_, err = io.ReadFull(f, b)
	return err == nil && string(b) == backupMagic
}

// secureRemove overwrites a file with zeros, syncs and deletes it. On
// copy-on-write or journaling file systems and SSDs an overwrite does not
// guarantee the old blocks are gone; the file is in a 0700 directory and the
// disk should be encrypted too (docs/ops/backup-restore.md).
func secureRemove(path string) {
	if fi, err := os.Lstat(path); err == nil && fi.Mode().IsRegular() {
		if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
			zero := make([]byte, 64<<10)
			for left := fi.Size(); left > 0; {
				n := int64(len(zero))
				if left < n {
					n = left
				}
				if _, err := f.Write(zero[:n]); err != nil {
					break
				}
				left -= n
			}
			_ = f.Sync()
			_ = f.Close()
		}
	}
	_ = os.Remove(path)
}

// secureRemoveAll securely removes every file under dir and then dir itself.
func secureRemoveAll(dir string) {
	ents, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range ents {
			p := dir + string(os.PathSeparator) + e.Name()
			if e.IsDir() {
				secureRemoveAll(p)
			} else {
				secureRemove(p)
			}
		}
	}
	_ = os.Remove(dir)
}
