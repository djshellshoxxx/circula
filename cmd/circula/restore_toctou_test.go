package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRestoreSignedSwapDuringPreRestoreBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db := seedDatabase(t, dir)
	cfg := testConfigFor(db)
	cfg.BackupKey = []byte("0123456789abcdef-backup-key")
	_, path, err := backupTo(ctx, cfg, filepath.Join(dir, "bk"), "manual", nil)
	if err != nil {
		t.Fatal(err)
	}
	// forged copy: same bytes except a credential swap; manifest untouched
	forged := filepath.Join(dir, "forged.db")
	if err := copyFileSync(path, forged); err != nil {
		t.Fatal(err)
	}
	raw, _ := openRaw(forged)
	if _, err := raw.Exec(`UPDATE persons SET password_hash = 'attacker'`); err != nil {
		t.Fatal(err)
	}
	_ = raw.Close()
	removeSideFiles(forged)
	// attacker: when restore starts its pre-restore backup, swap the file
	stop := make(chan struct{})
	swapped := make(chan bool, 1)
	go func() {
		bdir := filepath.Join(dir, "backups")
		for {
			select {
			case <-stop:
				swapped <- false
				return
			default:
			}
			if ents, _ := os.ReadDir(bdir); len(ents) > 0 {
				swapped <- os.Rename(forged, path) == nil
				return
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()
	keyFile := writeSecret(t, string(cfg.BackupKey), 0o600)
	var out, errOut bytes.Buffer
	code := run([]string{"restore", "-database", db, "-overwrite", "-backup-key-file", keyFile, path}, &out, &errOut)
	close(stop)
	t.Logf("swapped=%v code=%d\nout=%s\nerr=%s", <-swapped, code, out.String(), errOut.String())
	r2, _ := openRaw(db)
	var h string
	_ = r2.QueryRow(`SELECT password_hash FROM persons LIMIT 1`).Scan(&h)
	_ = r2.Close()
	if code == 0 && h == "attacker" && strings.Contains(out.String(), "signature: valid") {
		t.Fatalf("EXPLOIT SUCCEEDED: forged DB restored under valid-signature message")
	}
}
