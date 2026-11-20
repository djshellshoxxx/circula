package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/djshellshoxxx/circula/internal/domain"
	"github.com/djshellshoxxx/circula/internal/passwordwork"
	"github.com/djshellshoxxx/circula/internal/service"
)

func TestBackupPasswordAdmissionSharesAuthenticationBudget(t *testing.T) {
	svc, err := service.New(nil, service.Config{Argon: service.Argon2Params{MemoryKiB: 64, Time: 1, Threads: 1, KeyLen: 16, SaltLen: 8}})
	if err != nil {
		t.Fatal(err)
	}
	for range passwordwork.DefaultLimit {
		work, err := passwordwork.Process().Acquire(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer work.Release()
	}
	passphrase := []byte("backup admission fixture passphrase")
	salt := make([]byte, saltLen)
	if _, err := deriveBackupKey(passphrase, kdfArgon2, salt); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("backup passphrase bypassed process admission: %v", err)
	}
	if _, err := svc.LoginStep(t.Context(), "known@example.org", "unused credential"); !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("authentication did not share backup admission: %v", err)
	}
	var output bytes.Buffer
	if writer, err := newEncryptWriter(&output, passphrase, testChunk); writer != nil || !errors.Is(err, domain.ErrUnavailable) || output.Len() != 0 {
		t.Fatalf("busy backup published a partial header: writer=%v bytes=%d err=%v", writer, output.Len(), err)
	}
	// A valid inert header reaches decryption's same derivation boundary;
	// no encrypted payload or costly precomputation is needed for refusal.
	header := make([]byte, backupHeaderLen)
	copy(header, backupMagic)
	header[len(backupMagic)] = kdfArgon2
	binary.BigEndian.PutUint32(header[len(backupMagic)+1:], testChunk)
	if reader, err := newDecryptReader(bytes.NewReader(header), passphrase); reader != nil || !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("busy backup decryption: %v, %v", reader, err)
	}
	if key, err := deriveBackupKey(testKey(t), kdfHKDF, salt); err != nil || len(key) != 32 {
		t.Fatalf("raw-key HKDF should not consume Argon capacity: key length=%d err=%v", len(key), err)
	}
}

func TestBackupPasswordAdmissionRejectsCanceledDerivation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	key, err := deriveBackupKeyContext(ctx, []byte("backup admission fixture passphrase"), kdfArgon2, make([]byte, saltLen))
	if key != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled backup derivation: key length=%d err=%v", len(key), err)
	}
	for range passwordwork.DefaultLimit {
		work, err := passwordwork.Process().Acquire(t.Context())
		if err != nil {
			t.Fatalf("canceled derivation retained capacity: %v", err)
		}
		defer work.Release()
	}
}
